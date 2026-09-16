
ALTER TABLE users ADD COLUMN IF NOT EXISTS budget_max BIGINT CHECK (budget_max IS NULL OR budget_max >= 0);
ALTER TABLE users ADD COLUMN IF NOT EXISTS preferences JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE buildings ADD COLUMN IF NOT EXISTS district VARCHAR(255);
ALTER TABLE construction_progress ADD COLUMN IF NOT EXISTS risk_level VARCHAR(20) CHECK (risk_level IS NULL OR risk_level IN ('low', 'medium', 'high'));
ALTER TABLE construction_progress ADD COLUMN IF NOT EXISTS delay_days INT CHECK (delay_days IS NULL OR delay_days >= 0);
ALTER TABLE chat_sessions ADD COLUMN IF NOT EXISTS id_apartment INT REFERENCES apartments(id) ON DELETE SET NULL;
ALTER TABLE deals ADD COLUMN IF NOT EXISTS id_chat_session INT REFERENCES chat_sessions(id) ON DELETE SET NULL;

DO $$ BEGIN
    CREATE TYPE notification_type AS ENUM ('construction_delay', 'construction_risk', 'deal_update', 'general');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS notifications (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    deal_id INT REFERENCES deals(id) ON DELETE CASCADE,
    type notification_type NOT NULL DEFAULT 'general',
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    read_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS construction_notification_state (
    progress_id INT PRIMARY KEY REFERENCES construction_progress(id) ON DELETE CASCADE,
    fingerprint TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS deals_id_chat_session_idx ON deals (id_chat_session) WHERE id_chat_session IS NOT NULL;
CREATE INDEX IF NOT EXISTS notifications_user_id_idx ON notifications (user_id);
CREATE INDEX IF NOT EXISTS notifications_user_unread_idx ON notifications (user_id, is_read);
