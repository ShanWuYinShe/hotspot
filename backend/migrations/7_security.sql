-- ============================================================================
-- 08. 内容安全系统表
-- ============================================================================
-- 
-- 功能职责：
-- - 定义内容安全相关的所有表结构
-- - 支持举报、审核、封禁等安全功能
-- - 提供内容过滤和风险控制
-- - 记录安全事件和处理日志
--
-- 依赖关系：
-- - 依赖：01_extensions.sql, 02_enums.sql, 03_base_functions.sql, 04_users.sql
--
-- ============================================================================

-- 举报表
CREATE TABLE reports (
    id BIGSERIAL PRIMARY KEY,                   -- 举报ID（自增主键，用于高性能分页）
    report_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 举报唯一标识符（对外API使用，保护数据隐私）
    reporter_id BIGINT NOT NULL,               -- 举报人ID
    target_type report_target_enum NOT NULL,   -- 举报目标类型
    target_id TEXT NOT NULL,                   -- 举报目标ID
    target_user_id BIGINT,                     -- 被举报用户ID（如果适用）
    reason report_reason_enum NOT NULL,        -- 举报原因
    description TEXT,                          -- 详细描述
    evidence_urls TEXT[],                      -- 证据文件URL数组
    status report_status_enum NOT NULL DEFAULT 'pending', -- 处理状态
    priority INTEGER NOT NULL DEFAULT 1,       -- 优先级（1-5，5最高）
    -- 处理信息
    assigned_to BIGINT,                        -- 分配给的管理员ID
    handled_by BIGINT,                         -- 处理人ID
    handled_at TIMESTAMP WITH TIME ZONE,       -- 处理时间
    resolution TEXT,                           -- 处理结果说明
    -- 元数据
    reporter_ip INET,                          -- 举报人IP
    metadata JSONB NOT NULL DEFAULT '{}',     -- 其他元数据
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    deleted_at TIMESTAMP WITH TIME ZONE        -- 删除时间
);

-- 用户封禁记录表
CREATE TABLE user_bans (
    id BIGSERIAL PRIMARY KEY,                   -- 封禁记录ID
    ban_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 封禁唯一标识符
    user_id BIGINT NOT NULL,                   -- 被封禁用户ID
    banned_by BIGINT NOT NULL,                 -- 执行封禁的管理员ID
    ban_type TEXT NOT NULL,                    -- 封禁类型：'temporary'|'permanent'|'shadow'
    reason TEXT NOT NULL,                      -- 封禁原因
    description TEXT,                          -- 详细说明
    related_report_id TEXT,                    -- 相关举报ID
    -- 封禁范围
    ban_scope JSONB NOT NULL DEFAULT '{}',     -- 封禁范围配置（聊天、发瞬间、评论等）
    -- 时间相关
    starts_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 封禁开始时间
    expires_at TIMESTAMP WITH TIME ZONE,       -- 封禁到期时间（NULL表示永久）
    is_active BOOLEAN NOT NULL DEFAULT true,   -- 是否生效
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    deleted_at TIMESTAMP WITH TIME ZONE        -- 删除时间
);

-- 内容审核记录表
CREATE TABLE content_moderation (
    id BIGSERIAL PRIMARY KEY,                   -- 审核记录ID
    moderation_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 审核唯一标识符
    content_type TEXT NOT NULL,                -- 内容类型：'moment'|'comment'|'message'|'user_profile'
    content_id TEXT NOT NULL,                  -- 内容ID
    content_user_id BIGINT NOT NULL,           -- 内容作者ID
    moderator_id BIGINT,                       -- 审核员ID（NULL表示自动审核）
    moderation_type TEXT NOT NULL,             -- 审核类型：'auto'|'manual'|'appeal'
    status moderation_status_enum NOT NULL,    -- 审核状态
    reason TEXT,                               -- 审核原因
    confidence_score DOUBLE PRECISION,        -- 置信度分数（自动审核）
    flags TEXT[],                              -- 标记列表
    action_taken TEXT,                         -- 采取的行动
    notes TEXT,                                -- 审核备注
    metadata JSONB NOT NULL DEFAULT '{}',     -- 审核元数据
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()  -- 更新时间
);

