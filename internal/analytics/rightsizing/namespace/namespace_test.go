// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package namespace

import (
	"testing"

	"github.com/stolostron/multicluster-observability-addon/internal/analytics/rightsizing/common"
	"github.com/stretchr/testify/assert"
)

func TestGetComponentConfig(t *testing.T) {
	config := GetComponentConfig("test-namespace")

	assert.Equal(t, common.ComponentTypeNamespace, config.ComponentType)
	assert.Equal(t, ConfigMapName, config.ConfigMapName)
	assert.Equal(t, PlacementName, config.PlacementName)
	assert.Equal(t, PlacementBindingName, config.PlacementBindingName)
	assert.Equal(t, PrometheusRulePolicyName, config.PrometheusRulePolicyName)
	assert.Equal(t, common.DefaultNamespace, config.DefaultNamespace)
	assert.NotNil(t, config.GetDefaultConfigFunc)
}

func TestGetDefaultRSNamespaceConfig(t *testing.T) {
	config := GetDefaultRSNamespaceConfig()

	assert.NotNil(t, config)
	assert.Contains(t, config, "prometheusRuleConfig")
	assert.Contains(t, config, "placementConfiguration")
	assert.NotEmpty(t, config["prometheusRuleConfig"])
	assert.NotEmpty(t, config["placementConfiguration"])
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "rs-policyset-binding", PlacementBindingName)
	assert.Equal(t, "rs-placement", PlacementName)
	assert.Equal(t, "rs-prom-rules-policy", PrometheusRulePolicyName)
	assert.Equal(t, "acm-rs-namespace-prometheus-rules", PrometheusRuleName)
	assert.Equal(t, "rs-namespace-config", ConfigMapName)
}

func TestComponentState(t *testing.T) {
	assert.NotNil(t, ComponentState)
	assert.Equal(t, common.DefaultNamespace, ComponentState.Namespace)
	assert.False(t, ComponentState.Enabled)
}
