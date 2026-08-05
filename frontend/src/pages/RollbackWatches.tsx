/**
 * RollbackWatches — página admin de watches de rollback (Right-Sizing Studio R5).
 *
 * Lista todos os watches com filtros por status e serviço.
 * Permite rollback manual e cancelar watch.
 * SSE: invalida quando recebe evento do tópico "events" (rollback/optimized/cancelled).
 */
import { useState, useMemo } from "react"
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
import { Input } from "@/components/ui/input"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { formatBytes, formatCores } from "@/lib/utils"
import {
  Shield, RotateCcw, X, Clock, CheckCircle2, Activity, Ban,
  Search, ChevronDown, ChevronRight, Cpu, MemoryStick,
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
  optimized: { label: "Otimizado", icon: CheckCircle2, color: "bg-success/10 text-success border-success/30" },
  rolled_back: { label: "Revertido", icon: RotateCcw, color: "bg-warning/10 text-warning border-warning/30" },
  expired: { label: "Expirado", icon: Clock, color: "bg-muted text-muted-foreground border-border" },
  cancelled: { label: "Cancelado", icon: Ban, color: "bg-muted text-muted-foreground border-border" },
}

// Timestamp relativo (pt-BR)
function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime()
  const min = Math.floor(diff / 60000)
  if (min < 1) return "agora"
  if (min < 60) return `há ${min}min`
  const h = Math.floor(min / 60)
  if (h < 24) return `há ${h}h`
  const d = Math.floor(h / 24)
  return `há ${d}d`
}

const PAGE_SIZE = 20

