// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package common

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// AddonLifecycleAnnotation is the annotation key for addon lifecycle management
	AddonLifecycleAnnotation = "addon.open-cluster-management.io/lifecycle"
	// AddonLifecycleAddonManager indicates the addon is managed by addon-manager
	AddonLifecycleAddonManager = "addon-manager"
	// SpecHashAnnotation is used to track template spec changes for triggering ManifestWork updates
	SpecHashAnnotation = "observability.open-cluster-management.io/spec-hash"
)

// RightSizingAddonConfig holds configuration for creating a rightsizing addon
type RightSizingAddonConfig struct {
	// AddonName is the name of the ClusterManagementAddOn (e.g., "observability-rightsizing-namespace")
	AddonName string
	// TemplateName is the name of the AddOnTemplate
	TemplateName string
	// PlacementName is the name of the Placement resource
	PlacementName string
	// PlacementNamespace is the namespace where Placement is created
	PlacementNamespace string
	// PrometheusRule is the rule to be deployed to managed clusters
	PrometheusRule monitoringv1.PrometheusRule
	// PlacementSpec is the placement specification from ConfigMap
	PlacementSpec clusterv1beta1.PlacementSpec
}

// CreateOrUpdateRightSizingAddon creates or updates the ClusterManagementAddOn and AddOnTemplate
// for a rightsizing component
func CreateOrUpdateRightSizingAddon(ctx context.Context, c client.Client, config RightSizingAddonConfig) error {
	// 1. Create or update AddOnTemplate
	if err := createOrUpdateAddOnTemplate(ctx, c, config); err != nil {
		return fmt.Errorf("failed to create/update AddOnTemplate: %w", err)
	}

	// 2. Create or update Placement
	if err := createOrUpdatePlacement(ctx, c, config); err != nil {
		return fmt.Errorf("failed to create/update Placement: %w", err)
	}

	// 3. Create or update ClusterManagementAddOn
	if err := createOrUpdateClusterManagementAddOn(ctx, c, config); err != nil {
		return fmt.Errorf("failed to create/update ClusterManagementAddOn: %w", err)
	}

	log.Info("rs - rightsizing addon resources created/updated successfully", "addonName", config.AddonName)
	return nil
}

// createOrUpdateAddOnTemplate creates or updates the AddOnTemplate with PrometheusRule
func createOrUpdateAddOnTemplate(ctx context.Context, c client.Client, config RightSizingAddonConfig) error {
	// Convert PrometheusRule to unstructured for embedding in template
	promRuleMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&config.PrometheusRule)
	if err != nil {
		return fmt.Errorf("failed to convert PrometheusRule to unstructured: %w", err)
	}

	promRuleJSON, err := json.Marshal(promRuleMap)
	if err != nil {
		return fmt.Errorf("failed to marshal PrometheusRule: %w", err)
	}

	// Calculate hash of the PrometheusRule content to detect changes
	// This hash is used to trigger ManifestWork regeneration when content changes
	specHash := calculateSpecHash(promRuleJSON)

	template := &addonv1alpha1.AddOnTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.TemplateName,
		},
	}

	// Check if template exists
	err = c.Get(ctx, types.NamespacedName{Name: config.TemplateName}, template)
	templateExists := err == nil

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get AddOnTemplate: %w", err)
	}

	// Set annotations with spec hash to trigger ManifestWork updates on content changes
	if template.Annotations == nil {
		template.Annotations = make(map[string]string)
	}
	template.Annotations[SpecHashAnnotation] = specHash

	// Set template spec
	template.Spec = addonv1alpha1.AddOnTemplateSpec{
		AddonName: config.AddonName,
		AgentSpec: workv1.ManifestWorkSpec{
			Workload: workv1.ManifestsTemplate{
				Manifests: []workv1.Manifest{
					{
						RawExtension: runtime.RawExtension{
							Raw: promRuleJSON,
						},
					},
				},
			},
		},
	}

	if templateExists {
		if err := c.Update(ctx, template); err != nil {
			return fmt.Errorf("failed to update AddOnTemplate: %w", err)
		}
		log.Info("rs - updated AddOnTemplate", "name", config.TemplateName, "specHash", specHash)
	} else {
		if err := c.Create(ctx, template); err != nil {
			return fmt.Errorf("failed to create AddOnTemplate: %w", err)
		}
		log.Info("rs - created AddOnTemplate", "name", config.TemplateName, "specHash", specHash)
	}

	return nil
}

