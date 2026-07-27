"""RESMA ML recommender — lógica de análise de recursos.

Arquitetura: o ML sidecar NÃO acessa o DuckDB diretamente. Ele solicita
dados via HTTP aos endpoints internos do Go API (/api/internal/*).
O Go API é o único owner do DuckDB, evitando conflitos de lock.

Lógica ML (numpy, scipy, scikit-learn) permanece idêntica à versão anterior.
"""
import logging
import os
import time
from datetime import datetime, timedelta

import httpx
import numpy as np
from scipy import stats
from sklearn.linear_model import LinearRegression

logger = logging.getLogger("resma.ml.recommender")

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
API_URL = os.environ.get("RESMA_API_URL", "http://go-dev:8080")


class ResourceRecommender:
    """Recomendador de recursos — obtém dados via HTTP do Go API."""

    def __init__(self, client: httpx.Client):
        self.client = client
        self._last_recommendation_ts: dict[str, str] = {}
        self._last_sample_count: dict[str, int] = {}
        self._last_suggested: dict[str, dict] = {}
        # Cache de detect_alerts: o dashboard e a página /alerts chamam o
        # mesmo endpoint a cada evento SSE (~5s). Sem cache, o ML sidecar
        # faz N+1 chamadas HTTP ao Go API a cada 5s. Com cache de 30s,
        # apenas 1 chamada real é feita a cada 30s.
        self._alerts_cache: dict | None = None
        self._alerts_cache_ts: float = 0.0

    # --- chamadas HTTP ao Go API ---

    def _get_metrics(self, service: str, days: int = ANALYSIS_WINDOW_DAYS) -> list[dict]:
        """GET /api/internal/services/{service}/metrics"""
        r = self.client.get(
            f"{API_URL}/api/internal/services/{service}/metrics",
            params={"days": days},
            timeout=30.0,
        )
        r.raise_for_status()
        return r.json()

    def _get_oom_count(self, service: str, days: int = ANALYSIS_WINDOW_DAYS) -> int:
        """GET /api/internal/services/{service}/oom-count"""
        r = self.client.get(
            f"{API_URL}/api/internal/services/{service}/oom-count",
            params={"days": days},
            timeout=10.0,
        )
        r.raise_for_status()
        return r.json().get("count", 0)

    def _get_oom_count_since(self, service: str, since_ts: str) -> int:
        """GET /api/internal/services/{service}/oom-count?since=..."""
        r = self.client.get(
            f"{API_URL}/api/internal/services/{service}/oom-count",
            params={"since": since_ts},
            timeout=10.0,
        )
        r.raise_for_status()
        return r.json().get("count", 0)

    def _get_services_with_metrics(self, days: int = ANALYSIS_WINDOW_DAYS) -> list[str]:
        """GET /api/internal/services/with-metrics"""
        r = self.client.get(
            f"{API_URL}/api/internal/services/with-metrics",
            params={"days": days},
            timeout=10.0,
        )
        r.raise_for_status()
        return r.json()

    def _get_active_services(self, minutes: int = ALERT_ACTIVE_WINDOW_MIN) -> list[str]:
        """GET /api/internal/services/with-metrics?minutes=N

        Retorna apenas serviços com métricas nos últimos N minutos (ativos).
        O endpoint com minutes=5 usa INTERVAL 5 MINUTE em vez de INTERVAL 7 DAYS,
        reduzindo de ~11 para ~6 serviços e eliminando chamadas HTTP
        desnecessárias a serviços parados.
        """
        r = self.client.get(
            f"{API_URL}/api/internal/services/with-metrics",
            params={"minutes": minutes},
            timeout=10.0,
        )
        r.raise_for_status()
        return r.json()

    def _get_service_config(self, service: str) -> dict:
        """GET /api/internal/services/{service}/config"""
        r = self.client.get(
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

    def _get_volume_metrics(self, days: int = ANALYSIS_WINDOW_DAYS) -> list[dict]:
        """GET /api/internal/storage/volumes/metrics"""
        r = self.client.get(
            f"{API_URL}/api/internal/storage/volumes/metrics",
            params={"days": days},
            timeout=10.0,
        )
        r.raise_for_status()
        return r.json()

    # --- lógica ML (idêntica à versão anterior, apenas fonte de dados mudou) ---

    def analyze(self, service_name: str) -> dict:
        """Analisar um serviço e retornar recomendação de recursos."""
        current = self._get_service_config(service_name)
        raw = self._get_metrics(service_name)

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
        mem_p50 = float(np.percentile(mem_clean, 50))
        mem_p99 = float(np.percentile(mem_clean, MEM_PERCENTILE))

        leak = self._detect_leak(mem)
        pattern = self._classify_pattern(ts, cpu)
        forecast = self._forecast_memory(mem, FORECAST_DAYS)

        oom_count = self._get_oom_count(service_name)

        drift = self._detect_drift(cpu, mem)
        mem_margin, cpu_margin = self._data_driven_margin(
            cpu_p50, cpu_p95, mem_p50, mem_p99, pattern, oom_count, leak["has_leak"]
        )

        suggested_cpu = max((cpu_p95 * cpu_margin) / 100, 0.1)
        suggested_mem = int(mem_p99 * mem_margin)
        if leak["has_leak"]:
            forecast_mem = forecast["projected_mem_p99"]
            suggested_mem = max(suggested_mem, int(forecast_mem * 1.1))

        res_ratio = 0.75
        suggested = {
            "cpu_limit": round(suggested_cpu, 2),
            "mem_limit": suggested_mem,
            "cpu_reservation": round((cpu_p50 * res_ratio) / 100, 2),
            "mem_reservation": int(mem_p50 * res_ratio),
        }

        status = self._classify_status(current, suggested, cpu_p95, mem_p99, oom_count, leak["has_leak"], drift)
        suggested_apply_time = self._suggest_apply_time(pattern)

        result = {
            "service": service_name,
            "samples": len(raw),
            "status": status,
            "stack": None,
            "preset": "data-driven",
            "current": current,
            "outliers_removed": len(cpu) - len(cpu_clean),
            "cpu": {"p50": round(cpu_p50, 2), "p95": round(cpu_p95, 2)},
            "mem": {"p50": int(mem_p50), "p99": int(mem_p99)},
            "oom_events": oom_count,
            "has_drift": drift,
            "pattern": pattern,
            "memory_trend": leak,
            "forecast": forecast,
            "suggested": suggested,
            "suggested_apply_time": suggested_apply_time,
            "confidence": self._confidence(len(raw), leak["r_squared"]),
        }

        self._last_recommendation_ts[service_name] = ts[-1] if ts else None
        self._last_sample_count[service_name] = len(raw)
        self._last_suggested[service_name] = result["suggested"]
        return result

    def analyze_all(self) -> list[dict]:
        """Analisar todos os serviços com métricas."""
        services = self._get_services_with_metrics()
        results = []
        for svc in services:
            try:
                results.append(self.analyze(svc))
            except Exception as e:
                logger.error("erro analisando %s: %s", svc, e)
        return results

    def evaluate_triggers(self) -> list[dict]:
        """Avaliar triggers de reanálise."""
        services = self._get_services_with_metrics()
        triggers = []
        for svc in services:
            reasons = self._check_triggers(svc)
            if reasons:
                triggers.append({"service": svc, "reasons": reasons})
        return triggers

    def _check_triggers(self, service_name: str) -> list[str]:
        reasons = []
        last_ts = self._last_recommendation_ts.get(service_name)
        last_samples = self._last_sample_count.get(service_name, 0)
        last_suggested = self._last_suggested.get(service_name, {})

        oom_count = self._get_oom_count(service_name)

        if last_ts:
            oom_since = self._get_oom_count_since(service_name, last_ts)
            if oom_since > 0:
                reasons.append(f"oom_event ({oom_since} novos OOMs)")

        raw = self._get_metrics(service_name)
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

    def forecast(self, service_name: str, days_ahead: int = 7) -> dict:
        """Forecast de memória para um serviço."""
        raw = self._get_metrics(service_name)
        if len(raw) < 10:
            return {"service": service_name, "error": "insufficient data", "samples": len(raw)}
        mem = np.array([r["mem_usage"] for r in raw])
        return {"service": service_name, "forecast": self._forecast_memory(mem, days_ahead)}

    def analyze_storage(self) -> dict:
        """Análise de storage."""
        recommendations = []

        try:
            growth = self._get_volume_metrics()
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
            model = LinearRegression()
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

    def detect_alerts(self) -> dict:
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
        ao Go API a cada 5s. Com cache de ALERTS_CACHE_TTL_S segundos,
        apenas 1 chamada real é feita por janela de TTL.
        """
        # Cache: retorna resultado cacheado se ainda válido
        cache_ttl = float(os.environ.get("RESMA_ALERTS_CACHE_TTL_S", "30"))
        now_ts = time.time()
        if self._alerts_cache is not None and (now_ts - self._alerts_cache_ts) < cache_ttl:
            return self._alerts_cache

        leak_alerts: list[dict] = []
        drift_alerts: list[dict] = []

        try:
            # Buscar apenas serviços com métricas na janela ativa (5 min),
            # não todos com histórico de 7 dias. Isso reduz de ~11 para ~6
            # chamadas HTTP ao Go API.
            services = self._get_active_services()
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
                raw = self._get_metrics(svc)
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
        model = LinearRegression()
        model.fit(x, mem)
        slope = model.coef_[0]
        r2 = model.score(x, mem)
        daily_mb = (slope * 24) / (1024 * 1024)
        return {
            "slope_bytes_per_hour": float(slope),
            "daily_growth_mb": round(daily_mb, 2),
            "r_squared": round(r2, 3),
            "has_leak": bool(
                slope > 0 and r2 > LEAK_R2_THRESHOLD and daily_mb > LEAK_DAILY_MB_THRESHOLD
            ),
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

    def _early_recommendation(self) -> dict:
        return {"source": "template"}

    def _forecast_memory(self, mem: np.ndarray, days_ahead: int) -> dict:
        x = np.arange(len(mem)).reshape(-1, 1)
        model = LinearRegression()
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

    def _classify_status(self, current, suggested, cpu_p95, mem_p99, oom_count, has_leak, has_drift) -> str:
        has_config = (
            current.get("cpu_limit", 0) > 0 or current.get("mem_limit", 0) > 0
            or current.get("cpu_reservation", 0) > 0 or current.get("mem_reservation", 0) > 0
        )
        if not has_config:
            return "unconfigured"
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
