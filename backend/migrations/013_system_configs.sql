-- 系统配置表
CREATE TABLE system_configs (
    id BIGSERIAL PRIMARY KEY,
    config_key VARCHAR(100) NOT NULL UNIQUE,
    config_value TEXT NOT NULL,
    value_type VARCHAR(20) NOT NULL CHECK (value_type IN ('string', 'number', 'boolean')),
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 配置变更日志表
CREATE TABLE config_change_logs (
    id BIGSERIAL PRIMARY KEY,
    config_key VARCHAR(100) NOT NULL,
    old_value TEXT,
    new_value TEXT NOT NULL,
    changed_by BIGINT NOT NULL REFERENCES users(user_id) ON DELETE SET NULL,
    changed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    ip_address VARCHAR(45),
    user_agent TEXT
);

-- 索引
CREATE INDEX idx_system_configs_key ON system_configs(config_key);
CREATE INDEX idx_config_change_logs_key ON config_change_logs(config_key);
CREATE INDEX idx_config_change_logs_changed_by ON config_change_logs(changed_by);
CREATE INDEX idx_config_change_logs_changed_at ON config_change_logs(changed_at DESC);

-- 插入默认配置值
INSERT INTO system_configs (config_key, config_value, value_type, description) VALUES
    ('default_latitude', '35.154922', 'number', '默认纬度（南天门）'),
    ('default_longitude', '112.682524', 'number', '默认经度（南天门）'),
    ('default_location_name', '南天门', 'string', '默认位置名称'),
    ('location_change_threshold', '500', 'number', '位置变化记录阈值（米）'),
    ('access_token_ttl', '3600', 'number', '访问令牌有效期（秒）'),
    ('refresh_token_ttl', '2592000', 'number', '刷新令牌有效期（秒，30天）'),
    ('max_upload_size_avatar', '5242880', 'number', '头像最大上传大小（字节，5MB）'),
    ('max_upload_size_image', '10485760', 'number', '图片最大上传大小（字节，10MB）'),
    ('search_rate_limit', '100', 'number', '搜索接口频率限制（次/小时）'),
    ('message_rate_limit', '1000', 'number', '消息发送频率限制（次/小时）');

-- 触发器：自动更新 updated_at
CREATE OR REPLACE FUNCTION update_system_config_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_system_config_timestamp
BEFORE UPDATE ON system_configs
FOR EACH ROW
EXECUTE FUNCTION update_system_config_timestamp();




