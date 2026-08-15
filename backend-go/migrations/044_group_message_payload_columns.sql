-- 群消息补齐 share/file/reply 载荷列：Send 接受并广播这些字段但此前从未落库，
-- History 拉取时永远缺失（对齐 private_messages 的列集合与索引）。
ALTER TABLE group_messages
  ADD COLUMN IF NOT EXISTS share_code TEXT,
  ADD COLUMN IF NOT EXISTS file_id TEXT,
  ADD COLUMN IF NOT EXISTS reply_to_message_id TEXT NULL REFERENCES group_messages(message_id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_group_messages_group_id_reply_to ON group_messages(group_id, reply_to_message_id);
