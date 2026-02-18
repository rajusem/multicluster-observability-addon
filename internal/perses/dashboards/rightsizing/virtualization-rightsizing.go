// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package rightsizing

import (
	"github.com/perses/community-mixins/pkg/dashboards"
	"github.com/perses/community-mixins/pkg/promql"
	"github.com/perses/perses/go-sdk/dashboard"
	panelgroup "github.com/perses/perses/go-sdk/panel-group"
	listVar "github.com/perses/perses/go-sdk/variable/list-variable"
	labelValuesVar "github.com/perses/plugins/prometheus/sdk/go/variable/label-values"
	panels "github.com/stolostron/multicluster-observability-addon/internal/perses/panels/rightsizing"
)

func withVMOverviewGroup(datasource string, labelMatcher promql.LabelMatcher) dashboard.Option {
	return dashboard.AddPanelGroup("VM Overview",
		panelgroup.PanelsPerLine(3),
		panels.VMsAnalyzedPanel(datasource, labelMatcher),
		panels.VMCPUOverProvisionedPanel(datasource, labelMatcher),
		panels.VMMemoryOverProvisionedPanel(datasource, labelMatcher),
	)
}

func withVMCPURightSizingGroup(datasource string, labelMatcher promql.LabelMatcher) dashboard.Option {
	return dashboard.AddPanelGroup("VM CPU Right-Sizing",
		panelgroup.PanelsPerLine(1),
		panels.VMCPURequestVsUsagePanel(datasource, labelMatcher),
	)
}

func withVMMemoryRightSizingGroup(datasource string, labelMatcher promql.LabelMatcher) dashboard.Option {
	return dashboard.AddPanelGroup("VM Memory Right-Sizing",
		panelgroup.PanelsPerLine(1),
		panels.VMMemoryRequestVsUsagePanel(datasource, labelMatcher),
	)
}

func withVMRecommendationsGroup(datasource string, labelMatcher promql.LabelMatcher) dashboard.Option {
	return dashboard.AddPanelGroup("VM Recommendations",
		panelgroup.PanelsPerLine(1),
		panels.VMRecommendationsTablePanel(datasource, labelMatcher),
	)
}

// BuildVirtualizationRightSizing creates the virtualization right-sizing dashboard
func BuildVirtualizationRightSizing(project string, datasource string, clusterLabelName string) (dashboard.Builder, error) {
	clusterLabelMatcher := dashboards.GetClusterLabelMatcher(clusterLabelName)
	return dashboard.New("acm-rs-virtualization-overview",
		dashboard.ProjectName(project),
		dashboard.Name("ACM Virtualization Right-Sizing"),

		dashboard.AddVariable("cluster",
			listVar.List(
				labelValuesVar.PrometheusLabelValues("cluster",
					dashboards.AddVariableDatasource(datasource),
					labelValuesVar.Matchers(
						promql.SetLabelMatchers(
							"acm_rs_vm:namespace:cpu_request",
							[]promql.LabelMatcher{},
						)),
				),
				listVar.DisplayName("Cluster"),
				listVar.AllowAllValue(false),
				listVar.AllowMultiple(false),
			),
		),

		withVMOverviewGroup(datasource, clusterLabelMatcher),
		withVMCPURightSizingGroup(datasource, clusterLabelMatcher),
		withVMMemoryRightSizingGroup(datasource, clusterLabelMatcher),
		withVMRecommendationsGroup(datasource, clusterLabelMatcher),
	)
}
