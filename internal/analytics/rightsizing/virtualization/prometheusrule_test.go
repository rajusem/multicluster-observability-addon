// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package virtualization

import (
	"testing"

	"github.com/stolostron/multicluster-observability-addon/internal/analytics/rightsizing/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePrometheusRule(t *testing.T) {
	tests := []struct {
		name        string
		configData  common.RSNamespaceConfigMapData
		expectError bool
	}{
		{
			name: "basic configuration",
			configData: common.RSNamespaceConfigMapData{
				PrometheusRuleConfig: common.RSPrometheusRuleConfig{
					NamespaceFilterCriteria: struct {
						InclusionCriteria []string `yaml:"inclusionCriteria"`
						ExclusionCriteria []string `yaml:"exclusionCriteria"`
					}{
						ExclusionCriteria: []string{"openshift.*"},
					},
					RecommendationPercentage: 110,
				},
			},
			expectError: false,
		},
		{
			name: "with inclusion criteria",
			configData: common.RSNamespaceConfigMapData{
				PrometheusRuleConfig: common.RSPrometheusRuleConfig{
					NamespaceFilterCriteria: struct {
						InclusionCriteria []string `yaml:"inclusionCriteria"`
						ExclusionCriteria []string `yaml:"exclusionCriteria"`
					}{
						InclusionCriteria: []string{"virt-.*", "vm-.*"},
					},
					RecommendationPercentage: 120,
				},
			},
			expectError: false,
		},
		{
			name: "with label filters",
			configData: common.RSNamespaceConfigMapData{
				PrometheusRuleConfig: common.RSPrometheusRuleConfig{
					NamespaceFilterCriteria: struct {
						InclusionCriteria []string `yaml:"inclusionCriteria"`
						ExclusionCriteria []string `yaml:"exclusionCriteria"`
					}{
						ExclusionCriteria: []string{"openshift.*"},
					},
					LabelFilterCriteria: []common.RSLabelFilter{
						{
							LabelName:         "label_env",
							InclusionCriteria: []string{"prod"},
						},
					},
					RecommendationPercentage: 110,
				},
			},
			expectError: false,
		},
		{
			name: "invalid - both inclusion and exclusion for namespace",
			configData: common.RSNamespaceConfigMapData{
				PrometheusRuleConfig: common.RSPrometheusRuleConfig{
					NamespaceFilterCriteria: struct {
						InclusionCriteria []string `yaml:"inclusionCriteria"`
						ExclusionCriteria []string `yaml:"exclusionCriteria"`
					}{
						InclusionCriteria: []string{"virt-.*"},
						ExclusionCriteria: []string{"openshift.*"},
					},
					RecommendationPercentage: 110,
				},
			},
			expectError: true,
		},
		{
			name: "custom recommendation percentage",
			configData: common.RSNamespaceConfigMapData{
				PrometheusRuleConfig: common.RSPrometheusRuleConfig{
					RecommendationPercentage: 150,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := GeneratePrometheusRule(tt.configData)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, PrometheusRuleName, rule.Name)
			assert.Equal(t, common.MonitoringNamespace, rule.Namespace)
			assert.Equal(t, "PrometheusRule", rule.Kind)
			assert.Equal(t, "monitoring.coreos.com/v1", rule.APIVersion)

			// Verify rule groups exist
			assert.Len(t, rule.Spec.Groups, 4)

			// Verify group names for virtualization
			groupNames := make([]string, len(rule.Spec.Groups))
			for i, g := range rule.Spec.Groups {
				groupNames[i] = g.Name
			}
			assert.Contains(t, groupNames, "acm-vm-right-sizing-namespace-5m.rule")
			assert.Contains(t, groupNames, "acm-vm-right-sizing-namespace-1d.rules")
			assert.Contains(t, groupNames, "acm-vm-right-sizing-cluster-5m.rule")
			assert.Contains(t, groupNames, "acm-vm-right-sizing-cluster-1d.rule")
		})
	}
}

