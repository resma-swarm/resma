/**
 * RollbackWatches — página admin de watches de rollback (Right-Sizing Studio R5).
 *
 * Lista todos os watches com filtros por status e serviço.
 * Permite rollback manual e cancelar watch.
 * SSE: invalida quando recebe evento do tópico "events" (rollback/optimized/cancelled).
 */
import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { useEventSource } from "@/hooks/use-event-source"
import { api } from "@/api/client"
import { toast } from "sonner"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Shield, RotateCcw, X, Clock, CheckCircle2, Activity, Ban,
} from "lucide-react"

interface RollbackWatch {
  id: number
  change_log_id: number
  service: string
  strategy: string
  observation_window_hours: number
  criteria: { oom: boolean; throttle: boolean; mem_pressure: boolean }
  status: "monitoring" | "optimized" | "rolled_back" | "expired" | "cancelled"
  triggered_criteria: string | null
  started_at: string
  expires_at: string
  rolled_back_at: string | null
  cpu_limit_before: number | null
  mem_limit_before: number | null
  cpu_limit_after: number | null
  mem_limit_after: number | null
}

interface RollbackWatchesResponse {
  watches: RollbackWatch[]
  total: number
}

const statusConfig: Record<string, { label: string; icon: typeof Shield; color: string }> = {
  monitoring: { label: "Monitorando", icon: Activity, color: "bg-blue-500/10 text-blue-700 border-blue-500/30" },
  optimized: { label: "Otimizado", icon: CheckCircle2, color: "bg-green-500/10 text-green-700 border-green-500/30" },
  rolled_back: { label: "Revertido", icon: RotateCcw, color: "bg-orange-500/10 text-orange-700 border-orange-500/30" },
  expired: { label: "Expirado", icon: Clock, color: "bg-muted text-muted-foreground border-border" },
  cancelled: { label: "Cancelado", icon: Ban, color: "bg-muted text-muted-foreground border-border" },
}

