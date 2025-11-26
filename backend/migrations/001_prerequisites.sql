-- 001_prerequisites.sql
-- 数据库扩展、枚举类型、通用函数

-- ============================================================
-- 1. PostgreSQL扩展
-- ============================================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS postgis_topology;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ============================================================
-- 2. 枚举类型
-- ============================================================

-- 用户相关
CREATE TYPE user_type AS ENUM ('temp', 'normal', 'system');
CREATE TYPE privacy_level AS ENUM ('public', 'friends', 'private');

-- 群组相关
CREATE TYPE join_method AS ENUM ('open', 'approval', 'invite_only');
CREATE TYPE member_role AS ENUM ('owner', 'admin', 'member');
CREATE TYPE invitation_status AS ENUM ('pending', 'accepted', 'rejected', 'expired');

-- 消息相关
CREATE TYPE message_type AS ENUM ('text', 'image', 'video', 'audio', 'file', 'location', 'system');
CREATE TYPE reaction_type AS ENUM ('like', 'love', 'laugh', 'angry', 'sad', 'wow');

-- 举报相关
CREATE TYPE report_target_type AS ENUM ('user', 'moment', 'comment', 'message', 'group');
CREATE TYPE report_status AS ENUM ('pending', 'reviewing', 'resolved', 'dismissed');

-- 通知相关
CREATE TYPE notification_category AS ENUM ('social', 'system', 'content');
CREATE TYPE notification_severity AS ENUM ('info', 'warning', 'error');

-- 信息流相关
CREATE TYPE feed_channel AS ENUM ('following', 'nearby', 'recommended');
CREATE TYPE feed_item_type AS ENUM ('moment', 'group_card', 'system');

-- 排行榜相关
CREATE TYPE leaderboard_timeframe AS ENUM ('daily', 'weekly', 'monthly', 'all_time');
CREATE TYPE leaderboard_scope AS ENUM ('global', 'nearby', 'friends');

-- 设备相关
CREATE TYPE platform AS ENUM ('ios', 'android', 'web');

-- ============================================================
-- 3. 通用触发器函数
-- ============================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION sync_user_location()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.current_latitude IS NOT NULL AND NEW.current_longitude IS NOT NULL THEN
        NEW.location = ST_SetSRID(ST_MakePoint(NEW.current_longitude, NEW.current_latitude), 4326)::geography;
        NEW.location_updated_at = NOW();
    ELSE
        NEW.location = NULL;
        NEW.location_updated_at = NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION sync_group_location()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.latitude IS NOT NULL AND NEW.longitude IS NOT NULL THEN
        NEW.location = ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326)::geography;
    ELSE
        NEW.location = NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION sync_moment_location()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.latitude IS NOT NULL AND NEW.longitude IS NOT NULL THEN
        NEW.location = ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326)::geography;
    ELSE
        NEW.location = NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
