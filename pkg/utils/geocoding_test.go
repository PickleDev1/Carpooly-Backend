package utils

import (
	"testing"
)

func TestReverseGeocode(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected *GeocodingResult
		hasError bool
	}{
		{
			name:    "San Francisco",
			address: "San Francisco",
			expected: &GeocodingResult{
				Latitude:  37.7749,
				Longitude: -122.4194,
				Address:   "San Francisco",
			},
			hasError: false,
		},
		{
			name:    "New York",
			address: "New York",
			expected: &GeocodingResult{
				Latitude:  40.7128,
				Longitude: -74.0060,
				Address:   "New York",
			},
			hasError: false,
		},
		{
			name:    "Partial match - contains city name",
			address: "I'm going to San Francisco tomorrow",
			expected: &GeocodingResult{
				Latitude:  37.7749,
				Longitude: -122.4194,
				Address:   "I'm going to San Francisco tomorrow",
			},
			hasError: false,
		},
		{
			name:     "Empty address",
			address:  "",
			expected: nil,
			hasError: true,
		},
		{
			name:     "Unknown address",
			address:  "Unknown City XYZ",
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ReverseGeocode(tt.address)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for address '%s', but got none", tt.address)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for address '%s': %v", tt.address, err)
				return
			}

			if result == nil {
				t.Errorf("Expected result for address '%s', but got nil", tt.address)
				return
			}

			if result.Latitude != tt.expected.Latitude {
				t.Errorf("Expected latitude %.6f, got %.6f", tt.expected.Latitude, result.Latitude)
			}

			if result.Longitude != tt.expected.Longitude {
				t.Errorf("Expected longitude %.6f, got %.6f", tt.expected.Longitude, result.Longitude)
			}

			if result.Address != tt.expected.Address {
				t.Errorf("Expected address '%s', got '%s'", tt.expected.Address, result.Address)
			}
		})
	}
}

func TestGetCoordinatesForAddress(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected *GeocodingResult
	}{
		{
			name:    "Exact match - lowercase",
			address: "san francisco",
			expected: &GeocodingResult{
				Latitude:  37.7749,
				Longitude: -122.4194,
				Address:   "san francisco",
			},
		},
		{
			name:    "Exact match - mixed case",
			address: "San Francisco",
			expected: &GeocodingResult{
				Latitude:  37.7749,
				Longitude: -122.4194,
				Address:   "San Francisco",
			},
		},
		{
			name:    "Partial match",
			address: "I live in San Francisco",
			expected: &GeocodingResult{
				Latitude:  37.7749,
				Longitude: -122.4194,
				Address:   "I live in San Francisco",
			},
		},
		{
			name:     "No match",
			address:  "Unknown City",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getCoordinatesForAddress(tt.address)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil result for address '%s', but got %+v", tt.address, result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected result for address '%s', but got nil", tt.address)
				return
			}

			if result.Latitude != tt.expected.Latitude {
				t.Errorf("Expected latitude %.6f, got %.6f", tt.expected.Latitude, result.Latitude)
			}

			if result.Longitude != tt.expected.Longitude {
				t.Errorf("Expected longitude %.6f, got %.6f", tt.expected.Longitude, result.Longitude)
			}

			if result.Address != tt.expected.Address {
				t.Errorf("Expected address '%s', got '%s'", tt.expected.Address, result.Address)
			}
		})
	}
}
