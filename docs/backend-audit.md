# 后端潜在问题审计（2026-10-21）

一次性静态审计 + 代码走查结论。**本文列出的项已确认无问题，除非相关代码大改，不必重复排查**；
新发现的问题请更新本文并注明日期。
## 审计总览（第 17 轮，2026-10-24）

本文件为逐次审计的追加记录（约 690 行）。本节汇总**当前整体结论**，便于一次性 review；
细节与证据仍以各分节为准。

### 一、方法与证据链

- **静态**：`go vet` 全仓干净；`staticcheck` 生产代码 0 findings（测试代码噪声不计入门禁）；
  `govulncheck` 可达漏洞 0。
- **门禁**：`./scripts/verify-backend-ci.sh`（默认 / `--with-db` / `--with-e2e` / `--all`），
  与 `.github/workflows/ci.yml` 逐条对齐；审计期间每个提交都跑过 `--with-db`，合并后跑
  `--with-e2e`，主线 CI 全绿。
- **双实现契约测试基建**（`testdb.RunContract`）：同一批用例同时跑**内存实现**与**真实库实现**
  （`hotspot_it`，事务隔离），专门抓「两套实现语义漂移」与「只在真实库暴露的 SQL 缺陷」。
  已接入 **15 批**：privatechats、notify、users、groups、privatemessages、comments、moments、
  share、admin（审计日志 + 分页不变量）、system、reports、reactions、platform/auth 存储、
  media、groupmessages。
- **流程**：worktree 开发 → 契约测试/修复 → `--with-db` 门禁 → ff-only 合并 → `--with-e2e` →
  推送 main → CI 绿 → worktree/分支清理；每轮结论追加到本文件。

### 二、生产缺陷（真实库/线上路径）已修复

| # | 缺陷 | 影响 | 提交 |
|---|---|---|---|
| 1 | groups `AreMembers` SQL 返回 5 列却只 Scan 1 列 | 真实库必报错（feeds 批量成员判定不可用） | `c6d8a7dd` |
| 2 | privatemessages `MarkAsRead` 无 `up_to` 时返回空位点 | 已读事件负载缺 message_id/sent_at | `adc80f4b` |
| 3 | comments `Create` 回复响应 `parent_id` 恒 null | 回复层级信息丢失 | `1a214fba` |
| 4 | system 公告检索分页游标取「下一页首条」 | **每页边界丢一条公告** | `34b0588e` |
| 5 | reactions 点赞目标列表分页同族缺陷 | **每页边界丢一个点赞目标** | `95fbb1a2` |
| 6 | admin 标签/RBAC 管理员列表分页同族缺陷 | **每页边界丢一条** | `a88d64da` |
| 7 | Repository 事务上下文绕过（6 模块 20 处直连池） | 事务内读写逃逸：写回滚不生效/读不到本事务写入 | `a85b3c8f` |
| 8 | moments 收藏者 fanout 查询无 LIMIT | 热门瞬间删除时全量收藏行载入内存（内存放大） | `df493a19` |
| 9 | （此前批次）P0/P1/P2 系列 | 见本文件「全面代码审查」「P2 批次」等分节 | 各节 |

### 三、内存降级实现的语义漂移（已修复）

内存实现用于单测/无 DB 降级，语义漂移不直接影响线上，但会让单测与 E2E 掩盖真实行为。
已修复的主要漂移（按模块，均为契约测试发现）：

| 模块 | 漂移（示例） | 提交 |
|---|---|---|
| notify | 未读分类口径、时间格式 | `2c43ab1c` |
| users | 拉黑列表缺 avatar_url；游标格式/keyset | `cbe33876` 等 |
| groups | 成员判定/共同群计数/批量成员游标 | `c6d8a7dd`、`ff030d64` |
| privatemessages | 已读位点、消息顺序/召回、无效游标 | `adc80f4b` |
| comments | 内存游标非 keyset、回复层级 | `1a214fba` |
| moments | ListMine 明文游标、搜索大小写、软删/计数语义 | `235d7c55` |
| admin | 审计日志/标签/RBAC 管理员分页边界丢行、内存 seed 数据 | `9911d6ba`、`a88d64da` |
| system | repo 层不该做 trim、版本指纹 | `34b0588e` |
| reactions | 明文游标+不校验、reacted_at 秒精度、RFC3339Nano 变长字符串字典序排序 | `95fbb1a2`、`f820f991` |
| platform/auth | 刷新令牌 Rotate 对不存在旧 token 未返回冲突 | `43de30d7` |
| media | 完成时间缺失、file_id/upload_id 混淆、GC 顺序、GC 误删已完成 | `8bea94b4` |
| groupmessages | 已读计数重复累加、MarkAsRead 无成员校验、MarkAllRead 群范围、缺失消息计数、撤回消息 read_count | `fd3cf83f` |

另有系统性「分页游标取下一页首条」缺陷全场扫描（本轮之后又发现 5 处，见
「分页游标缺陷全场扫描」节），审计方法已沉淀：游标必须取**本页最后一条**。

### 四、spec 歧义 / 待产品确认（实现已按双实现一致口径收敛）

| # | 歧义点 | 当前实现口径 | 建议 |
|---|---|---|---|
| 1 | share「我的分享」列表的 `is_revoked`：spec 只定义为撤销状态，未定义 expired 的列表呈现 | 过期与撤销在列表中均呈现 `is_revoked=true`、`preview=null`（PG/内存一致；open_the_door 统一 10300） | spec 补一句 expired 列表口径 |
| 2 | groupmessages `GetReadCount` 对不存在消息的行为 spec 未定义 | PG COUNT 语义返回 0（非 10300），内存已对齐 | spec 明确「不存在返回 0」或改为 10300 |
| 3 | system `bootstrap_version` 的公告域指纹：spec 只要求「公告变更即变化」，未定义实现 | PG 取 `MAX(updated_at)::text`；同一事务内多次变更指纹相同（事务时间语义） | spec 注明显式指纹/计数也可以，或接受现状 |
| 4 | reactions `ListReactions/ListReactedTargets` 游标不透明性 | 已统一 AES-GCM 不透明游标 + 非法游标 10000 | 无需改 spec，仅记录 |

### 五、已知残留 / 非缺陷（已评估，不再重复排查）

- `search`、`mapmodule` 的内存实现是**测试专用占位**（源码注释明确声明不保证语义等价、
  生产不可达），因此不做双实现契约测试；如需覆盖生产语义，应给 Postgres 实现注入
  `persistence.Queryable` stub 做集成测试。
- `reports.TargetOwner`、`groupmessages` 成员/群名在内存模式依赖测试注入（跨模块真值不可得，
  已提供 `SeedTargetOwner`/`SeedMember`/`SetGroupName` 等测试口子）。
- `platform/auth` 的 session/refresh store 直连连接池（不感知 ctx 事务），调用链在事务外；
  `bootstrap` 启动期查询同理——已在「Repository 事务上下文一致性审计」节记录为残留项。
- `admin` 角色/管理员列表全量加载后 Go 侧分页：管理员数量有界，暂可接受；若增长需下推 SQL。
- media `GetObject` 读取整对象（EXIF 清洗链路）：受「declared size == Head size 且 ≤ 类别上限」
  约束，图片类别上限内可接受。

### 六、测试资产与维护

- 契约测试位置：各模块 `*_contract_test.go`（如 `moments/list_contract_test.go`、
  `groupmessages/repo_contract_test.go`）；PG 侧统一用 `testdb.BeginScope` 事务隔离；
  不支持事务的存储（platform/auth）用 pool + 用例级唯一 ID + 显式清理。
- 新增模块接入步骤：定义 `TestXxxRepositoryContract` → 两个 `ContractImpl`（memory/postgres）
  → 用例覆盖「分页不重不漏/过滤/哨兵/回填字段」→ 在门禁 `--with-db` 与 CI 中运行。
- 门禁命令（推送前按改动范围）：`./scripts/verify-backend-ci.sh [--with-db|--with-e2e|--all]`。

### 七、后续建议（非阻塞）

1. 上表 4 项 spec 歧义经确认后统一修订 `spec/`（本轮未擅自改动）。
2. 为 mapmodule/search 的生产实现补 Queryable stub 集成测试（真实聚合语义目前仅靠集成测试）。
3. 管理员列表若规模增长，下推 SQL keyset 分页；其余「全量读」路径已基本收口。
4. 发版前跑一次 `--all`（含 govulncheck），保持门禁与 CI 长期一致。

---


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

## P2 第六批修复（2026-10-24，设备逐出 TOCTOU）

原 Login/Guest 的设备逐出是「先 RevokeByUserDevice 再 IssueSession」两步：同一
设备的两个登录请求严格并发时可双双通过撤销检查、各自插入新会话——同一设备留下
两个活跃会话，旧会话的 refresh token 仍可轮转，与「单设备登录即逐出」语义冲突
（旧审计「未修限制 1」）。

修法（迁移 057 + 应用层重试）：
- auth_sessions 增加部分唯一索引 uq_auth_sessions_active_device：
  (user_id, device_id) WHERE revoked_at IS NULL AND device_id <> ''；迁移内先做
  一次性去重（每组保留最新会话，仅置 revoked_at 不删行），索引用普通
  CREATE INDEX（迁移禁 CONCURRENTLY），幂等可重复执行；
- 平台层 PostgresSessionStore.Save 识别该约束的 23505 → 返回
  ErrDeviceSessionConflict（内存实现同步同语义，降级路径也能暴露竞态）；
- 新增 Service.IssueSessionEvictingDevice：先插新会话，冲突则撤销该设备旧会话后
  重试（最多 2 次），由数据库约束保证最终至多一个活跃会话；
- Login/Guest 改为「快照旧会话 → 逐出式签发 → 关闭被逐出会话的 WS 连接」，
  原「先撤销再签发」的两步窗口消除。

验证：单测覆盖内存唯一性、逐出后仅剩一个活跃会话、8 并发登录最终恰好一个、
空 device_id 不参与约束；真实库 IT 断言索引存在 + 第二次插入被拦 + 逐出流程
收敛为 1 个活跃会话。

## P2 第七批修复（2026-10-24，正确性/分层小项）

- **users 内存游标 keyset 语义**（P2-15）：内存实现四个列表（拉黑/好友/关注/粉丝）
  的游标定位此前要求游标行仍在列表中，游标行被删除（取关/解封）时返回空页且
  静默成功；PG 是 keyset 谓词，游标行消失后仍应继续翻页。改为按「DESC by
  (at, id)」比较定位（与 PG 谓词同序），非法游标统一返回 `ErrInvalidCursor`
  （handler → 10400，与 PG 一致）。
- **admin 角色审计快照直查**（P2-23）：`findRole` 此前 `ListRoles` 全量扫描 +
  线性查找，角色数增长后每次角色写操作 O(全表)。新增 `Repository.GetRole(roleID)`
  直查（PG 复用既有 `getRoleByID` 单行 + 权限批量，内存实现同口径），service
  改用它；补双实现回归测试。
- **报告快照复核**：user-admin 报告中的多数项（UserExists 吞错误、Refresh 日志
  级别、privacy_level 枚举、StatsGeoDrilldown 幽灵游标、AccountData nullable 等）
  在早前批次已修复（代码内有「审查 P2-x」标记），本批逐条确认后不再重复。
- **内存实现定位澄清**：各业务模块构造时统一使用 `NewPostgresRepository(factory)`，
  未配置 DB 时方法返回 `ErrNotImplemented`，**不存在生产降级到内存实现的路径**；
  内存实现仅服务于单测。因此 auth 内存 `Register` 未 mirror PG 的三表写入属测试
  夹具语义差异（无生产影响），不再按缺陷修复——若要覆盖，需为 auth/user 模块引入
  共享内存后端（跨模块内存存储），成本高于收益，记录为已知限制。

## P2 第八批修复（2026-10-24，privatechats 列表分页下推）

**问题**（chat 域 P2-7）：`List` 每次请求全量拉取用户全部会话，再在 service 层对
全量 chatKeys 做 3 次批量聚合（末条消息/未读数/目标上下文）并内存排序裁剪——
翻页成本恒为 O(全量)，且游标编解码越层（应在 Repository）。

**修法**（不新增冗余列，利用既有 `idx_private_messages_chat_key_sent_at`）：
- `Repository.List` 接收 `ListOptions{Limit, Cursor}`，SQL 侧完成排序 + keyset
  + `LIMIT limit+1` 探测：`is_pinned DESC, last_activity_at DESC, target_id ASC`；
  last_activity 由 LATERAL 取「最近可见消息 sent_at」（按该用户 visible_since
  过滤），无消息回退 chat created_at——与 `buildChats` 原口径一致；
- 已删除且窗口内无可见消息的会话在 SQL 内排除（等价原 buildChats 跳过逻辑），
  JOIN users 过滤已注销目标，避免分页空页；
- 游标编解码移入 Repository 层（`decodeChatCursor`/`encodeChatCursor`），service
  只透传 limit/cursor 并拿到下一页游标；内存实现与测试桩共用
  `paginateChatStates` helper，保持同一套 keyset 语义；
