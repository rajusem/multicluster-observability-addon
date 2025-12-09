// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

// Package rightsizing provides resource right-sizing recommendation functionality
// for both namespace-level and virtualization workloads.
//
// The package is organized into the following sub-packages:
//   - common: Shared types, utilities, and functions used across right-sizing components
//   - namespace: Namespace-level right-sizing logic and PrometheusRule generation
//   - virtualization: Virtualization workload right-sizing logic and PrometheusRule generation
//   - handlers: Main entry point handlers for right-sizing operations
//   - manifests: Helm chart values and configuration for right-sizing
//   - dashboards: Grafana dashboard definitions for right-sizing metrics visualization
//
// Usage:
//
//	import (
//	    "github.com/stolostron/multicluster-observability-addon/internal/analytics/rightsizing/common"
//	    "github.com/stolostron/multicluster-observability-addon/internal/analytics/rightsizing/handlers"
//	)
//
//	// Create options
//	opts := common.RightSizingOptions{
//	    NamespaceEnabled:      true,
//	    VirtualizationEnabled: true,
//	    ConfigNamespace:       "open-cluster-management-observability",
//	}
//
//	// Handle right-sizing
//	err := handlers.HandleRightSizing(ctx, client, opts)
package rightsizing
