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
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func setupScheme(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, clusterv1beta1.AddToScheme(scheme))
	require.NoError(t, addonv1alpha1.AddToScheme(scheme))
	return scheme
}

func TestCleanupComponentResources(t *testing.T) {
	ctx := context.Background()
	scheme := setupScheme(t)

	configNamespace := "open-cluster-management-observability"
	bindingNamespace := "open-cluster-management-global-set"

	componentConfig := ComponentConfig{
		ComponentType:    ComponentTypeNamespace,
		ConfigMapName:    "rs-namespace-config",
		PlacementName:    "rs-namespace-placement",
		DefaultNamespace: DefaultNamespace,
		AddonName:        "observability-rightsizing-namespace",
		TemplateName:     "rs-namespace-template",
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
				&addonv1alpha1.ClusterManagementAddOn{
					ObjectMeta: metav1.ObjectMeta{
						Name: "observability-rightsizing-namespace",
					},
				},
				&addonv1alpha1.AddOnTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name: "rs-namespace-template",
					},
				},
				&clusterv1beta1.Placement{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rs-namespace-placement",
						Namespace: bindingNamespace,
					},
				},
			},
			bindingUpdated:  false,
			expectedDeleted: []string{"rs-namespace-config", "observability-rightsizing-namespace", "rs-namespace-template", "rs-namespace-placement"},
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
				&addonv1alpha1.ClusterManagementAddOn{
					ObjectMeta: metav1.ObjectMeta{
						Name: "observability-rightsizing-namespace",
					},
				},
				&addonv1alpha1.AddOnTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name: "rs-namespace-template",
					},
				},
				&clusterv1beta1.Placement{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rs-namespace-placement",
						Namespace: bindingNamespace,
					},
				},
			},
			bindingUpdated:     true,
			expectedDeleted:    []string{"observability-rightsizing-namespace", "rs-namespace-template", "rs-namespace-placement"},
			expectedNotDeleted: []string{"rs-namespace-config"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tc.existingResources...).
				Build()

			CleanupComponentResources(ctx, fakeClient, componentConfig, bindingNamespace, configNamespace, tc.bindingUpdated)

			// Verify expected deletions
			for _, name := range tc.expectedDeleted {
				var obj client.Object
				var key types.NamespacedName

				switch name {
				case "rs-namespace-config":
					obj = &corev1.ConfigMap{}
					key = types.NamespacedName{Name: name, Namespace: configNamespace}
				case "observability-rightsizing-namespace":
					obj = &addonv1alpha1.ClusterManagementAddOn{}
					key = types.NamespacedName{Name: name}
				case "rs-namespace-template":
					obj = &addonv1alpha1.AddOnTemplate{}
					key = types.NamespacedName{Name: name}
				case "rs-namespace-placement":
					obj = &clusterv1beta1.Placement{}
					key = types.NamespacedName{Name: name, Namespace: bindingNamespace}
				}

				err := fakeClient.Get(ctx, key, obj)
				assert.Error(t, err, "Resource %s should have been deleted", name)
			}

			// Verify resources that should NOT be deleted
			for _, name := range tc.expectedNotDeleted {
				var obj client.Object
				var key types.NamespacedName

				switch name {
				case "rs-namespace-config":
					obj = &corev1.ConfigMap{}
					key = types.NamespacedName{Name: name, Namespace: configNamespace}
				}

				err := fakeClient.Get(ctx, key, obj)
				assert.NoError(t, err, "Resource %s should NOT have been deleted", name)
			}
		})
	}
}

func TestHandleComponentRightSizing_DisabledCleansUp(t *testing.T) {
	ctx := context.Background()
	scheme := setupScheme(t)

	configNamespace := "open-cluster-management-observability"
	bindingNamespace := "open-cluster-management-global-set"

	// Create resources that should be cleaned up
	existingResources := []client.Object{
		&addonv1alpha1.ClusterManagementAddOn{
			ObjectMeta: metav1.ObjectMeta{
				Name: "observability-rightsizing-namespace",
			},
		},
		&addonv1alpha1.AddOnTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rs-namespace-template",
			},
		},
		&clusterv1beta1.Placement{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rs-namespace-placement",
				Namespace: bindingNamespace,
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rs-namespace-config",
				Namespace: configNamespace,
			},
			Data: map[string]string{"key": "value"},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existingResources...).
		Build()

	componentConfig := ComponentConfig{
		ComponentType:        ComponentTypeNamespace,
		ConfigMapName:        "rs-namespace-config",
		PlacementName:        "rs-namespace-placement",
		DefaultNamespace:     DefaultNamespace,
		AddonName:            "observability-rightsizing-namespace",
		TemplateName:         "rs-namespace-template",
		GetDefaultConfigFunc: func() map[string]string { return map[string]string{} },
		ApplyChangesFunc:     func(data RSNamespaceConfigMapData) error { return nil },
	}

	state := &ComponentState{
		Namespace: bindingNamespace,
		Enabled:   true,
	}

	opts := RightSizingOptions{
		NamespaceEnabled: false, // Disabled
		ConfigNamespace:  configNamespace,
	}

	err := HandleComponentRightSizing(ctx, fakeClient, opts, componentConfig, state)
	require.NoError(t, err)

	// Verify addon resources are cleaned up
	cmao := &addonv1alpha1.ClusterManagementAddOn{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "observability-rightsizing-namespace"}, cmao)
	assert.Error(t, err, "ClusterManagementAddOn should have been deleted")

	// Verify ConfigMap is cleaned up
	cm := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "rs-namespace-config", Namespace: configNamespace}, cm)
	assert.Error(t, err, "ConfigMap should have been deleted")

	// Verify state is updated
	assert.False(t, state.Enabled)
}

func TestComponentConfigFields(t *testing.T) {
	config := ComponentConfig{
		ComponentType:    ComponentTypeNamespace,
		ConfigMapName:    "test-config",
		PlacementName:    "test-placement",
		DefaultNamespace: "test-namespace",
		AddonName:        "test-addon",
		TemplateName:     "test-template",
	}

	assert.Equal(t, ComponentTypeNamespace, config.ComponentType)
	assert.Equal(t, "test-config", config.ConfigMapName)
	assert.Equal(t, "test-placement", config.PlacementName)
	assert.Equal(t, "test-namespace", config.DefaultNamespace)
	assert.Equal(t, "test-addon", config.AddonName)
	assert.Equal(t, "test-template", config.TemplateName)
}
