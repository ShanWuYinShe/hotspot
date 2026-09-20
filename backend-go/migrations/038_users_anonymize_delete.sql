-- 账户删除匿名化（spec api_spec_user.md delete_account「数据保留策略」）：
-- - 内容保留 + 作者匿名化：users 行不删除（保留 user_id 关联），
--   昵称/头像/bio 等个人信息替换为"已注销用户"占位；
--   已发布瞬间/评论/群消息等公开内容因不触发 ON DELETE CASCADE 而保留，
--   查询 JOIN users 时自然显示占位昵称
-- - is_deleted 标记：已注销用户不再可被搜索/访问（users/search、get_detail、
--   feed 等查询过滤），auth_accounts 已删（login_id 释放可重新注册）
-- - 个人数据（登录凭证/会话/设置/收藏/关注关系/通知/私聊记录）由 service
--   层在同一事务中显式清理（见 users module DeleteAccount）

ALTER TABLE users
ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_users_is_deleted ON users(is_deleted) WHERE is_deleted = TRUE;
