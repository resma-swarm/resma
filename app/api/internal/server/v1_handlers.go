// Package server — /api/v1/* handlers (público, API key).
//
// A maioria dos endpoints públicos de leitura delega para os handlers
// internos (mesma lógica, shape idêntico). A diferença é apenas o middleware
// de auth (API key vs JWT) e as anotações swaggo (apenas /v1/).
package server

import (
	"net/http"
)

// handleV1ListServices delega para handleListServices.
// @Summary List all services
// @Tags services
// @Security APIKeyAuth
// @Produce json
// @Success 200 {array} map[string]any
// @Router /v1/services [get]
func (s *Server) handleV1ListServices(w http.ResponseWriter, r *http.Request) {
	s.handleListServices(w, r)
}

// handleV1ServiceMetrics delega para handleServiceMetrics.
// @Summary Get service metrics
// @Tags services
// @Security APIKeyAuth
// @Produce json
// @Param name path string true "Service name"
// @Param range query string false "Time range (e.g. 7d)" default(7d)
// @Success 200 {array} map[string]any
// @Router /v1/services/{name}/metrics [get]
func (s *Server) handleV1ServiceMetrics(w http.ResponseWriter, r *http.Request) {
	s.handleServiceMetrics(w, r)
}

// handleV1ServiceStats delega para handleServiceStats.
// @Summary Get service statistics
// @Tags services
// @Security APIKeyAuth
// @Produce json
// @Param name path string true "Service name"
// @Success 200 {object} map[string]any
// @Router /v1/services/{name}/stats [get]
func (s *Server) handleV1ServiceStats(w http.ResponseWriter, r *http.Request) {
	s.handleServiceStats(w, r)
}

// handleV1ServiceContainers delega para handleServiceContainers.
// @Summary Get service containers
// @Tags services
// @Security APIKeyAuth
// @Produce json
// @Param name path string true "Service name"
// @Success 200 {array} map[string]any
// @Router /v1/services/{name}/containers [get]
func (s *Server) handleV1ServiceContainers(w http.ResponseWriter, r *http.Request) {
	s.handleServiceContainers(w, r)
}

// handleV1ContainerMetrics delega para handleContainerMetrics.
// @Summary Get container metrics
// @Tags containers
// @Security APIKeyAuth
// @Produce json
// @Param id path string true "Container ID"
// @Param range query string false "Time range (e.g. 7d)" default(7d)
// @Success 200 {array} map[string]any
// @Router /v1/containers/{id}/metrics [get]
func (s *Server) handleV1ContainerMetrics(w http.ResponseWriter, r *http.Request) {
	s.handleContainerMetrics(w, r)
}

// handleV1ContainerStats delega para handleContainerStats.
// @Summary Get container statistics
// @Tags containers
// @Security APIKeyAuth
// @Produce json
// @Param id path string true "Container ID"
// @Success 200 {object} map[string]any
// @Router /v1/containers/{id}/stats [get]
func (s *Server) handleV1ContainerStats(w http.ResponseWriter, r *http.Request) {
	s.handleContainerStats(w, r)
}

// handleV1ListNodes delega para handleListNodes.
// @Summary List all nodes
// @Tags nodes
// @Security APIKeyAuth
// @Produce json
// @Success 200 {array} map[string]any
// @Router /v1/nodes [get]
func (s *Server) handleV1ListNodes(w http.ResponseWriter, r *http.Request) {
	s.handleListNodes(w, r)
}

// handleV1NodeDetail delega para handleNodeDetail.
// @Summary Get node details
// @Tags nodes
// @Security APIKeyAuth
// @Produce json
// @Param id path string true "Node ID"
// @Success 200 {object} map[string]any
// @Router /v1/nodes/{id} [get]
func (s *Server) handleV1NodeDetail(w http.ResponseWriter, r *http.Request) {
	s.handleNodeDetail(w, r)
}

