import { useEffect, useRef, useState } from "react"
import { Link, Outlet, useLocation, useSearchParams } from "react-router-dom"
import { useQueryClient, useIsFetching } from "@tanstack/react-query"
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar"
import { AppSidebar } from "@/components/app-sidebar"
import { NavUser } from "@/components/nav-user"
import { CollectingBadge } from "@/components/collecting-badge"
import { Separator } from "@/components/ui/separator"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu"
import { ChevronRight, RefreshCw, Check } from "lucide-react"
import { useRefreshStore, getIntervalMs, type RefreshMode } from "@/stores/refresh-store"

function buildBreadcrumbs(pathname: string) {
  const segments = pathname.split("/").filter(Boolean)
  const crumbs: { label: string; to?: string }[] = [{ label: "RESMA", to: "/" }]

  if (segments.length === 0) return crumbs

  if (segments[0] === "services") {
    crumbs.push({ label: "Serviços", to: "/services" })
    if (segments[1]) {
      crumbs.push({ label: decodeURIComponent(segments[1]) })
      if (segments[2] === "containers" && segments[3]) {
        crumbs.push({ label: `Container ${decodeURIComponent(segments[3]).substring(0, 12)}` })
      }
    }
  } else if (segments[0] === "optimizations") {
    crumbs.push({ label: "Otimização de Recursos", to: "/optimizations" })
    if (segments[1] === "rollback-watches") {
      crumbs.push({ label: "Monitoramentos de Rollback", to: "/optimizations/rollback-watches" })
    }
  } else if (segments[0] === "rollback-watches") {
    crumbs.push({ label: "Monitoramentos de Rollback", to: "/optimizations/rollback-watches" })
  } else if (segments[0] === "schedules") {
    crumbs.push({ label: "Agendamentos", to: "/schedules" })
  } else if (segments[0] === "templates") {
    crumbs.push({ label: "Templates", to: "/templates" })
  } else if (segments[0] === "nodes") {
    crumbs.push({ label: "Nodes", to: "/nodes" })
    if (segments[1]) {
      crumbs.push({ label: decodeURIComponent(segments[1]) })
    }
  } else if (segments[0] === "tasks") {
    crumbs.push({ label: "Tasks", to: "/tasks" })
  } else if (segments[0] === "alerts") {
    crumbs.push({ label: "Alertas", to: "/alerts" })
  } else if (segments[0] === "rollback-watches") {
    crumbs.push({ label: "Rollback Watches", to: "/rollback-watches" })
  } else if (segments[0] === "settings") {
    crumbs.push({ label: "Configurações", to: "/settings" })
    if (segments[1] === "users") crumbs.push({ label: "Usuários" })
    else if (segments[1] === "api-keys") crumbs.push({ label: "API Keys" })
    else if (segments[1] === "parameters") crumbs.push({ label: "Parâmetros" })
    else if (segments[1] === "data") crumbs.push({ label: "Dados" })
  } else if (segments[0] === "profile") {
    crumbs.push({ label: "Perfil" })
  }

  return crumbs
}

const REFRESH_OPTIONS: { value: RefreshMode; label: string }[] = [
  { value: "auto", label: "Auto" },
  { value: "5s", label: "5s" },
  { value: "30s", label: "30s" },
  { value: "1m", label: "1m" },
  { value: "5m", label: "5m" },
  { value: "off", label: "Off" },
]

// Duração do spinner do refresh manual (tempo suficiente p/ refetch iniciar)
const MANUAL_REFRESH_SPINNER_MS = 600

// Mapeia RefreshMode → valor do parâmetro ?refresh= na URL
const MODE_TO_URL: Record<RefreshMode, string> = {
  auto: "auto",
  "5s": "5s",
  "30s": "30s",
  "1m": "1m",
  "5m": "5m",
  off: "off",
}

// Mapeia valor do parâmetro ?refresh= → RefreshMode
const URL_TO_MODE: Record<string, RefreshMode> = {
  auto: "auto",
  "5s": "5s",
  "30s": "30s",
  "1m": "1m",
  "5m": "5m",
  off: "off",
}

