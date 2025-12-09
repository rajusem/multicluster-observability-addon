// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package manifests

import (
	"testing"

	"github.com/stolostron/multicluster-observability-addon/internal/addon"
	"github.com/stretchr/testify/assert"
)

func TestEnableRightSizing(t *testing.T) {
	tests := []struct {
		name     string
		opts     addon.RightSizingOptions
		expected *RightSizingValues
	}{
		{
			name: "both disabled - returns nil",
			opts: addon.RightSizingOptions{
				NamespaceEnabled:      false,
				VirtualizationEnabled: false,
			},
			expected: nil,
		},
		{
			name: "namespace enabled only",
			opts: addon.RightSizingOptions{
				NamespaceEnabled:      true,
				VirtualizationEnabled: false,
			},
			expected: &RightSizingValues{
				NamespaceEnabled:      true,
				VirtualizationEnabled: false,
			},
		},
		{
			name: "virtualization enabled only",
			opts: addon.RightSizingOptions{
				NamespaceEnabled:      false,
				VirtualizationEnabled: true,
			},
			expected: &RightSizingValues{
				NamespaceEnabled:      false,
				VirtualizationEnabled: true,
			},
		},
		{
			name: "both enabled",
			opts: addon.RightSizingOptions{
				NamespaceEnabled:      true,
				VirtualizationEnabled: true,
			},
			expected: &RightSizingValues{
				NamespaceEnabled:      true,
				VirtualizationEnabled: true,
			},
		},
		{
			name: "with binding namespaces",
			opts: addon.RightSizingOptions{
				NamespaceEnabled:      true,
				NamespaceBinding:      "custom-namespace",
				VirtualizationEnabled: true,
				VirtualizationBinding: "virt-namespace",
			},
			expected: &RightSizingValues{
				NamespaceEnabled:      true,
				NamespaceBinding:      "custom-namespace",
				VirtualizationEnabled: true,
				VirtualizationBinding: "virt-namespace",
			},
		},
		{
			name:     "empty options - returns nil",
			opts:     addon.RightSizingOptions{},
			expected: nil,
		},
		{
			name: "namespace enabled with binding, virtualization disabled",
			opts: addon.RightSizingOptions{
				NamespaceEnabled:      true,
				NamespaceBinding:      "my-namespace",
				VirtualizationEnabled: false,
			},
			expected: &RightSizingValues{
				NamespaceEnabled:      true,
				NamespaceBinding:      "my-namespace",
				VirtualizationEnabled: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EnableRightSizing(tt.opts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRightSizingValuesFields(t *testing.T) {
	values := &RightSizingValues{
		NamespaceEnabled:      true,
		NamespaceBinding:      "ns-binding",
		VirtualizationEnabled: true,
		VirtualizationBinding: "virt-binding",
	}

	assert.True(t, values.NamespaceEnabled)
	assert.Equal(t, "ns-binding", values.NamespaceBinding)
	assert.True(t, values.VirtualizationEnabled)
	assert.Equal(t, "virt-binding", values.VirtualizationBinding)
}
