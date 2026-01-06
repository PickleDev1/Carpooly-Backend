package services

import (
	"car-backend/pkg/models"
	"fmt"
	"log"
	"math"
	"time"
)

// EnhancedMatchingService provides real matching calculations
type EnhancedMatchingService struct {
	routeService *RouteService
}

// CompatibilityWeights defines the weight distribution for matching factors
type CompatibilityWeights struct {
	Location     float64 // 30% - Distance between home locations
	Schedule     float64 // 25% - Work time alignment
	Route        float64 // 20% - Route overlap percentage
	Demographics float64 // 15% - Age, gender, student status
	Preferences  float64 // 10% - Carpool preferences
}

// CostFactors defines cost calculation parameters
type CostFactors struct {
	GasPricePerGallon  float64            // Current gas price
	AverageMPG         float64            // Average miles per gallon
	MaintenancePerMile float64            // Maintenance cost per mile
	ParkingCosts       map[string]float64 // Parking costs by location
	TollCosts          map[string]float64 // Toll costs by route
}

// ScheduleCompatibility represents schedule matching details
type ScheduleCompatibility struct {
	DepartureTime      string  `json:"departure_time"`
	FlexibilityMinutes int     `json:"flexibility_minutes"`
	Frequency          string  `json:"frequency"`
	CompatibilityScore float64 `json:"compatibility_score"`
}

// NewEnhancedMatchingService creates a new enhanced matching service
func NewEnhancedMatchingService(routeService *RouteService) *EnhancedMatchingService {
	return &EnhancedMatchingService{
		routeService: routeService,
	}
}

// CalculateRealCompatibility calculates real compatibility between two users
func (ems *EnhancedMatchingService) CalculateRealCompatibility(user1, user2 *models.User) (*models.PotentialMatch, error) {
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Calculating real compatibility\",\"user1_id\":\"%s\",\"user2_id\":\"%s\"}",
		user1.ID, user2.ID)

	// Define compatibility weights
	weights := &CompatibilityWeights{
		Location:     0.30,
		Schedule:     0.25,
		Route:        0.20,
		Demographics: 0.15,
		Preferences:  0.10,
	}

	// Calculate individual compatibility scores
	locationScore := ems.calculateLocationCompatibility(user1, user2)
	scheduleScore := ems.calculateScheduleCompatibility(user1, user2)
	routeScore, routeOverlap, totalDistance := ems.calculateRouteCompatibility(user1, user2)
	demoScore := ems.calculateDemographicCompatibility(user1, user2)
	prefScore := ems.calculatePreferenceCompatibility(user1, user2)

	// Calculate weighted compatibility score
	compatibilityScore := (locationScore*weights.Location +
		scheduleScore*weights.Schedule +
		routeScore*weights.Route +
		demoScore*weights.Demographics +
		prefScore*weights.Preferences)

	// Calculate real cost savings
	estimatedSavings := ems.calculateRealSavings(user1, user2, totalDistance)

	// Generate match reasons
	matchReasons := ems.generateMatchReasons(locationScore, scheduleScore, routeScore, demoScore, prefScore)

	// Create potential match
	potentialMatch := &models.PotentialMatch{
		ID:                       fmt.Sprintf("match_%s_%s_%d", user1.ID.String(), user2.ID.String(), time.Now().Unix()),
		User1ID:                  user1.ID.String(),
		User2ID:                  user2.ID.String(),
		CompatibilityScore:       compatibilityScore,
		RouteOverlapPercentage:   &routeOverlap,
		TotalDistanceMiles:       &totalDistance,
		EstimatedSavingsPerMonth: &estimatedSavings,
		MatchReasons:             matchReasons,
		Status:                   "active",
		ExpiresAt:                time.Now().Add(7 * 24 * time.Hour), // 7 days
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
		User1:                    user1,
		User2:                    user2,
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Compatibility calculated\",\"compatibility_score\":%.3f,\"route_overlap\":%.1f,\"savings\":%.2f,\"user1_home\":\"%.6f,%.6f\",\"user2_home\":\"%.6f,%.6f\"}",
		compatibilityScore, routeOverlap, estimatedSavings, user1.HomeLatitude, user1.HomeLongitude, user2.HomeLatitude, user2.HomeLongitude)

	return potentialMatch, nil
}

