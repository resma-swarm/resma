/**
 * RumConsentBanner — banner de consentimento LGPD para RUM.
 *
 * Aparece apenas se VITE_POSTHOG_ENABLED=true e usuário ainda não consentiu.
 * Usuário pode Permitir ou Negar — decisão persistida em localStorage.
 *
 * Spec: rum-posthog-setup.md §3 (Consentimento LGPD)
 */
import { useEffect, useState } from "react"
import { shouldShowConsentBanner, setRUMConsent } from "@/lib/rum"
import { optInRUM, optOutRUM } from "@/lib/posthog"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { BarChart3 } from "lucide-react"

export function RumConsentBanner() {
  const [show, setShow] = useState(false)

  useEffect(() => {
    setShow(shouldShowConsentBanner())
  }, [])

  if (!show) return null

  const handleAllow = () => {
    setRUMConsent(true)
    optInRUM()
    setShow(false)
  }

  const handleDeny = () => {
    setRUMConsent(false)
    optOutRUM()
    setShow(false)
  }

  return (
    <div className="fixed bottom-4 right-4 z-50 max-w-sm animate-in slide-in-from-bottom-4">
      <Card className="border-primary/20 shadow-lg">
        <CardContent className="p-4 space-y-3">
          <div className="flex items-start gap-2">
            <BarChart3 className="h-5 w-5 text-primary shrink-0 mt-0.5" />
            <div className="space-y-1">
              <p className="text-sm font-medium">Métricas de uso anônimas</p>
              <p className="text-xs text-muted-foreground">
                O RESMA coleta interações anônimas (cliques, expansões) para melhorar
                as recomendações. Sem dados de payload — apenas eventos de UX.
              </p>
            </div>
          </div>
          <div className="flex gap-2 justify-end">
            <Button size="sm" variant="ghost" onClick={handleDeny}>
              Não, obrigado
            </Button>
            <Button size="sm" onClick={handleAllow}>
              Permitir
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
