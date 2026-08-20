"""RESMA ML Sidecar — FastAPI minimal para análise de recursos.

Endpoints:
  GET  /health           — health check
  GET  /analyze           — analisar todos os serviços
  GET  /analyze/{service} — analisar um serviço específico
  GET  /triggers          — avaliar triggers de reanálise
  GET  /analyze/storage   — análise de storage
  GET  /forecast/{service} — forecast de memória para um serviço
  GET  /alerts            — detectar memory leaks e resource drifts (0b.8)

Arquitetura: o ML sidecar NÃO acessa o DuckDB diretamente. Ele solicita
dados via HTTP aos endpoints internos do Go API (/api/internal/*).
O Go API é o único owner do DuckDB, evitando conflitos de lock.

Memória (fix OOM produção):
- httpx.AsyncClient (não bloqueia event loop — sync Client em async def
  acumulava requests no backlog com numpy arrays vivos).
- LinearRegression singleton no recommender (BLAS não devolve memória).
- gc.collect() periódico + cache /alerts TTL 60s.
- anyio thread limiter reduzido (sidecar é I/O-bound, não CPU-bound).
"""
import logging
import os
from contextlib import asynccontextmanager
from datetime import datetime

import anyio
import httpx
from fastapi import FastAPI, HTTPException

from .recommender import ResourceRecommender

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")
logger = logging.getLogger("resma.ml")

API_URL = os.environ.get("RESMA_API_URL", "http://api:8080")


@asynccontextmanager
async def lifespan(app: FastAPI):
    logger.info("ML sidecar iniciando (dados via Go API %s)", API_URL)
    # Reduzir thread pool do anyio: o sidecar é I/O-bound (HTTP ao Go API)
    # e CPU-bound apenas em numpy/sklearn síncrono dentro do event loop.
    # Default 40 threads é excessivo — 10 basta e reduz memory footprint.
    limiter = anyio.to_thread.current_default_thread_limiter()
    limiter.total_tokens = int(os.environ.get("RESMA_THREAD_POOL_SIZE", "10"))
    # httpx.AsyncClient — não bloqueia o event loop durante chamadas HTTP
    # ao Go API. O sync Client anterior acumulava requests no backlog.
    async with httpx.AsyncClient(base_url=API_URL, timeout=60.0) as client:
        app.state.client = client
        app.state.recommender = ResourceRecommender(client)
        logger.info("ML sidecar pronto")
        yield
    logger.info("ML sidecar parando")


app = FastAPI(title="RESMA ML Sidecar", version="1.0.0", lifespan=lifespan)


@app.get("/health")
async def health():
    return {"status": "ok", "service": "ml", "timestamp": datetime.utcnow().isoformat()}


@app.get("/analyze")
async def analyze_all():
    """Analisar todos os serviços ativos."""
    try:
        rec = app.state.recommender
        results = await rec.analyze_all()
        return results
    except Exception as e:
        logger.error("erro em analyze_all: %s", e)
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/analyze/storage")
async def analyze_storage():
    """Análise de storage (volumes, imagens, reclaimable)."""
    try:
        rec = app.state.recommender
        result = await rec.analyze_storage()
        return result
    except Exception as e:
        logger.error("erro em analyze_storage: %s", e)
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/analyze/{service}")
async def analyze_service(service: str):
    """Analisar um serviço específico."""
    try:
        rec = app.state.recommender
        result = await rec.analyze(service)
        return result
    except Exception as e:
        logger.error("erro em analyze(%s): %s", service, e)
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/triggers")
async def evaluate_triggers():
    """Avaliar triggers de reanálise para todos os serviços."""
    try:
        rec = app.state.recommender
        triggers = await rec.evaluate_triggers()
        return triggers
    except Exception as e:
        logger.error("erro em evaluate_triggers: %s", e)
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/forecast/{service}")
async def forecast_service(service: str, days: int = 7):
    """Forecast de memória para um serviço."""
    try:
        rec = app.state.recommender
        result = await rec.forecast(service, days)
        return result
    except Exception as e:
        logger.error("erro em forecast(%s): %s", service, e)
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/alerts")
async def detect_alerts():
    """Detectar memory leaks e resource drifts para serviços ativos (0b.8).

    Retorna:
      {
        "leak_alerts":  [{ service, daily_growth_mb, r_squared }],
        "drift_alerts": [{ service, cpu_drift, mem_drift }]
      }

    Apenas serviços com métricas nos últimos ALERT_ACTIVE_WINDOW_MIN
    minutos são considerados (default 5 min, via env).
    """
    try:
        rec = app.state.recommender
        return await rec.detect_alerts()
    except Exception as e:
        logger.error("erro em detect_alerts: %s", e)
        raise HTTPException(status_code=500, detail=str(e))
