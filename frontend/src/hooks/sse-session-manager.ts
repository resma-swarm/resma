/**
 * SSESessionManager — singleton que gerencia a sessão SSE (cookie sse_session).
 *
 * Responsabilidades:
 *   1. Criar sessão SSE (trocar JWT por cookie HttpOnly)
 *   2. Renovar proativamente antes da expiração (refresh before expiry)
 *   3. Renovar reativamente quando solicitado (antes de reconectar EventSource)
 *
 * Por que um singleton?
 *   - Múltiplas instâncias de useEventSource (Dashboard assina "metrics" + "dashboard")
 *     não devem chamar createSSESession independentemente — isso geraria chamadas
 *     duplicadas e cookies sobrepostos.
 *   - O timer de refresh proativo deve existir uma única vez, não por hook.
 *
 * Padrão de mercado (validado contra benchmark):
 *   - server-sent-events.com: "refresh the token before expiry and reconnect proactively"
 *   - Onyx PR #12903: "refreshes the token proactively, outside the transport"
 *   - ADR ricardoqmd/auth: "refreshes proactively at expiresAt - now - BUFFER (30s)"
 */
import { createSSESession } from "@/hooks/sse-session-api"

/** Buffer antes da expiração para renovar (30s — padrão de mercado) */
const REFRESH_BUFFER_MS = 30_000

/** Floor mínimo entre refreshes (evita hot-loop se tokens vierem com TTL muito curto) */
const MIN_REFRESH_DELAY_MS = 10_000

type SessionState = "idle" | "active" | "refreshing"

class SSESessionManager {
  private expiresAt: Date | null = null
  private refreshTimer: ReturnType<typeof setTimeout> | null = null
  private state: SessionState = "idle"
  private refreshPromise: Promise<void> | null = null

  /**
   * Inicia/renova a sessão SSE. Idempotente — se já ativa e não expirando,
   * não faz nada. Se expirando ou inativa, cria nova sessão e agenda refresh.
   */
  async ensureSession(): Promise<void> {
    // Já ativa e não próxima da expiração → nada a fazer
    if (this.state === "active" && this.expiresAt && !this.isExpiringSoon()) {
      return
    }

    // Refresh em andamento → aguardar
    if (this.refreshPromise) {
      await this.refreshPromise
      return
    }

    await this.doRefresh()
  }

  /**
   * Força renovação da sessão. Usado antes de reconectar EventSource
   * (quando onerror dispara, o cookie pode ter expirado — renovar sempre
   * é mais seguro que tentar detectar 401, que EventSource não expõe).
   */
  async refreshForReconnect(): Promise<void> {
    if (this.refreshPromise) {
      await this.refreshPromise
      return
    }
    await this.doRefresh()
  }

  /**
   * Limpa a sessão e cancela timer. Usado no logout.
   */
  stop(): void {
    if (this.refreshTimer) {
      clearTimeout(this.refreshTimer)
      this.refreshTimer = null
    }
    this.expiresAt = null
    this.state = "idle"
  }

  /**
   * Retorna true se a sessão está ativa e válida (não expirada).
   */
  isActive(): boolean {
    return this.state === "active" && this.expiresAt !== null && Date.now() < this.expiresAt.getTime()
  }

  private isExpiringSoon(): boolean {
    if (!this.expiresAt) return true
    const remaining = this.expiresAt.getTime() - Date.now()
    return remaining < REFRESH_BUFFER_MS
  }

  private async doRefresh(): Promise<void> {
    this.refreshPromise = this._doRefreshImpl()
    try {
      await this.refreshPromise
    } finally {
      this.refreshPromise = null
    }
  }

  private async _doRefreshImpl(): Promise<void> {
    this.state = "refreshing"
    try {
      this.expiresAt = await createSSESession()
      this.state = "active"
      this.scheduleProactiveRefresh()
    } catch (err) {
      this.state = "idle"
      this.expiresAt = null
      // Não relançar — SSE session é opcional (fallback para polling)
      // Apenas logar para debug
      console.warn("[SSESessionManager] Failed to create SSE session:", err)
    }
  }

  /**
   * Agenda refresh proativo: expiresAt - 30s, com floor de 10s.
   * Padrão validado contra ADR ricardoqmd/auth.
   */
  private scheduleProactiveRefresh(): void {
    if (this.refreshTimer) {
      clearTimeout(this.refreshTimer)
    }

    if (!this.expiresAt) return

    const now = Date.now()
    const proactiveDelay = this.expiresAt.getTime() - now - REFRESH_BUFFER_MS
    const delay = Math.max(proactiveDelay, MIN_REFRESH_DELAY_MS)

    this.refreshTimer = setTimeout(() => {
      this.doRefresh()
    }, delay)
  }
}

// Singleton — uma única instância para toda a app
export const sseSessionManager = new SSESessionManager()
