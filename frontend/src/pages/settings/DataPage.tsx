import { useState, useEffect } from "react"
import { api } from "@/api/client"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
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

interface PruneCard {
  key: string
  label: string
  endpoint: string
  countKey: keyof PrunePreview
  description: string
  destructive: boolean
}

const PRUNE_CARDS: PruneCard[] = [
  {
    key: "services-stale",
    label: "Services Stale",
    endpoint: "/prune/services-stale",
    countKey: "services_stale",
    description: "Services marcados como stale (sem heartbeat)",
    destructive: false,
  },
  {
    key: "nodes-stale",
    label: "Nodes Stale",
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
    description: "Tasks com status removed/orphaned",
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
    label: "Change Log",
    endpoint: "/prune/change-log",
    countKey: "change_log",
    description: "TODO o histórico de mudanças (irreversível)",
    destructive: true,
  },
  {
    key: "volume-metrics",
    label: "Volume Metrics",
    endpoint: "/prune/volume-metrics",
    countKey: "volume_metrics",
    description: "TODAS as métricas de volumes (irreversível)",
    destructive: true,
  },
]

export function DataPage() {
  const [preview, setPreview] = useState<PrunePreview | null>(null)
  const [loading, setLoading] = useState(true)
  const [confirmed, setConfirmed] = useState(false)

  const loadPreview = async () => {
    try {
      const data = await api.get<PrunePreview>("/prune/preview")
      setPreview(data)
    } catch (e) {
      toast.error("Erro ao carregar preview: " + (e as Error).message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadPreview()
  }, [])

  const handlePrune = async (card: PruneCard) => {
    try {
      const data = await api.post<{ deleted: number }>(card.endpoint, { dry_run: false })
      toast.success(`${card.label}: ${data.deleted} linha(s) removida(s)`)
      setConfirmed(false)
      loadPreview()
    } catch (e) {
      toast.error("Erro: " + (e as Error).message)
    }
  }

  const handleDryRun = async (card: PruneCard) => {
    try {
      const data = await api.post<{ would_delete: number }>(card.endpoint, { dry_run: true })
      toast.info(`${card.label}: ${data.would_delete} linha(s) seriam removidas`)
    } catch (e) {
      toast.error("Erro: " + (e as Error).message)
    }
  }

  if (loading) return <div className="text-muted-foreground">Carregando...</div>

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
                {preview ? preview[card.countKey].toLocaleString() : "..."}
                <span className="text-sm font-normal text-muted-foreground ml-1.5">linha(s)</span>
              </div>

              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handleDryRun(card)}
                >
                  Dry-run
                </Button>

                {card.destructive ? (
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button variant="destructive" size="sm" disabled={preview?.[card.countKey] === 0}>
                        <Trash2 className="h-4 w-4" />
                        Prune
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Confirmar prune de {card.label}?</AlertDialogTitle>
                        <AlertDialogDescription>
                          Esta operação é IRREVERSÍVEL. {preview?.[card.countKey]} linha(s) serão removidas permanentemente.
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
                        <AlertDialogCancel onClick={() => setConfirmed(false)}>Cancelar</AlertDialogCancel>
                        <AlertDialogAction
                          onClick={() => handlePrune(card)}
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
                    disabled={preview?.[card.countKey] === 0}
                    onClick={() => handlePrune(card)}
                  >
                    <Trash2 className="h-4 w-4" />
                    Prune
                  </Button>
                )}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
