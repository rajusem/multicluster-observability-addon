// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package handlers

import (
	"context"
	"fmt"

	"github.com/stolostron/multicluster-observability-addon/internal/analytics/rightsizing/common"
	"github.com/stolostron/multicluster-observability-addon/internal/analytics/rightsizing/namespace"
	"github.com/stolostron/multicluster-observability-addon/internal/analytics/rightsizing/virtualization"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var (
	log = logf.Log.WithName("rightsizing")
)

// HandleRightSizing handles both namespace and virtualization right-sizing functionality
func HandleRightSizing(
	ctx context.Context,
	c client.Client,
	opts common.RightSizingOptions,
) error {
	log.V(1).Info("rs - inside handle right-sizing")

	// Handle namespace right-sizing
	if err := namespace.HandleRightSizing(ctx, c, opts); err != nil {
		return fmt.Errorf("failed to handle namespace right-sizing: %w", err)
	}

	// Handle virtualization right-sizing
	if err := virtualization.HandleRightSizing(ctx, c, opts); err != nil {
		return fmt.Errorf("failed to handle virtualization right-sizing: %w", err)
	}

	log.Info("rs - right-sizing handling completed")
	return nil
}

// GetNamespaceRSConfigMapPredicateFunc returns predicate for namespace right-sizing ConfigMap
func GetNamespaceRSConfigMapPredicateFunc(ctx context.Context, c client.Client, configNamespace string) predicate.Funcs {
	return namespace.GetNamespaceRSConfigMapPredicateFunc(ctx, c, configNamespace)
}

// GetVirtualizationRSConfigMapPredicateFunc returns predicate for virtualization right-sizing ConfigMap
func GetVirtualizationRSConfigMapPredicateFunc(ctx context.Context, c client.Client, configNamespace string) predicate.Funcs {
	return virtualization.GetVirtualizationRSConfigMapPredicateFunc(ctx, c, configNamespace)
}

// CleanupAllRightSizingResources cleans up all right-sizing resources
func CleanupAllRightSizingResources(ctx context.Context, c client.Client, configNamespace string) {
	log.V(1).Info("rs - cleaning up all right-sizing resources")

	// Clean up namespace right-sizing resources
	namespace.CleanupRSNamespaceResources(ctx, c, namespace.ComponentState.Namespace, configNamespace, false)

	// Clean up virtualization right-sizing resources
	virtualization.CleanupRSVirtualizationResources(ctx, c, virtualization.ComponentState.Namespace, configNamespace, false)

	log.Info("rs - all right-sizing resources cleaned up")
}

// IsRightSizingEnabled checks if any right-sizing feature is enabled
func IsRightSizingEnabled(opts common.RightSizingOptions) bool {
	return opts.NamespaceEnabled || opts.VirtualizationEnabled
}
