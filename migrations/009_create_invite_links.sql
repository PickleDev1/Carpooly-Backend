-- Create invite_links table
CREATE TABLE invite_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    carpool_id UUID NOT NULL REFERENCES carpools(id) ON DELETE CASCADE,
    invite_code VARCHAR(255) NOT NULL UNIQUE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT true,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    max_uses INTEGER DEFAULT -1, -- -1 means unlimited
    current_uses INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index for fast invite code lookups
CREATE INDEX idx_invite_links_code ON invite_links(invite_code);

-- Create index for carpool lookups
CREATE INDEX idx_invite_links_carpool ON invite_links(carpool_id);

-- Create index for active invites
CREATE INDEX idx_invite_links_active ON invite_links(is_active, expires_at);

-- Add trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_invite_links_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_invite_links_updated_at
    BEFORE UPDATE ON invite_links
    FOR EACH ROW
    EXECUTE FUNCTION update_invite_links_updated_at(); 