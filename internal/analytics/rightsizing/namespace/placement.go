// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package namespace

import (
	"context"

	"github.com/stolostron/multicluster-observability-addon/internal/analytics/rightsizing/common"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateUpdatePlacement creates the Placement resource
func CreateUpdatePlacement(ctx context.Context, c client.Client, placementConfig clusterv1beta1.Placement, namespace string) error {
	return common.CreateUpdateRSPlacement(ctx, c, PlacementName, namespace, placementConfig)
}