export function RollbackWatches() {
  const [statusFilter, setStatusFilter] = useState<string>("")
  const [searchQuery, setSearchQuery] = useState("")
  const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set())
  const [page, setPage] = useState(0)
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

  const allWatches = data?.watches ?? []
  const watches = useMemo(() => {
    let filtered = allWatches
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase()
      filtered = filtered.filter(w => w.service.toLowerCase().includes(q))
    }
    // Paginação
    const start = page * PAGE_SIZE
    return filtered.slice(start, start + PAGE_SIZE)
  }, [allWatches, searchQuery, page])

  const totalCount = searchQuery.trim()
    ? allWatches.filter(w => w.service.toLowerCase().includes(searchQuery.toLowerCase())).length
    : allWatches.length
  const totalPages = Math.ceil(totalCount / PAGE_SIZE)

  const statusCounts = allWatches.reduce<Record<string, number>>((acc, w) => {
    acc[w.status] = (acc[w.status] ?? 0) + 1
    return acc
  }, {})

  const toggleRow = (id: number) => {
    setExpandedRows(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  // Countdown: tempo restante até expiração
  const getCountdown = (expiresAt: string): { text: string; pct: number; urgent: boolean } => {
    const now = Date.now()
    const expires = new Date(expiresAt).getTime()
    const diffMs = expires - now
    if (diffMs <= 0) return { text: "expirado", pct: 100, urgent: false }
    const diffH = Math.floor(diffMs / (1000 * 60 * 60))
    const diffMin = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60))
    const totalH = 24 // assumption: janela padrão 24h
    const elapsed = totalH - (diffH + diffMin / 60)
    const pct = Math.min(100, (elapsed / totalH) * 100)
    const urgent = diffH < 2
    return {
      text: diffH > 0 ? `${diffH}h ${diffMin}min` : `${diffMin}min`,
      pct,
      urgent,
    }
  }

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
      <div className="grid gap-3 grid-cols-2 sm:grid-cols-3 lg:grid-cols-5">
        {Object.entries(statusConfig).map(([key, cfg]) => {
          const Icon = cfg.icon
          const count = statusCounts[key] ?? 0
          return (
            <Card key={key} className={statusFilter === key ? "ring-2 ring-primary" : ""}>
              <CardContent className="p-3">
                <button
                  className="flex w-full items-center justify-between"
                  onClick={() => setStatusFilter(statusFilter === key ? "" : key)}
                  aria-label={`Filtrar por status ${cfg.label} — ${count} watches`}
                  aria-pressed={statusFilter === key}
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
          <div className="flex items-center justify-between gap-3">
            <CardTitle className="text-base">
              Watches {statusFilter && `(${statusConfig[statusFilter]?.label})`}
            </CardTitle>
            <div className="relative w-64">
              <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Buscar por serviço..."
                value={searchQuery}
                onChange={(e) => { setSearchQuery(e.target.value); setPage(0) }}
                className="h-8 pl-8 text-sm"
                aria-label="Buscar watches por nome do serviço"
              />
            </div>
          </div>
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
            <Table aria-label="Lista de watches de rollback">
              <TableHeader>
                <TableRow>
                  <TableHead className="w-8" />
                  <TableHead>Serviço</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="hidden md:table-cell">Estratégia</TableHead>
                  <TableHead className="hidden lg:table-cell">Janela</TableHead>
                  <TableHead>Restam</TableHead>
                  <TableHead className="hidden xl:table-cell">Critério</TableHead>
                  <TableHead className="text-right">Ações</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {watches.map((w) => {
                  const sc = statusConfig[w.status] ?? statusConfig.monitoring
                  const StatusIcon = sc.icon
                  const isExpanded = expandedRows.has(w.id)
                  const countdown = w.status === "monitoring" ? getCountdown(w.expires_at) : null
                  return (
                    <>
                      <TableRow key={w.id} className="cursor-pointer hover:bg-muted/50" onClick={() => toggleRow(w.id)}>
                        <TableCell className="w-8">
                          {isExpanded
                            ? <ChevronDown className="h-4 w-4 text-muted-foreground" />
                            : <ChevronRight className="h-4 w-4 text-muted-foreground" />}
                        </TableCell>
                        <TableCell className="font-medium">{w.service}</TableCell>
                        <TableCell>
                          <Badge variant="outline" className={sc.color}>
                            <StatusIcon className="mr-1 h-3 w-3" />
                            {sc.label}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground hidden md:table-cell">{w.strategy}</TableCell>
                        <TableCell className="text-xs tabular-nums hidden lg:table-cell">{w.observation_window_hours}h</TableCell>
                        <TableCell>
                          {countdown ? (
                            <div className="flex flex-col gap-1 min-w-20">
                              <span className={`text-xs font-medium tabular-nums ${countdown.urgent ? "text-destructive" : "text-muted-foreground"}`}>
                                {countdown.text}
                              </span>
                              <div className="h-1.5 rounded-full bg-muted overflow-hidden">
                                <div
                                  className={`h-full rounded-full ${countdown.urgent ? "bg-destructive" : "bg-primary"}`}
                                  style={{ width: `${countdown.pct}%` }}
                                />
                              </div>
                            </div>
                          ) : (
                            <span className="text-xs text-muted-foreground">—</span>
                          )}
                        </TableCell>
                        <TableCell className="text-xs hidden xl:table-cell">
                          {w.triggered_criteria ? (
                            <TooltipProvider delayDuration={200}>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span className="font-mono text-warning cursor-help underline decoration-dotted">{w.triggered_criteria}</span>
                                </TooltipTrigger>
                                <TooltipContent>
                                  <p className="max-w-56">Critério que disparou o rollback automático: {w.triggered_criteria}</p>
                                </TooltipContent>
                              </Tooltip>
                            </TooltipProvider>
                          ) : (
                            <span className="text-muted-foreground">—</span>
                          )}
                        </TableCell>
                        <TableCell className="text-right" onClick={(e) => e.stopPropagation()}>
                          {w.status === "monitoring" && (
                            <div className="flex justify-end gap-1">
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-7 text-xs text-warning hover:text-warning/80"
                                onClick={() => rollbackMutation.mutate(w.id)}
                                disabled={rollbackMutation.isPending}
                                title={`Reverter ${w.service} para config anterior`}
                              >
                                <RotateCcw className="mr-1 h-3 w-3" />
                                Reverter
                              </Button>
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-7 text-xs text-muted-foreground"
                                onClick={() => cancelMutation.mutate(w.id)}
                                disabled={cancelMutation.isPending}
                                title={`Cancelar monitoramento de ${w.service}`}
                              >
                                <X className="mr-1 h-3 w-3" />
                                Cancelar
                              </Button>
                            </div>
                          )}
                        </TableCell>
                      </TableRow>
                      {isExpanded && (
                        <TableRow key={`${w.id}-detail`} className="bg-muted/30">
                          <TableCell colSpan={8} className="py-4">
                            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                              {/* Antes */}
                              <div className="space-y-1">
                                <div className="text-xs text-muted-foreground font-medium">Antes (limites)</div>
                                <div className="flex items-center gap-1.5 tabular-nums">
                                  <Cpu className="h-3.5 w-3.5 text-muted-foreground" />
                                  {w.cpu_limit_before != null ? `${formatCores(w.cpu_limit_before)} cores` : "—"}
                                </div>
                                <div className="flex items-center gap-1.5 tabular-nums">
                                  <MemoryStick className="h-3.5 w-3.5 text-muted-foreground" />
                                  {w.mem_limit_before != null ? formatBytes(w.mem_limit_before) : "—"}
                                </div>
                              </div>
                              {/* Depois */}
                              <div className="space-y-1">
                                <div className="text-xs text-muted-foreground font-medium">Depois (limites)</div>
                                <div className="flex items-center gap-1.5 tabular-nums">
                                  <Cpu className="h-3.5 w-3.5 text-muted-foreground" />
                                  {w.cpu_limit_after != null ? `${formatCores(w.cpu_limit_after)} cores` : "—"}
                                </div>
                                <div className="flex items-center gap-1.5 tabular-nums">
                                  <MemoryStick className="h-3.5 w-3.5 text-muted-foreground" />
                                  {w.mem_limit_after != null ? formatBytes(w.mem_limit_after) : "—"}
                                </div>
                              </div>
                              {/* Critérios monitorados */}
                              <div className="space-y-1">
                                <div className="text-xs text-muted-foreground font-medium">Critérios</div>
                                <div className="flex flex-wrap gap-1">
                                  {w.criteria.oom && <Badge variant="outline" className="text-[10px] py-0">OOM</Badge>}
                                  {w.criteria.throttle && <Badge variant="outline" className="text-[10px] py-0">Throttle</Badge>}
                                  {w.criteria.mem_pressure && <Badge variant="outline" className="text-[10px] py-0">Mem pressure</Badge>}
                                </div>
                              </div>
                              {/* Timestamps */}
                              <div className="space-y-1">
                                <div className="text-xs text-muted-foreground font-medium">Timeline</div>
                                <TooltipProvider delayDuration={200}>
                                  <Tooltip>
                                    <TooltipTrigger asChild>
                                      <span className="text-xs text-muted-foreground tabular-nums cursor-help">Início: {timeAgo(w.started_at)}</span>
                                    </TooltipTrigger>
                                    <TooltipContent>{new Date(w.started_at).toLocaleString("pt-BR")}</TooltipContent>
                                  </Tooltip>
                                </TooltipProvider>
                                <TooltipProvider delayDuration={200}>
                                  <Tooltip>
                                    <TooltipTrigger asChild>
                                      <span className="text-xs text-muted-foreground tabular-nums cursor-help">Expira: {timeAgo(w.expires_at)}</span>
                                    </TooltipTrigger>
                                    <TooltipContent>{new Date(w.expires_at).toLocaleString("pt-BR")}</TooltipContent>
                                  </Tooltip>
                                </TooltipProvider>
                                {w.rolled_back_at && (
                                  <TooltipProvider delayDuration={200}>
                                    <Tooltip>
                                      <TooltipTrigger asChild>
                                        <span className="text-xs text-warning tabular-nums cursor-help">Revertido: {timeAgo(w.rolled_back_at)}</span>
                                      </TooltipTrigger>
                                      <TooltipContent>{new Date(w.rolled_back_at).toLocaleString("pt-BR")}</TooltipContent>
                                    </Tooltip>
                                  </TooltipProvider>
                                )}
                              </div>
                            </div>
                          </TableCell>
                        </TableRow>
                      )}
                    </>
                  )
                })}
              </TableBody>
            </Table>
          )}
          {/* Paginação */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between pt-4 border-t">
              <span className="text-xs text-muted-foreground">
                Página {page + 1} de {totalPages} · {totalCount} watches
              </span>
              <div className="flex gap-1">
                <Button
                  variant="outline"
                  size="sm"
                  className="h-7 text-xs"
                  onClick={() => setPage(p => Math.max(0, p - 1))}
                  disabled={page === 0}
                  aria-label="Página anterior"
                >
                  Anterior
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-7 text-xs"
                  onClick={() => setPage(p => Math.min(totalPages - 1, p + 1))}
                  disabled={page >= totalPages - 1}
                  aria-label="Próxima página"
                >
                  Próxima
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
