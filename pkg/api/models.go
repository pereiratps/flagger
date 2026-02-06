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
	"time"

	flaggerv1 "github.com/fluxcd/flagger/pkg/apis/flagger/v1beta1"
)

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// CanaryListResponse represents a list of canaries
type CanaryListResponse struct {
	Items []CanaryResponse `json:"items"`
	Total int              `json:"total"`
}

// CanaryResponse represents a canary resource in API format
type CanaryResponse struct {
	Metadata CanaryMetadata `json:"metadata"`
	Spec     interface{}    `json:"spec"`
	Status   CanaryStatus   `json:"status"`
}

// CanaryMetadata represents canary metadata
type CanaryMetadata struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	CreatedAt         time.Time         `json:"createdAt"`
	Labels            map[string]string `json:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty"`
	ResourceVersion   string            `json:"resourceVersion,omitempty"`
	UID               string            `json:"uid,omitempty"`
	Generation        int64             `json:"generation,omitempty"`
}

// CanaryStatus represents canary status
type CanaryStatus struct {
	Phase              string    `json:"phase"`
	CanaryWeight       int       `json:"canaryWeight"`
	FailedChecks       int       `json:"failedChecks"`
	Iterations         int       `json:"iterations"`
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
}

// PromoteRequest represents a promote request
type PromoteRequest struct {
	Force        bool `json:"force,omitempty"`
	SkipAnalysis bool `json:"skipAnalysis,omitempty"`
}

// OperationResponse represents the response for control operations
type OperationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// MetricsResponse represents metrics for a canary
type MetricsResponse struct {
	Name    string         `json:"name"`
	Metrics []MetricResult `json:"metrics"`
}

// MetricResult represents a single metric result
type MetricResult struct {
	Name      string                 `json:"name"`
	Timestamp time.Time              `json:"timestamp"`
	Value     float64                `json:"value"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// EventsResponse represents events for a canary
type EventsResponse struct {
	Events []EventItem `json:"events"`
	Total  int         `json:"total"`
}

// EventItem represents a single event
type EventItem struct {
	Type      string    `json:"type"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source,omitempty"`
}

// ConvertCanaryToResponse converts a Canary CR to API response format
func ConvertCanaryToResponse(canary *flaggerv1.Canary) CanaryResponse {
	response := CanaryResponse{
		Metadata: CanaryMetadata{
			Name:            canary.Name,
			Namespace:       canary.Namespace,
			CreatedAt:       canary.CreationTimestamp.Time,
			Labels:          canary.Labels,
			Annotations:     canary.Annotations,
			ResourceVersion: canary.ResourceVersion,
			UID:             string(canary.UID),
			Generation:      canary.Generation,
		},
		Spec: canary.Spec,
		Status: CanaryStatus{
			Phase:        string(canary.Status.Phase),
			CanaryWeight: canary.Status.CanaryWeight,
			FailedChecks: canary.Status.FailedChecks,
			Iterations:   canary.Status.Iterations,
		},
	}

	if !canary.Status.LastTransitionTime.IsZero() {
		response.Status.LastTransitionTime = canary.Status.LastTransitionTime.Time
	}

	return response
}
