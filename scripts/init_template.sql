-- HotSpot 模板数据库初始化脚本
-- 
-- 此脚本完成以下任务：
-- 1. 创建数据库用户 hotspot_user（具有 CREATEDB 权限）
-- 2. 创建模板数据库 hotspot_template（预装所有必需扩展）
--
-- 注意：此脚本不创建用户数据库！
-- 用户数据库应通过 Shell 脚本或 SQLx 基于模板创建。
--
-- 使用方法：
--   ./setup_database.sh --init-template

-- ============================================================================
-- 1. 创建数据库用户（无密码，仅本地认证）
-- ============================================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_user WHERE usename = 'hotspot_user') THEN
    CREATE USER hotspot_user WITH CREATEDB;
    RAISE NOTICE '✓ 用户 hotspot_user 创建成功';
  ELSE
    RAISE NOTICE '✓ 用户 hotspot_user 已存在';
  END IF;
END
$$;

-- ============================================================================
-- 2. 创建模板数据库
-- ============================================================================
-- 检查模板数据库是否存在
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_database WHERE datname = 'hotspot_template') THEN
    -- 注意：CREATE DATABASE 不能在事务块中执行，所以我们使用 \gexec
    RAISE NOTICE '准备创建模板数据库 hotspot_template...';
  ELSE
    RAISE NOTICE '✓ 模板数据库 hotspot_template 已存在';
  END IF;
END
$$;

-- 创建模板数据库（如果不存在）
SELECT 'CREATE DATABASE hotspot_template OWNER hotspot_user'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'hotspot_template')\gexec

-- ============================================================================
-- 3. 连接到模板数据库并启用所有必需的扩展
-- ============================================================================
\c hotspot_template

-- 启用 PostGIS 核心扩展
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS postgis_raster;
CREATE EXTENSION IF NOT EXISTS postgis_topology;
CREATE EXTENSION IF NOT EXISTS postgis_sfcgal;

-- 启用其他必需扩展
CREATE EXTENSION IF NOT EXISTS fuzzystrmatch;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 显示已启用的扩展
SELECT '✓ 已启用扩展: ' || string_agg(extname, ', ') AS enabled_extensions
FROM pg_extension
WHERE extname IN ('postgis', 'postgis_raster', 'postgis_topology', 'postgis_sfcgal', 'fuzzystrmatch', 'uuid-ossp');

-- ============================================================================
-- 4. 授予用户权限
-- ============================================================================
-- 授予 schema 权限
GRANT ALL ON SCHEMA public TO hotspot_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO hotspot_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO hotspot_user;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO hotspot_user;

-- 设置默认权限（对未来创建的对象自动授权）
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO hotspot_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO hotspot_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON FUNCTIONS TO hotspot_user;

-- ============================================================================
-- 5. 将数据库标记为模板并禁止连接
-- ============================================================================
\c postgres

-- 标记为模板数据库并禁止连接（标准模板数据库做法）
UPDATE pg_database 
SET datistemplate = true, datallowconn = false 
WHERE datname = 'hotspot_template';

SELECT '✓ hotspot_template 已标记为模板数据库（不可连接）' AS status;

-- ============================================================================
-- 6. 验证配置
-- ============================================================================
SELECT '✓ 模板数据库初始化完成！' AS status;

