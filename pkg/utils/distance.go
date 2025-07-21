package utils

import "math"

// CalculateDistance calculates the distance between two GPS coordinates using the Haversine formula
// Returns distance in miles
func CalculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 3959 // Earth's radius in miles (was 6371 km)

	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lng1Rad := lng1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lng2Rad := lng2 * math.Pi / 180

	// Differences in coordinates
	deltaLat := lat2Rad - lat1Rad
	deltaLng := lng2Rad - lng1Rad

	// Haversine formula
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// Calculate distance
	distance := earthRadius * c

	// Round to 2 decimal places for consistency
	return math.Round(distance*100) / 100
}

// KilometersToMiles converts kilometers to miles
func KilometersToMiles(kilometers float64) float64 {
	return kilometers * 0.621371
}

// MilesToKilometers converts miles to kilometers
func MilesToKilometers(miles float64) float64 {
	return miles * 1.60934
}

// IsValidCoordinate checks if a coordinate is valid
func IsValidCoordinate(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

// IsValidDistance checks if a distance value is reasonable (0-1000 miles)
func IsValidDistance(distance float64) bool {
	return distance >= 0 && distance <= 1000
}
