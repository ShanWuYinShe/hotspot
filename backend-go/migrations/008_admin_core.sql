CREATE TABLE IF NOT EXISTS admin_roles (
  role_id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  description TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS admin_role_permissions (
  role_id TEXT NOT NULL REFERENCES admin_roles(role_id) ON DELETE CASCADE,
  permission_key TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (role_id, permission_key)
);

CREATE TABLE IF NOT EXISTS admin_users (
  user_id TEXT PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
  grant_all_permissions BOOLEAN NOT NULL DEFAULT FALSE,
  granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  revoked_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS admin_user_roles (
  user_id TEXT NOT NULL REFERENCES admin_users(user_id) ON DELETE CASCADE,
  role_id TEXT NOT NULL REFERENCES admin_roles(role_id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_admin_roles_active_name ON admin_roles(name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_admin_users_granted_at ON admin_users(granted_at DESC, user_id ASC) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_admin_user_roles_role_id ON admin_user_roles(role_id);
