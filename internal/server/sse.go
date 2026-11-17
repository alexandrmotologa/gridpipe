package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/alexandrmotologa/gridpipe/internal/pipeline"
)

// SSEHub manages active client connections for Server-Sent Events.
type SSEHub struct {
	mu      sync.Mutex
	clients map[chan pipeline.PipelineEvent]bool
}

// NewSSEHub creates a new broadcast hub for SSE.
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[chan pipeline.PipelineEvent]bool),
	}
}

// Subscribe registers a new client channel.
func (h *SSEHub) Subscribe() chan pipeline.PipelineEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan pipeline.PipelineEvent, 32)
	h.clients[ch] = true
	return ch
}

// Unsubscribe removes a client channel.
func (h *SSEHub) Unsubscribe(ch chan pipeline.PipelineEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[ch]; ok {
		delete(h.clients, ch)
		close(ch)
	}
}

// Broadcast sends a pipeline event to all connected SSE clients.
func (h *SSEHub) Broadcast(e pipeline.PipelineEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- e:
		default:
			// Non-blocking drop if channel buffer full
		}
	}
}

// HandleSSE serves HTTP Server-Sent Events stream.
func (h *SSEHub) HandleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientChan := h.Subscribe()
	defer h.Unsubscribe(clientChan)

	// Send initial ping
	fmt.Fprintf(w, "event: ping\ndata: {\"connected\": true}\n\n")
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case event, open := <-clientChan:
			if !open {
				return
			}
			data, err := json.Marshal(event)
			if err == nil {
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Stage, string(data))
				flusher.Flush()
			}
		}
	}
}
