// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package rightsizing

import (
	"encoding/json"
	"testing"

	"github.com/prometheus/prometheus/promql/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildPodRightSizing verifies the pod overview dashboard queries and drill-down link.
func TestBuildPodRightSizing(t *testing.T) {
	db, err := BuildPodRightSizing(testProject, testDatasource, testClusterLbl)
	require.NoError(t, err)
	assert.Equal(t, "acm-rs-pod-overview", db.Dashboard.Metadata.Name)
	raw, err := json.Marshal(db.Dashboard.Spec)
	require.NoError(t, err)
	specStr := string(raw)
	assert.Contains(t, specStr, "acm_rs:pod:cpu_usage")
	assert.Contains(t, specStr, "acm_rs:pod:memory_usage")
	assert.Contains(t, specStr, "acm-rs-pod-detail")
	assert.NotContains(t, specStr, "acm_rs:workload:", "pod overview must not read workload-level series")
	assert.NotContains(t, specStr, `"text":"N/A"`, "missing pod fields must stay blank")
}

// TestPodOverviewTablesKeepInventoryRowsWithoutResourceSamples checks pod tables keep inventory rows via acm_rs:pod:info.
func TestPodOverviewTablesKeepInventoryRowsWithoutResourceSamples(t *testing.T) {
	db, err := BuildPodRightSizing(testProject, testDatasource, testClusterLbl)
	require.NoError(t, err)
	require.Len(t, db.Dashboard.Spec.Panels, 2, "pod overview retains both resource tables")
	for name, p := range db.Dashboard.Spec.Panels {
		require.Equal(t, "Table", p.Spec.Plugin.Kind)
		t.Run(name, func(t *testing.T) {
			require.Len(t, p.Spec.Queries, 6)
			raw, err := json.Marshal(p.Spec.Queries[5].Spec.Plugin.Spec)
			require.NoError(t, err)
			assert.Contains(t, string(raw), "acm_rs:pod:info")
			assert.NotContains(t, string(raw), "$cpu_profile")
			assert.NotContains(t, string(raw), "$memory_profile")
			raw, err = json.Marshal(p.Spec.Plugin.Spec)
			require.NoError(t, err)
			assert.Contains(t, string(raw), `"name":"value #6","hide":true`)
			assert.NotContains(t, string(raw), `"text":"N/A"`)
		})
	}
}

// TestBuildPodDetail verifies the pod detail dashboard layout, variables, and panels.
func TestBuildPodDetail(t *testing.T) {
	db, err := BuildPodDetail(testProject, testDatasource, testClusterLbl)
	require.NoError(t, err)
	assert.Equal(t, "acm-rs-pod-detail", db.Dashboard.Metadata.Name)
	assert.Len(t, db.Dashboard.Spec.Layouts, 3, "back link, CPU group, Memory group")
	assert.Len(t, db.Dashboard.Spec.Panels, 13, "back link, ten stats, and two time-series charts")
	for name, p := range db.Dashboard.Spec.Panels {
		assert.NotEqual(t, "Table", p.Spec.Plugin.Kind, "pod detail panel %s must not be a table", name)
	}
	varNames := extractVarNames(db.Dashboard.Spec.Variables)
	assert.ElementsMatch(t, []string{"cluster", "namespace", "pod", "cpu_profile", "memory_profile", "days"}, varNames,
		"pod detail must have exactly these variables; workload and workload_type are unused")
	raw, err := json.Marshal(db.Dashboard.Spec)
	require.NoError(t, err)
	specStr := string(raw)
	assert.Contains(t, specStr, "acm_rs:pod:cpu_recommendation")
	assert.Contains(t, specStr, "acm_rs:pod:memory_recommendation")
	assert.Contains(t, specStr, "CPU Over Time - Pod")
	assert.Contains(t, specStr, "Memory Over Time - Pod")
	assert.NotContains(t, specStr, "acm_rs:workload:", "pod detail must not read workload-level series")

	// The cluster, namespace and pod selectors list values from the pod inventory
	// series, and each one is scoped by the selectors before it.
	type labelValuesPlugin struct {
		Kind string `json:"kind"`
		Spec struct {
			LabelName string   `json:"labelName"`
			Matchers  []string `json:"matchers"`
		} `json:"spec"`
	}
	var vars []struct {
		Spec struct {
			Name   string            `json:"name"`
			Plugin labelValuesPlugin `json:"plugin"`
		} `json:"spec"`
	}
	rawVars, err := json.Marshal(db.Dashboard.Spec.Variables)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(rawVars, &vars))
	plugins := make(map[string]labelValuesPlugin, len(vars))
	for _, v := range vars {
		plugins[v.Spec.Name] = v.Spec.Plugin
	}
	for _, tc := range []struct {
		name string
		want []string
	}{
		{"cluster", []string{`__name__="acm_rs:pod:info"`}},
		{"namespace", []string{`__name__="acm_rs:pod:info"`, `cluster="$cluster"`}},
		{"pod", []string{`__name__="acm_rs:pod:info"`, `cluster="$cluster"`, `namespace="$namespace"`}},
	} {
		t.Run(tc.name+" selector", func(t *testing.T) {
			plugin, ok := plugins[tc.name]
			require.True(t, ok, "missing %s variable", tc.name)
			assert.Equal(t, "PrometheusLabelValuesVariable", plugin.Kind)
			assert.Equal(t, tc.name, plugin.Spec.LabelName)
			require.Len(t, plugin.Spec.Matchers, 1)
			matchers, err := parser.ParseMetricSelector(plugin.Spec.Matchers[0])
			require.NoError(t, err)
			got := make([]string, 0, len(matchers))
			for _, m := range matchers {
				got = append(got, m.String())
			}
			assert.ElementsMatch(t, tc.want, got)
		})
	}
}
