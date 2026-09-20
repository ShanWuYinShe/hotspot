-- 陌生人配额计数补索引：
-- privatemessages/queryCountPrivateMessagesByTarget 按 (sender_user_id, sent_at)
-- 过滤当日各目标计数（enforceStrangerQuota，每条陌生人消息触发一次），
-- 此前 private_messages 只有 chat_key 前缀索引，该路径走 seq scan；
-- 对齐 040 给 group_messages 补 sender 索引的同款处理（私聊侧当时漏了）。
CREATE INDEX IF NOT EXISTS idx_private_messages_sender_user_id_sent_at
  ON private_messages(sender_user_id, sent_at);
