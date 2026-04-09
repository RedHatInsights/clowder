package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
)

func TestCJIGenerationCheck(t *testing.T) {
	tests := []struct {
		name             string
		appGeneration    int64
		statusGeneration int64
		expectGenError   bool
	}{
		{
			name:             "generation matches - should pass generation check",
			appGeneration:    5,
			statusGeneration: 5,
			expectGenError:   false,
		},
		{
			name:             "generation mismatch with statusGeneration > 0 - should requeue",
			appGeneration:    6,
			statusGeneration: 5,
			expectGenError:   true,
		},
		{
			name:             "statusGeneration 0 (backward compat) - should skip check",
			appGeneration:    5,
			statusGeneration: 0,
			expectGenError:   false,
		},
		{
			name:             "large generation gap - should requeue",
			appGeneration:    100,
			statusGeneration: 95,
			expectGenError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns := "test-gen-check"

			cji := &crd.ClowdJobInvocation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cji",
					Namespace: ns,
				},
				Spec: crd.ClowdJobInvocationSpec{
					AppName:       "test-app",
					RunOnNotReady: true,
					Jobs:          []string{"test-job"},
				},
			}

			app := &crd.ClowdApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-app",
					Namespace:  ns,
					Generation: tt.appGeneration,
				},
				Spec: crd.ClowdAppSpec{
					EnvName: "test-env",
					Jobs: []crd.Job{
						{
							Name: "test-job",
							PodSpec: crd.PodSpec{
								Image: "busybox:latest",
							},
						},
					},
				},
				Status: crd.ClowdAppStatus{
					Generation: tt.statusGeneration,
				},
			}

			cachedClient := fake.NewClientBuilder().
				WithScheme(Scheme).
				WithObjects(cji).
				WithStatusSubresource(&crd.ClowdJobInvocation{}).
				Build()

			apiReader := fake.NewClientBuilder().
				WithScheme(Scheme).
				WithObjects(app).
				Build()

			recorder := record.NewFakeRecorder(100)

			reconciler := &ClowdJobInvocationReconciler{
				Client:    cachedClient,
				APIReader: apiReader,
				Log:       ctrl.Log.WithName("test"),
				Scheme:    Scheme,
				Recorder:  recorder,
			}

			_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-cji",
					Namespace: ns,
				},
			})

			assert.Error(t, err)

			if tt.expectGenError {
				assert.Contains(t, err.Error(), "has not been reconciled with the current generation",
					"expected generation mismatch error, got: %s", err.Error())
			} else {
				assert.NotContains(t, err.Error(), "has not been reconciled with the current generation",
					"did not expect generation mismatch error, got: %s", err.Error())
			}
		})
	}
}

func TestCJIUsesAPIReaderNotCache(t *testing.T) {
	ns := "test-apireader"

	cji := &crd.ClowdJobInvocation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cji",
			Namespace: ns,
		},
		Spec: crd.ClowdJobInvocationSpec{
			AppName:       "test-app",
			RunOnNotReady: true,
			Jobs:          []string{"test-job"},
		},
	}

	newApp := &crd.ClowdApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  ns,
			Generation: 10,
		},
		Spec: crd.ClowdAppSpec{
			EnvName: "test-env",
			Jobs: []crd.Job{
				{
					Name: "test-job",
					PodSpec: crd.PodSpec{
						Image: "busybox:v2-new",
					},
				},
			},
		},
		Status: crd.ClowdAppStatus{
			Generation: 9,
		},
	}

	staleApp := &crd.ClowdApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  ns,
			Generation: 9,
		},
		Spec: crd.ClowdAppSpec{
			EnvName: "test-env",
			Jobs: []crd.Job{
				{
					Name: "test-job",
					PodSpec: crd.PodSpec{
						Image: "busybox:v1-old",
					},
				},
			},
		},
		Status: crd.ClowdAppStatus{
			Generation: 9,
		},
	}

	// Cached client has the CJI and the STALE ClowdApp (simulates
	// informer cache lag). The controller must NOT use this.
	cachedClient := fake.NewClientBuilder().
		WithScheme(Scheme).
		WithObjects(cji, staleApp).
		WithStatusSubresource(&crd.ClowdJobInvocation{}).
		Build()

	// APIReader has the NEW ClowdApp (simulates direct etcd read).
	// Generation 10 != status.generation 9, so the controller should
	// detect the mismatch and requeue rather than using the stale data.
	apiReader := fake.NewClientBuilder().
		WithScheme(Scheme).
		WithObjects(newApp).
		Build()

	recorder := record.NewFakeRecorder(100)

	reconciler := &ClowdJobInvocationReconciler{
		Client:    cachedClient,
		APIReader: apiReader,
		Log:       ctrl.Log.WithName("test"),
		Scheme:    Scheme,
		Recorder:  recorder,
	}

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-cji",
			Namespace: ns,
		},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has not been reconciled with the current generation")
	assert.Contains(t, err.Error(), "spec: 10, status: 9")
}

