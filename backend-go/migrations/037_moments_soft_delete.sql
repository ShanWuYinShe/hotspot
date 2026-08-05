-- 瞬间软删除（spec api_spec_moment.md「删除语义」）：
-- - 瞬间不物理删除，标记 is_deleted=true、description 清空（客户端展示"内容已删除"占位）
-- - 其下评论/反应数据保留（不级联删除）；作者 list_my、收藏者 list_favorites 返回软删占位项，
--   其余场景（详情/Feed/搜索/评论挂载）按不存在处理
-- - 删除后推送 MomentDeleted WS 事件（作者其他设备 + 收藏者）

ALTER TABLE moments
ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE;
