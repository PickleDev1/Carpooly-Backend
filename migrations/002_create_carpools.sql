-- First, create the base carpools table
CREATE TABLE carpools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id VARCHAR(255),
    carpool_name VARCHAR(255) NOT NULL, 
    status BOOLEAN DEFAULT TRUE, 
    recurring_option VARCHAR(50),
    available_seats INTEGER,
    destination_address TEXT, 
    seats INTEGER, 
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Then create carpool_members table (references users and carpools)
CREATE TABLE carpool_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), 
    carpool_id UUID NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (carpool_id) REFERENCES carpools(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE (carpool_id, user_id)
);

-- Create carpool_rides table (references users and carpools)
CREATE TABLE carpool_rides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    carpool_id UUID NOT NULL,
    driver_id UUID NOT NULL, 
    status INTEGER, 
    location_lat FLOAT,
    location_lng FLOAT, 
    miles_saved FLOAT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (carpool_id) REFERENCES carpools(id),
    FOREIGN KEY (driver_id) REFERENCES users(id) 
);

ALTER TABLE carpool_rides
ADD COLUMN participants JSONB DEFAULT '[]'::jsonb;

ALTER TABLE carpool_rides 
    ALTER COLUMN driver_id DROP NOT NULL,
    ALTER COLUMN location_lat DROP NOT NULL,
    ALTER COLUMN location_lng DROP NOT NULL;

    ALTER TABLE carpool_rides 
ADD COLUMN start_time TIMESTAMP WITH TIME ZONE;


-- Create carpool_stops table (references carpool_rides)
CREATE TABLE carpool_stops (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    carpool_ride_id UUID NOT NULL,
    address TEXT NOT NULL,
    stop_order INTEGER NOT NULL, 
    stop_type VARCHAR(20) CHECK (stop_type IN ('START', 'INTERMEDIATE', 'DESTINATION')),
    user_id UUID, 
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (carpool_ride_id) REFERENCES carpool_rides(id) ON DELETE CASCADE, 
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Finally, create invites table (references users and carpools)
CREATE TABLE invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_user UUID NOT NULL,
    to_user UUID NOT NULL,
    carpool_id UUID NOT NULL,
    message TEXT, 
    status INTEGER, 
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_user) REFERENCES users(id),
    FOREIGN KEY (to_user) REFERENCES users(id),
    FOREIGN KEY (carpool_id) REFERENCES carpools(id)
);

ALTER TABLE
  "public"."carpools"
ALTER COLUMN
  "creator_id"
TYPE uuid
USING "creator_id"::uuid;

ALTER TABLE
  "public"."invites"
DROP CONSTRAINT
  "invites_to_user_fkey";
  
ALTER TABLE invites 
RENAME COLUMN to_user TO to_user_email;

ALTER TABLE invites 
ALTER COLUMN to_user_email TYPE VARCHAR(255);

ALTER TABLE invites DROP COLUMN message; 




