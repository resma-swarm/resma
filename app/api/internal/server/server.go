// Package server implementa o HTTP server da API RESMA.
//
// Arquitetura: um único *http.ServeMux (Go 1.22+ pattern matching) com
// duas árvores de rotas:
//
//	/api/v1/*    — público, versionado, API key + scopes, OpenAPI
//	/api/*       — interno/UI, JWT, não-versionado
//	/api/sse/*   — streaming (0b.7)
//	/health      — infra, sem auth
//	/ready       — infra, sem auth
//	/swagger/*   — OpenAPI UI (0b.5, apenas /api/v1/*)
//
// Middlewares aplicados em cadeia:
//
//	CORS → Logging → Recovery → (JWT | APIKey | none) → handler
package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/resma-swarm/resma/app/api/internal/auth"
	"github.com/resma-swarm/resma/app/api/internal/config"
	"github.com/resma-swarm/resma/app/api/internal/db"
	"github.com/resma-swarm/resma/app/api/internal/docker"
	"github.com/resma-swarm/resma/app/api/internal/mlclient"
	"github.com/resma-swarm/resma/app/api/internal/sse"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Server encapsula o HTTP server e todas as dependências.
type Server struct {
	cfg     *config.Config
	db      *db.Store
	docker  *docker.Client
	auth    *auth.Service
	sse     *sse.Handler
	ml      *mlclient.Client
	log     *slog.Logger
	httpSrv *http.Server
}

// New cria um novo Server com todas as dependências injetadas.
func New(cfg *config.Config, database *db.Store, dc *docker.Client, authSvc *auth.Service) *Server {
	broker := sse.New()
	sseHandler := sse.NewHandler(broker, authSvc)
	mlClient := mlclient.New(cfg.MLURL, cfg.MLEnabled)
	return &Server{
		cfg:    cfg,
		db:     database,
		docker: dc,
		auth:   authSvc,
		sse:    sseHandler,
		ml:     mlClient,
		log:    slog.Default().With("component", "server"),
	}
}

// SSEHandler retorna o handler SSE para que o collector possa publicar eventos.
func (s *Server) SSEHandler() *sse.Handler {
	return s.sse
}

