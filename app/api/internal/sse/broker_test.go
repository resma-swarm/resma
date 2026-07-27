package sse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBrokerPublishSubscribe(t *testing.T) {
	broker := New()

	ch, cleanup := broker.Subscribe("test-topic")
	defer cleanup()

	// Publicar evento
	broker.Publish("test-topic", Event{Type: "metrics", Payload: map[string]int{"cpu": 50}})

	select {
	case event := <-ch:
		if event.Type != "metrics" {
			t.Errorf("event.Type = %q, want %q", event.Type, "metrics")
		}
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for event")
	}
}

func TestBrokerNonBlocking(t *testing.T) {
	broker := New()

	// Subscriber com buffer cheio (não consome)
	ch, cleanup := broker.Subscribe("slow-topic")
	defer cleanup()
	_ = ch

	// Publicar muitos eventos — não deve bloquear
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			broker.Publish("slow-topic", Event{Type: "metrics", Payload: i})
		}
	}()

	select {
	case <-done:
		// OK — publish não bloqueou
	case <-time.After(2 * time.Second):
		t.Error("Publish blocked on slow subscriber")
	}
}

func TestBrokerSubscriberCount(t *testing.T) {
	broker := New()

	if broker.SubscriberCount("topic") != 0 {
		t.Errorf("initial count = %d, want 0", broker.SubscriberCount("topic"))
	}

	_, cleanup1 := broker.Subscribe("topic")
	_, cleanup2 := broker.Subscribe("topic")

	if broker.SubscriberCount("topic") != 2 {
		t.Errorf("count = %d, want 2", broker.SubscriberCount("topic"))
	}

	cleanup1()
	if broker.SubscriberCount("topic") != 1 {
		t.Errorf("after cleanup1, count = %d, want 1", broker.SubscriberCount("topic"))
	}

	cleanup2()
	if broker.SubscriberCount("topic") != 0 {
		t.Errorf("after cleanup2, count = %d, want 0", broker.SubscriberCount("topic"))
	}
}

func TestBrokerSubscribedTopicsByPrefix(t *testing.T) {
	broker := New()

	// Sem subscribers — lista vazia
	if got := broker.SubscribedTopicsByPrefix("container-detail/"); len(got) != 0 {
		t.Errorf("initial prefix list = %v, want empty", got)
	}

	// Subscribers em tópicos dinâmicos container-detail/{id}
	_, cleanupA := broker.Subscribe("container-detail/abc123")
	_, cleanupB := broker.Subscribe("container-detail/def456")
	defer cleanupA()
	defer cleanupB()

	// Subscriber em tópico não relacionado (não deve aparecer)
	_, cleanupSvc := broker.Subscribe("service-detail/myapp")
	defer cleanupSvc()

	got := broker.SubscribedTopicsByPrefix("container-detail/")
	if len(got) != 2 {
		t.Fatalf("prefix list len = %d, want 2: %v", len(got), got)
	}
	want := map[string]bool{"container-detail/abc123": true, "container-detail/def456": true}
	for _, topic := range got {
		if !want[topic] {
			t.Errorf("unexpected topic %q in prefix list", topic)
		}
	}

	// Após cleanup, o tópico deve sair da lista
	cleanupA()
	got = broker.SubscribedTopicsByPrefix("container-detail/")
	if len(got) != 1 || got[0] != "container-detail/def456" {
		t.Errorf("after cleanupA, prefix list = %v, want [container-detail/def456]", got)
	}
}

func TestBrokerServeHTTP(t *testing.T) {
	broker := New()

	// Adicionar subscriber para garantir que o tópico existe
	_, cleanup := broker.Subscribe("test-http")
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/sse/test-http", nil)
	req = req.WithContext(context.Background())
	w := httptest.NewRecorder()

	// Goroutine para fechar o contexto após 100ms
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	broker.ServeHTTP(w, req, "test-http")

	resp := w.Result()
	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", resp.Header.Get("Content-Type"))
	}
	if resp.Header.Get("Cache-Control") != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", resp.Header.Get("Cache-Control"))
	}

	body := w.Body.String()
	if !strings.Contains(body, "event: connected") {
		t.Errorf("body should contain 'event: connected', got: %s", body)
	}
}
