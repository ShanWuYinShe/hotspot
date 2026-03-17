-- 016_region_boundary.sql
-- 行政区边界表与区域统计预聚合，以及业务表区域归属列

-- 统一区域边界表：所有 OSM 行政边界最终都汇聚到此表
-- 以 GPKG (osm_id, name, admin_level, geom) 为唯一真实来源，只在此基础上派生少量辅助字段。
CREATE TABLE IF NOT EXISTS region_boundary (
	region_id    bigserial PRIMARY KEY,
	-- 与 OSM admin_level 对齐，例如：2=国家、4=省/州、6=市/区县等
	admin_level  smallint NOT NULL,

	osm_id       bigint  NOT NULL,

	name         text    NOT NULL,

	centroid     geometry(POINT, 4326)        NOT NULL,
	geom         geometry(GEOMETRY, 4326)     NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_region_boundary_geom
  ON region_boundary USING GIST (geom);

CREATE INDEX IF NOT EXISTS idx_region_boundary_level
	ON region_boundary(admin_level);

-- 保证同一 OSM 实体只对应一条记录，便于幂等 upsert
CREATE UNIQUE INDEX IF NOT EXISTS idx_region_boundary_osm_unique
	ON region_boundary(admin_level, osm_id);

-- 按日期 + 行政区边界的预聚合统计表
CREATE TABLE IF NOT EXISTS region_boundary_stats_daily (
  date         date   NOT NULL,
  region_id    bigint NOT NULL,
  user_count   bigint NOT NULL DEFAULT 0,
  group_count  bigint NOT NULL DEFAULT 0,
  moment_count bigint NOT NULL DEFAULT 0,
  PRIMARY KEY (date, region_id)
);

CREATE INDEX IF NOT EXISTS idx_users_region_id
  ON users(region_id);

CREATE INDEX IF NOT EXISTS idx_groups_region_id
  ON groups(region_id);
