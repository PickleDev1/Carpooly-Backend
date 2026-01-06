package services

import (
	"car-backend/pkg/models"
	"context"
	"time"
)

// MatchingService interface for enhanced matching algorithm
type MatchingService interface {
	CalculateMatchScore(user1, user2 *models.UserProfile) *models.MatchScore
	FindPotentialMatches(userID string, filters *models.MatchFilters) ([]*models.EnhancedPotentialMatch, error)
	UpdateMatchScores(userID string) error
	GetUserProfile(userID string) (*models.UserProfile, error)
	GetAvailableUsers() ([]*models.UserProfile, error)
}

// MatchingServiceImpl implements MatchingService
type MatchingServiceImpl struct {
	userRepo        UserRepository
	matchingRepo    MatchingRepository
	userProfileRepo UserProfileRepository
}

// UserRepository interface for user operations
type UserRepository interface {
	GetUserByID(ctx context.Context, userID string) (*models.User, error)
}

// MatchingRepository interface for matching operations
type MatchingRepository interface {
	GetPotentialMatches(ctx context.Context, userID string, status string) ([]*models.PotentialMatch, error)
	CreatePotentialMatch(ctx context.Context, match *models.PotentialMatch) error
	UpsertMatchScore(ctx context.Context, score *models.MatchScore) error
}

// UserProfileRepository interface for user profile operations
type UserProfileRepository interface {
	GetUserProfile(ctx context.Context, userID string) (*models.UserProfile, error)
	UpsertUserProfile(ctx context.Context, profile *models.UserProfile) error
	GetAvailableUsers(ctx context.Context) ([]*models.UserProfile, error)
}

// NewMatchingService creates a new matching service
func NewMatchingService(
	userRepo UserRepository,
	matchingRepo MatchingRepository,
	userProfileRepo UserProfileRepository,
) MatchingService {
	return &MatchingServiceImpl{
		userRepo:        userRepo,
		matchingRepo:    matchingRepo,
		userProfileRepo: userProfileRepo,
	}
}

// CalculateMatchScore calculates comprehensive compatibility score between two users
func (s *MatchingServiceImpl) CalculateMatchScore(user1, user2 *models.UserProfile) *models.MatchScore {
	// Calculate individual scores
	locationScore := s.calculateLocationScore(user1, user2)
	scheduleScore := s.calculateScheduleScore(user1, user2)
	demographicScore := s.calculateDemographicScore(user1, user2)
	routeScore := s.calculateRouteScore(user1, user2)
	groupSizeScore := s.calculateGroupSizeScore(user1, user2)
	roleCompatibilityScore := s.calculateRoleCompatibilityScore(user1, user2)

	// Calculate weighted total score
	totalScore := locationScore.score*0.25 +
		scheduleScore.score*0.25 +
		demographicScore.score*0.20 +
		routeScore.score*0.15 +
		groupSizeScore.score*0.10 +
		roleCompatibilityScore.score*0.05

	// Combine all reasons and dealbreakers
	allReasons := []string{}
	allDealbreakers := []string{}

	allReasons = append(allReasons, locationScore.reasons...)
	allReasons = append(allReasons, scheduleScore.reasons...)
	allReasons = append(allReasons, demographicScore.reasons...)
	allReasons = append(allReasons, routeScore.reasons...)
	allReasons = append(allReasons, groupSizeScore.reasons...)
	allReasons = append(allReasons, roleCompatibilityScore.reasons...)

	allDealbreakers = append(allDealbreakers, locationScore.dealbreakers...)
	allDealbreakers = append(allDealbreakers, scheduleScore.dealbreakers...)
	allDealbreakers = append(allDealbreakers, demographicScore.dealbreakers...)
	allDealbreakers = append(allDealbreakers, routeScore.dealbreakers...)
	allDealbreakers = append(allDealbreakers, groupSizeScore.dealbreakers...)
	allDealbreakers = append(allDealbreakers, roleCompatibilityScore.dealbreakers...)

	return &models.MatchScore{
		UserID:                 user1.UserID,
		PotentialMatchID:       user2.UserID,
		TotalScore:             totalScore,
		LocationScore:          locationScore.score,
		ScheduleScore:          scheduleScore.score,
		DemographicScore:       demographicScore.score,
		RouteScore:             routeScore.score,
		GroupSizeScore:         groupSizeScore.score,
		RoleCompatibilityScore: roleCompatibilityScore.score,
		MatchReasons:           allReasons,
		Dealbreakers:           allDealbreakers,
	}
}

