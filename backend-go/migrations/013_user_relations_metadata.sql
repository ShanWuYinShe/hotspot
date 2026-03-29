-- 013_user_relations_metadata.sql
-- Add timestamps and reason metadata required by users spec.

ALTER TABLE user_relations ADD COLUMN IF NOT EXISTS followed_at TIMESTAMPTZ;
ALTER TABLE user_relations ADD COLUMN IF NOT EXISTS blocked_at TIMESTAMPTZ;
ALTER TABLE user_relations ADD COLUMN IF NOT EXISTS block_reason TEXT;

UPDATE user_relations
SET followed_at = COALESCE(followed_at, NOW())
WHERE is_following = TRUE;

UPDATE user_relations
SET blocked_at = COALESCE(blocked_at, NOW())
WHERE is_blocked = TRUE;

CREATE INDEX IF NOT EXISTS idx_user_relations_user_followed_at
  ON user_relations(user_id, followed_at DESC, target_user_id ASC)
  WHERE is_following = TRUE;

CREATE INDEX IF NOT EXISTS idx_user_relations_target_followed_at
  ON user_relations(target_user_id, followed_at DESC, user_id ASC)
  WHERE is_following = TRUE;

CREATE INDEX IF NOT EXISTS idx_user_relations_user_blocked_at
  ON user_relations(user_id, blocked_at DESC, target_user_id ASC)
  WHERE is_blocked = TRUE;
