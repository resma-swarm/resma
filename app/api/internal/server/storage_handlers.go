// Package server — /api/storage/* handlers.
//
// Portado de backend/routers/storage.py.
package server

import (
	"net/http"
	"strconv"

	"github.com/resma-swarm/resma/app/api/internal/db"
)

// registerStorageRoutes registra as rotas de storage no mux interno.
func (s *Server) registerStorageRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/storage/summary", s.handleStorageSummary)
	mux.HandleFunc("GET /api/storage/trend", s.handleStorageTrend)
	mux.HandleFunc("GET /api/storage/volumes/growth", s.handleVolumeGrowth)
	mux.HandleFunc("GET /api/storage/volumes/{volume_name}/growth", s.handleVolumeGrowthDetail)
}

// handleStorageSummary retorna system df live + latest snapshot do DB.
func (s *Server) handleStorageSummary(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildStorageSummary(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, data)
}

// handleStorageTrend retorna trend de storage dos últimos N dias.
func (s *Server) handleStorageTrend(w http.ResponseWriter, r *http.Request) {
	daysStr := queryValueDefault(r, "days", "7")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 {
		days = 7
	}
	trend, err := s.db.GetStorageTrend(r.Context(), days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if trend == nil {
		trend = []db.StorageTrendPoint{}
	}
	writeOK(w, trend)
}

// handleVolumeGrowth retorna growth de todos os volumes.
func (s *Server) handleVolumeGrowth(w http.ResponseWriter, r *http.Request) {
	daysStr := queryValueDefault(r, "days", "7")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 {
		days = 7
	}
	growth, err := s.db.GetVolumeGrowthAll(r.Context(), days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if growth == nil {
		growth = []db.VolumeGrowthAllPoint{}
	}
	writeOK(w, growth)
}

// handleVolumeGrowthDetail retorna growth de um volume específico.
func (s *Server) handleVolumeGrowthDetail(w http.ResponseWriter, r *http.Request) {
	volumeName := pathValue(r, "volume_name")
	daysStr := queryValueDefault(r, "days", "7")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 {
		days = 7
	}
	growth, err := s.db.GetVolumeGrowth(r.Context(), volumeName, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if growth == nil {
		growth = []db.VolumeGrowthPoint{}
	}
	writeOK(w, growth)
}
