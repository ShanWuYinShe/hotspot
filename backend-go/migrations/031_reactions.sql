-- 031_reactions.sql
-- Add unified emoji reactions for message / moment / comment targets.
--
-- Polymorphic by design: target_type discriminates which resource target_id
-- points at ('message' | 'moment' | 'comment'). We intentionally do NOT add
-- FK constraints on target_id because it references different tables per row;
-- referential integrity is enforced at the service layer instead.
--
-- Cardinality rule (spec api_spec_reaction.md): one user has at most one
-- reaction row per target — switching emoji updates the existing row rather
-- than inserting a new one. The PRIMARY KEY encodes this invariant.

CREATE TABLE IF NOT EXISTS reactions (
  target_type TEXT NOT NULL,
  target_id   TEXT NOT NULL,
  user_id     TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  emoji       TEXT NOT NULL,
  reacted_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (target_type, target_id, user_id)
);

-- List reactions for a single target, ordered by reacted_at DESC (most recent
-- first), user_id ASC as a stable tie-breaker. Backs reactions/list.
CREATE INDEX IF NOT EXISTS idx_reactions_target_reacted_at
  ON reactions(target_type, target_id, reacted_at DESC, user_id ASC);

-- "What did this user react to?" lookups (e.g. users/list_reacted_moments).
CREATE INDEX IF NOT EXISTS idx_reactions_user_reacted_at
  ON reactions(user_id, reacted_at DESC, target_id DESC);
