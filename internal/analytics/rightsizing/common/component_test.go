// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package common

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	policyv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func setupScheme(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, clusterv1beta1.AddToScheme(scheme))
	require.NoError(t, policyv1.AddToScheme(scheme))
	return scheme
}

func TestCleanupComponentResources(t *testing.T) {
	ctx := context.Background()
	scheme := setupScheme(t)

	configNamespace := "open-cluster-management-observability"
	bindingNamespace := "open-cluster-management-global-set"

	componentConfig := ComponentConfig{
		ComponentType:            ComponentTypeNamespace,
		ConfigMapName:            "rs-namespace-config",
		PlacementName:            "rs-placement",
		PlacementBindingName:     "rs-policyset-binding",
		PrometheusRulePolicyName: "rs-prom-rules-policy",
		DefaultNamespace:         DefaultNamespace,
	}

	tests := []struct {
		name               string
		existingResources  []client.Object
		bindingUpdated     bool
		expectedDeleted    []string
		expectedNotDeleted []string
	}{
		{
			name: "cleanup with bindingUpdated=false should delete all resources including ConfigMap",
			existingResources: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rs-namespace-config",
						Namespace: configNamespace,
					},
					Data: map[string]string{"key": "value"},
				},
				&policyv1.Policy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rs-prom-rules-policy",
						Namespace: bindingNamespace,
					},
				},
				&clusterv1beta1.Placement{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rs-placement",
						Namespace: bindingNamespace,
					},
				},
				&policyv1.PlacementBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rs-policyset-binding",
						Namespace: bindingNamespace,
					},
				},
			},
			bindingUpdated:  false,
			expectedDeleted: []string{"rs-namespace-config", "rs-prom-rules-policy", "rs-placement", "rs-policyset-binding"},
		},
		{
			name: "cleanup with bindingUpdated=true should NOT delete ConfigMap",
			existingResources: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rs-namespace-config",
						Namespace: configNamespace,
					},
					Data: map[string]string{"key": "value"},
				},
				&policyv1.Policy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rs-prom-rules-policy",
						Namespace: bindingNamespace,
					},
				},
				&clusterv1beta1.Placement{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rs-placement",
						Namespace: bindingNamespace,
					},
				},
				&policyv1.PlacementBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rs-policyset-binding",
						Namespace: bindingNamespace,
					},
				},
			},
			bindingUpdated:     true,
			expectedDeleted:    []string{"rs-prom-rules-policy", "rs-placement", "rs-policyset-binding"},
			expectedNotDeleted: []string{"rs-namespace-config"},
		},
		{
			name:               "cleanup when no resources exist should not error",
			existingResources:  []client.Object{},
			bindingUpdated:     false,
			expectedDeleted:    []string{},
			expectedNotDeleted: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.existingResources...).
				Build()

			// Call cleanup
			CleanupComponentResources(ctx, fakeClient, componentConfig, bindingNamespace, configNamespace, tt.bindingUpdated)

			// Verify deleted resources
			for _, name := range tt.expectedDeleted {
				var obj client.Object
				var namespace string

				if name == "rs-namespace-config" {
					obj = &corev1.ConfigMap{}
					namespace = configNamespace
				} else if name == "rs-placement" {
					obj = &clusterv1beta1.Placement{}
					namespace = bindingNamespace
				} else if name == "rs-policyset-binding" {
					obj = &policyv1.PlacementBinding{}
					namespace = bindingNamespace
				} else if name == "rs-prom-rules-policy" {
					obj = &policyv1.Policy{}
					namespace = bindingNamespace
				}

				err := fakeClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, obj)
				assert.Error(t, err, "Resource %s should have been deleted", name)
			}

			// Verify resources that should NOT be deleted
			for _, name := range tt.expectedNotDeleted {
				var obj client.Object
				var namespace string

				if name == "rs-namespace-config" {
					obj = &corev1.ConfigMap{}
					namespace = configNamespace
				}

				err := fakeClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, obj)
				assert.NoError(t, err, "Resource %s should NOT have been deleted", name)
			}
		})
	}
}

