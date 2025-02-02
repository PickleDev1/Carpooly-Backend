package repository

import (
	"car-backend/pkg/models"
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/google/uuid"
)

type CarpoolScheduleRepository struct {
	db *sql.DB
}

func NewCarpoolScheduleRepository(db *sql.DB) *CarpoolScheduleRepository {
	return &CarpoolScheduleRepository{db: db}
}

func (r *CarpoolScheduleRepository) CreateCarpoolSchedule(ctx context.Context, schedule *models.CarpoolSchedule) error {
	query := `
		INSERT INTO carpool_schedules (
			id, carpool_id, schedule_type, start_date, end_date,
			day_of_week, start_time, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
	`

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Creating carpool schedule\",\"schedule\":%+v}", schedule)

	_, err := r.db.ExecContext(ctx, query,
		schedule.ID,
		schedule.CarpoolID,
		schedule.ScheduleType,
		schedule.StartDate,
		schedule.EndDate,
		schedule.DayOfWeek,
		schedule.StartTime,
	)

	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Database error\",\"error\":\"%v\",\"query\":%q,\"values\":[\"%v\",\"%v\",\"%v\",\"%v\",\"%v\",\"%v\",\"%v\"]}",
			err, query,
			schedule.ID,
			schedule.CarpoolID,
			schedule.ScheduleType,
			schedule.StartDate,
			schedule.EndDate,
			schedule.DayOfWeek,
			schedule.StartTime,
		)
		return fmt.Errorf("failed to create carpool schedule: %v", err)
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Successfully created carpool schedule\",\"id\":\"%s\"}", schedule.ID)
	return nil
}

func (r *CarpoolScheduleRepository) GetCarpoolSchedules(ctx context.Context, carpoolID uuid.UUID) ([]models.CarpoolSchedule, error) {
	query := `
		SELECT id, carpool_id, schedule_type, start_date, end_date,
			   day_of_week, start_time, created_at, updated_at
		FROM carpool_schedules
		WHERE carpool_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, carpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to query schedules\",\"error\":\"%v\"}", err)
		return nil, fmt.Errorf("failed to query schedules: %v", err)
	}
	defer rows.Close()

	var schedules []models.CarpoolSchedule
	for rows.Next() {
		var schedule models.CarpoolSchedule
		err := rows.Scan(
			&schedule.ID,
			&schedule.CarpoolID,
			&schedule.ScheduleType,
			&schedule.StartDate,
			&schedule.EndDate,
			&schedule.DayOfWeek,
			&schedule.StartTime,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to scan schedule\",\"error\":\"%v\"}", err)
			return nil, fmt.Errorf("failed to scan schedule: %v", err)
		}
		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

func (r *CarpoolScheduleRepository) GetScheduleByID(ctx context.Context, scheduleID uuid.UUID) (*models.CarpoolSchedule, error) {
	query := `
		SELECT id, carpool_id, schedule_type, start_date, end_date,
			   day_of_week, start_time, created_at, updated_at
		FROM carpool_schedules
		WHERE id = $1
	`

	schedule := &models.CarpoolSchedule{}
	err := r.db.QueryRowContext(ctx, query, scheduleID).Scan(
		&schedule.ID,
		&schedule.CarpoolID,
		&schedule.ScheduleType,
		&schedule.StartDate,
		&schedule.EndDate,
		&schedule.DayOfWeek,
		&schedule.StartTime,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %v", err)
	}

	return schedule, nil
}
