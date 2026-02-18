// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package rightsizing

import (
	"time"

	"github.com/perses/community-mixins/pkg/dashboards"
	"github.com/perses/community-mixins/pkg/promql"
	"github.com/perses/perses/go-sdk/dashboard"
	panelgroup "github.com/perses/perses/go-sdk/panel-group"
	listVar "github.com/perses/perses/go-sdk/variable/list-variable"
	labelValuesVar "github.com/perses/plugins/prometheus/sdk/go/variable/label-values"
	staticListVar "github.com/perses/plugins/staticlistvariable/sdk/go"
	panels "github.com/stolostron/multicluster-observability-addon/internal/perses/panels/rightsizing"
)

func withCPUStatsGroup(datasource string) dashboard.Option {
	return dashboard.AddPanelGroup("CPU Stats",
		panelgroup.PanelsPerLine(4),
		panelgroup.PanelHeight(5),
		panels.CPURecommendationPanel(datasource),
		panels.CPUUsagePanel(datasource),
		panels.CPURequestPanel(datasource),
		panels.CPUUtilizationPanel(datasource),
	)
}

func withMemStatsGroup(datasource string) dashboard.Option {
	return dashboard.AddPanelGroup("Memory Stats",
		panelgroup.PanelsPerLine(4),
		panelgroup.PanelHeight(5),
		panels.MemRecommendationPanel(datasource),
		panels.MemUsagePanel(datasource),
		panels.MemRequestPanel(datasource),
		panels.MemUtilizationPanel(datasource),
	)
}

func withTopNamespacesGroup(datasource string) dashboard.Option {
	return dashboard.AddPanelGroup("Top Namespaces",
		panelgroup.PanelsPerLine(2),
		panels.CPUTopNamespacesPanel(datasource),
		panels.MemTopNamespacesPanel(datasource),
	)
}

func withQuotaTablesGroup(datasource string) dashboard.Option {
	return dashboard.AddPanelGroup("Quota Tables",
		panelgroup.PanelsPerLine(2),
		panels.CPUQuotaTablePanel(datasource),
		panels.MemQuotaTablePanel(datasource),
	)
}

// BuildNamespaceRightSizing creates the namespace right-sizing dashboard
func BuildNamespaceRightSizing(project string, datasource string, clusterLabelName string) (dashboard.Builder, error) {
	return dashboard.New("acm-rs-namespace-overview",
		dashboard.ProjectName(project),
		dashboard.Name("ACM Right-Sizing Namespace"),
		dashboard.Duration(time.Hour*24*7), // Default to 7 days

		// Cluster variable
		dashboard.AddVariable("cluster",
			listVar.List(
				labelValuesVar.PrometheusLabelValues("cluster",
					dashboards.AddVariableDatasource(datasource),
					labelValuesVar.Matchers(
						promql.SetLabelMatchers(
							"acm_rs:cluster:cpu_request",
							[]promql.LabelMatcher{},
						)),
				),
				listVar.DisplayName("Cluster"),
				listVar.DefaultValue("local-cluster"),
				listVar.AllowAllValue(false),
				listVar.AllowMultiple(false),
			),
		),

		// Profile variable
		dashboard.AddVariable("profile",
			listVar.List(
				labelValuesVar.PrometheusLabelValues("profile",
					dashboards.AddVariableDatasource(datasource),
					labelValuesVar.Matchers(
						promql.SetLabelMatchers(
							`acm_rs:namespace:cpu_usage{cluster="$cluster"}`,
							[]promql.LabelMatcher{},
						)),
				),
				listVar.DisplayName("Profile"),
				listVar.DefaultValue("Max OverAll"),
				listVar.AllowAllValue(false),
				listVar.AllowMultiple(false),
			),
		),

		// Days variable
		dashboard.AddVariable("days",
			listVar.List(
				staticListVar.StaticList(
					staticListVar.Values("1d", "2d", "5d", "10d", "30d", "60d", "90d"),
				),
				listVar.DisplayName("Days"),
				listVar.DefaultValue("10d"),
				listVar.AllowAllValue(false),
				listVar.AllowMultiple(false),
			),
		),

		withCPUStatsGroup(datasource),
		withMemStatsGroup(datasource),
		withTopNamespacesGroup(datasource),
		withQuotaTablesGroup(datasource),
	)
}
