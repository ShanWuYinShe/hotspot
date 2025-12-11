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
CREATE TYPE privacy_level AS ENUM ('public', 'friends', 'private');
-- 用户类型：temp-临时用户, normal-普通用户, admin-管理员用户, system-系统用户
CREATE TYPE user_type AS ENUM ('temp', 'normal', 'admin', 'system');
-- 私聊权限：everyone-所有人, friends-仅好友, nobody-禁止
CREATE TYPE chat_permission AS ENUM ('everyone', 'friends', 'nobody');

-- 群组相关
CREATE TYPE join_method AS ENUM ('open', 'approval', 'invite_only');
CREATE TYPE member_role AS ENUM ('owner', 'admin', 'member');
CREATE TYPE invitation_status AS ENUM ('pending', 'accepted', 'rejected', 'expired');
CREATE TYPE member_action AS ENUM ('join', 'leave', 'removed', 'banned', 'unbanned', 'role_changed', 'muted', 'unmuted');

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

-- 设备相关
CREATE TYPE platform AS ENUM ('ios', 'android', 'web', 'harmony');

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

-- 用户位置同步函数（带默认值）
CREATE OR REPLACE FUNCTION sync_user_location()
RETURNS TRIGGER AS $$
BEGIN
    -- 如果经纬度为空，使用默认值（南天门）
    IF NEW.current_latitude IS NULL THEN
        NEW.current_latitude := 35.154922;
    END IF;
    IF NEW.current_longitude IS NULL THEN
        NEW.current_longitude := 112.682524;
    END IF;
    IF NEW.current_location_name IS NULL THEN
        NEW.current_location_name := '南天门';
    END IF;
    
    -- 同步 geography 字段
    NEW.location := ST_SetSRID(ST_MakePoint(NEW.current_longitude, NEW.current_latitude), 4326)::geography;
    NEW.location_updated_at := NOW();
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION sync_user_location() IS '同步用户位置信息，确保 location geography 字段与经纬度字段保持一致，并为空值设置默认位置（南天门）';

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

CREATE OR REPLACE FUNCTION sync_location_history()
RETURNS TRIGGER AS $$
BEGIN
    NEW.location = ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326)::geography;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION sync_location_history() IS '同步位置历史记录的 location geography 字段';

-- ============================================================
-- 4. 用户ID序列
-- 系统用户区间：1 - 1,000,000
-- 临时用户和注册用户区间：1,000,001+
-- ============================================================
CREATE SEQUENCE user_id_seq START WITH 1000001 INCREMENT BY 1 NO MAXVALUE NO CYCLE;

COMMENT ON SEQUENCE user_id_seq IS '统一用户ID序列（临时用户和注册用户共用，系统用户使用保留区间1-1000000）';

-- 获取下一个用户ID的函数
CREATE OR REPLACE FUNCTION next_user_id() 
RETURNS BIGINT AS $$
BEGIN
    RETURN nextval('user_id_seq');
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION next_user_id() IS '获取下一个用户ID（用于临时用户和注册用户）';
