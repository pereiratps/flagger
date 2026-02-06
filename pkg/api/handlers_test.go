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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	flaggerv1 "github.com/fluxcd/flagger/pkg/apis/flagger/v1beta1"
	clientset "github.com/fluxcd/flagger/pkg/client/clientset/versioned"
	fakeflagger "github.com/fluxcd/flagger/pkg/client/clientset/versioned/fake"
	"github.com/fluxcd/flagger/pkg/logger"
)

func newTestHandler() (*Handler, clientset.Interface, error) {
	log, err := logger.NewLogger("debug")
	if err != nil {
		return nil, nil, err
	}

	kubeClient := fake.NewSimpleClientset()
	flaggerClient := fakeflagger.NewSimpleClientset()

	handler := NewHandler(kubeClient, flaggerClient, log)
	return handler, flaggerClient, nil
}

func TestListAllCanaries(t *testing.T) {
	handler, flaggerClient, err := newTestHandler()
	require.NoError(t, err)

	// Create test canaries
	canary1 := &flaggerv1.Canary{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-canary-1",
			Namespace: "default",
		},
		Spec: flaggerv1.CanarySpec{},
		Status: flaggerv1.CanaryStatus{
			Phase: flaggerv1.CanaryPhaseInitialized,
		},
	}
	canary2 := &flaggerv1.Canary{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-canary-2",
			Namespace: "production",
		},
		Spec: flaggerv1.CanarySpec{},
		Status: flaggerv1.CanaryStatus{
			Phase: flaggerv1.CanaryPhaseProgressing,
		},
	}

	_, err = flaggerClient.FlaggerV1beta1().Canaries("default").Create(context.Background(), canary1, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = flaggerClient.FlaggerV1beta1().Canaries("production").Create(context.Background(), canary2, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create request
	req, err := http.NewRequest("GET", "/api/v1/canaries", nil)
	require.NoError(t, err)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler.ListAllCanaries(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)

	var response CanaryListResponse
	err = json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, 2, response.Total)
	assert.Len(t, response.Items, 2)
}

func TestListCanariesInNamespace(t *testing.T) {
	handler, flaggerClient, err := newTestHandler()
	require.NoError(t, err)

	// Create test canaries
	canary1 := &flaggerv1.Canary{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-canary-1",
			Namespace: "default",
		},
		Spec: flaggerv1.CanarySpec{},
		Status: flaggerv1.CanaryStatus{
			Phase: flaggerv1.CanaryPhaseInitialized,
		},
	}
	canary2 := &flaggerv1.Canary{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-canary-2",
			Namespace: "production",
		},
		Spec: flaggerv1.CanarySpec{},
		Status: flaggerv1.CanaryStatus{
			Phase: flaggerv1.CanaryPhaseProgressing,
		},
	}

	_, err = flaggerClient.FlaggerV1beta1().Canaries("default").Create(context.Background(), canary1, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = flaggerClient.FlaggerV1beta1().Canaries("production").Create(context.Background(), canary2, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create request with namespace
	req, err := http.NewRequest("GET", "/api/v1/namespaces/default/canaries", nil)
	require.NoError(t, err)
	req = mux.SetURLVars(req, map[string]string{"namespace": "default"})

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler.ListCanariesInNamespace(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)

	var response CanaryListResponse
	err = json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, 1, response.Total)
	assert.Len(t, response.Items, 1)
	assert.Equal(t, "test-canary-1", response.Items[0].Metadata.Name)
}

func TestGetCanary(t *testing.T) {
	handler, flaggerClient, err := newTestHandler()
	require.NoError(t, err)

	// Create test canary
	canary := &flaggerv1.Canary{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-canary",
			Namespace: "default",
		},
		Spec: flaggerv1.CanarySpec{},
		Status: flaggerv1.CanaryStatus{
			Phase:        flaggerv1.CanaryPhaseProgressing,
			CanaryWeight: 50,
			Iterations:   5,
		},
	}

	_, err = flaggerClient.FlaggerV1beta1().Canaries("default").Create(context.Background(), canary, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create request
	req, err := http.NewRequest("GET", "/api/v1/namespaces/default/canaries/test-canary", nil)
	require.NoError(t, err)
	req = mux.SetURLVars(req, map[string]string{
		"namespace": "default",
		"name":      "test-canary",
	})

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler.GetCanary(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)

	var response CanaryResponse
	err = json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "test-canary", response.Metadata.Name)
	assert.Equal(t, "default", response.Metadata.Namespace)
	assert.Equal(t, "Progressing", response.Status.Phase)
	assert.Equal(t, 50, response.Status.CanaryWeight)
}