// Handler retorna o http.Handler com todas as rotas registradas.
// O caller pode usar com http.ServeMux ou diretamente com http.Server.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// --- infra (sem auth) ---
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /ready", s.handleReady)

	// --- /api/auth/* sem JWT (onboarding flow: status, onboarding, login, refresh) ---
	// Estes 4 endpoints precisam ser acessíveis sem autenticação.
	publicAuth := http.NewServeMux()
	s.registerPublicAuthRoutes(publicAuth)
	mux.Handle("/api/auth/status", s.corsMiddleware(s.recoveryMiddleware(s.loggingMiddleware(publicAuth))))
	mux.Handle("/api/auth/onboarding", s.corsMiddleware(s.recoveryMiddleware(s.loggingMiddleware(publicAuth))))
	mux.Handle("/api/auth/login", s.corsMiddleware(s.recoveryMiddleware(s.loggingMiddleware(publicAuth))))
	mux.Handle("/api/auth/refresh", s.corsMiddleware(s.recoveryMiddleware(s.loggingMiddleware(publicAuth))))

	// --- /api/sse/* (streaming, auth dual: cookie sse_session OU Authorization) ---
	// As rotas SSE têm auth própria (cookie ou bearer) e não usam JWTMiddleware.
	sseMux := http.NewServeMux()
	s.sse.RegisterRoutes(sseMux)
	mux.Handle("/api/sse/", s.corsMiddleware(s.recoveryMiddleware(s.loggingMiddleware(sseMux))))

	// --- /api/internal/* (ML sidecar, sem JWT — apenas rede Docker) ---
	internalML := http.NewServeMux()
	s.registerInternalMLRoutes(internalML)
	mux.Handle("/api/internal/", s.corsMiddleware(s.recoveryMiddleware(s.loggingMiddleware(internalML))))

	// --- /api/agent/* (multi-node agents, auth via bearer token compartilhado) ---
	// Fase 7 — agents em worker nodes fazem push de métricas/OOM/heartbeat.
	// Não usa JWT nem API key — usa RESMA_AGENT_TOKEN (constant-time compare).
	agentMux := http.NewServeMux()
	s.registerAgentRoutes(agentMux)
	mux.Handle("/api/agent/", s.agentTokenMiddleware(s.corsMiddleware(s.recoveryMiddleware(s.loggingMiddleware(agentMux)))))

	// --- /api/v1/* (público, API key) ---
	public := http.NewServeMux()
	s.registerPublicRoutes(public)
	mux.Handle("/api/v1/", s.auth.APIKeyMiddleware(s.corsMiddleware(s.recoveryMiddleware(s.loggingMiddleware(http.StripPrefix("/api/v1", public))))))

	// --- /api/* (interno, JWT) — registra depois de /api/v1/ e /api/auth/* sem JWT
	// para que o longest-path matching do Go 1.22 ServeMux funcione corretamente.
	internal := http.NewServeMux()
	s.registerInternalRoutes(internal)
	mux.Handle("/api/", s.auth.JWTMiddleware(s.corsMiddleware(s.recoveryMiddleware(s.loggingMiddleware(internal)))))

	// --- /swagger/* (swaggo UI, servido sem auth) ---
	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/swagger.json"),
	))

	// --- SPA frontend (estático) ---
	// Serve o frontend React buildado de /app/web (embutido na imagem Docker).
	// Em dev, o Vite roda separado na porta 5173 e o diretório não existe.
	if webDir := s.cfg.WebDir; webDir != "" {
		if _, err := os.Stat(webDir); err == nil {
			mux.Handle("/", s.spaHandler(webDir))
			s.log.Info("frontend estático servido", "dir", webDir)
		}
	}

	// Security headers em todas as respostas (Fase 2.7)
	return s.securityHeadersMiddleware(mux)
}

// spaHandler serve arquivos estáticos de um diretório com fallback para
// index.html (Single Page Application). Rotas /api/*, /health, /swagger/ já
// estão registradas no mux e têm precedência sobre o catch-all "/".
func (s *Server) spaHandler(webDir string) http.Handler {
	fs := http.FileServer(http.Dir(webDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(webDir, filepath.Clean(r.URL.Path))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// Arquivo não existe — serve index.html (SPA routing)
			http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
			return
		}
		// Desabilitar cache para index.html (sempre fresh); assets têm hash no nome
		if strings.HasSuffix(r.URL.Path, "index.html") {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}
		fs.ServeHTTP(w, r)
	})
}

// Start inicia o HTTP server em cfg.HTTPAddr.
func (s *Server) Start(ctx context.Context) error {
	s.httpSrv = &http.Server{
		Addr:              s.cfg.HTTPAddr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      0, // SSE exige sem timeout de escrita
		IdleTimeout:       120 * time.Second,
	}

	// Iniciar cleanup periódico de sessões SSE expiradas
	s.sse.StartCleanup(ctx)

	s.log.Info("servindo HTTP", "addr", s.cfg.HTTPAddr)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.httpSrv.Shutdown(shutdownCtx)
	}()

	if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// registerInternalRoutes registra todas as rotas /api/* (interno, JWT).
