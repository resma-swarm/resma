"""RESMA ML recommender — lógica de análise de recursos.

Arquitetura: o ML sidecar NÃO acessa o DuckDB diretamente. Ele solicita
dados via HTTP aos endpoints internos do Go API (/api/internal/*).
O Go API é o único owner do DuckDB, evitando conflitos de lock.

Lógica ML (numpy, scipy, scikit-learn) permanece idêntica à versão anterior.

Memória (fix OOM produção):
- httpx.AsyncClient (não bloqueia event loop — sync Client em async def
  acumulava requests no backlog com numpy arrays vivos).
- LinearRegression reutilizado como singleton (BLAS não devolve memória
  ao SO quando instanciado por chamada).
- gc.collect() a cada GC_COLLECT_EVERY_N requests para liberar buffers
  numpy/sklearn não referenciados.
- Cache de /alerts com TTL configurável (ALERTS_CACHE_TTL_S, default 60s).
"""
import gc
import logging
import os
import time
from datetime import datetime, timedelta

import httpx
import numpy as np
from scipy import stats
from sklearn.linear_model import LinearRegression

logger = logging.getLogger("resma.ml.recommender")

# gc.collect() a cada N requests em /alerts (endpoint mais frequente,
# chamado pelo SSE broker ~36x/min). Default 10.
GC_COLLECT_EVERY_N = int(os.environ.get("RESMA_GC_COLLECT_EVERY_N", "10"))

# Config via env (mesmos defaults do Python original)
ANALYSIS_WINDOW_DAYS = int(os.environ.get("RESMA_ANALYSIS_WINDOW_DAYS", "7"))
CPU_PERCENTILE = float(os.environ.get("RESMA_CPU_PERCENTILE", "95"))
MEM_PERCENTILE = float(os.environ.get("RESMA_MEM_PERCENTILE", "99"))
FORECAST_DAYS = int(os.environ.get("RESMA_FORECAST_DAYS", "7"))
OUTLIER_THRESHOLD = float(os.environ.get("RESMA_OUTLIER_THRESHOLD", "3.0"))
LEAK_R2_THRESHOLD = float(os.environ.get("RESMA_LEAK_R2_THRESHOLD", "0.7"))
LEAK_DAILY_MB_THRESHOLD = float(os.environ.get("RESMA_LEAK_DAILY_MB_THRESHOLD", "10"))
MIN_MARGIN = float(os.environ.get("RESMA_MIN_MARGIN", "1.2"))
MAX_MARGIN = float(os.environ.get("RESMA_MAX_MARGIN", "2.0"))

# Config de alertas (0b.8) — thresholds para leak/drift no /alerts
ALERT_LEAK_R2_THRESHOLD = float(os.environ.get("RESMA_ALERT_LEAK_R2_THRESHOLD", "0.8"))
ALERT_LEAK_DAILY_MB_THRESHOLD = float(os.environ.get("RESMA_ALERT_LEAK_DAILY_MB_THRESHOLD", "0"))
ALERT_DRIFT_THRESHOLD = float(os.environ.get("RESMA_ALERT_DRIFT_THRESHOLD", "0.30"))
ALERT_ACTIVE_WINDOW_MIN = int(os.environ.get("RESMA_ALERT_ACTIVE_WINDOW_MIN", "5"))

# URL do Go API para endpoints internos (nome do serviço Docker)
API_URL = os.environ.get("RESMA_API_URL", "http://api:8080")


