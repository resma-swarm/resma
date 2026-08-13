"""Smoke test for the ML sidecar FastAPI app.

Validates that the app can be imported and the /health endpoint
returns 200 with the expected shape.
"""
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from fastapi.testclient import TestClient

from main import app


def test_health():
    """Health endpoint should return 200 with status=ok."""
    with TestClient(app) as client:
        resp = client.get("/health")
        assert resp.status_code == 200
        body = resp.json()
        assert body["status"] == "ok"
        assert body["service"] == "ml"
