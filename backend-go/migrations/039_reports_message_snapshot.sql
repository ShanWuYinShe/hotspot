-- 消息举报内容快照（spec api_spec_types.md ReportSnapshot）：
-- target_type='message' 时必填，举报人提交举报时的消息内容快照，
-- 服务端原样存储，管理员基于快照审核（不直接查询私聊/群聊消息库）。
ALTER TABLE reports ADD COLUMN IF NOT EXISTS message_snapshot JSONB;
