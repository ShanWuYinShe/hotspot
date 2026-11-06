# 后端性能基线 / SLO / 压测手册

本文件是后端性能的「可验证口径」：怎么测、测什么、当前基线、SLO 目标与容量
模型假设。绝对数字依赖机器（本机 OrbStack 抖动可达 ±2x），**只用于同机/同会话
前后对比与回归门禁，不当作生产容量承诺**；生产容量以线上监控（Grafana
「HotSpot 后端总览」）+ 同构机器压测为准。

## 1. 测试工具与运行方式

| 工具 | 覆盖 | 命令 |
|------|------|------|
| Go benchmark（组件级） | 序列化/广播/指标中间件等纯 CPU 热路径，可 benchstat 对比 | `cd backend-go && go test -run '^$' -bench . -benchmem ./internal/platform/...` |
| `scripts/perf-smoke.sh`（全链路只读） | 真实 HTTP + DB + Redis + 中间件，固定 6 个读端点 | 见下 |
| Prometheus/Grafana（线上） | 持续 RPS/延迟/连接池/Redis fail-open | `OBSERVABILITY_ENABLED=1` 部署后查看「HotSpot 后端总览」 |

```bash
# 全链路只读基线（务必对 APP_ENV=test 的 backend-test 实例，避免限流桶污染）
docker compose -f docker-compose.yml -f docker-compose.dev.yml --profile test up -d backend-test
./scripts/perf-smoke.sh --base http://localhost:8082 --duration 10s --warmup 3s --concurrency 10

# CI/回归门禁模式（任一读端点非 200/code!=0、rps 低于下限或 p99 超限即退出 1）
./scripts/perf-smoke.sh --min-rps 100 --max-p99-ms 200
```

脚本默认坐标/关键词已对齐 dev 种子数据（徐州云龙湖 / 用户「林一」），避免了
历史默认「旧金山 + abc」导致附近/搜索测到空结果集；先预热再采样；输出
rps/p50/p95/p99 与业务码分布；业务码非 0 视为失败（后端契约错误走 HTTP 200 +
envelope code，不能只看 HTTP 状态码）。

## 2. 当前基线（backend-test，seed 数据，c=10，5s，warmup 2s）

2026-11-06 本机实测（相对值；复测请用同一命令/同一机器）：

| endpoint | rps | p50 ms | p95 ms | p99 ms |
|----------|-----|--------|--------|--------|
| users/account | ~4400-5400 | 1.2-1.8 | 2.8-4.8 | 4.2-8.5 |
| system/bootstrap | ~4000-4400 | 1.9-2.1 | 4.5-5.2 | 6.9-7.4 |
| notify/fetch | ~7700 | 1.1 | 2.7 | 4.4 |
| moments/fetch_following | ~3500-3700 | 2.1-2.3 | 5.9-6.5 | 12-14 |
| moments/fetch_nearby | ~350 | 14.8-15.2 | 71-72 | 79-81 |
| users/search | ~570-590 | 9.5-9.6 | 58-60 | 64-65 |

组件级 benchmark（同机）：

| benchmark | 结果 | 说明 |
|-----------|------|------|
| BenchmarkBroadcastSinksPrepared（100 连接） | ~2.2 µs/op，13 allocs | 一次 marshal + PreparedMessage 复用（socketSink 生产路径） |
| BenchmarkBroadcastSinksPlain（100 连接，逐连接 marshal） | ~115 µs/op，400 allocs | 回退路径；约 **50x** 差距，禁止退回逐连接序列化 |
| BenchmarkEventMarshal | ~1.1 µs/op，392 B | 单事件 JSON 体积回归基线 |
| BenchmarkWriteJSONPage（20 条列表） | ~12 µs/op，3 allocs | 统一响应 envelope 编码 |
| BenchmarkRequestMetricsMiddleware | ~0.54 µs/op，7 allocs | 每请求指标采集固定税 |

热点解读：nearby（S2 半径查询 + 聚合）与 users/search（trigram ILIKE 双分支）
是当前 p99 最高的两个读端点，容量规划优先关注；两者都已有 pg_trgm/S2 索引，
继续优化方向见 `.workwork/design/backend-improvement-plan.md` Batch D2/D5。

## 3. SLO 目标（单实例基线，c=10，seed 规模）

| SLI | 目标 | 备注 |
|-----|------|------|
| 读接口业务成功率（code=0 / HTTP 200） | ≥ 99.9% | 以 Prometheus `http_requests_total` + 业务日志交叉；错误走 envelope 需看 code |
| users/account、notify/fetch p99 | ≤ 50 ms | 高频轻查询 |
| moments/fetch_following p99 | ≤ 100 ms | 列表聚合 |
| moments/fetch_nearby、users/search p99 | ≤ 150 ms | 索引查询，随数据量线性验证 |
| 5xx 比例 | < 0.1% | 仅框架级异常（业务错误不走 5xx） |
| WS 广播丢弃 | 0（持续 >1/s 告警） | `realtime_ws_dropped_events_total` |
| Redis fail-open | 0（持续 >0 告警） | `hotspot_redis_failopen_total` |

## 4. 容量模型假设（单机小规模）

- 读流量按 10 并发基线乘 N 估算；先验证 p99，再看 RPS 饱和拐点，不以单点 rps
  峰值作为容量上限。
- DB 连接池、Redis、WS 连接数与 goroutine 均已在 Prometheus 暴露（Grafana 面板），
  扩容判断优先看 `pgxpool_empty_acquire_total` 增长率与 p99 是否同步恶化。
- 数据量增长会首先影响 nearby/search（索引选择与回表）；seed 规模无法代表线上，
  上线新体量前应在同构数据量上重跑本手册第 1 节命令并对比第 2 节基线。

## 5. 回归纪律

- 热路径（序列化、广播、指标、分页、缓存）改动需附 benchmark 前后对比（同机同会话）。
- `scripts/perf-smoke.sh` 可作为发版前冒烟：`--min-rps`/`--max-p99-ms` 按机器
  填保守下限，非零退出即阻断发布自查。
- 线上告警口径见 `docker/prometheus-alerts.yml`；采集栈启用方式见
  `docs/release-runbook.md` 与 `.env.example` 的 `OBSERVABILITY_ENABLED`。
