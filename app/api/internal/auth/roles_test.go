package auth

import "testing"

func TestIsValidRole(t *testing.T) {
	tests := []struct {
		role string
		want bool
	}{
		{"owner", true},
		{"admin", true},
		{"user", true},
		{"", false},
		{"superadmin", false},
		{"OWNER", false}, // case-sensitive
		{"root", false},
		{"guest", false},
	}
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			if got := IsValidRole(tt.role); got != tt.want {
				t.Errorf("IsValidRole(%q) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestRoleConstants(t *testing.T) {
	if RoleOwner != "owner" {
		t.Errorf("RoleOwner = %q, want 'owner'", RoleOwner)
	}
	if RoleAdmin != "admin" {
		t.Errorf("RoleAdmin = %q, want 'admin'", RoleAdmin)
	}
	if RoleUser != "user" {
		t.Errorf("RoleUser = %q, want 'user'", RoleUser)
	}
}
