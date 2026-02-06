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
	corev1 "k8s.io/api/core/v1"
)

// CanaryResponse represents a canary deployment in the API response
type CanaryResponse struct {
	Metadata CanaryMetadata `json:"metadata"`
	Spec     interface{}    `json:"spec"`
	Status   CanaryStatus   `json:"status"`
}

// CanaryMetadata contains canary metadata
type CanaryMetadata struct {
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	CreatedAt time.Time `json:"createdAt"`
}

// CanaryStatus contains canary status information
type CanaryStatus struct {
	Phase              string    `json:"phase"`
	CanaryWeight       int       `json:"canaryWeight"`
	FailedChecks       int       `json:"failedChecks"`
	Iterations         int       `json:"iterations"`
	LastTransitionTime time.Time `json:"lastTransitionTime"`
}

// CanaryListResponse represents a list of canaries
type CanaryListResponse struct {
	Items []CanaryResponse `json:"items"`
	Total int              `json:"total"`
}

// PromoteRequest represents a promote request
type PromoteRequest struct {
	Force        bool `json:"force,omitempty"`
	SkipAnalysis bool `json:"skipAnalysis,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

// MetricsResponse represents metrics for a canary
type MetricsResponse struct {
	Canary  string                 `json:"canary"`
	Metrics map[string]interface{} `json:"metrics"`
}

// EventResponse represents a Kubernetes event
type EventResponse struct {
	Type      string    `json:"type"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// EventListResponse represents a list of events
type EventListResponse struct {
	Items []EventResponse `json:"items"`
	Total int             `json:"total"`
}

// ToCanaryResponse converts a Flagger Canary to API response
func ToCanaryResponse(canary *flaggerv1.Canary) CanaryResponse {
	return CanaryResponse{
		Metadata: CanaryMetadata{
			Name:      canary.Name,
			Namespace: canary.Namespace,
			CreatedAt: canary.CreationTimestamp.Time,
		},
		Spec: canary.Spec,
		Status: CanaryStatus{
			Phase:              string(canary.Status.Phase),
			CanaryWeight:       canary.Status.CanaryWeight,
			FailedChecks:       canary.Status.FailedChecks,
			Iterations:         canary.Status.Iterations,
			LastTransitionTime: canary.Status.LastTransitionTime.Time,
		},
	}
}

// ToEventResponse converts a Kubernetes event to API response
func ToEventResponse(event *corev1.Event) EventResponse {
	return EventResponse{
		Type:      event.Type,
		Reason:    event.Reason,
		Message:   event.Message,
		Timestamp: event.LastTimestamp.Time,
	}
}
