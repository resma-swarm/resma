#!/usr/bin/env bash
#
# RESMA — Benchmark comparativo Go API vs Python (legacy)
#
# Mede:
#   - Latência p50, p95, p99 nos endpoints /health, /api/services, /api/dashboard
#   - RPS (requests per second) com wrk ou ab (Apache Bench)
#   - Uso de RAM (idle e sob carga SSE)
#   - SSE com 100 conexões simultâneas
#   - Compara com baseline Python (se disponível)
#
# Uso:
#   ./scripts/benchmark.sh [--go-port 8080] [--py-port 8000] [--duration 30] [--connections 100]
#
# Pré-requisitos:
#   - API Go rodando (docker stack deploy ou go run ./cmd/server)
#   - API Python legacy rodando (python -m backend.run) — opcional, para comparação
#   - wrk OU ab (Apache Bench) instalados
#   - curl instalado
#   - docker stats disponível (se rodando em containers)
#
# Saída:
#   - Console: tabela resumo
#   - scripts/benchmark-results.txt: relatório completo
#
# ---
# Fase 6 — Benchmarking
# Spec: docs/specs/oss/phase-6-benchmarking/spec.md

set -euo pipefail

# ============================================================================
# Configuração
# ============================================================================

GO_PORT="${GO_PORT:-8080}"
PY_PORT="${PY_PORT:-8000}"
DURATION="${DURATION:-30}"        # segundos por endpoint
CONNECTIONS="${CONNECTIONS:-100}" # conexões simultâneas
SSE_CONNECTIONS="${SSE_CONNECTIONS:-100}"
SSE_DURATION="${SSE_DURATION:-20}" # segundos para teste SSE
RESULTS_FILE="${RESULTS_FILE:-scripts/benchmark-results.txt}"
SKIP_PYTHON="${SKIP_PYTHON:-false}"
SKIP_SSE="${SKIP_SSE:-false}"

# Parse argumentos
while [[ $# -gt 0 ]]; do
    case "$1" in
        --go-port)      GO_PORT="$2"; shift 2 ;;
        --py-port)      PY_PORT="$2"; shift 2 ;;
        --duration)     DURATION="$2"; shift 2 ;;
        --connections)  CONNECTIONS="$2"; shift 2 ;;
        --sse-conns)    SSE_CONNECTIONS="$2"; shift 2 ;;
        --sse-duration) SSE_DURATION="$2"; shift 2 ;;
        --results)      RESULTS_FILE="$2"; shift 2 ;;
        --skip-python)  SKIP_PYTHON="true"; shift ;;
        --skip-sse)     SKIP_SSE="true"; shift ;;
        --help|-h)
            echo "Uso: $0 [opções]"
            echo ""
            echo "Opções:"
            echo "  --go-port PORT        Porta da API Go (default: 8080)"
            echo "  --py-port PORT        Porta da API Python legacy (default: 8000)"
            echo "  --duration SECS       Duração de cada teste HTTP (default: 30)"
            echo "  --connections N       Conexões simultâneas (default: 100)"
            echo "  --sse-conns N         Conexões SSE simultâneas (default: 100)"
            echo "  --sse-duration SECS   Duração do teste SSE (default: 20)"
            echo "  --results FILE        Arquivo de saída (default: scripts/benchmark-results.txt)"
            echo "  --skip-python         Não testar API Python"
            echo "  --skip-sse            Não testar SSE"
            exit 0 ;;
        *) echo "Opção desconhecida: $1" >&2; exit 1 ;;
    esac
done

GO_BASE="http://localhost:${GO_PORT}"
PY_BASE="http://localhost:${PY_PORT}"

# Endpoints para benchmark (caminhos idênticos em Go e Python)
ENDPOINTS=(
    "/health"
    "/api/services"
    "/api/dashboard"
)

