// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
)

func TestFormatYAML(t *testing.T) {
	tests := []struct {
		name     string
		config   RSPrometheusRuleConfig
		expected string
	}{
		{
			name: "basic config",
			config: RSPrometheusRuleConfig{
				RecommendationPercentage: 110,
			},
			expected: "namespaceFilterCriteria:\n  inclusionCriteria: []\n  exclusionCriteria: []\nlabelFilterCriteria: []\nrecommendationPercentage: 110\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatYAML(tt.config)
			assert.NotEmpty(t, result)
		})
	}
}

func TestGetDefaultRSPrometheusRuleConfig(t *testing.T) {
	config := GetDefaultRSPrometheusRuleConfig()

	assert.Equal(t, DefaultRecommendationPercentage, config.RecommendationPercentage)
	assert.Equal(t, []string{"openshift.*"}, config.NamespaceFilterCriteria.ExclusionCriteria)
	assert.Empty(t, config.NamespaceFilterCriteria.InclusionCriteria)
	assert.Empty(t, config.LabelFilterCriteria)
}

func TestBuildNamespaceFilter(t *testing.T) {
	tests := []struct {
		name        string
		config      RSPrometheusRuleConfig
		expected    string
		expectError bool
	}{
		{
			name: "with exclusion criteria",
			config: RSPrometheusRuleConfig{
				NamespaceFilterCriteria: struct {
					InclusionCriteria []string `yaml:"inclusionCriteria"`
					ExclusionCriteria []string `yaml:"exclusionCriteria"`
				}{
					ExclusionCriteria: []string{"openshift.*", "kube-.*"},
				},
			},
			expected:    `namespace!~"openshift.*|kube-.*"`,
			expectError: false,
		},
		{
			name: "with inclusion criteria",
			config: RSPrometheusRuleConfig{
				NamespaceFilterCriteria: struct {
					InclusionCriteria []string `yaml:"inclusionCriteria"`
					ExclusionCriteria []string `yaml:"exclusionCriteria"`
				}{
					InclusionCriteria: []string{"my-app-.*", "production-.*"},
				},
			},
			expected:    `namespace=~"my-app-.*|production-.*"`,
			expectError: false,
		},
		{
			name:        "with empty criteria",
			config:      RSPrometheusRuleConfig{},
			expected:    `namespace!=""`,
			expectError: false,
		},
		{
			name: "with both inclusion and exclusion - error",
			config: RSPrometheusRuleConfig{
				NamespaceFilterCriteria: struct {
					InclusionCriteria []string `yaml:"inclusionCriteria"`
					ExclusionCriteria []string `yaml:"exclusionCriteria"`
				}{
					InclusionCriteria: []string{"my-app-.*"},
					ExclusionCriteria: []string{"openshift.*"},
				},
			},
			expected:    "",
			expectError: true,
		},
		{
			name: "with single exclusion",
			config: RSPrometheusRuleConfig{
				NamespaceFilterCriteria: struct {
					InclusionCriteria []string `yaml:"inclusionCriteria"`
					ExclusionCriteria []string `yaml:"exclusionCriteria"`
				}{
					ExclusionCriteria: []string{"openshift.*"},
				},
			},
			expected:    `namespace!~"openshift.*"`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := BuildNamespaceFilter(tt.config)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestBuildLabelJoin(t *testing.T) {
	tests := []struct {
		name         string
		labelFilters []RSLabelFilter
		expected     string
		expectError  bool
	}{
		{
			name:         "empty filters",
			labelFilters: []RSLabelFilter{},
			expected:     "",
			expectError:  false,
		},
		{
			name: "filter with different label name - ignored",
			labelFilters: []RSLabelFilter{
				{
					LabelName:         "label_app",
					InclusionCriteria: []string{"app1"},
				},
			},
			expected:    "",
			expectError: false,
		},
		{
			name: "label_env with inclusion criteria",
			labelFilters: []RSLabelFilter{
				{
					LabelName:         "label_env",
					InclusionCriteria: []string{"prod", "staging"},
				},
			},
			expected:    `* on (namespace) group_left() (kube_namespace_labels{label_env=~"prod|staging"} or kube_namespace_labels{label_env=""})`,
			expectError: false,
		},
		{
			name: "label_env with exclusion criteria",
			labelFilters: []RSLabelFilter{
				{
					LabelName:         "label_env",
					ExclusionCriteria: []string{"dev", "test"},
				},
			},
			expected:    `* on (namespace) group_left() (kube_namespace_labels{label_env!~"dev|test"} or kube_namespace_labels{label_env=""})`,
			expectError: false,
		},
		{
			name: "label_env with both inclusion and exclusion - error",
			labelFilters: []RSLabelFilter{
				{
					LabelName:         "label_env",
					InclusionCriteria: []string{"prod"},
					ExclusionCriteria: []string{"dev"},
				},
			},
			expected:    "",
			expectError: true,
		},
		{
			name: "label_env with empty criteria",
			labelFilters: []RSLabelFilter{
				{
					LabelName: "label_env",
				},
			},
			expected:    "",
			expectError: false,
		},
		{
			name: "multiple filters with label_env",
			labelFilters: []RSLabelFilter{
				{
					LabelName:         "label_app",
					InclusionCriteria: []string{"app1"},
				},
				{
					LabelName:         "label_env",
					InclusionCriteria: []string{"prod"},
				},
			},
			expected:    `* on (namespace) group_left() (kube_namespace_labels{label_env=~"prod"} or kube_namespace_labels{label_env=""})`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := BuildLabelJoin(tt.labelFilters)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFormatYAMLPlacement(t *testing.T) {
	placement := clusterv1beta1.Placement{}
	result := FormatYAML(placement)
	assert.NotEmpty(t, result)
}