- `buildChats` 的批量聚合只处理当前页（≤100 个 chatKey），单页成本从 O(全量)
  降为 O(页大小)。

**已知差异**：内存实现不持有消息表（消息在 privatemessages 模块），无消息时间可
用，`last_activity` 回退 chat created_at；内存实现仅服务单测，PG（生产/E2E）由
真实库 IT 覆盖排序/分页/置顶/删除过滤/非法游标全场景。

**验证**：新增 `TestPostgresIT_ListPrivateChatsKeysetPagination`（真实库：按消息
时间排序、跨页 keyset、置顶优先、已删除无消息排除、非法游标 10400 语义）。


## P2 第九批修复（2026-10-24，双实现契约测试基建 + auth 参数配置化）

**双实现契约一致性测试基建**（审查遗留项：repo_memory/repo_postgres 漂移是
本项目最大 bug 温床，此前每次修漂移都是一次性用例，缺固定范式）：
- `platform/testdb` 新增 `RunContract` / `ContractImpl` / `ContractCase`：同一组
  用例分别跑内存实现与真实库实现（每个用例独立实例 + 事务回滚隔离），失败
  输出「实现名/用例名」以区分「两实现漂移」与「共同 bug」；
- 新增 `testdb.BeginScope`：调用方自行控制测试事务生命周期（契约测试需要
  「先建实现、跑多用例、统一回滚」）；
- 首个接入方：`privatechats/list_contract_test.go`（7 条契约：空列表、创建后
  可见、GetOrCreate 幂等、分页稳定不重不漏、limit 裁剪、置顶优先、非法游标）。
  **首跑即抓到一处真实漂移**：内存实现把非法游标吞成空页（PG 返回
  `ErrInvalidCursor`），已修正为两实现同语义。

**auth 阈值迁 AppConfig**（审查未修限制）：`AppConfig` 新增 `auth` section
（`verification_code_ttl_minutes` / `login_lockout_threshold` /
`login_lockout_minutes`，`Effective()` 对零值回退迁移前常量 10 分钟 / 5 次 /
15 分钟，存量配置行为不变）；auth 模块每次登录前把阈值同步到 tracker（内存与
Redis 实现都按配置生效、失败计数衰减窗口随锁期变化），验证码 TTL 也从配置读取。
**遗留依赖（已在「P0 契约缺口修复」批次解决）**：`spec/api_spec_admin.md` 的
`app_config/update` 原先未定义 auth section；用户授权改契约后已补入 spec 并接通
管理端 API（见下节）。

**遗留观察项**（不紧急，记录在案）：
- 契约测试基建已就位，但仅 privatechats 接入；后续对 notify/groups/users 等高频
  漂移模块逐步补契约用例（优先覆盖分页/错误映射/状态流转三类）；
- 双实现对比缺口示例：privatechats 内存实现无法复现「最近消息时间」排序（消息在
  privatemessages 模块，内存实现无跨模块存储），后续若引入共享内存后端可解决；
- P2 剩余项（各域报告里未修的约定/架构类）与 OBS 项存档于 `.workwork/backend-review/`。


## P0 契约缺口修复（2026-10-24，spec 审查落地）

对 `spec/` 全量审查（23 文件 / 8871 行，含实现与 Web/Harmony 交叉核对）后，按用户授权
同步修正契约与实现缺口；完整审查报告存档 `.workwork/spec-review/REPORT.md`。

**契约修正（P0/P1/P2）**：
- **bootstrap 缓存命中口径统一**：命中仅省略公共段 `meta`/`client_config`，私有段
  `profile`/`unread_summary` 恒返回（base 与 system.md 三处矛盾说法收敛为与实现一致）；
- **媒体读取契约补齐**：新增 `GET /public/v1/media/avatars/{file_id}`（匿名 302 代理）
  与「媒体读取」章节；`*_url` 规则放宽为「绝对 URL 或 `/public/v1/...` 代理相对路径」，
  与实现（avatar_url 存相对路径）及客户端解析方式对齐；补充 md5 = 32 位十六进制定义；
- **Token 失效语义改写**：删除「黑名单机制」实现描述，统一为**会话撤销、立即生效**
  （base/auth/public/authz 共 10+ 处）；JWT 载荷说明补 `session_id`；
- **tie-breaker 规则放宽**：删除「方向必须与主序一致」强制句（与 chat/reaction/group/admin
  等 7 个既有端点及实现冲突），改为「必须全序且游标谓词同序」；
- **计数器一致性**：删除「异步累计（队列+定时刷新）」机制描述（实现为同步更新，机制
  本不属契约），保留最终一致与客户端约束；
- **Idempotency-Key 格式**：由强制 `{client_id}-{timestamp_ms}-{random}` / 57-63 字符
  改为不透明 32-64 字符 + 推荐结构（服务端只校验长度与字符集；Web/Harmony 均把
  client_id 截断到 24 字符，实际约 51 字符）；
- **限流/配置字段补齐**：`ws_handshake_per_minute`（IP 握手限流）入 base 限流表与
  `RateLimitConfig`；`avatar_max_size`/`chat_file_max_size` 入 `UploadConfig`；
  `hot_moments_count` 入 `ResponseDefaults`；
- **非 RPC GET 例外登记**：`/healthz`、`/public/v1/media/avatars/{file_id}`、
  `/share/{code}`（爬虫 og 落地页，share.md 补说明）、`/metrics`；
- **文档纪律清理**：5 处端点错误码重复列隐含码（10000/10100）移除；`auth/fetch_sessions`
  补排序声明；CF IP 校验改为「部署拓扑保证」；WS 心跳改为固定值（30/10/90 秒）；
  authz 中「黑名单」歧义改为「拉黑关系」。

**实现对齐（配置缺口 + 空配置消费）**：
- **admin `app_config` 新增 `auth` section**：可读可写（Patch），默认 10/5/15 分钟/次，
  非正整数返回 10000；spec 已同步，管理页以只读 JSON 展示；
- **`response_defaults.group_announcements_count` 真正生效**：groups 公告列表条数改由该
  配置驱动（默认 10，repo 硬上限 50），内存实现同步排序（is_pinned/created_at/id）；
- **`response_defaults.hot_moments_count` 真正生效**：`moments/search` 热门模式未显式
  传 limit 时默认取该配置（默认 20，上限 100）；
- 补单测：admin auth section 读写/校验、groups 公告条数、moments 热门条数。

**验证**：`gofmt` / `go vet` / `go test -short ./...`；`./scripts/verify-backend-ci.sh
--with-db`；前端 `bun run check`（svelte-check + eslint + 主题/跨端一致性检查）全绿。


## 双实现契约测试第二批（2026-10-24，notify）

契约测试基建（`testdb.RunContract` + `BeginScope`）第二个接入方：notify 列表/未读/已读。

- 新增 `TestNotifyRepositoryContract`：同一组用例跑内存实现与真实库实现，覆盖落库后
  可读（payload type 归一化）、排序（`created_at DESC, notify_id DESC`）、keyset 分页
  不重不漏、`type`/`is_read` 过滤、六类未读计数、已读语义（ids/all + 跨用户隔离）、
  非法游标（`ErrInvalidNotifyRequest`）。**首跑抓到两处真实漂移**：
  1. 内存实现 `UnreadCount` 只累加 social/interaction/group/system，
     `report_resolved`/`report_dismissed` 与 `nearby_new_moment` 只计入 Total、
     ByType 恒 0（PG 正确）→ 补齐 report/nearby 分支；
  2. 真实库 `List` 把 TIMESTAMPTZ 直接扫进 string，返回 PostgreSQL 风格
     `2026-10-24 09:18:41+00`，违反契约「UTC ISO 8601」→ 改为扫 `time.Time` 后统一
     `Format(time.RFC3339)`（游标编码随之规范，SQL `$4::timestamptz` 仍可解析）。
- 后续接入优先级：groups（成员/申请列表）、users（关系列表）——均属分页 + 状态流转
  高频漂移面。

## 静态分析清零 + 热门搜索时钟一致性（2026-10-24，第十批）

### 工具化审计（首次纳入 staticcheck / govulncheck）

- `staticcheck ./...`：生产代码 7 处死代码（U1000），全部清理：
  - `comments/repo_postgres_queries.go` 旧递减语句（已被「软删+父评论递减」单条原子 SQL 取代）；
  - `groups/repo_postgres_queries.go` 旧在线成员计数（已由列表 LATERAL 子查询取代）、
    旧群主降级语句（已被 TransferOwnership 的 RETURNING 变体取代）；
  - `reactions/repo_postgres.go` 未使用 sentinel、`users/handler.go` 未使用泛型错误响应、
    `resourceguard/middleware.go` 未使用类型别名、`admin/types.go` 未使用转换函数。
  测试文件中的 U1000/S1011 等风格告警属测试基建噪音，未纳入门禁（不引入新工具链依赖）。
- `govulncheck ./...`：**代码可达漏洞 0**；依赖树仅 1 条不可达漏洞（不调用），无需处理。

### 修复：moments/search popular 分页时钟源不一致（真实缺陷）

- **问题**：popular 排序分值的衰减时钟在 SQL 里用数据库 `now()`（量化到 15 分钟桶），
  而 Go 侧编码游标时用**应用进程时钟**（`SearchDecayClock(time.Now())`）。两者跨桶边界
  （或应用/数据库时钟偏差）时，游标携带的分值与下一页 SQL 排序分值不一致，
  keyset 谓词 `(score, created_at, moment_id) < (...)` 会漏行或重行。
- **修复**：衰减时钟在 `Search()` 内取一次，量化后作为 SQL 参数（epoch 秒）下传；
  ORDER BY、keyset 谓词与游标编码三者同源，SQL 不再出现 `now()`。
  跨桶翻页（>15 分钟）仍有设计内的弱一致窗口，但同一次请求内绝对一致。
- 回归：SQL 形状断言改为「无 now() + 时钟作为绑定参数且位于 LIMIT 前」。

## 双实现契约测试第三批（2026-10-24，users）

`TestUsersRelationshipContract` 接入契约基建：同一组用例跑内存实现与真实库实现，
覆盖关注幂等（changed 语义）、关注/粉丝列表、好友列表（`followed_back_at`）、
取关清边、拉黑级联（双向解关 + 拉黑列表字段）、`BlockedEitherWay`、
`RelationshipBatch` 与单个查询一致、following keyset 分页不重不漏、非法游标。

**首跑抓到 1 处真实漂移并修复**：内存版 `ListBlocked` 只填 `user_id`/`nickname`，
漏掉头像（PG 版 `queryListBlockedUsers` 返回 `avatar_url`），导致同一数据在内存实现
（单测/无 DB 场景）与真实库下 API 字段不一致——补齐 `AvatarURL` 映射。

用例准备说明：两实现预置数据等价（PG 用 `testdb.InsertUser`（presence=online）+
UPDATE avatar，内存用 `PutProfile` 对齐 nickname/avatar/presence/last_online），
分页断言只校验「不重不漏 + 页大小」，不绑定同秒并列顺序。

## 双实现契约测试第四批（2026-10-24，groups）

`TestGroupsRepositoryContract` 接入契约基建，覆盖：建群/owner 成员种子、入群与成员列表
keyset 分页（role 过滤）、需审批入群的 pending 申请去重、invite_only 拒绝直加、
公告 pinned 优先 + 默认/硬上限截断、禁言/封禁/解封状态机、退群与「我的群组」、
共同群聊计数、非法游标。

**抓到 2 处真实问题**：

1. **生产缺陷：`AreMembers` 在真实库必然报错**（`repo_postgres.go`）。该批量成员判定
   复用了 `queryGetGroupMemberships`（SELECT 5 列）却只 `rows.Scan(&groupID)` 一个目标，
   pgx 直接返回「number of field descriptions must equal number of destinations, got 5 and 1」。
   WS 订阅批量鉴权（`groupAllowedBatch`，单次最多 100 个群）因此**静默回退逐条 IsMember**，
   批量优化从未生效，订阅延迟与 DB 往返被放大。修复：新增专用单列查询
   `queryAreGroupMembers`（只取 group_id），语义不变。
2. **内存实现 `CountCommonGroups` 漂移**：基于 `groupsByUser` 测试数据而非 memberships
   计算，经真实 Create/Join 流程后恒返回 0（PG 为成员交集）→ 改为按成员关系交集计算，
   已解散群由 groups 表缺失隐式排除。

**联动修正**：内存 `CountCommonGroups` 改为成员语义后，users 模块两处依赖
`SeedGroupsByUser` 夹具的 handler 测试（共同群计数）失效——改为经真实建群 + 入群流程
预置成员关系（新增 `seedCommonGroup` helper），`SeedGroupsByUser` 保留给 `ListGroupsByUser`
聚合场景。

**门禁覆盖修正**：契约测试的真实库半区此前不在 `-run TestPostgresIT` 正则内，CI 与
`scripts/verify-backend-ci.sh` 均改为 `-run "TestPostgresIT|Contract"`，避免此类回归
只在手工命令里验证。

