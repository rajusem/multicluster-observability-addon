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
	PlacementBindingName     = "rs-virt-policyset-binding"
	PlacementName            = "rs-virt-placement"
	PrometheusRulePolicyName = "rs-virt-prom-rules-policy"
	PrometheusRuleName       = "acm-rs-virt-prometheus-rules"
	ConfigMapName            = "rs-virt-config"
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
		ComponentType:            common.ComponentTypeVirtualization,
		ConfigMapName:            ConfigMapName,
		PlacementName:            PlacementName,
		PlacementBindingName:     PlacementBindingName,
		PrometheusRulePolicyName: PrometheusRulePolicyName,
		DefaultNamespace:         common.DefaultNamespace,
		GetDefaultConfigFunc:     GetDefaultRSVirtualizationConfig,
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

// ApplyRSVirtualizationConfigMapChanges updates PrometheusRule, Policy, Placement based on configmap changes
func ApplyRSVirtualizationConfigMapChanges(ctx context.Context, c client.Client, configData common.RSNamespaceConfigMapData, namespace string) error {
	prometheusRule, err := GeneratePrometheusRule(configData)
	if err != nil {
		return err
	}

	err = CreateOrUpdateVirtualizationPrometheusRulePolicy(ctx, c, prometheusRule, namespace)
	if err != nil {
		return err
	}

	err = CreateUpdateVirtualizationPlacement(ctx, c, configData.PlacementConfiguration, namespace)
	if err != nil {
		return err
	}

	err = CreateVirtualizationPlacementBinding(ctx, c, namespace)
	if err != nil {
		return err
	}

	// Create or update virtualization dashboards (in open-cluster-management-observability namespace)
	if err := common.CreateOrUpdateDashboards(ctx, c, common.VirtualizationDashboardFiles); err != nil {
		return err
	}

	log.Info("rs - virtualization configmap changes applied")

	return nil
}

// CleanupRSVirtualizationResources cleans up the resources created for virtualization right-sizing
func CleanupRSVirtualizationResources(ctx context.Context, c client.Client, namespace string, configNamespace string, bindingUpdated bool) {
	log.V(1).Info("rs - cleaning up virtualization resources if exist")
	componentConfig := common.ComponentConfig{
		ComponentType:            common.ComponentTypeVirtualization,
		ConfigMapName:            ConfigMapName,
		PlacementName:            PlacementName,
		PlacementBindingName:     PlacementBindingName,
		PrometheusRulePolicyName: PrometheusRulePolicyName,
		DefaultNamespace:         common.DefaultNamespace,
	}
	common.CleanupComponentResources(ctx, c, componentConfig, namespace, configNamespace, bindingUpdated)

	// Cleanup virtualization dashboards (from open-cluster-management-observability namespace)
	common.DeleteDashboards(ctx, c, common.VirtualizationDashboardFiles)
}
