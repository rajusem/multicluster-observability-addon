// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package manifests

import (
	"github.com/stolostron/multicluster-observability-addon/internal/addon"
)

// RightSizingValues represents the values for right-sizing Helm chart templates
type RightSizingValues struct {
	NamespaceEnabled      bool   `json:"namespaceEnabled"`
	NamespaceBinding      string `json:"namespaceBinding,omitempty"`
	VirtualizationEnabled bool   `json:"virtualizationEnabled"`
	VirtualizationBinding string `json:"virtualizationBinding,omitempty"`
}

// EnableRightSizing creates the RightSizingValues from the addon options
func EnableRightSizing(opts addon.RightSizingOptions) *RightSizingValues {
	if !opts.NamespaceEnabled && !opts.VirtualizationEnabled {
		return nil
	}
	return &RightSizingValues{
		NamespaceEnabled:      opts.NamespaceEnabled,
		NamespaceBinding:      opts.NamespaceBinding,
		VirtualizationEnabled: opts.VirtualizationEnabled,
		VirtualizationBinding: opts.VirtualizationBinding,
	}
}
