-- 018: Add acted_at and indexes to match_requests (safe, idempotent)

-- Column
ALTER TABLE match_requests
ADD COLUMN IF NOT EXISTS acted_at TIMESTAMPTZ NULL;

-- Indexes for list views
CREATE INDEX IF NOT EXISTS idx_mr_to_user ON match_requests(to_user_id, status);
CREATE INDEX IF NOT EXISTS idx_mr_from_user ON match_requests(from_user_id, status);
CREATE INDEX IF NOT EXISTS idx_mr_created_at ON match_requests(created_at);

-- Optional dedupe: prevent duplicate pending requests for same pair
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE c.relname = 'uniq_mr_pending_pair'
    ) THEN
        CREATE UNIQUE INDEX uniq_mr_pending_pair
        ON match_requests(from_user_id, to_user_id)
        WHERE status = 'pending';
    END IF;
END $$;


