import { useState } from "react"
import { Link, Outlet, useLocation } from "react-router-dom"
import { useQueryClient } from "@tanstack/react-query"
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
import { ChevronRight, RefreshCw } from "lucide-react"
import { useRefreshStore, type RefreshMode } from "@/stores/refresh-store"

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

export function Layout() {
  const location = useLocation()
  const refreshMode = useRefreshStore((s) => s.mode)
  const queryClient = useQueryClient()
  const [manualRefreshing, setManualRefreshing] = useState(false)

  const breadcrumbs = buildBreadcrumbs(location.pathname)

  // Refresh manual global: invalida todas as queries ativas.
  // Substitui o dropdown (disabled temporariamente — Frente C, fase final).
  const handleManualRefresh = () => {
    if (manualRefreshing) return
    setManualRefreshing(true)
    queryClient.invalidateQueries()
    // Reseta o spinner após 600ms (tempo suficiente p/ refetch iniciar)
    setTimeout(() => setManualRefreshing(false), 600)
  }

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
                    <RefreshCw className={`h-3.5 w-3.5 text-muted-foreground ${manualRefreshing ? "animate-spin" : ""}`} />
                    <span>Atualizar</span>
                  </button>
                </TooltipTrigger>
                <TooltipContent>Recarregar todos os dados agora</TooltipContent>
              </Tooltip>
            </TooltipProvider>
            <TooltipProvider delayDuration={200}>
              <Tooltip>
                <TooltipTrigger asChild>
                  <div className="flex h-8 items-center gap-1.5 rounded-md border border-input bg-transparent px-2.5 text-xs shadow-sm opacity-60 cursor-not-allowed">
                    <RefreshCw className="h-3.5 w-3.5 text-muted-foreground" />
                    <span>{REFRESH_OPTIONS.find((o) => o.value === refreshMode)?.label ?? "Auto"}</span>
                  </div>
                </TooltipTrigger>
                <TooltipContent>Ajuste de intervalo em implementação — use o refresh manual</TooltipContent>
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
