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
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	authenticationv1 "k8s.io/api/authentication/v1"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Middleware represents a HTTP middleware
type Middleware func(http.Handler) http.Handler

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(logger *zap.SugaredLogger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			logger.Infof("Started %s %s", r.Method, r.URL.Path)

			next.ServeHTTP(w, r)

			duration := time.Since(start)
			logger.Infof("Completed %s %s in %v", r.Method, r.URL.Path, duration)
		})
	}
}

// CORSMiddleware adds CORS headers
func CORSMiddleware(allowedOrigins []string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				if origin != "" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else if len(allowedOrigins) > 0 {
					w.Header().Set("Access-Control-Allow-Origin", allowedOrigins[0])
				}
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AuthMiddleware validates bearer tokens and checks Kubernetes RBAC
func AuthMiddleware(kubeClient kubernetes.Interface, logger *zap.SugaredLogger, enabled bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth if disabled
			if !enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, "Missing Authorization header", http.StatusUnauthorized)
				return
			}

			// Check for Bearer token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeError(w, "Invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			token := parts[1]
			if token == "" {
				writeError(w, "Missing token", http.StatusUnauthorized)
				return
			}

			// Verify token with Kubernetes TokenReview API
			ctx := context.Background()

			// First, validate the token using TokenReview
			tokenReview := &authenticationv1.TokenReview{
				Spec: authenticationv1.TokenReviewSpec{
					Token: token,
				},
			}
			
			tokenResult, err := kubeClient.AuthenticationV1().TokenReviews().Create(ctx, tokenReview, metav1.CreateOptions{})
			if err != nil {
				logger.Errorf("Failed to verify token: %v", err)
				writeError(w, "Token verification failed", http.StatusUnauthorized)
				return
			}
			
			if !tokenResult.Status.Authenticated {
				writeError(w, "Invalid token", http.StatusUnauthorized)
				return
			}
			
			// Extract user identity from token review result
			user := tokenResult.Status.User.Username
			groups := tokenResult.Status.User.Groups
			
			// Create a SubjectAccessReview to check permissions
			// Extract resource info from the request path
			verb := getVerbFromMethod(r.Method, r.URL.Path)

			sar := &authv1.SubjectAccessReview{
				Spec: authv1.SubjectAccessReviewSpec{
					User:   user,
					Groups: groups,
					ResourceAttributes: &authv1.ResourceAttributes{
						Namespace: "*",
						Verb:      verb,
						Group:     "flagger.app",
						Resource:  "canaries",
					},
				},
			}

			result, err := kubeClient.AuthorizationV1().SubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
			if err != nil {
				logger.Errorf("Failed to verify authorization: %v", err)
				writeError(w, "Authorization verification failed", http.StatusInternalServerError)
				return
			}

			if !result.Status.Allowed {
				writeError(w, "Forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getVerbFromMethod converts HTTP method and path to Kubernetes RBAC verb
func getVerbFromMethod(method string, path string) string {
	switch method {
	case "GET":
		if strings.Contains(path, "/canaries/") && !strings.HasSuffix(path, "/canaries") {
			return "get"
		}
		return "list"
	case "POST":
		// POST to operation endpoints (promote, pause, resume, rollback) is an update
		if strings.Contains(path, "/promote") || strings.Contains(path, "/pause") ||
			strings.Contains(path, "/resume") || strings.Contains(path, "/rollback") {
			return "update"
		}
		return "create"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return "get"
	}
}

// writeError writes an error response
func writeError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(code),
		Message: message,
		Code:    code,
	})
}