func TestHandleComponentRightSizing_Disable(t *testing.T) {
	ctx := context.Background()
	scheme := setupScheme(t)

	configNamespace := "open-cluster-management-observability"
	bindingNamespace := "open-cluster-management-global-set"

	// Create initial resources that should be cleaned up when feature is disabled
	existingResources := []client.Object{
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rs-namespace-config",
				Namespace: configNamespace,
			},
			Data: map[string]string{
				"prometheusRuleConfig":   "recommendationPercentage: 110",
				"placementConfiguration": "clusterSets: []",
			},
		},
		&policyv1.Policy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rs-prom-rules-policy",
				Namespace: bindingNamespace,
			},
		},
		&clusterv1beta1.Placement{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rs-placement",
				Namespace: bindingNamespace,
			},
		},
		&policyv1.PlacementBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rs-policyset-binding",
				Namespace: bindingNamespace,
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existingResources...).
		Build()

	componentConfig := ComponentConfig{
		ComponentType:            ComponentTypeNamespace,
		ConfigMapName:            "rs-namespace-config",
		PlacementName:            "rs-placement",
		PlacementBindingName:     "rs-policyset-binding",
		PrometheusRulePolicyName: "rs-prom-rules-policy",
		DefaultNamespace:         DefaultNamespace,
		GetDefaultConfigFunc: func() map[string]string {
			return map[string]string{
				"prometheusRuleConfig":   "recommendationPercentage: 110",
				"placementConfiguration": "clusterSets: []",
			}
		},
	}

	// Initial state: feature was enabled
	state := &ComponentState{
		Namespace: bindingNamespace,
		Enabled:   true,
	}

	// Options with feature DISABLED
	opts := RightSizingOptions{
		NamespaceEnabled:      false,
		NamespaceBinding:      bindingNamespace,
		VirtualizationEnabled: false,
		VirtualizationBinding: "",
		ConfigNamespace:       configNamespace,
	}

	// Call HandleComponentRightSizing with feature disabled
	err := HandleComponentRightSizing(ctx, fakeClient, opts, componentConfig, state)
	require.NoError(t, err)

	// Verify state was updated
	assert.False(t, state.Enabled, "State should be disabled")

	// Verify ConfigMap was deleted
	cm := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "rs-namespace-config", Namespace: configNamespace}, cm)
	assert.Error(t, err, "ConfigMap should have been deleted")

	// Verify other resources were deleted
	policy := &policyv1.Policy{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "rs-prom-rules-policy", Namespace: bindingNamespace}, policy)
	assert.Error(t, err, "Policy should have been deleted")

	placement := &clusterv1beta1.Placement{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "rs-placement", Namespace: bindingNamespace}, placement)
	assert.Error(t, err, "Placement should have been deleted")

	pb := &policyv1.PlacementBinding{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "rs-policyset-binding", Namespace: bindingNamespace}, pb)
	assert.Error(t, err, "PlacementBinding should have been deleted")
}

