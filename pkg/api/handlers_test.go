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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	flaggerv1 "github.com/fluxcd/flagger/pkg/apis/flagger/v1beta1"
	fakeFlagger "github.com/fluxcd/flagger/pkg/client/clientset/versioned/fake"
)

func TestListAllCanaries(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	kubeClient := fake.NewSimpleClientset()
	
	canary1 := &flaggerv1.Canary{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-canary-1",
			Namespace: "default",
		},
		Spec: flaggerv1.CanarySpec{
			TargetRef: flaggerv1.LocalObjectReference{
				Name: "test-app",
			},
		},
		Status: flaggerv1.CanaryStatus{
			Phase:        flaggerv1.CanaryPhaseInitialized,
			CanaryWeight: 0,
			Iterations:   0,
			FailedChecks: 0,
		},
	}
	
	canary2 := &flaggerv1.Canary{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-canary-2",
			Namespace: "production",
		},
		Spec: flaggerv1.CanarySpec{
			TargetRef: flaggerv1.LocalObjectReference{
				Name: "prod-app",
			},
		},
		Status: flaggerv1.CanaryStatus{
			Phase:        flaggerv1.CanaryPhaseProgressing,
			CanaryWeight: 20,
			Iterations:   2,
			FailedChecks: 0,
		},
	}
	
	flaggerClient := fakeFlagger.NewSimpleClientset(canary1, canary2)
	handler := NewHandler(kubeClient, flaggerClient, logger.Sugar())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/canaries", nil)
	w := httptest.NewRecorder()

	handler.ListAllCanaries(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response CanaryListResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, 2, response.Total)
	assert.Len(t, response.Items, 2)
}

func TestListCanariesInNamespace(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	kubeClient := fake.NewSimpleClientset()
	
	canary1 := &flaggerv1.Canary{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-canary-1",
			Namespace: "default",
		},
		Spec: flaggerv1.CanarySpec{
			TargetRef: flaggerv1.LocalObjectReference{
				Name: "test-app",
			},
		},
		Status: flaggerv1.CanaryStatus{
			Phase: flaggerv1.CanaryPhaseInitialized,
		},
	}
	
	canary2 := &flaggerv1.Canary{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-canary-2",
			Namespace: "production",
		},
		Spec: flaggerv1.CanarySpec{
			TargetRef: flaggerv1.LocalObjectReference{
				Name: "prod-app",
			},
		},
		Status: flaggerv1.CanaryStatus{
			Phase: flaggerv1.CanaryPhaseProgressing,
		},
	}
	
	flaggerClient := fakeFlagger.NewSimpleClientset(canary1, canary2)
	handler := NewHandler(kubeClient, flaggerClient, logger.Sugar())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/namespaces/default/canaries", nil)
	w := httptest.NewRecorder()

	handler.ListCanariesInNamespace(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response CanaryListResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, 1, response.Total)
	assert.Len(t, response.Items, 1)
	assert.Equal(t, "test-canary-1", response.Items[0].Metadata.Name)
}

func TestGetCanary(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	kubeClient := fake.NewSimpleClientset()
	
	canary := &flaggerv1.Canary{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-canary",
			Namespace: "default",
		},
		Spec: flaggerv1.CanarySpec{
			TargetRef: flaggerv1.LocalObjectReference{
				Name: "test-app",
			},
		},
		Status: flaggerv1.CanaryStatus{
			Phase:        flaggerv1.CanaryPhaseProgressing,
			CanaryWeight: 30,
			Iterations:   3,
			FailedChecks: 0,
		},
	}
	
	flaggerClient := fakeFlagger.NewSimpleClientset(canary)
	handler := NewHandler(kubeClient, flaggerClient, logger.Sugar())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/namespaces/default/canaries/test-canary", nil)
	w := httptest.NewRecorder()

	handler.GetCanary(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response CanaryResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "test-canary", response.Metadata.Name)
	assert.Equal(t, "default", response.Metadata.Namespace)
	assert.Equal(t, "Progressing", response.Status.Phase)
	assert.Equal(t, 30, response.Status.CanaryWeight)
}

