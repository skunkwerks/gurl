package main

import (
	"testing"
)

func TestColor(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		color    uint8
		expected string
	}{
		{
			name:     "gray color",
			str:      "test",
			color:    Gray,
			expected: "\033[90mtest\033[0m",
		},
		{
			name:     "magenta color",
			str:      "test",
			color:    Magenta,
			expected: "\033[95mtest\033[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Color(tt.str, tt.color)
			if result != tt.expected {
				t.Errorf("Color() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestColorfulRequest(t *testing.T) {
	input := "GET /path HTTP/1.1\nHost: example.com"
	expected := "GET /path HTTP/1.1\n\033[90mHost\033[0m:\033[96m example.com\033[0m\n"
	result := ColorfulRequest(input)
	if result != expected {
		t.Errorf("ColorfulRequest() = %v, want %v", result, expected)
	}
}

func TestColorfulJson(t *testing.T) {
	input := `{"key": "value"}`
	expected := "{\"[95mkey[0m\": \"[96mvalue[0m\"}\n"
	result := ColorfulJson(input)
	if result != expected {
		t.Errorf("ColorfulJson() = %v, want %v", result, expected)
	}
}
