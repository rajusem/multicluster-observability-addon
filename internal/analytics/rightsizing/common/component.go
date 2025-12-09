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
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	policyv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
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
func CleanupComponentResources(
	ctx context.Context,
	c client.Client,
	componentConfig ComponentConfig,
	namespace string,
	configNamespace string,
	bindingUpdated bool,
) {
	log.Info("rs - cleaning up resources",
		"component", componentConfig.ComponentType,
		"bindingNamespace", namespace,
		"configNamespace", configNamespace,
		"bindingUpdated", bindingUpdated)

	var resourcesToDelete []client.Object
	commonResources := []client.Object{
		&policyv1.PlacementBinding{ObjectMeta: metav1.ObjectMeta{Name: componentConfig.PlacementBindingName, Namespace: namespace}},
		&clusterv1beta1.Placement{ObjectMeta: metav1.ObjectMeta{Name: componentConfig.PlacementName, Namespace: namespace}},
		&policyv1.Policy{ObjectMeta: metav1.ObjectMeta{Name: componentConfig.PrometheusRulePolicyName, Namespace: namespace}},
	}

	if bindingUpdated {
		// If NamespaceBinding has been updated delete only common resources
		resourcesToDelete = commonResources
		log.V(1).Info("rs - bindingUpdated=true, ConfigMap will NOT be deleted", "component", componentConfig.ComponentType)
	} else {
		resourcesToDelete = append(commonResources,
			&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: componentConfig.ConfigMapName, Namespace: configNamespace}},
		)
		log.Info("rs - bindingUpdated=false, ConfigMap will be deleted",
			"component", componentConfig.ComponentType,
			"configMapName", componentConfig.ConfigMapName,
			"configMapNamespace", configNamespace)
	}

	// Delete related resources
	for _, resource := range resourcesToDelete {
		err := c.Delete(ctx, resource)
		if err != nil {
			if errors.IsNotFound(err) {
				log.V(1).Info("rs - resource not found, skipping delete",
					"name", resource.GetName(),
					"namespace", resource.GetNamespace(),
					"component", componentConfig.ComponentType)
			} else {
				log.Error(err, "rs - failed to delete resource",
					"name", resource.GetName(),
					"namespace", resource.GetNamespace(),
					"component", componentConfig.ComponentType)
			}
		} else {
			log.Info("rs - successfully deleted resource",
				"name", resource.GetName(),
				"namespace", resource.GetNamespace(),
				"component", componentConfig.ComponentType)
		}
	}
	log.Info("rs - cleanup completed", "component", componentConfig.ComponentType)
}
