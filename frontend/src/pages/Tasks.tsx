import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { useNavigate } from "react-router-dom"
import { api } from "@/api/client"
import { Button } from "@/components/ui/button"
import { useRefreshTimer } from "@/hooks/use-refresh"
import { useEventSource } from "@/hooks/use-event-source"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { EmptyState } from "@/components/empty-state"
import { Badge } from "@/components/ui/badge"
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table"
import { PageHeader } from "@/components/page-header"
import { ListTodo, Boxes, CheckCircle2, XCircle, Clock, Search, Filter, ChevronDown, X } from "lucide-react"
import { useFilterStore } from "@/stores/filter-store"
import { Input } from "@/components/ui/input"
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel } from "@/components/ui/dropdown-menu"

interface Task {
  task_id: string
  service: string
  node_id: string
  node_hostname: string
  slot: number
  status: string
  desired_state: string
  created_at: string | null
  updated_at: string | null
}

const STATUS_CONFIG: Record<string, { label: string; variant: "default" | "secondary" | "destructive" | "outline" | "success" | "warning"; dot: string }> = {
  running: { label: "Running", variant: "success", dot: "bg-green-500" },
  pending: { label: "Pending", variant: "secondary", dot: "bg-yellow-500" },
  assigned: { label: "Assigned", variant: "secondary", dot: "bg-yellow-500" },
  accepted: { label: "Accepted", variant: "secondary", dot: "bg-yellow-500" },
  preparing: { label: "Preparing", variant: "secondary", dot: "bg-blue-500" },
  starting: { label: "Starting", variant: "secondary", dot: "bg-blue-500" },
  complete: { label: "Complete", variant: "outline", dot: "bg-green-500" },
  failed: { label: "Failed", variant: "destructive", dot: "bg-red-500" },
  rejected: { label: "Rejected", variant: "destructive", dot: "bg-red-500" },
  shutdown: { label: "Shutdown", variant: "outline", dot: "bg-muted-foreground" },
  orphaned: { label: "Orphaned", variant: "warning", dot: "bg-orange-500" },
  remove: { label: "Remove", variant: "outline", dot: "bg-muted-foreground" },
}