## 双实现契约测试第五批（2026-10-24，privatemessages）

`TestPrivateMessagesRepositoryContract` 接入契约基建，覆盖：发送/双方历史视角、
keyset 分页不重不漏、`Recent` limit 下推与 `Last`、未读数与已读游标（up_to /
无 up_to / 只前进 / 跨会话 10300）、`is_read` 双方视角、撤回不计未读、非法 reply_to、
批量读（`LastByChatKeys`/`UnreadCountsByChatKeys`）与单查一致、`CountMessagesByTarget`。

**抓到 1 处生产缺陷 + 5 处内存漂移**：

1. **生产缺陷：`MarkAsRead`（无 `up_to_message_id`）返回空位点**。PG 分支只执行
   mark-all SQL 后返回 `MessageEventMeta{TargetUserID}`，`message_id`/`sent_at` 为空；
   service 据此广播 WS `PrivateMessageRead`，而契约该事件 `up_to_message_id` 为必填
   字符串（spec api_spec_websocket.md）→ 对端拿不到已读位点。修复：SQL 末尾
   `SELECT` 返回被推进到的最新消息 id/sent_at（upsert 仍只前进），repo 返回完整 meta。
2. 内存 `Recent` 经 `History` 默认 20 条窗口截断，`limit>20` 时静默少返回 → 直接基于
   会话快照过滤 + 截断；
3. 内存 `UnreadCount` 未排除已撤回消息（PG `is_recalled = FALSE`）→ 补齐；
4. 内存 `MarkAsRead` 允许已读游标回退（再次标记更早消息会回退游标/未读反弹）→ 与 PG
   upsert 的只前进守卫对齐；
5. 内存历史 `is_read` 恒 false（发送方永远看不到对方已读；接收方看自己收到的消息也非
   已读）→ 按 PG `queryPrivateMessagesHistory` 的 CASE 语义在读取时计算；
6. 内存 `Send` 对非法 `reply_to_message_id` 静默忽略（PG 返回 `ErrMessageNotFound`）
   → 补齐校验。

用例准备：PG 侧 `testdb.InsertUser` 预置三用户，内存侧无需用户表；批量读用例按调用方
约定只校验本人会话 key（PG 与内存的批量 SQL/实现都不校验参与者，由调用方保证）。

## 双实现契约测试第六批（2026-10-24，comments）

`TestCommentsRepositoryContract` 接入契约基建，覆盖：创建与顶层列表、回复递增
`replies_count`、软删（回复/顶层，含重复删除幂等与父计数只减一次）、顶层与回复
keyset 分页不重不漏、批量取每父最新回复。

**抓到 1 处生产缺陷 + 1 组内存漂移**：

1. **生产缺陷：`comments/create` 返回的回复对象 `parent_id` 恒为 null**。PG 版
   `Create` 落库时正确写入 parent_id，但返回的 `CommentItem` 漏填该字段；内存版有填。
   客户端创建回复后无法直接据响应串接回复（需再拉一次列表）。已修复返回投影。
2. 内存实现列表游标与 PG 不兼容：`List`/`ListReplies` 用**明文 comment_id** 作游标
   （PG 为 AES-GCM 不透明 `(at,id)` keyset），非法游标被静默忽略（PG 返回
   `ErrInvalidCursor`）；同时列表读取 moment 索引里的**创建时副本**，回复/软删后
   `replies_count`、`content`/`is_deleted` 陈旧。修复：复用 `encode/decodeCommentCursor`
   实现同语义 keyset（顶层 DESC、回复 ASC），读取以 `byID` 为真值来源，非法游标统一
   报错。

**测试基建**：新增 `testdb.InsertMoment`（comments 等依赖 moments 外键的用例前置数据）。

## 双实现契约测试第七批（2026-10-24，moments）

`TestMomentsRepositoryContract` 接入契约基建，覆盖：创建/读取字段回环、我的瞬间
keyset 分页、搜索（关键词大小写不敏感 + 标签过滤 + 隐私/删除排除 + 查看者收藏态）、
收藏生命周期（幂等/取消/占位项/分页）、计数维护（评论/分享/reaction 钳 0）、
编辑 patch 语义、软删语义（含删除不存在瞬间）。

**抓到 3 处漂移 + 1 处潜在连接使用缺陷**：

1. 内存 `ListMine` 用**明文 moment_id** 游标（PG 为 AES-GCM 不透明 `(at,id)` keyset），
   且按 id 字符串排序而非 `(created_at, moment_id)`；改为复用 `encode/decodeMomentMineCursor`
   与 SQL 同序 keyset；
2. 内存搜索关键词判定 `strings.Contains` **区分大小写**（PG 为 `ILIKE` 不敏感），搜索
   命中集与真实库不同；抽取 `containsFold` 同时用于过滤与 relevance 打分，与 SQL 对齐；
3. PG `Delete` 丢弃 Exec 行数，删除不存在瞬间静默成功（内存返回 `ErrMomentNotFound`）→
   SQL 加 `RETURNING` 并经 QueryRow 判定，0 行返回 `ErrMomentNotFound`；
4. **潜在连接使用缺陷**：PG `ListMine`/`ListFavorites` 在 rows 未关闭时发起嵌套查询
   （loadMomentTags/Mentions）。生产读路径走 pool 多连接掩盖了问题，但在事务/单连接场景
   直接报 `conn busy`（契约测试在 tx scope 下复现）→ 早退分支先 `rows.Close()` 再嵌套查询。

**测试基建说明**：moments PG 实现存在上述「rows 未关闭时嵌套查询」形态，契约测试的 PG
实现改用 pool（`testdb.Open`）+ 用例级唯一用户 id + 显式清理（子表 → moments → users），
而非 `BeginScope` 事务包裹；清理逻辑可作为后续大模块契约测试的模板。

另记录一处 repos 层语义差异（经 service 预检后不可达）：非作者 `Update` 内存返回
`ErrPermissionDenied`、PG 因 `WHERE user_id` 0 行返回 `ErrMomentNotFound`；契约用例接受两者。

## 双实现契约测试第八批（2026-10-24，share）

`TestShareRepositoryContract` 接入契约基建，覆盖 location 类型分享：创建/读取回环
（分享码字符集与长度、creator、payload、preview、created_at）、分享码唯一性、未知码
10404、过期分享（open_the_door 不可用但解析接口可命中且列表呈现）、撤销（幂等/归属校验/
列表与 preview 抑制）、`我的分享` 分页不重不漏与游标校验、创建者隔离。

**本轮结论**：

1. 未发现生产缺陷；确认两实现的一致口径：**过期分享在「我的分享」列表里呈现为
   `is_revoked=true` 且 `preview=null`**（PG 显式后处理，内存实现同步），`is_revoked`
   由此同时承担「撤销」与「过期」两种不可用状态。
2. **spec 歧义（观察项）**：`spec/api_spec_share.md` 的 `ShareListItem.is_revoked` 只
   定义为撤销状态，未说明 expired 的列表呈现；客户端无法从列表区分「已撤销」与
   「已过期」。当前实现口径一致且不影响 open_the_door 语义（过期/撤销统一 10300），
   建议 spec 补一句列表口径说明（本轮未擅自改 spec）。
3. 已知结构限制（非缺陷）：user/group/moment 类型的 preview 依赖跨模块真值（用户目录/
   目标表），内存实现天然无法对齐（如 creator nickname 回退 user_id、preview 只用 ID）
   → 契约测试只覆盖 location 类型；该限制与既有 `groups`/`moments` 内存实现同类。

### 契约测试 PG 实现的事务隔离纪律（2026-10-24 门禁实测）

- `go test ./internal/modules/...` 默认**跨包并行**执行；契约测试的 PG 实现若用 pool 直写
  （而非 BeginScope 事务回滚），测试行会在并行窗口内被其它包的全库断言看到——
  `admin` 的 `TestPostgresIT_StatsGeoAggregation`（基线+增量 totals 断言）即被 moments
  契约测试的 pool 直写触发偶发失败。**纪律：契约测试 PG 实现默认用 `testdb.BeginScope`
  （事务回滚隔离）**；只有实现路径本身无法在单连接事务下执行时才退化为 pool + 显式清理，
  并在用例注释说明原因。moments 读路径的「rows 未关闭即嵌套查询」已修复（先 `rows.Close()`），
  因此 moments 契约测试已回归事务隔离。

## 双实现契约测试第九批（2026-10-24，admin 审计日志）

`TestAdminAuditLogContract` 接入契约基建，覆盖审计日志：排序（created_at DESC + log_id
DESC tie-break）、四类字段过滤（action/operator_id/target_type/target_id，含组合与无命中）、
时间区间闭区间（两端边界值均命中）、keyset 分页不重不漏、`includeIPAddress` 开关只影响
`ip_address`（`ip_location` 恒可见）、details 与 nullable 字段回环、非法游标 → 10000。

**抓到 1 处漂移（内存实现）**：

- 内存 `ListAuditLogs` 的 `next` 游标取「第 limit+1 条」（下一页首条）而非「本页最后一条」，
  而解码谓词是严格排他 `(created_at, log_id) < 游标` → 下一页把游标项自身排除，
  **每个分页边界静默丢一条记录**（5 条数据 limit=2 只能翻出 4 条）。已改为取本页最后一条，
  与 PG `queryAuditList` 的 `(created_at, log_id) < ($7,$8)` 语义一致。

注：内存实现构造函数会预置 2 条演示审计数据，契约用例在 New 中清空（仅测试代码），
不影响生产（PG 为空表启动）。

## 双实现契约测试第十批（2026-10-24，system 公告）

`TestSystemAnnouncementContract` 接入契约基建，覆盖：创建/读取字段回环、公开列表（倒序、
limit、软删过滤）、后台检索（severity 过滤、keyset 分页不重不漏、含已删记录）、编辑 patch
与已删/不存在哨兵、删除幂等（重复删除成功、缺失报错）、`AnnouncementsVersion` 在
创建/编辑/删除后均变化。

**抓到 1 处生产缺陷 + 1 处内存漂移**：

1. **PG 与管理后台线上表现：公告检索分页每页边界丢一条**。`SearchAnnouncements` 把 `Next`
   游标设为「第 limit+1 条（下一页首条）」，而 SQL 谓词是严格排他
   `(created_at, announcement_id) < 游标` → 下一页把游标项自身排除。内存实现同构（复制了
   同一写法）。已同时改为取**本页最后一条**作游标（与 audit 日志、moments 等列表的正确
   写法一致）。影响：管理后台公告列表翻页时每页少一条、第 N 页后数据不一致。
2. 内存 repo 在 Create/Update 时对 title/content/action_url 做 `TrimSpace`，PG 原样持久化
   → 契约定型为「repo 只负责持久化，裁剪属 service 职责」（admin service 已 TrimSpace）；
   移除内存 repo 的裁剪，两实现一致。

## Repository 事务上下文一致性审计（2026-10-24）

Repository 双实现约定：凡可能被事务包裹的读写必须经 `persistence.RepoQueryable(ctx, factory)`
（或 `QueryableForTx`）取 queryable —— ctx 携带事务时复用该事务连接，否则回落连接池。
审计发现 **6 个模块共 20 处直接 `factory.DB.(persistence.QueryableDB)`**，绕过事务上下文：

- `reports`：Create/FindExistingReport/TargetOwner；
- `groups`：loadInvite/loadApplication（AcceptInvite/AcceptApplication 事务内调用）；
- `privatechats`：upsertStateImpl/getStateImpl（UpsertState 等事务包装内）；
- `privatemessages`：GetMessageForReply/GetMediaForMessage/getMessageMetaImpl；
- `users`：listFriends/listFollowing/listFollowers/listBlocked + execInTx；
- `groupmessages`：lookupMessageMetaFull。

**风险**：事务内读走池 → 读不到本事务刚写入的行（read-your-writes 失效，快照不一致）；
事务内写走池 → **写操作逃出事务**（回滚后仍生效，原子性被破坏；对持有行锁的事务还可能造成
自锁/等待）。生产上这些代码路径兜底可用（池连接能写成功），但事务语义名存实亡。

已全部改为 `persistence.RepoQueryable(ctx, factory)`；`privatemessages.getMessageMetaImpl` 原先
手写「ctx 优先、池回落」也统一收口到同一 helper。bootstrap/app.go 与 platform/auth 的
session/refresh store 仍直接用池：前者为启动期/无事务路径，后者调用链在事务外；若后续纳入
`RunInTransaction` 需一并改造（已记录）。

## 双实现契约测试第十一批（2026-10-24，reports）

`TestReportsRepositoryContract` 接入契约基建，覆盖：创建/列表回环（含消息举报快照
`message_snapshot` 冻结存储、非消息举报为空）、`status` / `target_type` 过滤、keyset 分页
不重不漏、非法游标 → 10000、`FindExistingReport` 去重键按（举报人, 目标类型, 目标 ID）
隔离、`TargetOwner` 对 user/moment 真实目标解析与缺失/未知类型回落。

该契约测试同时验证本轮事务上下文修复：reports repo 现在可在 `testdb.BeginScope` 事务内
读写（此前直连连接池，契约测试无法事务隔离）。

