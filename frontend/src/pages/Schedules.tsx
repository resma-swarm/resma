import { Link } from "react-router-dom"
import { useMemo, useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/api/client"
import { useEventSource } from "@/hooks/use-event-source"
import { useRefreshTimer } from "@/hooks/use-refresh"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { EmptyState } from "@/components/empty-state"
import { PageHeader } from "@/components/page-header"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { HelpIcon } from "@/components/help-icon"
import { formatBytes, formatCPU } from "@/lib/utils"
import { CalendarClock, CheckCircle2, XCircle, AlertCircle, Clock, X, History, Filter, Search, ArrowRight, FileText, ChevronLeft, ChevronRight } from "lucide-react"
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem } from "@/components/ui/dropdown-menu"
import { useFilterStore } from "@/stores/filter-store"
import { toast } from "sonner"

const PAGE_SIZE = 20

interface Schedule {
  id: number
  service: string
  cpu_limit: number | null
  mem_limit: number | null
  cpu_reservation: number | null
  mem_reservation: number | null
  scheduled_at: string
  status: string
  applied_at: string | null
  error: string | null
  attempts: number
  created_at: string
}

interface ChangeLogEntry {
  id: number
  service: string
  action: string
  source: string
  schedule_id: number | null
  cpu_limit_before: number | null
  mem_limit_before: number | null
  cpu_reservation_before: number | null
  mem_reservation_before: number | null
  cpu_limit_after: number | null
  mem_limit_after: number | null
  cpu_reservation_after: number | null
  mem_reservation_after: number | null
  user: string | null
  status: string
  error: string | null
  docker_response: string | null
  created_at: string
}

const statusConfig: Record<string, { label: string; variant: "success" | "secondary" | "destructive" | "outline" | "warning"; icon: typeof Clock }> = {
  pending: { label: "Pendente", variant: "warning", icon: Clock },
  running: { label: "Executando", variant: "secondary", icon: AlertCircle },
  completed: { label: "Concluído", variant: "success", icon: CheckCircle2 },
  failed: { label: "Falhou", variant: "destructive", icon: XCircle },
  cancelled: { label: "Cancelado", variant: "outline", icon: XCircle },
}

// Timestamp absoluto para tooltip (pt-BR completo com ano + segundos).
function formatTimestamp(ts: string | null): string {
  if (!ts) return "—"
  try {
    const d = new Date(ts)
    return d.toLocaleString("pt-BR", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    })
  } catch {
    return ts
  }
}

// timeAgo — timestamp relativo ("há 5min"), padrão do Studio/Alerts/RollbackWatches.
function timeAgo(ts: string | null): string {
  if (!ts) return "—"
  try {
    const d = new Date(ts)
    const diffMs = Date.now() - d.getTime()
    if (diffMs < 0) return formatTimestamp(ts)
    const diffMin = Math.floor(diffMs / 60000)
    if (diffMin < 1) return "agora"
    if (diffMin < 60) return `há ${diffMin}min`
    const diffH = Math.floor(diffMin / 60)
    if (diffH < 24) return `há ${diffH}h`
    const diffD = Math.floor(diffH / 24)
    return `há ${diffD}d`
  } catch {
    return ts
  }
}

function formatVal(v: number | null, fmt: "cpu" | "mem"): string {
  if (v === null || v === undefined) return "—"
  return fmt === "cpu" ? formatCPU(v) : formatBytes(v)
}

