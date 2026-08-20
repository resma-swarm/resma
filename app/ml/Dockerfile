# RESMA ML Sidecar — Dockerfile
# Python slim com numpy, scipy, scikit-learn para análise de recursos
# Build context: repo root (docker-compose usa context: .)
FROM python:3.12-slim AS base

WORKDIR /app

# Instalar dependências do sistema para scipy/numpy
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc g++ libc-dev \
    && rm -rf /var/lib/apt/lists/*

# Copiar requirements e instalar (path relativo ao build context = repo root)
COPY app/ml/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copiar código do ML sidecar
COPY app/ml/ ./app/ml/

# MALLOC_TRIM_THRESHOLD_: faz glibc devolver memória ao SO após free()
# quando o buffer passa do threshold (64KB). Sem isso, BLAS do sklearn/
# numpy retém memória no heap do processo, causando crescimento monotônico
# até OOM. Default do glibc é 128KB em chunks grandes; 64KB é mais agressivo.
ENV MALLOC_TRIM_THRESHOLD_=65536

# Health check
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8081/health')" || exit 1

EXPOSE 8081

# --workers 1 --limit-concurrency 10: evita backlog de requests que mantém
# numpy arrays vivos na memória. O sidecar é chamado pelo Go API (1 caller)
# e pelo SSE broker; 10 conexões simultâneas é folga suficiente.
CMD ["uvicorn", "app.ml.main:app", "--host", "0.0.0.0", "--port", "8081", \
     "--workers", "1", "--limit-concurrency", "10"]
