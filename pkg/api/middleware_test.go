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
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestAuthMiddleware_NoToken(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	middleware := AuthMiddleware("", logger.Sugar())
	wrappedHandler := middleware(handler)
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/canaries", nil)
	w := httptest.NewRecorder()
	
	wrappedHandler.ServeHTTP(w, req)
	
	// Should pass through when no token is configured
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	middleware := AuthMiddleware("secret-token", logger.Sugar())
	wrappedHandler := middleware(handler)
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/canaries", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	w := httptest.NewRecorder()
	
	wrappedHandler.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	middleware := AuthMiddleware("secret-token", logger.Sugar())
	wrappedHandler := middleware(handler)
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/canaries", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()
	
	wrappedHandler.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	middleware := AuthMiddleware("secret-token", logger.Sugar())
	wrappedHandler := middleware(handler)
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/canaries", nil)
	w := httptest.NewRecorder()
	
	wrappedHandler.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_HealthzEndpoint(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	middleware := AuthMiddleware("secret-token", logger.Sugar())
	wrappedHandler := middleware(handler)
	
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	
	wrappedHandler.ServeHTTP(w, req)
	
	// Health endpoints should bypass auth
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCORSMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	middleware := CORSMiddleware([]string{"http://example.com"})
	wrappedHandler := middleware(handler)
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/canaries", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	
	wrappedHandler.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "http://example.com", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_Wildcard(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	middleware := CORSMiddleware([]string{"*"})
	wrappedHandler := middleware(handler)
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/canaries", nil)
	w := httptest.NewRecorder()
	
	wrappedHandler.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	middleware := CORSMiddleware([]string{"*"})
	wrappedHandler := middleware(handler)
	
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/canaries", nil)
	w := httptest.NewRecorder()
	
	wrappedHandler.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoggingMiddleware(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	middleware := LoggingMiddleware(logger.Sugar())
	wrappedHandler := middleware(handler)
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/canaries", nil)
	w := httptest.NewRecorder()
	
	wrappedHandler.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}