func TestHandleComponentRightSizing_FirstEnable(t *testing.T) {
	ctx := context.Background()
	scheme := setupScheme(t)

	configNamespace := "open-cluster-management-observability"
	bindingNamespace := "open-cluster-management-global-set"

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	applyChangesCalled := false
	componentConfig := ComponentConfig{
		ComponentType:            ComponentTypeNamespace,
		ConfigMapName:            "rs-namespace-config",
		PlacementName:            "rs-placement",
		PlacementBindingName:     "rs-policyset-binding",
		PrometheusRulePolicyName: "rs-prom-rules-policy",
		DefaultNamespace:         DefaultNamespace,
		GetDefaultConfigFunc: func() map[string]string {
			return map[string]string{
				"prometheusRuleConfig":   "recommendationPercentage: 110",
				"placementConfiguration": "clusterSets: []",
			}
		},
		ApplyChangesFunc: func(configData RSNamespaceConfigMapData) error {
			applyChangesCalled = true
			return nil
		},
	}

	// Initial state: feature was NOT enabled before
	state := &ComponentState{
		Namespace: DefaultNamespace,
		Enabled:   false,
	}

	// Options with feature ENABLED
	opts := RightSizingOptions{
		NamespaceEnabled:      true,
		NamespaceBinding:      bindingNamespace,
		VirtualizationEnabled: false,
		VirtualizationBinding: "",
		ConfigNamespace:       configNamespace,
	}

	// Call HandleComponentRightSizing with feature enabled (first time)
	err := HandleComponentRightSizing(ctx, fakeClient, opts, componentConfig, state)
	require.NoError(t, err)

	// Verify state was updated
	assert.True(t, state.Enabled, "State should be enabled")
	assert.Equal(t, bindingNamespace, state.Namespace, "State namespace should be updated")

	// Verify ApplyChangesFunc was called (for first enable)
	assert.True(t, applyChangesCalled, "ApplyChangesFunc should have been called on first enable")

	// Verify ConfigMap was created
	cm := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "rs-namespace-config", Namespace: configNamespace}, cm)
	assert.NoError(t, err, "ConfigMap should have been created")
}

func TestHandleComponentRightSizing_ConfigMapUpdate(t *testing.T) {
	ctx := context.Background()
	scheme := setupScheme(t)

	configNamespace := "open-cluster-management-observability"
	bindingNamespace := "open-cluster-management-global-set"

	// Create existing ConfigMap with initial data
	existingResources := []client.Object{
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rs-namespace-config",
				Namespace: configNamespace,
			},
			Data: map[string]string{
				"prometheusRuleConfig":   "recommendationPercentage: 120", // Updated value
				"placementConfiguration": "clusterSets: []",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existingResources...).
		Build()

	applyChangesCalled := false
	var capturedConfigData RSNamespaceConfigMapData
	componentConfig := ComponentConfig{
		ComponentType:            ComponentTypeNamespace,
		ConfigMapName:            "rs-namespace-config",
		PlacementName:            "rs-placement",
		PlacementBindingName:     "rs-policyset-binding",
		PrometheusRulePolicyName: "rs-prom-rules-policy",
		DefaultNamespace:         DefaultNamespace,
		GetDefaultConfigFunc: func() map[string]string {
			return map[string]string{
				"prometheusRuleConfig":   "recommendationPercentage: 110",
				"placementConfiguration": "clusterSets: []",
			}
		},
		ApplyChangesFunc: func(configData RSNamespaceConfigMapData) error {
			applyChangesCalled = true
			capturedConfigData = configData
			return nil
		},
	}

	// State: feature was ALREADY enabled (simulating reconciliation after ConfigMap update)
	state := &ComponentState{
		Namespace: bindingNamespace,
		Enabled:   true, // Already enabled
	}

	// Options with feature ENABLED (no change from before)
	opts := RightSizingOptions{
		NamespaceEnabled:      true,
		NamespaceBinding:      bindingNamespace,
		VirtualizationEnabled: false,
		VirtualizationBinding: "",
		ConfigNamespace:       configNamespace,
	}

	// Call HandleComponentRightSizing - simulating a reconciliation after ConfigMap was updated
	err := HandleComponentRightSizing(ctx, fakeClient, opts, componentConfig, state)
	require.NoError(t, err)

	// Verify ApplyChangesFunc was called even though it's not first enable or binding change
	assert.True(t, applyChangesCalled, "ApplyChangesFunc should be called on every reconciliation when enabled")

	// Verify the updated ConfigMap data was passed to ApplyChangesFunc
	assert.Equal(t, 120, capturedConfigData.PrometheusRuleConfig.RecommendationPercentage,
		"ApplyChangesFunc should receive the updated recommendationPercentage from ConfigMap")
}
