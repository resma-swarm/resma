import { Navigate, useLocation } from "react-router-dom"
import { usePermissions, type Role } from "@/hooks/use-permissions"

interface RequireRoleProps {
  allowedRoles: Role[]
  children: React.ReactNode
  fallback?: React.ReactNode
}

/**
 * Componente de proteção de rota baseado em RBAC.
 * Se o usuário não tem o role permitido, redireciona para "/".
 *
 * Uso:
 * <RequireRole allowedRoles={["owner", "admin"]}>
 *   <SettingsPage />
 * </RequireRole>
 */
export function RequireRole({ allowedRoles, children, fallback }: RequireRoleProps) {
  const { role } = usePermissions()
  const location = useLocation()

  if (!allowedRoles.includes(role)) {
    if (fallback) return <>{fallback}</>
    return <Navigate to="/" state={{ from: location }} replace />
  }

  return <>{children}</>
}
