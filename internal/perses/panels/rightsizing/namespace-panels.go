// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package rightsizing

import (
	"github.com/perses/community-mixins/pkg/dashboards"
	commonSdk "github.com/perses/perses/go-sdk/common"
	"github.com/perses/perses/go-sdk/panel"
	panelgroup "github.com/perses/perses/go-sdk/panel-group"
	"github.com/perses/plugins/prometheus/sdk/go/query"
	tablePanel "github.com/perses/plugins/table/sdk/go"
	timeSeriesPanel "github.com/perses/plugins/timeserieschart/sdk/go"
)

// CPURecommendationPanel shows the CPU recommendation for the cluster
func CPURecommendationPanel(datasourceName string) panelgroup.Option {
	return BuildStatPanel(datasourceName, StatPanelConfig{
		Title:       "CPU Recommendation",
		Description: "CPU recommendation for the selected cluster",
		Query:       `max_over_time(sum by (cluster)(acm_rs:cluster:cpu_recommendation{cluster="$cluster", profile="$profile"})[$days:])`,
		Unit:        &dashboards.DecimalUnit,
		Decimals:    2,
		FontSize:    48,
	})
}

// CPUUsagePanel shows the CPU usage for the cluster
func CPUUsagePanel(datasourceName string) panelgroup.Option {
	return BuildStatPanel(datasourceName, StatPanelConfig{
		Title:       "CPU Usage",
		Description: "CPU usage for the selected cluster",
		Query:       `max_over_time(sum by (cluster)(acm_rs:cluster:cpu_usage{cluster="$cluster", profile="$profile"})[$days:])`,
		Unit:        &dashboards.DecimalUnit,
		Decimals:    2,
		FontSize:    48,
	})
}

// CPURequestPanel shows the CPU request for the cluster
func CPURequestPanel(datasourceName string) panelgroup.Option {
	return BuildStatPanel(datasourceName, StatPanelConfig{
		Title:       "CPU Request",
		Description: "CPU request for the selected cluster",
		Query:       `max_over_time(sum by (cluster)(acm_rs:cluster:cpu_request{cluster="$cluster", profile="$profile"})[$days:])`,
		Unit:        &dashboards.DecimalUnit,
		Decimals:    2,
		FontSize:    48,
	})
}

// CPUUtilizationPanel shows the CPU utilization percentage for the cluster
func CPUUtilizationPanel(datasourceName string) panelgroup.Option {
	return BuildStatPanel(datasourceName, StatPanelConfig{
		Title:       "CPU Utilization",
		Description: "CPU utilization percentage for the selected cluster",
		Query:       `max_over_time(sum by (cluster)(acm_rs:cluster:cpu_usage{cluster="$cluster", profile="$profile"})[$days:]) / max_over_time(sum by (cluster)(acm_rs:cluster:cpu_request{cluster="$cluster", profile="$profile"})[$days:])`,
		Unit:        &dashboards.PercentDecimalUnit,
		Decimals:    1,
		FontSize:    48,
	})
}

// MemRecommendationPanel shows the memory recommendation for the cluster
func MemRecommendationPanel(datasourceName string) panelgroup.Option {
	return BuildStatPanel(datasourceName, StatPanelConfig{
		Title:       "Mem Recommendation",
		Description: "Memory recommendation for the selected cluster",
		Query:       `max_over_time(sum by (cluster)(acm_rs:cluster:memory_recommendation{cluster="$cluster", profile="$profile"})[$days:])`,
		Unit:        &dashboards.BytesUnit,
		Decimals:    1,
		FontSize:    40,
	})
}

// MemUsagePanel shows the memory usage for the cluster
func MemUsagePanel(datasourceName string) panelgroup.Option {
	return BuildStatPanel(datasourceName, StatPanelConfig{
		Title:       "Mem Usage",
		Description: "Memory usage for the selected cluster",
		Query:       `max_over_time(sum by (cluster)(acm_rs:cluster:memory_usage{cluster="$cluster", profile="$profile"})[$days:])`,
		Unit:        &dashboards.BytesUnit,
		Decimals:    1,
		FontSize:    40,
	})
}

// MemRequestPanel shows the memory request for the cluster
func MemRequestPanel(datasourceName string) panelgroup.Option {
	return BuildStatPanel(datasourceName, StatPanelConfig{
		Title:       "Mem Request",
		Description: "Memory request for the selected cluster",
		Query:       `max_over_time(sum by (cluster)(acm_rs:cluster:memory_request{cluster="$cluster", profile="$profile"})[$days:])`,
		Unit:        &dashboards.BytesUnit,
		Decimals:    1,
		FontSize:    40,
	})
}

