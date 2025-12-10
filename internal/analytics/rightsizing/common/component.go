// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package common

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HandleComponentRightSizing handles the right-sizing functionality for any component type
func HandleComponentRightSizing(
	ctx context.Context,
	c client.Client,
	opts RightSizingOptions,
	componentConfig ComponentConfig,
	state *ComponentState,
) error {
	log.V(1).Info("rs - handling right-sizing", "component", componentConfig.ComponentType)

	// Get right-sizing configuration based on component type
	var isEnabled bool
	var newBinding string

	switch componentConfig.ComponentType {
	case ComponentTypeNamespace:
		isEnabled = opts.NamespaceEnabled
		newBinding = opts.NamespaceBinding
	case ComponentTypeVirtualization:
		isEnabled = opts.VirtualizationEnabled
		newBinding = opts.VirtualizationBinding
	default:
		return fmt.Errorf("unknown component type: %s", componentConfig.ComponentType)
	}

	// Set to default namespace if not given
	if newBinding == "" {
		newBinding = componentConfig.DefaultNamespace
	}

	// Check if right-sizing feature enabled or not
	// If disabled then cleanup related resources
	if !isEnabled {
		log.Info("rs - feature disabled, initiating cleanup",
			"component", componentConfig.ComponentType,
			"stateNamespace", state.Namespace,
			"configNamespace", opts.ConfigNamespace)
		CleanupComponentResources(ctx, c, componentConfig, state.Namespace, opts.ConfigNamespace, false)
		state.Namespace = newBinding
		state.Enabled = false
		return nil
	}

	// Check if this is first time enabling or if namespace binding has changed
	isFirstEnable := !state.Enabled
	namespaceBindingUpdated := state.Namespace != newBinding && state.Enabled

	// Set enabled flag which will be used for checking namespaceBindingUpdated condition next time
	state.Enabled = true

	// Retrieve the existing namespace and update the new namespace
	existingNamespace := state.Namespace
	state.Namespace = newBinding

	// Creating configmap with default values
	if err := EnsureRSConfigMapExists(ctx, c, componentConfig.ConfigMapName, opts.ConfigNamespace, componentConfig.GetDefaultConfigFunc); err != nil {
		return err
	}

	// Clean up old resources if namespace binding changed
	if namespaceBindingUpdated {
		// Clean up resources except config map to update NamespaceBinding
		CleanupComponentResources(ctx, c, componentConfig, existingNamespace, opts.ConfigNamespace, true)
	}

	// Get configmap
	cm := &corev1.ConfigMap{}
	if err := c.Get(ctx, client.ObjectKey{Name: componentConfig.ConfigMapName, Namespace: opts.ConfigNamespace}, cm); err != nil {
		return fmt.Errorf("rs - failed to get existing configmap: %w", err)
	}

	// Get configmap data into specified structure
	configData, err := GetRSConfigData(cm)
	if err != nil {
		return fmt.Errorf("rs - failed to extract config data: %w", err)
	}

	// Apply the Policy, Placement, PlacementBinding
	// Always apply to ensure ConfigMap changes are reflected
	if err := componentConfig.ApplyChangesFunc(configData); err != nil {
		return fmt.Errorf("rs - failed to apply configmap changes: %w", err)
	}

	if isFirstEnable {
		log.Info("rs - first enable, applied initial configuration", "component", componentConfig.ComponentType)
	} else if namespaceBindingUpdated {
		log.Info("rs - namespace binding updated, re-applied configuration", "component", componentConfig.ComponentType)
	}

	log.Info("rs - create component task completed", "component", componentConfig.ComponentType)
	return nil
}

// CleanupComponentResources cleans up the resources created for any component type
// This includes ClusterManagementAddOn, AddOnTemplate, Placement, and optionally ConfigMap
func CleanupComponentResources(
	ctx context.Context,
	c client.Client,
	componentConfig ComponentConfig,
	namespace string,
	configNamespace string,
	bindingUpdated bool,
) {
	log.Info("rs - cleaning up addon resources",
		"component", componentConfig.ComponentType,
		"placementNamespace", namespace,
		"configNamespace", configNamespace,
		"bindingUpdated", bindingUpdated)

	// Clean up addon resources (ClusterManagementAddOn, AddOnTemplate, Placement)
	CleanupRightSizingAddon(ctx, c, componentConfig.AddonName, componentConfig.TemplateName, componentConfig.PlacementName, namespace)

	// If not just a binding update, also delete the ConfigMap
	if !bindingUpdated {
		log.Info("rs - bindingUpdated=false, ConfigMap will be deleted",
			"component", componentConfig.ComponentType,
			"configMapName", componentConfig.ConfigMapName,
			"configMapNamespace", configNamespace)

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      componentConfig.ConfigMapName,
				Namespace: configNamespace,
			},
		}
		if err := c.Delete(ctx, cm); err != nil {
			if errors.IsNotFound(err) {
				log.V(1).Info("rs - ConfigMap not found, skipping delete",
					"name", componentConfig.ConfigMapName,
					"namespace", configNamespace)
			} else {
				log.Error(err, "rs - failed to delete ConfigMap",
					"name", componentConfig.ConfigMapName,
					"namespace", configNamespace)
			}
		} else {
			log.Info("rs - successfully deleted ConfigMap",
				"name", componentConfig.ConfigMapName,
				"namespace", configNamespace)
		}
	} else {
		log.V(1).Info("rs - bindingUpdated=true, ConfigMap will NOT be deleted", "component", componentConfig.ComponentType)
	}

	log.Info("rs - cleanup completed", "component", componentConfig.ComponentType)
}

// CleanupAddonResourcesOnly cleans up only the addon resources without touching ConfigMap
// This is useful for cleanup during binding namespace changes
func CleanupAddonResourcesOnly(
	ctx context.Context,
	c client.Client,
	addonName, templateName, placementName, placementNamespace string,
) {
	// Delete ClusterManagementAddOn (cluster-scoped)
	cmao := &addonv1alpha1.ClusterManagementAddOn{
		ObjectMeta: metav1.ObjectMeta{
			Name: addonName,
		},
	}
	deleteResource(ctx, c, cmao, "ClusterManagementAddOn")

	// Delete AddOnTemplate (cluster-scoped)
	template := &addonv1alpha1.AddOnTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: templateName,
		},
	}
	deleteResource(ctx, c, template, "AddOnTemplate")

	// Delete Placement (namespaced)
	placement := &clusterv1beta1.Placement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      placementName,
			Namespace: placementNamespace,
		},
	}
	deleteResource(ctx, c, placement, "Placement")
}

// deleteResource is a helper to delete a resource with proper logging
func deleteResource(ctx context.Context, c client.Client, obj client.Object, resourceType string) {
	if err := c.Delete(ctx, obj); err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("rs - resource not found, skipping delete",
				"type", resourceType,
				"name", obj.GetName(),
				"namespace", obj.GetNamespace())
		} else {
			log.Error(err, "rs - failed to delete resource",
				"type", resourceType,
				"name", obj.GetName(),
				"namespace", obj.GetNamespace())
		}
	} else {
		log.Info("rs - successfully deleted resource",
			"type", resourceType,
			"name", obj.GetName(),
			"namespace", obj.GetNamespace())
	}
}
