-- 016_region_boundary.sql
-- 行政区边界表与区域统计预聚合

-- 统一区域边界表：所有行政边界最终都汇聚到此表
-- 以 GPKG 中的边界几何 (geom) 为真实来源，只在此基础上派生少量辅助字段。
CREATE TABLE IF NOT EXISTS region_boundary (
    region_id    BIGSERIAL PRIMARY KEY,
    -- 与 OSM admin_level 语义对齐，例如：2=国家、4=省/州、6=市/区县等，但内部统一命名为 region_level
    region_level SMALLINT NOT NULL,
    name         TEXT     NOT NULL,
    centroid     GEOMETRY(POINT, 4326)        NOT NULL,
    geom         GEOMETRY(GEOMETRY, 4326)     NOT NULL,
    -- 用于判定几何是否完全相等（基于 ST_Normalize(geom) 的 EWKB 做哈希）
    geom_hash    BYTEA                        NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_region_boundary_geom
    ON region_boundary USING GIST (geom);

CREATE INDEX IF NOT EXISTS idx_region_boundary_centroid
    ON region_boundary USING GIST (centroid);

CREATE INDEX IF NOT EXISTS idx_region_boundary_level
    ON region_boundary(region_level);

-- 保证空间上完全相等的多边形只会存储一条记录，便于幂等 upsert
CREATE UNIQUE INDEX IF NOT EXISTS idx_region_boundary_geom_hash_unique
    ON region_boundary(geom_hash);

-- 按日期 + 行政区边界的预聚合统计表
CREATE TABLE IF NOT EXISTS region_boundary_stats_daily (
    date         DATE   NOT NULL,
    region_id    BIGINT NOT NULL,
    user_count   BIGINT NOT NULL DEFAULT 0,
    group_count  BIGINT NOT NULL DEFAULT 0,
    moment_count BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (date, region_id)
);

CREATE INDEX IF NOT EXISTS idx_region_boundary_stats_region_date
    ON region_boundary_stats_daily(region_id, date);
