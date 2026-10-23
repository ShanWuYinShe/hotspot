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

## P2 批次修复（2026-10-23 第二轮）

第一轮 P0/P1 修完后，按各域报告的 P2 清单又修了 26 项高价值问题（5 个提交）：

- **安全/校验**：媒体/share 类型消息统一 5000 字 content 上限（堵住 type=image +
  合法 file_id + ~1MB content 的广播绕过）；moments open_the_door 补双向拉黑；
  comments Create 拉黑检查 fail-open→fail-closed；media WebP 元数据清洗（纯 Go
  RIFF chunk 解析删 EXIF/XMP，零新依赖）；privacy_level 枚举校验；验证码投递
  失败不再假成功；Register 昵称 TrimSpace。
- **数据正确性**：公告 CreatedBy 昵称、入群 WS member_count 权威重读、
  fetch_joined 未读排除已撤回、moments Update 坐标精度、search (0,0) 哨兵、
  reactions 批量聚合排序、haversine 双实现收敛、notify prune 复合主键分批、
  幂等回放重发 Set-Cookie（web 注册重试不再丢会话，已端到端验证）。
- **无界/性能/契约**：群公告列表 LIMIT、UpdateAnnouncement 直查、群搜索游标
  精度、StatsGeoDrilldown 幽灵游标、UserExists 吞错、Refresh 日志级别、
  privatemessages 错误映射、reports seq 竞态、users memory search 过滤。
- **网关**：nginx（基线 + 生产 prod-tls）加 `set_real_ip_from`(CF 全部回源网段)
  + `real_ip_header CF-Connecting-IP`——limit_req 改按真实客户端 IP 计数，不再
  按 CF 边缘 IP 共享桶误伤同 POP 用户。
- **清理**：删死代码 3 文件 + 若干死链（-398 行）、cursor 解码去重、itoa64 手写
  替换；顺带修复 main 上「queryListFollowing/Followers SQL 已摘列但 scan 仍旧
  列数」的存量 bug（真 PG 下会 scan 报错）。

**部署拓扑已确认（2026-10-23，站长）**：全部流量必经 Cloudflare 回源，源站
mTLS origin-pull + CF 网段 allow/deny——故 CF-Connecting-IP 由 CF 强制设置、
后端信任 CF 头是正确口径（**不要清洗该头**）；此前「源站可直连伪造 IP」的
担忧不成立，nginx 侧只需 real_ip 还原供 limit_req/X-Real-IP 使用。

## P2 第三批修复（2026-10-23，安全/正确性/配置类）

- **安全**：账号存在性时延侧信道（Authenticate/CreatePasswordReset 的
  not-found 分支不做 bcrypt，攻击者可据耗时枚举账号）→ 补等代价 dummy 比对；
  share 游标由裸 base64(JSON) 改为 platform/cursor 的 AES-GCM（此前可解码/伪造
  (created_at, share_id)）；SMTP 会话补整体 deadline（此前仅 dial 超时，坏网关
  可挂住 handler goroutine）。
- **正确性**：share 撤销幂等分裂统一（按 code 撤销已 revoked 分享此前 404）；
  banUser 审计先于「会话撤销失败」判定落库（spec 要求封禁必审计）；admin 10600
  日志补真实 err（28 处，此前只记合成 sentinel，根因不入日志）。
- **配置**：media 六类上传大小上限从硬编码改为 AppConfig.Upload 驱动（新增
  AvatarMaxSize/ChatFileMaxSize 两字段保持默认行为不变，admin 侧可调）；
  Content-MD5 下发归一为标准 base64（此前是 hex，严格 S3 后端会拒绝）。
- **性能/口径**：moment 删除向收藏者广播加 2000 上限（此前无界全量载入 +
  请求路径逐个广播）；notify 内存分桶删除契约外死类型并订正 groups 三处
  「落库」错误注释；mapmodule/search 内存 stub 显式声明「非语义等价、禁止用于
  降级/生产路径」（已核实生产不可达）。

## P2 第四批修复（2026-10-23/24，性能/架构）

- **性能往返**（2026-10-23）：moments 标签写入逐条 Exec 改单条多值 INSERT；
  moments.Get 由 3 次往返（主查询+tags+mentions）合并为 1 次（LATERAL
  ARRAY_AGG，排序与共享 loader 同口径）；media Complete 由 Get→UPDATE→Get
  三次往返改为条件 UPDATE ... RETURNING 单往返（幂等语义不变，仅未命中时
  回读一次区分不存在/并发已完成）。
