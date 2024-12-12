package main

import (
	"testing"
)

func TestInSlice(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		slice    []string
		expected bool
	}{
		{
			name:     "empty slice",
			str:      "test",
			slice:    []string{},
			expected: false,
		},
		{
			name:     "string present",
			str:      "test",
			slice:    []string{"test", "other"},
			expected: true,
		},
		{
			name:     "string not present",
			str:      "missing",
			slice:    []string{"test", "other"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inSlice(tt.str, tt.slice)
			if result != tt.expected {
				t.Errorf("inSlice() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "bytes",
			bytes:    500,
			expected: "500 B",
		},
		{
			name:     "kilobytes",
			bytes:    1500,
			expected: "1.46 KB",
		},
		{
			name:     "megabytes",
			bytes:    1500000,
			expected: "1.43 MB",
		},
		{
			name:     "gigabytes",
			bytes:    1500000000,
			expected: "1.40 GB",
		},
		{
			name:     "terabytes",
			bytes:    1500000000000,
			expected: "1.36 TB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytes() = %v, want %v", result, tt.expected)
			}
		})
	}
}
