// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package namespace

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
		validate    func(t *testing.T, rule interface{})
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
			validate: func(t *testing.T, rule interface{}) {
				promRule := rule
				assert.NotNil(t, promRule)
			},
		},
		{
			name: "with inclusion criteria",
			configData: common.RSNamespaceConfigMapData{
				PrometheusRuleConfig: common.RSPrometheusRuleConfig{
					NamespaceFilterCriteria: struct {
						InclusionCriteria []string `yaml:"inclusionCriteria"`
						ExclusionCriteria []string `yaml:"exclusionCriteria"`
					}{
						InclusionCriteria: []string{"my-app-.*", "production-.*"},
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
							InclusionCriteria: []string{"prod", "staging"},
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
						InclusionCriteria: []string{"my-app-.*"},
						ExclusionCriteria: []string{"openshift.*"},
					},
					RecommendationPercentage: 110,
				},
			},
			expectError: true,
		},
		{
			name: "invalid - both inclusion and exclusion for label_env",
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
							ExclusionCriteria: []string{"dev"},
						},
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

			// Verify group names
			groupNames := make([]string, len(rule.Spec.Groups))
			for i, g := range rule.Spec.Groups {
				groupNames[i] = g.Name
			}
			assert.Contains(t, groupNames, "acm-right-sizing-namespace-5m.rule")
			assert.Contains(t, groupNames, "acm-right-sizing-namespace-1d.rules")
			assert.Contains(t, groupNames, "acm-right-sizing-cluster-5m.rule")
			assert.Contains(t, groupNames, "acm-right-sizing-cluster-1d.rule")

			if tt.validate != nil {
				tt.validate(t, rule)
			}
		})
	}
}

func TestGeneratePrometheusRuleRuleGroups(t *testing.T) {
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

	// Test namespace 5m rules
	namespace5mGroup := rule.Spec.Groups[0]
	assert.Equal(t, "acm-right-sizing-namespace-5m.rule", namespace5mGroup.Name)
	assert.Len(t, namespace5mGroup.Rules, 6)

	// Verify rule record names in namespace 5m group
	namespace5mRecords := []string{
		"acm_rs:namespace:cpu_request_hard:5m",
		"acm_rs:namespace:cpu_request:5m",
		"acm_rs:namespace:cpu_usage:5m",
		"acm_rs:namespace:memory_request_hard:5m",
		"acm_rs:namespace:memory_request:5m",
		"acm_rs:namespace:memory_usage:5m",
	}
	for i, expectedRecord := range namespace5mRecords {
		assert.Equal(t, expectedRecord, namespace5mGroup.Rules[i].Record)
	}

	// Test namespace 1d rules
	namespace1dGroup := rule.Spec.Groups[1]
	assert.Equal(t, "acm-right-sizing-namespace-1d.rules", namespace1dGroup.Name)
	assert.Len(t, namespace1dGroup.Rules, 8)

	// Verify recommendation rules have proper labels
	for _, r := range namespace1dGroup.Rules {
		if r.Labels != nil {
			assert.Equal(t, "Max OverAll", r.Labels["profile"])
			assert.Equal(t, "1d", r.Labels["aggregation"])
		}
	}

	// Test cluster 5m rules
	cluster5mGroup := rule.Spec.Groups[2]
	assert.Equal(t, "acm-right-sizing-cluster-5m.rule", cluster5mGroup.Name)
	assert.Len(t, cluster5mGroup.Rules, 6)

	// Verify rule record names in cluster 5m group
	cluster5mRecords := []string{
		"acm_rs:cluster:cpu_request_hard:5m",
		"acm_rs:cluster:cpu_request:5m",
		"acm_rs:cluster:cpu_usage:5m",
		"acm_rs:cluster:memory_request_hard:5m",
		"acm_rs:cluster:memory_request:5m",
		"acm_rs:cluster:memory_usage:5m",
	}
	for i, expectedRecord := range cluster5mRecords {
		assert.Equal(t, expectedRecord, cluster5mGroup.Rules[i].Record)
	}

	// Test cluster 1d rules
	cluster1dGroup := rule.Spec.Groups[3]
	assert.Equal(t, "acm-right-sizing-cluster-1d.rule", cluster1dGroup.Name)
	assert.Len(t, cluster1dGroup.Rules, 8)
}

func TestGeneratePrometheusRuleWithLabelJoin(t *testing.T) {
	configData := common.RSNamespaceConfigMapData{
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
	}

	rule, err := GeneratePrometheusRule(configData)
	require.NoError(t, err)

	// Check that rules contain the label join expression
	namespace5mGroup := rule.Spec.Groups[0]
	for _, r := range namespace5mGroup.Rules {
		exprStr := r.Expr.String()
		assert.Contains(t, exprStr, "on (namespace) group_left()")
		assert.Contains(t, exprStr, "kube_namespace_labels")
	}
}

func TestGeneratePrometheusRuleRecommendationPercentage(t *testing.T) {
	percentages := []int{100, 110, 120, 150, 200}

	for _, pct := range percentages {
		t.Run(
			"percentage_"+string(rune(pct)),
			func(t *testing.T) {
				configData := common.RSNamespaceConfigMapData{
					PrometheusRuleConfig: common.RSPrometheusRuleConfig{
						RecommendationPercentage: pct,
					},
				}

				rule, err := GeneratePrometheusRule(configData)
				require.NoError(t, err)

				// Check 1d rules for recommendation percentage
				namespace1dGroup := rule.Spec.Groups[1]
				for _, r := range namespace1dGroup.Rules {
					if r.Record == "acm_rs:namespace:cpu_recommendation" ||
						r.Record == "acm_rs:namespace:memory_recommendation" {
						exprStr := r.Expr.String()
						assert.Contains(t, exprStr, "* (")
						assert.Contains(t, exprStr, "/100)")
					}
				}
			},
		)
	}
}
