-- 设备推送Token管理表
-- 用于管理用户设备的推送通知Token
CREATE TABLE IF NOT EXISTS device_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    platform VARCHAR(20) NOT NULL CHECK (platform IN ('ios', 'android', 'web')),
    token TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    
    CONSTRAINT fk_device_tokens_user FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CONSTRAINT uq_device_tokens_user_platform_token UNIQUE (user_id, platform, token)
);

-- 索引
CREATE INDEX idx_device_tokens_user_id ON device_tokens(user_id);
CREATE INDEX idx_device_tokens_token ON device_tokens(token);
CREATE INDEX idx_device_tokens_active ON device_tokens(user_id, is_active);

-- 注释
COMMENT ON TABLE device_tokens IS '设备推送Token管理表';
COMMENT ON COLUMN device_tokens.user_id IS '用户ID';
COMMENT ON COLUMN device_tokens.platform IS '设备平台：ios, android, web';
COMMENT ON COLUMN device_tokens.token IS '推送Token';
COMMENT ON COLUMN device_tokens.is_active IS '是否激活';
COMMENT ON COLUMN device_tokens.last_used_at IS '最后使用时间';






