/**
 * HelpIcon — ícone de ajuda reutilizável com tooltip.
 *
 * Renderiza um HelpCircle que, ao passar o mouse, exibe um tooltip com título
 * e descrição. Usado para fornecer ajuda contextual sobre parâmetros, métricas
 * e configurações.
 *
 * Uso simples (apenas descrição):
 *   <HelpIcon text="Segundos entre coletas de métricas" />
 *
 * Uso com título + descrição:
 *   <HelpIcon title="OOM (Out of Memory)" text="Eventos de kill por falta de memória" />
 *
 * Props:
 *   - text:     texto principal da ajuda (obrigatório)
 *   - title:    título opcional exibido em negrito acima do texto
 *   - side:     lado do tooltip ("top" | "bottom" | "left" | "right"), default "top"
 *   - className: classes extras para o ícone (ex: "h-4 w-4")
 */
import { HelpCircle } from "lucide-react"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"

interface HelpIconProps {
  text: string
  title?: string
  side?: "top" | "bottom" | "left" | "right"
  className?: string
}

export function HelpIcon({ text, title, side = "top", className }: HelpIconProps) {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <HelpCircle
            className={`h-3.5 w-3.5 text-muted-foreground/60 hover:text-muted-foreground cursor-help ${className ?? ""}`}
          />
        </TooltipTrigger>
        <TooltipContent side={side} className="max-w-sm">
          {title ? (
            <div className="space-y-1">
              <p className="font-semibold">{title}</p>
              <p className="text-background/80">{text}</p>
            </div>
          ) : (
            <p>{text}</p>
          )}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
