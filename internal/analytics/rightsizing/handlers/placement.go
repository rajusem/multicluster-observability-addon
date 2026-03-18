// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package handlers

import (
	"context"
	"fmt"

	"github.com/stolostron/multicluster-observability-addon/internal/addon"
	"github.com/stolostron/multicluster-observability-addon/internal/analytics/rightsizing"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcilePlacements ensures Placement resources exist for each enabled RS feature.
// Called from ResourceCreator (hub-wide, not per-cluster) to avoid race conditions.
func (o *OptionsBuilder) ReconcilePlacements(ctx context.Context, opts addon.Options) error {
	if !opts.Platform.Enabled {
		return nil
	}

	if opts.Platform.AnalyticsOptions.RightSizing.NamespaceEnabled {
		configData, err := o.getConfigData(ctx, rightsizing.NamespaceConfigMapName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				configData.PlacementConfiguration = rightsizing.GetDefaultRSPlacement()
			} else {
				return fmt.Errorf("failed to get namespace config: %w", err)
			}
		}
		if err := o.ensurePlacement(ctx, rightsizing.NamespacePlacementName, configData.PlacementConfiguration); err != nil {
			return fmt.Errorf("failed to ensure namespace placement: %w", err)
		}
	}

	if opts.Platform.AnalyticsOptions.RightSizing.VirtualizationEnabled {
		configData, err := o.getConfigData(ctx, rightsizing.VirtualizationConfigMapName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				configData.PlacementConfiguration = rightsizing.GetDefaultRSPlacement()
			} else {
				return fmt.Errorf("failed to get virtualization config: %w", err)
			}
		}
		if err := o.ensurePlacement(ctx, rightsizing.VirtualizationPlacementName, configData.PlacementConfiguration); err != nil {
			return fmt.Errorf("failed to ensure virtualization placement: %w", err)
		}
	}

	return nil
}

// ensurePlacement creates or updates a Placement resource with owner reference.
// Handles AlreadyExists race condition gracefully.
func (o *OptionsBuilder) ensurePlacement(ctx context.Context, placementName string, placementConfig clusterv1beta1.Placement) error {
	key := types.NamespacedName{Name: placementName, Namespace: rightsizing.PlacementNamespace}
	placement := &clusterv1beta1.Placement{}

	err := o.Client.Get(ctx, key, placement)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get placement %s: %w", placementName, err)
		}

		// Create new placement
		placement = &clusterv1beta1.Placement{
			ObjectMeta: metav1.ObjectMeta{
				Name:      placementName,
				Namespace: rightsizing.PlacementNamespace,
				Labels: map[string]string{
					"app.kubernetes.io/component":  "right-sizing",
					"app.kubernetes.io/managed-by": "multicluster-observability-addon",
				},
			},
			Spec: placementConfig.Spec,
		}

		// Add owner reference for tracking
		cmao := &addonv1alpha1.ClusterManagementAddOn{}
		if err := o.Client.Get(ctx, types.NamespacedName{Name: MCOAClusterManagementAddOnName}, cmao); err == nil {
			placement.OwnerReferences = []metav1.OwnerReference{{
				APIVersion: addonv1alpha1.SchemeGroupVersion.String(),
				Kind:       "ClusterManagementAddOn",
				Name:       cmao.Name,
				UID:        cmao.UID,
			}}
		}

		if err := o.Client.Create(ctx, placement); err != nil {
			if apierrors.IsAlreadyExists(err) {
				// Concurrent create — fall through to update
				if err := o.Client.Get(ctx, key, placement); err != nil {
					return fmt.Errorf("failed to re-fetch placement after AlreadyExists: %w", err)
				}
			} else {
				return fmt.Errorf("failed to create placement %s: %w", placementName, err)
			}
		} else {
			o.Logger.V(1).Info("Created right-sizing Placement", "name", placementName, "namespace", rightsizing.PlacementNamespace)
			return nil
		}
	}

	// Update existing placement spec
	placement.Spec = placementConfig.Spec
	if err := o.Client.Update(ctx, placement); err != nil {
		return fmt.Errorf("failed to update placement %s: %w", placementName, err)
	}
	o.Logger.V(1).Info("Updated right-sizing Placement", "name", placementName, "namespace", rightsizing.PlacementNamespace)
	return nil
}

// isClusterSelectedByPlacement checks if a cluster is selected by a Placement
// by reading the PlacementDecisions associated with that Placement.
func (o *OptionsBuilder) isClusterSelectedByPlacement(ctx context.Context, placementName, clusterName string) (bool, error) {
	placementDecisionList := &clusterv1beta1.PlacementDecisionList{}
	err := o.Client.List(ctx, placementDecisionList,
		client.InNamespace(rightsizing.PlacementNamespace),
		client.MatchingLabels{rightsizing.PlacementDecisionLabel: placementName},
	)
	if err != nil {
		return false, fmt.Errorf("failed to list PlacementDecisions for %s: %w", placementName, err)
	}

	if len(placementDecisionList.Items) == 0 {
		// No PlacementDecisions yet — Placement may be newly created.
		// Default to true (fail-open) to avoid blocking deployment while scheduler catches up.
		// Window is typically 10-30 seconds. Rules on wrong clusters briefly is benign.
		o.Logger.V(1).Info("No PlacementDecisions found, defaulting to selected",
			"placement", placementName, "cluster", clusterName)
		return true, nil
	}

	for _, pd := range placementDecisionList.Items {
		for _, decision := range pd.Status.Decisions {
			if decision.ClusterName == clusterName {
				return true, nil
			}
		}
	}

	o.Logger.V(1).Info("Cluster not selected by placement",
		"placement", placementName, "cluster", clusterName)
	return false, nil
}
