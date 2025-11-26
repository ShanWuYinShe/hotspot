-- 008_add_is_deleted_to_group_members.sql
-- 为 group_members 表添加软删除支持

-- 添加 is_deleted 和 deleted_at 列
ALTER TABLE group_members 
ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- 如果表中已有 is_left 列，将其数据迁移到 is_deleted
-- 并将 left_at 迁移到 deleted_at
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'group_members' AND column_name = 'is_left'
    ) THEN
        UPDATE group_members SET is_deleted = is_left WHERE is_left = TRUE;
        UPDATE group_members SET deleted_at = left_at WHERE left_at IS NOT NULL;
    END IF;
END $$;

-- 创建索引以优化软删除查询
CREATE INDEX IF NOT EXISTS idx_group_members_deleted 
ON group_members (group_chat_id, is_deleted) 
WHERE is_deleted = FALSE;

-- 更新现有索引以包含 is_deleted 条件（如果需要）
DROP INDEX IF EXISTS idx_group_members_cursor_id;
CREATE INDEX idx_group_members_cursor_id 
ON group_members (cursor_id) 
WHERE is_deleted = FALSE;

DROP INDEX IF EXISTS idx_group_members_group_chat_id;
CREATE INDEX idx_group_members_group_chat_id 
ON group_members (group_chat_id, joined_at DESC) 
WHERE is_deleted = FALSE;

DROP INDEX IF EXISTS idx_group_members_user_id;
CREATE INDEX idx_group_members_user_id 
ON group_members (user_id, joined_at DESC) 
WHERE is_deleted = FALSE;

DROP INDEX IF EXISTS idx_group_members_count;
CREATE INDEX idx_group_members_count 
ON group_members (group_chat_id) 
WHERE is_deleted = FALSE;

DROP INDEX IF EXISTS idx_group_members_role;
CREATE INDEX idx_group_members_role 
ON group_members (group_chat_id, role) 
WHERE is_deleted = FALSE;




