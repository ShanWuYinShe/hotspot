-- ============================================================
-- 001_foundations.sql
-- 基础环境：扩展、枚举类型、工具函数、分区管理
-- ============================================================

-- ─── 扩展 ───────────────────────────────────────────────────
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "postgis";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gin";
CREATE EXTENSION IF NOT EXISTS "btree_gist";

-- ─── 枚举类型 ──────────────────────────────────────────────

-- 用户
CREATE TYPE user_type AS ENUM ('temp', 'normal', 'admin', 'system');
CREATE TYPE user_status AS ENUM ('normal', 'banned');

-- 隐私
CREATE TYPE privacy_level AS ENUM ('public', 'friends', 'private');

-- 互动
CREATE TYPE reaction_type AS ENUM ('like', 'love', 'laugh', 'angry', 'sad', 'wow');

-- 群组
CREATE TYPE group_join_method AS ENUM ('open', 'approval', 'invite_only');
CREATE TYPE group_role AS ENUM ('owner', 'admin', 'member');

-- 消息
CREATE TYPE message_type AS ENUM ('text', 'image', 'video', 'audio', 'file', 'share', 'system');

-- 申请/审批
CREATE TYPE application_status AS ENUM ('pending', 'accepted', 'rejected');

-- 举报
CREATE TYPE report_status AS ENUM ('pending', 'reviewing', 'resolved', 'dismissed');
CREATE TYPE report_target_type AS ENUM ('user', 'moment', 'comment', 'message', 'group');
CREATE TYPE report_handle_action AS ENUM ('resolve', 'dismiss', 'escalate');

-- 公告
CREATE TYPE announcement_severity AS ENUM ('info', 'warning', 'error');

-- 分享
CREATE TYPE share_type AS ENUM ('user', 'group', 'moment', 'reference_point');

-- 媒体
CREATE TYPE upload_status AS ENUM ('pending', 'processing', 'completed', 'failed');
CREATE TYPE media_category AS ENUM ('avatar', 'group_avatar', 'moment_image', 'chat_image', 'chat_video', 'chat_audio', 'chat_file');

-- 通知
CREATE TYPE notification_category AS ENUM ('social', 'group', 'system');

-- 关注来源
CREATE TYPE follow_source AS ENUM ('moment', 'search', 'recommendation', 'profile', 'group');

-- 群组成员变更操作
CREATE TYPE member_action AS ENUM ('join', 'leave', 'removed', 'banned', 'role_changed', 'muted', 'unmuted');

-- 验证码类型
CREATE TYPE verification_code_type AS ENUM ('register', 'reset_password', 'change_phone', 'login');

-- 不可发消息原因（应用层使用，非列类型）
CREATE TYPE cant_message_reason AS ENUM ('blocked', 'privacy', 'stranger_limit', 'account_suspended');

-- ─── 工具函数 ──────────────────────────────────────────────

-- 自动更新 updated_at
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 安全递减（下限为 0）
CREATE OR REPLACE FUNCTION safe_decrement(current_val INTEGER)
RETURNS INTEGER AS $$
BEGIN
    RETURN GREATEST(current_val - 1, 0);
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Bucket 分片选择（基于实体 ID 哈希，用于热点计数器分散写入）
CREATE OR REPLACE FUNCTION bucket_for_key(p_key BIGINT, p_buckets INTEGER DEFAULT 8)
RETURNS SMALLINT AS $$
BEGIN
    RETURN (abs(p_key) % p_buckets)::SMALLINT;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- UUID 版本的 bucket 分片
CREATE OR REPLACE FUNCTION bucket_for_uuid(p_key UUID, p_buckets INTEGER DEFAULT 8)
RETURNS SMALLINT AS $$
BEGIN
    RETURN (abs(('x' || substr(p_key::TEXT, 1, 8))::BIT(32)::INTEGER) % p_buckets)::SMALLINT;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- 生成随机分享码（8 位字母数字）
CREATE OR REPLACE FUNCTION generate_share_code(p_length INTEGER DEFAULT 8)
RETURNS TEXT AS $$
DECLARE
    chars TEXT := 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789';
    result TEXT := '';
    i INTEGER;
BEGIN
    FOR i IN 1..p_length LOOP
        result := result || substr(chars, floor(random() * length(chars) + 1)::INTEGER, 1);
    END LOOP;
    RETURN result;
END;
$$ LANGUAGE plpgsql VOLATILE;

-- ─── 分区管理 ──────────────────────────────────────────────

-- 分区配置元数据
CREATE TABLE IF NOT EXISTS partition_templates (
    table_name      TEXT        NOT NULL,
    partition_prefix TEXT       NOT NULL,
    retention_days   INTEGER   NOT NULL DEFAULT 730,
    partition_interval VARCHAR(20) NOT NULL DEFAULT 'year',
    create_ahead     INTEGER   NOT NULL DEFAULT 2,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (table_name)
);

-- 创建单个分区
CREATE OR REPLACE FUNCTION create_partition(
    p_table_name    TEXT,
    p_partition_name TEXT,
    p_start_date    DATE,
    p_end_date      DATE
) RETURNS VOID AS $$
BEGIN
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF %s FOR VALUES FROM (%L) TO (%L)',
        p_partition_name, p_table_name, p_start_date, p_end_date
    );
END;
$$ LANGUAGE plpgsql;

-- 确保未来分区存在（由 cron 定期调用）
CREATE OR REPLACE FUNCTION ensure_partitions()
RETURNS void AS $$
DECLARE
    tmpl    RECORD;
    y       INT;
    cur_yr  INT := EXTRACT(YEAR FROM NOW())::INT;
BEGIN
    FOR tmpl IN SELECT * FROM partition_templates LOOP
        FOR y IN (cur_yr - 1)..(cur_yr + tmpl.create_ahead) LOOP
            PERFORM create_partition(
                tmpl.table_name,
                tmpl.partition_prefix || '_' || y,
                make_date(y, 1, 1),
                make_date(y + 1, 1, 1)
            );
        END LOOP;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- 清理过期分区
CREATE OR REPLACE FUNCTION drop_old_partitions()
RETURNS void AS $$
DECLARE
    tmpl        RECORD;
    cutoff      DATE;
    part        RECORD;
    part_year   INT;
BEGIN
    FOR tmpl IN SELECT * FROM partition_templates LOOP
        cutoff := (NOW() - (tmpl.retention_days || ' days')::INTERVAL)::DATE;

        FOR part IN
            SELECT c.relname AS partition_name
            FROM pg_inherits i
            JOIN pg_class c ON i.inhrelid = c.oid
            WHERE i.inhparent = tmpl.table_name::regclass
              AND c.relname ~ ('_\d{4}$')
              AND c.relname != tmpl.partition_prefix || '_default'
        LOOP
            part_year := substring(part.partition_name FROM '\d{4}$')::INT;
            -- 仅当该分区的结束时间（year+1 的 1 月 1 日）早于 cutoff 时才删除
            IF make_date(part_year + 1, 1, 1) < cutoff THEN
                EXECUTE format('DROP TABLE IF EXISTS %I', part.partition_name);
                RAISE NOTICE 'Dropped partition: %', part.partition_name;
            END IF;
        END LOOP;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- 一键维护（由 pg_cron 或外部调度器定期执行）
CREATE OR REPLACE FUNCTION run_partition_maintenance()
RETURNS void AS $$
BEGIN
    PERFORM ensure_partitions();
    PERFORM drop_old_partitions();
END;
$$ LANGUAGE plpgsql;
