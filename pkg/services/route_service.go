package services

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"googlemaps.github.io/maps"
)

// Location represents a geographical point
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Route represents a calculated route with detailed information
type Route struct {
	TotalDistance float64    `json:"total_distance"` // in miles
	TotalDuration int        `json:"total_duration"` // in seconds
	Steps         []Step     `json:"steps"`          // detailed route steps
	Polyline      string     `json:"polyline"`       // encoded polyline
	StartLocation Location   `json:"start_location"`
	EndLocation   Location   `json:"end_location"`
	Waypoints     []Location `json:"waypoints"`
	CreatedAt     time.Time  `json:"created_at"`
}

// Step represents a single step in a route
type Step struct {
	Distance      float64  `json:"distance"` // in miles
	Duration      int      `json:"duration"` // in seconds
	Instruction   string   `json:"instruction"`
	StartLocation Location `json:"start_location"`
	EndLocation   Location `json:"end_location"`
	Polyline      string   `json:"polyline"`
}

// RouteOverlap represents the overlap between two routes
type RouteOverlap struct {
	OverlapPercentage float64   `json:"overlap_percentage"`
	CommonSegments    []Segment `json:"common_segments"`
	TotalOverlapMiles float64   `json:"total_overlap_miles"`
}

// Segment represents a road segment
type Segment struct {
	StartLocation Location `json:"start_location"`
	EndLocation   Location `json:"end_location"`
	Distance      float64  `json:"distance"`
	RoadName      string   `json:"road_name"`
}

// RouteService handles all route-related calculations
type RouteService struct {
	client *maps.Client
	cache  map[string]*CachedRoute
}

// CachedRoute represents a cached route result
type CachedRoute struct {
	Route     *Route
	ExpiresAt time.Time
}

// NewRouteService creates a new RouteService instance
func NewRouteService(apiKey string) (*RouteService, error) {
	client, err := maps.NewClient(maps.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Google Maps client: %w", err)
	}

	return &RouteService{
		client: client,
		cache:  make(map[string]*CachedRoute),
	}, nil
}

// GetRoute calculates a route between two points
func (rs *RouteService) GetRoute(origin, destination Location) (*Route, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%.6f,%.6f-%.6f,%.6f",
		origin.Latitude, origin.Longitude,
		destination.Latitude, destination.Longitude)

	if cached, exists := rs.cache[cacheKey]; exists && time.Now().Before(cached.ExpiresAt) {
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Route cache hit\",\"cache_key\":\"%s\"}", cacheKey)
		return cached.Route, nil
	}

	// Create directions request
	request := &maps.DirectionsRequest{
		Origin:        fmt.Sprintf("%.6f,%.6f", origin.Latitude, origin.Longitude),
		Destination:   fmt.Sprintf("%.6f,%.6f", destination.Latitude, destination.Longitude),
		Mode:          maps.TravelModeDriving,
		DepartureTime: "now",
		Units:         maps.UnitsImperial,
	}

	// Make API call
	routes, _, err := rs.client.Directions(context.Background(), request)
	if err != nil {
		return nil, fmt.Errorf("failed to get directions: %w", err)
	}

	if len(routes) == 0 {
		return nil, fmt.Errorf("no routes found")
	}

	// Parse the first route
	route := rs.parseRoute(routes[0], origin, destination)

	// Cache the result (expires in 1 hour)
	rs.cache[cacheKey] = &CachedRoute{
		Route:     route,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Route calculated\",\"distance\":%.2f,\"duration\":%d}",
		route.TotalDistance, route.TotalDuration)

	return route, nil
}

// GetOptimizedRoute calculates an optimized route with waypoints
func (rs *RouteService) GetOptimizedRoute(waypoints []Location) (*Route, error) {
	if len(waypoints) < 2 {
		return nil, fmt.Errorf("at least 2 waypoints required")
	}

	// Check cache first
	cacheKey := rs.generateWaypointCacheKey(waypoints)
	if cached, exists := rs.cache[cacheKey]; exists && time.Now().Before(cached.ExpiresAt) {
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Optimized route cache hit\",\"cache_key\":\"%s\"}", cacheKey)
		return cached.Route, nil
	}

	// Create waypoint strings
	waypointStrings := make([]string, len(waypoints)-2) // Exclude start and end
	for i, wp := range waypoints[1 : len(waypoints)-1] {
		waypointStrings[i] = fmt.Sprintf("%.6f,%.6f", wp.Latitude, wp.Longitude)
	}

	// Create directions request with waypoints
	request := &maps.DirectionsRequest{
		Origin:        fmt.Sprintf("%.6f,%.6f", waypoints[0].Latitude, waypoints[0].Longitude),
		Destination:   fmt.Sprintf("%.6f,%.6f", waypoints[len(waypoints)-1].Latitude, waypoints[len(waypoints)-1].Longitude),
		Waypoints:     waypointStrings,
		Mode:          maps.TravelModeDriving,
		DepartureTime: "now",
		Units:         maps.UnitsImperial,
		Optimize:      true, // Optimize waypoint order
	}

	// Make API call
	routes, _, err := rs.client.Directions(context.Background(), request)
	if err != nil {
		return nil, fmt.Errorf("failed to get optimized directions: %w", err)
	}

	if len(routes) == 0 {
		return nil, fmt.Errorf("no optimized routes found")
	}

	// Parse the first route
	route := rs.parseRoute(routes[0], waypoints[0], waypoints[len(waypoints)-1])
	route.Waypoints = waypoints

	// Cache the result (expires in 1 hour)
	rs.cache[cacheKey] = &CachedRoute{
		Route:     route,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Optimized route calculated\",\"distance\":%.2f,\"duration\":%d,\"waypoints\":%d}",
		route.TotalDistance, route.TotalDuration, len(waypoints))

	return route, nil
}

