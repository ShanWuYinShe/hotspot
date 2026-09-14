-- 群已读人数（read_count）结构性虚高修复，第一步：让已读状态随成员关系级联消亡。
--
-- 背景：read_count = 「已读人数」，其 SQL 统计 group_message_read_state 中
-- last_read_sent_at >= 消息 sent_at 的行数。但该表与 group_members 之间没有
-- 任何引用约束，而退群（LeaveGroup）、移除（RemoveMember）、封禁（BanMember）
-- 三条路径都只删 group_members 行，从不清理已读状态——于是退群成员的已读行
-- 长期残留，发送方看到的「N 人已读」包含已退群的人，且随时间只增不减。
--
-- 治法是加一条指向 group_members(group_id, user_id) 的复合外键 + ON DELETE
-- CASCADE：任何位置删除成员行（含未来新增的路径）都会由数据库自动清理对应
-- 已读状态，不需要在 groups 模块的每条删除路径上重复写清理语句，也避免
-- groupmessages 与 groups 之间的反向耦合。
--
-- 注意：group_members 的主键 (group_id, user_id) 已可被外键引用，无需额外
-- 索引。先清理存量孤儿行，否则加约束会失败。

-- 1) 清理存量孤儿行（成员关系已不存在，但已读状态仍在）。
DELETE FROM group_message_read_state grs
WHERE NOT EXISTS (
  SELECT 1 FROM group_members gm
  WHERE gm.group_id = grs.group_id AND gm.user_id = grs.user_id
);

-- 2) 加复合外键（幂等：重复执行时约束已存在则跳过）。
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_group_message_read_state_member'
  ) THEN
    ALTER TABLE group_message_read_state
      ADD CONSTRAINT fk_group_message_read_state_member
      FOREIGN KEY (group_id, user_id)
      REFERENCES group_members(group_id, user_id)
      ON DELETE CASCADE;
  END IF;
END $$;