func (s *Server) registerInternalRoutes(mux *http.ServeMux) {
	// auth (4 sem auth + 3 com auth)
	s.registerAuthRoutes(mux)

	// dashboard, config, oom, change-log, alerts
	mux.HandleFunc("GET /api/dashboard", s.handleDashboard)
	mux.HandleFunc("GET /api/config", s.handleConfig)
	mux.HandleFunc("GET /api/alerts", s.handleAlerts)
	mux.HandleFunc("GET /api/oom-events", s.handleOOMEvents)
	mux.HandleFunc("GET /api/change-log", s.handleChangeLog)
	mux.HandleFunc("GET /api/change-log/{service}", s.handleChangeLogByService)

	// services
	s.registerServiceRoutes(mux)

	// nodes
	s.registerNodeRoutes(mux)

	// recommendations
	s.registerRecommendationRoutes(mux)

	// Right-Sizing Studio R5 — rollback watches admin
	s.registerRollbackWatchRoutes(mux)

	// Right-Sizing Studio R6 — simulate batch + export YAML
	s.registerSimulateRoutes(mux)

	// schedules
	s.registerScheduleRoutes(mux)

	// templates
	s.registerTemplateRoutes(mux)

	// storage
	s.registerStorageRoutes(mux)

	// api-keys (CRUD — admin via UI)
	s.registerAPIKeyRoutes(mux)

	// Fase 8 — users (CRUD — owner/admin via UI)
	s.registerUserRoutes(mux)

	// Fase 8 — settings (two-tier config — owner/admin via UI)
	s.registerSettingsRoutes(mux)

	// Fase 8 — prune (data management — owner/admin via UI)
	s.registerPruneRoutes(mux)

	// Fase 7 — agents admin + tasks + service health
	s.registerAgentAdminRoutes(mux)
	s.registerTaskRoutes(mux)
}

// registerPublicRoutes registra todas as rotas /api/v1/* (público, API key).
// Apenas endpoints de leitura no v1. Mutações no público são v1.1+.
// As rotas são registradas SEM o prefixo /v1 — o http.StripPrefix no
// Handler() remove /api/v1 antes de delegar para este mux.
func (s *Server) registerPublicRoutes(mux *http.ServeMux) {
	// services (leitura)
	mux.HandleFunc("GET /services", s.handleV1ListServices)
	mux.HandleFunc("GET /services/{name}/metrics", s.handleV1ServiceMetrics)
	mux.HandleFunc("GET /services/{name}/stats", s.handleV1ServiceStats)
	mux.HandleFunc("GET /services/{name}/containers", s.handleV1ServiceContainers)
	mux.HandleFunc("GET /containers/{id}/metrics", s.handleV1ContainerMetrics)
	mux.HandleFunc("GET /containers/{id}/stats", s.handleV1ContainerStats)

	// nodes (leitura)
	mux.HandleFunc("GET /nodes", s.handleV1ListNodes)
	mux.HandleFunc("GET /nodes/{id}", s.handleV1NodeDetail)
	mux.HandleFunc("GET /nodes/{id}/metrics", s.handleV1NodeMetrics)
	mux.HandleFunc("GET /nodes/{id}/services", s.handleV1NodeServices)
	mux.HandleFunc("GET /cluster", s.handleV1ClusterInfo)

	// oom, storage, recommendations, change-log (leitura)
	mux.HandleFunc("GET /oom-events", s.handleOOMEvents) // compartilhado
	mux.HandleFunc("GET /storage/trend", s.handleV1StorageTrend)
	mux.HandleFunc("GET /storage/volumes/growth", s.handleV1VolumeGrowth)
	mux.HandleFunc("GET /storage/volumes/{name}/growth", s.handleV1VolumeGrowthDetail)
	mux.HandleFunc("GET /recommendations", s.handleV1ListRecommendations)
	mux.HandleFunc("GET /recommendations/{service}", s.handleV1GetRecommendation)
	mux.HandleFunc("GET /recommendations/storage", s.handleV1StorageRecommendations)
	mux.HandleFunc("GET /change-log", s.handleChangeLog) // compartilhado

	// Fase 7 — agents + tasks + service health (leitura)
	mux.HandleFunc("GET /agents", s.handleListAgents)
	mux.HandleFunc("GET /agents/{node_id}", s.handleAgentDetail)
	mux.HandleFunc("GET /tasks", s.handleListTasks)
	mux.HandleFunc("GET /tasks/{service}", s.handleServiceTasks)
	mux.HandleFunc("GET /tasks/{service}/history", s.handleTaskHistory)
	mux.HandleFunc("GET /services/health", s.handleServicesHealth)
}