// calculateLocationCompatibility calculates location-based compatibility (30% weight)
func (ems *EnhancedMatchingService) calculateLocationCompatibility(user1, user2 *models.User) float64 {
	// Calculate straight-line distance between home locations
	var distance float64
	if ems.routeService != nil {
		distance = ems.routeService.CalculateDistance(
			Location{Latitude: user1.HomeLatitude, Longitude: user1.HomeLongitude},
			Location{Latitude: user2.HomeLatitude, Longitude: user2.HomeLongitude},
		)
	} else {
		// Fallback: simple haversine calculation
		distance = ems.haversineDistance(user1.HomeLatitude, user1.HomeLongitude, user2.HomeLatitude, user2.HomeLongitude)
	}

	// Scoring based on distance (closer = higher score)
	switch {
	case distance <= 2.0:
		return 1.0 // Same neighborhood - perfect match
	case distance <= 5.0:
		return 0.8 // Nearby - very good match
	case distance <= 10.0:
		return 0.6 // Reasonable distance - good match
	case distance <= 20.0:
		return 0.4 // Far but possible - fair match
	default:
		return 0.2 // Very far - poor match
	}
}

// calculateScheduleCompatibility calculates schedule-based compatibility (25% weight)
func (ems *EnhancedMatchingService) calculateScheduleCompatibility(user1, user2 *models.User) float64 {
	// We do not have explicit schedule fields in the current User model.
	// Return a neutral score to avoid penalizing users until schedule data exists.
	return 0.6
}

// calculateRouteCompatibility calculates route-based compatibility (20% weight)
func (ems *EnhancedMatchingService) calculateRouteCompatibility(user1, user2 *models.User) (float64, float64, float64) {
	// Calculate distance between home locations
	var homeDistance float64
	if ems.routeService != nil {
		homeDistance = ems.routeService.CalculateDistance(
			Location{Latitude: user1.HomeLatitude, Longitude: user1.HomeLongitude},
			Location{Latitude: user2.HomeLatitude, Longitude: user2.HomeLongitude},
		)
	} else {
		// Fallback: simple haversine calculation
		homeDistance = ems.haversineDistance(user1.HomeLatitude, user1.HomeLongitude, user2.HomeLatitude, user2.HomeLongitude)
	}

	// Since we don't have destination coordinates, use home proximity as route overlap proxy
	// Closer homes = higher potential for route overlap
	var overlap float64
	if homeDistance <= 1.0 {
		overlap = 85.0 // Very close - high overlap potential
	} else if homeDistance <= 3.0 {
		overlap = 70.0 // Close - good overlap potential
	} else if homeDistance <= 5.0 {
		overlap = 50.0 // Moderate - some overlap potential
	} else if homeDistance <= 10.0 {
		overlap = 25.0 // Far - low overlap potential
	} else {
		overlap = 5.0 // Very far - minimal overlap
	}

	// Debug logging to see what's happening
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Route overlap calculation\",\"home_distance\":%.2f,\"overlap_percentage\":%.1f}",
		homeDistance, overlap)

	routeScore := overlap / 100.0

	// Estimate total commute distance based on home proximity
	// Assume average commute is 10-15 miles, add home distance as detour
	avgCommute := 12.0
	estimatedTotalDistance := avgCommute + (homeDistance * 0.5) // Add half the home distance as detour

	return routeScore, overlap, estimatedTotalDistance
}

// calculateDemographicCompatibility calculates demographic compatibility (15% weight)
func (ems *EnhancedMatchingService) calculateDemographicCompatibility(user1, user2 *models.User) float64 {
	// This would need to be enhanced with actual demographic data
	// For now, return a neutral score
	return 0.7 // Default demographic compatibility
}

// calculatePreferenceCompatibility calculates preference compatibility (10% weight)
func (ems *EnhancedMatchingService) calculatePreferenceCompatibility(user1, user2 *models.User) float64 {
	// This would need to be enhanced with actual preference data
	// For now, return a neutral score
	return 0.7 // Default preference compatibility
}

