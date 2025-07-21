package utils

import (
	"math"
	"testing"
)

func TestCalculateDistance(t *testing.T) {
	// Test case: San Francisco to New York (approximately 2575 miles)
	sfLat, sfLng := 37.7749, -122.4194
	nyLat, nyLng := 40.7128, -74.0060

	distance := CalculateDistance(sfLat, sfLng, nyLat, nyLng)

	// The actual distance is approximately 2575 miles, but we'll allow some tolerance
	expectedMin := 2500.0
	expectedMax := 2650.0

	if distance < expectedMin || distance > expectedMax {
		t.Errorf("Distance calculation failed. Expected between %.1f and %.1f miles, got %.2f", expectedMin, expectedMax, distance)
	}

	t.Logf("Calculated distance from SF to NY: %.2f miles", distance)
}

func TestKilometersToMiles(t *testing.T) {
	kilometers := 100.0
	expectedMiles := 62.1371

	miles := KilometersToMiles(kilometers)

	if math.Abs(miles-expectedMiles) > 0.01 {
		t.Errorf("KilometersToMiles failed. Expected %.4f miles, got %.4f", expectedMiles, miles)
	}
}

func TestMilesToKilometers(t *testing.T) {
	miles := 62.1371
	expectedKilometers := 100.0

	kilometers := MilesToKilometers(miles)

	if math.Abs(kilometers-expectedKilometers) > 0.01 {
		t.Errorf("MilesToKilometers failed. Expected %.1f kilometers, got %.1f", expectedKilometers, kilometers)
	}
}

func TestIsValidCoordinate(t *testing.T) {
	tests := []struct {
		name     string
		lat, lng float64
		expected bool
	}{
		{"Valid coordinates", 37.7749, -122.4194, true},
		{"Invalid latitude too high", 91.0, -122.4194, false},
		{"Invalid latitude too low", -91.0, -122.4194, false},
		{"Invalid longitude too high", 37.7749, 181.0, false},
		{"Invalid longitude too low", 37.7749, -181.0, false},
		{"Edge case valid", 90.0, 180.0, true},
		{"Edge case valid negative", -90.0, -180.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidCoordinate(tt.lat, tt.lng)
			if result != tt.expected {
				t.Errorf("IsValidCoordinate(%f, %f) = %v, expected %v", tt.lat, tt.lng, result, tt.expected)
			}
		})
	}
}

func TestIsValidDistance(t *testing.T) {
	tests := []struct {
		name     string
		distance float64
		expected bool
	}{
		{"Valid distance", 100.0, true},
		{"Zero distance", 0.0, true},
		{"Negative distance", -10.0, false},
		{"Too large distance", 1001.0, false},
		{"Edge case valid", 1000.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidDistance(tt.distance)
			if result != tt.expected {
				t.Errorf("IsValidDistance(%f) = %v, expected %v", tt.distance, result, tt.expected)
			}
		})
	}
}
