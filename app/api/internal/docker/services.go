// Package docker — Swarm services.
//
// Melhoria 0b.3 vs Python: GetAllServiceResources() usa ServiceList() que
// retorna Spec.Resources em uma única chamada. No Python, cada serviço
// exigia um inspect() separado (N+1). Em Go, uma chamada resolve.
package docker

import (
	"context"
	"fmt"

	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
)

// GetServices lista todos os serviços do Swarm.
func (c *Client) GetServices(ctx context.Context) ([]ServiceInfo, error) {
	result, err := c.cli.ServiceList(ctx, client.ServiceListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]ServiceInfo, 0, len(result.Items))
	for _, svc := range result.Items {
		spec := svc.Spec
		image := ""
		if spec.TaskTemplate.ContainerSpec != nil {
			image = spec.TaskTemplate.ContainerSpec.Image
		}
		replicas := int64(1)
		if spec.Mode.Replicated != nil && spec.Mode.Replicated.Replicas != nil {
			replicas = int64(*spec.Mode.Replicated.Replicas)
		}
		out = append(out, ServiceInfo{
			ID:       svc.ID,
			Name:     spec.Name,
			Image:    image,
			Replicas: replicas,
			Labels:   spec.Labels,
		})
	}
	return out, nil
}

// GetServiceStatusMap retorna um map service_name -> "online"/"offline".
func (c *Client) GetServiceStatusMap(ctx context.Context) (ServiceStatusMap, error) {
	svcResult, err := c.cli.ServiceList(ctx, client.ServiceListOptions{})
	if err != nil {
		c.log.Error("erro ao listar serviços para status map", "err", err)
		return nil, err
	}
	serviceNames := make(map[string]string, len(svcResult.Items)) // id -> name
	for _, svc := range svcResult.Items {
		serviceNames[svc.ID] = svc.Spec.Name
	}

	taskResult, err := c.cli.TaskList(ctx, client.TaskListOptions{})
	if err != nil {
		c.log.Error("erro ao listar tasks para status map", "err", err)
		return nil, err
	}
	runningByService := make(map[string]int32)
	for _, task := range taskResult.Items {
		if task.Status.State == swarm.TaskStateRunning {
			runningByService[task.ServiceID]++
		}
	}

	result := make(ServiceStatusMap, len(serviceNames))
	for svcID, name := range serviceNames {
		if runningByService[svcID] > 0 {
			result[name] = "online"
		} else {
			result[name] = "offline"
		}
	}
	return result, nil
}

// GetServiceLabels retorna os labels de um serviço pelo nome.
func (c *Client) GetServiceLabels(ctx context.Context, serviceName string) (map[string]string, error) {
	result, err := c.cli.ServiceList(ctx, client.ServiceListOptions{})
	if err != nil {
		return nil, err
	}
	for _, svc := range result.Items {
		if svc.Spec.Name == serviceName {
			return svc.Spec.Labels, nil
		}
	}
	return nil, nil
}

// GetAllServiceResources retorna limites/reservations de TODOS os serviços
// em uma única chamada ServiceList (elimina N+1 do Python).
func (c *Client) GetAllServiceResources(ctx context.Context) (map[string]ServiceResources, error) {
	result, err := c.cli.ServiceList(ctx, client.ServiceListOptions{})
	if err != nil {
		c.log.Error("erro ao obter recursos de todos os serviços", "err", err)
		return nil, err
	}
	out := make(map[string]ServiceResources, len(result.Items))
	for _, svc := range result.Items {
		name := svc.Spec.Name
		if name == "" {
			continue
		}
		out[name] = extractResources(svc.Spec.TaskTemplate)
	}
	return out, nil
}

// GetServiceResources retorna limites/reservations de um serviço pelo nome.
func (c *Client) GetServiceResources(ctx context.Context, serviceName string) (ServiceResources, error) {
	result, err := c.cli.ServiceList(ctx, client.ServiceListOptions{})
	if err != nil {
		return ServiceResources{}, err
	}
	for _, svc := range result.Items {
		if svc.Spec.Name == serviceName {
			return extractResources(svc.Spec.TaskTemplate), nil
		}
	}
	return ServiceResources{}, nil
}

// extractResources extrai CPU/mem limit/reservation de um TaskSpec.
// Nota: Limits é *swarm.Limit, Reservations é *swarm.Resources (tipos diferentes).
func extractResources(taskTemplate swarm.TaskSpec) ServiceResources {
	var sr ServiceResources
	if taskTemplate.Resources == nil {
		return sr
	}
	res := taskTemplate.Resources
	if res.Limits != nil {
		sr.CPULimit = float64(res.Limits.NanoCPUs) / 1e9
		sr.MemLimit = res.Limits.MemoryBytes
	}
	if res.Reservations != nil {
		sr.CPUReservation = float64(res.Reservations.NanoCPUs) / 1e9
		sr.MemReservation = res.Reservations.MemoryBytes
	}
	return sr
}

// UpdateServiceResources atualiza limites/reservations de um serviço.
// Retorna um UpdateServiceResult estruturado.
func (c *Client) UpdateServiceResources(ctx context.Context, serviceName string,
	cpuLimit, memLimit *float64, cpuReservation, memReservation *int64,
) (*UpdateServiceResult, error) {
	result := &UpdateServiceResult{}

	inspectResult, err := c.cli.ServiceInspect(ctx, serviceName, client.ServiceInspectOptions{})
	if err != nil {
		result.Error = fmt.Sprintf("service %s not found: %v", serviceName, err)
		return result, err
	}
	svc := inspectResult.Service
	result.VersionBefore = int64(svc.Version.Index)

	spec := svc.Spec
	res := spec.TaskTemplate.Resources
	if res == nil {
		res = &swarm.ResourceRequirements{}
	}

	// Limits
	if cpuLimit != nil || memLimit != nil {
		limits := res.Limits
		if limits == nil {
			limits = &swarm.Limit{}
		}
		if cpuLimit != nil {
			limits.NanoCPUs = int64(*cpuLimit * 1e9)
		}
		if memLimit != nil {
			limits.MemoryBytes = int64(*memLimit)
		}
		res.Limits = limits
	}

	// Reservations (tipo *swarm.Resources, diferente de Limits)
	if cpuReservation != nil || memReservation != nil {
		reservations := res.Reservations
		if reservations == nil {
			reservations = &swarm.Resources{}
		}
		if cpuReservation != nil {
			reservations.NanoCPUs = int64(*cpuReservation * 1e9)
		}
		if memReservation != nil {
			reservations.MemoryBytes = *memReservation
		}
		res.Reservations = reservations
	}

	spec.TaskTemplate.Resources = res

	updateResult, err := c.cli.ServiceUpdate(ctx, svc.ID, client.ServiceUpdateOptions{
		Version: svc.Version,
		Spec:    spec,
	})
	if err != nil {
		result.Error = err.Error()
		c.log.Error("erro ao atualizar serviço", "name", serviceName, "err", err)
		return result, err
	}
	result.Warnings = updateResult.Warnings
	result.Success = true
	c.log.Info("recursos do serviço atualizados", "name", serviceName)

	// Buscar versão após update
	if updated, err := c.cli.ServiceInspect(ctx, serviceName, client.ServiceInspectOptions{}); err == nil {
		result.VersionAfter = int64(updated.Service.Version.Index)
	}

	return result, nil
}
