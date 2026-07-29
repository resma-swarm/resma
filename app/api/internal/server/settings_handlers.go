// Package server — /api/settings/* handlers (Fase 8: Two-tier config).
//
// Endpoints para ler e atualizar parâmetros operacionais persistidos em
// app_settings (DB). Env vars permanecem como defaults/infra; DB sobrepõe.
//
// GET  /api/settings         — retorna 9 parâmetros operacionais
// PUT  /api/settings         — atualiza um ou mais parâmetros (valida tipos)
// GET  /api/config           — deprecated, redirect para /api/settings
package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/resma-swarm/resma/app/api/internal/auth"
)

// settingSpec define um parâmetro operacional persistível em DB.
type settingSpec struct {
	Key  string // chave em app_settings
	Type string // "int" | "float"
}

// operationalSettings lista os 9 parâmetros que migram para app_settings.
// Ordem determina a serialização no GET.
var operationalSettings = []settingSpec{
	{"collect_interval", "int"},
	{"retention_days", "int"},
	{"outlier_threshold", "float"},
	{"leak_r2_threshold", "float"},
	{"leak_daily_mb_threshold", "float"},
	{"analysis_window_days", "int"},
	{"cluster_interval", "int"},
	{"storage_interval", "int"},
	{"stale_service_days", "int"},
}

// registerSettingsRoutes registra as rotas de settings.
// Fase 8: leitura para todos (JWT), escrita para owner/admin.
func (s *Server) registerSettingsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/settings", s.handleGetSettings)

	// Escrita — owner/admin apenas
	rbac := auth.RequireRole(auth.RoleOwner, auth.RoleAdmin)
	mux.Handle("PUT /api/settings", rbac(http.HandlerFunc(s.handlePutSettings)))
}

// handleGetSettings retorna todos os parâmetros operacionais (DB ou default env).
func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	out := make(map[string]any, len(operationalSettings))
	for _, spec := range operationalSettings {
		val, err := s.db.GetSetting(r.Context(), spec.Key)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if val == "" {
			// Fallback para default da env var
			out[spec.Key] = s.envDefault(spec.Key)
			continue
		}
		// Converter string do DB para o tipo correto
		switch spec.Type {
		case "int":
			n, err := strconv.Atoi(val)
			if err != nil {
				out[spec.Key] = val
			} else {
				out[spec.Key] = n
			}
		case "float":
			f, err := strconv.ParseFloat(val, 64)
			if err != nil {
				out[spec.Key] = val
			} else {
				out[spec.Key] = f
			}
		default:
			out[spec.Key] = val
		}
	}
	writeOK(w, out)
}

// handlePutSettings atualiza um ou mais parâmetros operacionais.
// Body: {"collect_interval": 5, "retention_days": 60, ...}
// Apenas chaves conhecidas são aceitas; tipos são validados.
func (s *Server) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req) == 0 {
		writeError(w, http.StatusBadRequest, "no settings provided")
		return
	}

	updated := make(map[string]any)
	for key, val := range req {
		spec, ok := findSettingSpec(key)
		if !ok {
			writeError(w, http.StatusBadRequest, "unknown setting: "+key)
			return
		}
		strVal, err := validateSetting(spec, val)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := s.db.SetSetting(r.Context(), key, strVal); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		updated[key] = val
	}

	writeOK(w, map[string]any{"updated": updated})
}

// envDefault retorna o valor default de uma setting a partir da Config atual.
func (s *Server) envDefault(key string) any {
	switch key {
	case "collect_interval":
		return int(s.cfg.CollectInterval.Seconds())
	case "retention_days":
		return s.cfg.RetentionDays
	case "outlier_threshold":
		return s.cfg.OutlierThreshold
	case "leak_r2_threshold":
		return s.cfg.LeakR2Threshold
	case "leak_daily_mb_threshold":
		return s.cfg.LeakDailyMBThreshold
	case "analysis_window_days":
		return s.cfg.AnalysisWindowDays
	case "cluster_interval":
		return int(s.cfg.ClusterInterval.Seconds())
	case "storage_interval":
		return int(s.cfg.StorageInterval.Seconds())
	case "stale_service_days":
		return s.cfg.StaleServiceDays
	default:
		return nil
	}
}

// findSettingSpec busca um spec pelo key.
func findSettingSpec(key string) (settingSpec, bool) {
	for _, spec := range operationalSettings {
		if spec.Key == key {
			return spec, true
		}
	}
	return settingSpec{}, false
}

// validateSetting valida o tipo do valor e retorna a string para persistir.
func validateSetting(spec settingSpec, val any) (string, error) {
	switch spec.Type {
	case "int":
		switch v := val.(type) {
		case float64: // JSON numbers vêm como float64
			if v != float64(int(v)) {
				return "", &settingError{key: spec.Key, msg: "must be an integer"}
			}
			return strconv.Itoa(int(v)), nil
		case int:
			return strconv.Itoa(v), nil
		default:
			return "", &settingError{key: spec.Key, msg: "must be an integer"}
		}
	case "float":
		switch v := val.(type) {
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64), nil
		case int:
			return strconv.FormatFloat(float64(v), 'f', -1, 64), nil
		default:
			return "", &settingError{key: spec.Key, msg: "must be a number"}
		}
	default:
		return "", &settingError{key: spec.Key, msg: "unknown type"}
	}
}

// settingError implementa error com contexto da key.
type settingError struct {
	key string
	msg string
}

func (e *settingError) Error() string {
	return e.key + ": " + e.msg
}

// handleConfig permanece em misc_handlers.go (deprecated, mantido para compat).
// Fase 8: /api/settings é o endpoint canônico; /api/config retorna 3 campos originais.

// _ = strings para evitar import não usado se removido no futuro
var _ = strings.TrimSpace
