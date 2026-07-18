//go:build !integration

package utils

import (
	"slices"
	"testing"

	"github.com/MadalinGOIAN/food-stock/internal/utils"
)

const (
	testOrigin1 = "https://test.food-stock.xyz"
	testOrigin2 = "http://localhost:8080"
)

func TestParseOrigins_Valid(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected []string
	}{
		{"single", testOrigin1, []string{testOrigin1}},
		{"multiple", testOrigin1 + "," + testOrigin2, []string{testOrigin1, testOrigin2}},
		{"trims spaces", "  " + testOrigin1 + " , " + testOrigin2 + "  ", []string{testOrigin1, testOrigin2}},
		{"skips empty fields", testOrigin1 + ",," + testOrigin2, []string{testOrigin1, testOrigin2}},
		{"trailing comma", testOrigin1 + ",", []string{testOrigin1}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			received, err := utils.ParseOrigins(tc.raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(received, tc.expected) {
				t.Errorf("ParseOrigins(%q) = %v, expected %v", tc.raw, received, tc.expected)
			}
		})
	}
}

func TestParseOrigins_Invalid(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"commas only", ",,"},
		{"blank fields only", " , , "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			received, err := utils.ParseOrigins(tc.raw)
			if err == nil {
				t.Fatalf("ParseOrigins(%q) = %v, expected error", tc.raw, received)
			}
			if received != nil {
				t.Errorf("ParseOrigins(%q) returned %v, expected nil slice on error", tc.raw, received)
			}
		})
	}
}
