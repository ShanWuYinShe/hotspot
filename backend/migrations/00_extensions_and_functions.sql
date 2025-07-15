-- 数据库扩展和公共函数
-- PostgreSQL扩展和通用功能

-- 启用PostGIS扩展（地理位置功能）
CREATE EXTENSION IF NOT EXISTS postgis CASCADE;

-- 启用UUID生成扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 通用地理位置更新函数
-- 在插入或更新经纬度时自动更新geom字段
CREATE OR REPLACE FUNCTION update_geom_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.geom = ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326)::geography;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
