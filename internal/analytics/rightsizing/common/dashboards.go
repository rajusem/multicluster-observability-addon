// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package common

import (
	"context"
	"embed"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

//go:embed dashboards/*.yaml
var dashboardFS embed.FS

var dashboardLog = logf.Log.WithName("rs-dashboards")

// Dashboard file paths (relative to the embed directive)
const (
	NamespaceDashboardFile                      = "dashboards/dash-acm-right-sizing-namespace.yaml"
	VirtualizationMainDashboardFile             = "dashboards/dash-acm-right-sizing-virtualization.yaml"
	VirtualizationOverestimationDashboardFile   = "dashboards/dash-acm-right-sizing-virtualization-overestimation.yaml"
	VirtualizationUnderestimationDashboardFile  = "dashboards/dash-acm-right-sizing-virtualization-underestimation.yaml"
)

// NamespaceDashboardFiles contains the dashboard files for namespace rightsizing
var NamespaceDashboardFiles = []string{
	NamespaceDashboardFile,
}

// VirtualizationDashboardFiles contains the dashboard files for virtualization rightsizing
var VirtualizationDashboardFiles = []string{
	VirtualizationMainDashboardFile,
	VirtualizationOverestimationDashboardFile,
	VirtualizationUnderestimationDashboardFile,
}

// CreateOrUpdateDashboards creates or updates dashboard ConfigMaps from embedded files
// Dashboards are always created in open-cluster-management-observability namespace (from YAML)
func CreateOrUpdateDashboards(ctx context.Context, c client.Client, dashboardFiles []string) error {
	for _, file := range dashboardFiles {
		if err := createOrUpdateDashboardFromFile(ctx, c, file); err != nil {
			return fmt.Errorf("failed to create/update dashboard from %s: %w", file, err)
		}
	}
	return nil
}

// createOrUpdateDashboardFromFile creates or updates a single dashboard ConfigMap from an embedded file
// Note: The namespace from the YAML file is used (open-cluster-management-observability)
func createOrUpdateDashboardFromFile(ctx context.Context, c client.Client, filePath string) error {
	data, err := dashboardFS.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read dashboard file %s: %w", filePath, err)
	}

	cm := &corev1.ConfigMap{}
	if err := yaml.Unmarshal(data, cm); err != nil {
		return fmt.Errorf("failed to unmarshal dashboard ConfigMap from %s: %w", filePath, err)
	}

	// Use the namespace from the YAML file (should be open-cluster-management-observability)

	// Ensure the ConfigMap has the required label for Grafana to pick it up
	if cm.Labels == nil {
		cm.Labels = make(map[string]string)
	}
	cm.Labels["grafana-custom-dashboard"] = "true"

	// Check if the ConfigMap already exists
	existing := &corev1.ConfigMap{}
	err = c.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create the ConfigMap
			dashboardLog.Info("Creating dashboard ConfigMap", "name", cm.Name, "namespace", cm.Namespace)
			if err := c.Create(ctx, cm); err != nil {
				return fmt.Errorf("failed to create dashboard ConfigMap %s: %w", cm.Name, err)
			}
			return nil
		}
		return fmt.Errorf("failed to get existing dashboard ConfigMap %s: %w", cm.Name, err)
	}

	// Update the existing ConfigMap
	existing.Data = cm.Data
	existing.Labels = cm.Labels
	existing.Annotations = cm.Annotations
	dashboardLog.Info("Updating dashboard ConfigMap", "name", cm.Name, "namespace", cm.Namespace)
	if err := c.Update(ctx, existing); err != nil {
		return fmt.Errorf("failed to update dashboard ConfigMap %s: %w", cm.Name, err)
	}

	return nil
}

// DeleteDashboards deletes the dashboard ConfigMaps
// Dashboards are always in open-cluster-management-observability namespace (from YAML)
func DeleteDashboards(ctx context.Context, c client.Client, dashboardFiles []string) {
	for _, file := range dashboardFiles {
		if err := deleteDashboardFromFile(ctx, c, file); err != nil {
			// Log but don't fail on deletion errors
			dashboardLog.Error(err, "Failed to delete dashboard", "file", file)
		}
	}
}

// deleteDashboardFromFile deletes a dashboard ConfigMap based on the embedded file
// Note: The namespace from the YAML file is used (open-cluster-management-observability)
func deleteDashboardFromFile(ctx context.Context, c client.Client, filePath string) error {
	data, err := dashboardFS.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read dashboard file %s: %w", filePath, err)
	}

	cm := &corev1.ConfigMap{}
	if err := yaml.Unmarshal(data, cm); err != nil {
		return fmt.Errorf("failed to unmarshal dashboard ConfigMap from %s: %w", filePath, err)
	}

	// Use the namespace from the YAML file (should be open-cluster-management-observability)

	// Try to delete the ConfigMap
	existing := &corev1.ConfigMap{}
	err = c.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			// Already deleted
			return nil
		}
		return fmt.Errorf("failed to get dashboard ConfigMap %s: %w", cm.Name, err)
	}

	dashboardLog.Info("Deleting dashboard ConfigMap", "name", cm.Name, "namespace", cm.Namespace)
	if err := c.Delete(ctx, existing); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete dashboard ConfigMap %s: %w", cm.Name, err)
	}

	return nil
}
