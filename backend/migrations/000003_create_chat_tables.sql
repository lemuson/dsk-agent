
CREATE TYPE chat_session_status AS ENUM ('open', 'in_progress', 'close');

CREATE TABLE IF NOT EXISTS chat_sessions (
    id SERIAL PRIMARY KEY,
    id_user INT REFERENCES users(id) ON DELETE SET NULL,
    id_employee INT REFERENCES users(id) ON DELETE SET NULL,
    id_apartment INT REFERENCES apartments(id) ON DELETE SET NULL,
    guest_name VARCHAR(255),
    guest_email VARCHAR(255),
    guest_phone VARCHAR(50),
    status chat_session_status NOT NULL DEFAULT 'open',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS chat_session_rejections (
    id SERIAL PRIMARY KEY,
    id_chat_sessions INT NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    id_employee INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason VARCHAR(500) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS messages (
    id SERIAL PRIMARY KEY,
    id_chat_session INT NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    id_user INT REFERENCES users(id) ON DELETE SET NULL,
    sender_type VARCHAR(50) NOT NULL DEFAULT 'client',
    content TEXT NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    sended_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

