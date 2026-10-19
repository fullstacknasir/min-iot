package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"miniot/apps/ingestor/internal/mqtt"
	"miniot/apps/ingestor/internal/storage"
)

type Server struct {
	httpServer *http.Server
	store      *storage.Storage
	subscriber *mqtt.Subscriber
}

type HealthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}

func NewServer(port string, store *storage.Storage, subscriber *mqtt.Subscriber) *Server {
	s := &Server{
		store:      store,
		subscriber: subscriber,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	return s
}

func (s *Server) Start() error {
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// server stopped
		}
	}()
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	checks := make(map[string]string)
	isHealthy := true

	if s.store != nil && s.store.Pool != nil {
		if err := s.store.Pool.Ping(ctx); err != nil {
			checks["timescaledb"] = "unreachable"
			isHealthy = false
		} else {
			checks["timescaledb"] = "connected"
		}
	} else {
		checks["timescaledb"] = "uninitialized"
		isHealthy = false
	}

	if s.store != nil && s.store.Redis != nil {
		if err := s.store.Redis.Ping(ctx).Err(); err != nil {
			checks["redis"] = "unreachable"
			isHealthy = false
		} else {
			checks["redis"] = "connected"
		}
	} else {
		checks["redis"] = "uninitialized"
		isHealthy = false
	}

	if s.subscriber != nil && s.subscriber.IsConnected() {
		checks["mqtt_broker"] = "connected"
	} else {
		checks["mqtt_broker"] = "disconnected"
		isHealthy = false
	}

	statusCode := http.StatusOK
	statusText := "healthy"
	if !isHealthy {
		statusCode = http.StatusServiceUnavailable
		statusText = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(HealthResponse{
		Status:    statusText,
		Service:   "miniot-ingestor",
		Timestamp: time.Now().UTC(),
		Checks:    checks,
	})
}
