import { LucideIcon } from "lucide-react"
import { CardContent } from "@/components/ui/card"

interface EmptyStateProps {
  icon: LucideIcon
  message: string
  className?: string
}

export function EmptyState({ icon: Icon, message, className }: EmptyStateProps) {
  return (
    <CardContent className={`flex flex-col items-center justify-center py-16 text-center ${className ?? ""}`}>
      <Icon className="h-10 w-10 text-muted-foreground mb-3" />
      <p className="text-sm text-muted-foreground">{message}</p>
    </CardContent>
  )
}
