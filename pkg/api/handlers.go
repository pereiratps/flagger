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
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	clientset "github.com/fluxcd/flagger/pkg/client/clientset/versioned"
)

// Handler handles API requests
type Handler struct {
	kubeClient    kubernetes.Interface
	flaggerClient clientset.Interface
	logger        *zap.SugaredLogger
}

// NewHandler creates a new API handler
func NewHandler(kubeClient kubernetes.Interface, flaggerClient clientset.Interface, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		kubeClient:    kubeClient,
		flaggerClient: flaggerClient,
		logger:        logger,
	}
}

// ListAllCanaries handles GET /api/v1/canaries
func (h *Handler) ListAllCanaries(w http.ResponseWriter, r *http.Request) {
	canaries, err := h.flaggerClient.FlaggerV1beta1().Canaries("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		h.logger.Errorw("Failed to list canaries", "error", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list canaries: %v", err))
		return
	}

	items := make([]CanaryResponse, 0, len(canaries.Items))
	for _, canary := range canaries.Items {
		items = append(items, ConvertCanaryToResponse(&canary))
	}

	response := CanaryListResponse{
		Items: items,
		Total: len(items),
	}

	respondWithJSON(w, http.StatusOK, response)
}

// ListCanariesInNamespace handles GET /api/v1/namespaces/{namespace}/canaries
func (h *Handler) ListCanariesInNamespace(w http.ResponseWriter, r *http.Request) {
	namespace := extractPathParam(r.URL.Path, 3)
	if namespace == "" {
		respondWithError(w, http.StatusBadRequest, "Namespace is required")
		return
	}

	canaries, err := h.flaggerClient.FlaggerV1beta1().Canaries(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		h.logger.Errorw("Failed to list canaries", "namespace", namespace, "error", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list canaries: %v", err))
		return
	}

	items := make([]CanaryResponse, 0, len(canaries.Items))
	for _, canary := range canaries.Items {
		items = append(items, ConvertCanaryToResponse(&canary))
	}

	response := CanaryListResponse{
		Items: items,
		Total: len(items),
	}

	respondWithJSON(w, http.StatusOK, response)
}

// GetCanary handles GET /api/v1/namespaces/{namespace}/canaries/{name}
func (h *Handler) GetCanary(w http.ResponseWriter, r *http.Request) {
	namespace := extractPathParam(r.URL.Path, 3)
	name := extractPathParam(r.URL.Path, 5)

	if namespace == "" || name == "" {
		respondWithError(w, http.StatusBadRequest, "Namespace and name are required")
		return
	}

	canary, err := h.flaggerClient.FlaggerV1beta1().Canaries(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			respondWithError(w, http.StatusNotFound, fmt.Sprintf("Canary %s/%s not found", namespace, name))
			return
		}
		h.logger.Errorw("Failed to get canary", "namespace", namespace, "name", name, "error", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get canary: %v", err))
		return
	}

	response := ConvertCanaryToResponse(canary)
	respondWithJSON(w, http.StatusOK, response)
}

// GetCanaryMetrics handles GET /api/v1/namespaces/{namespace}/canaries/{name}/metrics
func (h *Handler) GetCanaryMetrics(w http.ResponseWriter, r *http.Request) {
	namespace := extractPathParam(r.URL.Path, 3)
	name := extractPathParam(r.URL.Path, 5)

	if namespace == "" || name == "" {
		respondWithError(w, http.StatusBadRequest, "Namespace and name are required")
		return
	}

	// Verify canary exists
	canary, err := h.flaggerClient.FlaggerV1beta1().Canaries(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			respondWithError(w, http.StatusNotFound, fmt.Sprintf("Canary %s/%s not found", namespace, name))
			return
		}
		h.logger.Errorw("Failed to get canary", "namespace", namespace, "name", name, "error", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get canary: %v", err))
		return
	}

	// Return metrics from canary analysis configuration
	metrics := make([]MetricResult, 0)
	if canary.GetAnalysis() != nil && canary.GetAnalysis().Metrics != nil {
		for _, m := range canary.GetAnalysis().Metrics {
			metrics = append(metrics, MetricResult{
				Name:     m.Name,
				Metadata: map[string]interface{}{
					"interval": m.Interval,
					"query":    m.Query,
				},
			})
		}
	}

	response := MetricsResponse{
		Name:    name,
		Metrics: metrics,
	}

	respondWithJSON(w, http.StatusOK, response)
}

// GetCanaryEvents handles GET /api/v1/namespaces/{namespace}/canaries/{name}/events
func (h *Handler) GetCanaryEvents(w http.ResponseWriter, r *http.Request) {
	namespace := extractPathParam(r.URL.Path, 3)
	name := extractPathParam(r.URL.Path, 5)

	if namespace == "" || name == "" {
		respondWithError(w, http.StatusBadRequest, "Namespace and name are required")
		return
	}

	// Verify canary exists
	_, err := h.flaggerClient.FlaggerV1beta1().Canaries(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			respondWithError(w, http.StatusNotFound, fmt.Sprintf("Canary %s/%s not found", namespace, name))
			return
		}
		h.logger.Errorw("Failed to get canary", "namespace", namespace, "name", name, "error", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get canary: %v", err))
		return
	}

	// Get events related to the canary
	events, err := h.kubeClient.CoreV1().Events(namespace).List(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Canary", name),
	})
	if err != nil {
		h.logger.Errorw("Failed to get events", "namespace", namespace, "name", name, "error", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get events: %v", err))
		return
	}

	eventItems := make([]EventItem, 0, len(events.Items))
	for _, event := range events.Items {
		eventItems = append(eventItems, EventItem{
			Type:      event.Type,
			Reason:    event.Reason,
			Message:   event.Message,
			Timestamp: event.LastTimestamp.Time,
			Source:    event.Source.Component,
		})
	}

	response := EventsResponse{
		Events: eventItems,
		Total:  len(eventItems),
	}

	respondWithJSON(w, http.StatusOK, response)
}

// PromoteCanary handles POST /api/v1/namespaces/{namespace}/canaries/{name}/promote
func (h *Handler) PromoteCanary(w http.ResponseWriter, r *http.Request) {
	namespace := extractPathParam(r.URL.Path, 3)
	name := extractPathParam(r.URL.Path, 5)

	if namespace == "" || name == "" {
		respondWithError(w, http.StatusBadRequest, "Namespace and name are required")
		return
	}

	// Parse request body
	var req PromoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If body is empty or invalid, use defaults
		req = PromoteRequest{}
	}

	// Get the canary
	canary, err := h.flaggerClient.FlaggerV1beta1().Canaries(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			respondWithError(w, http.StatusNotFound, fmt.Sprintf("Canary %s/%s not found", namespace, name))
			return
		}
		h.logger.Errorw("Failed to get canary", "namespace", namespace, "name", name, "error", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get canary: %v", err))
		return
	}

	// Set skipAnalysis if requested
	if req.SkipAnalysis {
		canary.Spec.SkipAnalysis = true
	}

	// Update the canary to trigger promotion
	_, err = h.flaggerClient.FlaggerV1beta1().Canaries(namespace).Update(context.TODO(), canary, metav1.UpdateOptions{})
	if err != nil {
		h.logger.Errorw("Failed to update canary", "namespace", namespace, "name", name, "error", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to promote canary: %v", err))
		return
	}

	h.logger.Infow("Canary promotion requested", "namespace", namespace, "name", name, "skipAnalysis", req.SkipAnalysis)
	
	respondWithJSON(w, http.StatusOK, OperationResponse{
		Success: true,
		Message: "Canary promotion initiated",
	})
}

// PauseCanary handles POST /api/v1/namespaces/{namespace}/canaries/{name}/pause
func (h *Handler) PauseCanary(w http.ResponseWriter, r *http.Request) {
	namespace := extractPathParam(r.URL.Path, 3)
	name := extractPathParam(r.URL.Path, 5)

	if namespace == "" || name == "" {
		respondWithError(w, http.StatusBadRequest, "Namespace and name are required")
		return
	}

	// Get the canary
	canary, err := h.flaggerClient.FlaggerV1beta1().Canaries(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			respondWithError(w, http.StatusNotFound, fmt.Sprintf("Canary %s/%s not found", namespace, name))
			return
		}
		h.logger.Errorw("Failed to get canary", "namespace", namespace, "name", name, "error", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get canary: %v", err))
		return
	}

	// Set suspend to true
	canary.Spec.Suspend = true

	// Update the canary
	_, err = h.flaggerClient.FlaggerV1beta1().Canaries(namespace).Update(context.TODO(), canary, metav1.UpdateOptions{})
	if err != nil {
		h.logger.Errorw("Failed to update canary", "namespace", namespace, "name", name, "error", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to pause canary: %v", err))
		return
	}

	h.logger.Infow("Canary paused", "namespace", namespace, "name", name)
	
	respondWithJSON(w, http.StatusOK, OperationResponse{
		Success: true,
		Message: "Canary paused",
	})
}

// ResumeCanary handles POST /api/v1/namespaces/{namespace}/canaries/{name}/resume
func (h *Handler) ResumeCanary(w http.ResponseWriter, r *http.Request) {
	namespace := extractPathParam(r.URL.Path, 3)
	name := extractPathParam(r.URL.Path, 5)

	if namespace == "" || name == "" {
		respondWithError(w, http.StatusBadRequest, "Namespace and name are required")
		return
	}

	// Get the canary
	canary, err := h.flaggerClient.FlaggerV1beta1().Canaries(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			respondWithError(w, http.StatusNotFound, fmt.Sprintf("Canary %s/%s not found", namespace, name))
			return
		}
		h.logger.Errorw("Failed to get canary", "namespace", namespace, "name", name, "error", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get canary: %v", err))
		return
	}

	// Set suspend to false
	canary.Spec.Suspend = false

	// Update the canary
	_, err = h.flaggerClient.FlaggerV1beta1().Canaries(namespace).Update(context.TODO(), canary, metav1.UpdateOptions{})
	if err != nil {
		h.logger.Errorw("Failed to update canary", "namespace", namespace, "name", name, "error", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to resume canary: %v", err))
		return
	}

	h.logger.Infow("Canary resumed", "namespace", namespace, "name", name)
	
	respondWithJSON(w, http.StatusOK, OperationResponse{
		Success: true,
		Message: "Canary resumed",
	})
}

// RollbackCanary handles POST /api/v1/namespaces/{namespace}/canaries/{name}/rollback
func (h *Handler) RollbackCanary(w http.ResponseWriter, r *http.Request) {
	namespace := extractPathParam(r.URL.Path, 3)
	name := extractPathParam(r.URL.Path, 5)

	if namespace == "" || name == "" {
		respondWithError(w, http.StatusBadRequest, "Namespace and name are required")
		return
	}

	// Get the canary
	canary, err := h.flaggerClient.FlaggerV1beta1().Canaries(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			respondWithError(w, http.StatusNotFound, fmt.Sprintf("Canary %s/%s not found", namespace, name))
			return
		}
		h.logger.Errorw("Failed to get canary", "namespace", namespace, "name", name, "error", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get canary: %v", err))
		return
	}

	// Annotate canary to trigger rollback by setting the last applied spec to empty
	// This will cause the controller to detect a change and rollback
	if canary.Annotations == nil {
		canary.Annotations = make(map[string]string)
	}
	canary.Annotations["flagger.app/rollback"] = "true"

	// Update the canary
	_, err = h.flaggerClient.FlaggerV1beta1().Canaries(namespace).Update(context.TODO(), canary, metav1.UpdateOptions{})
	if err != nil {
		h.logger.Errorw("Failed to update canary", "namespace", namespace, "name", name, "error", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to rollback canary: %v", err))
		return
	}

	h.logger.Infow("Canary rollback requested", "namespace", namespace, "name", name)
	
	respondWithJSON(w, http.StatusOK, OperationResponse{
		Success: true,
		Message: "Canary rollback initiated",
	})
}

// Helper functions

func extractPathParam(path string, index int) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if index < len(parts) {
		return parts[index]
	}
	return ""
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Internal server error"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, ErrorResponse{
		Error:   http.StatusText(code),
		Message: message,
		Code:    code,
	})
}
