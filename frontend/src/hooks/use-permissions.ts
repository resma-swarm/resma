import { useAuth } from "@/contexts/AuthContext"

export type Role = "owner" | "admin" | "user"

/**
 * Hook para verificar permissões de RBAC no frontend.
 * Baseado no role do usuário autenticado.
 *
 * Roles (hierarquia): owner > admin > user
 * - owner: acesso total (único, criado via onboarding)
 * - admin: mesmo acesso do owner (exceto deletar users)
 * - user: somente leitura
 */
export function usePermissions() {
  const { user } = useAuth()
  const role = (user?.role ?? "user") as Role

  return {
    role,
    isOwner: role === "owner",
    isAdmin: role === "admin",
    isUser: role === "user",
    canWrite: role === "owner" || role === "admin",
    canManageUsers: role === "owner" || role === "admin",
    canDeleteUsers: role === "owner",
    canManageAPIKeys: role === "owner" || role === "admin",
    canManageSettings: role === "owner" || role === "admin",
    canPrune: role === "owner" || role === "admin",
  }
}
