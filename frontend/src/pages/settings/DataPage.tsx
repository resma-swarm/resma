import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/api/client"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"
import { Database, Trash2, AlertTriangle } from "lucide-react"
import { toast } from "sonner"

interface PrunePreview {
  services_stale: number
  nodes_stale: number
  tasks_orphan: number
  metrics: number
  change_log: number
  volume_metrics: number
}

interface PruneCardConfig {
  key: string
  label: string
  endpoint: string
  countKey: keyof PrunePreview
  description: string
  destructive: boolean
}

// PRUNE_CARDS — labels em PT-BR (C7: i18n consistente com o restante do app).
const PRUNE_CARDS: PruneCardConfig[] = [
  {
    key: "services-stale",
    label: "Serviços Obsoletos",
    endpoint: "/prune/services-stale",
    countKey: "services_stale",
    description: "Serviços marcados como stale (sem heartbeat)",
    destructive: false,
  },
  {
    key: "nodes-stale",
    label: "Nodes Obsoletos",
    endpoint: "/prune/nodes-stale",
    countKey: "nodes_stale",
    description: "Nodes marcados como stale (sem heartbeat)",
    destructive: false,
  },
  {
    key: "tasks-orphan",
    label: "Tasks Órfãs",
    endpoint: "/prune/tasks-orphan",
    countKey: "tasks_orphan",
    description: "Tasks com status removida/órfã",
    destructive: false,
  },
  {
    key: "metrics",
    label: "Métricas",
    endpoint: "/prune/metrics",
    countKey: "metrics",
    description: "TODAS as métricas coletadas (irreversível)",
    destructive: true,
  },
  {
    key: "change-log",
    label: "Histórico de Mudanças",
    endpoint: "/prune/change-log",
    countKey: "change_log",
    description: "TODO o histórico de mudanças (irreversível)",
    destructive: true,
  },
  {
    key: "volume-metrics",
    label: "Métricas de Volumes",
    endpoint: "/prune/volume-metrics",
    countKey: "volume_metrics",
    description: "TODAS as métricas de volumes (irreversível)",
    destructive: true,
  },
]

// PruneCardItem — cada card tem seu próprio estado `confirmed`, isolando
// o consentimento por operação destrutiva. Resolve B1: antes o estado era
// compartilhado e um checkbox marcado em um dialog vazava para o próximo.
function PruneCardItem({
  card,
  count,
  onPrune,
  onDryRun,
  pruning,
}: {
  card: PruneCardConfig
  count: number | undefined
  onPrune: (card: PruneCardConfig) => void
  onDryRun: (card: PruneCardConfig) => void
  pruning: boolean
}) {
  // Estado isolado por card — resetado quando o dialog fecha (onOpenChange).
  const [confirmed, setConfirmed] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)

  const handlePrune = () => {
    onPrune(card)
    setConfirmed(false)
    setDialogOpen(false)
  }

  return (
    <Card key={card.key}>
      <CardHeader>
        <CardTitle className="text-base flex items-center gap-2">
          {card.destructive && <AlertTriangle className="h-4 w-4 text-warning" />}
          {card.label}
        </CardTitle>
        <CardDescription>{card.description}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="text-2xl font-bold">
          {count !== undefined ? count.toLocaleString("pt-BR") : "..."}
          <span className="text-sm font-normal text-muted-foreground ml-1.5">linha(s)</span>
        </div>

        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => onDryRun(card)}
            disabled={pruning}
          >
            Simular
          </Button>

          {card.destructive ? (
            <AlertDialog
              open={dialogOpen}
              onOpenChange={(open) => {
                setDialogOpen(open)
                // Resetar consentimento sempre que o dialog fecha —
                // seja por Cancelar, ESC ou click fora.
                if (!open) setConfirmed(false)
              }}
            >
              <AlertDialogTrigger asChild>
                <Button variant="destructive" size="sm" disabled={count === 0 || pruning}>
                  <Trash2 className="h-4 w-4" />
                  Limpar
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Confirmar limpeza de {card.label}?</AlertDialogTitle>
                  <AlertDialogDescription>
                    Esta operação é IRREVERSÍVEL. {count?.toLocaleString("pt-BR")} linha(s) serão removidas permanentemente.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <div className="flex items-center gap-2 py-2">
                  <Checkbox
                    id={`confirm-${card.key}`}
                    checked={confirmed}
                    onCheckedChange={(v) => setConfirmed(v === true)}
                  />
                  <Label htmlFor={`confirm-${card.key}`} className="text-sm">
                    Entendo que esta operação é irreversível
                  </Label>
                </div>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancelar</AlertDialogCancel>
                  <AlertDialogAction
                    onClick={handlePrune}
                    disabled={!confirmed}
                    className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                  >
                    Excluir
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          ) : (
            <Button
              variant="destructive"
              size="sm"
              disabled={count === 0 || pruning}
              onClick={() => onPrune(card)}
            >
              <Trash2 className="h-4 w-4" />
              Limpar
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

export function DataPage() {
  const queryClient = useQueryClient()

  const { data: preview, isLoading } = useQuery<PrunePreview>({
    queryKey: ["prune-preview"],
    queryFn: () => api.get<PrunePreview>("/prune/preview"),
  })

  const pruneMutation = useMutation({
    mutationFn: (card: PruneCardConfig) =>
      api.post<{ deleted: number }>(card.endpoint, { dry_run: false }),
    onSuccess: (data, card) => {
      toast.success(`${card.label}: ${data.deleted.toLocaleString("pt-BR")} linha(s) removida(s)`)
      queryClient.invalidateQueries({ queryKey: ["prune-preview"] })
    },
    onError: (e) => toast.error("Erro: " + (e as Error).message),
  })

  const dryRunMutation = useMutation({
    mutationFn: (card: PruneCardConfig) =>
      api.post<{ would_delete: number }>(card.endpoint, { dry_run: true }),
    onSuccess: (data, card) => {
      toast.info(`${card.label}: ${data.would_delete.toLocaleString("pt-BR")} linha(s) seriam removidas`)
    },
    onError: (e) => toast.error("Erro: " + (e as Error).message),
  })

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Card>
          <CardHeader>
            <Skeleton className="h-6 w-48" />
            <Skeleton className="h-4 w-80" />
          </CardHeader>
        </Card>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Card key={i}>
              <CardHeader>
                <Skeleton className="h-5 w-32" />
                <Skeleton className="h-4 w-48" />
              </CardHeader>
              <CardContent className="space-y-3">
                <Skeleton className="h-8 w-24" />
                <div className="flex gap-2">
                  <Skeleton className="h-8 w-20" />
                  <Skeleton className="h-8 w-20" />
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Database className="h-5 w-5" />
            Gestão de Dados
          </CardTitle>
          <CardDescription>
            Visualize e limpe dados acumulados. Operações destrutivas exigem confirmação.
          </CardDescription>
        </CardHeader>
      </Card>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {PRUNE_CARDS.map((card) => (
          <PruneCardItem
            key={card.key}
            card={card}
            count={preview ? preview[card.countKey] : undefined}
            onPrune={(c) => pruneMutation.mutate(c)}
            onDryRun={(c) => dryRunMutation.mutate(c)}
            pruning={pruneMutation.isPending || dryRunMutation.isPending}
          />
        ))}
      </div>
    </div>
  )
}
