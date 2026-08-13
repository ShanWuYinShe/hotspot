-- 群成员检索开关（spec types.md GroupSettings.members_searchable）：
-- 是否允许非成员检索群成员列表（groups/members/search），默认 true。
ALTER TABLE groups ADD COLUMN IF NOT EXISTS members_searchable BOOLEAN NOT NULL DEFAULT TRUE;