class ResourceRecommender:
    """Recomendador de recursos — obtém dados via HTTP do Go API."""

    def __init__(self, client: httpx.AsyncClient):
        self.client = client
        self._last_recommendation_ts: dict[str, str] = {}
        self._last_sample_count: dict[str, int] = {}
        self._last_suggested: dict[str, dict] = {}
        # Cache de detect_alerts: o dashboard e a página /alerts chamam o
        # mesmo endpoint a cada evento SSE (~5s). Sem cache, o ML sidecar
        # faz N+1 chamadas HTTP ao Go API a cada 5s. Com cache de
        # ALERTS_CACHE_TTL_S segundos (default 60s), apenas 1 chamada real
        # é feita por janela de TTL.
        self._alerts_cache: dict | None = None
        self._alerts_cache_ts: float = 0.0
        # Singleton LinearRegression: reutilizar a mesma instância evita
        # alocar buffers BLAS a cada chamada (sklearn não devolve memória
        # ao SO quando o modelo é descartado). O fit() reescreve coef_.
        self._lr_model = LinearRegression()
        # Contador de requests para gc.collect() periódico em /alerts.
        self._alerts_request_count = 0

    def _maybe_gc(self) -> None:
        """gc.collect() a cada GC_COLLECT_EVERY_N calls em /alerts."""
        self._alerts_request_count += 1
        if self._alerts_request_count >= GC_COLLECT_EVERY_N:
            gc.collect()
            self._alerts_request_count = 0

    # --- chamadas HTTP ao Go API (async — não bloqueiam o event loop) ---

    async def _get_metrics(self, service: str, days: int = ANALYSIS_WINDOW_DAYS) -> list[dict]:
        """GET /api/internal/services/{service}/metrics"""
        r = await self.client.get(
            f"{API_URL}/api/internal/services/{service}/metrics",
            params={"days": days},
            timeout=30.0,
        )
        r.raise_for_status()
        return r.json()

    async def _get_oom_count(self, service: str, days: int = ANALYSIS_WINDOW_DAYS) -> int:
        """GET /api/internal/services/{service}/oom-count"""
        r = await self.client.get(
            f"{API_URL}/api/internal/services/{service}/oom-count",
            params={"days": days},
            timeout=10.0,
        )
        r.raise_for_status()
        return r.json().get("count", 0)

    async def _get_oom_count_since(self, service: str, since_ts: str) -> int:
        """GET /api/internal/services/{service}/oom-count?since=..."""
        r = await self.client.get(
            f"{API_URL}/api/internal/services/{service}/oom-count",
            params={"since": since_ts},
            timeout=10.0,
        )
        r.raise_for_status()
        return r.json().get("count", 0)

    async def _get_last_apply(self, service: str) -> dict | None:
        """GET /api/internal/services/{service}/last-apply

        Retorna o timestamp do último apply bem-sucedido, ou None se não houver.
        Usado para distinguir OOMs antes vs depois do apply e classificar
        o status 'observing' (em observação pós-apply).
        """
        try:
            r = await self.client.get(
                f"{API_URL}/api/internal/services/{service}/last-apply",
                timeout=10.0,
            )
            r.raise_for_status()
            data = r.json()
            if not data.get("applied_at"):
                return None
            return data
        except Exception as e:
            logger.debug("last-apply indisponível para %s: %s", service, e)
            return None

    async def _get_services_with_metrics(self, days: int = ANALYSIS_WINDOW_DAYS) -> list[str]:
        """GET /api/internal/services/with-metrics"""
        r = await self.client.get(
            f"{API_URL}/api/internal/services/with-metrics",
            params={"days": days},
            timeout=10.0,
        )
        r.raise_for_status()
        return r.json()

    async def _get_active_services(self, minutes: int = ALERT_ACTIVE_WINDOW_MIN) -> list[str]:
        """GET /api/internal/services/with-metrics?minutes=N

        Retorna apenas serviços com métricas nos últimos N minutos (ativos).
        O endpoint com minutes=5 usa INTERVAL 5 MINUTE em vez de INTERVAL 7 DAYS,
        reduzindo de ~11 para ~6 serviços e eliminando chamadas HTTP
        desnecessárias a serviços parados.
        """
        r = await self.client.get(
            f"{API_URL}/api/internal/services/with-metrics",
            params={"minutes": minutes},
            timeout=10.0,
        )
        r.raise_for_status()
        return r.json()

    async def _get_service_config(self, service: str) -> dict:
        """GET /api/internal/services/{service}/config"""
        r = await self.client.get(
            f"{API_URL}/api/internal/services/{service}/config",
            timeout=10.0,
        )
        r.raise_for_status()
        data = r.json()
        return {
            "cpu_limit": data.get("cpu_limit", 0) or 0,
            "mem_limit": data.get("mem_limit", 0) or 0,
            "cpu_reservation": data.get("cpu_reservation", 0) or 0,
            "mem_reservation": data.get("mem_reservation", 0) or 0,
        }

    async def _get_volume_metrics(self, days: int = ANALYSIS_WINDOW_DAYS) -> list[dict]:
        """GET /api/internal/storage/volumes/metrics"""
        r = await self.client.get(
            f"{API_URL}/api/internal/storage/volumes/metrics",
            params={"days": days},
            timeout=10.0,
        )
        r.raise_for_status()
        return r.json()

    # --- lógica ML (idêntica à versão anterior, apenas fonte de dados mudou) ---

    async def analyze(self, service_name: str) -> dict:
        """Analisar um serviço e retornar recomendação de recursos."""
        current = await self._get_service_config(service_name)
        raw = await self._get_metrics(service_name)

        if len(raw) < 100:
            early = self._early_recommendation()
            return {
                "service": service_name,
                "samples": len(raw),
                "status": "collecting_data",
                "stack": None,
                "preset": "template",
                "current": current,
                "suggested": early,
                "suggested_apply_time": None,
                # Right-Sizing Studio — estado collecting_data (spec ml-payload-schema §5)
                "suggested_tiers": None,
                "risk": {
                    "level": "attention",
                    "score": 3,
                    "color": "yellow",
                    "reasons": [f"dados insuficientes (<100 amostras: {len(raw)})"],
                    "margin_cpu": 0,
                    "margin_mem": 0,
                    "forecast_vs_limit_pct": 0,
                },
                "explainability": {
                    "summary": f"Coletando dados — {len(raw)} de 100 amostras mínimas. Recomendação disponível em ~1 minuto.",
                    "factors": [
                        {
                            "label": "Amostras",
                            "value": f"{len(raw)}/100",
                            "detail": "mínimo de 100 amostras para análise estatística",
                        }
                    ],
                },
                "histograms": None,
                "resources_freed": None,
                "confidence": "low",
            }

        ts = [r["ts"] for r in raw]
        cpu = np.array([r["cpu_percent"] for r in raw])
        mem = np.array([r["mem_usage"] for r in raw])

        cpu_clean = self._remove_outliers(cpu)
        mem_clean = self._remove_outliers(mem)
        if len(cpu_clean) == 0 or len(mem_clean) == 0:
            cpu_clean = cpu
            mem_clean = mem

        cpu_p50 = float(np.percentile(cpu_clean, 50))
        cpu_p95 = float(np.percentile(cpu_clean, CPU_PERCENTILE))
        cpu_p99 = float(np.percentile(cpu_clean, 99))
        mem_p50 = float(np.percentile(mem_clean, 50))
        mem_p95 = float(np.percentile(mem_clean, 95))
        mem_p99 = float(np.percentile(mem_clean, MEM_PERCENTILE))

        leak = self._detect_leak(mem)
        pattern = self._classify_pattern(ts, cpu)
        forecast = self._forecast_memory(mem, FORECAST_DAYS)

        oom_count = await self._get_oom_count(service_name)

        # Buscar último apply bem-sucedido para distinguir OOMs antes vs depois.
        # Se houve apply, o status passa a "observing" (em observação) enquanto
        # não houver OOMs novos pós-apply — o usuário já tomou uma ação e o
        # serviço está sendo reavaliado.
        last_apply = await self._get_last_apply(service_name)
        oom_count_since_apply = 0
        applied_at = None
        if last_apply and last_apply.get("applied_at"):
            applied_at = last_apply["applied_at"]
            oom_count_since_apply = await self._get_oom_count_since(service_name, applied_at)

        drift = self._detect_drift(cpu, mem)
        mem_margin, cpu_margin = self._data_driven_margin(
            cpu_p50, cpu_p95, mem_p50, mem_p99, pattern, oom_count, leak["has_leak"]
        )

        # Right-Sizing Studio: 3 tiers (conservative/balanced/aggressive).
        # `suggested` continua existindo como alias de suggested_tiers.balanced
        # (backward compatibility — frontend antigo só lê `suggested`).
        suggested_tiers = self._calculate_tiers(
            cpu_p95, mem_p99, cpu_p50, mem_p50, current, pattern,
            oom_count, leak["has_leak"], forecast,
        )
        balanced = suggested_tiers["balanced"]
        suggested = {
            "cpu_limit": balanced["cpu_limit"],
            "mem_limit": balanced["mem_limit"],
            "cpu_reservation": balanced["cpu_reservation"],
            "mem_reservation": balanced["mem_reservation"],
        }

        status = self._classify_status(
            current, suggested, cpu_p95, mem_p99, oom_count, leak["has_leak"], drift,
            oom_count_since_apply=oom_count_since_apply, applied_at=applied_at,
        )
        suggested_apply_time = self._suggest_apply_time(pattern)

        risk = self._calculate_risk(
            cpu_p95, mem_p99, suggested, oom_count, leak["has_leak"], drift, forecast, status,
            oom_count_since_apply=oom_count_since_apply,
        )
        explainability = self._build_explainability(
            service_name, cpu_p95, mem_p99, cpu_p50, mem_p50, pattern,
            oom_count, leak, cpu_margin, suggested, len(raw),
        )
        histograms = self._build_histograms(cpu_clean, mem_clean)

        result = {
            "service": service_name,
            "samples": len(raw),
            "status": status,
            "stack": None,
            "preset": "data-driven",
            "current": current,
            "outliers_removed": len(cpu) - len(cpu_clean),
            "cpu": {"p50": round(cpu_p50, 2), "p95": round(cpu_p95, 2), "p99": round(cpu_p99, 2)},
            "mem": {"p50": int(mem_p50), "p95": int(mem_p95), "p99": int(mem_p99)},
            "oom_events": oom_count,
            "oom_events_since_apply": oom_count_since_apply,
            "applied_at": applied_at,
            "has_drift": drift,
            "pattern": pattern,
            "memory_trend": leak,
            "forecast": forecast,
            "suggested": suggested,
            "suggested_apply_time": suggested_apply_time,
            "confidence": self._confidence(len(raw), leak["r_squared"]),
            # Right-Sizing Studio — campos novos (ml-payload-schema.md §3)
            "suggested_tiers": suggested_tiers,
            "risk": risk,
            "explainability": explainability,
            "histograms": histograms,
            "resources_freed": {"balanced": balanced["resources_freed"]},
        }

        self._last_recommendation_ts[service_name] = ts[-1] if ts else None
        self._last_sample_count[service_name] = len(raw)
        self._last_suggested[service_name] = result["suggested"]
        return result

    async def analyze_all(self) -> list[dict]:
        """Analisar todos os serviços com métricas."""
        services = await self._get_services_with_metrics()
        results = []
        for svc in services:
            try:
                results.append(await self.analyze(svc))
            except Exception as e:
                logger.error("erro analisando %s: %s", svc, e)
        return results

    async def evaluate_triggers(self) -> list[dict]:
        """Avaliar triggers de reanálise."""
        services = await self._get_services_with_metrics()
        triggers = []
        for svc in services:
            reasons = await self._check_triggers(svc)
            if reasons:
                triggers.append({"service": svc, "reasons": reasons})
        return triggers

    async def _check_triggers(self, service_name: str) -> list[str]:
        reasons = []
        last_ts = self._last_recommendation_ts.get(service_name)
        last_samples = self._last_sample_count.get(service_name, 0)
        last_suggested = self._last_suggested.get(service_name, {})

        oom_count = await self._get_oom_count(service_name)

        if last_ts:
            oom_since = await self._get_oom_count_since(service_name, last_ts)
            if oom_since > 0:
                reasons.append(f"oom_event ({oom_since} novos OOMs)")

        raw = await self._get_metrics(service_name)
        if len(raw) < 100:
            return reasons

        new_samples = len(raw) - last_samples
        if new_samples >= 1000:
            reasons.append(f"new_samples (+{new_samples} amostras)")

        mem = np.array([r["mem_usage"] for r in raw])
        leak = self._detect_leak(mem)
        if leak["has_leak"]:
            reasons.append("memory_leak_detectado")

        if last_suggested:
            cpu_arr = np.array([r["cpu_percent"] for r in raw])
            cpu_clean = self._remove_outliers(cpu_arr)
            if len(cpu_clean) == 0:
                cpu_clean = cpu_arr
            mem_clean = self._remove_outliers(mem)
            if len(mem_clean) == 0:
                mem_clean = mem

            cpu_p95 = float(np.percentile(cpu_clean, CPU_PERCENTILE))
            mem_p99 = float(np.percentile(mem_clean, MEM_PERCENTILE))

            sug_cpu = last_suggested.get("cpu_limit", 0)
            sug_mem = last_suggested.get("mem_limit", 0)

            if sug_cpu > 0:
                current_sug_cpu = max((cpu_p95 * 1.3) / 100, 0.1)
                cpu_drift = abs(current_sug_cpu - sug_cpu) / sug_cpu
                if cpu_drift > 0.30:
                    reasons.append(f"cpu_drift ({cpu_drift:.0%})")

            if sug_mem > 0:
                current_sug_mem = int(mem_p99 * 1.3)
                mem_drift = abs(current_sug_mem - sug_mem) / sug_mem
                if mem_drift > 0.30:
                    reasons.append(f"mem_drift ({mem_drift:.0%})")

        return reasons

    async def forecast(self, service_name: str, days_ahead: int = 7) -> dict:
        """Forecast de memória para um serviço."""
        raw = await self._get_metrics(service_name)
        if len(raw) < 10:
            return {"service": service_name, "error": "insufficient data", "samples": len(raw)}
        mem = np.array([r["mem_usage"] for r in raw])
        return {"service": service_name, "forecast": self._forecast_memory(mem, days_ahead)}

    async def analyze_storage(self) -> dict:
        """Análise de storage."""
        recommendations = []

        try:
            growth = await self._get_volume_metrics()
        except Exception:
            growth = []

        volume_series: dict[str, list] = {}
        for point in growth:
            name = point["volume_name"]
            ts_val = point["ts"]
            size = point["size_bytes"]
            volume_series.setdefault(name, []).append((ts_val, size))

        for vol_name, series in volume_series.items():
            if len(series) < 5:
                continue
            sizes = np.array([s[1] for s in series])
            x = np.arange(len(sizes)).reshape(-1, 1)
            model = self._lr_model
            model.fit(x, sizes)
            slope = model.coef_[0]
            r2 = model.score(x, sizes)

            if slope > 0 and r2 > 0.5:
                daily_growth_mb = (slope * 24) / (1024 * 1024)
                if daily_growth_mb > 1:
                    current_size = sizes[-1]
                    hours_to_fill = (current_size * 0.5) / slope
                    days_to_fill = hours_to_fill / 24

                    rec = {
                        "type": "volume_growth",
                        "severity": "critical" if days_to_fill < 7 else "warning",
                        "volume": vol_name,
                        "current_size_bytes": int(current_size),
                        "daily_growth_mb": round(daily_growth_mb, 2),
                        "r_squared": round(r2, 3),
                        "message": f"Volume '{vol_name}' crescendo {daily_growth_mb:.1f} MB/dia (R²={r2:.2f})",
                    }
                    if days_to_fill < 30:
                        rec["days_to_double"] = round(days_to_fill, 1)
                        rec["message"] += f" — vai dobrar de tamanho em ~{days_to_fill:.0f} dias"
                    rec["action"] = f"Investigar conteúdo de '{vol_name}'"
                    rec["action_label"] = "Investigar crescimento"
                    recommendations.append(rec)

        return {
            "summary": {"recommendation_count": len(recommendations)},
            "recommendations": recommendations,
        }

    async def detect_alerts(self) -> dict:
        """Detectar memory leaks e resource drifts para serviços ativos.

        Serviço "ativo" = tem métricas nos últimos ALERT_ACTIVE_WINDOW_MIN
        minutos. Retorna no shape esperado pelo frontend Dashboard:
            leak_alerts:  [{ service, daily_growth_mb, r_squared }]
            drift_alerts: [{ service, cpu_drift, mem_drift }]

        Thresholds (env, defaults 0b.8):
            - leak:   daily_growth_mb > 0 e r_squared > 0.8
            - drift:  delta P50/P95 entre janela atual vs anterior > 0.30

        Cache: o dashboard e a página /alerts chamam este método a cada
        evento SSE (~5s). Sem cache, o ML sidecar faz N+1 chamadas HTTP
        ao Go API a cada 5s. Com cache de ALERTS_CACHE_TTL_S segundos
        (default 60s), apenas 1 chamada real é feita por janela de TTL.

        Memória: gc.collect() a cada GC_COLLECT_EVERY_N requests para
        liberar buffers numpy/sklearn não referenciados (fix OOM produção).
        """
        # gc.collect() periódico — /alerts é o endpoint mais frequente
        self._maybe_gc()

        # Cache: retorna resultado cacheado se ainda válido
        cache_ttl = float(os.environ.get("RESMA_ALERTS_CACHE_TTL_S", "60"))
        now_ts = time.time()
        if self._alerts_cache is not None and (now_ts - self._alerts_cache_ts) < cache_ttl:
            return self._alerts_cache

        leak_alerts: list[dict] = []
        drift_alerts: list[dict] = []

        try:
            # Buscar apenas serviços com métricas na janela ativa (5 min),
            # não todos com histórico de 7 dias. Isso reduz de ~11 para ~6
            # chamadas HTTP ao Go API.
            services = await self._get_active_services()
        except Exception as e:
            logger.error("detect_alerts: erro listando serviços: %s", e)
            result = {"leak_alerts": leak_alerts, "drift_alerts": drift_alerts}
            self._alerts_cache = result
            self._alerts_cache_ts = now_ts
            return result

        now = datetime.utcnow()
        active_cutoff = now - timedelta(minutes=ALERT_ACTIVE_WINDOW_MIN)

        for svc in services:
            try:
                raw = await self._get_metrics(svc)
            except Exception as e:
                logger.warning("detect_alerts: sem métricas para %s: %s", svc, e)
                continue

            if len(raw) < 2:
                continue

            # Filtro ativos: último ts deve ser dentro da janela
            last_ts = self._parse_ts(raw[-1]["ts"])
            if last_ts is None or last_ts < active_cutoff:
                continue

            mem = np.array([r["mem_usage"] for r in raw], dtype=float)
            cpu = np.array([r["cpu_percent"] for r in raw], dtype=float)

            # --- Memory leak: regressão linear sobre mem_usage temporal ---
            leak = self._detect_leak(mem)
            if (
                leak["daily_growth_mb"] > ALERT_LEAK_DAILY_MB_THRESHOLD
                and leak["r_squared"] > ALERT_LEAK_R2_THRESHOLD
            ):
                leak_alerts.append({
                    "service": svc,
                    "daily_growth_mb": leak["daily_growth_mb"],
                    "r_squared": leak["r_squared"],
                })

            # --- Resource drift: P50/P95 atual vs anterior ---
            drift = self._detect_drift_detailed(cpu, mem)
            if drift["cpu_drift"] > ALERT_DRIFT_THRESHOLD or drift["mem_drift"] > ALERT_DRIFT_THRESHOLD:
                drift_alerts.append({
                    "service": svc,
                    "cpu_drift": round(drift["cpu_drift"], 3),
                    "mem_drift": round(drift["mem_drift"], 3),
                })

        # Ordenar por severidade (maior crescimento / maior drift primeiro)
        leak_alerts.sort(key=lambda a: a["daily_growth_mb"], reverse=True)
        drift_alerts.sort(key=lambda a: max(a["cpu_drift"], a["mem_drift"]), reverse=True)

        result = {"leak_alerts": leak_alerts, "drift_alerts": drift_alerts}
        # Salvar no cache
        self._alerts_cache = result
        self._alerts_cache_ts = now_ts
        return result

    @staticmethod
    def _parse_ts(ts: str):
        """Parse de ts ISO (do Go API) para datetime UTC. Retorna None se inválido."""
        if not ts:
            return None
        try:
            # Go envia RFC3339Nano; fromisoformat aceita a maioria dos casos
            return datetime.fromisoformat(ts.replace("Z", "+00:00")).replace(tzinfo=None)
        except (ValueError, TypeError):
            return None

    # --- helpers ML (inalterados) ---

    def _remove_outliers(self, values: np.ndarray) -> np.ndarray:
        if len(values) < 3:
            return values
        z = stats.zscore(values)
        return values[np.abs(z) < OUTLIER_THRESHOLD]

    def _detect_leak(self, mem: np.ndarray) -> dict:
        x = np.arange(len(mem)).reshape(-1, 1)
        model = self._lr_model
        model.fit(x, mem)
        slope = model.coef_[0]
        r2 = model.score(x, mem)
        daily_mb = (slope * 24) / (1024 * 1024)
        # projected_mem_7d = última amostra + slope * 24h * 7d
        projected_7d = float(mem[-1] + slope * 24 * 7) if len(mem) > 0 else 0.0
        return {
            "slope_bytes_per_hour": float(slope),
            "daily_growth_mb": round(daily_mb, 2),
            "r_squared": round(r2, 3),
            "has_leak": bool(
                slope > 0 and r2 > LEAK_R2_THRESHOLD and daily_mb > LEAK_DAILY_MB_THRESHOLD
            ),
            "projected_mem_7d": int(projected_7d),
        }

    def _detect_drift(self, cpu: np.ndarray, mem: np.ndarray) -> bool:
        if len(cpu) < 60:
            return False
        mid = len(cpu) // 2
        cpu_old, cpu_new = cpu[:mid], cpu[mid:]
        mem_old, mem_new = mem[:mid], mem[mid:]
        cpu_old_p95 = float(np.percentile(cpu_old, 95)) if len(cpu_old) > 0 else 0
        cpu_new_p95 = float(np.percentile(cpu_new, 95)) if len(cpu_new) > 0 else 0
        mem_old_p99 = float(np.percentile(mem_old, 99)) if len(mem_old) > 0 else 0
        mem_new_p99 = float(np.percentile(mem_new, 99)) if len(mem_new) > 0 else 0
        cpu_drift = abs(cpu_new_p95 - cpu_old_p95) / max(cpu_old_p95, 0.1)
        mem_drift = abs(mem_new_p99 - mem_old_p99) / max(mem_old_p99, 1.0)
        return cpu_drift > 0.30 or mem_drift > 0.30

    def _detect_drift_detailed(self, cpu: np.ndarray, mem: np.ndarray) -> dict:
        """Drift detalhado: compara P50/P95 de CPU e memória entre janela
        atual vs anterior. Retorna {cpu_drift, mem_drift} como frações (0.30 = 30%).

        Usa P50 e P95 (não apenas P95/P99) para capturar tanto shift de
        baseline quanto de picos. O delta reportado é o máximo entre
        delta_P50 e delta_P95 por recurso.
        """
        if len(cpu) < 4:
            return {"cpu_drift": 0.0, "mem_drift": 0.0}
        mid = len(cpu) // 2
        cpu_old, cpu_new = cpu[:mid], cpu[mid:]
        mem_old, mem_new = mem[:mid], mem[mid:]

        def _drift_ratio(old_p50, new_p50, old_p95, new_p95, floor):
            d50 = abs(new_p50 - old_p50) / max(old_p50, floor)
            d95 = abs(new_p95 - old_p95) / max(old_p95, floor)
            return max(d50, d95)

        cpu_old_p50 = float(np.percentile(cpu_old, 50)) if len(cpu_old) > 0 else 0.0
        cpu_new_p50 = float(np.percentile(cpu_new, 50)) if len(cpu_new) > 0 else 0.0
        cpu_old_p95 = float(np.percentile(cpu_old, 95)) if len(cpu_old) > 0 else 0.0
        cpu_new_p95 = float(np.percentile(cpu_new, 95)) if len(cpu_new) > 0 else 0.0
        mem_old_p50 = float(np.percentile(mem_old, 50)) if len(mem_old) > 0 else 0.0
        mem_new_p50 = float(np.percentile(mem_new, 50)) if len(mem_new) > 0 else 0.0
        mem_old_p95 = float(np.percentile(mem_old, 95)) if len(mem_old) > 0 else 0.0
        mem_new_p95 = float(np.percentile(mem_new, 95)) if len(mem_new) > 0 else 0.0

        cpu_drift = _drift_ratio(cpu_old_p50, cpu_new_p50, cpu_old_p95, cpu_new_p95, 0.1)
        mem_drift = _drift_ratio(mem_old_p50, mem_new_p50, mem_old_p95, mem_new_p95, 1.0)
        return {"cpu_drift": cpu_drift, "mem_drift": mem_drift}

    def _classify_pattern(self, ts, cpu) -> str:
        hourly = {}
        for i, t in enumerate(ts):
            # ts agora é string ISO (do JSON), extrair hora
            if isinstance(t, str):
                h = int(t[11:13])
            elif hasattr(t, "hour"):
                h = t.hour
            else:
                h = int(str(t)[11:13])
            hourly.setdefault(h, []).append(cpu[i])
        avgs = {h: float(np.mean(v)) for h, v in hourly.items()}
        if len(avgs) < 12:
            return "unknown"
        vals = list(avgs.values())
        ratio = max(vals) / min(vals) if min(vals) > 0 else 999
        if ratio > 3:
            return "business_hours"
        elif ratio < 1.5:
            return "constant"
        return "batch"

    def _suggest_apply_time(self, pattern: str) -> str:
        now = datetime.now()
        if pattern == "business_hours":
            days_ahead = (5 - now.weekday()) % 7
            if days_ahead == 0 and now.hour >= 2:
                days_ahead = 7
            target = now + timedelta(days=days_ahead)
            target = target.replace(hour=2, minute=0, second=0, microsecond=0)
        elif pattern == "batch":
            target = now + timedelta(days=1)
            target = target.replace(hour=3, minute=0, second=0, microsecond=0)
        elif pattern == "constant":
            target = now + timedelta(days=1)
            target = target.replace(hour=2, minute=0, second=0, microsecond=0)
        else:
            days_ahead = (5 - now.weekday()) % 7
            if days_ahead == 0 and now.hour >= 2:
                days_ahead = 7
            target = now + timedelta(days=days_ahead)
            target = target.replace(hour=2, minute=0, second=0, microsecond=0)
        return target.isoformat()

    def _data_driven_margin(self, cpu_p50, cpu_p95, mem_p50, mem_p99, pattern, oom, has_leak):
        cpu_stability = cpu_p50 / cpu_p95 if cpu_p95 > 0 else 1.0
        mem_stability = mem_p50 / mem_p99 if mem_p99 > 0 else 1.0
        cpu_margin = MIN_MARGIN + (1 - cpu_stability) * (MAX_MARGIN - MIN_MARGIN)
        mem_margin = MIN_MARGIN + (1 - mem_stability) * (MAX_MARGIN - MIN_MARGIN)
        cpu_margin = min(max(cpu_margin, MIN_MARGIN), MAX_MARGIN)
        mem_margin = min(max(mem_margin, MIN_MARGIN), MAX_MARGIN)
        if pattern == "business_hours":
            cpu_margin *= 1.15
        if oom > 0:
            mem_margin += 0.3
            cpu_margin += 0.2
        if has_leak:
            mem_margin = max(mem_margin, 1.5)
        return mem_margin, cpu_margin

    # --- Right-Sizing Studio: helpers para payload estendido (ml-payload-schema.md §3) ---

    def _calc_freed(self, current: dict, cpu_limit: float, mem_limit: int) -> dict:
        """Delta de recursos liberados (current - suggested), nunca negativo."""
        cur_cpu = current.get("cpu_limit", 0) or 0
        cur_mem = current.get("mem_limit", 0) or 0
        cpu_freed = max(cur_cpu - cpu_limit, 0)
        mem_freed = max(cur_mem - mem_limit, 0)
        cpu_pct = int((cpu_freed / cur_cpu * 100)) if cur_cpu > 0 else 0
        mem_pct = int((mem_freed / cur_mem * 100)) if cur_mem > 0 else 0
        return {
            "cpu_cores": round(cpu_freed, 2),
            "mem_bytes": int(mem_freed),
            "cpu_pct": cpu_pct,
            "mem_pct": mem_pct,
        }

    def _calculate_tiers(
        self, cpu_p95: float, mem_p99: float, cpu_p50: float, mem_p50: float,
        current: dict, pattern: str, oom: int, has_leak: bool, forecast: dict,
    ) -> dict:
        """3 tiers: conservative (margem fixa 2.0/1.8), balanced (data-driven),
        aggressive (fixa 1.1/1.1). `suggested` (top-level) é alias de balanced."""
        mem_margin_balanced, cpu_margin_balanced = self._data_driven_margin(
            cpu_p50, cpu_p95, mem_p50, mem_p99, pattern, oom, has_leak
        )
        res_ratio = 0.75
        tier_margins = {
            "conservative": (2.0, 1.8),
            "balanced": (cpu_margin_balanced, mem_margin_balanced),
            "aggressive": (1.1, 1.1),
        }
        tiers = {}
        for name, (cpu_m, mem_m) in tier_margins.items():
            cpu_limit = max((cpu_p95 * cpu_m) / 100, 0.1)
            mem_limit = int(mem_p99 * mem_m)
            if has_leak:
                mem_limit = max(mem_limit, int(forecast["projected_mem_p99"] * 1.1))
            tiers[name] = {
                "cpu_limit": round(cpu_limit, 2),
                "mem_limit": mem_limit,
                "cpu_reservation": round((cpu_p50 * res_ratio) / 100, 2),
                "mem_reservation": int(mem_p50 * res_ratio),
                "margin_cpu": round(cpu_m, 2),
                "margin_mem": round(mem_m, 2),
                "resources_freed": self._calc_freed(current, cpu_limit, mem_limit),
            }
        return tiers

    def _calculate_risk(
        self, cpu_p95: float, mem_p99: float, suggested: dict,
        oom_count: int, has_leak: bool, has_drift: bool, forecast: dict, status: str,
        oom_count_since_apply: int = 0,
    ) -> dict:
        """Score de risco estruturado (5 níveis: very_low/low/attention/high/critical).

        Se houve apply e não há OOMs novos pós-apply, o risk score é reduzido
        — o serviço está em observação e os OOMs antigos não devem mantê-lo
        como "high"/"critical".
        """
        # OOMs relevantes para risk: se houve apply, usa apenas OOMs pós-apply.
        # Se não houve apply, usa o total.
        oom_for_risk = oom_count_since_apply if oom_count_since_apply > 0 or status == "observing" else oom_count
        observing = status == "observing"

        margin_cpu = (suggested["cpu_limit"] * 100 / cpu_p95) if cpu_p95 > 0 else 99
        margin_mem = (suggested["mem_limit"] / mem_p99) if mem_p99 > 0 else 99
        forecast_pct = (
            (forecast["projected_mem_p99"] / suggested["mem_limit"] * 100)
            if suggested["mem_limit"] > 0 else 0
        )

        reasons = []
        if oom_count == 0:
            reasons.append("0 OOMs em 30d")
        elif observing and oom_count_since_apply == 0:
            reasons.append(f"{oom_count} OOMs antes do apply, 0 após")
        else:
            reasons.append(f"{oom_count} OOMs em 30d ({oom_count_since_apply} após apply)")
        reasons.append("sem leak detectado" if not has_leak else "leak detectado")
        if margin_cpu >= 1.3:
            reasons.append(f"margem CPU {margin_cpu:.1f}x")
        if forecast_pct < 90:
            reasons.append(f"forecast {forecast_pct:.0f}% do limite")
        if observing:
            reasons.append("em observação pós-apply")

        if status == "under_provisioned":
            level, score, color = "critical", 5, "red"
        elif oom_for_risk > 0 or has_leak:
            level, score, color = "high", 4, "orange"
        elif observing:
            # Em observação: risk baixo (ação já tomada, sem novos problemas)
            level, score, color = "low", 2, "blue"
        elif margin_cpu < 1.2 or margin_mem < 1.2 or forecast_pct > 90:
            level, score, color = "attention", 3, "yellow"
        elif margin_cpu >= 1.3 and oom_count == 0 and not has_leak:
            level, score, color = "very_low", 1, "green"
        else:
            level, score, color = "low", 2, "green"

        return {
            "level": level,
            "score": score,
            "color": color,
            "reasons": reasons,
            "margin_cpu": round(margin_cpu, 2),
            "margin_mem": round(margin_mem, 2),
            "forecast_vs_limit_pct": round(forecast_pct, 0),
        }

    def _pattern_label(self, pattern: str) -> str:
        """Mapeia valores reais do ML para rótulos de exibição amigáveis.
        NÃO usar web/db/mixed — não existem no payload."""
        return {
            "constant": "Constante",
            "business_hours": "Horário comercial",
            "batch": "Batch",
            "unknown": "Desconhecido",
        }.get(pattern, pattern.capitalize())

    def _pattern_detail(self, pattern: str) -> str:
        return {
            "constant": "uso constante sem spikes — permite margem menor",
            "business_hours": "uso concentrado em horário comercial — margem intermediária",
            "batch": "spikes periódicos — requer margem maior",
            "unknown": "padrão não classificado — margem conservadora",
        }.get(pattern, "padrão não classificado")

    def _build_explainability(
        self, service: str, cpu_p95: float, mem_p99: float, cpu_p50: float,
        mem_p50: float, pattern: str, oom_count: int, leak: dict,
        margin: float, suggested: dict, samples: int,
    ) -> dict:
        """Texto em linguagem natural + fatores estruturados para o frontend."""
        cpu_p95_cores = cpu_p95 / 100
        leak_text = (
            "sem leak detectado" if not leak["has_leak"]
            else f"leak detectado (R²={leak['r_squared']:.2f})"
        )
        margin_reason = (
            "data-driven: baixa variabilidade" if margin < 1.5
            else "data-driven: alta variabilidade"
        )
        pattern_label = self._pattern_label(pattern)

        summary = (
            f"Sugerido {suggested['cpu_limit']:.2f} cores porque P95 dos últimos 7d = {cpu_p95:.1f}% "
            f"({cpu_p95_cores:.2f} cores), padrão {pattern_label}, "
            f"{oom_count} OOMs, {leak_text}. "
            f"Margem {margin:.1f}x aplicada ({margin_reason})."
        )

        factors = [
            {"label": "P95 CPU", "value": f"{cpu_p95:.1f}%", "detail": "percentil 95 do uso de CPU nos últimos 7d"},
            {"label": "P99 Mem", "value": f"{mem_p99 / 1e6:.0f}MB", "detail": "percentil 99 do uso de memória nos últimos 7d"},
            {"label": "Padrão", "value": pattern_label, "detail": self._pattern_detail(pattern)},
            {"label": "OOMs", "value": str(oom_count), "detail": f"OOM kills nos últimos {ANALYSIS_WINDOW_DAYS}d"},
            {"label": "Leak", "value": "não" if not leak["has_leak"] else "sim",
             "detail": f"R²={leak['r_squared']:.2f} (threshold {LEAK_R2_THRESHOLD})"},
            {"label": "Margem", "value": f"{margin:.1f}x", "detail": margin_reason},
        ]
        return {"summary": summary, "factors": factors}

    def _build_histograms(self, cpu_clean: np.ndarray, mem_clean: np.ndarray) -> dict:
        """Distribuição pré-bucketed para o frontend renderizar histogramas."""
        cpu_buckets = [0, 5, 10, 15, 20, 25, 30, 40, 50, 75, 100]
        cpu_counts, _ = np.histogram(cpu_clean, bins=cpu_buckets)

        mem_mb = mem_clean / 1e6
        mem_buckets_mb = [0, 100, 200, 300, 350, 400, 500, 750, 1000, 1500, 2000, 4000]
        mem_counts, _ = np.histogram(mem_mb, bins=mem_buckets_mb)

        return {
            "cpu": {"buckets": cpu_buckets, "counts": cpu_counts.tolist()},
            "mem": {"buckets_mb": mem_buckets_mb, "counts": mem_counts.tolist()},
        }

    def _early_recommendation(self) -> dict:
        return {"source": "template"}

    def _forecast_memory(self, mem: np.ndarray, days_ahead: int) -> dict:
        x = np.arange(len(mem)).reshape(-1, 1)
        model = self._lr_model
        model.fit(x, mem)
        slope = model.coef_[0]
        samples_per_day = len(mem) / ANALYSIS_WINDOW_DAYS
        future_points = int(samples_per_day * days_ahead)
        projected = model.predict(np.array([[len(mem) + future_points]]))
        projected_mem = float(projected[0])
        residuals = mem - model.predict(x)
        residual_std = float(np.std(residuals)) if len(residuals) > 1 else 0.0
        projected_p99 = projected_mem + 2.33 * residual_std
        return {
            "days_ahead": days_ahead,
            "projected_mem": int(projected_mem),
            "projected_mem_p99": int(projected_p99),
            "slope_bytes_per_hour": float(slope),
        }

    def _confidence(self, samples: int, r2: float) -> str:
        if samples > 5000:
            return "high"
        elif samples > 1000:
            return "medium"
        return "low"

    def _classify_status(
        self, current, suggested, cpu_p95, mem_p99, oom_count, has_leak, has_drift,
        oom_count_since_apply: int = 0, applied_at: str | None = None,
    ) -> str:
        has_config = (
            current.get("cpu_limit", 0) > 0 or current.get("mem_limit", 0) > 0
            or current.get("cpu_reservation", 0) > 0 or current.get("mem_reservation", 0) > 0
        )
        if not has_config:
            return "unconfigured"

        # Se houve apply e não há OOMs novos desde o apply, o serviço está em
        # observação — o usuário já tomou uma ação e o serviço está sendo
        # reavaliado. O status "observing" tem precedência sobre "alerted" e
        # "under_provisioned" porque as métricas atuais ainda incluem dados
        # pré-apply (janela de 7 dias) e não refletem a nova config.
        # Precisa de tempo para que métricas pós-apply se acumulem.
        if applied_at and oom_count_since_apply == 0 and not has_leak and not has_drift:
            return "observing"

        # Se houve apply mas há OOMs novos pós-apply, o problema persiste —
        # mantém "alerted" para indicar que a ação não resolveu.
        if oom_count > 0 or has_leak or has_drift:
            return "alerted"
        cpu_limit = current.get("cpu_limit", 0)
        mem_limit = current.get("mem_limit", 0)
        if cpu_limit > 0 and cpu_p95 > 0:
            if cpu_p95 / (cpu_limit * 100) > 0.80:
                return "under_provisioned"
        if mem_limit > 0 and mem_p99 > 0:
            if mem_p99 / mem_limit > 0.80:
                return "under_provisioned"
        sug_cpu = suggested.get("cpu_limit", 0)
        sug_mem = suggested.get("mem_limit", 0)
        if cpu_limit > 0 and sug_cpu > 0 and cpu_limit > sug_cpu * 2:
            return "over_provisioned"
        if mem_limit > 0 and sug_mem > 0 and mem_limit > sug_mem * 2:
            return "over_provisioned"
        return "healthy"