export function Layout() {
  const location = useLocation()
  const [searchParams, setSearchParams] = useSearchParams()
  const refreshMode = useRefreshStore((s) => s.mode)
  const setMode = useRefreshStore((s) => s.setMode)
  const queryClient = useQueryClient()
  const [manualRefreshing, setManualRefreshing] = useState(false)
  const refreshTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Número de queries sendo fetchadas no momento — alimenta o spinner do
  // botão de refresh (alinhado com Grafana: queryController.isRunning).
  const isFetching = useIsFetching()

  const breadcrumbs = buildBreadcrumbs(location.pathname)

  // Cleanup: garante que o timeout não sete state em componente desmontado
  useEffect(() => {
    return () => {
      if (refreshTimer.current) clearTimeout(refreshTimer.current)
    }
  }, [])

  // URL sync — on mount: se ?refresh= estiver na URL, sobrescreve o modo
  // persistido (bookmarkable, como no Grafana: ?refresh=5s).
  useEffect(() => {
    const urlRefresh = searchParams.get("refresh")
    if (urlRefresh && URL_TO_MODE[urlRefresh] && URL_TO_MODE[urlRefresh] !== refreshMode) {
      setMode(URL_TO_MODE[urlRefresh])
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Refresh manual global: invalida queries ativas.
  const handleManualRefresh = () => {
    if (manualRefreshing) return
    setManualRefreshing(true)
    queryClient.invalidateQueries({ refetchType: "active" })
    refreshTimer.current = setTimeout(() => setManualRefreshing(false), MANUAL_REFRESH_SPINNER_MS)
  }

  // Quando o usuário muda o modo via dropdown, atualiza store + URL
  const handleModeChange = (mode: RefreshMode) => {
    setMode(mode)
    const next = new URLSearchParams(searchParams)
    next.set("refresh", MODE_TO_URL[mode])
    setSearchParams(next, { replace: true })
  }

  const currentLabel = REFRESH_OPTIONS.find((o) => o.value === refreshMode)?.label ?? "Auto"
  const currentIntervalMs = getIntervalMs(refreshMode)
  // Tooltip descritivo: mostra o intervalo e explica o modo Auto (30s fixo,
  // independente da taxa de coleta do backend — alinhado ao Grafana).
  const refreshTooltip = currentIntervalMs
    ? refreshMode === "auto"
      ? "Auto = 30s fixo (independente da coleta do backend)"
      : `Reconciliação a cada ${currentLabel}`
    : "Atualização automática desativada (só manual)"

  // Spinner ativo: refresh manual OU queries em fetch (como no Grafana)
  const showSpinner = manualRefreshing || isFetching > 0

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset className="h-svh overflow-hidden">
        <header className="flex h-14 shrink-0 items-center gap-3 border-b bg-card px-4">
          <SidebarTrigger />
          <div className="flex items-center gap-1.5 text-sm text-muted-foreground overflow-hidden">
            {breadcrumbs.map((crumb, i) => (
              <div key={i} className="flex items-center gap-1.5 min-w-0">
                {i > 0 && <ChevronRight className="h-3.5 w-3.5 shrink-0" />}
                {crumb.to ? (
                  <Link to={crumb.to} className="hover:text-foreground transition-colors truncate">{crumb.label}</Link>
                ) : (
                  <span className="font-medium text-foreground truncate">{crumb.label}</span>
                )}
              </div>
            ))}
          </div>
          <div className="ml-auto flex items-center gap-2">
            <CollectingBadge />
            <TooltipProvider delayDuration={200}>
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    onClick={handleManualRefresh}
                    disabled={manualRefreshing}
                    className="flex h-8 items-center gap-1.5 rounded-md border border-input bg-transparent px-2.5 text-xs shadow-sm hover:bg-accent transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    aria-label="Atualizar dados"
                  >
                    <RefreshCw className={`h-3.5 w-3.5 text-muted-foreground ${showSpinner ? "animate-spin" : ""}`} />
                    <span>Atualizar</span>
                  </button>
                </TooltipTrigger>
                <TooltipContent>Recarregar todos os dados agora</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <button
                        className="flex h-8 items-center gap-1.5 rounded-md border border-input bg-transparent px-2.5 text-xs shadow-sm hover:bg-accent transition-colors"
                        aria-label="Modo de refresh"
                      >
                        <RefreshCw className={`h-3.5 w-3.5 text-muted-foreground ${showSpinner ? "animate-spin" : ""}`} />
                        <span>{currentLabel}</span>
                      </button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" className="min-w-40">
                      <DropdownMenuLabel>Intervalo de refresh</DropdownMenuLabel>
                      <DropdownMenuSeparator />
                      {REFRESH_OPTIONS.map((opt) => (
                        <DropdownMenuItem
                          key={opt.value}
                          onClick={() => handleModeChange(opt.value)}
                          className="gap-2 justify-between"
                        >
                          <span>{opt.label}</span>
                          {refreshMode === opt.value && <Check className="h-3.5 w-3.5" />}
                        </DropdownMenuItem>
                      ))}
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TooltipTrigger>
                <TooltipContent>{refreshTooltip}</TooltipContent>
              </Tooltip>
            </TooltipProvider>
            <Separator orientation="vertical" className="h-5" />
            <NavUser />
          </div>
        </header>

        <main className="flex-1 overflow-auto p-6">
          <Outlet />
        </main>
      </SidebarInset>
    </SidebarProvider>
  )
}
