// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package virtualization

import (
	"context"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stolostron/multicluster-observability-addon/internal/analytics/rightsizing/common"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateOrUpdateVirtualizationPrometheusRulePolicy creates or updates the PrometheusRule policy for virtualization
func CreateOrUpdateVirtualizationPrometheusRulePolicy(
	ctx context.Context,
	c client.Client,
	prometheusRule monitoringv1.PrometheusRule,
	namespace string,
) error {
	return common.CreateOrUpdateRSPrometheusRulePolicy(ctx, c, PrometheusRulePolicyName, namespace, prometheusRule)
}
