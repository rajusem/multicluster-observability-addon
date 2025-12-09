// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstants(t *testing.T) {
	assert.Equal(t, 110, DefaultRecommendationPercentage)
	assert.Equal(t, "openshift-monitoring", MonitoringNamespace)
	assert.Equal(t, "open-cluster-management-global-set", DefaultNamespace)
}

func TestComponentType(t *testing.T) {
	assert.Equal(t, ComponentType("namespace"), ComponentTypeNamespace)
	assert.Equal(t, ComponentType("virtualization"), ComponentTypeVirtualization)
}

func TestRSLabelFilter(t *testing.T) {
	filter := RSLabelFilter{
		LabelName:         "label_env",
		InclusionCriteria: []string{"prod", "staging"},
		ExclusionCriteria: []string{},
	}

	assert.Equal(t, "label_env", filter.LabelName)
	assert.Equal(t, []string{"prod", "staging"}, filter.InclusionCriteria)
	assert.Empty(t, filter.ExclusionCriteria)
}

func TestRSPrometheusRuleConfig(t *testing.T) {
	config := RSPrometheusRuleConfig{
		RecommendationPercentage: 120,
	}
	config.NamespaceFilterCriteria.ExclusionCriteria = []string{"openshift.*"}
	config.LabelFilterCriteria = []RSLabelFilter{
		{
			LabelName:         "label_env",
			InclusionCriteria: []string{"prod"},
		},
	}

	assert.Equal(t, 120, config.RecommendationPercentage)
	assert.Equal(t, []string{"openshift.*"}, config.NamespaceFilterCriteria.ExclusionCriteria)
	assert.Empty(t, config.NamespaceFilterCriteria.InclusionCriteria)
	assert.Len(t, config.LabelFilterCriteria, 1)
}

func TestComponentConfig(t *testing.T) {
	getDefaultConfig := func() map[string]string {
		return map[string]string{"key": "value"}
	}

	config := ComponentConfig{
		ComponentType:            ComponentTypeNamespace,
		ConfigMapName:            "test-configmap",
		PlacementName:            "test-placement",
		PlacementBindingName:     "test-binding",
		PrometheusRulePolicyName: "test-policy",
		DefaultNamespace:         DefaultNamespace,
		GetDefaultConfigFunc:     getDefaultConfig,
	}

	assert.Equal(t, ComponentTypeNamespace, config.ComponentType)
	assert.Equal(t, "test-configmap", config.ConfigMapName)
	assert.Equal(t, "test-placement", config.PlacementName)
	assert.Equal(t, "test-binding", config.PlacementBindingName)
	assert.Equal(t, "test-policy", config.PrometheusRulePolicyName)
	assert.Equal(t, DefaultNamespace, config.DefaultNamespace)
	assert.NotNil(t, config.GetDefaultConfigFunc)

	result := config.GetDefaultConfigFunc()
	assert.Equal(t, "value", result["key"])
}

func TestComponentState(t *testing.T) {
	state := ComponentState{
		Namespace: "test-namespace",
		Enabled:   true,
	}

	assert.Equal(t, "test-namespace", state.Namespace)
	assert.True(t, state.Enabled)
}

func TestRightSizingOptions(t *testing.T) {
	opts := RightSizingOptions{
		NamespaceEnabled:      true,
		NamespaceBinding:      "ns-binding",
		VirtualizationEnabled: true,
		VirtualizationBinding: "virt-binding",
		ConfigNamespace:       "config-ns",
	}

	assert.True(t, opts.NamespaceEnabled)
	assert.Equal(t, "ns-binding", opts.NamespaceBinding)
	assert.True(t, opts.VirtualizationEnabled)
	assert.Equal(t, "virt-binding", opts.VirtualizationBinding)
	assert.Equal(t, "config-ns", opts.ConfigNamespace)
}

func TestRSNamespaceConfigMapData(t *testing.T) {
	data := RSNamespaceConfigMapData{
		PrometheusRuleConfig: RSPrometheusRuleConfig{
			RecommendationPercentage: 110,
		},
	}

	assert.Equal(t, 110, data.PrometheusRuleConfig.RecommendationPercentage)
}
