-- backend-go 初始迁移：用户与会话表（无 PostGIS）

CREATE TABLE IF NOT EXISTS users (
    user_id       TEXT PRIMARY KEY,
    login_id      TEXT UNIQUE,
    nickname      TEXT        NOT NULL,
    password_hash TEXT,
    user_type     TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_login_id
    ON users(login_id);

CREATE TABLE IF NOT EXISTS auth_sessions (
    session_id    TEXT PRIMARY KEY,
    user_id       TEXT        NOT NULL,
    device_type   TEXT,
    device_name   TEXT,
    ip_address    TEXT,
    ip_location   TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at    TIMESTAMPTZ,
    CONSTRAINT fk_auth_sessions_user
        FOREIGN KEY (user_id) REFERENCES users(user_id)
);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_user
    ON auth_sessions(user_id);
