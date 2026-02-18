// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package handlers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stolostron/multicluster-observability-addon/internal/addon"
	"github.com/stolostron/multicluster-observability-addon/internal/addon/common"
	addoncfg "github.com/stolostron/multicluster-observability-addon/internal/addon/config"
	"github.com/stolostron/multicluster-observability-addon/internal/analytics/rightsizing"
	rsnamespace "github.com/stolostron/multicluster-observability-addon/internal/analytics/rightsizing/namespace"
	rsvirtualization "github.com/stolostron/multicluster-observability-addon/internal/analytics/rightsizing/virtualization"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// MCOAClusterManagementAddOnName is the name of the MCOA ClusterManagementAddOn
	MCOAClusterManagementAddOnName = "multicluster-observability-addon"

	// RightSizingCapableAnnotation indicates MCOA should handle right-sizing deployment
	// If this annotation is not present, MCO handles right-sizing via Policy
	RightSizingCapableAnnotation = "observability.open-cluster-management.io/right-sizing-capable"
)

// OptionsBuilder builds right-sizing options for the helm chart
type OptionsBuilder struct {
	Client client.Client
	Logger logr.Logger
}

// Build builds the right-sizing options based on the addon options and cluster
func (o *OptionsBuilder) Build(ctx context.Context, cluster *clusterv1.ManagedCluster, opts addon.Options) (Options, error) {
	ret := Options{}

	// Skip if platform is not enabled or analytics options are not set
	if !opts.Platform.Enabled {
		return ret, nil
	}

	// Check if this is an OpenShift cluster - right-sizing only works on OpenShift
	if !common.IsOpenShiftVendor(cluster) {
		o.Logger.V(2).Info("Skipping right-sizing for non-OpenShift cluster", "cluster", cluster.Name)
		return ret, nil
	}

	// Check if MCOA should handle right-sizing (annotation must be present with valid version)
	// If annotation is not present or version is unsupported, MCO handles right-sizing via Policy
	capable := o.isRightSizingCapable(ctx)
	o.Logger.V(1).Info("Right-sizing capability check", "capable", capable, "cluster", cluster.Name)
	if !capable {
		o.Logger.V(1).Info("MCOA right-sizing-capable annotation not present or invalid, MCO will handle via Policy", "cluster", cluster.Name)
		return ret, nil
	}
	o.Logger.V(1).Info("MCOA is right-sizing capable, will deploy PrometheusRules via ManifestWork", "cluster", cluster.Name)

	namespaceEnabled := opts.Platform.AnalyticsOptions.RightSizing.NamespaceEnabled
	virtualizationEnabled := opts.Platform.AnalyticsOptions.RightSizing.VirtualizationEnabled

	// Build namespace right-sizing options
	if namespaceEnabled {
		// Ensure ConfigMap exists on hub (MCOA owns all RS resources)
		if err := o.ensureNamespaceConfigMap(ctx); err != nil {
			o.Logger.Error(err, "Failed to ensure namespace ConfigMap exists, continuing with defaults")
		}

		nsOpts, err := o.buildNamespaceOptions(ctx)
		if err != nil {
			return ret, fmt.Errorf("failed to build namespace right-sizing options: %w", err)
		}
		ret.NamespaceRightSizing = nsOpts
	}

	// Build virtualization right-sizing options
	if virtualizationEnabled {
		// Ensure ConfigMap exists on hub (MCOA owns all RS resources)
		if err := o.ensureVirtualizationConfigMap(ctx); err != nil {
			o.Logger.Error(err, "Failed to ensure virtualization ConfigMap exists, continuing with defaults")
		}

		virtOpts, err := o.buildVirtualizationOptions(ctx)
		if err != nil {
			return ret, fmt.Errorf("failed to build virtualization right-sizing options: %w", err)
		}
		ret.VirtualizationRightSizing = virtOpts
	}

	// Generate ScrapeConfig for metrics federation if any right-sizing is enabled
	if namespaceEnabled || virtualizationEnabled {
		scrapeConfig := rightsizing.GenerateScrapeConfig(namespaceEnabled, virtualizationEnabled)
		if scrapeConfig != nil {
			// Add ScrapeConfig to namespace options (it will be merged with platform ScrapeConfigs)
			if namespaceEnabled {
				ret.NamespaceRightSizing.ScrapeConfigs = append(ret.NamespaceRightSizing.ScrapeConfigs, scrapeConfig)
			} else {
				ret.VirtualizationRightSizing.ScrapeConfigs = append(ret.VirtualizationRightSizing.ScrapeConfigs, scrapeConfig)
			}
			o.Logger.V(2).Info("Generated right-sizing ScrapeConfig for metrics federation",
				"namespaceEnabled", namespaceEnabled,
				"virtualizationEnabled", virtualizationEnabled)
		}
	}

	return ret, nil
}