// MemUtilizationPanel shows the memory utilization percentage for the cluster
func MemUtilizationPanel(datasourceName string) panelgroup.Option {
	return BuildStatPanel(datasourceName, StatPanelConfig{
		Title:       "Mem Utilization",
		Description: "Memory utilization percentage for the selected cluster",
		Query:       `max_over_time(sum by (cluster)(acm_rs:cluster:memory_usage{cluster="$cluster", profile="$profile"})[$days:]) / max_over_time(sum by (cluster)(acm_rs:cluster:memory_request{cluster="$cluster", profile="$profile"})[$days:])`,
		Unit:        &dashboards.PercentDecimalUnit,
		Decimals:    1,
		FontSize:    40,
	})
}

// CPUTopNamespacesPanel shows CPU utilization of top namespaces
func CPUTopNamespacesPanel(datasourceName string) panelgroup.Option {
	return panelgroup.AddPanel("CPU Utilization of Top Namespaces",
		panel.Description("CPU utilization of the top 20 namespaces by usage/request ratio"),
		timeSeriesPanel.Chart(
			timeSeriesPanel.WithYAxis(timeSeriesPanel.YAxis{
				Format: &commonSdk.Format{
					Unit: &dashboards.PercentDecimalUnit,
				},
			}),
			timeSeriesPanel.WithLegend(timeSeriesPanel.Legend{
				Position: timeSeriesPanel.BottomPosition,
				Mode:     timeSeriesPanel.ListMode,
			}),
			timeSeriesPanel.WithVisual(timeSeriesPanel.Visual{
				Display:      timeSeriesPanel.LineDisplay,
				ConnectNulls: true,
				LineWidth:    1.25,
				AreaOpacity:  0.3,
				PointRadius:  2.75,
			}),
		),
		panel.AddQuery(
			query.PromQL(
				`topk(20, sum by (namespace) (acm_rs:namespace:cpu_usage{cluster="$cluster", profile="$profile"}) / sum by (namespace) (acm_rs:namespace:cpu_request{cluster="$cluster", profile="$profile"}))`,
				dashboards.AddQueryDataSource(datasourceName),
				query.SeriesNameFormat("{{namespace}}"),
			),
		),
	)
}

// MemTopNamespacesPanel shows memory utilization of top namespaces
func MemTopNamespacesPanel(datasourceName string) panelgroup.Option {
	return panelgroup.AddPanel("Memory Utilization of Top Namespaces",
		panel.Description("Memory utilization of the top 20 namespaces by usage/request ratio"),
		timeSeriesPanel.Chart(
			timeSeriesPanel.WithYAxis(timeSeriesPanel.YAxis{
				Format: &commonSdk.Format{
					Unit: &dashboards.PercentDecimalUnit,
				},
			}),
			timeSeriesPanel.WithLegend(timeSeriesPanel.Legend{
				Position: timeSeriesPanel.BottomPosition,
				Mode:     timeSeriesPanel.ListMode,
			}),
			timeSeriesPanel.WithVisual(timeSeriesPanel.Visual{
				Display:      timeSeriesPanel.LineDisplay,
				ConnectNulls: true,
				LineWidth:    1.25,
				AreaOpacity:  0.2,
				PointRadius:  2.75,
			}),
		),
		panel.AddQuery(
			query.PromQL(
				`topk(20, sum by (namespace) (acm_rs:namespace:memory_usage{cluster="$cluster", profile="$profile"}) / sum by (namespace) (acm_rs:namespace:memory_request{cluster="$cluster", profile="$profile"}))`,
				dashboards.AddQueryDataSource(datasourceName),
				query.SeriesNameFormat("{{namespace}}"),
			),
		),
	)
}

