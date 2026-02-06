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
	"github.com/stretchr/testify/require"

	"github.com/fluxcd/flagger/pkg/logger"
)

func TestCORSMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins []string
		requestOrigin  string
		expectHeader   bool
		expectedOrigin string
	}{
		{
			name:           "wildcard allows any origin",
			allowedOrigins: []string{"*"},
			requestOrigin:  "http://example.com",
			expectHeader:   true,
			expectedOrigin: "http://example.com",
		},
		{
			name:           "specific origin allowed",
			allowedOrigins: []string{"http://example.com"},
			requestOrigin:  "http://example.com",
			expectHeader:   true,
			expectedOrigin: "http://example.com",
		},
		{
			name:           "origin not allowed",
			allowedOrigins: []string{"http://example.com"},
			requestOrigin:  "http://other.com",
			expectHeader:   false,
		},
		{
			name:           "multiple origins with match",
			allowedOrigins: []string{"http://example.com", "http://test.com"},
			requestOrigin:  "http://test.com",
			expectHeader:   true,
			expectedOrigin: "http://test.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := CORSMiddleware(tt.allowedOrigins)
			wrappedHandler := middleware(handler)

			req, err := http.NewRequest("GET", "/test", nil)
			require.NoError(t, err)
			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}

			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			if tt.expectHeader {
				assert.Equal(t, tt.expectedOrigin, rr.Header().Get("Access-Control-Allow-Origin"))
			} else {
				assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

func TestCORSMiddlewarePreflightRequest(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should not be called for OPTIONS request
		t.Error("Handler should not be called for OPTIONS request")
	})

	middleware := CORSMiddleware([]string{"*"})
	wrappedHandler := middleware(handler)

	req, err := http.NewRequest("OPTIONS", "/test", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://example.com")

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "http://example.com", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.NotEmpty(t, rr.Header().Get("Access-Control-Allow-Methods"))
}

func TestLoggingMiddleware(t *testing.T) {
	log, err := logger.NewLogger("debug")
	require.NoError(t, err)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := LoggingMiddleware(log)
	wrappedHandler := middleware(handler)

	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthMiddlewareDisabled(t *testing.T) {
	log, err := logger.NewLogger("debug")
	require.NoError(t, err)

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := AuthMiddleware(nil, log, false)
	wrappedHandler := middleware(handler)

	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.True(t, called, "Handler should be called when auth is disabled")
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthMiddlewareMissingHeader(t *testing.T) {
	log, err := logger.NewLogger("debug")
	require.NoError(t, err)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called without auth header")
	})

	middleware := AuthMiddleware(nil, log, true)
	wrappedHandler := middleware(handler)

	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthMiddlewareInvalidFormat(t *testing.T) {
	log, err := logger.NewLogger("debug")
	require.NoError(t, err)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with invalid auth format")
	})

	middleware := AuthMiddleware(nil, log, true)
	wrappedHandler := middleware(handler)

	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "InvalidFormat")

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestParseAllowedOrigins(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single origin",
			input:    "http://example.com",
			expected: []string{"http://example.com"},
		},
		{
			name:     "multiple origins",
			input:    "http://example.com,http://test.com",
			expected: []string{"http://example.com", "http://test.com"},
		},
		{
			name:     "origins with spaces",
			input:    "http://example.com, http://test.com , http://other.com",
			expected: []string{"http://example.com", "http://test.com", "http://other.com"},
		},
		{
			name:     "wildcard",
			input:    "*",
			expected: []string{"*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseAllowedOrigins(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
