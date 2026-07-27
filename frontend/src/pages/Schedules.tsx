import { useState } from "react"
import { Link } from "react-router-dom"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/api/client"
import { useEventSource } from "@/hooks/use-event-source"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { EmptyState } from "@/components/empty-state"
import { PageHeader } from "@/components/page-header"
import { formatBytes, formatCPU } from "@/lib/utils"
import { CalendarClock, CheckCircle2, XCircle, AlertCircle, Clock, X, History, Filter } from "lucide-react"
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem } from "@/components/ui/dropdown-menu"
import { toast } from "sonner"

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

function formatDate(ts: string | null): string {
  if (!ts) return "—"
  const d = new Date(ts)
  return d.toLocaleString("pt-BR", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" })
}

function formatVal(v: number | null, fmt: "cpu" | "mem"): string {
  if (v === null || v === undefined) return "—"
  return fmt === "cpu" ? formatCPU(v) : formatBytes(v)
}

export default function Schedules() {
  const queryClient = useQueryClient()
  const [statusFilter, setStatusFilter] = useState<string>("all")
  const [logFilter, setLogFilter] = useState<string>("all")

  // SSE: assina change-log (Gap C) — disparado pelo scheduler quando um
  // schedule executa/falha/pula. Substitui polling fixo de 30s por tempo real.
  const { isConnected: sseConnected } = useEventSource({
    topic: "change-log",
    invalidateQueries: [["schedules"], ["change-log"]],
  })
  // SSE ativo = zero polling (SSE publica payload completo + reconciliação 30s)
  const fallbackInterval = sseConnected ? false : 30000

  const { data: allSchedules, isLoading } = useQuery<Schedule[]>({
    queryKey: ["schedules"],
    queryFn: () => api.get<Schedule[]>("/schedules"),
    refetchInterval: fallbackInterval,
  })

  const { data: changeLog } = useQuery<ChangeLogEntry[]>({
    queryKey: ["change-log"],
    queryFn: () => api.get<ChangeLogEntry[]>("/change-log"),
    refetchInterval: fallbackInterval,
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

  const filteredSchedules = statusFilter === "all" ? schedules : schedules.filter((s) => s.status === statusFilter)
  const filteredLog = logFilter === "all" ? (changeLog || []) : (changeLog || []).filter((e) => e.source === logFilter)

  const summaryCards = [
    { label: "Pendentes", value: pending.length, icon: Clock, color: "text-warning", bg: "bg-warning/10" },
    { label: "Concluídos", value: completed.length, icon: CheckCircle2, color: "text-success", bg: "bg-success/10" },
    { label: "Falhas", value: failed.length, icon: XCircle, color: "text-destructive", bg: "bg-destructive/10" },
    { label: "Cancelados", value: cancelled.length, icon: XCircle, color: "text-muted-foreground", bg: "bg-muted" },
  ]

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

      <div className="grid gap-4 md:grid-cols-4">
        {summaryCards.map((card) => (
          <Card key={card.label} className="hover:border-primary/40 transition-all">
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

      <Tabs defaultValue="schedules">
        <TabsList>
          <TabsTrigger value="schedules">
            <CalendarClock className="mr-1.5 h-3.5 w-3.5" />
            Agendamentos
          </TabsTrigger>
          <TabsTrigger value="history">
            <History className="mr-1.5 h-3.5 w-3.5" />
            Histórico de Alterações
          </TabsTrigger>
        </TabsList>

        <TabsContent value="schedules" className="space-y-4">
          <div className="flex items-center justify-between">
            <span className="text-sm text-muted-foreground">
              {filteredSchedules.length} de {schedules.length} agendamentos
            </span>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm" className="gap-1.5">
                  <Filter className="h-3.5 w-3.5" />
                  {statusFilter === "all" ? "Todos" : statusConfig[statusFilter]?.label ?? statusFilter}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => setStatusFilter("all")}>Todos ({schedules.length})</DropdownMenuItem>
                <DropdownMenuItem onClick={() => setStatusFilter("pending")}>Pendentes ({pending.length})</DropdownMenuItem>
                <DropdownMenuItem onClick={() => setStatusFilter("completed")}>Concluídos ({completed.length})</DropdownMenuItem>
                <DropdownMenuItem onClick={() => setStatusFilter("failed")}>Falhas ({failed.length})</DropdownMenuItem>
                <DropdownMenuItem onClick={() => setStatusFilter("cancelled")}>Cancelados ({cancelled.length})</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>

          {filteredSchedules.length === 0 ? (
            <Card>
              <EmptyState icon={CalendarClock} message="Nenhum agendamento encontrado" />
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
                      <TableHead className="text-center">Tentativas</TableHead>
                      <TableHead className="w-12"></TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredSchedules.map((s) => {
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
                          <TableCell className="text-sm text-muted-foreground">{formatDate(s.scheduled_at)}</TableCell>
                          <TableCell className="text-sm text-muted-foreground">{formatDate(s.applied_at)}</TableCell>
                          <TableCell className="text-center">
                            <Badge variant="outline" className="tabular-nums">{s.attempts}</Badge>
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

        <TabsContent value="history" className="space-y-4">
          <div className="flex items-center justify-between">
            <span className="text-sm text-muted-foreground">
              {filteredLog.length} de {(changeLog || []).length} alterações
            </span>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm" className="gap-1.5">
                  <Filter className="h-3.5 w-3.5" />
                  {logFilter === "all" ? "Todas as fontes" : logFilter === "manual" ? "Manual" : "Scheduler"}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => setLogFilter("all")}>Todas as fontes</DropdownMenuItem>
                <DropdownMenuItem onClick={() => setLogFilter("manual")}>Manual</DropdownMenuItem>
                <DropdownMenuItem onClick={() => setLogFilter("scheduler")}>Scheduler</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>

          {filteredLog.length === 0 ? (
            <Card>
              <EmptyState icon={History} message="Nenhuma alteração registrada" />
            </Card>
          ) : (
            <Card>
              <CardContent className="p-0">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Serviço</TableHead>
                      <TableHead>Ação</TableHead>
                      <TableHead>Fonte</TableHead>
                      <TableHead>Usuário</TableHead>
                      <TableHead className="text-right">CPU Antes→Depois</TableHead>
                      <TableHead className="text-right">Mem Antes→Depois</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Data</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredLog.map((e) => (
                      <TableRow key={e.id}>
                        <TableCell className="font-medium">
                          <Link to={`/services/${e.service}`} className="text-primary hover:underline">{e.service}</Link>
                        </TableCell>
                        <TableCell>
                          <Badge variant="outline" className="text-xs">
                            {e.action === "apply" ? "Aplicar" : e.action === "scheduled_apply" ? "Agendado" : e.action}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <Badge variant={e.source === "manual" ? "secondary" : "outline"} className="text-xs">
                            {e.source === "manual" ? "Manual" : "Scheduler"}
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
                        <TableCell className="text-sm text-muted-foreground">{formatDate(e.created_at)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          )}
        </TabsContent>
      </Tabs>
    </div>
  )
}
