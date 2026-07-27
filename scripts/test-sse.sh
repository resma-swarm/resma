#!/bin/sh
# Teste de integração SSE — verifica que eventos reais chegam nos tópicos.
# Uso: docker compose exec go-dev sh /src/scripts/test-sse.sh
set -e
BASE=http://localhost:8080
COOKIE_JAR=/tmp/sse-cookies.txt
rm -f "$COOKIE_JAR"

echo "=== 1. Login ==="
LOGIN=$(curl -s -X POST "$BASE/api/auth/login" -H "Content-Type: application/json" -d '{"username":"owner","password":"owner123"}')
TOKEN=$(echo "$LOGIN" | sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')
if [ -z "$TOKEN" ]; then
  echo "FAIL: não conseguiu extrair token. Login response: $LOGIN"
  exit 1
fi
echo "OK: token obtido (len=${#TOKEN})"

echo "=== 2. Criar sessão SSE (cookie) ==="
curl -s -X POST "$BASE/api/sse/session" -H "Authorization: Bearer $TOKEN" -c "$COOKIE_JAR" -o /dev/null -w "session_status=%{http_code}\n"

capture_topic() {
  topic="$1"
  expect_type="$2"
  echo "=== 3. Capturar eventos de /api/sse/$topic por 12s (espera event: $expect_type) ==="
  OUT=$(curl -s -N -m 12 -b "$COOKIE_JAR" "$BASE/api/sse/$topic" 2>/dev/null || true)
  echo "--- raw output (primeiros 1200 chars) ---"
  echo "$OUT" | head -c 1200
  echo ""
  echo "--- análise ---"
  if echo "$OUT" | grep -q "^event: connected"; then
    echo "OK: evento inicial 'connected' recebido"
  else
    echo "WARN: evento connected não encontrado"
  fi
  if echo "$OUT" | grep -q "^event: $expect_type"; then
    echo "PASS: evento '$expect_type' recebido no tópico $topic"
  else
    echo "FAIL: evento '$expect_type' NÃO recebido no tópico $topic"
  fi
  if echo "$OUT" | grep -q "^data:"; then
    echo "OK: payload data: presente"
  fi
}

capture_topic "metrics" "metrics"
echo ""
capture_topic "nodes" "nodes"
echo ""
capture_topic "services" "services"
echo ""
echo "=== 4. Capturar eventos de /api/sse/dashboard por 65s (ClusterInterval=60s) ==="
OUT=$(curl -s -N -m 65 -b "$COOKIE_JAR" "$BASE/api/sse/dashboard" 2>/dev/null || true)
echo "--- eventos recebidos ---"
echo "$OUT" | grep -E "^event: " | sort | uniq -c
if echo "$OUT" | grep -q "^event: cluster"; then
  echo "PASS: evento 'cluster' recebido no tópico dashboard"
else
  echo "FAIL: evento 'cluster' NÃO recebido no tópico dashboard (ClusterInterval=60s)"
fi
echo ""
echo "=== Teste concluído ==="
