// Package auth — RBAC roles (Fase 8).
//
// Modelo de 3 roles com hierarquia simples:
//   - owner: acesso total (único, criado via onboarding)
//   - admin: mesmo acesso do owner (exceto deletar users)
//   - user:  somente leitura
package auth

// Role representa um papel de usuário no RBAC.
type Role string

const (
	RoleOwner Role = "owner" // Único, primeiro usuário via onboarding
	RoleAdmin Role = "admin" // Mesmo acesso do owner (exceto deletar users)
	RoleUser  Role = "user"  // Somente leitura
)

// IsValidRole valida se uma string é um role válido.
func IsValidRole(role string) bool {
	switch Role(role) {
	case RoleOwner, RoleAdmin, RoleUser:
		return true
	default:
		return false
	}
}