func (o *OptionsBuilder) buildNamespaceOptions(ctx context.Context) (ComponentOptions, error) {
	opts := ComponentOptions{
		Enabled: true,
	}

	// Try to get the config from the hub ConfigMap
	configData, err := o.getConfigData(ctx, rightsizing.NamespaceConfigMapName)
	if err != nil {
		// If ConfigMap doesn't exist, use default config
		if apierrors.IsNotFound(err) {
			o.Logger.V(2).Info("Namespace right-sizing ConfigMap not found, using defaults")
			configData = rightsizing.RSConfigMapData{
				PrometheusRuleConfig: rightsizing.GetDefaultRSPrometheusRuleConfig(),
			}
		} else {
			return opts, fmt.Errorf("failed to get namespace config: %w", err)
		}
	}

	// Generate PrometheusRule
	rule, err := rsnamespace.GeneratePrometheusRule(configData)
	if err != nil {
		return opts, fmt.Errorf("failed to generate namespace PrometheusRule: %w", err)
	}

	opts.PrometheusRules = []*monitoringv1.PrometheusRule{&rule}
	return opts, nil
}

func (o *OptionsBuilder) buildVirtualizationOptions(ctx context.Context) (ComponentOptions, error) {
	opts := ComponentOptions{
		Enabled: true,
	}

	// Try to get the config from the hub ConfigMap
	configData, err := o.getConfigData(ctx, rightsizing.VirtualizationConfigMapName)
	if err != nil {
		// If ConfigMap doesn't exist, use default config
		if apierrors.IsNotFound(err) {
			o.Logger.V(2).Info("Virtualization right-sizing ConfigMap not found, using defaults")
			configData = rightsizing.RSConfigMapData{
				PrometheusRuleConfig: rightsizing.GetDefaultRSPrometheusRuleConfig(),
			}
		} else {
			return opts, fmt.Errorf("failed to get virtualization config: %w", err)
		}
	}

	// Generate PrometheusRule
	rule, err := rsvirtualization.GeneratePrometheusRule(configData)
	if err != nil {
		return opts, fmt.Errorf("failed to generate virtualization PrometheusRule: %w", err)
	}

	opts.PrometheusRules = []*monitoringv1.PrometheusRule{&rule}
	return opts, nil
}

func (o *OptionsBuilder) getConfigData(ctx context.Context, configMapName string) (rightsizing.RSConfigMapData, error) {
	cm, err := common.GetConfigMap(ctx, o.Client, addoncfg.InstallNamespace, configMapName)
	if err != nil {
		return rightsizing.RSConfigMapData{}, err
	}

	return rightsizing.ParseConfigMapData(cm.Data)
}

// ensureNamespaceConfigMap ensures the namespace right-sizing ConfigMap exists on the hub.
// MCOA owns all right-sizing resources including ConfigMaps for cleaner architecture.
func (o *OptionsBuilder) ensureNamespaceConfigMap(ctx context.Context) error {
	_, err := common.GetConfigMap(ctx, o.Client, addoncfg.InstallNamespace, rightsizing.NamespaceConfigMapName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			o.Logger.Info("Creating namespace right-sizing ConfigMap with defaults",
				"name", rightsizing.NamespaceConfigMapName,
				"namespace", addoncfg.InstallNamespace)
			return o.createDefaultConfigMap(ctx, rightsizing.NamespaceConfigMapName, rightsizing.GetDefaultNamespaceConfigData())
		}
		return err
	}
	// ConfigMap already exists
	return nil
}

