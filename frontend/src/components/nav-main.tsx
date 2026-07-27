import { NavLink } from "react-router-dom"
import {
  LayoutDashboard,
  Server,
  Boxes,
  ListTodo,
  ShieldAlert,
  Lightbulb,
  FileCode,
  CalendarClock,
  type LucideIcon,
} from "lucide-react"
import {
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
} from "@/components/ui/sidebar"

interface NavItem {
  to: string
  label: string
  icon: LucideIcon
}

const navItems: NavItem[] = [
  { to: "/", label: "Dashboard", icon: LayoutDashboard },
  { to: "/nodes", label: "Nodes", icon: Server },
  { to: "/services", label: "Serviços", icon: Boxes },
  { to: "/tasks", label: "Tasks", icon: ListTodo },
  { to: "/alerts", label: "Alertas", icon: ShieldAlert },
  { to: "/recommendations", label: "Recomendações", icon: Lightbulb },
  { to: "/templates", label: "Templates", icon: FileCode },
  { to: "/schedules", label: "Agendamentos", icon: CalendarClock },
]

export function NavMain() {
  return (
    <SidebarMenu>
      {navItems.map((item) => (
        <SidebarMenuItem key={item.to}>
          <SidebarMenuButton asChild>
            <NavLink to={item.to} end={item.to === "/"}>
              <item.icon className="h-4.5 w-4.5" />
              <span>{item.label}</span>
            </NavLink>
          </SidebarMenuButton>
        </SidebarMenuItem>
      ))}
    </SidebarMenu>
  )
}
