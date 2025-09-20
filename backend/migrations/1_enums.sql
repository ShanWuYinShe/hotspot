-- ============================================================================
-- 02. 枚举类型定义
-- ============================================================================
-- 
-- 功能职责：
-- - 定义系统中所有的枚举类型
-- - 确保数据一致性和类型安全
-- - 支持业务逻辑的类型约束
--
-- 依赖关系：
-- - 依赖：01_extensions.sql
--
-- ============================================================================

-- 群组类型枚举
CREATE TYPE group_type_enum AS ENUM ('public_chat', 'private_chat');

-- 成员角色枚举
CREATE TYPE member_role_enum AS ENUM ('owner', 'admin', 'member');

-- 成员状态枚举
CREATE TYPE member_status_enum AS ENUM ('active', 'muted', 'banned', 'left');

-- 消息类型枚举
CREATE TYPE message_type_enum AS ENUM ('text', 'image', 'video', 'audio', 'file', 'location', 'system');

-- 媒体类型枚举
CREATE TYPE media_type_enum AS ENUM ('image', 'video', 'audio', 'file', 'location');

-- 反应类型枚举
CREATE TYPE reaction_type_enum AS ENUM ('like', 'love', 'laugh', 'angry', 'sad', 'wow');

-- 可见性枚举（用于生活瞬间）
CREATE TYPE visibility_enum AS ENUM ('public', 'friends', 'private');

-- 用户关系类型枚举
CREATE TYPE relationship_type_enum AS ENUM ('friend', 'follow', 'block');

-- 用户关系状态枚举
CREATE TYPE relationship_status_enum AS ENUM ('pending', 'accepted', 'rejected', 'active');

-- 举报目标类型枚举
CREATE TYPE report_target_enum AS ENUM ('user', 'group', 'message', 'moment', 'comment');

-- 举报原因枚举
CREATE TYPE report_reason_enum AS ENUM ('spam', 'harassment', 'inappropriate_content', 'violence', 'hate_speech', 'fake_information', 'copyright_violation', 'other');

-- 举报状态枚举
CREATE TYPE report_status_enum AS ENUM ('pending', 'investigating', 'resolved', 'dismissed');

-- 通知类型枚举
CREATE TYPE notification_type_enum AS ENUM ('system', 'message', 'group_invite', 'friend_request', 'moment_like', 'moment_comment', 'mention');

-- 通知状态枚举
CREATE TYPE notification_status_enum AS ENUM ('unread', 'read', 'archived');

-- 用户状态枚举
CREATE TYPE user_status_enum AS ENUM ('active', 'inactive', 'suspended', 'deleted');

-- 群组状态枚举
CREATE TYPE group_status_enum AS ENUM ('active', 'archived', 'suspended', 'deleted');

-- 内容审核状态枚举
CREATE TYPE moderation_status_enum AS ENUM ('pending', 'approved', 'rejected', 'flagged');

-- 活动类型枚举（用于用户活动记录）
CREATE TYPE activity_type_enum AS ENUM (
    'login', 'logout', 'register', 'profile_update', 'location_update',
    'create_group', 'join_group', 'leave_group', 'group_invite', 'group_kick',
    'send_message', 'delete_message', 'edit_message',
    'create_moment', 'delete_moment', 'like_moment', 'unlike_moment',
    'add_friend', 'remove_friend', 'block_user', 'unblock_user',
    'create_report', 'handle_report'
);

-- 添加枚举类型注释
COMMENT ON TYPE group_type_enum IS '群组类型：public_chat=公开群聊, private_chat=私聊';
COMMENT ON TYPE member_role_enum IS '成员角色：owner=群主, admin=管理员, member=普通成员';
COMMENT ON TYPE member_status_enum IS '成员状态：active=活跃, muted=禁言, banned=封禁, left=已退出';
COMMENT ON TYPE message_type_enum IS '消息类型：text=文本, image=图片, video=视频, audio=音频, file=文件, location=位置, system=系统消息';
COMMENT ON TYPE media_type_enum IS '媒体类型：image=图片, video=视频, audio=音频, file=文件, location=位置';
COMMENT ON TYPE reaction_type_enum IS '反应类型：like=点赞, love=喜爱, laugh=大笑, angry=愤怒, sad=悲伤, wow=惊讶';
COMMENT ON TYPE visibility_enum IS '可见性：public=公开, friends=好友可见, private=私有';
COMMENT ON TYPE relationship_type_enum IS '用户关系类型：friend=好友, follow=关注, block=拉黑';
COMMENT ON TYPE relationship_status_enum IS '关系状态：pending=待处理, accepted=已接受, rejected=已拒绝, active=活跃';
COMMENT ON TYPE report_target_enum IS '举报目标类型：user=用户, group=群组, message=消息, moment=瞬间, comment=评论';
COMMENT ON TYPE report_reason_enum IS '举报原因：spam=垃圾信息, harassment=骚扰, inappropriate_content=不当内容, violence=暴力, hate_speech=仇恨言论, fake_information=虚假信息, copyright_violation=版权侵犯, other=其他';
COMMENT ON TYPE report_status_enum IS '举报状态：pending=待处理, investigating=调查中, resolved=已解决, dismissed=已驳回';
COMMENT ON TYPE notification_type_enum IS '通知类型：system=系统通知, message=消息通知, group_invite=群组邀请, friend_request=好友请求, moment_like=瞬间点赞, moment_comment=瞬间评论, mention=提及通知';
COMMENT ON TYPE notification_status_enum IS '通知状态：unread=未读, read=已读, archived=已归档';
COMMENT ON TYPE user_status_enum IS '用户状态：active=活跃, inactive=不活跃, suspended=暂停, deleted=已删除';
COMMENT ON TYPE group_status_enum IS '群组状态：active=活跃, archived=已归档, suspended=暂停, deleted=已删除';
COMMENT ON TYPE moderation_status_enum IS '审核状态：pending=待审核, approved=已通过, rejected=已拒绝, flagged=已标记';
COMMENT ON TYPE activity_type_enum IS '活动类型：用于记录用户各种操作活动';
