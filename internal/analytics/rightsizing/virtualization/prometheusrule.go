// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package virtualization

import (
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stolostron/multicluster-observability-addon/internal/analytics/rightsizing/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// GeneratePrometheusRule builds PrometheusRule based on configdata
func GeneratePrometheusRule(configData common.RSNamespaceConfigMapData) (monitoringv1.PrometheusRule, error) {
	nsFilter, err := common.BuildNamespaceFilter(configData.PrometheusRuleConfig)
	if err != nil {
		return monitoringv1.PrometheusRule{}, err
	}

	labelJoin, err := common.BuildLabelJoin(configData.PrometheusRuleConfig.LabelFilterCriteria)
	if err != nil {
		return monitoringv1.PrometheusRule{}, err
	}

	// Define durations
	duration5m := monitoringv1.Duration("5m")
	duration1d := monitoringv1.Duration("15m")

	// Helper for rules
	rule := func(record, metricExpr string) monitoringv1.Rule {
		expr := metricExpr
		if labelJoin != "" {
			expr = fmt.Sprintf("%s %s", metricExpr, labelJoin)
		}
		return monitoringv1.Rule{
			Record: record,
			Expr:   intstr.FromString(expr),
		}
	}

	ruleWithLabels := func(record, expr string) monitoringv1.Rule {
		return monitoringv1.Rule{
			Record: record,
			Expr:   intstr.FromString(expr),
			Labels: map[string]string{
				"profile":     "Max OverAll",
				"aggregation": "1d",
			},
		}
	}

	return monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PrometheusRuleName,
			Namespace: common.MonitoringNamespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "PrometheusRule",
			APIVersion: "monitoring.coreos.com/v1",
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name:     "acm-vm-right-sizing-namespace-5m.rule",
					Interval: &duration5m,
					Rules:    buildNamespaceRules5m(nsFilter, rule),
				},
				{
					Name:     "acm-vm-right-sizing-namespace-1d.rules",
					Interval: &duration1d,
					Rules:    buildNamespaceRules1d(configData, ruleWithLabels),
				},
				{
					Name:     "acm-vm-right-sizing-cluster-5m.rule",
					Interval: &duration5m,
					Rules:    buildClusterRules5m(nsFilter, rule),
				},
				{
					Name:     "acm-vm-right-sizing-cluster-1d.rule",
					Interval: &duration1d,
					Rules:    buildClusterRules1d(configData, ruleWithLabels),
				},
			},
		},
	}, nil
}

func buildNamespaceRules5m(
	nsFilter string,
	rule func(string, string) monitoringv1.Rule,
) []monitoringv1.Rule {
	return []monitoringv1.Rule{
		rule(
			"acm_rs_vm:namespace:cpu_request:5m",
			fmt.Sprintf(
				`max_over_time(sum (
				  (kubevirt_vm_resource_requests{%s, unit="cores", resource="cpu"} *
				  on(name,namespace,resource)
				  kubevirt_vm_resource_requests{%s, unit="sockets", resource="cpu"} *
				  on(name,namespace,resource)
				  kubevirt_vm_resource_requests{%s, unit="threads", resource="cpu"})
				) by (name, namespace)[5m:])`,
				nsFilter, nsFilter, nsFilter,
			),
		),
		rule(
			"acm_rs_vm:namespace:memory_request:5m",
			fmt.Sprintf(
				`max_over_time(sum (
				  kubevirt_vm_resource_requests{%s, resource="memory"}
				) by (name,namespace)[5m:])`,
				nsFilter,
			),
		),
		rule(
			"acm_rs_vm:namespace:cpu_usage:5m",
			fmt.Sprintf(
				`max_over_time(sum (
				  rate(kubevirt_vmi_cpu_usage_seconds_total{%s}[5m:])
				) by (name,namespace)[5m:])`,
				nsFilter,
			),
		),
		rule(
			"acm_rs_vm:namespace:memory_usage:5m",
			fmt.Sprintf(
				`max_over_time(sum (
				  kubevirt_vmi_memory_available_bytes{%s} -
				  kubevirt_vmi_memory_usable_bytes{%s}
				) by (name,namespace)[5m:])`,
				nsFilter, nsFilter,
			),
		),
	}
}