-- 敏感词表
CREATE TABLE sensitive_words (
    id BIGSERIAL PRIMARY KEY,                   -- 敏感词ID
    word TEXT UNIQUE NOT NULL,                 -- 敏感词
    category TEXT NOT NULL,                    -- 分类：'profanity'|'politics'|'violence'|'spam'等
    severity INTEGER NOT NULL DEFAULT 1,       -- 严重程度（1-5）
    action TEXT NOT NULL DEFAULT 'flag',       -- 处理动作：'block'|'flag'|'replace'
    replacement TEXT,                          -- 替换词（如果action为replace）
    is_regex BOOLEAN NOT NULL DEFAULT false,   -- 是否为正则表达式
    is_active BOOLEAN NOT NULL DEFAULT true,   -- 是否启用
    created_by BIGINT,                         -- 创建人ID
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()  -- 更新时间
);

-- 安全事件日志表
CREATE TABLE security_events (
    id BIGSERIAL PRIMARY KEY,                   -- 事件ID
    event_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 事件唯一标识符
    event_type TEXT NOT NULL,                  -- 事件类型：'login_failure'|'suspicious_activity'|'content_violation'等
    user_id BIGINT,                            -- 相关用户ID
    ip_address INET,                           -- IP地址
    user_agent TEXT,                           -- 用户代理
    severity TEXT NOT NULL DEFAULT 'low',      -- 严重程度：'low'|'medium'|'high'|'critical'
    description TEXT NOT NULL,                 -- 事件描述
    details JSONB NOT NULL DEFAULT '{}',      -- 事件详情
    is_resolved BOOLEAN NOT NULL DEFAULT false, -- 是否已解决
    resolved_by BIGINT,                        -- 解决人ID
    resolved_at TIMESTAMP WITH TIME ZONE,      -- 解决时间
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW() -- 创建时间
);

-- IP黑名单表
CREATE TABLE ip_blacklist (
    id BIGSERIAL PRIMARY KEY,                   -- 记录ID
    ip_address INET NOT NULL,                  -- IP地址
    ip_range CIDR,                             -- IP范围（可选）
    reason TEXT NOT NULL,                      -- 封禁原因
    blocked_by BIGINT NOT NULL,                -- 执行封禁的管理员ID
    expires_at TIMESTAMP WITH TIME ZONE,       -- 到期时间（NULL表示永久）
    is_active BOOLEAN NOT NULL DEFAULT true,   -- 是否生效
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    
    UNIQUE(ip_address)                         -- IP地址唯一
);

-- 风险评分表（用户风险评估）
CREATE TABLE user_risk_scores (
    id BIGSERIAL PRIMARY KEY,                   -- 记录ID
    user_id BIGINT UNIQUE NOT NULL,            -- 用户ID
    risk_score INTEGER NOT NULL DEFAULT 0,     -- 风险分数（0-100）
    risk_level TEXT NOT NULL DEFAULT 'low',    -- 风险等级：'low'|'medium'|'high'|'critical'
    factors JSONB NOT NULL DEFAULT '{}',      -- 风险因子详情
    last_calculated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 最后计算时间
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()  -- 更新时间
);

