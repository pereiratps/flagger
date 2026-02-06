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
	"sort"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	flaggerv1 "github.com/fluxcd/flagger/pkg/apis/flagger/v1beta1"
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

// ListAllCanaries lists all canaries across all namespaces
func (h *Handler) ListAllCanaries(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	canaries, err := h.flaggerClient.FlaggerV1beta1().Canaries("").List(ctx, metav1.ListOptions{})
	if err != nil {
		h.logger.Errorf("Failed to list canaries: %v", err)
		h.writeError(w, "Failed to list canaries", http.StatusInternalServerError)
		return
	}

	items := make([]CanaryResponse, len(canaries.Items))
	for i, canary := range canaries.Items {
		items[i] = ToCanaryResponse(&canary)
	}

	response := CanaryListResponse{
		Items: items,
		Total: len(items),
	}

	h.writeJSON(w, response, http.StatusOK)
}

// ListCanariesInNamespace lists canaries in a specific namespace
func (h *Handler) ListCanariesInNamespace(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]

	ctx := context.Background()

	canaries, err := h.flaggerClient.FlaggerV1beta1().Canaries(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		h.logger.Errorf("Failed to list canaries in namespace %s: %v", namespace, err)
		h.writeError(w, fmt.Sprintf("Failed to list canaries in namespace %s", namespace), http.StatusInternalServerError)
		return
	}

	items := make([]CanaryResponse, len(canaries.Items))
	for i, canary := range canaries.Items {
		items[i] = ToCanaryResponse(&canary)
	}

	response := CanaryListResponse{
		Items: items,
		Total: len(items),
	}

	h.writeJSON(w, response, http.StatusOK)
}

// GetCanary gets a specific canary
func (h *Handler) GetCanary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]

	ctx := context.Background()

	canary, err := h.flaggerClient.FlaggerV1beta1().Canaries(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			h.writeError(w, fmt.Sprintf("Canary %s/%s not found", namespace, name), http.StatusNotFound)
			return
		}
		h.logger.Errorf("Failed to get canary %s/%s: %v", namespace, name, err)
		h.writeError(w, fmt.Sprintf("Failed to get canary %s/%s", namespace, name), http.StatusInternalServerError)
		return
	}

	response := ToCanaryResponse(canary)
	h.writeJSON(w, response, http.StatusOK)
}

// PromoteCanary promotes a canary to production
func (h *Handler) PromoteCanary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]

	var req PromoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default to empty request if body is invalid
		req = PromoteRequest{}
	}

	ctx := context.Background()

	canary, err := h.flaggerClient.FlaggerV1beta1().Canaries(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			h.writeError(w, fmt.Sprintf("Canary %s/%s not found", namespace, name), http.StatusNotFound)
			return
		}
		h.logger.Errorf("Failed to get canary %s/%s: %v", namespace, name, err)
		h.writeError(w, fmt.Sprintf("Failed to get canary %s/%s", namespace, name), http.StatusInternalServerError)
		return
	}

	// Set skipAnalysis if requested
	if req.SkipAnalysis {
		canary.Spec.SkipAnalysis = true
	}

	// Update the canary to trigger promotion
	// This is done by setting the phase to Promoting
	canary.Status.Phase = flaggerv1.CanaryPhasePromoting

	_, err = h.flaggerClient.FlaggerV1beta1().Canaries(namespace).UpdateStatus(ctx, canary, metav1.UpdateOptions{})
	if err != nil {
		h.logger.Errorf("Failed to promote canary %s/%s: %v", namespace, name, err)
		h.writeError(w, fmt.Sprintf("Failed to promote canary %s/%s", namespace, name), http.StatusInternalServerError)
		return
	}

	h.logger.Infof("Canary %s/%s promoted", namespace, name)
	h.writeJSON(w, map[string]string{"message": "Canary promotion initiated"}, http.StatusOK)
}

// PauseCanary pauses a canary rollout
func (h *Handler) PauseCanary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]

	ctx := context.Background()

	canary, err := h.flaggerClient.FlaggerV1beta1().Canaries(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			h.writeError(w, fmt.Sprintf("Canary %s/%s not found", namespace, name), http.StatusNotFound)
			return
		}
		h.logger.Errorf("Failed to get canary %s/%s: %v", namespace, name, err)
		h.writeError(w, fmt.Sprintf("Failed to get canary %s/%s", namespace, name), http.StatusInternalServerError)
		return
	}

	// Pause the canary by setting Suspend to true
	canary.Spec.Suspend = true

	_, err = h.flaggerClient.FlaggerV1beta1().Canaries(namespace).Update(ctx, canary, metav1.UpdateOptions{})
	if err != nil {
		h.logger.Errorf("Failed to pause canary %s/%s: %v", namespace, name, err)
		h.writeError(w, fmt.Sprintf("Failed to pause canary %s/%s", namespace, name), http.StatusInternalServerError)
		return
	}

	h.logger.Infof("Canary %s/%s paused", namespace, name)
	h.writeJSON(w, map[string]string{"message": "Canary paused"}, http.StatusOK)
}

