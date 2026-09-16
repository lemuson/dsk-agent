
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    budget_max BIGINT CHECK (budget_max IS NULL OR budget_max >= 0),
    preferences JSONB NOT NULL DEFAULT '{}'::jsonb
);
