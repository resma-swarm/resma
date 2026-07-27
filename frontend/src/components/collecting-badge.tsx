/**
 * CollectingBadge — badge que reflete o estado REAL da coleta de métricas.
 *
 * Usa o hook useCollectionStatus que escuta eventos SSE do tópico "metrics"
 * e refetch do React Query (polling fallback).
 *
 * Três estados visuais:
 *
 *   - "collecting"   → ponto pulsante + "Coletando"
 *     - SSE:     verde (tempo real, baixa latência)
 *     - Polling: azul  (fallback, SSE caiu)
 *   - "idle"         → ponto amarelo estático + "Aguardando"
 *   - "reconnecting" → ponto vermelho pulsante + "Reconectando"
 *
 * O tooltip exibe a fonte ([sse] ou [pool]) + data/hora da última coleta,
 * no fuso horário do usuário (via toLocaleString).
 *
 * O estilo (h-8, rounded-md, border, text-xs) é idêntico ao botão de intervalo
 * do header top, para manter consistência visual.
 */
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { useCollectionStatus, type CollectionStatus, type CollectionSource } from "@/hooks/use-collection-status"

interface StatusConfig {
  label: string
  dotClass: string
  pulse: boolean
}

/** Config por status + source. SSE = verde, Polling = azul. */
const STATUS_CONFIG: Record<CollectionStatus, Record<CollectionSource, StatusConfig>> = {
  collecting: {
    sse: { label: "Coletando", dotClass: "bg-success", pulse: true },
    polling: { label: "Coletando", dotClass: "bg-primary", pulse: true },
  },
  idle: {
    sse: { label: "Aguardando", dotClass: "bg-warning", pulse: false },
    polling: { label: "Aguardando", dotClass: "bg-warning", pulse: false },
  },
  reconnecting: {
    sse: { label: "Reconectando", dotClass: "bg-destructive", pulse: true },
    polling: { label: "Reconectando", dotClass: "bg-destructive", pulse: true },
  },
}

function formatLastEvent(date: Date): string {
  return date.toLocaleString("pt-BR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  })
}

export function CollectingBadge() {
  const { status, source, lastEventAt } = useCollectionStatus()
  const config = STATUS_CONFIG[status][source]

  const sourceTag = source === "sse" ? "[sse]" : "[pool]"
  const tooltipText = lastEventAt
    ? `${sourceTag} Última coleta: ${formatLastEvent(lastEventAt)}`
    : status === "reconnecting"
      ? `${sourceTag} SSE desconectado — tentando reconectar`
      : `${sourceTag} Aguardando primeira coleta`

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="flex h-8 items-center gap-1.5 rounded-md border border-input bg-transparent px-2.5 text-xs shadow-sm cursor-default">
            <span className="relative flex h-2 w-2">
              <span
                className={`inline-flex h-full w-full rounded-full ${config.dotClass} ${config.pulse ? "animate-pulse-dot" : ""}`}
              />
            </span>
            <span>{config.label}</span>
          </div>
        </TooltipTrigger>
        <TooltipContent>{tooltipText}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
