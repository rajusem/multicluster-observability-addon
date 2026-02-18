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
	statPanel "github.com/perses/plugins/statchart/sdk/go"
)

// StatPanelConfig defines configuration for building a stat panel
type StatPanelConfig struct {
	Title       string
	Description string
	Query       string
	Unit        *string
	Decimals    int
	FontSize    int
}

// BuildStatPanel creates a stat panel with the given configuration.
// This provides a reusable way to create consistent stat panels across dashboards.
func BuildStatPanel(datasourceName string, cfg StatPanelConfig) panelgroup.Option {
	return panelgroup.AddPanel(cfg.Title,
		panel.Description(cfg.Description),
		statPanel.Chart(
			statPanel.Format(commonSdk.Format{
				Unit:          cfg.Unit,
				DecimalPlaces: cfg.Decimals,
			}),
			statPanel.ValueFontSize(cfg.FontSize),
		),
		panel.AddQuery(
			query.PromQL(cfg.Query, dashboards.AddQueryDataSource(datasourceName)),
		),
	)
}