func buildNamespaceRules1d(
	configData common.RSNamespaceConfigMapData,
	ruleWithLabels func(string, string) monitoringv1.Rule,
) []monitoringv1.Rule {
	rp := configData.PrometheusRuleConfig.RecommendationPercentage
	return []monitoringv1.Rule{
		ruleWithLabels("acm_rs_vm:namespace:cpu_request", `max_over_time(acm_rs_vm:namespace:cpu_request:5m[1d])`),
		ruleWithLabels("acm_rs_vm:namespace:cpu_usage", `max_over_time(acm_rs_vm:namespace:cpu_usage:5m[1d])`),
		ruleWithLabels("acm_rs_vm:namespace:memory_request", `max_over_time(acm_rs_vm:namespace:memory_request:5m[1d])`),
		ruleWithLabels("acm_rs_vm:namespace:memory_usage", `max_over_time(acm_rs_vm:namespace:memory_usage:5m[1d])`),
		ruleWithLabels(
			"acm_rs_vm:namespace:cpu_recommendation",
			fmt.Sprintf(`max_over_time(acm_rs_vm:namespace:cpu_usage:5m[1d])*(%d/100)`, rp),
		),
		ruleWithLabels(
			"acm_rs_vm:namespace:memory_recommendation",
			fmt.Sprintf(`max_over_time(acm_rs_vm:namespace:memory_usage:5m[1d])*(%d/100)`, rp),
		),
	}
}

func buildClusterRules5m(
	nsFilter string,
	rule func(string, string) monitoringv1.Rule,
) []monitoringv1.Rule {
	return []monitoringv1.Rule{
		rule(
			"acm_rs_vm:cluster:cpu_request:5m",
			fmt.Sprintf(
				`max_over_time(sum (
				  (kubevirt_vm_resource_requests{%s, unit="cores", resource="cpu"} *
				  on(name,namespace,resource)
				  kubevirt_vm_resource_requests{%s, unit="sockets", resource="cpu"} *
				  on(name,namespace,resource)
				  kubevirt_vm_resource_requests{%s, unit="threads", resource="cpu"})
				) by (cluster)[5m:])`,
				nsFilter, nsFilter, nsFilter,
			),
		),
		rule(
			"acm_rs_vm:cluster:cpu_usage:5m",
			fmt.Sprintf(
				`max_over_time(sum (
				  rate(kubevirt_vmi_cpu_usage_seconds_total{%s}[5m:])
				) by (cluster)[5m:])`,
				nsFilter,
			),
		),
		rule(
			"acm_rs_vm:cluster:memory_request:5m",
			fmt.Sprintf(
				`max_over_time(sum (
				  kubevirt_vm_resource_requests{%s, resource="memory"}
				) by (cluster)[5m:])`,
				nsFilter,
			),
		),
		rule(
			"acm_rs_vm:cluster:memory_usage:5m",
			fmt.Sprintf(
				`max_over_time(sum (
				  kubevirt_vmi_memory_available_bytes{%s} -
				  kubevirt_vmi_memory_usable_bytes{%s}
				) by (cluster)[5m:])`,
				nsFilter, nsFilter,
			),
		),
	}
}

func buildClusterRules1d(
	configData common.RSNamespaceConfigMapData,
	ruleWithLabels func(string, string) monitoringv1.Rule,
) []monitoringv1.Rule {
	rp := configData.PrometheusRuleConfig.RecommendationPercentage
	return []monitoringv1.Rule{
		ruleWithLabels("acm_rs_vm:cluster:cpu_request", `max_over_time(acm_rs_vm:cluster:cpu_request:5m[1d])`),
		ruleWithLabels("acm_rs_vm:cluster:cpu_usage", `max_over_time(acm_rs_vm:cluster:cpu_usage:5m[1d])`),
		ruleWithLabels(
			"acm_rs_vm:cluster:cpu_recommendation",
			fmt.Sprintf(
				`max_over_time(acm_rs_vm:cluster:cpu_usage:5m[1d]) * (%d/100)`,
				rp,
			),
		),
		ruleWithLabels("acm_rs_vm:cluster:memory_request", `max_over_time(acm_rs_vm:cluster:memory_request:5m[1d])`),
		ruleWithLabels("acm_rs_vm:cluster:memory_usage", `max_over_time(acm_rs_vm:cluster:memory_usage:5m[1d])`),
		ruleWithLabels(
			"acm_rs_vm:cluster:memory_recommendation",
			fmt.Sprintf(
				`max_over_time(acm_rs_vm:cluster:memory_usage:5m[1d]) * (%d/100)`,
				rp,
			),
		),
	}
}
