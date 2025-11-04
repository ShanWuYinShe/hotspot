#!/bin/bash
set -e

echo "🔧 设置 HotSpot 数据库..."

# 创建数据库用户和数据库
sudo -u postgres psql << SQL
-- 创建用户（如果不存在）
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_user WHERE usename = 'hotspot') THEN
    CREATE USER hotspot WITH PASSWORD 'hotspot_password';
  END IF;
END
\$\$;

-- 创建数据库（如果不存在）
SELECT 'CREATE DATABASE hotspot OWNER hotspot'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'hotspot')\gexec

-- 授予权限
GRANT ALL PRIVILEGES ON DATABASE hotspot TO hotspot;
SQL

echo "✅ 数据库和用户创建成功"

# 连接到 hotspot 数据库并启用 PostGIS
sudo -u postgres psql -d hotspot << SQL
-- 启用 PostGIS 扩展
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS postgis_topology;

-- 授予 schema 权限
GRANT ALL ON SCHEMA public TO hotspot;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO hotspot;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO hotspot;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO hotspot;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO hotspot;
SQL

echo "✅ PostGIS 扩展已启用"
echo "✅ 数据库设置完成！"
echo ""
echo "📋 数据库信息："
echo "  - 数据库名: hotspot"
echo "  - 用户名: hotspot"
echo "  - 密码: hotspot_password"
echo "  - 连接字符串: postgresql://hotspot:hotspot_password@localhost:5432/hotspot"