func TestGetCanaryNotFound(t *testing.T) {
	handler, _, err := newTestHandler()
	require.NoError(t, err)

	// Create request for non-existent canary
	req, err := http.NewRequest("GET", "/api/v1/namespaces/default/canaries/not-found", nil)
	require.NoError(t, err)
	req = mux.SetURLVars(req, map[string]string{
		"namespace": "default",
		"name":      "not-found",
	})

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler.GetCanary(rr, req)

	// Check response
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestPauseCanary(t *testing.T) {
	handler, flaggerClient, err := newTestHandler()
	require.NoError(t, err)

	// Create test canary
	canary := &flaggerv1.Canary{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-canary",
			Namespace: "default",
		},
		Spec: flaggerv1.CanarySpec{
			Suspend: false,
		},
		Status: flaggerv1.CanaryStatus{
			Phase: flaggerv1.CanaryPhaseProgressing,
		},
	}

	_, err = flaggerClient.FlaggerV1beta1().Canaries("default").Create(context.Background(), canary, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create request
	req, err := http.NewRequest("POST", "/api/v1/namespaces/default/canaries/test-canary/pause", nil)
	require.NoError(t, err)
	req = mux.SetURLVars(req, map[string]string{
		"namespace": "default",
		"name":      "test-canary",
	})

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler.PauseCanary(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify canary was paused
	updatedCanary, err := flaggerClient.FlaggerV1beta1().Canaries("default").Get(context.Background(), "test-canary", metav1.GetOptions{})
	require.NoError(t, err)
	assert.True(t, updatedCanary.Spec.Suspend)
}

func TestResumeCanary(t *testing.T) {
	handler, flaggerClient, err := newTestHandler()
	require.NoError(t, err)

	// Create test canary (paused)
	canary := &flaggerv1.Canary{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-canary",
			Namespace: "default",
		},
		Spec: flaggerv1.CanarySpec{
			Suspend: true,
		},
		Status: flaggerv1.CanaryStatus{
			Phase: flaggerv1.CanaryPhaseWaiting,
		},
	}

	_, err = flaggerClient.FlaggerV1beta1().Canaries("default").Create(context.Background(), canary, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create request
	req, err := http.NewRequest("POST", "/api/v1/namespaces/default/canaries/test-canary/resume", nil)
	require.NoError(t, err)
	req = mux.SetURLVars(req, map[string]string{
		"namespace": "default",
		"name":      "test-canary",
	})

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler.ResumeCanary(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify canary was resumed
	updatedCanary, err := flaggerClient.FlaggerV1beta1().Canaries("default").Get(context.Background(), "test-canary", metav1.GetOptions{})
	require.NoError(t, err)
	assert.False(t, updatedCanary.Spec.Suspend)
}

func TestPromoteCanary(t *testing.T) {
	handler, flaggerClient, err := newTestHandler()
	require.NoError(t, err)

	// Create test canary
	canary := &flaggerv1.Canary{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-canary",
			Namespace: "default",
		},
		Spec: flaggerv1.CanarySpec{},
		Status: flaggerv1.CanaryStatus{
			Phase: flaggerv1.CanaryPhaseProgressing,
		},
	}

	_, err = flaggerClient.FlaggerV1beta1().Canaries("default").Create(context.Background(), canary, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create request with body
	reqBody := PromoteRequest{
		SkipAnalysis: true,
	}
	bodyBytes, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/namespaces/default/canaries/test-canary/promote", bytes.NewReader(bodyBytes))
	require.NoError(t, err)
	req = mux.SetURLVars(req, map[string]string{
		"namespace": "default",
		"name":      "test-canary",
	})

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler.PromoteCanary(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHealthz(t *testing.T) {
	handler, _, err := newTestHandler()
	require.NoError(t, err)

	req, err := http.NewRequest("GET", "/healthz", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.Healthz(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "OK", rr.Body.String())
}

func TestReadyz(t *testing.T) {
	handler, _, err := newTestHandler()
	require.NoError(t, err)

	req, err := http.NewRequest("GET", "/readyz", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.Readyz(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "OK", rr.Body.String())
}
