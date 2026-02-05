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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	// Build namespace right-sizing options
	if opts.Platform.AnalyticsOptions.RightSizing.NamespaceEnabled {
		nsOpts, err := o.buildNamespaceOptions(ctx)
		if err != nil {
			return ret, fmt.Errorf("failed to build namespace right-sizing options: %w", err)
		}
		ret.NamespaceRightSizing = nsOpts
	}

	// Build virtualization right-sizing options
	if opts.Platform.AnalyticsOptions.RightSizing.VirtualizationEnabled {
		virtOpts, err := o.buildVirtualizationOptions(ctx)
		if err != nil {
			return ret, fmt.Errorf("failed to build virtualization right-sizing options: %w", err)
		}
		ret.VirtualizationRightSizing = virtOpts
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