// calculateSpecHash computes a SHA256 hash of the given data and returns it as a hex string
func calculateSpecHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// createOrUpdatePlacement creates or updates the Placement resource
func createOrUpdatePlacement(ctx context.Context, c client.Client, config RightSizingAddonConfig) error {
	placement := &clusterv1beta1.Placement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.PlacementName,
			Namespace: config.PlacementNamespace,
		},
	}

	// Check if placement exists
	err := c.Get(ctx, types.NamespacedName{
		Name:      config.PlacementName,
		Namespace: config.PlacementNamespace,
	}, placement)
	placementExists := err == nil

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get Placement: %w", err)
	}

	// Set placement spec from config
	placement.Spec = config.PlacementSpec

	if placementExists {
		if err := c.Update(ctx, placement); err != nil {
			return fmt.Errorf("failed to update Placement: %w", err)
		}
		log.Info("rs - updated Placement", "name", config.PlacementName, "namespace", config.PlacementNamespace)
	} else {
		if err := c.Create(ctx, placement); err != nil {
			return fmt.Errorf("failed to create Placement: %w", err)
		}
		log.Info("rs - created Placement", "name", config.PlacementName, "namespace", config.PlacementNamespace)
	}

	return nil
}

// createOrUpdateClusterManagementAddOn creates or updates the ClusterManagementAddOn
func createOrUpdateClusterManagementAddOn(ctx context.Context, c client.Client, config RightSizingAddonConfig) error {
	cmao := &addonv1alpha1.ClusterManagementAddOn{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.AddonName,
			Annotations: map[string]string{
				AddonLifecycleAnnotation: AddonLifecycleAddonManager,
			},
		},
	}

	// Check if CMAO exists
	err := c.Get(ctx, types.NamespacedName{Name: config.AddonName}, cmao)
	cmaoExists := err == nil

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get ClusterManagementAddOn: %w", err)
	}

	// Set CMAO spec
	cmao.Spec = addonv1alpha1.ClusterManagementAddOnSpec{
		AddOnMeta: addonv1alpha1.AddOnMeta{
			DisplayName: fmt.Sprintf("Observability Right-Sizing (%s)", config.AddonName),
			Description: "Deploys PrometheusRule resources for ACM right-sizing metrics collection",
		},
		SupportedConfigs: []addonv1alpha1.ConfigMeta{
			{
				ConfigGroupResource: addonv1alpha1.ConfigGroupResource{
					Group:    "addon.open-cluster-management.io",
					Resource: "addontemplates",
				},
				DefaultConfig: &addonv1alpha1.ConfigReferent{
					Name: config.TemplateName,
				},
			},
		},
		InstallStrategy: addonv1alpha1.InstallStrategy{
			Type: addonv1alpha1.AddonInstallStrategyPlacements,
			Placements: []addonv1alpha1.PlacementStrategy{
				{
					PlacementRef: addonv1alpha1.PlacementRef{
						Name:      config.PlacementName,
						Namespace: config.PlacementNamespace,
					},
					RolloutStrategy: clusterv1alpha1.RolloutStrategy{
						Type: clusterv1alpha1.All,
					},
				},
			},
		},
	}

	// Ensure annotation is set
	if cmao.Annotations == nil {
		cmao.Annotations = make(map[string]string)
	}
	cmao.Annotations[AddonLifecycleAnnotation] = AddonLifecycleAddonManager

	if cmaoExists {
		if err := c.Update(ctx, cmao); err != nil {
			return fmt.Errorf("failed to update ClusterManagementAddOn: %w", err)
		}
		log.Info("rs - updated ClusterManagementAddOn", "name", config.AddonName)
	} else {
		if err := c.Create(ctx, cmao); err != nil {
			return fmt.Errorf("failed to create ClusterManagementAddOn: %w", err)
		}
		log.Info("rs - created ClusterManagementAddOn", "name", config.AddonName)
	}

	return nil
}

// CleanupRightSizingAddon deletes the ClusterManagementAddOn, AddOnTemplate, and Placement
func CleanupRightSizingAddon(ctx context.Context, c client.Client, addonName, templateName, placementName, placementNamespace string) {
	// Delete ClusterManagementAddOn
	cmao := &addonv1alpha1.ClusterManagementAddOn{
		ObjectMeta: metav1.ObjectMeta{
			Name: addonName,
		},
	}
	if err := c.Delete(ctx, cmao); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "rs - failed to delete ClusterManagementAddOn", "name", addonName)
		}
	} else {
		log.Info("rs - deleted ClusterManagementAddOn", "name", addonName)
	}

	// Delete AddOnTemplate
	template := &addonv1alpha1.AddOnTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: templateName,
		},
	}
	if err := c.Delete(ctx, template); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "rs - failed to delete AddOnTemplate", "name", templateName)
		}
	} else {
		log.Info("rs - deleted AddOnTemplate", "name", templateName)
	}

	// Delete Placement
	placement := &clusterv1beta1.Placement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      placementName,
			Namespace: placementNamespace,
		},
	}
	if err := c.Delete(ctx, placement); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "rs - failed to delete Placement", "name", placementName, "namespace", placementNamespace)
		}
	} else {
		log.Info("rs - deleted Placement", "name", placementName, "namespace", placementNamespace)
	}
}