func TestGeneratePrometheusRuleVMRuleGroups(t *testing.T) {
	configData := common.RSNamespaceConfigMapData{
		PrometheusRuleConfig: common.RSPrometheusRuleConfig{
			NamespaceFilterCriteria: struct {
				InclusionCriteria []string `yaml:"inclusionCriteria"`
				ExclusionCriteria []string `yaml:"exclusionCriteria"`
			}{
				ExclusionCriteria: []string{"openshift.*"},
			},
			RecommendationPercentage: 110,
		},
	}

	rule, err := GeneratePrometheusRule(configData)
	require.NoError(t, err)

	// Test VM namespace 5m rules
	vmNamespace5mGroup := rule.Spec.Groups[0]
	assert.Equal(t, "acm-vm-right-sizing-namespace-5m.rule", vmNamespace5mGroup.Name)
	assert.Len(t, vmNamespace5mGroup.Rules, 4)

	// Verify rule record names in VM namespace 5m group
	vmNamespace5mRecords := []string{
		"acm_rs_vm:namespace:cpu_request:5m",
		"acm_rs_vm:namespace:memory_request:5m",
		"acm_rs_vm:namespace:cpu_usage:5m",
		"acm_rs_vm:namespace:memory_usage:5m",
	}
	for i, expectedRecord := range vmNamespace5mRecords {
		assert.Equal(t, expectedRecord, vmNamespace5mGroup.Rules[i].Record)
	}

	// Verify VM-specific metrics are used
	for _, r := range vmNamespace5mGroup.Rules {
		exprStr := r.Expr.String()
		assert.True(t,
			containsAny(exprStr, "kubevirt_vm_resource_requests", "kubevirt_vmi_cpu_usage_seconds_total", "kubevirt_vmi_memory"),
			"Expression should contain KubeVirt metrics: %s", exprStr,
		)
	}

	// Test VM namespace 1d rules
	vmNamespace1dGroup := rule.Spec.Groups[1]
	assert.Equal(t, "acm-vm-right-sizing-namespace-1d.rules", vmNamespace1dGroup.Name)
	assert.Len(t, vmNamespace1dGroup.Rules, 6)

	// Verify recommendation rules have proper labels
	for _, r := range vmNamespace1dGroup.Rules {
		if r.Labels != nil {
			assert.Equal(t, "Max OverAll", r.Labels["profile"])
			assert.Equal(t, "1d", r.Labels["aggregation"])
		}
	}

	// Test VM cluster 5m rules
	vmCluster5mGroup := rule.Spec.Groups[2]
	assert.Equal(t, "acm-vm-right-sizing-cluster-5m.rule", vmCluster5mGroup.Name)
	assert.Len(t, vmCluster5mGroup.Rules, 4)

	// Verify rule record names in VM cluster 5m group
	vmCluster5mRecords := []string{
		"acm_rs_vm:cluster:cpu_request:5m",
		"acm_rs_vm:cluster:cpu_usage:5m",
		"acm_rs_vm:cluster:memory_request:5m",
		"acm_rs_vm:cluster:memory_usage:5m",
	}
	for i, expectedRecord := range vmCluster5mRecords {
		assert.Equal(t, expectedRecord, vmCluster5mGroup.Rules[i].Record)
	}

	// Test VM cluster 1d rules
	vmCluster1dGroup := rule.Spec.Groups[3]
	assert.Equal(t, "acm-vm-right-sizing-cluster-1d.rule", vmCluster1dGroup.Name)
	assert.Len(t, vmCluster1dGroup.Rules, 6)
}

func TestGeneratePrometheusRuleVMRecommendationPercentage(t *testing.T) {
	percentages := []int{100, 110, 120, 150, 200}

	for _, pct := range percentages {
		configData := common.RSNamespaceConfigMapData{
			PrometheusRuleConfig: common.RSPrometheusRuleConfig{
				RecommendationPercentage: pct,
			},
		}

		rule, err := GeneratePrometheusRule(configData)
		require.NoError(t, err)

		// Check 1d rules for recommendation percentage
		vmNamespace1dGroup := rule.Spec.Groups[1]
		for _, r := range vmNamespace1dGroup.Rules {
			if r.Record == "acm_rs_vm:namespace:cpu_recommendation" ||
				r.Record == "acm_rs_vm:namespace:memory_recommendation" {
				exprStr := r.Expr.String()
				assert.Contains(t, exprStr, "(")
				assert.Contains(t, exprStr, "/100)")
			}
		}
	}
}

func TestGeneratePrometheusRuleKubeVirtMetrics(t *testing.T) {
	configData := common.RSNamespaceConfigMapData{
		PrometheusRuleConfig: common.RSPrometheusRuleConfig{
			NamespaceFilterCriteria: struct {
				InclusionCriteria []string `yaml:"inclusionCriteria"`
				ExclusionCriteria []string `yaml:"exclusionCriteria"`
			}{
				ExclusionCriteria: []string{"openshift.*"},
			},
			RecommendationPercentage: 110,
		},
	}

	rule, err := GeneratePrometheusRule(configData)
	require.NoError(t, err)

	// Verify KubeVirt specific metrics are used
	vmNamespace5mGroup := rule.Spec.Groups[0]

	// Check CPU request rule uses KubeVirt metrics with cores, sockets, threads
	cpuRequestRule := vmNamespace5mGroup.Rules[0]
	assert.Equal(t, "acm_rs_vm:namespace:cpu_request:5m", cpuRequestRule.Record)
	exprStr := cpuRequestRule.Expr.String()
	assert.Contains(t, exprStr, "kubevirt_vm_resource_requests")
	assert.Contains(t, exprStr, "cores")
	assert.Contains(t, exprStr, "sockets")
	assert.Contains(t, exprStr, "threads")

	// Check memory request rule
	memRequestRule := vmNamespace5mGroup.Rules[1]
	assert.Equal(t, "acm_rs_vm:namespace:memory_request:5m", memRequestRule.Record)
	assert.Contains(t, memRequestRule.Expr.String(), "kubevirt_vm_resource_requests")
	assert.Contains(t, memRequestRule.Expr.String(), "memory")

	// Check CPU usage rule
	cpuUsageRule := vmNamespace5mGroup.Rules[2]
	assert.Equal(t, "acm_rs_vm:namespace:cpu_usage:5m", cpuUsageRule.Record)
	assert.Contains(t, cpuUsageRule.Expr.String(), "kubevirt_vmi_cpu_usage_seconds_total")

	// Check memory usage rule
	memUsageRule := vmNamespace5mGroup.Rules[3]
	assert.Equal(t, "acm_rs_vm:namespace:memory_usage:5m", memUsageRule.Record)
	assert.Contains(t, memUsageRule.Expr.String(), "kubevirt_vmi_memory_available_bytes")
	assert.Contains(t, memUsageRule.Expr.String(), "kubevirt_vmi_memory_usable_bytes")
}

// Helper function
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
