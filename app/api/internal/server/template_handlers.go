// Package server — /api/templates/* handlers.
//
// Portado de backend/routers/templates.py.
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/resma/api/internal/auth"
	"github.com/resma/api/internal/db"
)

// templateResponse é o contrato JSON exposto para o frontend (snake_case).
// O campo stacks é serializado como array de strings (não como JSON string bruta),
// para corresponder ao que o React espera (Template.stacks: string[]).
type templateResponse struct {
	ID          int32    `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	YAMLContent string   `json:"yaml_content"`
	Stacks      []string `json:"stacks"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

// toTemplateResponse converte um db.Template para o contrato da API.
// O campo Stacks é armazenado no DB como um JSON array string (ex: '["postgres"]').
func toTemplateResponse(t db.Template) templateResponse {
	var stacks []string
	if t.Stacks != "" {
		_ = json.Unmarshal([]byte(t.Stacks), &stacks)
	}
	if stacks == nil {
		stacks = []string{}
	}
	return templateResponse{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		YAMLContent: t.YAMLContent,
		Stacks:      stacks,
		CreatedAt:   t.CreatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
		UpdatedAt:   t.UpdatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
	}
}

// stacksToJSONString serializa um array de strings para o formato armazenado no DB
// (JSON array string, ex: '["postgres","mysql"]').
func stacksToJSONString(stacks []string) string {
	if stacks == nil {
		stacks = []string{}
	}
	b, err := json.Marshal(stacks)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// registerTemplateRoutes registra as rotas de templates.
// Fase 8: rotas de escrita (POST/PUT/DELETE/apply) exigem role owner ou admin.
func (s *Server) registerTemplateRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/templates", s.handleListTemplates)
	mux.HandleFunc("GET /api/templates/{name}", s.handleGetTemplate)

	// Rotas de escrita — owner/admin apenas
	rbac := auth.RequireRole(auth.RoleOwner, auth.RoleAdmin)
	mux.Handle("POST /api/templates", rbac(http.HandlerFunc(s.handleCreateTemplate)))
	mux.Handle("PUT /api/templates/{template_id}", rbac(http.HandlerFunc(s.handleUpdateTemplate)))
	mux.Handle("DELETE /api/templates/{template_id}", rbac(http.HandlerFunc(s.handleDeleteTemplate)))
	mux.Handle("POST /api/templates/{name}/apply/{service}", rbac(http.HandlerFunc(s.handleApplyTemplate)))
}

// handleListTemplates lista todos os templates.
func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := s.db.ListTemplates(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]templateResponse, 0, len(templates))
	for _, t := range templates {
		out = append(out, toTemplateResponse(t))
	}
	writeOK(w, out)
}

// handleGetTemplate retorna um template pelo nome.
func (s *Server) handleGetTemplate(w http.ResponseWriter, r *http.Request) {
	name := pathValue(r, "name")
	tmpl, err := s.db.GetTemplateByName(r.Context(), name)
	if err != nil || tmpl == nil {
		writeError(w, http.StatusNotFound, "Template não encontrado")
		return
	}
	writeOK(w, toTemplateResponse(*tmpl))
}

// handleCreateTemplate cria um novo template.
func (s *Server) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		YAMLContent string   `json:"yaml_content"`
		Stacks      []string `json:"stacks"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Name == "" || req.YAMLContent == "" {
		writeError(w, http.StatusBadRequest, "name and yaml_content required")
		return
	}
	existing, _ := s.db.GetTemplateByName(r.Context(), req.Name)
	if existing != nil {
		writeError(w, http.StatusConflict, "Já existe um template com este nome")
		return
	}
	id, err := s.db.CreateTemplate(r.Context(), req.Name, req.Description, req.YAMLContent, stacksToJSONString(req.Stacks))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"success": true,
		"id":      id,
		"message": fmt.Sprintf("Template '%s' criado", req.Name),
	})
}

// handleUpdateTemplate atualiza um template existente.
func (s *Server) handleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	idStr := pathValue(r, "template_id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid template id")
		return
	}
	var req struct {
		Name        *string   `json:"name"`
		Description *string   `json:"description"`
		YAMLContent *string   `json:"yaml_content"`
		Stacks      *[]string `json:"stacks"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	tmpl, _ := s.db.GetTemplateByID(r.Context(), int32(id))
	if tmpl == nil {
		writeError(w, http.StatusNotFound, "Template não encontrado")
		return
	}
	// Só incluir `name` no UPDATE se mudou — o DuckDB tem limitação de
	// over-eager unique constraint checking em colunas UNIQUE.
	var name *string
	if req.Name != nil && *req.Name != tmpl.Name {
		// Verificar conflito de nome com outro template.
		existing, _ := s.db.GetTemplateByName(r.Context(), *req.Name)
		if existing != nil && existing.ID != tmpl.ID {
			writeError(w, http.StatusConflict, "Já existe um template com este nome")
			return
		}
		name = req.Name
	}
	var desc *string
	if req.Description != nil {
		desc = req.Description
	}
	var yaml *string
	if req.YAMLContent != nil {
		yaml = req.YAMLContent
	}
	var stacks *string
	if req.Stacks != nil {
		s := stacksToJSONString(*req.Stacks)
		stacks = &s
	}
	if err := s.db.UpdateTemplate(r.Context(), int32(id), name, desc, yaml, stacks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, map[string]any{"success": true, "message": "Template atualizado"})
}

// handleDeleteTemplate remove um template.
func (s *Server) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	idStr := pathValue(r, "template_id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid template id")
		return
	}
	if err := s.db.DeleteTemplate(r.Context(), int32(id)); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(w, map[string]any{"success": true, "message": "Template removido"})
}

// handleApplyTemplate aplica um template a um serviço.
func (s *Server) handleApplyTemplate(w http.ResponseWriter, r *http.Request) {
	name := pathValue(r, "name")
	service := pathValue(r, "service")
	// TODO 0b.8: parse YAML e aplicar via docker.UpdateServiceResources
	// Por enquanto, retornar stub
	writeOK(w, map[string]any{
		"success": true,
		"message": fmt.Sprintf("Template '%s' aplicado ao serviço '%s' (stub — ML sidecar 0b.8)", name, service),
	})
}
