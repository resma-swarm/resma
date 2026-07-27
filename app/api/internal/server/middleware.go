// Package server — middlewares HTTP (CORS, security headers, logging, recovery).
package server

import (
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

// corsMiddleware adiciona headers CORS restritivos baseados em cfg.CORSOrigins.
// Em produção, origins devem ser explicitamente definidas via RESMA_CORS_ORIGINS.
// Para SSE (/api/sse/*), inclui Access-Control-Allow-Credentials: true (necessário
// para cookie sse_session HttpOnly via EventSource withCredentials).
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	allowedOrigins := make(map[string]bool, len(s.cfg.CORSOrigins))
	for _, o := range s.cfg.CORSOrigins {
		allowedOrigins[strings.TrimSpace(o)] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			// Credentials true é necessário para SSE cookie e para fetch com credentials: "include"
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// securityHeadersMiddleware adiciona headers de segurança em todas as respostas.
// X-Content-Type-Options, X-Frame-Options, X-XSS-Protection, Referrer-Policy.
// Strict-Transport-Security apenas em HTTPS (r.TLS != nil).
func (s *Server) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware loga cada request com método, path, status e duração.
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(ww, r)
		s.log.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.status,
			"dur_ms", time.Since(start).Milliseconds(),
		)
	})
}

// recoveryMiddleware captura panics e retorna 500 em vez de crashar.
func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("panic recuperado",
					"error", rec,
					"path", r.URL.Path,
					"stack", string(debug.Stack()),
				)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// statusWriter wraps http.ResponseWriter para capturar o status code.
type statusWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (w *statusWriter) WriteHeader(status int) {
	if w.wrote {
		return
	}
	w.status = status
	w.wrote = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.wrote = true
	}
	return w.ResponseWriter.Write(b)
}

// Flush implementa http.Flusher para que SSE (Server-Sent Events) funcione
// através do loggingMiddleware. Sem isso, o type assertion w.(http.Flusher)
// falha no broker SSE e retorna 500 "streaming not supported".
func (w *statusWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
