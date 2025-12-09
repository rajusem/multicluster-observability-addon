// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package common

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
)

// FormatYAML converts a Go data structure to a YAML-formatted string
func FormatYAML[T RSPrometheusRuleConfig | clusterv1beta1.Placement](data T) string {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		log.Error(err, "rs - error marshaling data to yaml: %v")
		return ""
	}
	return string(yamlData)
}

// GetDefaultRSPrometheusRuleConfig creates a default prometheus rule configuration for right-sizing
func GetDefaultRSPrometheusRuleConfig() RSPrometheusRuleConfig {
	var ruleConfig RSPrometheusRuleConfig
	ruleConfig.NamespaceFilterCriteria.ExclusionCriteria = []string{"openshift.*"}
	ruleConfig.RecommendationPercentage = DefaultRecommendationPercentage
	return ruleConfig
}

// BuildNamespaceFilter creates a namespace filter string for Prometheus queries
func BuildNamespaceFilter(nsConfig RSPrometheusRuleConfig) (string, error) {
	ns := nsConfig.NamespaceFilterCriteria
	if len(ns.InclusionCriteria) > 0 && len(ns.ExclusionCriteria) > 0 {
		return "", fmt.Errorf("only one of inclusion or exclusion criteria allowed for namespacefiltercriteria")
	}
	if len(ns.InclusionCriteria) > 0 {
		return fmt.Sprintf(`namespace=~"%s"`, strings.Join(ns.InclusionCriteria, "|")), nil
	}
	if len(ns.ExclusionCriteria) > 0 {
		return fmt.Sprintf(`namespace!~"%s"`, strings.Join(ns.ExclusionCriteria, "|")), nil
	}
	return `namespace!=""`, nil
}

// BuildLabelJoin creates a label join string for Prometheus queries
func BuildLabelJoin(labelFilters []RSLabelFilter) (string, error) {
	for _, l := range labelFilters {
		if l.LabelName != "label_env" {
			continue
		}
		if len(l.InclusionCriteria) > 0 && len(l.ExclusionCriteria) > 0 {
			return "", fmt.Errorf("only one of inclusion or exclusion allowed for label_env")
		}
		var selector string
		if len(l.InclusionCriteria) > 0 {
			selector = fmt.Sprintf(`kube_namespace_labels{label_env=~"%s"}`, strings.Join(l.InclusionCriteria, "|"))
		} else if len(l.ExclusionCriteria) > 0 {
			selector = fmt.Sprintf(`kube_namespace_labels{label_env!~"%s"}`, strings.Join(l.ExclusionCriteria, "|"))
		} else {
			continue
		}
		return fmt.Sprintf(`* on (namespace) group_left() (%s or kube_namespace_labels{label_env=""})`, selector), nil
	}
	return "", nil
}