func TestGetCanary_NotFound(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	kubeClient := fake.NewSimpleClientset()
	flaggerClient := fakeFlagger.NewSimpleClientset()
	handler := NewHandler(kubeClient, flaggerClient, logger.Sugar())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/namespaces/default/canaries/nonexistent", nil)
	w := httptest.NewRecorder()

	handler.GetCanary(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Contains(t, response.Message, "not found")
}

func TestPromoteCanary(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	kubeClient := fake.NewSimpleClientset()
	
	canary := &flaggerv1.Canary{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-canary",
			Namespace: "default",
		},
		Spec: flaggerv1.CanarySpec{
			TargetRef: flaggerv1.LocalObjectReference{
				Name: "test-app",
			},
			SkipAnalysis: false,
		},
		Status: flaggerv1.CanaryStatus{
			Phase: flaggerv1.CanaryPhaseProgressing,
		},
	}
	
	flaggerClient := fakeFlagger.NewSimpleClientset(canary)
	handler := NewHandler(kubeClient, flaggerClient, logger.Sugar())

	reqBody := PromoteRequest{
		SkipAnalysis: true,
	}
	body, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest(http.MethodPost, "/api/v1/namespaces/default/canaries/test-canary/promote", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.PromoteCanary(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response OperationResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response.Success)
	assert.Contains(t, response.Message, "promotion")
}

func TestPauseCanary(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	kubeClient := fake.NewSimpleClientset()
	
	canary := &flaggerv1.Canary{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-canary",
			Namespace: "default",
		},
		Spec: flaggerv1.CanarySpec{
			TargetRef: flaggerv1.LocalObjectReference{
				Name: "test-app",
			},
			Suspend: false,
		},
		Status: flaggerv1.CanaryStatus{
			Phase: flaggerv1.CanaryPhaseProgressing,
		},
	}
	
	flaggerClient := fakeFlagger.NewSimpleClientset(canary)
	handler := NewHandler(kubeClient, flaggerClient, logger.Sugar())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/namespaces/default/canaries/test-canary/pause", nil)
	w := httptest.NewRecorder()

	handler.PauseCanary(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response OperationResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response.Success)
	assert.Contains(t, response.Message, "paused")
}

func TestResumeCanary(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	kubeClient := fake.NewSimpleClientset()
	
	canary := &flaggerv1.Canary{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-canary",
			Namespace: "default",
		},
		Spec: flaggerv1.CanarySpec{
			TargetRef: flaggerv1.LocalObjectReference{
				Name: "test-app",
			},
			Suspend: true,
		},
		Status: flaggerv1.CanaryStatus{
			Phase: flaggerv1.CanaryPhaseWaiting,
		},
	}
	
	flaggerClient := fakeFlagger.NewSimpleClientset(canary)
	handler := NewHandler(kubeClient, flaggerClient, logger.Sugar())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/namespaces/default/canaries/test-canary/resume", nil)
	w := httptest.NewRecorder()

	handler.ResumeCanary(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response OperationResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response.Success)
	assert.Contains(t, response.Message, "resumed")
}

func TestRollbackCanary(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	kubeClient := fake.NewSimpleClientset()
	
	canary := &flaggerv1.Canary{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-canary",
			Namespace: "default",
		},
		Spec: flaggerv1.CanarySpec{
			TargetRef: flaggerv1.LocalObjectReference{
				Name: "test-app",
			},
		},
		Status: flaggerv1.CanaryStatus{
			Phase: flaggerv1.CanaryPhaseProgressing,
		},
	}
	
	flaggerClient := fakeFlagger.NewSimpleClientset(canary)
	handler := NewHandler(kubeClient, flaggerClient, logger.Sugar())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/namespaces/default/canaries/test-canary/rollback", nil)
	w := httptest.NewRecorder()

	handler.RollbackCanary(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response OperationResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response.Success)
	assert.Contains(t, response.Message, "rollback")
}
