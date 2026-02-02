-- =====================================================
-- HotSpot - 通用分享模块
-- =====================================================
-- 功能：支持多种类型的分享（位置、动态、群组、用户等）
-- 依赖：users 表（001 模块）
-- =====================================================

-- =====================================================
-- 1. shares 表（通用分享主表）
-- =====================================================
CREATE TABLE shares (
    share_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    share_code VARCHAR(6) UNIQUE NOT NULL,
    share_type VARCHAR(20) NOT NULL CHECK (share_type IN ('location', 'moment', 'group', 'user')),
    creator_id BIGINT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE UNIQUE INDEX idx_shares_share_code ON shares(share_code);
CREATE INDEX idx_shares_creator_id ON shares(creator_id, created_at DESC);
CREATE INDEX idx_shares_type ON shares(share_type);
CREATE INDEX idx_shares_created_at ON shares(created_at DESC);

-- 空间索引（用于位置分享的地理查询）
CREATE INDEX idx_shares_location ON shares USING GIST(
    ST_SetSRID(ST_MakePoint(
        CAST(payload->>'longitude' AS DOUBLE PRECISION),
        CAST(payload->>'latitude' AS DOUBLE PRECISION)
    ), 4326)
) WHERE share_type = 'location' 
  AND payload->>'latitude' IS NOT NULL 
  AND payload->>'longitude' IS NOT NULL;

-- 注释
COMMENT ON TABLE shares IS '通用分享表（支持位置、动态、群组、用户等多种类型）';
COMMENT ON COLUMN shares.share_id IS '分享 ID (UUID)';
COMMENT ON COLUMN shares.share_code IS '6位分享码（用于 URL，字母数字，避免混淆字符）';
COMMENT ON COLUMN shares.share_type IS '分享类型：location（位置）, moment（动态）, group（群组）, user（用户）';
COMMENT ON COLUMN shares.creator_id IS '分享创建者 ID';
COMMENT ON COLUMN shares.payload IS '类型特定数据（JSONB）';
COMMENT ON COLUMN shares.created_at IS '创建时间';

-- =====================================================
-- 2. Payload 数据结构示例
-- =====================================================
-- location 类型：
-- {
--   "latitude": 39.9042,
--   "longitude": 116.4074,
--   "location_name": "天安门广场"
-- }

-- moment 类型（规划中）：
-- {
--   "moment_id": "550e8400-e29b-41d4-a716-446655440000"
-- }

-- group 类型（规划中）：
-- {
--   "group_id": "550e8400-e29b-41d4-a716-446655440000"
-- }

-- user 类型（规划中）：
-- {
--   "user_id": 1001
-- }

-- =====================================================
-- 3. 数据清理策略（可选）
-- =====================================================
-- 建议：定期清理超过 90 天的分享记录（可选）
-- DELETE FROM shares WHERE created_at < NOW() - INTERVAL '90 days';