## 双实现契约测试第十二批（2026-10-24，reactions）

`TestReactionsRepositoryContract` 接入契约基建，覆盖：React 三态（inserted/switched/
unchanged）、Unreact 幂等与不存在 no-op、点赞列表（倒序/emoji 过滤/keyset 分页）、切换
emoji 刷新 reacted_at（列表重排）、聚合（计数/排序/reacted_by_me 匿名与登录视角/目标隔离）、
`AggregateBatch` 与逐目标聚合等价（含空批与缺失目标）、`ListReactedTargets` 分页/时间倒序/
类型隔离/无效游标。

**抓到 1 处生产缺陷 + 2 处内存漂移**：

1. **PG `ListReactedTargets` 分页每页边界丢一个目标（生产缺陷）**：游标取的是第 `limit+1`
   行（下一页首条）而非本页最后一条，配 strict `<` 谓词 → `users/fetch_reacted_moments`
   翻页每页少一条（尾页可能直接为空）。已改为取本页最后一条；
2. 内存 `ListReactions`/`ListReactedTargets` 用**明文 user_id / target_id 作游标**且对非法
   游标静默从头开始；PG 为 AES-GCM 不透明游标 + `ErrInvalidCursor`。已改为同一套
   encode/decode helper、`(reacted_at, id)` 严格 keyset 谓词、非法游标返回哨兵；同步更新
   旧单测期望（此前断言「未知游标从头开始」）；
3. 内存 `reacted_at` 秒级精度（PG 为 RFC3339Nano）→ 对齐纳秒，避免响应精度与游标精度
   低于真实库。

### 补充修复：内存点赞列表按字符串排序时间导致分页重叠（2026-10-24）

上一提交把内存 `reacted_at` 对齐为 RFC3339Nano 后，`-race` 门禁暴露新问题：内存 `entriesSorted`
与 `ListReactedTargets` 直接比较 RFC3339Nano **字符串**。该格式的小数位会去掉尾零（变长），
字符串字典序与真实时间序在「一串是另一串前缀」时相反（如 `...15403Z` vs `...154035Z`），
游标谓词却按解析后的时间比较 → 排序与谓词不一致，翻页出现重叠/丢行（压测 20 次可稳定复现）。
已改为按 `parseReactionTime` 解析后的 `time.Time` 比较（同秒 tie-break 仍按 ID 降序）。

教训：时间字段一律按解析后的时间比较，禁止对 RFC3339Nano 变长字符串做字典序比较。

## 分页游标缺陷全场扫描（2026-10-24）

在 reactions 契约测试抓到「next 游标取下一页首条 + 严格排他谓词 → 每页边界丢一行」后，
按两类代码特征对全部双实现做了一次全场扫描：`idx >= limit` 早退分支、`page.Next =` 赋值点。
新发现 **另外 5 处同族缺陷**（此前未覆盖的列表）：

- `admin` 标签管理列表 `SearchTags`：Postgres 版（SQL `HAVING COUNT(*) < $2 OR (= AND name > $3)`）
  与内存版（`usageCount > cursor.usageCount` / 相等时 `name <= cursor.name`）均把第 limit+1 条
  当游标 → 每个分页边界丢一个标签；
- `admin` 内存版公告检索 `SearchAnnouncements`（无 DB 降级路径）同缺陷；
- `admin` RBAC 管理员列表 `ListAdmins`：内存版与 Postgres 版（先全量加载再 Go 过滤）同缺陷，
  每页边界丢一个管理员。

均已改为「游标取本页最后一条」，并新增 `pagination_contract_test.go` 锁定不变量：翻页必须能
枚举全部行、不重不漏（覆盖内存/真实库标签列表、内存公告列表、内存 RBAC 管理员列表）。

**审计方法沉淀**：新增/修改列表分页后，用 `grep -n 'idx >= limit\|page.Next ='` 复核游标
取值来源；游标必须是**本页最后一条**，不能是第 limit+1 条（严格排他谓词会把它自身排除）。

## 认证存储契约测试（2026-10-24，platform/auth 会话 + 刷新令牌）

`TestSessionStoreContract` / `TestRefreshTokenStoreContract` 接入契约基建（第 13 个接入方），
覆盖：会话 Save/Get 字段回环、Touch 刷新活跃时间（revoked/missing no-op）、ListActiveByUser
（排除 revoked + 活跃时间倒序 + 用户隔离）、Revoke 幂等与缺失哨兵、RevokeOthers 计数与
排除当前/已撤销会话、同设备活跃会话冲突（内存 TOCTOU 检查 vs PG 部分唯一索引）；刷新令牌
Save/Get 回环、Rotate 链接旧 token（RotatedTo/GraceUntil/GraceTokenEnc/next 链）、二次轮换
与不存在旧 token 均返回 `ErrRefreshRotationConflict`、Save upsert 覆盖。

**抓到 1 处漂移**：内存 `InMemoryRefreshStore.Rotate` 对**不存在的旧 token** 直接插入 next，
而 PG 的 `UPDATE ... WHERE rotated_to IS NULL` 对不存在 token 是 0 行命中 → 返回
`ErrRefreshRotationConflict`。两实现行为不一致（内存模式会让失效 token 轮换出新链）。已改为
同样返回冲突。

**测试基建说明**：`PostgresSessionStore`/`PostgresRefreshStore` 的 `queryable()` 不感知 ctx
事务（见「事务上下文」残留项），故契约测试用 pool + 用例级唯一 ID + 显式清理（refresh →
session → user）。已知差异（有意保留，记录备查）：内存实现 Get 对过期/空闲会话做惰性过期
（降级模式防内存增长），PG 依赖 sweeper 与调用方过期判断；两者在正常 TTL 内行为一致。

## 双实现契约测试第十四批（2026-10-24，media 上传会话）

`TestMediaRepositoryContract` 接入契约基建，覆盖：pending 会话 Save/Get 字段回环、
Complete 状态机（一次性完成、最终元数据/幂等键/完成时间落库、二次完成不改写）、
`ResolveByFileIDs`（owner + category + completed 三重过滤、缺失/空输入）、
`CompletedFileMetaByID`（不限定 owner、未完成/不存在不可读）、`ListStaleUploads` 保留期 GC
（只列未完成、created_at 升序、limit）与 `DeleteUploadRow`（幂等、不得删已完成会话）、
Save upsert 基本字段覆盖。

**抓到 4 处内存漂移（均为降级模式语义/数据安全相关）**：

1. 内存 `Complete` 未写 `CompletedAt`（PG 写 `completed_at`）→ 完成态审计字段在内存模式恒空；
2. 内存 `CompletedFileMetaByID` 以 upload_id 充当 file_id → 两者不同值时   举报快照等跨模块读取在内存模式恒查不到；改为按完成态 `File.FileID` 查找；
3. 内存 `ListStaleUploads` 按 map 随机序并提前 break（PG 为 `ORDER BY created_at ASC LIMIT`）   → GC 处理顺序与批次不稳定；改为按 created_at 升序 + tie-break upload_id；
4. 内存 `DeleteUploadRow` 无条件删除（PG `WHERE status <> 'completed'`）→ 保留期 GC 在内存模式   可能误删已完成上传（业务数据）；改为只删未完成会话。

配套更新：`internal/bootstrap/reports_media_source_test.go` 的种子 helper 改为按完成态
`File.FileID`（服务真实生成的 file_id，与 upload_id 不同）查找，替换原先「按 upload_id 查找」
的旧约定——原约定正是内存实现 bug 的镜像，修复后一并纠正测试。

## 双实现契约测试第十五批（2026-10-24，groupmessages 群消息核心）

`TestGroupMessagesRepositoryContract` 接入契约基建，覆盖：发送 + 历史 keyset 分页（倒序/
不重不漏/非法游标）、编辑（所有者/文本/未撤回三重守卫，edited_at 回填）、撤回（时间窗
内成功、重复撤回 10300 族、window<=0 禁用、负数窗口、`RecallAsAdmin` 绕过窗口并写
recalled_by）、**已读位点与已读计数**（重复标记不重复计数、后加入成员不覆盖更早消息、
`MarkAllRead` 返回用户成员群、非成员 up_to=10300 / 无 up_to=no-op、不存在消息计数为 0）、
成员过滤（IsMember/FilterMemberIDs）、辅助查询（MessageExists/MessageGroupID/LookupMessageMeta/
GetMessageForReply/GetGroupName、撤回后 MessageGroupID 按不存在处理而其余仍可读真值）、
举报快照读取。

**抓到 5 处内存漂移（已读语义为主，影响测试与无 DB 降级模式）**：

1. **已读计数按次数累加而非按用户位点**：同一用户重复 `MarkAsRead` 会重复计数，PG 用
   `group_message_read_state` 每用户一行去重；改为内存维护 per-user 已读位点（含单调前进
   守卫）并按 PG 口径动态计数（`last_read_sent_at >= 消息 sent_at` 且 `joined_at <= 消息
   sent_at`）；
2. 内存 `MarkAsRead` 无成员校验：up_to 路径此前任意用户均可成功，PG JOIN `group_members`
   非成员 0 行 → 10300；改为一致（无 up_to 的非成员为 no-op 成功，同 PG upsert 0 行）；
3. 内存 `MarkAllRead` 返回「有消息的全部群」（含用户不在的群），PG 返回用户所在群；已改；
4. 内存 `GetReadCount` 对不存在消息返回 10300，PG COUNT 查询返回 0；已改；
5. 历史列表对已撤回消息回填 read_count，PG 口径恒 null（不适用）；已改。

配套：新增 `SeedMember(groupID, userID, joinedAt)` 供契约测试/降级模式注入成员（生产成员权威
在 groups 模块的 group_members 表，内存实现无法跨模块读取，与 reports.TargetOwner 同类
已知限制）；更新既有内存单测（预置成员、缺失消息计数 0）。

## 幂等 Redis 后端的「过期命中」语义修复（2026-10-24，第 18 轮）

**问题（多实例部署才暴露）**：`idempotency.RedisStore` 用 `SET NX` 原子占位，记录 TTL
恰好等于业务有效期（`ExpiresAt`）。键到期后 Redis 直接删除，客户端用**同一个旧
`Idempotency-Key` 重试时 `SET NX` 成功** → 请求被当成全新请求**静默重新执行**；而单实例
`InMemoryStore` 的惰性过期会返回 `StatusExpired`，replay 层映射为 `22001`（spec
`api_spec_errors.md`：幂等缓存已过期，换新 key 重试）。结果：同一契约行为随「是否配置
Redis/单实例还是多实例」而不同，且失败方向是**重复执行写操作**（比报错更危险）。

**修复**：

1. Redis 记录 TTL = 业务有效期 + `expiredRecordGrace`（10 分钟墓碑窗口）；
2. `Get`/`Reserve` 读到 `ExpiresAt` 已过的记录时返回 `StatusExpired`（与内存实现惰性
   过期同口径，replay 层返回 22001），而不是重新占位；
3. `Save` 的 Lua 覆写同样按延长后的 TTL 写入（完成态响应在墓碑窗口内仍可识别过期）；
4. `Delete` 保持硬删除（producer panic 回滚路径要求同一 key 可立即重试，不能留墓碑）。

**测试**：新增 `internal/platform/idempotency/store_redis_test.go`（miniredis，`-race` 下运行）：
原子占位/重复占位不声明执行权、24 并发仅 1 次占位、完成态回放与 Lua「已完成不覆写」
守卫、过期墓碑识别（`Get`/`Reserve` 均返回 `StatusExpired`）与墓碑 GC 后重新占位、
`Delete` 允许立即重试、Redis 宕机 fail-open（占位直通 + Save/Delete 不 panic）、
Redis 调用 context 带超时。此前 `RedisStore`（生产多实例必经路径）无任何单测。

**审计方法沉淀**：跨实现/跨部署拓扑的语义必须用同一组行为用例验证；「TTL 到期删除」
与「惰性过期 + 墓碑」是两种不同语义，后者才能稳定返回 22001。

## 幂等回放全链路回归测试（2026-10-24，第 19 轮）

上一轮修复了 Redis 后端「过期即删键 → 旧 key 静默重新执行」的问题，但测试只覆盖到
`RedisStore` 单层；生产多实例的真实路径是「HTTP 中间件 `ReplayOrStore` + `RedisStore`」
组合，而既有 `replay_test.go` 只用 `InMemoryStore`。

新增 `internal/platform/idempotency/replay_redis_test.go`（miniredis，`-race`）：

1. 首次请求执行 producer 并缓存响应；相同 key+请求体第二次命中完成态记录：producer
   不执行、回放首次响应体、**重发 Set-Cookie**（会话下发链路）；
2. 同一 key 换请求体 → `22000`（幂等上下文不匹配），不得回放首次响应；
3. 预置一条 `ExpiresAt` 已过的记录（模拟 24h 前写入）→ 同一 key 重试返回 `22001`，
   **不得重新执行**（锁定上一轮修复的契约；TTL 粒度用直接预置过期时间，避免测试等待）；
