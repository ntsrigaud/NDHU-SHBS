-- 000003_ai_fields.sql
-- Add AI processing state and more precise status values.

-- Add ai_processed flag to track which listings still need background analysis.
ALTER TABLE book_listings ADD COLUMN IF NOT EXISTS ai_processed BOOLEAN NOT NULL DEFAULT FALSE;

-- The roadmap uses 'condition_score' for the numeric confidence (0.000-1.000).
-- The current schema has 'ai_confidence', we will keep both for compatibility or just use ai_confidence.
-- Actually, let's align with the roadmap.
ALTER TABLE book_listings ADD COLUMN IF NOT EXISTS condition_score NUMERIC(4,3);

-- Update status constraint to include 'pending' for listings awaiting AI or review.
ALTER TABLE book_listings DROP CONSTRAINT IF EXISTS book_listings_status_check;
ALTER TABLE book_listings ADD CONSTRAINT book_listings_status_check 
    CHECK (status IN ('active', 'pending', 'reserved', 'sold', 'delisted'));

-- Set default status for new listings to 'pending' if they need AI processing.
-- Note: existing code might expect 'active'. We will handle this in Go logic.
