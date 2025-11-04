-- 创建所有 ENUM 类型（公共模块）
-- 说明：所有枚举类型定义必须在建表之前，确保建表时直接使用正确的类型

-- ============================================================
-- 1. 成员角色枚举
-- ============================================================
CREATE TYPE member_role AS ENUM (
    'owner',        -- 群主
    'admin',        -- 管理员
    'member'        -- 普通成员
);

-- ============================================================
-- 2. 房间类型枚举
-- ============================================================
CREATE TYPE room_type AS ENUM (
    'direct',          -- 私聊
    'public_group',    -- 公开群组
    'private_group'    -- 私密群组
);

-- ============================================================
-- 3. 关系类型枚举
-- ============================================================
CREATE TYPE relationship_type AS ENUM (
    'any',          -- 匹配所有值（仅用于查询过滤，不可存储）
    'friend',       -- 好友
    'following',    -- 关注
    'blocked'       -- 拉黑
);

-- ============================================================
-- 4. 关系状态枚举
-- ============================================================
CREATE TYPE relationship_status AS ENUM (
    'any',          -- 匹配所有值（仅用于查询过滤，不可存储）
    'pending',      -- 待处理
    'accepted',     -- 已接受
    'rejected'      -- 已拒绝
);

-- ============================================================
-- 5. 消息类型枚举
-- ============================================================
CREATE TYPE message_type AS ENUM (
    'any',          -- 匹配所有值（仅用于查询过滤，不可存储）
    'text',         -- 文本
    'image',        -- 图片
    'video',        -- 视频
    'audio',        -- 音频
    'file',         -- 文件
    'location',     -- 位置
    'system'        -- 系统消息
);

-- ============================================================
-- 6. 举报状态枚举
-- ============================================================
CREATE TYPE report_status AS ENUM (
    'any',          -- 匹配所有值（仅用于查询过滤，不可存储）
    'pending',      -- 待处理
    'reviewing',    -- 审核中
    'resolved',     -- 已解决
    'dismissed'     -- 已驳回
);

-- ============================================================
-- 7. 邀请状态枚举（群组邀请、入群申请等）
-- ============================================================
CREATE TYPE invite_status AS ENUM (
    'any',          -- 匹配所有值（仅用于查询过滤，不可存储）
    'pending',      -- 待处理
    'accepted',     -- 已接受
    'rejected',     -- 已拒绝
    'approved',     -- 已批准（用于申请）
    'expired'       -- 已过期
);

-- ============================================================
-- 8. 收件箱分类枚举
-- ============================================================
CREATE TYPE inbox_category AS ENUM (
    'any',          -- 匹配所有值（仅用于查询过滤，不可存储）
    'social',       -- 社交互动
    'content',      -- 内容互动
    'system'        -- 系统通知
);

-- ============================================================
-- 注释说明
-- ============================================================
COMMENT ON TYPE member_role IS '成员角色枚举：owner=群主, admin=管理员, member=普通成员';
COMMENT ON TYPE room_type IS '房间类型枚举：direct=私聊, public_group=公开群组, private_group=私密群组';
COMMENT ON TYPE relationship_type IS '关系类型枚举：friend=好友, following=关注, blocked=拉黑';
COMMENT ON TYPE relationship_status IS '关系状态枚举：pending=待处理, accepted=已接受, rejected=已拒绝';
COMMENT ON TYPE message_type IS '消息类型枚举：text=文本, image=图片, video=视频, audio=音频, file=文件, location=位置, system=系统消息';
COMMENT ON TYPE report_status IS '举报状态枚举：pending=待处理, reviewing=审核中, resolved=已解决, dismissed=已驳回';
COMMENT ON TYPE invite_status IS '邀请状态枚举：pending=待处理, accepted=已接受, rejected=已拒绝, approved=已批准, expired=已过期';
COMMENT ON TYPE inbox_category IS '收件箱分类枚举：social=社交互动, content=内容互动, system=系统通知';

