import { NavLink } from "react-router-dom"
import { Users, KeyRound, SlidersHorizontal, Database } from "lucide-react"
import {
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
} from "@/components/ui/sidebar"
import { usePermissions } from "@/hooks/use-permissions"

interface SettingsItem {
  to: string
  label: string
  icon: typeof Users
  show: boolean
}

export function NavSettings() {
  const perms = usePermissions()

  const items: SettingsItem[] = [
    { to: "/settings/users", label: "Usuários", icon: Users, show: perms.canManageUsers },
    { to: "/settings/api-keys", label: "API Keys", icon: KeyRound, show: perms.canManageAPIKeys },
    { to: "/settings/parameters", label: "Parâmetros", icon: SlidersHorizontal, show: perms.canManageSettings },
    { to: "/settings/data", label: "Dados", icon: Database, show: perms.canPrune },
  ]

  const visibleItems = items.filter((i) => i.show)
  if (visibleItems.length === 0) return null

  return (
    <SidebarMenu>
      {visibleItems.map((item) => (
        <SidebarMenuItem key={item.to}>
          <SidebarMenuButton asChild>
            <NavLink to={item.to}>
              <item.icon className="h-4.5 w-4.5" />
              <span>{item.label}</span>
            </NavLink>
          </SidebarMenuButton>
        </SidebarMenuItem>
      ))}
    </SidebarMenu>
  )
}
