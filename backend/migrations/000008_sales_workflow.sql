
CREATE TABLE IF NOT EXISTS discount_policies (
    id SERIAL PRIMARY KEY,
    building_id INT NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('manager', 'supervisor')),
    max_discount_percent DECIMAL(5, 2) NOT NULL CHECK (max_discount_percent >= 0 AND max_discount_percent <= 100),
    version INT NOT NULL CHECK (version > 0),
    valid_from TIMESTAMP NOT NULL,
    valid_to TIMESTAMP,
    created_by INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT discount_policies_valid_period CHECK (valid_to IS NULL OR valid_to > valid_from),
    UNIQUE (building_id, role, version)
);
CREATE INDEX IF NOT EXISTS discount_policies_active_idx
    ON discount_policies (building_id, role, valid_from DESC, version DESC);

ALTER TABLE buildings ADD COLUMN IF NOT EXISTS readiness_percent INT CHECK (readiness_percent IS NULL OR readiness_percent BETWEEN 0 AND 100);
ALTER TABLE buildings ADD COLUMN IF NOT EXISTS forecast_date DATE;
ALTER TABLE buildings ADD COLUMN IF NOT EXISTS delivery_shift_days INT CHECK (delivery_shift_days IS NULL OR delivery_shift_days >= 0);

CREATE TABLE IF NOT EXISTS ancillary_units (
    id SERIAL PRIMARY KEY,
    building_id INT NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    kind VARCHAR(30) NOT NULL CHECK (kind IN ('parking', 'storage')),
    number VARCHAR(50) NOT NULL,
    area DECIMAL(10, 2),
    price BIGINT NOT NULL CHECK (price >= 0),
    status apartment_status NOT NULL DEFAULT 'free',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (building_id, kind, number)
);
CREATE INDEX IF NOT EXISTS ancillary_units_building_idx ON ancillary_units (building_id, kind, status);

ALTER TABLE offers ADD COLUMN IF NOT EXISTS parking_unit_id INT REFERENCES ancillary_units(id) ON DELETE SET NULL;
ALTER TABLE offers ADD COLUMN IF NOT EXISTS storage_unit_id INT REFERENCES ancillary_units(id) ON DELETE SET NULL;
ALTER TABLE offers ADD COLUMN IF NOT EXISTS rejected_by INT REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE offers ADD COLUMN IF NOT EXISTS rejected_at TIMESTAMP;
ALTER TABLE offers ADD COLUMN IF NOT EXISTS rejection_reason VARCHAR(500);
ALTER TABLE offers ADD COLUMN IF NOT EXISTS version INT;
WITH ranked_offers AS (
    SELECT id, ROW_NUMBER() OVER (PARTITION BY deal_id ORDER BY created_at, id)::INT AS version
    FROM offers
)
UPDATE offers SET version = ranked_offers.version
FROM ranked_offers
WHERE offers.id = ranked_offers.id AND offers.version IS NULL;
ALTER TABLE offers ALTER COLUMN version SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS offers_deal_version_uidx ON offers (deal_id, version);

CREATE TABLE IF NOT EXISTS offer_documents (
    id SERIAL PRIMARY KEY,
    offer_id INT NOT NULL UNIQUE REFERENCES offers(id) ON DELETE CASCADE,
    content BYTEA NOT NULL,
    content_type VARCHAR(100) NOT NULL DEFAULT 'application/pdf',
    checksum_sha256 VARCHAR(64) NOT NULL,
    generated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS offer_deliveries (
    id SERIAL PRIMARY KEY,
    offer_id INT NOT NULL REFERENCES offers(id) ON DELETE CASCADE,
    recipient VARCHAR(255) NOT NULL,
    channel VARCHAR(30) NOT NULL DEFAULT 'email' CHECK (channel IN ('email')),
    status VARCHAR(30) NOT NULL CHECK (status IN ('pending', 'sent', 'failed')),
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sent_at TIMESTAMP
);
CREATE INDEX IF NOT EXISTS offer_deliveries_offer_idx ON offer_deliveries (offer_id, created_at DESC);

ALTER TABLE competitors ADD COLUMN IF NOT EXISTS source_url TEXT;
ALTER TABLE competitors ADD COLUMN IF NOT EXISTS observed_at TIMESTAMP;
ALTER TABLE competitors ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE competitors ADD COLUMN IF NOT EXISTS rooms INT CHECK (rooms IS NULL OR rooms >= 0);
ALTER TABLE competitors ADD COLUMN IF NOT EXISTS area DECIMAL(10, 2) CHECK (area IS NULL OR area > 0);

CREATE TABLE IF NOT EXISTS erp_events (
    id SERIAL PRIMARY KEY,
    building_id INT NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    kind VARCHAR(40) NOT NULL CHECK (kind IN ('schedule', 'supply', 'material', 'project_change')),
    title VARCHAR(255) NOT NULL,
    details TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('low', 'medium', 'high')),
    affects_delivery BOOLEAN NOT NULL DEFAULT FALSE,
    delay_days INT CHECK (delay_days IS NULL OR delay_days >= 0),
    occurred_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS erp_events_building_idx ON erp_events (building_id, occurred_at DESC);

CREATE TABLE IF NOT EXISTS material_stocks (
    id SERIAL PRIMARY KEY,
    building_id INT NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    material_name VARCHAR(255) NOT NULL,
    quantity DECIMAL(15, 3) NOT NULL CHECK (quantity >= 0),
    unit VARCHAR(30) NOT NULL,
    minimum_quantity DECIMAL(15, 3) NOT NULL DEFAULT 0 CHECK (minimum_quantity >= 0),
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (building_id, material_name)
);

CREATE TABLE IF NOT EXISTS production_schedules (
    id SERIAL PRIMARY KEY,
    building_id INT NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    product_name VARCHAR(255) NOT NULL,
    planned_quantity INT NOT NULL CHECK (planned_quantity >= 0),
    produced_quantity INT NOT NULL DEFAULT 0 CHECK (produced_quantity >= 0),
    planned_date DATE NOT NULL,
    status VARCHAR(30) NOT NULL CHECK (status IN ('planned', 'in_progress', 'completed', 'delayed')),
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS production_schedules_building_idx
    ON production_schedules (building_id, planned_date);

CREATE TABLE IF NOT EXISTS ai_audit_log (
    id BIGSERIAL PRIMARY KEY,
    actor_id INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    deal_id INT REFERENCES deals(id) ON DELETE SET NULL,
    chat_session_id INT REFERENCES chat_sessions(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    request_text TEXT NOT NULL,
    response_text TEXT,
    agent VARCHAR(50),
    intent VARCHAR(100),
    status VARCHAR(20) NOT NULL CHECK (status IN ('success', 'failed')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS ai_audit_actor_idx ON ai_audit_log (actor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS ai_audit_deal_idx ON ai_audit_log (deal_id, created_at DESC);

CREATE TABLE IF NOT EXISTS staff_reminders (
    id BIGSERIAL PRIMARY KEY,
    assigned_to INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_by INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    deal_id INT REFERENCES deals(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    due_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS staff_reminders_assignee_idx
    ON staff_reminders (assigned_to, completed_at, due_at);
