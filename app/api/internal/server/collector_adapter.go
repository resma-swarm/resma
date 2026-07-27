// Package server — adapter que implementa collector.DataBuilder.
//
// O collector precisa de uma interface DataBuilder para construir payloads
// completos para SSE. O Server tem os métodos build* mas com assinaturas
// diferentes (retornam tipos concretos, não `any`). Este adapter faz a
// pontação sem acoplar o collector ao pacote server.
package server

import "context"

// CollectorDataBuilder adapta *Server para collector.DataBuilder.
// Os métodos chamam os build* do Server e convertem o retorno para any.
type CollectorDataBuilder struct {
	s *Server
}

// NewCollectorDataBuilder cria um adapter para o Server.
func NewCollectorDataBuilder(s *Server) *CollectorDataBuilder {
	return &CollectorDataBuilder{s: s}
}

// BuildDashboardData adapta buildDashboardData.
func (b *CollectorDataBuilder) BuildDashboardData(ctx context.Context) (map[string]any, error) {
	return b.s.buildDashboardData(ctx)
}

// BuildServicesList adapta buildServicesList.
func (b *CollectorDataBuilder) BuildServicesList(ctx context.Context) ([]map[string]any, error) {
	return b.s.buildServicesList(ctx)
}

// BuildServiceSparklines adapta buildServiceSparklines.
func (b *CollectorDataBuilder) BuildServiceSparklines(ctx context.Context, points int) (map[string][]map[string]any, error) {
	return b.s.buildServiceSparklines(ctx, points)
}

// BuildNodesList adapta buildNodesList.
func (b *CollectorDataBuilder) BuildNodesList(ctx context.Context) ([]map[string]any, error) {
	return b.s.buildNodesList(ctx)
}

// BuildClusterInfo adapta buildClusterInfo.
func (b *CollectorDataBuilder) BuildClusterInfo(ctx context.Context) (map[string]any, error) {
	return b.s.buildClusterInfo(ctx)
}

// BuildStorageSummary adapta buildStorageSummary.
func (b *CollectorDataBuilder) BuildStorageSummary(ctx context.Context) (map[string]any, error) {
	return b.s.buildStorageSummary(ctx)
}

// BuildTasksList adapta buildTasksList.
func (b *CollectorDataBuilder) BuildTasksList(ctx context.Context) ([]map[string]any, error) {
	return b.s.buildTasksList(ctx)
}

// BuildServicesHealth adapta buildServicesHealth.
// Retorna any para satisfazer a interface collector.DataBuilder (que usa any
// para evitar import cycle: server.ServiceHealth não é visível no collector).
func (b *CollectorDataBuilder) BuildServicesHealth(ctx context.Context, days int) (any, error) {
	return b.s.buildServicesHealth(ctx, days)
}

// BuildAgentsList adapta buildAgentsList.
func (b *CollectorDataBuilder) BuildAgentsList(ctx context.Context) ([]map[string]any, error) {
	return b.s.buildAgentsList(ctx)
}

// BuildRecommendations adapta buildRecommendations.
func (b *CollectorDataBuilder) BuildRecommendations(ctx context.Context) (any, error) {
	return b.s.buildRecommendations(ctx)
}

// BuildSchedulesList adapta buildSchedulesList (para scheduler.DataBuilder).
func (b *CollectorDataBuilder) BuildSchedulesList(ctx context.Context, status string) (any, error) {
	return b.s.buildSchedulesList(ctx, status)
}

// BuildChangeLog adapta buildChangeLog (para scheduler.DataBuilder).
func (b *CollectorDataBuilder) BuildChangeLog(ctx context.Context, service string, limit int32) (any, error) {
	return b.s.buildChangeLog(ctx, service, limit)
}

// BuildServiceDetailData adapta BuildServiceDetailData (para collector.DataBuilder).
// Constrói o payload completo do ServiceDetail (stats+metrics+containers+tasks+health)
// para publicação via SSE no tópico "service-detail/{name}".
func (b *CollectorDataBuilder) BuildServiceDetailData(ctx context.Context, service string) (map[string]any, error) {
	return b.s.BuildServiceDetailData(ctx, service)
}

// BuildContainerDetailData adapta BuildContainerDetailData (para collector.DataBuilder).
// Constrói o payload completo do ContainerDetail (stats+metrics+network) para
// publicação via SSE no tópico "container-detail/{id}".
func (b *CollectorDataBuilder) BuildContainerDetailData(ctx context.Context, containerID string) (map[string]any, error) {
	return b.s.BuildContainerDetailData(ctx, containerID)
}
