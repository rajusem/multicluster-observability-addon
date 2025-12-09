// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package virtualization

import (
	"testing"

	"github.com/stolostron/multicluster-observability-addon/internal/analytics/rightsizing/common"
	"github.com/stretchr/testify/assert"
)

func TestConstants(t *testing.T) {
	assert.Equal(t, "rs-virt-policyset-binding", PlacementBindingName)
	assert.Equal(t, "rs-virt-placement", PlacementName)
	assert.Equal(t, "rs-virt-prom-rules-policy", PrometheusRulePolicyName)
	assert.Equal(t, "acm-rs-virt-prometheus-rules", PrometheusRuleName)
	assert.Equal(t, "rs-virt-config", ConfigMapName)
}

func TestComponentState(t *testing.T) {
	assert.NotNil(t, ComponentState)
	assert.Equal(t, common.DefaultNamespace, ComponentState.Namespace)
	assert.False(t, ComponentState.Enabled)
}

func TestGetDefaultRSVirtualizationConfig(t *testing.T) {
	config := GetDefaultRSVirtualizationConfig()

	assert.NotNil(t, config)
	assert.Contains(t, config, "prometheusRuleConfig")
	assert.Contains(t, config, "placementConfiguration")
	assert.NotEmpty(t, config["prometheusRuleConfig"])
	assert.NotEmpty(t, config["placementConfiguration"])

	// Verify the config contains expected values
	promConfig := config["prometheusRuleConfig"]
	assert.Contains(t, promConfig, "recommendationPercentage")
	assert.Contains(t, promConfig, "namespaceFilterCriteria")
}
