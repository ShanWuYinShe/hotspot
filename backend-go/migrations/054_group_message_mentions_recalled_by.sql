-- 群消息补齐 mentions / recalled_by 载荷列：Send 接受并广播 @ 列表、撤回
-- 事件携带操作者，但两者从未落库，group_messages/fetch 拉取历史时 mentions
-- 恒缺、recalled_by 恒 null（对齐 spec api_spec_chat.md GroupMessageItem；
-- 同类补齐先例见 044 的 share/file/reply 列）。
ALTER TABLE group_messages
  ADD COLUMN IF NOT EXISTS mentions TEXT[] NOT NULL DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS recalled_by TEXT NULL REFERENCES users(user_id) ON DELETE SET NULL;