// ResumeCanary resumes a paused canary rollout
func (h *Handler) ResumeCanary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]

	ctx := context.Background()

	canary, err := h.flaggerClient.FlaggerV1beta1().Canaries(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			h.writeError(w, fmt.Sprintf("Canary %s/%s not found", namespace, name), http.StatusNotFound)
			return
		}
		h.logger.Errorf("Failed to get canary %s/%s: %v", namespace, name, err)
		h.writeError(w, fmt.Sprintf("Failed to get canary %s/%s", namespace, name), http.StatusInternalServerError)
		return
	}

	// Resume the canary by setting Suspend to false
	canary.Spec.Suspend = false

	_, err = h.flaggerClient.FlaggerV1beta1().Canaries(namespace).Update(ctx, canary, metav1.UpdateOptions{})
	if err != nil {
		h.logger.Errorf("Failed to resume canary %s/%s: %v", namespace, name, err)
		h.writeError(w, fmt.Sprintf("Failed to resume canary %s/%s", namespace, name), http.StatusInternalServerError)
		return
	}

	h.logger.Infof("Canary %s/%s resumed", namespace, name)
	h.writeJSON(w, map[string]string{"message": "Canary resumed"}, http.StatusOK)
}

// RollbackCanary rolls back a canary deployment
func (h *Handler) RollbackCanary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]

	ctx := context.Background()

	canary, err := h.flaggerClient.FlaggerV1beta1().Canaries(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			h.writeError(w, fmt.Sprintf("Canary %s/%s not found", namespace, name), http.StatusNotFound)
			return
		}
		h.logger.Errorf("Failed to get canary %s/%s: %v", namespace, name, err)
		h.writeError(w, fmt.Sprintf("Failed to get canary %s/%s", namespace, name), http.StatusInternalServerError)
		return
	}

	// Trigger rollback by setting the phase to Failed
	canary.Status.Phase = flaggerv1.CanaryPhaseFailed
	canary.Status.CanaryWeight = 0

	_, err = h.flaggerClient.FlaggerV1beta1().Canaries(namespace).UpdateStatus(ctx, canary, metav1.UpdateOptions{})
	if err != nil {
		h.logger.Errorf("Failed to rollback canary %s/%s: %v", namespace, name, err)
		h.writeError(w, fmt.Sprintf("Failed to rollback canary %s/%s", namespace, name), http.StatusInternalServerError)
		return
	}

	h.logger.Infof("Canary %s/%s rolled back", namespace, name)
	h.writeJSON(w, map[string]string{"message": "Canary rollback initiated"}, http.StatusOK)
}

// GetCanaryMetrics gets metrics for a canary
func (h *Handler) GetCanaryMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]

	// For now, return a placeholder response
	// In a full implementation, this would query Prometheus or the metrics provider
	response := MetricsResponse{
		Canary: fmt.Sprintf("%s/%s", namespace, name),
		Metrics: map[string]interface{}{
			"requestSuccessRate": 99.5,
			"requestDuration":    250,
		},
	}

	h.writeJSON(w, response, http.StatusOK)
}

// GetCanaryEvents gets events for a canary
func (h *Handler) GetCanaryEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]

	ctx := context.Background()

	// Get events related to the canary
	events, err := h.kubeClient.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Canary", name),
	})
	if err != nil {
		h.logger.Errorf("Failed to get events for canary %s/%s: %v", namespace, name, err)
		h.writeError(w, fmt.Sprintf("Failed to get events for canary %s/%s", namespace, name), http.StatusInternalServerError)
		return
	}

	// Convert events to response format
	items := make([]EventResponse, len(events.Items))
	for i, event := range events.Items {
		items[i] = ToEventResponse(&event)
	}

	// Sort events by timestamp (newest first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp.After(items[j].Timestamp)
	})

	response := EventListResponse{
		Items: items,
		Total: len(items),
	}

	h.writeJSON(w, response, http.StatusOK)
}

// Healthz handles health check requests
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Readyz handles readiness check requests
func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	// Check if we can connect to Kubernetes API
	_, err := h.kubeClient.Discovery().ServerVersion()
	if err != nil {
		h.writeError(w, "Not ready", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// writeJSON writes a JSON response
func (h *Handler) writeJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Errorf("Failed to encode JSON response: %v", err)
	}
}

// writeError writes an error response
func (h *Handler) writeError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(code),
		Message: message,
		Code:    code,
	})
}