- **路由元数据收敛**（2026-10-24）：限流读/写分类（原 ratelimit.classifyRoute）
  与资源守卫规则（原 resourceguard.guardRules）合并为唯一来源
  `internal/platform/routemeta`；新增 `AuditRoutes` 双向审计——已挂载路由必须
  被限流分类覆盖、每条守卫规则必须命中至少一条路由。审计在启动时以 Warn
  暴露漂移，并由 `bootstrap/route_metadata_test.go` 在 CI 硬断言（当前 172 条
  路由全绿）。**审计首次运行即发现真实缺口：`/api/v1/reactions/{fetch,set}`
  从未被分类，限流中间件对未识别路径直接返回空规则集 → 该模块此前完全没有
  限流**，已补分类并加回归用例。

## P2 第五批修复（2026-10-24，内存实现漂移）

内存实现是「DB 未配置时优雅降级」的生产路径（不是纯测试桩），本批把 chat 域
报告的 P2-8 剩余项与 user-admin 的报告项收敛（均附回归测试）：

- **半提交状态**：`AcceptInvite`/`ApproveApplication` 先置 accepted 再查容量，
  群满返回 `ErrGroupFull` 时状态已被污染（「已接受但人没入群」）→ 容量检查
  前置，对齐 PG 事务顺序；
- **数据覆盖**：`RejectApplication` 用拒绝原因覆盖申请人原始留言（PG 不碰
  message）→ 不再覆盖；
- **清理残留**：`Dismiss` 漏清 banned/announcements（PG 六表全清，同 ID 重建群
  会读到幽灵数据）→ 补齐；
- **错误映射**：公告 Update/Delete 在 PG 侧 0 行命中静默成功、内存侧返回
  `ErrAnnouncementNotFound`，而 handler 的 not-found 分支缺失（降级路径落 10600）
  → PG 改条件 UPDATE/`RETURNING` 0 行映射 not-found，handler 补分支，两实现统一；
- **游标不兼容**：内存版成员/邀请/申请列表游标是明文 `时间:id`，与 PG 的 AES-GCM
  游标互不兼容，且 RFC3339 时间串自带冒号无法可靠拆分（分页静默错页）→ 统一改用
  共享 `groupPageCursorCodec`；`ListAdminMembers` 此前完全忽略游标，补 keyset 消费；
- **分页参数被忽略**：内存版 `ListMyInvites`/`ListSentInvites`/`ListApplications`
  全量返回且 next_cursor 恒空 → 补 limit 裁剪与游标；
- **搜索语义**：`SearchMembers` 内存版仅精确匹配，PG 是 ILIKE 子串 → 同步为
  大小写不敏感子串匹配；
- **并发守卫**：admin 内存 `HandleReport` 无 pending 守卫（PG 有 `WHERE status=
  'pending'` + `errReportAlreadyHandled`），降级路径可重复执行解散/封禁副作用
  → 补守卫；

本轮同时确认 user-admin P2-14（内存 `SearchProfiles` 漏过滤 is_deleted/
privacy_level）在早前批次已修复，补记于此。

**遗留观察项**（不紧急，记录在案）：
- 双实现（repo_memory/repo_postgres）漂移仍是本项目最大 bug 温床（四轮共修 35+ 处）；
  本批起 drift 修复均附内存实现回归测试，但跨实现「同输入同输出」对比测试基建仍缺；
- **未修的已知限制**（需产品/架构决策，非缺陷）：
  1. Login/Guest 的同设备逐出为「先撤销旧会话再建新会话」，两请求严格并发时仍
     可能留下两个活跃会话——彻底修需要 auth_sessions 上的部分唯一索引
     (user_id, device_id) WHERE revoked_at IS NULL，涉及存量数据去重迁移，未做；
  2. auth 的验证码 TTL、登录锁定阈值仍为代码常量（迁 AppConfig 需同步前端管理页）；
  3. privatechats List 仍在 service 层全量拉取后内存排序（keyset 下推依赖
     last_activity_at 冗余列，改动面较大）；
- P2 剩余项（各域报告里未修的约定/架构类）与 OBS 项存档于 `.workwork/backend-review/`。