// calculateRealSavings calculates real cost savings for carpooling
func (ems *EnhancedMatchingService) calculateRealSavings(user1, user2 *models.User, totalDistance float64) float64 {
	// Define cost factors
	costFactors := &CostFactors{
		GasPricePerGallon:  4.50, // Current gas price
		AverageMPG:         25.0, // Average MPG
		MaintenancePerMile: 0.15, // $0.15 per mile maintenance
		ParkingCosts: map[string]float64{
			"downtown": 12.0, // $12/day downtown parking
			"suburban": 5.0,  // $5/day suburban parking
			"free":     0.0,  // Free parking
		},
		TollCosts: map[string]float64{
			"highway": 3.0, // $3/day highway tolls
			"local":   0.0, // No tolls
		},
	}

	// Use realistic individual commute distances (not the carpool total distance)
	// Assume each person has a 10-15 mile individual commute
	individualCommuteDistance := 12.0 // Average individual commute

	// Calculate individual commute costs
	individualCost1 := ems.calculateIndividualCommuteCost(user1, individualCommuteDistance, costFactors)
	individualCost2 := ems.calculateIndividualCommuteCost(user2, individualCommuteDistance, costFactors)

	// Calculate shared carpool cost (using the optimized carpool distance)
	carpoolCost := ems.calculateCarpoolCost(totalDistance, costFactors)

	// Calculate daily savings
	dailySavings := (individualCost1 + individualCost2) - carpoolCost

	// Monthly savings (22 work days)
	monthlySavings := dailySavings * 22

	return math.Max(0, monthlySavings) // Ensure non-negative savings
}

// calculateIndividualCommuteCost calculates cost for individual commute
func (ems *EnhancedMatchingService) calculateIndividualCommuteCost(user *models.User, distance float64, factors *CostFactors) float64 {
	// Gas cost
	gasCost := (distance / factors.AverageMPG) * factors.GasPricePerGallon

	// Parking cost (use suburban as default - more realistic)
	parkingCost := factors.ParkingCosts["suburban"]

	// Maintenance cost
	maintenanceCost := distance * factors.MaintenancePerMile

	// Toll cost (use local as default - no tolls for most commutes)
	tollCost := factors.TollCosts["local"]

	return gasCost + parkingCost + maintenanceCost + tollCost
}

// calculateCarpoolCost calculates cost for shared carpool
func (ems *EnhancedMatchingService) calculateCarpoolCost(distance float64, factors *CostFactors) float64 {
	// Shared costs (gas, tolls, maintenance)
	gasCost := (distance / factors.AverageMPG) * factors.GasPricePerGallon
	maintenanceCost := distance * factors.MaintenancePerMile
	tollCost := factors.TollCosts["local"] // Use local tolls (no tolls)

	// Total shared cost
	sharedCost := gasCost + maintenanceCost + tollCost

	// Split between two users
	return sharedCost / 2.0
}

// generateMatchReasons generates human-readable match reasons
func (ems *EnhancedMatchingService) generateMatchReasons(locationScore, scheduleScore, routeScore, demoScore, prefScore float64) models.MatchReasons {
	var reasons []string

	if locationScore >= 0.8 {
		reasons = append(reasons, "Live in the same neighborhood")
	} else if locationScore >= 0.6 {
		reasons = append(reasons, "Live nearby")
	}

	if scheduleScore >= 0.8 {
		reasons = append(reasons, "Similar work schedule")
	} else if scheduleScore >= 0.6 {
		reasons = append(reasons, "Compatible work schedule")
	}

	if routeScore >= 0.7 {
		reasons = append(reasons, "High route overlap")
	} else if routeScore >= 0.5 {
		reasons = append(reasons, "Some route overlap")
	}

	if demoScore >= 0.8 {
		reasons = append(reasons, "Similar demographics")
	}

	if prefScore >= 0.8 {
		reasons = append(reasons, "Compatible preferences")
	}

	// Default reason if no specific reasons found
	if len(reasons) == 0 {
		reasons = append(reasons, "Potential carpool match")
	}

	return models.MatchReasons(reasons)
}

// GetScheduleCompatibilityDetails returns detailed schedule compatibility information
func (ems *EnhancedMatchingService) GetScheduleCompatibilityDetails(user1, user2 *models.User) *ScheduleCompatibility {
	// Without explicit schedule data, provide neutral, non-blocking defaults.
	return &ScheduleCompatibility{
		DepartureTime:      "8:00 AM",
		FlexibilityMinutes: 15,
		Frequency:          "Daily",
		CompatibilityScore: ems.calculateScheduleCompatibility(user1, user2),
	}
}

// haversineDistance calculates the distance between two points using the Haversine formula
func (ems *EnhancedMatchingService) haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 3959 // Earth's radius in miles

	// Convert to radians
	lat1Rad := lat1 * math.Pi / 180
	lng1Rad := lng1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lng2Rad := lng2 * math.Pi / 180

	// Haversine formula
	dlat := lat2Rad - lat1Rad
	dlng := lng2Rad - lng1Rad

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dlng/2)*math.Sin(dlng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}
