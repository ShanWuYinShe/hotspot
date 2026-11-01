-- 059_pg_stat_statements_extension.sql
-- 查询级统计可观测性（2026-11-01 线上性能排查 P3 项）。
--
-- 现状：compose 的 postgres 自始用 shared_preload_libraries=pg_stat_statements
-- 与 pg_stat_statements.track=all 预加载（docker-compose.yml），但线上库从未
-- CREATE EXTENSION（pg_extension 仅 plpgsql/pg_trgm），pg_stat_statements 视图
-- 不可用——查询级 p50/p95 只能靠 500ms 慢日志，历史保留有限（见线上排查报告
-- origin-perf.md §5.3）。
--
-- 权限事实（本机 PG 18.4 实测）：pg_stat_statements 不是 trusted extension，
-- 非超级用户 CREATE EXTENSION 报 "Must be superuser"（同类 pg_trgm 是 trusted，
-- 普通库 owner 即可创建，故 040/045/052 的 CREATE EXTENSION pg_trgm 不受影响）。
-- 本仓库部署形态的后端 DB 用户由 docker-compose.yml 硬编码为 postgres 超级用户
-- （POSTGRES_USER / POSTGRES_DSN；部署脚本备份也默认 DEPLOY_DB_USER=postgres），
-- 正常路径可建。
--
-- 失败影响与保护：迁移出错会让 EnsureSchema 返回错误、后端拒绝启动（cmd/migrate
-- 与 server 启动同路径）。「诊断用扩展」不值得成为启动链路的硬依赖，因此用
-- DO + EXCEPTION 降级：无权限或扩展不可用（如托管 PG 未提供该 contrib）时只
-- WARNING 跳过，业务与启动不受影响；成功后统计立即开始累积（预加载已生效）。
-- 未预加载的库即使建了扩展，查询该视图会报 "must be loaded via
-- shared_preload_libraries"，但建扩展本身成功、无副作用（实测）。
--
-- 幂等：IF NOT EXISTS；重复执行/基线重放均无副作用。
-- 兼容：扩展不改变任何业务 SQL 语义，旧版二进制无需感知（回滚 = 旧镜像 + 保留
-- 新 schema）；如需移除可直接 DROP EXTENSION pg_stat_statements。
DO $$
BEGIN
  CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
EXCEPTION
  WHEN OTHERS THEN
    RAISE WARNING 'pg_stat_statements 扩展未创建（可选观测项，忽略即可）：%', SQLERRM;
END
$$;
