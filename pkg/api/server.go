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

// ServerConfig holds the configuration for the API server
type ServerConfig struct {
	Port           string
	AuthEnabled    bool
	AllowedOrigins []string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
}

// Server represents the REST API server
type Server struct {
	config        ServerConfig
	kubeClient    kubernetes.Interface
	flaggerClient clientset.Interface
	logger        *zap.SugaredLogger
	httpServer    *http.Server
}

// NewServer creates a new API server
func NewServer(
	config ServerConfig,
	kubeClient kubernetes.Interface,
	flaggerClient clientset.Interface,
	logger *zap.SugaredLogger,
) *Server {
	return &Server{
		config:        config,
		kubeClient:    kubeClient,
		flaggerClient: flaggerClient,
		logger:        logger,
	}
}

// Start starts the API server
func (s *Server) Start(stopCh <-chan struct{}) error {
	// Create handler
	handler := NewHandler(s.kubeClient, s.flaggerClient, s.logger)

	// Setup router
	router := SetupRouter(handler)

	// Apply middleware
	var middlewares []Middleware
	middlewares = append(middlewares, LoggingMiddleware(s.logger))

	if len(s.config.AllowedOrigins) > 0 {
		middlewares = append(middlewares, CORSMiddleware(s.config.AllowedOrigins))
	}

	if s.config.AuthEnabled {
		middlewares = append(middlewares, AuthMiddleware(s.kubeClient, s.logger, true))
	}

	finalHandler := ApplyMiddleware(router, middlewares...)

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         ":" + s.config.Port,
		Handler:      finalHandler,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	s.logger.Infof("Starting REST API server on port %s", s.config.Port)

	// Start server in background
	go func() {
		if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			s.logger.Fatalf("REST API server crashed: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-stopCh

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Errorf("REST API server graceful shutdown failed: %v", err)
		return err
	}

	s.logger.Info("REST API server stopped")
	return nil
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Port:           "8081",
		AuthEnabled:    false,
		AllowedOrigins: []string{"*"},
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   60 * time.Second,
		IdleTimeout:    15 * time.Second,
	}
}

// ParseAllowedOrigins parses a comma-separated list of allowed origins
func ParseAllowedOrigins(origins string) []string {
	if origins == "" {
		return []string{}
	}

	parts := strings.Split(origins, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
