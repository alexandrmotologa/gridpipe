package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Server encapsulates the HTTP server for gridpipe studio.
type Server struct {
	httpServer *http.Server
	Hub        *SSEHub
	Port       int
}

// NewServer initializes a new server on the specified port.
func NewServer(port int) *Server {
	if port <= 0 {
		port = 7575
	}

	hub := NewSSEHub()
	mux := http.NewServeMux()

	// Register API and SSE routes
	RegisterRoutes(mux, hub)

	// Register embedded web studio handler
	mux.Handle("/", WebHandler())

	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      mux,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 60 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		Hub:  hub,
		Port: port,
	}
}

// Start runs the HTTP listener and gracefully handles SIGINT/SIGTERM.
func (s *Server) Start() error {
	log.Printf("[gridpipe] Sheet Studio listening on http://localhost:%d", s.Port)

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case sig := <-stopChan:
		log.Printf("[gridpipe] Received signal %v, shutting down server...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}
}
