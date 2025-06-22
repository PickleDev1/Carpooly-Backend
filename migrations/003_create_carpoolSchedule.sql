CREATE TABLE carpool_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    carpool_id UUID NOT NULL,
    schedule_type VARCHAR(20) NOT NULL CHECK (schedule_type IN ('one_time', 'daily', 'weekly')),
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP,
    day_of_week INTEGER CHECK (day_of_week BETWEEN 0 AND 6), -- For weekly events (0 = Sunday)
    start_time TIME NOT NULL,
    repeat_interval INTEGER DEFAULT 1, -- e.g., every 1 week, every 2 weeks
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (carpool_id) REFERENCES carpools(id) ON DELETE CASCADE
);

ALTER TABLE carpool_schedules DROP COLUMN repeat_interval;

-- Fix start_time column type in carpool_schedules table
-- Change from TIME to TIMESTAMP to match Go time.Time type
-- Use start_date as the base date for the time conversion

ALTER TABLE carpool_schedules 
ALTER COLUMN start_time TYPE TIMESTAMP WITH TIME ZONE 
USING (start_date + start_time)::timestamp with time zone; 