// handleV1NodeMetrics delega para handleNodeMetrics.
// @Summary Get node metrics
// @Tags nodes
// @Security APIKeyAuth
// @Produce json
// @Param id path string true "Node ID"
// @Success 200 {array} map[string]any
// @Router /v1/nodes/{id}/metrics [get]
func (s *Server) handleV1NodeMetrics(w http.ResponseWriter, r *http.Request) {
	s.handleNodeMetrics(w, r)
}

// handleV1NodeServices delega para handleNodeServices.
// @Summary Get services running on a node
// @Tags nodes
// @Security APIKeyAuth
// @Produce json
// @Param id path string true "Node ID"
// @Success 200 {array} map[string]any
// @Router /v1/nodes/{id}/services [get]
func (s *Server) handleV1NodeServices(w http.ResponseWriter, r *http.Request) {
	s.handleNodeServices(w, r)
}

// handleV1ClusterInfo delega para handleClusterInfo.
// @Summary Get cluster info
// @Tags cluster
// @Security APIKeyAuth
// @Produce json
// @Success 200 {object} map[string]any
// @Router /v1/cluster [get]
func (s *Server) handleV1ClusterInfo(w http.ResponseWriter, r *http.Request) {
	s.handleClusterInfo(w, r)
}

// handleV1StorageTrend delega para handleStorageTrend.
// @Summary Get storage trend
// @Tags storage
// @Security APIKeyAuth
// @Produce json
// @Param days query int false "Number of days" default(7)
// @Success 200 {array} map[string]any
// @Router /v1/storage/trend [get]
func (s *Server) handleV1StorageTrend(w http.ResponseWriter, r *http.Request) {
	s.handleStorageTrend(w, r)
}

// handleV1VolumeGrowth delega para handleVolumeGrowth.
// @Summary Get volume growth (all volumes)
// @Tags storage
// @Security APIKeyAuth
// @Produce json
// @Param days query int false "Number of days" default(7)
// @Success 200 {array} map[string]any
// @Router /v1/storage/volumes/growth [get]
func (s *Server) handleV1VolumeGrowth(w http.ResponseWriter, r *http.Request) {
	s.handleVolumeGrowth(w, r)
}

// handleV1VolumeGrowthDetail delega para handleVolumeGrowthDetail.
// @Summary Get volume growth (specific volume)
// @Tags storage
// @Security APIKeyAuth
// @Produce json
// @Param name path string true "Volume name"
// @Param days query int false "Number of days" default(7)
// @Success 200 {array} map[string]any
// @Router /v1/storage/volumes/{name}/growth [get]
func (s *Server) handleV1VolumeGrowthDetail(w http.ResponseWriter, r *http.Request) {
	s.handleVolumeGrowthDetail(w, r)
}

// handleV1ListRecommendations delega para handleListRecommendations.
// @Summary List all recommendations
// @Tags recommendations
// @Security APIKeyAuth
// @Produce json
// @Success 200 {array} map[string]any
// @Router /v1/recommendations [get]
func (s *Server) handleV1ListRecommendations(w http.ResponseWriter, r *http.Request) {
	s.handleListRecommendations(w, r)
}

// handleV1GetRecommendation delega para handleGetRecommendation.
// @Summary Get recommendation for a service
// @Tags recommendations
// @Security APIKeyAuth
// @Produce json
// @Param service path string true "Service name"
// @Success 200 {object} map[string]any
// @Router /v1/recommendations/{service} [get]
func (s *Server) handleV1GetRecommendation(w http.ResponseWriter, r *http.Request) {
	s.handleGetRecommendation(w, r)
}

// handleV1StorageRecommendations delega para handleStorageRecommendations.
// @Summary Get storage recommendations
// @Tags recommendations
// @Security APIKeyAuth
// @Produce json
// @Success 200 {object} map[string]any
// @Router /v1/recommendations/storage [get]
func (s *Server) handleV1StorageRecommendations(w http.ResponseWriter, r *http.Request) {
	s.handleStorageRecommendations(w, r)
}