func TestCJIAppNotFound(t *testing.T) {
	ns := "test-app-not-found"

	cji := &crd.ClowdJobInvocation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cji",
			Namespace: ns,
		},
		Spec: crd.ClowdJobInvocationSpec{
			AppName:       "missing-app",
			RunOnNotReady: true,
			Jobs:          []string{"test-job"},
		},
	}

	cachedClient := fake.NewClientBuilder().
		WithScheme(Scheme).
		WithObjects(cji).
		WithStatusSubresource(&crd.ClowdJobInvocation{}).
		Build()

	apiReader := fake.NewClientBuilder().
		WithScheme(Scheme).
		Build()

	recorder := record.NewFakeRecorder(100)

	reconciler := &ClowdJobInvocationReconciler{
		Client:    cachedClient,
		APIReader: apiReader,
		Log:       ctrl.Log.WithName("test"),
		Scheme:    Scheme,
		Recorder:  recorder,
	}

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-cji",
			Namespace: ns,
		},
	})

	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "has not been reconciled with the current generation")
}

func TestCJICompletedSkipsReconciliation(t *testing.T) {
	ns := "test-completed"

	cji := &crd.ClowdJobInvocation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cji",
			Namespace: ns,
		},
		Spec: crd.ClowdJobInvocationSpec{
			AppName: "test-app",
			Jobs:    []string{"test-job"},
		},
		Status: crd.ClowdJobInvocationStatus{
			Completed: true,
		},
	}

	cachedClient := fake.NewClientBuilder().
		WithScheme(Scheme).
		WithObjects(cji).
		WithStatusSubresource(&crd.ClowdJobInvocation{}).
		Build()

	apiReader := fake.NewClientBuilder().
		WithScheme(Scheme).
		Build()

	recorder := record.NewFakeRecorder(100)

	reconciler := &ClowdJobInvocationReconciler{
		Client:    cachedClient,
		APIReader: apiReader,
		Log:       ctrl.Log.WithName("test"),
		Scheme:    Scheme,
		Recorder:  recorder,
	}

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-cji",
			Namespace: ns,
		},
	})

	assert.NoError(t, err)
	assert.Zero(t, result.RequeueAfter)
}

func TestCJIExistingJobMapSkipsReconciliation(t *testing.T) {
	ns := "test-existing-jobmap"

	cji := &crd.ClowdJobInvocation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cji",
			Namespace: ns,
		},
		Spec: crd.ClowdJobInvocationSpec{
			AppName: "test-app",
			Jobs:    []string{"test-job"},
		},
		Status: crd.ClowdJobInvocationStatus{
			JobMap: map[string]crd.JobConditionState{
				"some-job": crd.JobInvoked,
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-job",
			Namespace: ns,
		},
	}

	cachedClient := fake.NewClientBuilder().
		WithScheme(Scheme).
		WithObjects(cji, job).
		WithStatusSubresource(&crd.ClowdJobInvocation{}).
		Build()

	apiReader := fake.NewClientBuilder().
		WithScheme(Scheme).
		Build()

	recorder := record.NewFakeRecorder(100)

	reconciler := &ClowdJobInvocationReconciler{
		Client:    cachedClient,
		APIReader: apiReader,
		Log:       ctrl.Log.WithName("test"),
		Scheme:    Scheme,
		Recorder:  recorder,
	}

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-cji",
			Namespace: ns,
		},
	})

	assert.NoError(t, err)
	assert.Zero(t, result.RequeueAfter)
}