// ensureVirtualizationConfigMap ensures the virtualization right-sizing ConfigMap exists on the hub.
// MCOA owns all right-sizing resources including ConfigMaps for cleaner architecture.
func (o *OptionsBuilder) ensureVirtualizationConfigMap(ctx context.Context) error {
	_, err := common.GetConfigMap(ctx, o.Client, addoncfg.InstallNamespace, rightsizing.VirtualizationConfigMapName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			o.Logger.Info("Creating virtualization right-sizing ConfigMap with defaults",
				"name", rightsizing.VirtualizationConfigMapName,
				"namespace", addoncfg.InstallNamespace)
			return o.createDefaultConfigMap(ctx, rightsizing.VirtualizationConfigMapName, rightsizing.GetDefaultVirtualizationConfigData())
		}
		return err
	}
	// ConfigMap already exists
	return nil
}

// createDefaultConfigMap creates a ConfigMap with the provided data.
// The ConfigMap is labeled to indicate it's managed by MCOA for right-sizing.
// An owner reference to the ClusterManagementAddOn is added for tracking purposes.
// Note: Cross-scope owner references (cluster-scoped to namespace-scoped) don't enable
// Kubernetes garbage collection, but they help with ownership tracking and tooling.
func (o *OptionsBuilder) createDefaultConfigMap(ctx context.Context, name string, data map[string]string) error {
	// Get the ClusterManagementAddOn for owner reference
	cmao := &addonv1alpha1.ClusterManagementAddOn{}
	if err := o.Client.Get(ctx, types.NamespacedName{Name: MCOAClusterManagementAddOnName}, cmao); err != nil {
		o.Logger.Error(err, "Failed to get ClusterManagementAddOn for owner reference, creating ConfigMap without owner")
		// Continue without owner reference rather than failing
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: addoncfg.InstallNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/component":  "right-sizing",
				"app.kubernetes.io/managed-by": "multicluster-observability-addon",
			},
		},
		Data: data,
	}

	// Add owner reference if we got the ClusterManagementAddOn
	// Note: Cross-scope owner references don't enable garbage collection,
	// but they help with ownership tracking and tooling visibility.
	if cmao.Name != "" {
		cm.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: addonv1alpha1.SchemeGroupVersion.String(),
				Kind:       "ClusterManagementAddOn",
				Name:       cmao.Name,
				UID:        cmao.UID,
			},
		}
	}

	if err := o.Client.Create(ctx, cm); err != nil {
		return fmt.Errorf("failed to create ConfigMap %s: %w", name, err)
	}

	o.Logger.V(1).Info("Created right-sizing ConfigMap", "name", name, "namespace", addoncfg.InstallNamespace)
	return nil
}

// isRightSizingCapable checks if the MCOA ClusterManagementAddOn has the right-sizing-capable annotation
// with a supported version value.
// If the annotation is present with value "v1", MCOA handles right-sizing deployment via ManifestWork.
// If the annotation is not present or has an unsupported version, MCO handles right-sizing via Policy
// (for backward compatibility).
func (o *OptionsBuilder) isRightSizingCapable(ctx context.Context) bool {
	cmao := &addonv1alpha1.ClusterManagementAddOn{}
	err := o.Client.Get(ctx, types.NamespacedName{Name: MCOAClusterManagementAddOnName}, cmao)
	if err != nil {
		// If we can't get the ClusterManagementAddOn, assume MCO should handle
		o.Logger.Error(err, "Failed to get ClusterManagementAddOn, assuming MCO handles right-sizing")
		return false
	}

	o.Logger.V(2).Info("Got ClusterManagementAddOn", "name", cmao.Name, "annotations", cmao.Annotations)

	if cmao.Annotations == nil {
		o.Logger.V(2).Info("ClusterManagementAddOn has no annotations")
		return false
	}

	value, exists := cmao.Annotations[RightSizingCapableAnnotation]
	o.Logger.V(2).Info("Annotation check result", "annotation", RightSizingCapableAnnotation, "exists", exists, "value", value)

	if !exists {
		return false
	}

	// Version check - currently only v1 is supported
	if value != "v1" {
		o.Logger.V(1).Info("Unsupported right-sizing capability version", "version", value, "expected", "v1")
		return false
	}

	return true
}
