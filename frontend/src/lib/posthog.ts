/**
 * posthog.ts — setup do PostHog (RUM — Real User Monitoring).
 *
 * Opt-in: só inicializa se VITE_POSTHOG_ENABLED=true e VITE_POSTHOG_KEY definida.
 * Privacy: autocapture=false (só eventos explícitos via trackRUM),
 *           disable_session_recording=true, opt_out_capturing_by_default=true.
 *
 * Spec: rum-posthog-setup.md §3
 */
import posthog from "posthog-js"

let initialized = false

export function initPosthog() {
  const enabled = import.meta.env.VITE_POSTHOG_ENABLED === "true"
  const key = import.meta.env.VITE_POSTHOG_KEY
  if (!enabled || !key) return

  posthog.init(key, {
    api_host: import.meta.env.VITE_POSTHOG_HOST || "https://app.posthog.com",
    autocapture: false,
    disable_session_recording: true,
    opt_out_capturing_by_default: true,
    persistence: "localStorage",
  })
  initialized = true
}

export function isPosthogInitialized() {
  return initialized
}

export function optInRUM() {
  if (!initialized) return
  posthog.opt_in_capturing()
}

export function optOutRUM() {
  if (!initialized) return
  posthog.opt_out_capturing()
}

export function identifyUser(userId: string, role: string) {
  if (!initialized) return
  posthog.identify(userId, { role })
}

export function resetUser() {
  if (!initialized) return
  posthog.reset()
}

export { posthog }
