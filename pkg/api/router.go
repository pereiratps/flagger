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
	"net/http"

	"github.com/gorilla/mux"
)

// SetupRouter sets up the API router with all routes
func SetupRouter(handler *Handler) *mux.Router {
	router := mux.NewRouter()
	
	// API v1 routes
	apiV1 := router.PathPrefix("/api/v1").Subrouter()
	
	// List all canaries
	apiV1.HandleFunc("/canaries", handler.ListAllCanaries).Methods("GET")
	
	// Namespace-specific routes
	apiV1.HandleFunc("/namespaces/{namespace}/canaries", handler.ListCanariesInNamespace).Methods("GET")
	apiV1.HandleFunc("/namespaces/{namespace}/canaries/{name}", handler.GetCanary).Methods("GET")
	
	// Canary operations
	apiV1.HandleFunc("/namespaces/{namespace}/canaries/{name}/promote", handler.PromoteCanary).Methods("POST")
	apiV1.HandleFunc("/namespaces/{namespace}/canaries/{name}/pause", handler.PauseCanary).Methods("POST")
	apiV1.HandleFunc("/namespaces/{namespace}/canaries/{name}/resume", handler.ResumeCanary).Methods("POST")
	apiV1.HandleFunc("/namespaces/{namespace}/canaries/{name}/rollback", handler.RollbackCanary).Methods("POST")
	
	// Metrics and events
	apiV1.HandleFunc("/namespaces/{namespace}/canaries/{name}/metrics", handler.GetCanaryMetrics).Methods("GET")
	apiV1.HandleFunc("/namespaces/{namespace}/canaries/{name}/events", handler.GetCanaryEvents).Methods("GET")
	
	// Health checks
	router.HandleFunc("/healthz", handler.Healthz).Methods("GET")
	router.HandleFunc("/readyz", handler.Readyz).Methods("GET")
	
	return router
}

// ApplyMiddleware applies middleware to a handler
func ApplyMiddleware(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
