// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package virtualization

import (
	"context"

	"github.com/stolostron/multicluster-observability-addon/internal/analytics/rightsizing/common"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	// Virtualization-specific resource names
	PlacementName      = "rs-virt-placement"
	PrometheusRuleName = "acm-rs-virt-prometheus-rules"
	ConfigMapName      = "rs-virt-config"
	// Addon-based deployment names
	AddonName    = "observability-rightsizing-virtualization"
	TemplateName = "rs-virt-template"
)

var (
	log = logf.Log.WithName("rs-virtualization")

	// ComponentState holds the runtime state
	ComponentState = &common.ComponentState{
		Namespace: common.DefaultNamespace,
		Enabled:   false,
	}
)

// HandleRightSizing handles the virtualization right-sizing functionality
func HandleRightSizing(ctx context.Context, c client.Client, opts common.RightSizingOptions) error {
	log.V(1).Info("rs - handling virtualization right-sizing")

	componentConfig := common.ComponentConfig{
		ComponentType:        common.ComponentTypeVirtualization,
		ConfigMapName:        ConfigMapName,
		PlacementName:        PlacementName,
		DefaultNamespace:     common.DefaultNamespace,
		GetDefaultConfigFunc: GetDefaultRSVirtualizationConfig,
		AddonName:            AddonName,
		TemplateName:         TemplateName,
		ApplyChangesFunc: func(configData common.RSNamespaceConfigMapData) error {
			return ApplyRSVirtualizationConfigMapChanges(ctx, c, configData, ComponentState.Namespace)
		},
	}

	return common.HandleComponentRightSizing(ctx, c, opts, componentConfig, ComponentState)
}

// GetDefaultRSVirtualizationConfig returns default config data
func GetDefaultRSVirtualizationConfig() map[string]string {
	// get default config data with PrometheusRule config and placement config
	ruleConfig := common.GetDefaultRSPrometheusRuleConfig()
	placement := common.GetDefaultRSPlacement()

	return map[string]string{
		"prometheusRuleConfig":   common.FormatYAML(ruleConfig),
		"placementConfiguration": common.FormatYAML(placement),
	}
}

// GetRightSizingVirtualizationConfigData extracts and unmarshals the data from the ConfigMap into RSVirtualizationConfigMapData
func GetRightSizingVirtualizationConfigData(cm *corev1.ConfigMap) (common.RSNamespaceConfigMapData, error) {
	return common.GetRSConfigData(cm)
}

// GetVirtualizationRSConfigMapPredicateFunc returns predicate for virtualization right-sizing ConfigMap
func GetVirtualizationRSConfigMapPredicateFunc(ctx context.Context, c client.Client, configNamespace string) predicate.Funcs {
	return common.GetRSConfigMapPredicateFunc(ctx, c, ConfigMapName, configNamespace, func(ctx context.Context, c client.Client, configData common.RSNamespaceConfigMapData) error {
		return ApplyRSVirtualizationConfigMapChanges(ctx, c, configData, ComponentState.Namespace)
	})
}

// ApplyRSVirtualizationConfigMapChanges creates/updates the addon resources based on configmap changes
// This creates ClusterManagementAddOn, AddOnTemplate (with PrometheusRule), and Placement
func ApplyRSVirtualizationConfigMapChanges(ctx context.Context, c client.Client, configData common.RSNamespaceConfigMapData, namespace string) error {
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

	// Create or update virtualization dashboards (in open-cluster-management-observability namespace)
	if err := common.CreateOrUpdateDashboards(ctx, c, common.VirtualizationDashboardFiles); err != nil {
		return err
	}

	log.Info("rs - virtualization addon resources applied")

	return nil
}

// CleanupRSVirtualizationResources cleans up the resources created for virtualization right-sizing
func CleanupRSVirtualizationResources(ctx context.Context, c client.Client, namespace string, configNamespace string, bindingUpdated bool) {
	log.V(1).Info("rs - cleaning up virtualization addon resources if exist")
	componentConfig := common.ComponentConfig{
		ComponentType:        common.ComponentTypeVirtualization,
		ConfigMapName:        ConfigMapName,
		PlacementName:        PlacementName,
		DefaultNamespace:     common.DefaultNamespace,
		AddonName:            AddonName,
		TemplateName:         TemplateName,
	}
	common.CleanupComponentResources(ctx, c, componentConfig, namespace, configNamespace, bindingUpdated)

	// Cleanup virtualization dashboards (from open-cluster-management-observability namespace)
	common.DeleteDashboards(ctx, c, common.VirtualizationDashboardFiles)
}