4. producer panic → 回滚 processing 记录 → 同一 key 可立即重新执行（不返回 22002）。

该组合此前无任何测试；本轮补齐后，幂等「单实例/多实例」两条生产路径均有全链路回归。

## realtime RedisBridge 跨实例 fanout 全链路测试（2026-10-24，第 20 轮）

多实例部署的跨实例事件分发由 `realtime.RedisBridge`（Redis Pub/Sub 模式订阅）承担，
此前只有信封解码测试，**接收循环/频道路由/控制通道没有任何测试**（平台包覆盖率 ~72%）。

新增 `redis_bridge_test.go`（miniredis 起两条独立连接模拟两个实例，`-race`）：

1. 五个频道的跨实例投递：`user:`（私聊/通知）、`usersub:`（在线状态关注者）、`group:`（群事件）、
   `all:`（系统公告）、`session:`（会话撤销：下发事件并关闭连接）；每个频道断言恰好一次；
2. **本实例 origin 跳过**：bridgeA 发布的事件不得回流 hubA（避免跨实例重复投递）；
3. 撤销控制通道：`revoke:` 在远端按 `group_id` 摘除该用户群订阅后再广播收不到；`groupdrop:`
   摘除整群全部订阅者；
4. 容错：非法 JSON / 缺 origin / 未知频道只记 Warn，不打断接收循环，后续合法事件仍投递；
5. `Close` 幂等、关闭后 `Publish` 变 no-op；其它实例的事件仍可正常投递。

测试要点沉淀：Pub/Sub 订阅建立是异步的，跨实例断言需先发心跳事件直到收到再清零计数
（并等待管道排空，避免心跳尾巴污染「恰好一次」断言）；同实例发布不会经 bridge 回流，
验证「其它实例 → 本实例」必须用第三个 origin 或对端 bridge 发布。

## RedisResumeStore 边界/降级专项 + 两处防御性加固（2026-10-24，第 21 轮）

`RedisResumeStore` 是断线续传（resume）的正确性底座，正常路径已有测试；本轮补边界与
故障语义（`resume_store_edge_test.go`，miniredis + `-race`）：

1. 空库 `LastSeq=0`/`Replay=nil`；user 流与 group 流交错写入共享同一全局 seq 且严格单调；
2. 回放窗口：`fromSeq=0` 全量按 seq 升序归并、不泄漏其它用户事件；`fromSeq=第 N 条` 只回放
   后续；`fromSeq` 为最新值 → nil；
3. 脏 Stream entry（缺 `e` 字段/非法 JSON）被跳过，合法事件仍回放且 seq 正确；
4. 损坏 state JSON / 空 token / 不存在 token 均按「无状态」处理（回退全量，不 panic）；
5. Redis 宕机：Record/RecordGroup/RecordAndPublish/Replay/LastSeq/SaveState/LoadState
   全部降级（0/false/nil），不阻塞、不 panic；
6. `Close` 不关闭共享 `*redis.Client`（生命周期归 bootstrap），Close 后仍可用；
7. `origin` 为空时 `Record*AndPublish` 返回 false（不支持跨实例发布），但普通 `Record` 正常。

**测试暴露并修复的两处防御性缺陷**（`RedisResumeStore.Replay`）：

1. **负 `fromSeq`**：`XRANGE` 以 `fromSeq+1` 作为起始 Stream ID，负值会生成非法 ID，
   每个流报错并整体降级为空回放（日志噪声 + 语义丢失）。正常状态不会为负，但旧格式/
   损坏 resume 状态可能出现；钳到 0（从头回放）保持语义正确；
2. **重复 groupID**：`groupIDs` 来自客户端订阅快照（`SetSubscriptions` 的 items），重复 ID
   会让同一 Stream 被 `XRANGE` 两次 → 回放事件重复（客户端需按 `server_seq` 去重才不重复
   渲染）。回放前按首个出现顺序去重。

（注：重复订阅的同类问题也可能出现在实时广播注册表；handler 侧若未去重，同一连接可能
收到重复群事件——本轮未改动，记录为后续观察项。）

## CI 真实库集成测试串行化（2026-10-24，第 21 轮追加）

**现象**：main 推送后 CI `Go Backend (Postgres Integration)` 偶发红灯，失败断言：

`admin.TestPostgresIT_StatsGeoAggregation` → `unexpected today trend: {Date:…, User:4, Group:1, Moment:1}`。

**根因**：该 job 用 `go test ... ./internal/modules/...`，默认**按包并行**；各模块的
`TestPostgresIT` 会通过连接池**提交** fixture 行（事务隔离只覆盖用 `testdb.BeginScope`
的契约测试，IT 测试仍是直接建数据）。admin 统计测试断言「今日新增 = 基线 + 本用例插入量」，
并行包在基线之后插入的行会被计入，导致偶发不一致。此前本地门禁多次通过属运气。

**修复**：共享真实库的测试必须串行——`go test -p 1` 同时加到
`scripts/verify-backend-ci.sh`（本地门禁，与 CI 对齐）与 `.github/workflows/ci.yml`
的 DB 集成步骤，并注释说明原因。

**纪律沉淀**：共享外部状态（同一数据库/Redis 实例）的测试套件不要按包并行；
要么各包用事务回滚隔离（BeginScope），要么显式串行（`-p 1`），两者不能混用后仍假设互不影响。

## worker 后台常驻循环测试补齐（2026-10-24，第 22 轮）

`internal/platform/worker` 是全仓库后台循环的统一启动器（panic 不击穿进程），此前
`GoRestartable` 的重启/退避/停机行为无测试（包覆盖率 23.8%）。

新增 `worker_restart_test.go`：

1. panic 后按退避重启、干净返回后**不得再重启**（否则长驻 worker 停机路径变忙循环）；
2. 指数退避与封顶（max < 2×initial 时取封顶值）；
3. `runRecovered` 正确上报 crashed；
4. 退避参数提为包级变量（默认 1s/30s 不变）仅为测试注入毫秒级退避，生产路径不修改。

顺带核查全部 `GoRestartable` 使用点的契约（fn 不得持外部 WaitGroup/Add-Done）：auth sweeper、
idempotency sweeper、resourceguard、ratelimit、hotcache、notify prune、auth retention、media
retention 均为纯内聚循环；realtime typing sweeper 与 redis bridge 因自带 `wg.Done`/`done.Done`
必须用 `worker.Go`（源码注释已说明，bridge 已有 Close 幂等测试）。

包覆盖率 23.8% → **100%**。

## 统计 IT 假红灯：容器时钟偏移（2026-10-25，第 22 轮追加）

门禁 `--with-db` 曾失败于 `admin.TestPostgresIT_StatsGeoAggregation`：
`windowed A region: <nil>`（时间窗内区域 A 消失）。单测/整包/循环 20 次均无法复现。

**根因**：容器（OrbStack 重启后）与宿主机存在**毫秒级时钟偏移**，实测同一时刻：

```
host  : 2026-10-24T16:18:23.370Z
postgres: 2026-10-24 16:18:23.571   # 快 ~200ms
```

测试用 Go `time.Now()` 作为窗口上界 `windowEnd`，而种子行的 `created_at = NOW()` 取 Postgres
时钟；DB 时钟略快时刚播种的行满足 `created_at > windowEnd`，被时间窗查询排除，
区域/地点聚合为空 → 假红灯。此前同一测试还曾以「today trend 多 1」形式失败，同源。

**修复**：该测试所有时间基准改用**数据库时钟**（`SELECT now()` 取 `dbNow`）：时间窗上下界、
趋势「今天」计算、下钻窗口上界全部以 `dbNow` 为准——种子的 `NOW()` 与断言同一时钟源，
彻底消除偏移影响。

**纪律**：凡断言「DB 写入时间落在窗口内」的真库测试，窗口边界必须用数据库时钟（或留足
偏移余量），不能混用 Go `time.Now()` 与 `NOW()`。

## 热缓存生命周期加固（2026-10-25，第 23 轮）

`hotcache.Cache` 在无 Redis 时启动本地 TTL 清扫协程（`GoRestartable`），此前后台生命周期
有两处缺口：

1. **`Close` 不幂等**：实现直接 `close(c.stop)`，二次调用 panic（close of closed channel）。
   `Close` 在优雅停机路径（App.Shutdown）与测试 cleanup 都可能被调用，重复清理不应崩溃；
   改为 `sync.Once` 幂等，并补测试（重复 Close、Close 后 Get/Set/Delete 不 panic、nil 安全）。
2. **App 停机未关闭清扫协程**：`NewApp` 创建 hotCache 后未保存在 App、`Shutdown` 也未 Close，
   本地模式（未配 Redis）优雅停机后 sweeper goroutine 仍存活到进程退出。现在 App 持有
   `hotCache` 并在 `Shutdown` 中（WS 排空/worker 停止之后、DB 池关闭之前）显式 Close。

附带：测试辅助 `BuildModules()` 每次调用都 `hotcache.New(nil)`，会为每个测试构建泄漏一个
清扫协程；改为传 nil 缓存（`*Cache` 的 nil 接收者方法均为 no-op miss），生产路径不受影响。

审计方法沉淀：**后台资源（goroutine/ticker/连接）的 Close 必须幂等**，且必须在 App.Shutdown
（或对应所有者）显式释放——不能依赖进程退出；测试辅助构建函数不应启动长驻后台循环。

## 数据保留期清理测试补齐（2026-10-25，第 24 轮）

保留期清理承担 PII 与孤儿对象回收（auth_sessions 的 IP、验证码明文、未完成上传的
存储对象），此前只有「启动/幂等」层面的单测，**清理语义本身无测试**。

1. **auth `SweepRetention` 真实库测试**（`repo_postgres_retention_it_test.go`）：
   - 已撤销且超过 30 天的会话删除，近期撤销/活跃会话保留；
   - 注册验证码、密码重置请求「过期超过 7 天」删除，未超保留期保留；
   - 返回删除行数正确（3），第二轮清理幂等返回 0。
2. **media 上传会话 GC 失败路径测试**（`retention_test.go`）：
   - 超过 7 天的未完成会话：先删存储对象再删行；新鲜会话与**已完成上传**不受影响；
   - **对象删除失败时必须保留行**（下一轮重试），否则对象成为永久孤儿；恢复后下一轮成功
     删除对象与行；
   - 第二轮无可清理项时不再调用对象存储。

审计方法沉淀：清理/GC 类代码的正确性必须覆盖「失败保留（可重试）」与「幂等重跑」两个
分支，而不是只测成功删除；删除顺序应保证不会产生无法回收的孤儿（对象先删、行后删，
失败保留行）。

## 附近新瞬间通知目标查询下沉 LIMIT（2026-10-25，第 25 轮）

`moments.notifyNearbyNewMoment` 在瞬间创建（同步请求路径）时向「5km 内开启附近通知」
的用户推送。此前 `users.queryNearbyMomentNotifyTargets` **无 LIMIT**：调用方先全量读入内存、
再截断到扇出上限 200。密集区域（如市中心 5km 内数万用户）意味着每次创建瞬间都要把全部
候选行载入内存，内存与查询耗时随用户密度线性增长——与第 16 轮收藏者 fanout 同类的
「先全量读再截断」缺陷。

修复：`NearbyMomentNotifyTargets` 增加 `limit` 参数（`limit<=0` = 不限制），PG 查询以
`LIMIT $6::bigint`（NULL = 不限制）在数据库侧限量，内存实现按 limit 提前 break；moments
创建链路传 `nearbyNotifyMaxTargets`（200），调用方仍保留防御性截断。

测试：
- users 内存单测扩展：圈内开启通知用户数 + limit=1 限量；
- 新增真实库 IT：limit=0 不限制、limit=1 SQL 限量、作者排除与圈外/未开启过滤语义一致。

审计方法沉淀：请求路径上的 fanout/候选查询，上限必须下沉到 SQL；`LIMIT NULL` 可安全表达
「不限制」而无需动态拼接 SQL。

## 第 18–26 轮补充审计结论（2026-10-25）

> 本文第 17 轮曾收敛「审计总览」；本节补齐其后 9 轮的结论，并把两个系统性扫描
> （无 LIMIT SELECT、外键索引）的结论固化，避免后续重复排查。

### 一、新增生产修复（第 18–26 轮）

| # | 问题 | 影响 | 提交 |
|---|---|---|---|
| 1 | 幂等 Redis 后端记录 TTL = 业务有效期，到期即删键 | 多实例部署下旧 Idempotency-Key 重试被静默「重新执行」；单实例返回 22001。修复：过期墓碑窗口 + 惰性过期判断 | `f5ac1771` |
| 2 | 附近新瞬间通知目标查询无 LIMIT | moments 创建同步路径先把 5km 内全部候选用户载入内存再截断（扇出上限 200），密集区域内存/耗时随用户密度线性增长。修复：limit 下沉 SQL（`LIMIT NULL` 表达不限制） | `3b732677` |
| 3 | hotcache.Close 非幂等、App 停机未关闭本地清扫协程 | 重复清理 panic（close of closed channel）；无 Redis 模式优雅停机后 sweeper goroutine 存活到进程退出。修复：sync.Once + App.Shutdown 释放 + 测试辅助构建不再泄漏 sweeper | `52512951` |