export default function Tasks() {
  const navigate = useNavigate()
  const [search, setSearch] = useState("")
  const { tasksStatus: statusFilter, tasksService: serviceFilter, setTasksStatus: setStatusFilter, setTasksService: setServiceFilter } = useFilterStore()

  // SSE: invalida queries de tasks quando receber evento
  useEventSource({
    topic: "tasks",
    invalidateQueries: [["tasks"], ["services-health"]],
  })

  // Reconciliação safety-net controlada pelo dropdown (Frente C).
  useRefreshTimer([["tasks"], ["services-health"]])

  const { data: tasks, isLoading } = useQuery<Task[]>({
    queryKey: ["tasks"],
    queryFn: () => api.get<Task[]>("/tasks"),
    refetchInterval: false,
  })

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-10 w-64" />
        <div className="grid gap-4 md:grid-cols-4">
          <Skeleton className="h-28" />
          <Skeleton className="h-28" />
          <Skeleton className="h-28" />
          <Skeleton className="h-28" />
        </div>
        <Skeleton className="h-96" />
      </div>
    )
  }

  if (!tasks || tasks.length === 0) {
    return (
      <div className="space-y-6">
        <PageHeader title="Tasks" description="Tasks do Docker Swarm (lifecycle em tempo real)" />
        <Card>
          <EmptyState icon={ListTodo} message="Nenhuma task encontrada. Aguardando coleta do Swarm..." />
        </Card>
      </div>
    )
  }

  const running = tasks.filter((t) => t.status === "running").length
  const failed = tasks.filter((t) => t.status === "failed" || t.status === "rejected").length
  const services = new Set(tasks.map((t) => t.service)).size

  const statCards = [
    { label: "Tasks", value: tasks.length, icon: ListTodo, iconBg: "bg-chart-1/15", iconColor: "text-chart-1", valueColor: "text-chart-1" },
    { label: "Running", value: running, icon: CheckCircle2, iconBg: "bg-success/15", iconColor: "text-success", valueColor: "text-success" },
    { label: "Failed", value: failed, icon: XCircle, iconBg: "bg-destructive/15", iconColor: "text-destructive", valueColor: "text-destructive" },
    { label: "Serviços", value: services, icon: Boxes, iconBg: "bg-chart-2/15", iconColor: "text-chart-2", valueColor: "text-chart-2" },
  ]

  const serviceNames = Array.from(new Set(tasks.map((t) => t.service).filter(Boolean))).sort()

  const filteredTasks = tasks.filter((t) => {
    if (search) {
      const q = search.toLowerCase()
      if (!t.service.toLowerCase().includes(q) && !t.node_hostname.toLowerCase().includes(q) && !t.task_id.toLowerCase().includes(q)) return false
    }
    if (statusFilter !== "all" && t.status !== statusFilter) return false
    if (serviceFilter !== "all" && t.service !== serviceFilter) return false
    return true
  })

  return (
    <div className="space-y-6">
      <PageHeader title="Tasks" description="Tasks do Docker Swarm (lifecycle em tempo real)">
        <div className="flex items-center gap-2">
          <Badge variant="outline" className="gap-1.5">
            <Clock className="h-3 w-3" />
            {running}/{tasks.length} running
          </Badge>
        </div>
      </PageHeader>

      <div className="grid gap-4 md:grid-cols-4">
        {statCards.map((stat) => (
          <Card key={stat.label} className="hover:bg-accent/30 transition-colors">
            <CardContent className="flex flex-row items-center justify-between pb-2 pt-6">
              <div>
                <div className="text-sm font-medium text-muted-foreground">{stat.label}</div>
                <div className={`text-3xl font-bold tabular-nums ${stat.valueColor}`}>{stat.value}</div>
              </div>
              <div className={`flex h-9 w-9 items-center justify-center rounded-lg ${stat.iconBg}`}>
                <stat.icon className={`h-4.5 w-4.5 ${stat.iconColor}`} />
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardContent className="p-0">
          <div className="flex items-center gap-3 flex-wrap px-4 py-3 border-b">
            <div className="relative flex-1 min-w-48">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Buscar serviço, node ou task ID..."
                className="pl-9"
              />
            </div>

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm" className="gap-1.5">
                  <Filter className="h-3.5 w-3.5" />
                  Serviço
                  {serviceFilter !== "all" && (
                    <Badge variant="secondary" className="text-[10px] px-1 py-0 max-w-32 truncate">{serviceFilter}</Badge>
                  )}
                  <ChevronDown className="h-3.5 w-3.5 opacity-50" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" className="max-h-64 overflow-y-auto">
                <DropdownMenuLabel>Serviço</DropdownMenuLabel>
                <DropdownMenuItem onClick={() => setServiceFilter("all")}>
                  {serviceFilter === "all" && "✓ "}Todos
                </DropdownMenuItem>
                {serviceNames.map((svc) => (
                  <DropdownMenuItem key={svc} onClick={() => setServiceFilter(svc)}>
                    {serviceFilter === svc && "✓ "}{svc}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm" className="gap-1.5">
                  <Filter className="h-3.5 w-3.5" />
                  Status
                  {statusFilter !== "all" && (
                    <Badge variant="secondary" className="text-[10px] px-1 py-0">{STATUS_CONFIG[statusFilter]?.label ?? statusFilter}</Badge>
                  )}
                  <ChevronDown className="h-3.5 w-3.5 opacity-50" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start">
                <DropdownMenuLabel>Status</DropdownMenuLabel>
                <DropdownMenuItem onClick={() => setStatusFilter("all")}>
                  {statusFilter === "all" && "✓ "}Todos
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setStatusFilter("running")}>
                  {statusFilter === "running" && "✓ "}Running
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setStatusFilter("failed")}>
                  {statusFilter === "failed" && "✓ "}Failed
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setStatusFilter("pending")}>
                  {statusFilter === "pending" && "✓ "}Pending
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setStatusFilter("complete")}>
                  {statusFilter === "complete" && "✓ "}Complete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>

            {(search || statusFilter !== "all" || serviceFilter !== "all") && (
              <Button
                variant="ghost"
                size="sm"
                className="gap-1.5 text-muted-foreground"
                onClick={() => { setSearch(""); setStatusFilter("all"); setServiceFilter("all") }}
              >
                <X className="h-3.5 w-3.5" />
              </Button>
            )}

            <span className="text-xs text-muted-foreground ml-auto">
              {filteredTasks.length} de {tasks.length} tasks
            </span>
          </div>

          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Serviço</TableHead>
                <TableHead className="text-center">Slot</TableHead>
                <TableHead className="text-center">Status</TableHead>
                <TableHead className="text-center">Desired</TableHead>
                <TableHead>Node</TableHead>
                <TableHead className="font-mono text-xs">Task ID</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredTasks.map((t) => {
                const statusCfg = STATUS_CONFIG[t.status] ?? { label: t.status, variant: "outline" as const, dot: "bg-muted-foreground" }
                return (
                  <TableRow
                    key={t.task_id}
                    className="cursor-pointer hover:bg-accent/40 transition-colors duration-150"
                    onClick={() => navigate(`/services/${encodeURIComponent(t.service)}`)}
                  >
                    <TableCell className="font-medium">{t.service || "—"}</TableCell>
                    <TableCell className="text-center">
                      <Badge variant="outline" className="tabular-nums">{t.slot}</Badge>
                    </TableCell>
                    <TableCell className="text-center">
                      <Badge variant={statusCfg.variant} className="gap-1.5">
                        <span className={`h-2 w-2 rounded-full ${statusCfg.dot}`} />
                        {statusCfg.label}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-center">
                      <Badge variant="outline">{t.desired_state || "—"}</Badge>
                    </TableCell>
                    <TableCell className="text-sm">{t.node_hostname || t.node_id?.substring(0, 12) || "—"}</TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">{t.task_id.substring(0, 12)}</TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
