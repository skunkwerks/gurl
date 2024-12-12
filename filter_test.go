package main

import (
	"reflect"
	"testing"
)

func TestFilter(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedMethod string
		expectedURL    string
		expectedRest   []string
	}{
		{
			name:           "GET request",
			args:           []string{"http://example.com"},
			expectedMethod: "GET",
			expectedURL:    "http://example.com",
			expectedRest:   []string{},
		},
		{
			name:           "POST with data",
			args:           []string{"http://example.com", "key=value"},
			expectedMethod: "POST",
			expectedURL:    "http://example.com",
			expectedRest:   []string{"key=value"},
		},
		{
			name:           "Explicit POST",
			args:           []string{"POST", "http://example.com"},
			expectedMethod: "POST",
			expectedURL:    "http://example.com",
			expectedRest:   []string{},
		},
		{
			name:           "POST with file",
			args:           []string{"http://example.com", "file@test.txt"},
			expectedMethod: "POST",
			expectedURL:    "http://example.com",
			expectedRest:   []string{"file@test.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset globals before each test
			*method = "GET"
			*URL = ""

			rest := filter(tt.args)

			if *method != tt.expectedMethod {
				t.Errorf("method = %v, want %v", *method, tt.expectedMethod)
			}
			if *URL != tt.expectedURL {
				t.Errorf("URL = %v, want %v", *URL, tt.expectedURL)
			}
			if !reflect.DeepEqual(rest, tt.expectedRest) {
				t.Errorf("rest = %v, want %v", rest, tt.expectedRest)
			}
		})
	}
}
