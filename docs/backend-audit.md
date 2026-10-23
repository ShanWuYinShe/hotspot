# 后端潜在问题审计（2026-10-21）

一次性静态审计 + 代码走查结论。**本文列出的项已确认无问题，除非相关代码大改，不必重复排查**；
新发现的问题请更新本文并注明日期。

## 审计清单与结果

| 检查项 | 方法 | 结论 |
|---|---|---|
| goroutine panic 带走进程 | 全仓 `go func` 逐处核对 | 5 处：worker 包 2 处用 `Go`/`GoRestartable`；`cmd/server` 2 处（http/pprof server，`net/http` 自带 per-request recover）；**hotcache 本地清扫循环原先为裸 goroutine，已改用 `GoRestartable`**（2026-10-21）|
| context cancel 泄漏 | 搜 `_, _ = context.With*` / 丢弃 cancel | 0 处 |
| 循环内 `time.After` 泄漏 | 搜 `time.After` 与 `for` 同段 | 0 处 |
| HTTP client 无超时 | 搜 `http.Client{` | 仅 1 处（对象存储），**已带 Timeout** |
| `rows` 未关闭（连接泄漏）| 逐处核对 `.Query` 的 Close/CollectRows | 全部有 `defer rows.Close()`、显式 Close 或包装器语义（`persistence` 的 Query 返回 rows 给调用方）|
| 事务回滚 | 搜 `Rollback` | 统一在 `persistence.Transaction` helper 内 `defer`，业务层无需手写 |
| 忽略错误 `_ =` | 归类 73 处 | 均为可忽略场景：WebSocket 写失败（连接已坏，read loop 会收尾）、`Set{Read,Write}Deadline`、`tx.Rollback`、`Close` |
| SQL 注入 | 搜 `fmt.Sprintf` 拼 SQL | 8 处全部安全：`$%d` 占位符编号、`LIMIT %d` 用常量、`scoreExpr` 来自 `searchScoreSQL` 的白名单 switch；无用户输入直拼 |
| 无界查询 | 搜 `LIMIT` 与硬上限 | **2026-10-23 修正**：原结论「各列表查询均有 LIMIT」被全面审查推翻——至少 8 例反例（admin users/groups/reports/audit 四条列表全量物化、groups ListMyApplications/ListBannedMembers、announcements/fetch、moments 附近通知目标查询无上限）。全部已于同日修复（keyset+LIMIT / 常量上限），修复明细见 git log 2026-10-23 各「修复(...)」提交。**后续新增列表端点须在 PR 自查 LIMIT 与 keyset 谓词**。|

## 性能基线（同批实测）

- pprof：`cmd/server` 已挂独立 localhost server（`PPROF_ADDR`，默认 `127.0.0.1:6060`，置空可关）；
- pgx 连接池：`POSTGRES_MAX_CONNS=25` / `POSTGRES_MIN_CONNS=2`（env 可配）；
- HTTP server：Read 15s / Write 15s / Idle 60s / ReadHeader 10s；
- Redis 命令超时：dial 500ms、read/write 1s —— **有意 fail-slow**，避免 Redis 故障放大到热路径；
- 本地热读接口（nearby/search/groups）：约 3ms，当前量级不需要读缓存；
- 批量查询：反应数、关注关系等已走 batch 接口，无 N+1 聚合。

## 附：用 curl 造真实流量做性能观察（2026-10-21 实测）

排查/验证指标时经常需要真实 DB 流量，踩过的路径坑记录在此：

- **节点前缀**：匿名 `/public/v1`、注册用户 `/api/v1`、群管理 `/api/v1/groups/admin`、
  平台管理 `/admin/api/v1`（前端 `src/api/endpoints.ts` 是权威对照）。
- **所有业务端点只接受 POST**：GET 返回 `{"code":10000,"message":"请求方法不被支持"}`。
- **登录**：`POST /public/v1/auth/signin`，body `{"login_id","password","client_type":"web","device_id"}`；
  web 客户端 token 走 **cookie**（响应 `data` 里没有 access_token）→ curl 必须
  `-c cookies.txt` 保存、后续 `-b cookies.txt` 带上。否则受保护端点返回
  `{"code":10100,"message":"Access Token 无效或过期"}`。
- **分页**：`PaginationParams = { limit?, cursor? }`（默认 20，最大 100）。
- **`moments/fetch_nearby` 等"流"端点还需位置参数**（reference point），只传 `{limit}` 会得到
  `参数错误` —— 造流量时优先用 `moments/search`（热门模式）这类参数简单的端点。
- **注意**：前端 `bun run browser:qa` 走 **MSW mock**，**不产生后端流量**，不能拿它压后端或验证 SQL 性能。
- **慢查询观察**：`SLOW_QUERY_THRESHOLD=10ms docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d backend`
  → 打接口 → `docker compose ... logs backend | grep "slow query"`（日志含 duration/sql/err）。

## 慢查询实测结论（2026-10-21，真实流量）

用本仓库的慢查询日志（`SLOW_QUERY_THRESHOLD`）在真实流量下扫了一遍读路径：

- 流量：web cookie 登录后，打 `moments/search`（popular/newest/keyword 三种）、
  `moments/fetch_created`、`groups/search`、`users/search` 各 10 轮，其中 `limit=100`；
- **10ms 阈值：0 条慢查询**；
- **5ms 阈值：0 条慢查询**。

结论：本地 dev 数据量下，**读路径查询全部 < 5ms**，未发现需要加索引或缓存的慢查询。
后续数据量上来后，按同法把阈值降到 5–10ms 复测即可定位热点 SQL（日志会打出完整语句与耗时）。
## 全面代码审查（2026-10-23，静态走查 + 修复）

覆盖全部 17 个业务模块 + 26 个平台包的非测试代码（约 6 万行），维度为正确性/性能/
架构/优雅性（本节上方 2026-10-21 基线之外的维度）。完整分级报告存档于
`.workwork/backend-review/`（SUMMARY + 四份分报告，AI 会话中间产物）。
**结论**：架构合理（依赖方向干净、三层纪律执行好、横切基建成熟）；业务模块尾部积累的
数据正确性缺陷已按优先级修复完毕——P0×3、数据正确性/防护/并发 P1×19、平台层口径问题×5，
共 6 个提交（git log 2026-10-23 `修复(...)` 系列），每项带防回归测试。

**已确认无问题、不必重复排查**（各域报告附完整清单，共 46 项）：
- platform：WS 生命周期（CSWSH 防护/连接上限/非阻塞 sink/close 竞态）、resume 机制
  （seq 单调/基线先于注销捕获）、auth token 链（轮换守卫/宽限期/重用吊销/escrow）、
  嵌套事务防池死锁、XFF 右遍历解析、trusted proxy 配置；
- 业务域：token 轮换与会话撤销竞态、RBAC、幂等三项保证、注销链路等（详见分报告）。

**遗留观察项**（不紧急，记录在案）：
- CF 前置部署假设未闭环：nginx 未清洗 CF-Connecting-IP 且无 real_ip 还原/CF 白名单
  （源站可直连时可伪造 IP 绕过限流；limit_req 按 CF 边缘 IP 计数）——需结合线上
  防火墙策略决策后修（涉及 docker/nginx.conf 与 requestmeta 信任模型）；
- ratelimit classifyRoute 与 resourceguard guardRules 两份硬编码路由元数据清单存在
  新端点漂移风险（建议后续收敛为模块注册时声明）；
- 双实现（repo_memory/repo_postgres）漂移是本项目最大 bug 温床（本轮修了 10+ 处），
  建议为高频变更模块补「双实现契约一致性」对比测试。

