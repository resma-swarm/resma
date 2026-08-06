import { Activity, Settings2 } from "lucide-react"
import { Link } from "react-router-dom"
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarRail,
} from "@/components/ui/sidebar"
import { NavMain } from "@/components/nav-main"
import { NavSettings } from "@/components/nav-settings"

// Versão do RESMA (fixa por enquanto — futuramente gerada via build)
const RESMA_VERSION = "0.9.0"

export function AppSidebar() {
  return (
    <Sidebar collapsible="icon" className="border-border">
      <SidebarHeader className="p-0">
        <Link
          to="/"
          className="flex h-14 items-center gap-2.5 border-b border-border bg-card px-4 transition-colors hover:bg-accent group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:px-0"
          title="Ir para o Dashboard"
        >
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary/15">
            <Activity className="h-4.5 w-4.5 text-primary" />
          </div>
          <div className="flex flex-col group-data-[collapsible=icon]:hidden">
            <span className="text-sm font-bold tracking-tight">RESMA</span>
            <span className="text-[10px] text-muted-foreground">Optimize Infrastructure</span>
          </div>
        </Link>
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Monitoramento</SidebarGroupLabel>
          <SidebarGroupContent>
            <NavMain />
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup>
          <SidebarGroupLabel>
            <Settings2 className="h-3.5 w-3.5" />
            Configurações
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <NavSettings />
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <div className="mt-auto border-t border-border px-4 py-3 group-data-[collapsible=icon]:hidden">
        <div className="flex items-center justify-between text-[10px] text-muted-foreground">
          <span>RESMA</span>
          <span className="font-mono">v{RESMA_VERSION}</span>
        </div>
      </div>

      <SidebarRail />
    </Sidebar>
  )
}
