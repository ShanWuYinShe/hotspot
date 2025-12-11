-- 008_tag_module.sql
-- 标签模块：标签主表、用户标签、群组标签、瞬间标签

-- ============================================================
-- 标签主表（全局标签池）
-- ============================================================
CREATE TABLE tags (
    id BIGSERIAL PRIMARY KEY,
    tag_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL,
    usage_count INTEGER NOT NULL DEFAULT 0,  -- 使用次数（冗余字段，用于热门标签查询）
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    CONSTRAINT unique_tag_name UNIQUE (name)
);

CREATE INDEX idx_tags_tag_id ON tags (tag_id);
CREATE INDEX idx_tags_name ON tags (name);
CREATE INDEX idx_tags_usage_count ON tags (usage_count DESC);

CREATE TRIGGER trigger_tags_updated_at
    BEFORE UPDATE ON tags FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 用户标签关联表
-- ============================================================
CREATE TABLE user_tags (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    tag_id BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_user_tag UNIQUE (user_id, tag_id)
);

CREATE INDEX idx_user_tags_user ON user_tags (user_id);
CREATE INDEX idx_user_tags_tag ON user_tags (tag_id);

-- ============================================================
-- 群组标签关联表
-- ============================================================
CREATE TABLE group_tags (
    id BIGSERIAL PRIMARY KEY,
    group_chat_id UUID NOT NULL,
    tag_id BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_group_tag UNIQUE (group_chat_id, tag_id)
);

CREATE INDEX idx_group_tags_group ON group_tags (group_chat_id);
CREATE INDEX idx_group_tags_tag ON group_tags (tag_id);

-- ============================================================
-- 瞬间标签关联表
-- ============================================================
CREATE TABLE moment_tags (
    id BIGSERIAL PRIMARY KEY,
    moment_id UUID NOT NULL,
    tag_id BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_moment_tag UNIQUE (moment_id, tag_id)
);

CREATE INDEX idx_moment_tags_moment ON moment_tags (moment_id);
CREATE INDEX idx_moment_tags_tag ON moment_tags (tag_id);

-- ============================================================
-- 更新标签使用次数的触发器函数
-- ============================================================
CREATE OR REPLACE FUNCTION update_tag_usage_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE tags SET usage_count = usage_count + 1 WHERE id = NEW.tag_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE tags SET usage_count = GREATEST(usage_count - 1, 0) WHERE id = OLD.tag_id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- 为各关联表添加触发器
CREATE TRIGGER trigger_user_tags_usage
    AFTER INSERT OR DELETE ON user_tags
    FOR EACH ROW EXECUTE FUNCTION update_tag_usage_count();

CREATE TRIGGER trigger_group_tags_usage
    AFTER INSERT OR DELETE ON group_tags
    FOR EACH ROW EXECUTE FUNCTION update_tag_usage_count();

CREATE TRIGGER trigger_moment_tags_usage
    AFTER INSERT OR DELETE ON moment_tags
    FOR EACH ROW EXECUTE FUNCTION update_tag_usage_count();