export function RollbackWatches() {
  const [statusFilter, setStatusFilter] = useState<string>("")
  const queryClient = useQueryClient()

  // SSE: invalida lista quando recebe evento de rollback/optimized/cancelled
  const { isConnected: sseEvents } = useEventSource({
    topic: "events",
    invalidateQueries: [["rollback-watches"], ["change-log"], ["oom-events"]],
    onEvent: (event) => {
      if (event.type === "rollback") {
        const payload = event.payload as { service: string; reason: string }
        toast.error(`⚠️ ${payload.service} revertido automaticamente (${payload.reason})`)
      } else if (event.type === "optimized") {
        const payload = event.payload as { service: string }
        toast.success(`✅ ${payload.service} otimizado com sucesso (janela sem incidente)`)
      } else if (event.type === "rollback_manual") {
        const payload = event.payload as { service: string; user: string }
        toast.info(`🔄 ${payload.service} revertido manualmente por ${payload.user}`)
      } else if (event.type === "watch_cancelled") {
        const payload = event.payload as { service: string }
        toast.info(`Monitoramento de ${payload.service} cancelado`)
      }
    },
  })

  const { data, isLoading } = useQuery<RollbackWatchesResponse>({
    queryKey: ["rollback-watches", statusFilter],
    queryFn: () => api.get<RollbackWatchesResponse>(
      `/rollback-watches${statusFilter ? `?status=${statusFilter}` : ""}`,
    ),
    refetchInterval: sseEvents ? false : 30000,
  })

  const rollbackMutation = useMutation({
    mutationFn: (id: number) => api.post(`/rollback-watches/${id}/rollback`),
    onSuccess: () => {
      toast.success("Rollback manual executado")
      queryClient.invalidateQueries({ queryKey: ["rollback-watches"] })
    },
    onError: (err: Error) => toast.error(`Erro: ${err.message}`),
  })

  const cancelMutation = useMutation({
    mutationFn: (id: number) => api.post(`/rollback-watches/${id}/cancel`),
    onSuccess: () => {
      toast.success("Watch cancelado")
      queryClient.invalidateQueries({ queryKey: ["rollback-watches"] })
    },
    onError: (err: Error) => toast.error(`Erro: ${err.message}`),
  })

  const watches = data?.watches ?? []
  const statusCounts = watches.reduce<Record<string, number>>((acc, w) => {
    acc[w.status] = (acc[w.status] ?? 0) + 1
    return acc
  }, {})

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Monitoramentos de Rollback</h1>
          <p className="text-sm text-muted-foreground">
            Monitoramento pós-apply com reversão automática
          </p>
        </div>
        <Badge variant="outline" className={sseEvents ? "border-green-500/30 text-green-700" : ""}>
          {sseEvents ? "SSE ativo" : "Polling 30s"}
        </Badge>
      </div>

      {/* Stats por status */}
      <div className="grid gap-3 sm:grid-cols-5">
        {Object.entries(statusConfig).map(([key, cfg]) => {
          const Icon = cfg.icon
          const count = statusCounts[key] ?? 0
          return (
            <Card key={key} className={statusFilter === key ? "ring-2 ring-primary" : ""}>
              <CardContent className="p-3">
                <button
                  className="flex w-full items-center justify-between"
                  onClick={() => setStatusFilter(statusFilter === key ? "" : key)}
                >
                  <div className="flex items-center gap-2">
                    <Icon className="h-4 w-4 text-muted-foreground" />
                    <span className="text-xs font-medium">{cfg.label}</span>
                  </div>
                  <span className="text-lg font-bold tabular-nums">{count}</span>
                </button>
              </CardContent>
            </Card>
          )
        })}
      </div>

      {/* Tabela de watches */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">
            Watches {statusFilter && `(${statusConfig[statusFilter]?.label})`}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : watches.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <Shield className="h-10 w-10 text-muted-foreground/50" />
              <p className="mt-2 text-sm text-muted-foreground">
                Nenhum watch ativo. Aplique uma recomendação com rollback habilitado para criar um watch.
              </p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Serviço</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Estratégia</TableHead>
                  <TableHead>Janela</TableHead>
                  <TableHead>Critério disparou</TableHead>
                  <TableHead>Iniciado</TableHead>
                  <TableHead>Expira</TableHead>
                  <TableHead className="text-right">Ações</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {watches.map((w) => {
                  const sc = statusConfig[w.status] ?? statusConfig.monitoring
                  const StatusIcon = sc.icon
                  return (
                    <TableRow key={w.id}>
                      <TableCell className="font-medium">{w.service}</TableCell>
                      <TableCell>
                        <Badge variant="outline" className={sc.color}>
                          <StatusIcon className="mr-1 h-3 w-3" />
                          {sc.label}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">{w.strategy}</TableCell>
                      <TableCell className="text-xs tabular-nums">{w.observation_window_hours}h</TableCell>
                      <TableCell className="text-xs">
                        {w.triggered_criteria ? (
                          <span className="font-mono text-orange-600">{w.triggered_criteria}</span>
                        ) : (
                          <span className="text-muted-foreground">—</span>
                        )}
                      </TableCell>
                      <TableCell className="text-xs tabular-nums text-muted-foreground">
                        {new Date(w.started_at).toLocaleString("pt-BR", { hour: "2-digit", minute: "2-digit", day: "2-digit", month: "2-digit" })}
                      </TableCell>
                      <TableCell className="text-xs tabular-nums text-muted-foreground">
                        {new Date(w.expires_at).toLocaleString("pt-BR", { hour: "2-digit", minute: "2-digit", day: "2-digit", month: "2-digit" })}
                      </TableCell>
                      <TableCell className="text-right">
                        {w.status === "monitoring" && (
                          <div className="flex justify-end gap-1">
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-7 text-xs text-orange-600 hover:text-orange-700"
                              onClick={() => rollbackMutation.mutate(w.id)}
                              disabled={rollbackMutation.isPending}
                            >
                              <RotateCcw className="mr-1 h-3 w-3" />
                              Reverter
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-7 text-xs text-muted-foreground"
                              onClick={() => {
                                cancelMutation.mutate(w.id)
                              }}
                              disabled={cancelMutation.isPending}
                            >
                              <X className="mr-1 h-3 w-3" />
                              Cancelar
                            </Button>
                          </div>
                        )}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
