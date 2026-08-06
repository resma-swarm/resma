import { ReactNode } from "react"
import { LucideIcon } from "lucide-react"
import { CardContent } from "@/components/ui/card"

interface EmptyStateProps {
  icon: LucideIcon
  message: string
  className?: string
  /** Ação opcional (CTA) exibida abaixo da mensagem — botão/link para a próxima etapa. */
  action?: ReactNode
}

export function EmptyState({ icon: Icon, message, className, action }: EmptyStateProps) {
  return (
    <CardContent className={`flex flex-col items-center justify-center py-16 text-center ${className ?? ""}`}>
      <Icon className="h-10 w-10 text-muted-foreground mb-3" />
      <p className="text-sm text-muted-foreground">{message}</p>
      {action && <div className="mt-4">{action}</div>}
    </CardContent>
  )
}