// CPUQuotaTablePanel shows CPU quota details per namespace in a table
func CPUQuotaTablePanel(datasourceName string) panelgroup.Option {
	return panelgroup.AddPanel("CPU Quota Table",
		panel.Description("CPU utilization, usage, request, and recommendation per namespace"),
		tablePanel.Table(
			tablePanel.WithColumnSettings([]tablePanel.ColumnSettings{
				{
					Name:   "timestamp",
					Header: "Timestamp",
					Hide:   true,
				},
				{
					Name:   "namespace",
					Header: "Namespace",
					Align:  tablePanel.LeftAlign,
				},
				{
					Name:   "value #1",
					Header: "CPU Utilization %",
					Align:  tablePanel.RightAlign,
					Format: &commonSdk.Format{
						Unit:          &dashboards.PercentDecimalUnit,
						DecimalPlaces: 2,
					},
				},
				{
					Name:   "value #2",
					Header: "CPU Usage",
					Align:  tablePanel.RightAlign,
					Format: &commonSdk.Format{
						Unit:          &dashboards.DecimalUnit,
						DecimalPlaces: 3,
					},
				},
				{
					Name:   "value #3",
					Header: "CPU Request",
					Align:  tablePanel.RightAlign,
					Format: &commonSdk.Format{
						Unit:          &dashboards.DecimalUnit,
						DecimalPlaces: 3,
					},
				},
				{
					Name:   "value #4",
					Header: "CPU Recommendation",
					Align:  tablePanel.RightAlign,
					Format: &commonSdk.Format{
						Unit:          &dashboards.DecimalUnit,
						DecimalPlaces: 2,
					},
				},
			}),
			tablePanel.Transform([]commonSdk.Transform{
				{
					Kind: commonSdk.MergeSeriesKind,
					Spec: commonSdk.MergeSeriesSpec{},
				},
			}),
		),
		panel.AddQuery(
			query.PromQL(
				`max_over_time(sum by (namespace) (acm_rs:namespace:cpu_usage{cluster="$cluster", profile="$profile"})[$days:]) / max_over_time(sum by (namespace) (acm_rs:namespace:cpu_request{cluster="$cluster", profile="$profile"})[$days:])`,
				dashboards.AddQueryDataSource(datasourceName),
			),
		),
		panel.AddQuery(
			query.PromQL(
				`max_over_time(sum by (namespace) (acm_rs:namespace:cpu_usage{cluster="$cluster", profile="$profile"})[$days:])`,
				dashboards.AddQueryDataSource(datasourceName),
			),
		),
		panel.AddQuery(
			query.PromQL(
				`max_over_time(sum by (namespace) (acm_rs:namespace:cpu_request{cluster="$cluster", profile="$profile"})[$days:])`,
				dashboards.AddQueryDataSource(datasourceName),
			),
		),
		panel.AddQuery(
			query.PromQL(
				`max_over_time(sum by (namespace) (acm_rs:namespace:cpu_recommendation{cluster="$cluster", profile="$profile"})[$days:])`,
				dashboards.AddQueryDataSource(datasourceName),
			),
		),
	)
}

// MemQuotaTablePanel shows memory quota details per namespace in a table
func MemQuotaTablePanel(datasourceName string) panelgroup.Option {
	return panelgroup.AddPanel("Memory Quota Table",
		panel.Description("Memory utilization, usage, request, and recommendation per namespace"),
		tablePanel.Table(
			tablePanel.WithColumnSettings([]tablePanel.ColumnSettings{
				{
					Name:   "timestamp",
					Header: "Timestamp",
					Hide:   true,
				},
				{
					Name:   "namespace",
					Header: "Namespace",
					Align:  tablePanel.LeftAlign,
				},
				{
					Name:   "value #1",
					Header: "Memory Utilization %",
					Align:  tablePanel.RightAlign,
					Format: &commonSdk.Format{
						Unit:          &dashboards.PercentDecimalUnit,
						DecimalPlaces: 2,
					},
				},
				{
					Name:   "value #2",
					Header: "Memory Usage",
					Align:  tablePanel.RightAlign,
					Format: &commonSdk.Format{
						Unit: &dashboards.BytesUnit,
					},
				},
				{
					Name:   "value #3",
					Header: "Memory Request",
					Align:  tablePanel.RightAlign,
					Format: &commonSdk.Format{
						Unit: &dashboards.BytesUnit,
					},
				},
				{
					Name:   "value #4",
					Header: "Memory Recommendation",
					Align:  tablePanel.RightAlign,
					Format: &commonSdk.Format{
						Unit: &dashboards.BytesUnit,
					},
				},
			}),
			tablePanel.Transform([]commonSdk.Transform{
				{
					Kind: commonSdk.MergeSeriesKind,
					Spec: commonSdk.MergeSeriesSpec{},
				},
			}),
		),
		panel.AddQuery(
			query.PromQL(
				`max_over_time(sum by (namespace) (acm_rs:namespace:memory_usage{cluster="$cluster", profile="$profile"})[$days:]) / max_over_time(sum by (namespace) (acm_rs:namespace:memory_request{cluster="$cluster", profile="$profile"})[$days:])`,
				dashboards.AddQueryDataSource(datasourceName),
			),
		),
		panel.AddQuery(
			query.PromQL(
				`max_over_time(sum by (namespace) (acm_rs:namespace:memory_usage{cluster="$cluster", profile="$profile"})[$days:])`,
				dashboards.AddQueryDataSource(datasourceName),
			),
		),
		panel.AddQuery(
			query.PromQL(
				`max_over_time(sum by (namespace) (acm_rs:namespace:memory_request{cluster="$cluster", profile="$profile"})[$days:])`,
				dashboards.AddQueryDataSource(datasourceName),
			),
		),
		panel.AddQuery(
			query.PromQL(
				`max_over_time(sum by (namespace) (acm_rs:namespace:memory_recommendation{cluster="$cluster", profile="$profile"})[$days:])`,
				dashboards.AddQueryDataSource(datasourceName),
			),
		),
	)
}
