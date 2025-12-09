// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package virtualization

import (
	"context"

	"github.com/stolostron/multicluster-observability-addon/internal/analytics/rightsizing/common"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateVirtualizationPlacementBinding creates the PlacementBinding resource for virtualization
func CreateVirtualizationPlacementBinding(ctx context.Context, c client.Client, namespace string) error {
	return common.CreateRSPlacementBinding(ctx, c, PlacementBindingName, namespace, PlacementName, PrometheusRulePolicyName)
}
