// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package common

import (
	"testing"
)

func TestCalculateSpecHash(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		wantLen  int
		wantSame bool
	}{
		{
			name:    "basic hash calculation",
			data:    []byte(`{"test": "data"}`),
			wantLen: 64, // SHA256 produces 32 bytes = 64 hex characters
		},
		{
			name:    "empty data",
			data:    []byte{},
			wantLen: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateSpecHash(tt.data)
			if len(got) != tt.wantLen {
				t.Errorf("calculateSpecHash() returned hash of length %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestCalculateSpecHash_Deterministic(t *testing.T) {
	data := []byte(`{"prometheusRule": "test", "values": [1, 2, 3]}`)

	hash1 := calculateSpecHash(data)
	hash2 := calculateSpecHash(data)

	if hash1 != hash2 {
		t.Errorf("calculateSpecHash() should be deterministic, got %s and %s", hash1, hash2)
	}
}

func TestCalculateSpecHash_DifferentData(t *testing.T) {
	data1 := []byte(`{"recommendationPercentage": 80}`)
	data2 := []byte(`{"recommendationPercentage": 90}`)

	hash1 := calculateSpecHash(data1)
	hash2 := calculateSpecHash(data2)

	if hash1 == hash2 {
		t.Errorf("calculateSpecHash() should produce different hashes for different data")
	}
}
