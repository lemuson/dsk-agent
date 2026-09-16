
CREATE TYPE deal_status AS ENUM ('pending', 'contract', 'completed', 'cancelled');

CREATE TABLE IF NOT EXISTS deals (
    id SERIAL PRIMARY KEY,
    id_user INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    id_employee INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    id_apartment INT NOT NULL REFERENCES apartments(id) ON DELETE CASCADE,
    id_chat_session INT REFERENCES chat_sessions(id) ON DELETE SET NULL,
    base_price DECIMAL(15, 2) NOT NULL,
    percent_discount DECIMAL(5, 2) NOT NULL DEFAULT 0.00,
    total_price DECIMAL(15, 2) NOT NULL,
    status deal_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS deals_id_chat_session_idx ON deals (id_chat_session) WHERE id_chat_session IS NOT NULL;

CREATE TABLE IF NOT EXISTS offers (
    id SERIAL PRIMARY KEY,
    deal_id INT NOT NULL REFERENCES deals(id) ON DELETE CASCADE,
    created_by INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    base_price BIGINT NOT NULL,
    discount_percent DECIMAL(5, 2) NOT NULL,
    final_price BIGINT NOT NULL,
    generated_text TEXT NOT NULL,
    status VARCHAR(30) NOT NULL,
    approval_required BOOLEAN NOT NULL,
    approved_by INT REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMP,
    request_id VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT offers_base_price_positive CHECK (base_price > 0),
    CONSTRAINT offers_final_price_nonnegative CHECK (final_price >= 0),
    CONSTRAINT offers_discount_valid CHECK (discount_percent >= 0 AND discount_percent <= 100),
    CONSTRAINT offers_status_valid CHECK (status IN ('draft', 'pending_approval', 'approved', 'rejected'))
);

CREATE INDEX IF NOT EXISTS offers_deal_id_idx ON offers (deal_id);
