/*
Copyright 2020 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"

	clientset "github.com/fluxcd/flagger/pkg/client/clientset/versioned"
)

// ServerConfig holds the API server configuration
type ServerConfig struct {
	Port           string
	BearerToken    string
	AllowedOrigins []string
	EnableAPI      bool
}

// Server represents the API server
type Server struct {
	config        ServerConfig
	handler       *Handler
	logger        *zap.SugaredLogger
	httpServer    *http.Server
}

// NewServer creates a new API server
func NewServer(config ServerConfig, kubeClient kubernetes.Interface, flaggerClient clientset.Interface, logger *zap.SugaredLogger) *Server {
	handler := NewHandler(kubeClient, flaggerClient, logger)
	
	return &Server{
		config:  config,
		handler: handler,
		logger:  logger,
	}
}

// ListenAndServe starts the API server
func (s *Server) ListenAndServe(stopCh <-chan struct{}) {
	if !s.config.EnableAPI {
		s.logger.Info("API server is disabled")
		return
	}

	mux := http.NewServeMux()

	// Health endpoints
	mux.HandleFunc("/healthz", s.healthzHandler)
	mux.HandleFunc("/readyz", s.readyzHandler)

	// API v1 endpoints
	mux.HandleFunc("/api/v1/canaries", s.routeCanariesHandler)
	mux.HandleFunc("/api/v1/namespaces/", s.routeNamespacedHandler)

	// Apply middleware
	var handler http.Handler = mux
	
	// Apply CORS middleware
	if len(s.config.AllowedOrigins) > 0 {
		handler = CORSMiddleware(s.config.AllowedOrigins)(handler)
	}
	
	// Apply auth middleware
	if s.config.BearerToken != "" {
		handler = AuthMiddleware(s.config.BearerToken, s.logger)(handler)
	}
	
	// Apply logging middleware
	handler = LoggingMiddleware(s.logger)(handler)

	s.httpServer = &http.Server{
		Addr:         ":" + s.config.Port,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Infof("Starting API server on port %s", s.config.Port)

	// Run server in background
	go func() {
		if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			s.logger.Fatalf("API server crashed: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-stopCh
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Errorf("API server graceful shutdown failed: %v", err)
	} else {
		s.logger.Info("API server stopped")
	}
}

// Health check handlers
func (s *Server) healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) readyzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Route handlers
func (s *Server) routeCanariesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	s.handler.ListAllCanaries(w, r)
}

func (s *Server) routeNamespacedHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/namespaces/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		respondWithError(w, http.StatusNotFound, "Not found")
		return
	}

	// Extract resource
	resource := parts[1]

	if resource != "canaries" {
		respondWithError(w, http.StatusNotFound, "Not found")
		return
	}

	// Route based on path structure
	if len(parts) == 2 {
		// /api/v1/namespaces/{namespace}/canaries
		if r.Method != http.MethodGet {
			respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		s.handler.ListCanariesInNamespace(w, r)
		return
	}

	if len(parts) == 3 {
		// /api/v1/namespaces/{namespace}/canaries/{name}
		if r.Method != http.MethodGet {
			respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		s.handler.GetCanary(w, r)
		return
	}

	if len(parts) == 4 {
		// /api/v1/namespaces/{namespace}/canaries/{name}/{action}
		action := parts[3]
		
		switch action {
		case "metrics":
			if r.Method != http.MethodGet {
				respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			s.handler.GetCanaryMetrics(w, r)
		case "events":
			if r.Method != http.MethodGet {
				respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			s.handler.GetCanaryEvents(w, r)
		case "promote":
			if r.Method != http.MethodPost {
				respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			s.handler.PromoteCanary(w, r)
		case "pause":
			if r.Method != http.MethodPost {
				respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			s.handler.PauseCanary(w, r)
		case "resume":
			if r.Method != http.MethodPost {
				respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			s.handler.ResumeCanary(w, r)
		case "rollback":
			if r.Method != http.MethodPost {
				respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			s.handler.RollbackCanary(w, r)
		default:
			respondWithError(w, http.StatusNotFound, "Not found")
		}
		return
	}

	respondWithError(w, http.StatusNotFound, "Not found")
}