// TimestampCell — timestamp relativo com tooltip mostrando absoluto.
// Padrão evoluído: Alerts, Studio, RollbackWatches.
function TimestampCell({ ts }: { ts: string | null }) {
  if (!ts) return <span className="text-muted-foreground">—</span>
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="cursor-help text-sm text-muted-foreground">{timeAgo(ts)}</span>
        </TooltipTrigger>
        <TooltipContent>
          <p>{formatTimestamp(ts)}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

export default function Schedules() {
  const queryClient = useQueryClient()
  const {
    schedulesStatus: statusFilter,
    setSchedulesStatus: setStatusFilter,
    schedulesLogSource: logFilter,
    setSchedulesLogSource: setLogFilter,
    schedulesSearch: search,
    setSchedulesSearch: setSearch,
    schedulesTab: activeTab,
    setSchedulesTab: setActiveTab,
  } = useFilterStore()

  // SSE: assina change-log — disparado pelo scheduler quando um
  // schedule executa/falha/pula. Substitui polling fixo de 30s por tempo real.
  useEventSource({
    topic: "change-log",
    invalidateQueries: [["schedules"], ["change-log"]],
  })

  // Reconciliação safety-net controlada pelo dropdown (Frente C).
  useRefreshTimer([["schedules"], ["change-log"]])

  const { data: allSchedules, isLoading } = useQuery<Schedule[]>({
    queryKey: ["schedules"],
    queryFn: () => api.get<Schedule[]>("/schedules"),
    refetchInterval: false,
  })

  const { data: changeLog } = useQuery<ChangeLogEntry[]>({
    queryKey: ["change-log"],
    queryFn: () => api.get<ChangeLogEntry[]>("/change-log"),
    refetchInterval: false,
  })

  const cancelMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/schedules/${id}`),
    onSuccess: () => {
      toast.success("Agendamento cancelado")
      queryClient.invalidateQueries({ queryKey: ["schedules"] })
    },
    onError: () => toast.error("Erro ao cancelar agendamento"),
  })

  const schedules = allSchedules || []
  const pending = schedules.filter((s) => s.status === "pending")
  const completed = schedules.filter((s) => s.status === "completed")
  const failed = schedules.filter((s) => s.status === "failed")
  const cancelled = schedules.filter((s) => s.status === "cancelled")
  const historySchedules = [...completed, ...failed, ...cancelled]

  // Busca textual — filtra por service e message/error em todas as abas.
  const searchFilter = (item: { service: string; error?: string | null; action?: string }) => {
    if (!search) return true
    const q = search.toLowerCase()
    if (item.service.toLowerCase().includes(q)) return true
    if (item.error && item.error.toLowerCase().includes(q)) return true
    if (item.action && item.action.toLowerCase().includes(q)) return true
    return false
  }

  // Aba Pendentes: só pending, filtrado por busca.
  const filteredPending = pending.filter(searchFilter)

  // Aba Histórico: completed/failed/cancelled, filtrado por status + busca.
  const filteredHistory = (statusFilter === "all" ? historySchedules : historySchedules.filter((s) => s.status === statusFilter)).filter(searchFilter)

  // Aba Auditoria: change-log, filtrado por source + busca.
  const filteredLog = (logFilter === "all" ? (changeLog || []) : (changeLog || []).filter((e) => e.source === logFilter)).filter(searchFilter)

  // Paginação client-side (PAGE_SIZE=20) — apenas Histórico e Auditoria (volume cresce).
  // Aba Pendentes não pagina (volume baixo por natureza).
  const [historyPage, setHistoryPage] = useState(0)
  const [auditPage, setAuditPage] = useState(0)

  // Reset de página quando filtros/busca/aba mudam (evita página fora do range).
  // useEffect implícito via useMemo — recalcula slice a partir dos filtros atuais.
  const historyTotalPages = Math.max(1, Math.ceil(filteredHistory.length / PAGE_SIZE))
  const auditTotalPages = Math.max(1, Math.ceil(filteredLog.length / PAGE_SIZE))
  const safeHistoryPage = Math.min(historyPage, historyTotalPages - 1)
  const safeAuditPage = Math.min(auditPage, auditTotalPages - 1)

  const pagedHistory = useMemo(() => {
    const start = safeHistoryPage * PAGE_SIZE
    return filteredHistory.slice(start, start + PAGE_SIZE)
  }, [filteredHistory, safeHistoryPage])

  const pagedLog = useMemo(() => {
    const start = safeAuditPage * PAGE_SIZE
    return filteredLog.slice(start, start + PAGE_SIZE)
  }, [filteredLog, safeAuditPage])

  const summaryCards = [
    { label: "Pendentes", value: pending.length, icon: Clock, color: "text-warning", bg: "bg-warning/10", status: "pending", tab: "pending" },
    { label: "Concluídos", value: completed.length, icon: CheckCircle2, color: "text-success", bg: "bg-success/10", status: "completed", tab: "history" },
    { label: "Falhas", value: failed.length, icon: XCircle, color: "text-destructive", bg: "bg-destructive/10", status: "failed", tab: "history" },
    { label: "Cancelados", value: cancelled.length, icon: XCircle, color: "text-muted-foreground", bg: "bg-muted", status: "cancelled", tab: "history" },
  ]

  // Click no stat card → troca aba + filtra por status.
  const handleCardClick = (card: typeof summaryCards[number]) => {
    setActiveTab(card.tab)
    if (card.tab === "history") {
      setStatusFilter(card.status)
    }
  }

  // Card ativo: quando a aba e o filtro correspondem ao card.
  const isCardActive = (card: typeof summaryCards[number]) => {
    if (card.tab !== activeTab) return false
    if (card.tab === "pending") return true
    return statusFilter === card.status
  }

  const hasFilters = search !== "" || statusFilter !== "all" || logFilter !== "all"

  const clearFilters = () => {
    setSearch("")
    setStatusFilter("all")
    setLogFilter("all")
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-10 w-64" />
        <div className="grid gap-4 md:grid-cols-4">
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
        </div>
        <Skeleton className="h-96" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader title="Agendamentos" description="Gerencie agendamentos e visualize o histórico de alterações">
        <Badge variant="outline" className="gap-1.5">
          <CalendarClock className="h-3.5 w-3.5" />
          {schedules.length} total
        </Badge>
      </PageHeader>

      {/* Stat cards clicáveis — filtram a tabela ao clicar (padrão RollbackWatches). */}
      <div className="grid gap-4 md:grid-cols-4">
        {summaryCards.map((card) => (
          <Card
            key={card.label}
            className={`hover:border-primary/40 transition-all cursor-pointer ${isCardActive(card) ? "ring-2 ring-primary" : ""}`}
            onClick={() => handleCardClick(card)}
          >
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <CardTitle className="text-sm font-medium text-muted-foreground">{card.label}</CardTitle>
                <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${card.bg}`}>
                  <card.icon className={`h-4 w-4 ${card.color}`} />
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <span className="text-2xl font-bold tabular-nums">{card.value}</span>
            </CardContent>
          </Card>
        ))}
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="pending">
            <CalendarClock className="mr-1.5 h-3.5 w-3.5" />
            Pendentes
            {pending.length > 0 && <Badge variant="warning" className="ml-1.5 h-4 px-1 text-xs">{pending.length}</Badge>}
          </TabsTrigger>
          <TabsTrigger value="history">
            <History className="mr-1.5 h-3.5 w-3.5" />
            Histórico
          </TabsTrigger>
          <TabsTrigger value="audit">
            <FileText className="mr-1.5 h-3.5 w-3.5" />
            Auditoria
          </TabsTrigger>
        </TabsList>

        {/* Aba Pendentes — schedules aguardando execução (acionável: cancelar). */}
        <TabsContent value="pending" className="space-y-4">
          <div className="flex items-center gap-3 flex-wrap">
            <div className="relative flex-1 min-w-48">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Buscar serviço ou mensagem..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-8"
              />
            </div>
            <span className="text-sm text-muted-foreground ml-auto">
              {filteredPending.length} de {pending.length} pendentes
            </span>
            {hasFilters && (
              <Button variant="ghost" size="sm" onClick={clearFilters} className="gap-1">
                <X className="h-3.5 w-3.5" />
                Limpar
              </Button>
            )}
          </div>

          {filteredPending.length === 0 ? (
            <Card>
              <EmptyState
                icon={CalendarClock}
                message={pending.length === 0 ? "Nenhum agendamento pendente." : "Nenhum resultado para a busca."}
                action={
                  pending.length === 0 ? (
                    <Button asChild variant="outline" size="sm" className="gap-1.5">
                      <Link to="/optimizations">
                        Ver recomendações
                        <ArrowRight className="h-3.5 w-3.5" />
                      </Link>
                    </Button>
                  ) : undefined
                }
              />
            </Card>
          ) : (
            <Card>
              <CardContent className="p-0">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Serviço</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead className="text-right">CPU Lim</TableHead>
                      <TableHead className="text-right">Mem Lim</TableHead>
                      <TableHead>Agendado para</TableHead>
                      <TableHead className="w-12"></TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredPending.map((s) => {
                      const cfg = statusConfig[s.status] ?? { label: s.status, variant: "outline" as const, icon: AlertCircle }
                      return (
                        <TableRow key={s.id}>
                          <TableCell className="font-medium">
                            <Link to={`/services/${s.service}`} className="text-primary hover:underline">{s.service}</Link>
                          </TableCell>
                          <TableCell>
                            <Badge variant={cfg.variant} className="gap-1">
                              <cfg.icon className="h-3 w-3" />
                              {cfg.label}
                            </Badge>
                          </TableCell>
                          <TableCell className="text-right tabular-nums text-muted-foreground">{formatVal(s.cpu_limit, "cpu")}</TableCell>
                          <TableCell className="text-right tabular-nums text-muted-foreground">{formatVal(s.mem_limit, "mem")}</TableCell>
                          <TableCell>
                            <div className="flex items-center gap-1">
                              <TimestampCell ts={s.scheduled_at} />
                              <HelpIcon text="Data e hora em que o schedule está configurado para executar automaticamente." />
                            </div>
                          </TableCell>
                          <TableCell>
                            {s.status === "pending" && (
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-7 w-7 text-destructive hover:bg-destructive/10"
                                onClick={() => cancelMutation.mutate(s.id)}
                              >
                                <X className="h-3.5 w-3.5" />
                              </Button>
                            )}
                          </TableCell>
                        </TableRow>
                      )
                    })}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {/* Aba Histórico — schedules já executados (completed/failed/cancelled). */}
        <TabsContent value="history" className="space-y-4">
          <div className="flex items-center gap-3 flex-wrap">
            <div className="relative flex-1 min-w-48">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Buscar serviço ou mensagem..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-8"
              />
            </div>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm" className="gap-1.5">
                  <Filter className="h-3.5 w-3.5" />
                  {statusFilter === "all" ? "Todos" : statusConfig[statusFilter]?.label ?? statusFilter}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => setStatusFilter("all")}>Todos ({historySchedules.length})</DropdownMenuItem>
                <DropdownMenuItem onClick={() => setStatusFilter("completed")}>Concluídos ({completed.length})</DropdownMenuItem>
                <DropdownMenuItem onClick={() => setStatusFilter("failed")}>Falhas ({failed.length})</DropdownMenuItem>
                <DropdownMenuItem onClick={() => setStatusFilter("cancelled")}>Cancelados ({cancelled.length})</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
            <span className="text-sm text-muted-foreground ml-auto">
              {filteredHistory.length} de {historySchedules.length} agendamentos
            </span>
            {hasFilters && (
              <Button variant="ghost" size="sm" onClick={clearFilters} className="gap-1">
                <X className="h-3.5 w-3.5" />
                Limpar
              </Button>
            )}
          </div>

          {filteredHistory.length === 0 ? (
            <Card>
              <EmptyState
                icon={History}
                message={historySchedules.length === 0 ? "Nenhum agendamento executado ainda." : "Nenhum resultado para os filtros selecionados."}
                action={
                  historySchedules.length === 0 ? (
                    <Button asChild variant="outline" size="sm" className="gap-1.5">
                      <Link to="/optimizations">
                        Ver recomendações
                        <ArrowRight className="h-3.5 w-3.5" />
                      </Link>
                    </Button>
                  ) : undefined
                }
              />
            </Card>
          ) : (
            <Card>
              <CardContent className="p-0">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Serviço</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead className="text-right">CPU Lim</TableHead>
                      <TableHead className="text-right">Mem Lim</TableHead>
                      <TableHead>Agendado para</TableHead>
                      <TableHead>Aplicado em</TableHead>
                      <TableHead className="text-center">
                        <span className="inline-flex items-center gap-1">
                          Tentativas
                          <HelpIcon text="Número de tentativas de aplicação (retry automático em caso de falha transiente)." />
                        </span>
                      </TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {pagedHistory.map((s) => {
                      const cfg = statusConfig[s.status] ?? { label: s.status, variant: "outline" as const, icon: AlertCircle }
                      return (
                        <TableRow key={s.id}>
                          <TableCell className="font-medium">
                            <Link to={`/services/${s.service}`} className="text-primary hover:underline">{s.service}</Link>
                          </TableCell>
                          <TableCell>
                            <Badge variant={cfg.variant} className="gap-1">
                              <cfg.icon className="h-3 w-3" />
                              {cfg.label}
                            </Badge>
                          </TableCell>
                          <TableCell className="text-right tabular-nums text-muted-foreground">{formatVal(s.cpu_limit, "cpu")}</TableCell>
                          <TableCell className="text-right tabular-nums text-muted-foreground">{formatVal(s.mem_limit, "mem")}</TableCell>
                          <TableCell><TimestampCell ts={s.scheduled_at} /></TableCell>
                          <TableCell><TimestampCell ts={s.applied_at} /></TableCell>
                          <TableCell className="text-center">
                            <Badge variant="outline" className="tabular-nums">{s.attempts}</Badge>
                          </TableCell>
                        </TableRow>
                      )
                    })}
                  </TableBody>
                </Table>
              </CardContent>
              {historyTotalPages > 1 && (
                <div className="flex items-center justify-between p-4 border-t">
                  <span className="text-xs text-muted-foreground">
                    Página {safeHistoryPage + 1} de {historyTotalPages} · {filteredHistory.length} agendamentos
                  </span>
                  <div className="flex gap-1">
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 text-xs"
                      onClick={() => setHistoryPage(p => Math.max(0, p - 1))}
                      disabled={safeHistoryPage === 0}
                      aria-label="Página anterior"
                    >
                      <ChevronLeft className="h-3.5 w-3.5" />
                      Anterior
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 text-xs"
                      onClick={() => setHistoryPage(p => Math.min(historyTotalPages - 1, p + 1))}
                      disabled={safeHistoryPage >= historyTotalPages - 1}
                      aria-label="Próxima página"
                    >
                      Próxima
                      <ChevronRight className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>
              )}
            </Card>
          )}
        </TabsContent>

        {/* Aba Auditoria — change-log (audit de todas mudanças: manual/scheduler/rollback). */}
        <TabsContent value="audit" className="space-y-4">
          <div className="flex items-center gap-3 flex-wrap">
            <div className="relative flex-1 min-w-48">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Buscar serviço ou mensagem..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-8"
              />
            </div>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm" className="gap-1.5">
                  <Filter className="h-3.5 w-3.5" />
                  {logFilter === "all" ? "Todas as fontes" : logFilter === "manual" ? "Manual" : logFilter === "scheduler" ? "Scheduler" : logFilter === "auto" ? "Auto" : logFilter}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => setLogFilter("all")}>Todas as fontes</DropdownMenuItem>
                <DropdownMenuItem onClick={() => setLogFilter("manual")}>Manual</DropdownMenuItem>
                <DropdownMenuItem onClick={() => setLogFilter("scheduler")}>Scheduler</DropdownMenuItem>
                <DropdownMenuItem onClick={() => setLogFilter("auto")}>Auto (rollback)</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
            <span className="text-sm text-muted-foreground ml-auto">
              {filteredLog.length} de {(changeLog || []).length} alterações
            </span>
            {hasFilters && (
              <Button variant="ghost" size="sm" onClick={clearFilters} className="gap-1">
                <X className="h-3.5 w-3.5" />
                Limpar
              </Button>
            )}
          </div>

          {filteredLog.length === 0 ? (
            <Card>
              <EmptyState
                icon={FileText}
                message={(changeLog || []).length === 0 ? "Nenhuma alteração registrada." : "Nenhum resultado para os filtros selecionados."}
              />
            </Card>
          ) : (
            <Card>
              <CardContent className="p-0">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Serviço</TableHead>
                      <TableHead>
                        <span className="inline-flex items-center gap-1">
                          Ação
                          <HelpIcon text="Tipo de mudança aplicada: apply (aplicar limites), scheduled_apply (agendado), rollback (reverter para valores anteriores)." />
                        </span>
                      </TableHead>
                      <TableHead>
                        <span className="inline-flex items-center gap-1">
                          Fonte
                          <HelpIcon text="Origem da mudança: manual (usuário aplicou via Studio), scheduler (agendamento executou automaticamente), auto (rollback automático por instabilidade)." />
                        </span>
                      </TableHead>
                      <TableHead>Usuário</TableHead>
                      <TableHead className="text-right">CPU Antes→Depois</TableHead>
                      <TableHead className="text-right">Mem Antes→Depois</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Data</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {pagedLog.map((e) => (
                      <TableRow key={e.id}>
                        <TableCell className="font-medium">
                          <Link to={`/services/${e.service}`} className="text-primary hover:underline">{e.service}</Link>
                        </TableCell>
                        <TableCell>
                          <Badge variant="outline" className="text-xs">
                            {e.action === "apply" ? "Aplicar" : e.action === "scheduled_apply" ? "Agendado" : e.action === "rollback" ? "Rollback" : e.action}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <Badge variant={e.source === "manual" ? "secondary" : "outline"} className="text-xs">
                            {e.source === "manual" ? "Manual" : e.source === "scheduler" ? "Scheduler" : e.source === "auto" ? "Auto" : e.source}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground">{e.user ?? "—"}</TableCell>
                        <TableCell className="text-right text-sm tabular-nums text-muted-foreground">
                          {formatVal(e.cpu_limit_before, "cpu")} → {formatVal(e.cpu_limit_after, "cpu")}
                        </TableCell>
                        <TableCell className="text-right text-sm tabular-nums text-muted-foreground">
                          {formatVal(e.mem_limit_before, "mem")} → {formatVal(e.mem_limit_after, "mem")}
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant={e.status === "completed" ? "success" : "destructive"}
                            className="text-xs"
                          >
                            {e.status === "completed" ? "OK" : "Erro"}
                          </Badge>
                        </TableCell>
                        <TableCell><TimestampCell ts={e.created_at} /></TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
              {auditTotalPages > 1 && (
                <div className="flex items-center justify-between p-4 border-t">
                  <span className="text-xs text-muted-foreground">
                    Página {safeAuditPage + 1} de {auditTotalPages} · {filteredLog.length} alterações
                  </span>
                  <div className="flex gap-1">
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 text-xs"
                      onClick={() => setAuditPage(p => Math.max(0, p - 1))}
                      disabled={safeAuditPage === 0}
                      aria-label="Página anterior"
                    >
                      <ChevronLeft className="h-3.5 w-3.5" />
                      Anterior
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 text-xs"
                      onClick={() => setAuditPage(p => Math.min(auditTotalPages - 1, p + 1))}
                      disabled={safeAuditPage >= auditTotalPages - 1}
                      aria-label="Próxima página"
                    >
                      Próxima
                      <ChevronRight className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>
              )}
            </Card>
          )}
        </TabsContent>
      </Tabs>
    </div>
  )
}