### 二、测试基建/CI 修复（非产品缺陷，但影响门禁可信度）

| # | 问题 | 修复 |
|---|---|---|
| 1 | 真实库集成测试按包并行，跨包提交的 fixture 污染统计断言（`admin.TestPostgresIT_StatsGeoAggregation` 偶发红灯） | 门禁与 CI 的 DB 阶段统一 `go test -p 1`（共享真实库必须串行）——`bfcd2d50` |
| 2 | 统计 IT 混用 Go 时钟与 DB `NOW()`：容器时钟偏移（实测 PG 快 ~200ms）导致时间窗误排除刚播种行 | 测试时间基准统一用 `SELECT now()` —— `8b867e09` |

### 三、系统性扫描结论（可视为已收口，不需重复排查）

**1. 无 LIMIT 的 SELECT 全量扫描（2026-10-25）**
对 `internal/modules/*/repo_postgres*.go` 全部查询常量静态扫描：115 条 SELECT 无 `LIMIT`，
逐条分类：
- 单行主键/唯一键查询（Get/Exists/Lookup 等）——有界；
- `COUNT`/`EXISTS`/`MAX` 聚合——结果集 1 行；
- `ANY($n)` 输入有界（批次、订阅上限、成员资格、ID 列表由 service 裁剪）——有界；
- admin 列表/成员列表等规模有界项（管理员数量、群成员上限 `max_members`）——当前可接受，
  已在「已知残留」记录；
- **真·请求路径全量读**：仅 `users.queryNearbyMomentNotifyTargets` 一处，已修复（第 26 轮）。
结论：请求路径上的「先全量读再截断」模式已收口；新增 fanout/候选查询时上限必须下沉 SQL
（`LIMIT NULL` 可表达「不限制」，无需动态拼 SQL）。

**2. 外键列索引扫描（2026-10-25）**
解析迁移中全部 64 个外键，与索引/PK 前导列比对；未发现「被查询谓词使用但无前导索引」的外键：
- 无索引者（comments.user_id、comment_likes.user_id、group_bans.user_id、reports.handled_by_user_id 等）
  **均不出现在查询谓词**中；用户删除走「匿名化」而非级联删（`038_users_anonymize_delete`），
  级联扫描不存在；
- group_bans/read_state 等有 `(group_id, user_id)` 复合主键覆盖前导列；
- moments/moment_tags/moment_mentions 的 `moment_id` 由复合主键覆盖前导列。
结论：当前不需要新增索引；后续若新增「按 user_id 列评论/点赞」类查询，需同时补前导索引。

### 四、平台/后台测试资产补充（第 18–26 轮）

- `idempotency`：RedisStore 单测 + 「回放中间件 + Redis」全链路（22001/22000/Set-Cookie/panic 回滚）；
- `realtime`：RedisBridge 跨实例 fanout 全链路（五频道/ origin 跳过 / revoke·groupdrop / 容错 / Close）；
  RedisResumeStore 边界（负 seq 钳制、重复 groupID 去重、脏 Stream、损坏 state、宕机降级）；
- `worker`：GoRestartable 重启/退避封顶/干净返回停止，包覆盖率 100%；
- `hotcache`：Close 幂等、Close 后数据面可用；
- 保留期清理：auth `SweepRetention` 真实库语义、media GC「对象删除失败保留行待重试」；
- 管理后台：统计 IT 时钟基准修复；契约测试新增分页不变量用例。

### 五、已知残留更新

- （已解决）admin 统计 IT 跨包污染——改为串行；时钟漂移——改用 DB 时钟；
- （新增记录）`notify.StopPruneSweeper` 仅测试使用，不等待在飞的 `pruneOnce`（其使用独立 1min
  超时 ctx）；生产无停止调用，影响可忽略；
- 其余残留项（search/mapmodule 内存占位、reports/groupmessages 跨模块注入、platform/auth 直连池、
  admin 全量列表）维持第 17 轮总览的评估。

### 六、结论

第 18–26 轮未发现新的大类问题：主要收获为 2 处生产修复（幂等过期语义、附近通知全量读）、
1 处生命周期加固（hotcache），以及 2 处会让门禁「假红/假绿」的测试基建问题；
配合前面 17 轮，后端在「双实现语义一致、请求路径有界、清理可重试、停机可释放、门禁与 CI 同构」
五个维度已达到可交付状态。剩余待办仅为 spec 歧义确认（4 项）与可选的生产语义补测。

## 全量门禁 + 运行态核验（2026-10-25，第 27 轮）

在 main（`19f95ab2`）执行发版前同级核验：`./scripts/verify-backend-ci.sh --all`
（静态 + race + 覆盖率 + 真实库 IT + E2E + govulncheck），并做 dev compose 运行态检查。

### 一、全量门禁结果

- 静态：`go vet`/`staticcheck` 干净；`gofmt` 无差异；
- `-race` + `internal/modules` 覆盖率 ≥70%：通过；
- 真实库 IT（`-p 1` 串行）+ 双实现契约测试：通过；
- HTTP 全链路 E2E（`test/integration`）：通过；
- `govulncheck`（CI 固定 v1.8.0）：**No vulnerabilities found**——代码直接调用路径 0 个可达漏洞；
  依赖模块有 1 个漏洞，但代码不调用（不可达）。
- 结论：`ALL CHECKS PASSED`。

### 二、dev 运行态核验（2026-10-25 01:04 +08）

- 7 个容器全部 healthy（nginx/backend/frontend/postgres/redis/seaweedfs/mailpit）；
- `/healthz` 返回 `code=0, status=ok`；
- `/metrics`（仅 compose 网络，`:8081/metrics`）正常输出：`go_goroutines 21`、
  `pgxpool_total_conns 2 / idle 2 / acquiring 0`（无连接池饱和）、`empty_acquire_total 1`
  （启动至今仅 1 次等待连接，符合低频 dev 流量）、HTTP 计数器按 route 维度正常；
- backend 近 2 小时无 error 级日志；warn 仅两类预期噪声：
  1. `ws: auth failed, token invalid` 每 1–2 秒一次——来源是**本机遗留的 HeadlessChrome QA
     标签页**（旧会话/过期 token 自动重连）经 nginx 反复握手，非后端缺陷；该重连受 WS 握手
     限流约束，不会放大为资源问题；
  2. Postgres `duplicate key ... uq_auth_sessions_active_device` / `auth_accounts_pkey`——
     并发登录设备逐出与重复注册的**预期唯一约束冲突**，业务层已按设计处理（不是未捕获异常）。
- 结论：运行态健康，无新发现问题；可作为后续发版前的本地核验基线。

### 三、说明

- 本轮为核验类工作，未改代码；上述结论在下一版发版前可直接复用（若涉及 SQL/依赖变更需重跑）。

## resourceguard 清扫线程测试（2026-10-25，第 28 轮）

`resourceguard.DBChecker.StartSweeper` 负责物理清理 missing 缓存（否则扫描器用随机 ID
探测可让 map 无界增长，是内存泄漏向量），此前无测试。新增 `sweeper_test.go` 锁定：

1. 过期 missing 条目被后台清扫移除（惰性删除之外的物理清理路径）；
2. ctx 取消后循环退出——之后写入的 missing 条目不再被后台清理（仅惰性过期生效）；
3. 惰性过期在 sweeper 停止后仍正确工作。

包覆盖率 77.4% → 88.7%。未发现实现缺陷。

## 趋势统计上界改用数据库时钟（2026-10-25，第 28 轮实测修复）

门禁 `--with-db` 再次在 `admin.TestPostgresIT_StatsGeoAggregation` 失败：今日趋势
`{Date:2026-10-25 User:0 Group:0 Moment:0}`（期望 3/1/1）。定位为**生产代码**同类时钟偏移：

- `admin.statsTrends` 用 Go `time.Now().UTC()` 作为趋势查询上界（`created_at < $2`），
  而 `created_at` 由数据库 `NOW()` 写入；容器/VM 时钟偏移时（实测 Postgres 比宿主机快
  ~200ms），刚写入的行 `created_at` 可能晚于 Go 侧的 now → **今日趋势计数被静默排除**；
- 同类问题在测试侧此前以「windowed region 为空」出现（测试已改用 DB 时钟），但生产代码
  未修，属于真实线上行为缺陷（管理员统计页数字短暂少计，取决于时钟偏移量）。

修复：趋势 SQL 的上界改用数据库时钟 `created_at <= now()`（长事务内 `now()` = 事务开始时间，
与 `DEFAULT NOW()` 写入同源、边界包含），去掉 Go 传入的 end 参数；`TrendStart` 仍由服务层
计算（仅作为下界，偏差只可能发生在跨本地午夜的毫秒级窗口，影响可忽略，后续如引入 DB 时钟
注入再彻底消除）。

验证：该 IT 连续 10 次通过（修复前在时钟偏移下必现）；门禁 `--with-db` 全绿。

## spec 契约歧义澄清（2026-10-25，第 29 轮）

依据用户此前授权（「可以改，只要能让这个项目更加完美」），对审计中记录的 4 项 spec 歧义做**最小澄清**
（全部与现有双实现口径一致，不改变行为，只补全契约文字）：

| # | 歧义 | 澄清（已写入 spec） |
|---|---|---|
| 1 | share「我的分享」列表过期呈现未定义 | `api_spec_share.md` fetch_created 增加：`expires_at` 已过与主动撤销同口径呈现（`is_revoked=true`、`preview=null`，`expires_at` 保留原值）；`is_revoked=true` 统一表示「当前不可用（撤销或过期）」，客户端无需区分 |
| 2 | `read_count` 对不存在消息的内部行为未定义 | `api_spec_chat.md` 字段注补充「消息不存在时不会出现在列表/事件中」——该行为是内部 repo 语义（COUNT 返回 0），不对外暴露接口 |
| 3 | `bootstrap_version` 未定义实现算法 | `api_spec_system.md` 增加实现口径：只要求「相关维度变化时值必变、不变时可不变」，客户端只做字符串相等比较；允许单调指纹（构建版本 + AppConfig 更新时间 + 公告域 max(updated_at)），不要求严格递增整数 |
| 4 | reactions 游标不透明性 | 无需改 spec：`api_spec_base.md` 已规定 cursor 不透明、客户端仅透传；实现已统一 AES-GCM 游标 |

本轮无代码改动；spec 为文档改动，按项目例外跳过本地 DB/E2E（CI 全量仍会执行）。

## Go 时钟 vs 数据库时钟边界审计（2026-10-25，第 30 轮）

第 28 轮修复了趋势统计「Go 时钟作 SQL 上界」的真实缺陷后，对全仓做同类扫描并分类收口：

**方法**：列出所有生产代码 `time.Now()` 使用点与 SQL 时间谓词，按「时间写入源 × 比较源」分类。

| 类别 | 代表位置 | 一致性 | 结论 |
|---|---|---|---|
| 时间由 Go 写入、由 Go 比较 | auth 会话/验证码过期、groups 邀请/公告过期、admin 封禁到期（`state.Expired(time.Now())`） | 自洽（同一时钟源） | 无风险 |
| 时间由 DB 写入、由 SQL 比较 | read_state、群成员禁言（`muted_until > NOW()`）、趋势统计（第 28 轮改为 SQL `now()`）、DB 侧默认值比较 | 自洽 | 无风险 |
| 时间由 DB 写入、由 **Go** 比较（残余） | ① 保留期清理（auth 7/30 天、media 7 天）：Go `now` 作 cutoff；② 时间窗类列表（moments 24h/7d、admin 报表由客户端传时间范围）；③ 趋势统计**下界** `TrendStart`（服务层 Go 计算本地日锚点） | 毫秒级偏差 | ① 保留期以天计，±200ms 无影响；② 窗口以小时/天计，边界偏差无业务影响；③ 仅在「Go 与 DB 分处本地午夜两侧」的 ~200ms 窗口内可使趋势首日锚点错一天（且只影响该瞬间新写入的行），量级可忽略——已在文档记录，如未来引入 DB 时钟注入再彻底消除 |
| 已修 | 趋势统计**上界**（`created_at < time.Now()`） | 曾真实少计「今日」 | 第 28 轮改为 `created_at <= now()`（`3f9cccb9`） |

**观测方法**（可复现）：容器时钟与宿主机可能短暂偏移（2026-10-25 实测 Postgres 比宿主机快约 200ms，
OrbStack 重启后更常见）：
`date -u +%Y-%m-%dT%H:%M:%S.%NZ` vs
`docker compose exec -T postgres psql -U postgres -tAc "select now() at time zone 'UTC'"`。

**纪律**：
1. 真库测试断言「DB 写入时间落在窗口内」时必须用数据库时钟（第 22 轮已修 `TestPostgresIT_StatsGeoAggregation`）；
2. 生产代码中用「另一时钟」被比较的列必须评估边界；毫秒级且窗口≥小时的偏差可接受，日/秒级边界的必须同源；
3. 跨容器比较时间时优先 SQL 内 `now()`，避免把 Go 时钟经参数传入作为 DB 行的时间边界。

