/**
 * Funções de API para sessão SSE (cookie sse_session HttpOnly).
 *
 * Separadas do hook useEventSource para evitar import circular:
 *   use-event-source → sse-session-manager → sse-session-api (sem ciclo)
 *
 * Usa o api client (que tem refresh automático de JWT no 401) em vez de
 * fetch bruto, para que a renovação da sessão SSE funcione mesmo quando
 * o JWT access token já expirou (o api client faz refresh transparente).
 */
import { api } from "@/api/client"

interface SSESessionResponse {
  success: boolean
  expires: string
  message: string
}

/**
 * createSSESession — troca JWT bearer por cookie sse_session HttpOnly.
 * Deve ser chamado após login/onboarding e renovado antes de expirar.
 *
 * @returns Date com o timestamp de expiração da sessão (para agendar refresh)
 */
export async function createSSESession(): Promise<Date> {
  const data = await api.post<SSESessionResponse>("/sse/session")
  return new Date(data.expires)
}

/**
 * deleteSSESession — invalida cookie sse_session.
 * Deve ser chamado no logout.
 */
export async function deleteSSESession(): Promise<void> {
  await api.delete("/sse/session")
}