# ============================================================================
# Helpers
# ============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log()     { echo -e "${BLUE}[benchmark]${NC} $*"; }
ok()      { echo -e "${GREEN}[OK]${NC} $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
err()     { echo -e "${RED}[ERR]${NC} $*" >&2; }

# Detecta ferramenta de benchmark disponível (wrk preferido, ab como fallback)
detect_bench_tool() {
    if command -v wrk &>/dev/null; then
        BENCH_TOOL="wrk"
        log "Ferramenta de benchmark: wrk"
    elif command -v ab &>/dev/null; then
        BENCH_TOOL="ab"
        log "Ferramenta de benchmark: ab (Apache Bench)"
    else
        err "Nem wrk nem ab estão instalados."
        err "Instale um deles:"
        err "  Ubuntu/Debian: sudo apt install wrk   (ou)   sudo apt install apache2-utils"
        err "  macOS:         brew install wrk       (ou)   brew install httpd"
        err "  Alpine:        apk add wrk            (ou)   apk add apache2-ab"
        exit 1
    fi
}

# Verifica se um endpoint responde
check_endpoint() {
    local url="$1"
    local name="$2"
    local code
    code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$url" 2>/dev/null || echo "000")
    if [[ "$code" == "200" || "$code" == "401" ]]; then
        # 401 é esperado em endpoints que requerem JWT — conta como "respondendo"
        ok "$name respondendo (HTTP $code)"
        return 0
    else
        warn "$name não respondeu adequadamente (HTTP $code) — pulando"
        return 1
    fi
}

# ============================================================================
# Benchmark HTTP com wrk
# ============================================================================
# wrk output exemplo:
#   Requests/sec: 12500.50
#   Latency:   1.23ms (p50), 5.67ms (p99)
#
# Para percentis, wrk usa --latency que mostra:
#   Latency Distribution
#     50%    1.23ms
#     75%    2.45ms
#     90%    4.10ms
#     99%    8.90ms

bench_wrk() {
    local url="$1"
    local threads=4
    local result

    # wrk com --latency para obter percentis
    result=$(wrk -t${threads} -c${CONNECTIONS} -d${DURATION}s --latency "$url" 2>&1)

    # Extrair RPS
    local rps
    rps=$(echo "$result" | grep "Requests/sec" | awk '{printf "%.1f", $2}')

    # Extrair latência média
    local latency_avg
    latency_avg=$(echo "$result" | grep "Latency" | head -1 | awk '{print $2}')

    # Extrair percentis (50%, 95%, 99%)
    local p50 p95 p99
    p50=$(echo "$result" | grep "50%" | awk '{print $2}')
    p95=$(echo "$result" | grep -E "^\s+95%" | awk '{print $2}')
    p99=$(echo "$result" | grep "99%" | awk '{print $2}')

    # Fallback: se wrk não reportar 95%, interpolar entre 90% e 99%
    if [[ -z "$p95" ]]; then
        local p90
        p90=$(echo "$result" | grep "90%" | awk '{print $2}')
        p95="${p90:-N/A}"
    fi

    # Transfer rate
    local transfer
    transfer=$(echo "$result" | grep "Transfer/sec" | awk '{print $2}')

    echo "${rps:-N/A}|${latency_avg:-N/A}|${p50:-N/A}|${p95:-N/A}|${p99:-N/A}|${transfer:-N/A}"
}

# ============================================================================
# Benchmark HTTP com ab (Apache Bench)
# ============================================================================
# ab output exemplo:
#   Requests per second:    12500.50 [#/sec] (mean)
#   Time per request:       1.234 [ms] (mean)
#   Percentage of the requests served within a certain time (ms)
#     50%      1.234
#     95%      5.678
#     99%      8.901

bench_ab() {
    local url="$1"
    local total_requests=$((CONNECTIONS * 100))
    local result

    # ab com -e para CSV de percentis e -g para gnuplot (não usado)
    # Usamos -r para não sair em socket errors
    result=$(ab -n "$total_requests" -c "$CONNECTIONS" -r -e /dev/stdout "$url" 2>/dev/null <<'EOF'
EOF
)

    # O -e /dev/stdout gera CSV de percentis. Mas é mais fácil parsear stdout normal.
    # Re-executar sem -e para obter texto legível
    result=$(ab -n "$total_requests" -c "$CONNECTIONS" -r "$url" 2>&1)

    local rps
    rps=$(echo "$result" | grep "Requests per second" | awk '{printf "%.1f", $4}')

    local latency_avg
    latency_avg=$(echo "$result" | grep "Time per request" | head -1 | awk '{print $5 "ms"}')

    # Percentis do ab
    local p50 p95 p99
    p50=$(echo "$result" | grep "50%" | awk '{print $2 "ms"}')
    p95=$(echo "$result" | grep "95%" | awk '{print $2 "ms"}')
    p99=$(echo "$result" | grep "99%" | awk '{print $2 "ms"}')

    local transfer
    transfer=$(echo "$result" | grep "Transfer rate" | awk '{print $3 "KB/s"}')

    echo "${rps:-N/A}|${latency_avg:-N/A}|${p50:-N/A}|${p95:-N/A}|${p99:-N/A}|${transfer:-N/A}"
}

# Wrapper unificado — chama wrk ou ab conforme disponível
bench_endpoint() {
    local url="$1"
    if [[ "$BENCH_TOOL" == "wrk" ]]; then
        bench_wrk "$url"
    else
        bench_ab "$url"
    fi
}

# ============================================================================
# Medição de RAM
# ============================================================================

# Tenta obter RAM do container via docker stats
# $1 = nome do serviço/container, $2 = duração da amostra em segundos
measure_ram_docker() {
    local container_pattern="$1"
    local sample_secs="${2:-5}"
    local ram_samples=()

    # Coletar amostras de docker stats (no-stream para uma leitura)
    for _ in $(seq 1 3); do
        local mem
        mem=$(docker stats --no-stream --format "{{.MemUsage}}" --filter "name=${container_pattern}" 2>/dev/null | head -1 | awk '{print $1}')
        if [[ -n "$mem" && "$mem" != "" ]]; then
            ram_samples+=("$mem")
        fi
        sleep $((sample_secs / 3))
    done

    if [[ ${#ram_samples[@]} -eq 0 ]]; then
        echo "N/A"
    else
        # Retornar a primeira amostra (idle) — para sob carga, pegar a maior
        echo "${ram_samples[0]}"
    fi
}

# Mede RAM do processo pelo PID (fallback se não estiver em Docker)
measure_ram_pid() {
    local pid="$1"
    if [[ ! -d "/proc/$pid" ]]; then
        echo "N/A"
        return
    fi
    # VmRSS em kB
    local rss
    rss=$(grep VmRSS "/proc/$pid/status" 2>/dev/null | awk '{print $2}')
    if [[ -n "$rss" ]]; then
        echo "$((rss / 1024))MB"
    else
        echo "N/A"
    fi
}

# ============================================================================
# Teste SSE — 100 conexões simultâneas
# ============================================================================
# RESMA usa SSE (Server-Sent Events) via net/http + http.Flusher.
# Endpoints SSE: /api/sse/metrics, /api/sse/dashboard, /api/sse/events, etc.
# Autenticação: cookie sse_session HttpOnly OU header Authorization.
#
# Para o benchmark, usamos curl em background mantendo conexões abertas.
# Medimos: número de conexões estabelecidas, RAM durante, eventos recebidos.

bench_sse() {
    local base_url="$1"
    local label="$2"
    local sse_endpoint="${base_url}/api/sse/metrics"
    local pids=()
    local temp_dir
    temp_dir=$(mktemp -d)
    local connected=0
    local failed=0

    log "Iniciando teste SSE: ${SSE_CONNECTIONS} conexões por ${SSE_DURATION}s contra ${sse_endpoint}"

    # Nota: SSE requer auth. Se houver JWT disponível, usar.
    # Para benchmark, tentamos sem auth primeiro — se 401, registramos.
    # Em ambiente de teste, pode-se desabilitar auth ou usar um token de teste.
    local auth_header=""
    if [[ -n "${RESMA_BENCH_JWT:-}" ]]; then
        auth_header="-H \"Authorization: Bearer ${RESMA_BENCH_JWT}\""
    fi

    # Lançar conexões SSE em background
    for i in $(seq 1 "$SSE_CONNECTIONS"); do
        (
            if [[ -n "$auth_header" ]]; then
                eval curl -s -N --max-time "$((SSE_DURATION + 5))" "$auth_header" "$sse_endpoint" > "${temp_dir}/sse_${i}.out" 2>/dev/null
            else
                curl -s -N --max-time "$((SSE_DURATION + 5))" "$sse_endpoint" > "${temp_dir}/sse_${i}.out" 2>/dev/null
            fi
        ) &
        pids+=($!)
    done

    # Aguardar estabelecimento das conexões
    sleep 2

    # Contar conexões ativas (curl processes rodando)
    connected=0
    for pid in "${pids[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            ((connected++))
        fi
    done

    log "Conexões SSE ativas (${label}): ${connected}/${SSE_CONNECTIONS}"

    # Medir RAM durante o período de conexões ativas
    sleep "$((SSE_DURATION - 4))"

    # Coletar RAM (tentar docker, fallback para contagem de processos)
    local ram_during
    if [[ "${label}" == "Go" ]]; then
        ram_during=$(measure_ram_docker "resma-api\|api" 3 2>/dev/null || echo "N/A")
    else
        ram_during=$(measure_ram_docker "resma-python\|backend" 3 2>/dev/null || echo "N/A")
    fi

    # Contar eventos recebidos (linhas começando com "data:" nos arquivos de output)
    local total_events=0
    for i in $(seq 1 "$SSE_CONNECTIONS"); do
        if [[ -f "${temp_dir}/sse_${i}.out" ]]; then
            local events
            events=$(grep -c "^data:" "${temp_dir}/sse_${i}.out" 2>/dev/null || echo "0")
            total_events=$((total_events + events))
        fi
    done

    # Aguardar todas as conexões terminarem
    for pid in "${pids[@]}"; do
        kill "$pid" 2>/dev/null || true
        wait "$pid" 2>/dev/null || true
    done

    # Limpar
    rm -rf "$temp_dir"

    failed=$((SSE_CONNECTIONS - connected))
    echo "${connected}|${failed}|${total_events}|${ram_during}"
}

# ============================================================================
# Relatório
# ============================================================================

write_header() {
    {
        echo "================================================================"
        echo "  RESMA — Benchmark Report"
        echo "  Data: $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
        echo "  Ferramenta: ${BENCH_TOOL}"
        echo "  Duração por endpoint: ${DURATION}s"
        echo "  Conexões simultâneas: ${CONNECTIONS}"
        echo "  SSE: ${SSE_CONNECTIONS} conexões x ${SSE_DURATION}s"
        echo "================================================================"
        echo ""
    } | tee -a "$RESULTS_FILE"
}

write_section() {
    echo "" | tee -a "$RESULTS_FILE"
    echo "--- $1 ---" | tee -a "$RESULTS_FILE"
    echo "" | tee -a "$RESULTS_FILE"
}

# ============================================================================
# Main
# ============================================================================

main() {
    log "Iniciando benchmark RESMA"
    log "Go API: ${GO_BASE}"
    if [[ "$SKIP_PYTHON" != "true" ]]; then
        log "Python API: ${PY_BASE}"
    fi
    echo ""

    detect_bench_tool

    # Limpar arquivo de resultados
    echo "" > "$RESULTS_FILE"
    write_header

    # --- Verificar disponibilidade ---
    write_section "Verificação de disponibilidade"

    local go_available=true
    check_endpoint "${GO_BASE}/health" "Go API" || go_available=false

    local py_available=false
    if [[ "$SKIP_PYTHON" != "true" ]]; then
        if check_endpoint "${PY_BASE}/health" "Python API"; then
            py_available=true
        fi
    fi

    if [[ "$go_available" == "false" ]]; then
        err "Go API não está disponível em ${GO_BASE}. Abortando."
        err "Inicie com: .\scripts\deploy-swarm.ps1 ou docker stack deploy -c docker-compose.swarm.yml resma"
        exit 1
    fi

    # --- RAM Idle ---
    write_section "RAM — Idle (sem carga)"

    local go_ram_idle py_ram_idle
    go_ram_idle=$(measure_ram_docker "resma-api\|api" 5 2>/dev/null || echo "N/A")
    echo "Go API RAM (idle):      ${go_ram_idle}" | tee -a "$RESULTS_FILE"

    if [[ "$py_available" == "true" ]]; then
        py_ram_idle=$(measure_ram_docker "backend\|resma-python" 5 2>/dev/null || echo "N/A")
        echo "Python API RAM (idle):  ${py_ram_idle}" | tee -a "$RESULTS_FILE"
    else
        py_ram_idle="N/A (não disponível)"
        echo "Python API RAM (idle):  ${py_ram_idle}" | tee -a "$RESULTS_FILE"
    fi

    # --- Benchmark HTTP por endpoint ---
    write_section "Benchmark HTTP — Latência e RPS"

    # Tabela
    printf "%-20s | %-8s | %-12s | %-10s | %-10s | %-10s | %-10s\n" \
        "Endpoint" "API" "RPS" "Latency avg" "p50" "p95" "p99" | tee -a "$RESULTS_FILE"
    printf "%s\n" "$(printf '%.0s-' {1..95})" | tee -a "$RESULTS_FILE"

    # Resultados para comparação final
    declare -A go_results
    declare -A py_results

    for endpoint in "${ENDPOINTS[@]}"; do
        local go_url="${GO_BASE}${endpoint}"
        local py_url="${PY_BASE}${endpoint}"

        # Go
        local go_res
        if check_endpoint "$go_url" "Go ${endpoint}" 2>/dev/null; then
            log "Benchmarking Go ${endpoint} (${DURATION}s, ${CONNECTIONS} conexões)..."
            go_res=$(bench_endpoint "$go_url")
        else
            go_res="N/A|N/A|N/A|N/A|N/A|N/A"
        fi
        go_results["$endpoint"]="$go_res"

        IFS='|' read -r go_rps go_lat go_p50 go_p95 go_p99 go_xfer <<< "$go_res"
        printf "%-20s | %-8s | %-12s | %-10s | %-10s | %-10s | %-10s\n" \
            "$endpoint" "Go" "${go_rps}" "${go_lat}" "${go_p50}" "${go_p95}" "${go_p99}" | tee -a "$RESULTS_FILE"

        # Python (se disponível)
        if [[ "$py_available" == "true" ]]; then
            local py_res
            if check_endpoint "$py_url" "Python ${endpoint}" 2>/dev/null; then
                log "Benchmarking Python ${endpoint} (${DURATION}s, ${CONNECTIONS} conexões)..."
                py_res=$(bench_endpoint "$py_url")
            else
                py_res="N/A|N/A|N/A|N/A|N/A|N/A"
            fi
            py_results["$endpoint"]="$py_res"

            IFS='|' read -r py_rps py_lat py_p50 py_p95 py_p99 py_xfer <<< "$py_res"
            printf "%-20s | %-8s | %-12s | %-10s | %-10s | %-10s | %-10s\n" \
                "$endpoint" "Python" "${py_rps}" "${py_lat}" "${py_p50}" "${py_p95}" "${py_p99}" | tee -a "$RESULTS_FILE"
        fi
        echo "" | tee -a "$RESULTS_FILE"
    done

    # --- Teste SSE ---
    if [[ "$SKIP_SSE" != "true" ]]; then
        write_section "SSE — Conexões Simultâneas"

        printf "%-12s | %-18s | %-15s | %-18s | %-15s\n" \
            "API" "Conexões ativas" "Conexões falharam" "Eventos recebidos" "RAM sob carga" | tee -a "$RESULTS_FILE"
        printf "%s\n" "$(printf '%.0s-' {1..85})" | tee -a "$RESULTS_FILE"

        # Go SSE
        log "Testando SSE Go (${SSE_CONNECTIONS} conexões)..."
        local go_sse_res
        go_sse_res=$(bench_sse "$GO_BASE" "Go")
        IFS='|' read -r go_sse_conn go_sse_fail go_sse_events go_sse_ram <<< "$go_sse_res"
        printf "%-12s | %-18s | %-15s | %-18s | %-15s\n" \
            "Go" "${go_sse_conn}/${SSE_CONNECTIONS}" "${go_sse_fail}" "${go_sse_events}" "${go_sse_ram}" | tee -a "$RESULTS_FILE"

        # Python SSE (se disponível)
        if [[ "$py_available" == "true" ]]; then
            log "Testando SSE Python (${SSE_CONNECTIONS} conexões)..."
            local py_sse_res
            py_sse_res=$(bench_sse "$PY_BASE" "Python")
            IFS='|' read -r py_sse_conn py_sse_fail py_sse_events py_sse_ram <<< "$py_sse_res"
            printf "%-12s | %-18s | %-15s | %-18s | %-15s\n" \
                "Python" "${py_sse_conn}/${SSE_CONNECTIONS}" "${py_sse_fail}" "${py_sse_events}" "${py_sse_ram}" | tee -a "$RESULTS_FILE"
        fi
    fi

    # --- Resumo comparativo ---
    write_section "Resumo Comparativo Go vs Python"

    {
        echo "Métrica                    | Go API              | Python API (legacy)"
        echo "---------------------------|---------------------|---------------------"
        echo "RAM idle                   | ${go_ram_idle:-N/A}        | ${py_ram_idle:-N/A}"
        echo "RAM @ ${SSE_CONNECTIONS} SSE conexões      | ${go_sse_ram:-N/A}        | ${py_sse_ram:-N/A}"

        # /health
        IFS='|' read -r h_rps h_lat h_p50 h_p95 h_p99 _ <<< "${go_results[/health]:-N/A|N/A|N/A|N/A|N/A|N/A}"
        IFS='|' read -r ph_rps ph_lat ph_p50 ph_p95 ph_p99 _ <<< "${py_results[/health]:-N/A|N/A|N/A|N/A|N/A|N/A}"
        echo "/health RPS                | ${h_rps}             | ${ph_rps}"
        echo "/health p50                | ${h_p50}             | ${ph_p50}"
        echo "/health p99                | ${h_p99}             | ${ph_p99}"

        # /api/services
        IFS='|' read -r s_rps s_lat s_p50 s_p95 s_p99 _ <<< "${go_results[/api/services]:-N/A|N/A|N/A|N/A|N/A|N/A}"
        IFS='|' read -r ps_rps ps_lat ps_p50 ps_p95 ps_p99 _ <<< "${py_results[/api/services]:-N/A|N/A|N/A|N/A|N/A|N/A}"
        echo "/api/services RPS          | ${s_rps}             | ${ps_rps}"
        echo "/api/services p50          | ${s_p50}             | ${ps_p50}"
        echo "/api/services p99          | ${s_p99}             | ${ps_p99}"

        # /api/dashboard
        IFS='|' read -r d_rps d_lat d_p50 d_p95 d_p99 _ <<< "${go_results[/api/dashboard]:-N/A|N/A|N/A|N/A|N/A|N/A}"
        IFS='|' read -r pd_rps pd_lat pd_p50 pd_p95 pd_p99 _ <<< "${py_results[/api/dashboard]:-N/A|N/A|N/A|N/A|N/A|N/A}"
        echo "/api/dashboard RPS         | ${d_rps}             | ${pd_rps}"
        echo "/api/dashboard p50         | ${d_p50}             | ${pd_p50}"
        echo "/api/dashboard p99         | ${d_p99}             | ${pd_p99}"

        echo ""
        echo "SSE conexões sustentadas   | ${go_sse_conn:-N/A}/${SSE_CONNECTIONS}       | ${py_sse_conn:-N/A}/${SSE_CONNECTIONS}"
        echo "SSE eventos recebidos      | ${go_sse_events:-N/A}              | ${py_sse_events:-N/A}"
    } | tee -a "$RESULTS_FILE"

    # --- Notas finais ---
    write_section "Notas"

    {
        echo "- Resultados gerados com ${BENCH_TOOL} em $(date -u '+%Y-%m-%d')"
        echo "- CPU/RAM do host influencia resultados. Para comparação justa,"
        echo "  rodar ambas as APIs no mesmo hardware, mesma carga, sem outros processos pesados."
        echo "- SSE: conexões usam curl --max-time. Eventos contados como linhas 'data:'."
        echo "- Endpoints /api/services e /api/dashboard requerem JWT. Se retornaram 401,"
        echo "  o benchmark mediu apenas o overhead de rejeição, não o processamento real."
        echo "  Para benchmark completo, definir RESMA_BENCH_JWT com um token válido:"
        echo "    export RESMA_BENCH_JWT=\"<seu-jwt-aqui>\""
        echo "- Go API: net/http stdlib, sem framework, DuckDB embedded (go-duckdb)"
        echo "- Python API: FastAPI + uvicorn + aiodocker + DuckDB"
        echo ""
        echo "Para reproduzir:"
        echo "  1. Iniciar Go API:  .\scripts\deploy-swarm.ps1"
        echo "  2. Iniciar Python:  python -m backend.run  (opcional)"
        echo "  3. Executar:        ./scripts/benchmark.sh"
        echo ""
        echo "Relatório completo: ${RESULTS_FILE}"
    } | tee -a "$RESULTS_FILE"

    echo ""
    ok "Benchmark concluído! Resultados em: ${RESULTS_FILE}"
}

main "$@"
