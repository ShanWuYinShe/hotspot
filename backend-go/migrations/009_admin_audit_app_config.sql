-- 编号 009 段依赖字典序执行：009_admin_audit < 009_admin_moderation。
CREATE TABLE IF NOT EXISTS admin_audit_logs (
  log_id TEXT PRIMARY KEY,
  action TEXT NOT NULL,
  operator_id TEXT NOT NULL,
  operator_name TEXT NOT NULL,
  operator_avatar_url TEXT NULL,
  target_type TEXT NOT NULL,
  target_id TEXT NOT NULL,
  target_name TEXT NULL,
  details_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  ip_location TEXT NULL,
  ip_address TEXT NULL,
  device_summary TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_created_at ON admin_audit_logs(created_at DESC, log_id DESC);
CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_operator_created_at ON admin_audit_logs(operator_id, created_at DESC, log_id DESC);
CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_target_created_at ON admin_audit_logs(target_type, target_id, created_at DESC, log_id DESC);
CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_action_created_at ON admin_audit_logs(action, created_at DESC, log_id DESC);

CREATE TABLE IF NOT EXISTS app_configs (
  config_id TEXT PRIMARY KEY,
  payload_json JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_by_user_id TEXT NULL REFERENCES users(user_id) ON DELETE SET NULL
);
