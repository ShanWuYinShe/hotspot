ALTER TABLE users
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'normal',
  ADD COLUMN IF NOT EXISTS banned_reason TEXT,
  ADD COLUMN IF NOT EXISTS banned_duration_days INTEGER,
  ADD COLUMN IF NOT EXISTS banned_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS register_ip TEXT,
  ADD COLUMN IF NOT EXISTS register_location TEXT,
  ADD COLUMN IF NOT EXISTS last_login_ip TEXT,
  ADD COLUMN IF NOT EXISTS last_login_location TEXT;

ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS owner_id TEXT REFERENCES users(user_id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS group_members (
  group_id TEXT NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  role TEXT NOT NULL DEFAULT 'member',
  joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (group_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_group_members_user_id ON group_members(user_id);
CREATE INDEX IF NOT EXISTS idx_groups_owner_created_at ON groups(owner_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_groups_active_created_at ON groups(created_at DESC, group_id ASC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_status_created_at ON users(status, created_at DESC, user_id ASC);

INSERT INTO group_members (group_id, user_id, role, joined_at)
SELECT g.group_id,
       COALESCE(g.owner_id, 'admin'),
       'owner',
       COALESCE(g.created_at, NOW())
FROM groups g
WHERE g.owner_id IS NOT NULL
ON CONFLICT (group_id, user_id) DO NOTHING;

ALTER TABLE reports
  ADD COLUMN IF NOT EXISTS handled_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS handled_by_user_id TEXT REFERENCES users(user_id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS handle_note TEXT;

CREATE INDEX IF NOT EXISTS idx_reports_status_created_at ON reports(status, created_at DESC, report_id DESC);
CREATE INDEX IF NOT EXISTS idx_reports_target_type_created_at ON reports(target_type, created_at DESC, report_id DESC);

CREATE TABLE IF NOT EXISTS admin_audit_logs (
  log_id TEXT PRIMARY KEY,
  action TEXT NOT NULL,
  operator_user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  target_type TEXT NOT NULL,
  target_id TEXT NOT NULL,
  target_name TEXT,
  details JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_created_at ON admin_audit_logs(created_at DESC, log_id DESC);
CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_target ON admin_audit_logs(target_type, target_id, created_at DESC);
