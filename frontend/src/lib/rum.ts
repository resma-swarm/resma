/**
 * rum.ts — helper para tracking de eventos RUM (Right-Sizing Studio).
 *
 * trackRUM é no-op se PostHog não estiver inicializado ou usuário não consentiu.
 * Privacy: NÃO capturar valores de CPU/mem, nomes de serviços, tokens.
 *
 * Eventos definidos: rum-posthog-setup.md §4
 */
import { isPosthogInitialized, posthog } from "@/lib/posthog"

const CONSENT_KEY = "resma-rum-consent"

export function hasRUMConsent(): boolean {
  return localStorage.getItem(CONSENT_KEY) === "granted"
}

export function setRUMConsent(granted: boolean) {
  localStorage.setItem(CONSENT_KEY, granted ? "granted" : "denied")
}

export function shouldShowConsentBanner(): boolean {
  if (import.meta.env.VITE_POSTHOG_ENABLED !== "true") return false
  return localStorage.getItem(CONSENT_KEY) === null
}

/**
 * trackRUM envia um evento ao PostHog. No-op se:
 * - PostHog não inicializado (disabled)
 * - Usuário não consentiu
 * - PostHog indisponível (try/catch silencioso)
 */
export function trackRUM(event: string, properties?: Record<string, any>) {
  if (!isPosthogInitialized()) return
  if (!hasRUMConsent()) return
  try {
    posthog.capture(event, {
      ...properties,
      env: import.meta.env.MODE,
    })
  } catch {
    // silencioso — RUM não deve quebrar a UX
  }
}