// ScoreResult represents a score with reasons and dealbreakers
type ScoreResult struct {
	score        float64
	reasons      []string
	dealbreakers []string
}

// calculateLocationScore calculates location compatibility (25% weight)
func (s *MatchingServiceImpl) calculateLocationScore(user1, user2 *models.UserProfile) ScoreResult {
	// For now, return a placeholder score
	// In production, this would calculate actual distances using Haversine formula
	score := 0.8 // Placeholder score
	reasons := []string{"Close pickup location", "Good route overlap"}
	dealbreakers := []string{}

	return ScoreResult{score: score, reasons: reasons, dealbreakers: dealbreakers}
}

// calculateScheduleScore calculates schedule compatibility (25% weight)
func (s *MatchingServiceImpl) calculateScheduleScore(user1, user2 *models.UserProfile) ScoreResult {
	// For now, return a placeholder score
	// In production, this would compare actual schedules
	score := 0.7 // Placeholder score
	reasons := []string{"Similar departure time", "Flexible schedules"}
	dealbreakers := []string{}

	return ScoreResult{score: score, reasons: reasons, dealbreakers: dealbreakers}
}

// calculateDemographicScore calculates demographic compatibility (20% weight)
func (s *MatchingServiceImpl) calculateDemographicScore(user1, user2 *models.UserProfile) ScoreResult {
	// For now, return a placeholder score
	// In production, this would check actual demographic preferences
	score := 0.9 // Placeholder score
	reasons := []string{"Age preference match", "Gender preference match"}
	dealbreakers := []string{}

	return ScoreResult{score: score, reasons: reasons, dealbreakers: dealbreakers}
}

// calculateRouteScore calculates route compatibility (15% weight)
func (s *MatchingServiceImpl) calculateRouteScore(user1, user2 *models.UserProfile) ScoreResult {
	// For now, return a placeholder score
	// In production, this would calculate actual route metrics
	score := 0.6 // Placeholder score
	reasons := []string{"Reasonable detour time"}
	dealbreakers := []string{}

	return ScoreResult{score: score, reasons: reasons, dealbreakers: dealbreakers}
}

// calculateGroupSizeScore calculates group size compatibility (10% weight)
func (s *MatchingServiceImpl) calculateGroupSizeScore(user1, user2 *models.UserProfile) ScoreResult {
	// For now, return a placeholder score
	// In production, this would check actual group size preferences
	score := 0.8 // Placeholder score
	reasons := []string{"Group size preference match"}
	dealbreakers := []string{}

	return ScoreResult{score: score, reasons: reasons, dealbreakers: dealbreakers}
}

// calculateRoleCompatibilityScore calculates role compatibility (5% weight)
func (s *MatchingServiceImpl) calculateRoleCompatibilityScore(user1, user2 *models.UserProfile) ScoreResult {
	// For now, return a placeholder score
	// In production, this would check actual driver/passenger preferences
	score := 0.9 // Placeholder score
	reasons := []string{"Role preference match"}
	dealbreakers := []string{}

	return ScoreResult{score: score, reasons: reasons, dealbreakers: dealbreakers}
}

// FindPotentialMatches finds potential matches with enhanced scoring
func (s *MatchingServiceImpl) FindPotentialMatches(userID string, filters *models.MatchFilters) ([]*models.EnhancedPotentialMatch, error) {
	// This is a placeholder implementation
	// In production, this would use the actual matching algorithm
	return []*models.EnhancedPotentialMatch{}, nil
}

// UpdateMatchScores updates match scores for a user
func (s *MatchingServiceImpl) UpdateMatchScores(userID string) error {
	// This is a placeholder implementation
	// In production, this would recalculate all match scores
	return nil
}

// GetUserProfile gets a user's enhanced profile
func (s *MatchingServiceImpl) GetUserProfile(userID string) (*models.UserProfile, error) {
	// This is a placeholder implementation
	// In production, this would fetch from the database
	return &models.UserProfile{
		UserID: userID,
		Schedule: models.Schedule{
			DepartureTime:      "08:30",
			Frequency:          "daily",
			FlexibilityMinutes: 30,
			DaysOfWeek:         []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
		},
		CurrentGroupSize:       1,
		IsAvailableForMatching: true,
		LastActive:             time.Now(),
	}, nil
}

// GetAvailableUsers gets all users available for matching
func (s *MatchingServiceImpl) GetAvailableUsers() ([]*models.UserProfile, error) {
	// This is a placeholder implementation
	// In production, this would fetch from the database
	return []*models.UserProfile{}, nil
}
