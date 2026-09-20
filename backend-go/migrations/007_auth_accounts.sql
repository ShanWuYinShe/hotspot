CREATE TABLE IF NOT EXISTS auth_accounts (
  login_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL UNIQUE REFERENCES users(user_id) ON DELETE CASCADE,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS auth_password_reset_requests (
  request_id TEXT PRIMARY KEY,
  login_id TEXT NOT NULL REFERENCES auth_accounts(login_id) ON DELETE CASCADE,
  verification_code TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  used_at TIMESTAMPTZ NULL
);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_auth_password_reset_requests_login_id_created_at
  ON auth_password_reset_requests(login_id, created_at DESC);
