// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package common

import (
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	log = logf.Log.WithName("rightsizing-common")
)

// Common constants
const (
	DefaultRecommendationPercentage = 110
	MonitoringNamespace             = "openshift-monitoring"
	DefaultNamespace                = "open-cluster-management-global-set"
)

// ComponentType represents the type of right-sizing component
type ComponentType string

const (
	ComponentTypeNamespace      ComponentType = "namespace"
	ComponentTypeVirtualization ComponentType = "virtualization"
)

// RSLabelFilter represents label filtering criteria for right-sizing
type RSLabelFilter struct {
	LabelName         string   `yaml:"labelName"`
	InclusionCriteria []string `yaml:"inclusionCriteria,omitempty"`
	ExclusionCriteria []string `yaml:"exclusionCriteria,omitempty"`
}

// RSPrometheusRuleConfig represents the Prometheus rule configuration for right-sizing
type RSPrometheusRuleConfig struct {
	NamespaceFilterCriteria struct {
		InclusionCriteria []string `yaml:"inclusionCriteria"`
		ExclusionCriteria []string `yaml:"exclusionCriteria"`
	} `yaml:"namespaceFilterCriteria"`
	LabelFilterCriteria      []RSLabelFilter `yaml:"labelFilterCriteria"`
	RecommendationPercentage int             `yaml:"recommendationPercentage"`
}

// RSNamespaceConfigMapData represents the configmap data structure for right-sizing namespace
type RSNamespaceConfigMapData struct {
	PrometheusRuleConfig   RSPrometheusRuleConfig   `yaml:"prometheusRuleConfig"`
	PlacementConfiguration clusterv1beta1.Placement `yaml:"placementConfiguration"`
}

// ComponentConfig holds configuration for a right-sizing component
type ComponentConfig struct {
	ComponentType        ComponentType
	ConfigMapName        string
	PlacementName        string
	DefaultNamespace     string
	GetDefaultConfigFunc func() map[string]string
	ApplyChangesFunc     func(RSNamespaceConfigMapData) error
	// Addon-based deployment fields
	AddonName    string // Name of the ClusterManagementAddOn (e.g., "observability-rightsizing-namespace")
	TemplateName string // Name of the AddOnTemplate
}

// ComponentState holds the runtime state for a component
type ComponentState struct {
	Namespace string
	Enabled   bool
}

// RightSizingOptions holds the configuration options for right-sizing features
type RightSizingOptions struct {
	NamespaceEnabled         bool
	NamespaceBinding         string
	VirtualizationEnabled    bool
	VirtualizationBinding    string
	ConfigNamespace          string
}
