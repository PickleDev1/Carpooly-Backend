package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// GeocodingResult represents the result of a geocoding operation
type GeocodingResult struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address"`
}

// ReverseGeocode converts an address to GPS coordinates using a geocoding service
// For now, we'll use a simple implementation that can be extended with actual geocoding APIs
func ReverseGeocode(address string) (*GeocodingResult, error) {
	// For development/testing, we'll use a simple approach
	// In production, you would integrate with Google Maps API, OpenStreetMap, or similar

	// Clean the address
	cleanAddress := strings.TrimSpace(address)
	if cleanAddress == "" {
		return nil, fmt.Errorf("empty address provided")
	}

	// For now, we'll use a simple mapping for common addresses
	// In production, replace this with actual geocoding API calls
	coordinates := getCoordinatesForAddress(cleanAddress)
	if coordinates != nil {
		return &GeocodingResult{
			Latitude:  coordinates.Latitude,
			Longitude: coordinates.Longitude,
			Address:   cleanAddress,
		}, nil
	}

	// If no predefined coordinates found, try using a geocoding API
	// For now, we'll return an error indicating geocoding is needed
	return nil, fmt.Errorf("geocoding not implemented for address: %s", cleanAddress)
}

// getCoordinatesForAddress provides predefined coordinates for common addresses
// This is a temporary solution for development - replace with actual geocoding API
func getCoordinatesForAddress(address string) *GeocodingResult {
	// Convert to lowercase for case-insensitive matching
	addressLower := strings.ToLower(address)

	// Predefined coordinates for common addresses
	addressMap := map[string]GeocodingResult{
		"san francisco": {
			Latitude:  37.7749,
			Longitude: -122.4194,
			Address:   address,
		},
		"new york": {
			Latitude:  40.7128,
			Longitude: -74.0060,
			Address:   address,
		},
		"los angeles": {
			Latitude:  34.0522,
			Longitude: -118.2437,
			Address:   address,
		},
		"chicago": {
			Latitude:  41.8781,
			Longitude: -87.6298,
			Address:   address,
		},
		"miami": {
			Latitude:  25.7617,
			Longitude: -80.1918,
			Address:   address,
		},
		"seattle": {
			Latitude:  47.6062,
			Longitude: -122.3321,
			Address:   address,
		},
		"austin": {
			Latitude:  30.2672,
			Longitude: -97.7431,
			Address:   address,
		},
		"denver": {
			Latitude:  39.7392,
			Longitude: -104.9903,
			Address:   address,
		},
		"boston": {
			Latitude:  42.3601,
			Longitude: -71.0589,
			Address:   address,
		},
		"atlanta": {
			Latitude:  33.7490,
			Longitude: -84.3880,
			Address:   address,
		},
	}

	// Check for exact matches first
	if result, exists := addressMap[addressLower]; exists {
		return &result
	}

	// Check for partial matches
	for key, result := range addressMap {
		if strings.Contains(addressLower, key) {
			result.Address = address
			return &result
		}
	}

	return nil
}

// ReverseGeocodeWithAPI is a placeholder for actual geocoding API integration
// This would typically use Google Maps API, OpenStreetMap, or similar services
func ReverseGeocodeWithAPI(address string) (*GeocodingResult, error) {
	// Example implementation using a hypothetical geocoding API
	// Replace with actual API integration

	// URL encode the address
	encodedAddress := url.QueryEscape(address)

	// Example API call (replace with actual geocoding service)
	apiURL := fmt.Sprintf("https://api.example.com/geocode?address=%s", encodedAddress)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to call geocoding API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geocoding API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read API response: %v", err)
	}

	// Parse the API response (this would depend on the actual API format)
	var result GeocodingResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %v", err)
	}

	return &result, nil
}

// ReverseGeocodeCoordinates converts GPS coordinates to an address using Google Maps API
func ReverseGeocodeCoordinates(latitude, longitude float64, apiKey string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("Google Maps API key is required for reverse geocoding")
	}

	// Construct the Google Maps Geocoding API URL
	apiURL := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/geocode/json?latlng=%f,%f&key=%s",
		latitude, longitude, apiKey,
	)

	// Make the HTTP request
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to call Google Maps API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Google Maps API returned status: %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read API response: %v", err)
	}

	// Parse the JSON response
	var result struct {
		Status  string `json:"status"`
		Results []struct {
			FormattedAddress string `json:"formatted_address"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse API response: %v", err)
	}

	// Check if the API call was successful
	if result.Status != "OK" {
		return "", fmt.Errorf("Google Maps API returned status: %s", result.Status)
	}

	// Return the formatted address if available
	if len(result.Results) > 0 {
		return result.Results[0].FormattedAddress, nil
	}

	return "", fmt.Errorf("no address found for coordinates: %f, %f", latitude, longitude)
}
