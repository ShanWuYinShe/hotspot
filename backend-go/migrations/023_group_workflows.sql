CREATE TABLE IF NOT EXISTS group_applications (
  application_id TEXT PRIMARY KEY,
  group_id TEXT NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
  applicant_user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  message TEXT,
  status TEXT NOT NULL DEFAULT 'pending',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  handled_at TIMESTAMPTZ,
  handled_by_user_id TEXT REFERENCES users(user_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_group_applications_group_created_at
  ON group_applications(group_id, created_at DESC, application_id DESC);
CREATE INDEX IF NOT EXISTS idx_group_applications_applicant_created_at
  ON group_applications(applicant_user_id, created_at DESC, application_id DESC);

CREATE TABLE IF NOT EXISTS group_invites (
  invite_id TEXT PRIMARY KEY,
  group_id TEXT NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
  inviter_user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  invitee_user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  message TEXT,
  status TEXT NOT NULL DEFAULT 'pending',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMPTZ,
  responded_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_group_invites_group_created_at
  ON group_invites(group_id, created_at DESC, invite_id DESC);
CREATE INDEX IF NOT EXISTS idx_group_invites_invitee_created_at
  ON group_invites(invitee_user_id, created_at DESC, invite_id DESC);