// CalculateDistance calculates the straight-line distance between two points
func (rs *RouteService) CalculateDistance(point1, point2 Location) float64 {
	return rs.haversineDistance(point1.Latitude, point1.Longitude, point2.Latitude, point2.Longitude)
}

// CalculateRouteOverlap calculates the overlap percentage between two routes
func (rs *RouteService) CalculateRouteOverlap(route1, route2 *Route) (*RouteOverlap, error) {
	// Extract road segments from both routes
	segments1 := rs.extractRoadSegments(route1)
	segments2 := rs.extractRoadSegments(route2)

	// Find common segments
	commonSegments := rs.findCommonSegments(segments1, segments2)

	// Calculate overlap metrics
	totalOverlapMiles := 0.0
	for _, segment := range commonSegments {
		totalOverlapMiles += segment.Distance
	}

	totalUniqueSegments := len(segments1) + len(segments2) - len(commonSegments)
	overlapPercentage := 0.0
	if totalUniqueSegments > 0 {
		overlapPercentage = float64(len(commonSegments)) / float64(totalUniqueSegments) * 100
	}

	return &RouteOverlap{
		OverlapPercentage: overlapPercentage,
		CommonSegments:    commonSegments,
		TotalOverlapMiles: totalOverlapMiles,
	}, nil
}

// Helper methods

func (rs *RouteService) parseRoute(route maps.Route, origin, destination Location) *Route {
	totalDistance := 0.0
	totalDuration := 0
	var steps []Step

	for _, leg := range route.Legs {
		totalDistance += float64(leg.Distance.Meters) * 0.000621371 // Convert meters to miles
		totalDuration += int(leg.Duration.Seconds())

		for _, step := range leg.Steps {
			stepDistance := float64(step.Distance.Meters) * 0.000621371
			stepDuration := int(step.Duration.Seconds())

			steps = append(steps, Step{
				Distance:      stepDistance,
				Duration:      stepDuration,
				Instruction:   step.HTMLInstructions,
				StartLocation: Location{Latitude: step.StartLocation.Lat, Longitude: step.StartLocation.Lng},
				EndLocation:   Location{Latitude: step.EndLocation.Lat, Longitude: step.EndLocation.Lng},
				Polyline:      step.Polyline.Points,
			})
		}
	}

	return &Route{
		TotalDistance: totalDistance,
		TotalDuration: totalDuration,
		Steps:         steps,
		Polyline:      route.OverviewPolyline.Points,
		StartLocation: origin,
		EndLocation:   destination,
		CreatedAt:     time.Now(),
	}
}

func (rs *RouteService) extractRoadSegments(route *Route) []Segment {
	var segments []Segment

	for _, step := range route.Steps {
		segments = append(segments, Segment{
			StartLocation: step.StartLocation,
			EndLocation:   step.EndLocation,
			Distance:      step.Distance,
			RoadName:      step.Instruction,
		})
	}

	return segments
}

func (rs *RouteService) findCommonSegments(segments1, segments2 []Segment) []Segment {
	var commonSegments []Segment

	for _, seg1 := range segments1 {
		for _, seg2 := range segments2 {
			if rs.segmentsOverlap(seg1, seg2) {
				commonSegments = append(commonSegments, seg1)
				break
			}
		}
	}

	return commonSegments
}

func (rs *RouteService) segmentsOverlap(seg1, seg2 Segment) bool {
	// Check if segments are within 50 meters of each other
	const threshold = 0.0310686 // ~50 meters in degrees

	latDiff1 := math.Abs(seg1.StartLocation.Latitude - seg2.StartLocation.Latitude)
	lngDiff1 := math.Abs(seg1.StartLocation.Longitude - seg2.StartLocation.Longitude)
	latDiff2 := math.Abs(seg1.EndLocation.Latitude - seg2.EndLocation.Latitude)
	lngDiff2 := math.Abs(seg1.EndLocation.Longitude - seg2.EndLocation.Longitude)

	return (latDiff1 < threshold && lngDiff1 < threshold) ||
		(latDiff2 < threshold && lngDiff2 < threshold)
}

func (rs *RouteService) generateWaypointCacheKey(waypoints []Location) string {
	key := ""
	for _, wp := range waypoints {
		key += fmt.Sprintf("%.6f,%.6f-", wp.Latitude, wp.Longitude)
	}
	return key[:len(key)-1] // Remove trailing dash
}

func (rs *RouteService) haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
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
