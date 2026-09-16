
CREATE TABLE IF NOT EXISTS competitors (
    id SERIAL PRIMARY KEY,
    project_name VARCHAR(255) NOT NULL,
    district VARCHAR(255) NOT NULL,
    price_per_sqm BIGINT,
    advantages TEXT,
    disadvantages TEXT,
    CONSTRAINT competitors_price_nonnegative
        CHECK (price_per_sqm IS NULL OR price_per_sqm >= 0)
);
CREATE INDEX IF NOT EXISTS competitors_district_idx ON competitors (LOWER(district));

CREATE TABLE IF NOT EXISTS recommendations (
    id SERIAL PRIMARY KEY,
    deal_id INT NOT NULL REFERENCES deals(id) ON DELETE CASCADE,
    kind VARCHAR(100) NOT NULL,
    recommendation TEXT NOT NULL,
    request_id TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS recommendations_deal_id_idx ON recommendations (deal_id);
