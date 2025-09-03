-- 创建好友关系表
-- 支持好友请求、接受、拒绝等状态管理

CREATE TABLE IF NOT EXISTS friendships (
    id BIGSERIAL PRIMARY KEY,
    friendship_id UUID NOT NULL DEFAULT gen_random_uuid(),
    requester_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    addressee_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 添加约束
ALTER TABLE friendships ADD CONSTRAINT check_status 
CHECK (status IN ('pending', 'accepted', 'rejected', 'blocked'));

ALTER TABLE friendships ADD CONSTRAINT no_self_friendship 
CHECK (requester_id != addressee_id);

-- 创建索引优化查询性能
CREATE INDEX IF NOT EXISTS idx_friendships_requester_id ON friendships(requester_id);
CREATE INDEX IF NOT EXISTS idx_friendships_addressee_id ON friendships(addressee_id);
CREATE INDEX IF NOT EXISTS idx_friendships_status ON friendships(status);
CREATE INDEX IF NOT EXISTS idx_friendships_created_at ON friendships(created_at);

-- 创建复合索引优化好友列表查询
CREATE INDEX IF NOT EXISTS idx_friendships_user_status ON friendships(requester_id, status) WHERE status = 'accepted';
CREATE INDEX IF NOT EXISTS idx_friendships_addressee_status ON friendships(addressee_id, status) WHERE status = 'accepted';

-- 创建更新时间自动更新触发器
CREATE OR REPLACE FUNCTION update_friendships_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_friendships_updated_at
    BEFORE UPDATE ON friendships
    FOR EACH ROW
    EXECUTE FUNCTION update_friendships_updated_at();

-- 添加注释
COMMENT ON TABLE friendships IS '好友关系表，管理用户之间的好友请求和关系状态';
COMMENT ON COLUMN friendships.friendship_id IS '好友关系唯一标识符';
COMMENT ON COLUMN friendships.requester_id IS '发起好友请求的用户ID';
COMMENT ON COLUMN friendships.addressee_id IS '接收好友请求的用户ID';
COMMENT ON COLUMN friendships.status IS '好友关系状态：pending-待处理，accepted-已接受，rejected-已拒绝，blocked-已屏蔽';
COMMENT ON COLUMN friendships.created_at IS '好友请求创建时间';
COMMENT ON COLUMN friendships.updated_at IS '好友关系最后更新时间';