## 依赖与迁移卫生核验（2026-10-25，第 31 轮）

### 一、Go 模块卫生

- `go mod verify`：**all modules verified**（本地模块缓存内容与 go.sum 一致，无篡改/缺失）；
- `go mod tidy -diff`：**空 diff**（go.mod/go.sum 已是最小闭包，无多余 require、无缺失 checksum）；
- 11 个直接依赖全部被实际 import（`go mod why -m` 逐个核对：chi、jwt、golang/geo、gorilla/websocket、
  pgx、prometheus client、go-redis、zap、x/crypto/bcrypt、x/sync/errgroup、miniredis 测试用）；
- 无可清理的未使用依赖。

**环境事实（非缺陷）**：本机 Go `1.27.3`，CI 固定 `1.26.10`（`go.mod` `go 1.26.0`）。本地工具链比 CI 新，
可能掩盖只在新版标准库/语言特性下可编译的写法；因此本轮**没有**把 `go mod tidy -diff` 加入 CI 门禁
（跨工具链版本可能产生 go.mod 文本漂移导致假红灯），保留为本地/发版前人工检查项。若需要 CI 化，
应先统一本地与 CI 的 Go patch 版本（以 `workwork/versions.env` 为准）。

### 二、迁移与库结构卫生（库 `hotspot_it`）

- 迁移文件数 = `schema_migrations` 记录数 = **49**（无漏登记/无重复执行）；
- `go run ./cmd/migrate` 连续执行两次均成功（幂等）；第二次无任何变更；
- `pg_index` 中 **invalid 索引数 = 0**（无 `CREATE INDEX CONCURRENTLY` 残留的 INVALID 索引）；
- public schema 表数 45，与迁移定义的业务表一致（含内部表 `schema_migrations`/`data_migrations`）。

结论：依赖与迁移两个「长期运行才暴露」的卫生面均干净，无需改动；后续发版前可直接复跑上述命令。

## 趋势统计窗口完全下沉 SQL（2026-10-25，第 32 轮）

第 28 轮已把趋势上界改为数据库时钟，但**下界锚点**（`TrendStart`）仍由服务层用 Go 时钟计算，
在「Go 与 DB 分处本地午夜两侧」的毫秒窗口仍可能错位（已记录为可忽略残余）。本轮彻底消除：

- `queryAdminStatsTrends` 改为在 SQL 内生成完整日期序列：
  `days AS (SELECT generate_series((now() AT TIME ZONE 'Asia/Shanghai')::date - ($1-1), now()::date, '1 day'))`，
  `agg` 侧仅按 `created_at <= now()` 过滤并按 UTC+8 自然日分组，LEFT JOIN 后空日补零；
- 参数只剩窗口天数（`$1`，服务层已保证 7–30）；上界、下界、时区锚点全部来自数据库时钟；
- `statsTrends` 不再做 Go 侧补零（`fillTrendDays` 保留给内存实现共用），逐行映射返回；
- 效果：PG 生产实现不再混用 Go 时钟，admin 趋势统计在容器时钟偏移下仍严谨；内存实现维持原口径
  （单测/无 DB 模式，正常时钟下与 PG 一致）。

测试：更新 stub 用例（SQL 直接产出 7 行、参数为天数）；`TestPostgresIT_StatsGeoAggregation`
连续 5 次通过（含今日趋势、空日补零、时间窗断言）。

## 发版前同级验收（2026-10-25 02:05，第 33 轮）

在最近一轮代码改动（`6d19a7bb`，趋势统计窗口下沉 SQL）之后，重新执行**全量验收**，
确认 main 处于可交付状态：

### 一、`verify-backend-ci.sh --all`（全绿）

| 阶段 | 结果 |
|---|---|
| 静态（vet/staticcheck/gofmt） | 通过 |
| `-race` + `internal/modules` 每包覆盖率 ≥70% | 通过 |
| 真实库 IT（`-p 1` 串行）+ 双实现契约 | 通过 |
| HTTP 全链路 E2E（`test/integration`） | 通过（6.1s） |
| govulncheck（CI 固定 v1.8.0） | **No vulnerabilities found**（可达漏洞 0） |
| 总结果 | **ALL CHECKS PASSED** |

### 二、dev 运行态

- 7 容器 healthy；`/healthz` → `code=0, status=ok`；
- `/metrics`：`go_goroutines 22`、`pgxpool_total_conns 2 / idle 2 / acquiring 0`（无饱和）；
- backend 日志 error 级共 2 条，**时间戳均为 2026-10-23T14:17Z**（旧容器重启前保留的历史日志，
  内容为当时已修复的 `auth.CreateGuest` profile insert 参数问题）；当前版本无新的 error 级日志。

### 三、结论

最近 5 轮（28–32）的代码改动（趋势统计时钟修复、resourceguard 清扫测试、附近通知 LIMIT、
spec 澄清、依赖/迁移核验）在合并后均通过 ---all 全量验收，main 与 origin 同步；
可作为下一次发版 tag 前的本地验证基线。

## 分享爬虫 OG 落地页纳入限流（2026-10-25，第 34 轮）

审计发现 **GET /share/{code}（爬虫 OG 落地页）实际没有限流**：

- 该路由挂了 `ratelimit.Middleware`，但限流中间件依赖 `routemeta.ClassifyPath` 选择规则桶，
  而 `/share/{code}` 被登记为例外、**未参与分类** → `classifyRules` 对未识别路径返回 nil，
  中间件空转；
- 结果是匿名的 `GET /share/{code}` 每次请求直接查库解析 share code（不存在也查），
  可被伪造爬虫 UA 的脚本用随机 code 无限枚举，放大数据库查询（与 open_the_door 同类探测面，
  后者已有 probe 桶防枚举）。

修复（最小改动，复用既有机制）：

1. `platform/routemeta/classify.go` 为 `/share/` 前缀增加只读分类；
2. `platform/ratelimit/middleware.go` 对 `/share/` 复用 **probe 桶**（`SearchPerMinute`，
   默认 60/min/IP，与 open_the_door 同一致的「存在性探测」语义），并同时受全局 IP 小时桶约束；
3. 测试：`routemeta.TestClassifyPath` 增加 `/share/abc123456` 分类用例；
   `ratelimit.TestShareOGCrawlerRateLimitByIP` 锁定第 2 次同 IP 请求返回 10800（限流）。

安全性说明：OG 页本身输出已全量 `html.EscapeString`（本轮同时复核：title/description/cover/链接
均转义，无 XSS）；JWT（HS256 强校验 + 必须含 exp）、Cookie（HttpOnly/Secure/SameSite）、
SQL 动态拼接（仅占位符序号/白名单表达式，用户值全部参数化且 LIKE 通配符已转义）复核无问题。

## 群组模块文本长度校验补全（2026-10-25，第 35 轮）

审计「契约字段约束 ↔ 实现校验」时发现：**群组模块除非空外没有任何长度上限校验**。
DB 列均为 TEXT，不拦超长值；1MiB 请求体上限内，单个字段可写入接近 1MiB 的群名/描述/
公告，列表与详情响应随之放大（存储 + 带宽放大），并直接违反契约（`api_spec_base.md`
「通用字段约束 / 群组字段约束 / 公告字段约束」）。

修复：新增 `groups/validation.go`，在 Service 入口（与 moments/comments 的长度校验同层）
按 **rune 计数** 校验并返回 10000：

| 字段 | 约束（spec） | 落点 |
|---|---|---|
| `name` | 1-50 字符 | create/update（非空仍由 Handler 校验，Service 兜底上限） |
| `description` | 0-500 | create/update |
| `tags` | 最多 10 个、每个 1-30 | create/update |
| `location.location_name` | 0-100 | create/update |
| `password`（群密码） | 8-32 | create/update（空=不设置） |
| `max_members` | 2-2000（0=默认） | create/update |
| 公告 `title` / `content` | 1-100 / 1-5000 | announcements create/update |
| 邀请/申请附言 `message` | 0-200 | invites/create、join_group |

测试：新增 `validation_test.go` 覆盖各字段边界（含中文按 rune 计数）、公告/附言边界，
并用 nil repo 的 Service 验证「校验先于任何 DB 访问」（返回 10000 而非 panic/10600）。

**同轮发现的相邻缺口（记录，后续轮次处理）**：
1. moments `tags` 只校验数量（≤10），未校验每个标签 1-30 字符；moments/users/groups 的搜索
   `keyword` 未校验 1-100 上限；
2. `AppConfig.limits.max_group_members`（平台可调的群成员上限）在 groups 创建/更新时未强制，
   须确认「超平台上限应拒绝还是静默裁剪」后修（涉及客户端 `max_members` 可选项口径）。

## 契约长度校验缺口收尾：标签/搜索关键字/群成员上限（2026-10-25，第 36 轮）

接第 35 轮（群组文本长度）继续处理同轮记录的相邻缺口：

### 1. moments 标签逐条长度
此前只校验 tags 数量（≤10），单个标签可无限长（违反 base「每个标签 1-30 字符」，且
moment_tags 逐条 INSERT 会放大写放大）。新增 `tagsMaxRunes=30` 与 `ErrInvalidTag`：
`ensureTagsLimit` 同时校验数量与逐条 `1-30`（rune），Create/Update 映射 10000。

### 2. 搜索关键字/标签长度上限（keyword 1-100、tag 1-30）
- `moments/search`：keyword/tag rune 上限；
- `users/search`：keyword/tag rune 上限；
- `groups/search`：keyword/tag rune 上限；
- `groups/members/search`：keyword rune 上限；
超限统一 10000（此前超长 keyword 直接进入 ILIKE 查询与服务端内存排序）。

### 3. 平台群成员上限 `AppConfig.limits.max_group_members` 落地
此前该配置只在 ClientConfig 下发、**服务端从未强制**：用户可创建 `max_members` 高达 2000
的群，突破平台默认 500 的运行参数。现在 create/update 显式 `max_members` 超过平台上限
（`>0` 时）返回 10000；未携带（0=默认）保持服务端默认值不变。语义取「拒绝」而非静默裁剪：
静默改小会让客户端展示与实际不一致，显式拒绝更符合「参数必显式」纪律。

测试：moments 标签边界/搜索 keyword+tag 超长（handler）、users/groups 搜索超长、
群成员上限边界（=cap 通过、>cap 拒绝）、Service 接线。

## 迁移 expand-contract 守卫（2026-10-25，第 37 轮）

审计迁移纪律时确认：`scripts/check-migrations.sh` 此前只校验**序号唯一**，
`AGENTS.md`/迁移 skill 里的「禁止 CREATE INDEX CONCURRENTLY」「迁移必须兼容上一发布版本
二进制」只是文字纪律、无机器校验——一次疏忽（DROP/RENAME 列、SET NOT NULL、TYPE 变更、
CONCURRENTLY）只会在 CI 之外或线上启动时才暴露。

本轮把纪律升级为 CI 硬校验（`scripts/check-migrations.sh` 在 backend job 与本地门禁
`verify-backend-ci.sh` 均执行）：

1. **禁止 CREATE INDEX CONCURRENTLY**（全量文件）：当前迁移架构整文件单消息执行会落
   隐式事务并以 25001 拒绝；提示拆分单语句迁移文件的做法；
2. **新增迁移（序号 >= 058）禁止破坏性 DDL**：`DROP COLUMN/TABLE/CONSTRAINT`、
   `RENAME COLUMN`、`SET NOT NULL`、`ALTER COLUMN ... TYPE`。理由：发布回滚 = 旧镜像 +
   保留新 schema，上述 DDL 会让旧二进制读不到列/写不进去；正确做法是 expand-contract
   两阶段（先加新列双写兼容，跨版本再清理）。
3. 匹配前剥离 `--` 行注释，避免把说明文字当 DDL；历史文件（042 列改名、050 主键重建等
   已发布迁移）作为白名单不回改。

验证（三种场景实跑）：当前迁移树 → OK；临时 `058_breaking.sql`（DROP COLUMN）→ 失败并
提示 expand-contract；临时 `058_conc.sql`（CONCURRENTLY）→ 失败；删除后再次 OK。
迁移 skill 同步补充「CI 强制」说明。

**回滚兼容性说明（057）**：最后一个发版 tag `v1.3.2` 已包含 001–056 及对应代码，回滚到
`v1.3.2` 不受影响；057（设备会话唯一索引）为 tag 之后新增，旧二进制顺序登录仍先撤旧再插新
（不会撞索引），仅**并发登录竞态**会收到 23505（迁移注释已登记为部署切换期可接受）。

## 静态分析层：staticcheck SA 接入门禁/CI（2026-10-25，第 38 轮）

此前门禁与 CI 的静态检查只有 `gofmt` + `go vet`（`govulncheck` 只覆盖漏洞符号），
没有 staticcheck。本轮用固定版本 `2026.2.2` 跑 `-checks 'SA*'`（correctness 规则）：

