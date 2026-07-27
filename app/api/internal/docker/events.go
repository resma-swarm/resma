// Package docker — Docker events listener.
//
// Mantém o padrão do Python: um subscriber de eventos que notifica containers
// start/stop/die/destroy para adicionar/remover do cache automaticamente.
package docker

import (
	"context"

	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
)

// EventType classifica o tipo de evento de container.
type EventType string

const (
	EventContainerStart   EventType = "start"
	EventContainerStop    EventType = "stop"
	EventContainerDie     EventType = "die"
	EventContainerDestroy EventType = "destroy"
)

// ContainerEvent é um evento de container parseado.
type ContainerEvent struct {
	Type        EventType
	ContainerID string
	ExitCode    string
	Service     string
}

// ListenEvents abre um stream de eventos do Docker e envia ContainerEvents
// para o channel retornado. O channel é fechado quando o ctx é cancelado
// ou o stream termina.
//
// O caller deve fazer range sobre o channel e processar eventos. Para
// container start: AddContainerToCache + StartStatsStream. Para stop/die/
// destroy: RemoveContainerFromCache.
func (c *Client) ListenEvents(ctx context.Context) (<-chan ContainerEvent, error) {
	filters := make(client.Filters).Add("type", "container")
	result := c.cli.Events(ctx, client.EventsListOptions{
		Filters: filters,
	})

	out := make(chan ContainerEvent, 64)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-result.Messages:
				if !ok {
					// Stream ended; check error channel
					select {
					case err := <-result.Err:
						if err != nil && ctx.Err() == nil {
							c.log.Warn("docker events stream encerrado", "err", err)
						}
					default:
					}
					return
				}
				ce := parseEvent(msg)
				if ce.ContainerID != "" {
					select {
					case out <- ce:
					case <-ctx.Done():
						return
					}
				}
			case err := <-result.Err:
				if err != nil && ctx.Err() == nil {
					c.log.Warn("docker events erro", "err", err)
				}
				return
			}
		}
	}()
	return out, nil
}

// parseEvent converte um events.Message em ContainerEvent.
func parseEvent(msg events.Message) ContainerEvent {
	ce := ContainerEvent{
		Type:        EventType(msg.Action),
		ContainerID: msg.Actor.ID,
	}
	if msg.Actor.Attributes != nil {
		ce.ExitCode = msg.Actor.Attributes["exitCode"]
		ce.Service = msg.Actor.Attributes["com.docker.swarm.service.name"]
	}
	return ce
}