-- 添加外键约束
ALTER TABLE reports ADD CONSTRAINT fk_reports_reporter_id 
    FOREIGN KEY (reporter_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE reports ADD CONSTRAINT fk_reports_target_user_id 
    FOREIGN KEY (target_user_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE reports ADD CONSTRAINT fk_reports_assigned_to 
    FOREIGN KEY (assigned_to) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE reports ADD CONSTRAINT fk_reports_handled_by 
    FOREIGN KEY (handled_by) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE user_bans ADD CONSTRAINT fk_user_bans_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE user_bans ADD CONSTRAINT fk_user_bans_banned_by 
    FOREIGN KEY (banned_by) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE user_bans ADD CONSTRAINT fk_user_bans_related_report_id 
    FOREIGN KEY (related_report_id) REFERENCES reports(report_id) ON DELETE SET NULL;

ALTER TABLE content_moderation ADD CONSTRAINT fk_content_moderation_content_user_id 
    FOREIGN KEY (content_user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE content_moderation ADD CONSTRAINT fk_content_moderation_moderator_id 
    FOREIGN KEY (moderator_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE sensitive_words ADD CONSTRAINT fk_sensitive_words_created_by 
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE security_events ADD CONSTRAINT fk_security_events_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE security_events ADD CONSTRAINT fk_security_events_resolved_by 
    FOREIGN KEY (resolved_by) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE ip_blacklist ADD CONSTRAINT fk_ip_blacklist_blocked_by 
    FOREIGN KEY (blocked_by) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE user_risk_scores ADD CONSTRAINT fk_user_risk_scores_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- 添加检查约束
ALTER TABLE reports ADD CONSTRAINT check_reports_priority 
    CHECK (priority >= 1 AND priority <= 5);

ALTER TABLE reports ADD CONSTRAINT check_reports_description_length 
    CHECK (description IS NULL OR length(description) <= 2000);

ALTER TABLE user_bans ADD CONSTRAINT check_user_bans_expires_at 
    CHECK (expires_at IS NULL OR expires_at > starts_at);

ALTER TABLE content_moderation ADD CONSTRAINT check_content_moderation_confidence_score 
    CHECK (confidence_score IS NULL OR (confidence_score >= 0 AND confidence_score <= 1));

ALTER TABLE sensitive_words ADD CONSTRAINT check_sensitive_words_severity 
    CHECK (severity >= 1 AND severity <= 5);

ALTER TABLE sensitive_words ADD CONSTRAINT check_sensitive_words_word_length 
    CHECK (length(word) >= 1 AND length(word) <= 100);

ALTER TABLE user_risk_scores ADD CONSTRAINT check_user_risk_scores_risk_score 
    CHECK (risk_score >= 0 AND risk_score <= 100);

-- 添加表注释
COMMENT ON TABLE reports IS '举报表，记录用户举报的内容和处理状态';
COMMENT ON TABLE user_bans IS '用户封禁记录表，管理用户封禁状态和期限';
COMMENT ON TABLE content_moderation IS '内容审核记录表，记录内容审核过程和结果';
COMMENT ON TABLE sensitive_words IS '敏感词表，用于内容过滤和自动审核';
COMMENT ON TABLE security_events IS '安全事件日志表，记录系统安全相关事件';
COMMENT ON TABLE ip_blacklist IS 'IP黑名单表，管理被封禁的IP地址';
COMMENT ON TABLE user_risk_scores IS '用户风险评分表，评估用户的风险等级';

-- 添加列注释
COMMENT ON COLUMN reports.target_type IS '举报目标类型：user=用户, group=群组, message=消息, moment=瞬间, comment=评论';
COMMENT ON COLUMN reports.reason IS '举报原因：spam=垃圾信息, harassment=骚扰, inappropriate_content=不当内容等';
COMMENT ON COLUMN reports.status IS '处理状态：pending=待处理, investigating=调查中, resolved=已解决, dismissed=已驳回';
COMMENT ON COLUMN user_bans.ban_scope IS '封禁范围配置，JSON格式，定义具体的封禁功能';
COMMENT ON COLUMN content_moderation.moderation_type IS '审核类型：auto=自动审核, manual=人工审核, appeal=申诉审核';
COMMENT ON COLUMN sensitive_words.action IS '处理动作：block=阻止发布, flag=标记审核, replace=替换词语';
COMMENT ON COLUMN user_risk_scores.factors IS '风险因子详情，JSON格式，包含各种风险指标';
