import { Outlet, NavLink } from "react-router-dom"
import { PageHeader } from "@/components/page-header"
import { Users, KeyRound, SlidersHorizontal, Database } from "lucide-react"
import { usePermissions } from "@/hooks/use-permissions"
import { cn } from "@/lib/utils"
import { RequireRole } from "@/components/RequireRole"

interface SettingsTab {
  to: string
  label: string
  icon: typeof Users
  show: boolean
}

export function Settings() {
  const perms = usePermissions()

  const tabs: SettingsTab[] = [
    { to: "/settings/users", label: "Usuários", icon: Users, show: perms.canManageUsers },
    { to: "/settings/api-keys", label: "API Keys", icon: KeyRound, show: perms.canManageAPIKeys },
    { to: "/settings/parameters", label: "Parâmetros", icon: SlidersHorizontal, show: perms.canManageSettings },
    { to: "/settings/data", label: "Dados", icon: Database, show: perms.canPrune },
  ]

  const visibleTabs = tabs.filter((t) => t.show)

  return (
    <RequireRole allowedRoles={["owner", "admin"]}>
      <div className="space-y-6">
        <PageHeader title="Configurações" description="Gestão de usuários, API keys, parâmetros e dados" />

        <div className="flex flex-wrap gap-1 border-b">
          {visibleTabs.map((tab) => (
            <NavLink
              key={tab.to}
              to={tab.to}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-2 px-4 py-2.5 text-sm font-medium transition-colors border-b-2 -mb-px",
                  isActive
                    ? "border-primary text-primary"
                    : "border-transparent text-muted-foreground hover:text-foreground"
                )
              }
            >
              <tab.icon className="h-4 w-4" />
              {tab.label}
            </NavLink>
          ))}
        </div>

        <Outlet />
      </div>
    </RequireRole>
  )
}
