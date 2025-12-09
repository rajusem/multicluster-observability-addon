// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package handlers

import (
	"testing"

	"github.com/stolostron/multicluster-observability-addon/internal/analytics/rightsizing/common"
	"github.com/stretchr/testify/assert"
)

func TestIsRightSizingEnabled(t *testing.T) {
	tests := []struct {
		name     string
		opts     common.RightSizingOptions
		expected bool
	}{
		{
			name: "both disabled",
			opts: common.RightSizingOptions{
				NamespaceEnabled:      false,
				VirtualizationEnabled: false,
			},
			expected: false,
		},
		{
			name: "namespace enabled only",
			opts: common.RightSizingOptions{
				NamespaceEnabled:      true,
				VirtualizationEnabled: false,
			},
			expected: true,
		},
		{
			name: "virtualization enabled only",
			opts: common.RightSizingOptions{
				NamespaceEnabled:      false,
				VirtualizationEnabled: true,
			},
			expected: true,
		},
		{
			name: "both enabled",
			opts: common.RightSizingOptions{
				NamespaceEnabled:      true,
				VirtualizationEnabled: true,
			},
			expected: true,
		},
		{
			name: "with binding namespaces",
			opts: common.RightSizingOptions{
				NamespaceEnabled:      true,
				NamespaceBinding:      "custom-namespace",
				VirtualizationEnabled: true,
				VirtualizationBinding: "virt-namespace",
			},
			expected: true,
		},
		{
			name: "empty options",
			opts: common.RightSizingOptions{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRightSizingEnabled(tt.opts)
			assert.Equal(t, tt.expected, result)
		})
	}
}
