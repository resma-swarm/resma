# RESMA ML Sidecar — Dockerfile
# Python slim com numpy, scipy, scikit-learn para análise de recursos
# Build context: repo root (docker-compose usa context: .)
FROM python:3.14-slim AS base

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

# Health check
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8081/health')" || exit 1

EXPOSE 8081

CMD ["uvicorn", "app.ml.main:app", "--host", "0.0.0.0", "--port", "8081"]
