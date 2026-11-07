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
# 1) 启动 RATE_LIMIT_BYPASS=1 的 backend-test（跳过后端限流，避免压测把租户/IP 桶打满）
docker compose -f docker-compose.yml -f docker-compose.dev.yml --profile test up -d backend-test

# 2) 给 hotspot_test 灌种子数据——否则 nearby/search 测到的是空结果集（历史失真根因）。
#    复用 dev backend 容器内的 Go SDK 与 /devdata 素材挂载，仅覆盖 DSN 到测试库：
docker exec hotspot-backend-1 sh -c 'export POSTGRES_DSN="$(printf "%s" "$POSTGRES_DSN" | sed "s#/hotspot?#/hotspot_test?#")"; cd /src && go run ./cmd/seed -assets-dir /devdata/seed/media'

# 3) 全链路只读基线
./scripts/perf-smoke.sh --base http://localhost:8082 --duration 10s --warmup 3s --concurrency 10

# CI/回归门禁模式（任一读端点非 200/code!=0、rps 低于下限或 p99 超限即退出 1）
./scripts/perf-smoke.sh --min-rps 100 --max-p99-ms 200
```

脚本默认坐标/关键词已对齐 dev 种子数据（徐州云龙湖 / 用户「林一」），避免了
历史默认「旧金山 + abc」导致附近/搜索测到空结果集；先预热再采样；输出
rps/p50/p95/p99 与业务码分布；业务码非 0 视为失败（后端契约错误走 HTTP 200 +
envelope code，不能只看 HTTP 状态码）。

## 2. 当前基线（backend-test，seed 数据，c=10，5s，warmup 2s）

2026-11-07 本机实测（相对值；复测请用同一命令/同一机器）。本组数字在
D2 优化后采集：PGroonga 规划器启动预热、users/search 回退链改轻量位置查询、
nearby 详情查询合并 reactions/作者关系（4→2 往返）。全部端点 code=0 且
`scripts/perf-budget.sh` 通过：

| endpoint | rps | p50 ms | p95 ms | p99 ms |
|----------|-----|--------|--------|--------|
| users/account | ~9500 | 0.9 | 1.8 | 3.5 |
| system/bootstrap | ~8550 | 1.1 | 1.8 | 3.1 |
| notify/fetch | ~14400 | 0.6 | 1.1 | 2.5 |
| moments/fetch_following | ~10200 | 0.9 | 1.5 | 2.2 |
| moments/fetch_nearby | ~1190 | 4.1 | 52.6 | 58.2 |
| users/search | ~3500（2.4k–5k 波动） | 2.1 | 4.3 | 26.3 |
| moments/search | ~9800 | 0.9 | 1.7 | 3.9 |
| groups/search | ~6200 | 1.2 | 2.3 | 18.7 |

专项对比（users/search keyword=游客，单端点 loadgen c=10/10s）：
- 轻量位置回退修复：1799→2139 rps，p95 25.2→8.5ms，p99 40.2→28.8ms；
- 去掉冗余 ILIKE 复核：2139→**3279 rps**（+53%），p95 5.1ms，**p99 7.3ms**
  （-75%）；全链路 perf-smoke：3562→**5090 rps**，p99 25.3→6.0ms。

nearby p99 仍最高（~58ms），但 D2 三轮优化后已从最初 ~480rps/81ms 提升到
~1190rps/58ms（候选 OR BETWEEN + 详情范围聚合）；DB CPU 仍是瓶颈。users/search
已从 p99 榜第二降至中游。

**读法注意**：Postgres/OS 缓存冷热会显著影响 users/search（实测 2.4k–5k rps /
p99 6–40ms）与 nearby；本表取多次重复中的典型暖态。验收一律以
`scripts/perf-budget.sh` 的相对预算（rps≥60% baseline、p99≤2.5×baseline）
为准，不要用单次绝对值做回归判断。

热点解读：nearby（S2 半径查询 + 聚合）与 users/search（trigram/PGroonga 双分支）
仍是 p99 最高的两个读端点，容量规划优先关注；本轮架构调整未使其退化（相对
baseline JSON 的预算门禁全部 OK）。nearby 的 DB 往返已收敛为 4 次（候选含关注 +
详情含图片 + reactions 聚合 + 作者关系），继续优化方向见
`.workwork/design/backend-improvement-plan.md` Batch D2/D5。

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
- 网关并发上限与后端对齐：nginx 主配置 `docker/nginx.main.conf` 为
  `worker_connections 16384`/worker + `worker_rlimit_nofile 65535`，compose 对
  nginx 与 backend 均设 `ulimits.nofile=65535`。每个 WS 在网关占 2 条连接
  （客户端 + 上游），单实例 10k WS ≈ 20k+ 连接；只调大 `WS_MAX_CONNECTIONS`
  而不改这些配置会在网关侧先触顶（镜像默认 1024 连接 ≈ 500 WS）。
- 上量前复核：`docker exec hotspot-nginx-1 nginx -T | grep worker_connections`、
  `docker exec hotspot-nginx-1 sh -c 'ulimit -n'`、后端容器 `ulimit -n`。

## 5. 回归纪律

- 热路径（序列化、广播、指标、分页、缓存）改动需附 benchmark 前后对比（同机同会话）。
- `scripts/perf-smoke.sh` 可作为发版前冒烟：`--min-rps`/`--max-p99-ms` 按机器
  填保守下限，非零退出即阻断发布自查。
- 线上告警口径见 `docker/prometheus-alerts.yml`；采集栈启用方式见
  `docs/release-runbook.md` 与 `.env.example` 的 `OBSERVABILITY_ENABLED`。

## 6. 短关键词搜索索引（PGroonga，2026-11）

**问题**：中文姓名常见 1–2 字符关键词，pg_trgm 需要 ≥3 字符模式才能走 GIN
索引；短关键词搜索退化为 users 全表 seq scan。100k 用户合成库实测
（keyword=游客，2 字符）：**85 ms/次**（Seq Scan 100k 行）。

**方案**：数据库镜像切换为 groonga/pgroonga:4.0.6-alpine-18（PostgreSQL 18 +
PGroonga 扩展，官方公开镜像），迁移 063 启用扩展并为 users 昵称/简介建 PGroonga
GIN 索引；搜索 SQL 先以 &@ 子串预过滤走索引，再保留原 ILIKE 精确复核，
返回语义不变（1 字符起有效）。

**实测**：同一 100k 库、同一 SQL 形状（keyword=游客）：

| 版本 | 计划 | 执行时间 |
|---|---|---|
| 旧（trgm/seq scan） | Seq Scan on users | 85.3 ms |
| 新（PGroonga + ILIKE 复核） | BitmapAnd(BitmapOr(nickname/bio pgroonga), ...) | **2.1 ms** |

**运维约束**：所有环境（dev/CI/生产/备份恢复）必须使用同一 PostgreSQL 18 +
PGroonga 镜像；只替换为官方 postgres:18 会导致 063 迁移失败、搜索索引缺失。

### 6.1 moments/groups 搜索索引（迁移 064）

同一短关键词问题推广到 moments/search、groups/search：100k 瞬间 / 50k 群组合成
库（keyword=游客，2 字符）实测：

| 端点 | 旧（无可用索引，seq scan） | 新（PGroonga 索引） | 提升 |
|---|---|---|---|
| moments/search | 41.9 ms | 0.73 ms | ~57× |
| groups/search | 19.7 ms | 0.73 ms | ~27× |

迁移 064 为 moments.description、groups.name/description 建 PGroonga GIN 索引；
SQL 增加 &@ 预过滤 + 原 ILIKE 精确复核（语义不变）。

### 6.2 nearby 候选查询拉黑过滤物化（2026-11）

**问题**：moments/fetch_nearby 候选粗筛用两个相关 NOT EXISTS 过滤双向拉黑，
每个候选行各扫一次 user_relations；478 候选时 prepared 执行 6.7ms/次，
backend-test 压测 DB CPU 成为瓶颈（p50 82ms）。

**方案**：CTE 一次物化查看者双向拉黑集合 + m.user_id NOT IN (...)，语义不变。

**SQL 级实测**（hotspot_test，prepared statement，478 候选）：

| 版本 | 执行时间 |
|---|---|
| 旧（相关 NOT EXISTS × 2） | 6.7 ms |
| 新（CTE blocked + NOT IN） | **0.9 ms** |

### 6.3 端点级回归（backend-test，c=10，5s，2026-11）

合并 PGroonga + nearby 拉黑物化后，perf-smoke 8 端点（热点数据：hotspot_test
累计 E2E 数据）实测：

| 端点 | rps | p50 | p95 | p99 |
|---|---|---|---|---|
| users/account | 8864 | 1.0 | 2.0 | 2.9 |
| system/bootstrap | 7435 | 1.2 | 2.3 | 3.3 |
| notify/fetch | 12728 | 0.7 | 1.5 | 3.0 |
| moments/fetch_following | 8460 | 1.1 | 2.1 | 3.2 |
| moments/fetch_nearby | 743 | 7.1 | 54.8 | 60.4 |
| users/search | 2161 | 3.6 | 9.3 | 26.8 |
| moments/search | 6360 | 1.3 | 3.3 | 5.3 |
| groups/search | 4431 | 1.8 | 4.6 | 11.6 |

关键回归对比（同机同数据）：
- moments/fetch_nearby：**134-158 rps / p50 82-85ms / p99 187ms** →
  **743 rps / p50 7.1ms / p99 60ms**（约 5 倍吞吐、p50 约 11 倍）。
- users/search（短中文关键词）：旧基线 570-637 rps / p99 65-68ms →
  **2161 rps / p99 26.8ms**（约 3.5 倍）。

## 7. 性能回归门禁（perf-budget.sh）

`scripts/perf-budget.sh` 把 perf-smoke 的结果与 `docs/perf-baseline.json` 比较，
任一端点 rps 低于基线 60% 或 p99 超过基线 2.5 倍即非零退出（阈值可参数覆盖）。

```bash
./scripts/perf-budget.sh --base http://localhost:8082   # 默认 backend-test
./scripts/perf-budget.sh --update-baseline              # 有意调整基线后刷新 JSON
```

基线口径：backend-test / hotspot_test（含 E2E 累计数据），c=10、warmup=2s、
duration=5s，8 个只读端点。实测两条路径：

- 正常：8/8 端点 OK（rps 均 >= 60% baseline、p99 <= 2.5x）；
- 注入 `--max-p99-mult 0.0001`：exit 1 并逐端点标记 REGRESSION，
  证明门禁确实拦截回归。

该门禁不进默认 CI（共享 runner 噪声大），定位是「改完性能敏感代码/发版前
本机回归」；业务码与传输错误由 perf-smoke.sh 自身校验并透传退出码。

### 6.4 群标签过滤 GIN 索引（2026-11）

**问题**：groups/search 的 tag 过滤为 = ANY(g.tags)；该写法即使建了 GIN(tags)
也不会走索引，选择性标签在 10 万群规模下退化为全表扫描。

**方案**：迁移 065 新增 idx_groups_tags_gin，查询改为数组包含
g.tags @> ARRAY[$1]::text[]（语义等价）。

**实测**（hotspot_perf，100k 群，选择性标签命中 99 行）：

| 写法 | 执行时间 |
|---|---|
| = ANY(g.tags)，无索引 | 13.4 ms（Seq Scan） |
| = ANY(g.tags)，有 GIN | 11.3 ms（仍 Seq Scan，GIN 不可用） |
| @> + GIN | 0.28 ms（Bitmap Index Scan） |

非选择性标签（95% 群命中）仍是顺序扫描 + LIMIT 提前终止（0.3ms 级），无回退。
