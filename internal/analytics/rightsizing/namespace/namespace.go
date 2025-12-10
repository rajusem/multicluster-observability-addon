// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package namespace

import (
	"context"

	"github.com/stolostron/multicluster-observability-addon/internal/analytics/rightsizing/common"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	// Namespace-specific resource names
	PlacementName      = "rs-namespace-placement"
	PrometheusRuleName = "acm-rs-namespace-prometheus-rules"
	ConfigMapName      = "rs-namespace-config"
	// Addon-based deployment names
	AddonName    = "observability-rightsizing-namespace"
	TemplateName = "rs-namespace-template"
)

var (
	log = logf.Log.WithName("rs-namespace")

	// ComponentState holds the runtime state
	ComponentState = &common.ComponentState{
		Namespace: common.DefaultNamespace,
		Enabled:   false,
	}
)

// GetComponentConfig returns the component configuration for namespace right-sizing
func GetComponentConfig(bindingNamespace string) common.ComponentConfig {
	return common.ComponentConfig{
		ComponentType:        common.ComponentTypeNamespace,
		ConfigMapName:        ConfigMapName,
		PlacementName:        PlacementName,
		DefaultNamespace:     common.DefaultNamespace,
		GetDefaultConfigFunc: GetDefaultRSNamespaceConfig,
		AddonName:            AddonName,
		TemplateName:         TemplateName,
		ApplyChangesFunc: func(configData common.RSNamespaceConfigMapData) error {
			// This will be set up with proper context when called
			return nil
		},
	}
}

// HandleRightSizing handles the namespace right-sizing functionality
func HandleRightSizing(ctx context.Context, c client.Client, opts common.RightSizingOptions) error {
	log.V(1).Info("rs - handling namespace right-sizing")

	componentConfig := common.ComponentConfig{
		ComponentType:        common.ComponentTypeNamespace,
		ConfigMapName:        ConfigMapName,
		PlacementName:        PlacementName,
		DefaultNamespace:     common.DefaultNamespace,
		GetDefaultConfigFunc: GetDefaultRSNamespaceConfig,
		AddonName:            AddonName,
		TemplateName:         TemplateName,
		ApplyChangesFunc: func(configData common.RSNamespaceConfigMapData) error {
			return ApplyRSNamespaceConfigMapChanges(ctx, c, configData, ComponentState.Namespace)
		},
	}

	return common.HandleComponentRightSizing(ctx, c, opts, componentConfig, ComponentState)
}

// GetDefaultRSNamespaceConfig returns default config data
func GetDefaultRSNamespaceConfig() map[string]string {
	// get default config data with PrometheusRule config and placement config
	ruleConfig := common.GetDefaultRSPrometheusRuleConfig()
	placement := common.GetDefaultRSPlacement()

	return map[string]string{
		"prometheusRuleConfig":   common.FormatYAML(ruleConfig),
		"placementConfiguration": common.FormatYAML(placement),
	}
}

// GetRightSizingConfigData extracts and unmarshals the data from the ConfigMap into RightSizingConfigData
func GetRightSizingConfigData(cm *corev1.ConfigMap) (common.RSNamespaceConfigMapData, error) {
	return common.GetRSConfigData(cm)
}

// GetNamespaceRSConfigMapPredicateFunc gets the namespace rightsizing predicate function
func GetNamespaceRSConfigMapPredicateFunc(ctx context.Context, c client.Client, configNamespace string) predicate.Funcs {
	return common.GetRSConfigMapPredicateFunc(ctx, c, ConfigMapName, configNamespace, func(ctx context.Context, c client.Client, configData common.RSNamespaceConfigMapData) error {
		return ApplyRSNamespaceConfigMapChanges(ctx, c, configData, ComponentState.Namespace)
	})
}

// ApplyRSNamespaceConfigMapChanges creates/updates the addon resources based on configmap changes
// This creates ClusterManagementAddOn, AddOnTemplate (with PrometheusRule), and Placement
func ApplyRSNamespaceConfigMapChanges(ctx context.Context, c client.Client, configData common.RSNamespaceConfigMapData, namespace string) error {
	prometheusRule, err := GeneratePrometheusRule(configData)
	if err != nil {
		return err
	}

	// Create addon configuration
	addonConfig := common.RightSizingAddonConfig{
		AddonName:          AddonName,
		TemplateName:       TemplateName,
		PlacementName:      PlacementName,
		PlacementNamespace: namespace,
		PrometheusRule:     prometheusRule,
		PlacementSpec:      configData.PlacementConfiguration.Spec,
	}

	// Create or update the addon resources
	if err := common.CreateOrUpdateRightSizingAddon(ctx, c, addonConfig); err != nil {
		return err
	}

	// Create or update namespace dashboards (in open-cluster-management-observability namespace)
	if err := common.CreateOrUpdateDashboards(ctx, c, common.NamespaceDashboardFiles); err != nil {
		return err
	}

	log.Info("rs - namespace addon resources applied")

	return nil
}

// CleanupRSNamespaceResources cleans up the resources created for namespace right-sizing
func CleanupRSNamespaceResources(ctx context.Context, c client.Client, namespace string, configNamespace string, bindingUpdated bool) {
	log.V(1).Info("rs - cleaning up namespace addon resources if exist")
	componentConfig := common.ComponentConfig{
		ComponentType:        common.ComponentTypeNamespace,
		ConfigMapName:        ConfigMapName,
		PlacementName:        PlacementName,
		DefaultNamespace:     common.DefaultNamespace,
		AddonName:            AddonName,
		TemplateName:         TemplateName,
	}
	common.CleanupComponentResources(ctx, c, componentConfig, namespace, configNamespace, bindingUpdated)

	// Cleanup namespace dashboards (from open-cluster-management-observability namespace)
	common.DeleteDashboards(ctx, c, common.NamespaceDashboardFiles)
}