- **生产代码（非 _test）零发现**；
- 测试代码共 20 条，逐条修复后清零，其中两条暴露了「测试实际没测到东西」：
  1. `auth/api_errors_coverage_test.go` 声称验证「%w 包装的 APIError 能解包」，实际只是把
     `errors.New` 的返回值覆盖成一个裸 `*APIError`（SA4006/SA4017）——改成真正的
     `fmt.Errorf("...: %w", &httpx.APIError{...})`，锁住 `apiErrorFrom` 的 `errors.As` 路径；
  2. `mapmodule/repo_memory_test.go` 的 Heatmap 位置断言是**空分支**（`if lat==0 && lng==0`
     里只有一个空 `if`，SA9003）——改为非零坐标断言，真正覆盖 Location 字段；
  另修复：多处测试把 `nil` 当 `context.Context` 传（SA1012，改 `context.Background()`）、
  `repo_postgres_read_test` 未使用的 `argIndex`（SA4006）、integration helper 中永远不可达的
  两个 `*ApiResponse[T]` case（SA4020，所有泛型实例均满足先前的 `codeGetter`）。

**接入方式**：`scripts/verify-backend-ci.sh` 在 `go vet` 后新增 staticcheck 步骤（缺失时
`go install ...@2026.2.2`，与 govulncheck 同模式）；`.github/workflows/ci.yml` backend job 同步新增
步骤。版本固定避免 @latest 漂移；规则暂限 `SA*`（correctness），风格类/未使用测试辅助
（U1000 等）不阻塞。

验证：`staticcheck -checks 'SA*' ./...` 0 输出；门禁 `ALL CHECKS PASSED`；CI 通过。

## 不可信输入 fuzz 覆盖（2026-10-25，第 39 轮）

审计「客户端完全可控输入」的解析面时，为 5 个高风险入口新增 Go fuzz 目标（种子随普通 `go test`
在 CI 执行，`-fuzz` 可按需长跑）：

| 目标 | 契约 |
|---|---|
| `cursor.FuzzDecodeNeverPanics` | 游标（AES-GCM + base64 + JSON 解包）任意字节输入不 panic，错误只能是 `ErrInvalidCursor` |
| `cursor.FuzzDecodeStringNeverPanics` | 同上（DecodeString 共用 decrypt 通道） |
| `httpx.FuzzDecodeStrictJSON` | 请求体任意字节：不 panic；失败必须是 10000（含「请求体过大」分支不误判为格式错误） |
| `httpx.FuzzRequireEmptyObject` | 空对象端点：容忍未知字段，拒绝非对象/坏 JSON，任意输入不 panic |
| `idempotency.FuzzRequireKey` | `Idempotency-Key` 任意 header：不 panic；成功时 context 键与 header 一致，失败码只能是 10000/22002 |

本地实跑（2026-10-25）：`-fuzztime=10~15s` 五个目标全部 PASS，累计约 **570 万次执行**、0 panic；
发现的 interesting 输入留在 Go 本地 fuzz cache，不进仓库。

意义：这些入口此前只有手写单测（固定样本），fuzz 覆盖了畸形长度/字节/JSON 边界，防止未来改动
引入 panic（进程级 500 或 DoS）。CI 目前只跑种子（成本可忽略），需要深挖时本地执行：
`go test -run='^$' -fuzz='FuzzDecodeNeverPanics$' -fuzztime=30s ./internal/platform/cursor/`。

## 后台 sweeper 停止等待语义（2026-10-25，第 41 轮）

审计遗留的可选项：`notify.StopPruneSweeper` 与 `ratelimit.StopSweeper` 此前只 cancel ctx、
**不等待循环退出**，Stop→Start 序列可能短暂并行两个清扫循环（清理幂等所以无害，但语义不严谨）；
`resourceguard.DBChecker.StartSweeper` 则完全没有 Stop，重复 Start 会启动两个循环。

本轮统一为「Stop 等待循环退出（有界）」：
- 三个 sweeper 的 goroutine 包装器仅在**干净退出**时 `close(done)`；panic 交回
  `worker.GoRestartable` 重启（重启后 ctx 已取消会立即干净退出并关闭 done）；
- `Stop` 取消 ctx 后最多等 1–2s（notify 的在飞 `pruneOnce` 用独立 1min ctx，不在取消范围）；
- `resourceguard` 新增 `StopSweeper()` 并让 `StartSweeper` 幂等（已有循环时 no-op）；
- 测试更新：notify/resourceguard 生命周期用例改用 Stop 等待语义并断言包级状态复位 + 可重启；
  ratelimit 新增 `TestSweeperStopWaitsAndRestartable`（幂等 Start/Stop + 复位 + 重启）。

### `AppConfig.limits.max_group_members` 调小后的存量群策略（记录，非缺陷）
当前实现：平台上限只在**创建/更新群**时校验（显式 `max_members` 超上限返回 10000）；
管理员把平台上限调小**不会回溯修改存量群**的 `max_members`，也不自动裁剪。理由：存量群的容量
契约已对外承诺，静默裁剪会破坏成员上限语义；如需要「调小后强制收敛」，应作为显式管理操作
（逐群 update 或迁移）另行评审。

## 备份/恢复链路实测（2026-10-25，第 42 轮）

审计运维数据安全面：不依赖文档结论，直接在本机 dev 栈把 `scripts/backup-postgres.sh` 与
`scripts/backup-seaweedfs.sh` 各跑通一遍，并做一次**真实恢复**验证。

### Postgres（`scripts/backup-postgres.sh`）
- `BACKUP_DIR=/tmp/... sh scripts/backup-postgres.sh` → 产出 `hotspot_YYYYmmdd_HHMMSS.dump`（256K，
  自定义格式 `-Fc`），脚本内 `pg_restore -l` 完整性校验通过；
- 恢复实测：在容器内新建 `hotspot_restore_check` 库 → `pg_restore --no-owner` 导入 → **exit 0 无告警**；
- 数据一致性（源库 vs 恢复库）：`users 226/226`、`groups 18/18`、`moments 33/33`、
  `comments 22/22`、`notifications 188/188`、`auth_sessions 635/635`、`schema_migrations 49/49`；
  表总数 `45/45`；
- 验证后已 DROP 临时库、删除临时 dump，未污染 dev 数据。

### SeaweedFS 媒体卷（`scripts/backup-seaweedfs.sh`）
- 卷名 `hotspot_seaweedfs_data` 与脚本默认一致；运行产出 `seaweedfs_*.tar.gz`（416K，123 个条目），
  归档内为 `data/...`（与文档 `--strip-components=1` 的恢复示例一致）；
- 验证后删除临时备份。

### 结论
两条备份脚本、恢复流程与 `docs/backup-restore.md` 的还原示例在 dev 环境均可用，未发现脚本缺陷；
建议发版前在准生产再跑一次（数据量/权限更接近线上）。

## 安全扫描（gosec）接入与逐条 triage（2026-10-25，第 43 轮）

按「继续审计」建议接入更强的安全扫描：gosec v2.30.0 全量扫描后端（365 文件 / 60k 行），
共 **58 条**发现，逐规则 triage 如下：

| 规则 | 数量 | 处置 |
|---|---|---|
| G104（未处理错误） | 18 | 全部为 `defer rows.Close()` / 已显式忽略的清理错误；复核后不阻塞（清理失败由上层状态兜底），medium 级别不进 high/high 门禁 |
| G115（整数溢出转换） | 13 | s2 cell id 的 uint64/int64 转换属 S2 不变式；media EXIF 宽高来自 uint32 元数据；bcrypt cost 来自常量/已校验配置。逐个复核无用户可控溢出路径 |
| G704（SSRF） | 8 | `media/provider_object_storage.go` 请求 URL 由受信 `S3_ENDPOINT` 配置 + 服务端对象键签名生成，host 不受请求方控制 → 加 `#nosec G704 -- 理由` |
| G101（硬编码凭据） | 7 | 全部是 SQL 常量含 `login_id/password` 等字段名，非凭据；low confidence 不门禁 |
| G124（Cookie 属性） | 5 | auth cookie 的 `Secure` 来自 `cookieSecure()` 动态值（生产 true）；device_id cookie 有意非 HttpOnly（已文档化） |
| G501/G401（MD5） | 2+1 | MD5 仅用于 S3 ETag/内容校验兼容，非安全哈希 |
| G707（SMTP 注入） | 1 | 注册入口加固：邮箱校验拒绝空白/控制字符、要求恰好一个 `@`、域名含 `.` 且非 `.` 开头/结尾/连续；`net/smtp` 的 `validateLine` 另二次拒绝 CR/LF → `#nosec G707 -- 理由` |
| G710（开放重定向） | 1 | 头像 302 加固：目标必须是绝对 http(s) URL（host 非空），否则 404 → `#nosec` |
| G304（文件包含） | 1 | `migrate.ApplyDir` 仅运维 CLI 以受信目录调用（服务端用 embed FS） |
| G602（切片越界） | 1 | 索引在 `for i := range items` 内，边界安全 |
| G108（pprof 暴露） | 1 | pprof 仅绑定 127.0.0.1:6060 独立 server（`PPROF_ADDR` 可禁用）→ `#nosec G108` |

**门禁化**：`scripts/verify-backend-ci.sh` 与 CI backend job 新增 gosec 步骤（固定 v2.30.0），
只门禁 **high severity + high confidence**（当前 0 条），并要求每个 `#nosec` 必须带规则号与
`-- 理由`（`-nosec-require-justification -nosec-require-rules`），防止无理由压制。

**本轮顺带修复的真实加固点**：
1. 邮箱登录标识校验过松（如 `a@b.c` 后接 CRLF 内容也能通过）→ auth/users 两处统一收紧，
   注册/升级入口在进入 SMTP 前拒绝非法地址；
2. 头像 302 未校验 URL 形态 → 只允许绝对 http(s) 目标，异常统一 404；
3. pprof/SSRF/MD5 等已知安全例外以带理由的 `#nosec` 固化，后续新增同类代码仍会被扫出。

验证：gosec high/high 0 发现；受影响包 `go test -short` 全绿；门禁/CI 新增步骤在本轮提交后验证。

## 只读接口压测基线（2026-10-25，第 44 轮）

按「继续审计」建议做压测：新增可复用的 `backend-go/cmd/loadgen`（仅 stdlib，固定并发、
输出 RPS/状态码/业务码/延迟分位数），对 **backend-test**（`APP_ENV=test` 跳过限流、独立
`hotspot_test` 库，端口 8082）做只读路径基线。

### 方法
```
docker compose -f docker-compose.yml -f docker-compose.dev.yml --profile test up -d backend-test
cd backend-go && go run ./cmd/loadgen -url http://localhost:8082/<path> -c 50 -d 10~20s [--method POST --body ... --H 'Authorization: Bearer <guest token>']
```
环境：Apple Silicon Mac（压测进程与容器同机共享 CPU），并发 50 keep-alive，单后端实例，
`hotspot_test` 为 E2E 遗留的小数据量；**数字用于相对比较与瓶颈定位，不是生产容量承诺**。

### 结果（2026-10-25 10:33 +08）

| 端点 | 类型 | RPS | p50 / p95 / p99 | 错误 |
|---|---|---|---|---|
| `GET /healthz` | 纯 HTTP（无 DB） | **19,133** | 0.91 / 3.06 / 7.51 ms | 0 |
| `POST /public/v1/system/health` | DB + Redis 健康检查 | **11,422** | 1.88 / 6.69 / 33.96 ms | 0 |
| `POST /public/v1/system/announcements/fetch` | 公共 DB 读 | **7,287** | 3.82 / 15.02 / 48.46 ms | 0 |
| `POST /api/v1/users/account` | 认证 + DB 读（会话校验 + 资料） | **1,351** | 17.16 / 84.16 / 161.11 ms | 0 |
| `POST /api/v1/system/bootstrap` | 认证聚合（profile + unread） | **1,091** | 25.30 / 91.33 / 169.28 ms | 0 |

压测前后指标：goroutines 21→21；`pgxpool` 由 total 3 / idle 3 升至 **total 25 / idle 25**
（默认 `MaxConns=25` 达到上限，`acquiring=0`）。所有响应 HTTP 200 且业务 code=0，无错误。

### 结论
1. HTTP 栈与公共读路径余量充足；认证读取约 **1.1–1.35k RPS**，瓶颈在**数据库连接池/查询**
   （每请求至少 1 次会话校验 + 1 次资料查询），而非 Go 路由/JSON；
2. `p99 ~160ms` 与 `MaxConns=25` 饱和相关：50 并发下请求排队等连接。生产默认池 25 对当前
   社区规模足够；若未来认证 QPS 目标提高，优先调 `DB_MAX_CONNS`/DB 规格，其次评估**短 TTL
   会话校验缓存**（需权衡撤销即时性：会话/封禁当前为每请求查库，缓存会延迟强制下线）；
3. 各端点最大延迟 ~600ms 的离群值出现在同机压测（压测进程抢 CPU + GC），非稳定态特征，
   需在准生产（独立压测机、真实数据量）复测后再下容量结论。

复跑方式见上方命令；工具已入库，后续性能回归可直接复用。
