import { Badge } from "@/components/ui/badge"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"

export const STATUS_CONFIG: Record<string, { label: string; variant: "default" | "secondary" | "destructive" | "outline"; dot: string }> = {
  online: { label: "Online", variant: "outline", dot: "bg-green-500" },
  offline: { label: "Offline", variant: "outline", dot: "bg-amber-500" },
  legado: { label: "Legado", variant: "outline", dot: "bg-muted-foreground" },
}

export function StatusBadge({ status, lastSeen }: { status: string; lastSeen?: string | null }) {
  const cfg = STATUS_CONFIG[status] ?? STATUS_CONFIG.legado
  const badge = (
    <Badge variant={cfg.variant} className="gap-1.5">
      <span className={`h-2 w-2 rounded-full ${cfg.dot}`} />
      {cfg.label}
    </Badge>
  )

  if (status === "legado" && lastSeen) {
    const date = new Date(lastSeen)
    const diffMs = Date.now() - date.getTime()
    const diffH = Math.floor(diffMs / (1000 * 60 * 60))
    const label = diffH > 24 ? `${Math.floor(diffH / 24)}d atrás` : `${diffH}h atrás`
    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <span>{badge}</span>
          </TooltipTrigger>
          <TooltipContent>
            <p>Última coleta: {label}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    )
  }

  return badge
}
