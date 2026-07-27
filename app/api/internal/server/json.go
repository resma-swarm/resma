// Package server — JSON helpers.
package server

import (
	"encoding/json"
	"net/http"
)

// writeJSON escreve um valor como JSON no ResponseWriter.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Já escrevemos o status code; logar o erro mas não mudar a resposta.
		_, _ = w.Write([]byte(`{"error":"internal error"}`))
	}
}

// writeError escreve uma resposta de erro padronizada.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// writeOK escreve 200 OK com o valor.
func writeOK(w http.ResponseWriter, v any) {
	writeJSON(w, http.StatusOK, v)
}

// decodeJSON decodifica o body da request em v. Retorna false se houver erro
// (já escreve a resposta de erro).
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

// pathValue extrai um path parameter da request (Go 1.22+).
func pathValue(r *http.Request, key string) string {
	return r.PathValue(key)
}

// queryValue extrai um query parameter, ou "" se não existir.
func queryValue(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

// queryValueDefault extrai um query parameter com default.
func queryValueDefault(r *http.Request, key, def string) string {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	return v
}
