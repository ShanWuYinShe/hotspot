CREATE TABLE IF NOT EXISTS shares (
  share_id TEXT PRIMARY KEY,
  share_code TEXT NOT NULL UNIQUE,
  creator_user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  share_type TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMPTZ NULL,
  is_revoked BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_shares_creator_created_at ON shares(creator_user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS reports (
  report_id TEXT PRIMARY KEY,
  creator_user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  target_type TEXT NOT NULL,
  target_id TEXT NOT NULL,
  reason TEXT NOT NULL,
  description TEXT,
  status TEXT NOT NULL DEFAULT 'pending',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reports_creator_created_at ON reports(creator_user_id, created_at DESC);
