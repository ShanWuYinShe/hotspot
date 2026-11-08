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
- **注意**：前端 `bun run browser:qa` 走**真实后端**（seed 数据 + 真实账号登录），会产生后端流量，但它是 UI 巡检（单线程、页面加载型），不能代替压测或 SQL 性能验证。
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

**部署拓扑已确认（2026-10-23，站长；2026-10-30 调整）**：全部流量必经
Cloudflare 回源，源站以 mTLS origin-pull 作为唯一入口闸——2026-10-30 起
chaoset.com / www.chaoset.com 使用 **per-hostname AOP（自备证书，源站只信任
自有根 CA）**，其它 CF zone 的共享证书不再被接受；直连无客户端证书一律 400，
80 端口对公网只做 301。原「CF 网段
allow/deny」白名单已移除：它与 real_ip 还原互斥（CF-Connecting-IP 改写后的
$remote_addr 是真实访客 IP，命中不了 CF 网段，2026-10-30 全站 403 事故根因），
且写死网段易过时。real_ip 还原改为只在 443 server 内信任所有对端（能到达即
已通过 mTLS），CF-Connecting-IP 仍由 CF 强制设置、可信——后端信任 CF 头的
口径不变（**不要清洗该头**）。

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

## 剩余改进方向盘点 + 发版镜像演练（2026-10-25，第 45 轮）

按用户要求盘点「还有哪些方向需要改进」，以当前代码/配置实际状态为依据（非泛泛建议）。

### P0（建议尽快做）
1. **E2E 缺 WebSocket 全链路**：`backend-go/test/integration` 有 92 个 HTTP 用例，但**没有任何 WS 用例**；
   WS 目前只靠单元测试（handler/hub/redis_bridge）。建议补 `connect → Auth → Subscribe → 事件推送 →
   心跳/关闭码` 用例（gorilla/websocket 已是直接依赖，测试成本低、回归价值高）。
2. **写路径未压测**：本轮 `cmd/loadgen` 只覆盖只读接口；创建/发送/入群等写路径涉及事务、唯一约束、
   fanout 与幂等键，尚无容量数据。建议在 backend-test 库上做受控写压测（含 Idempotency-Key 与清理）。
3. **无告警规则**：`docker/prometheus.yml` 只有 scrape 配置，没有 alert rules——指标齐全但
   pgxpool 打满、5xx 突增、WS 掉线等无阈值告警。建议补 Prometheus alert rules + 通知渠道。

### P1（中期，需要权衡）
4. **会话校验/封禁每请求查库**：压测认证端点 ~1.35k RPS、p99 161ms、pgxpool 达 MaxConns 25；
   若提高目标需短 TTL 会话缓存。权衡：缓存会延迟「登出/封禁即时生效」，需配合主动失效（撤销时删缓存）。
5. **数据保留策略未定义**：`admin_audit_logs`、`reports` 无清理逻辑，spec 也未规定保留期；
   需产品决定（审计合规保留 vs 表体积）。
6. **无可观测性 tracing**：现有日志 + 指标（request_id 串联），跨 HTTP→DB→Redis 无 trace；
   规模上来后排障成本上升，可评估 OpenTelemetry。
7. **本地 Go 1.27.3 vs CI 1.26.10**：本地工具链更新可能掩盖 CI 编译差异；建议统一版本
   （go.mod toolchain 或 mise/asdf 固定），避免「本地绿 CI 红」的变种。

### P2（按需/长期）
8. **大数据量查询验证**：无 synthetic dataset 与 EXPLAIN 基线；搜索/Feed/统计在真实数据量下的索引
   效果未验证。建议造数脚本 + 关键 SQL EXPLAIN 检查。
9. **发版/回滚演练**：本轮已做镜像构建与 OCI 指纹核对（见下）；完整 deploy→rollback 演练待准生产。
10. **审计文档结构**：`docs/backend-audit.md` 已 1474 行，建议按主题拆分到 `docs/backend/` 便于检索。
11. **备份异地/恢复**：脚本与本地恢复已验证（第 42 轮），`OFFSITE_CMD` 与 cron 尚需准生产验证。

### 发版前本地镜像演练（Release 门禁步骤 3/4）
本轮在 `756a70f2` 上执行（不 tag、不部署）：
```
GIT_VERSION=v1.4.0-rc1 GIT_COMMIT=$(git rev-parse HEAD) GIT_BUILD_TIME=$(date -u +...) \
  IMAGE_TAG=v1.4.0-rc1 docker compose --profile prod build backend frontend-build
```
- 构建成功：`hotspot-backend:v1.4.0-rc1`（61MB）、`hotspot-frontend:v1.4.0-rc1`（17MB）；
- OCI 指纹核对：两个镜像 `org.opencontainers.image.version=v1.4.0-rc1`、`revision=756a70f2...`
  与构建提交一致（Release workflow 同款断言）；演练镜像已删除，未推送 registry。

### 建议执行顺序
P0-1 WS E2E → P0-3 告警规则 → P0-2 写路径压测 → P1-4 会话缓存评估（需产品确认撤销语义）→
其余按需。

## WS E2E 全链路用例（2026-10-25，第 46 轮）

补上 P0-1 缺口：`backend-go/test/integration/websocket_test.go` 新增真实 compose 环境的
WebSocket 用例（此前 E2E 只有 HTTP，WS 仅 realtime 包单测）：

1. `TestWebSocket_ConnectAuthHeartbeatSubscriptions`：guest token 握手 → `Auth` →
   收到 `ConnectionEstablished`（校验 connection_id/user_id/server_time(RFC3339)/
   server_seq(int64-as-string)/resume_token/ping_interval=30/ping_timeout=10/max_subscriptions）；
   `Ping→Pong`；`SetSubscriptions`（订阅自己）→ `SubscribeAck` accepted 且回显 client_request_id；
   `GetSubscriptions` → `SubscriptionsSnapshot` 含订阅 ID；发送 close 帧后服务端关闭连接。
2. `TestWebSocket_AuthFailureClosesWith4001`：
   - 未鉴权先发消息 → `Error(code=20000,severity=fatal)` + close code `4001`；
   - 非法 token 鉴权 → 同样 `Error` + close `4001`。

验证：`BACKEND_URL=http://localhost:8082 go test -count=1 ./test/integration/` 全绿（6.3s，含原 92 个
HTTP 用例与新增 WS 用例）；`-race` 门禁与 CI E2E job 同样执行该套件，后续 WS 协议回归有据可依。

## Prometheus 告警规则（2026-10-25，第 47 轮）

补 P0-3 缺口：此前 `observability` profile 只有 scrape，没有 alert rules。新增
`docker/prometheus-alerts.yml`（7 条规则）并在 `prometheus.yml` 挂载，compose 增加只读卷：

| 告警 | 表达式口径 | 级别 |
|---|---|---|
| HotspotBackendDown | `up{job=hotspot-backend} == 0` 持续 2m | critical |
| HotspotHTTP5xxRateHigh | 5xx / 全部请求 > 1% 持续 5m（契约下业务错误是 HTTP 200，只覆盖框架级异常） | critical |
| HotspotP99LatencyHigh | 全局 p99 > 1s 持续 10m | warning |
| HotspotDBPoolStarved | `rate(pgxpool_empty_acquire_total[5m])` > 5/s 持续 10m | warning |
| HotspotWSDroppedEvents | 出站事件丢弃速率 > 1/s 持续 10m | warning |
| HotspotMemoryHigh | RSS > 850MiB 持续 10m（容器 limit 1g、GOMEMLIMIT 900MiB） | warning |
| HotspotGoroutineLeak | `go_goroutines > 2000` 持续 15m | warning |

**验证（本机实测）**：
- `promtool check config /etc/prometheus/prometheus.yml` → `SUCCESS: 1 rule files found` /
  `SUCCESS: 7 rules found`；
- `docker compose --profile observability up -d prometheus` 启动后：
  `/api/v1/rules` 返回 1 组 7 条规则，`/api/v1/targets` 显示 `hotspot-backend` target
  `health=up`（lastScrapeDuration 33ms），`up{job="hotspot-backend"}=1`；验证后已停止该 profile；
- CI Infra job 新增 `promtool check config` 步骤（镜像 tag 与 `PROMETHEUS_VERSION` 默认值 v3.4.1 对齐）。

**未包含 Alertmanager**：外发通知（邮件/IM）需要单独部署 Alertmanager 并配置 receiver；
当前告警仅在 Prometheus UI / Grafana 可见，接入方式在文件头注释说明，属部署侧决策。

## 写路径压测（2026-10-25，第 48 轮）

补 P0-2：`cmd/loadgen` 扩展写压测能力（`-idem-unique` 每请求唯一幂等键、`-body-seq` 请求体
`{{seq}}` 唯一序号、`-token-file` 多用户轮询 token），在 backend-test（APP_ENV=test）上压三类
写路径，并发 20、各 15s；结束后按 nickname 标记 `DELETE FROM users` 级联清理（ON DELETE CASCADE）。

### 结果
| 写路径 | RPS | p50 / p95 / p99 | 结果 |
|---|---|---|---|
| `POST /public/v1/auth/guest`（建用户+会话） | **1,824** | 9.4 / 17.0 / 44.5 ms | 27,366 reqs 全 code=0 |
| `POST /api/v1/moments/create`（瞬间+地理位置） | **584** | 16.3 / 80.1 / 85.9 ms | 8,762 reqs 全 code=0 |
| `POST /api/v1/comments/create`（评论+通知 fanout） | **509** | 28.3 / 96.7 / 143.3 ms | 7,652 reqs 全 code=0 |

pgxpool：total 3→21、acquiring 0、`empty_acquire_total` 1→19（约 0.3/s，未饱和）；goroutines 21 稳定。
清理：按标记删除 27,547 个用户并级联清除其内容/会话；测试库无残留。

### 结论
1. **guest 注册**写入吞吐最高（1.8k RPS）：users+sessions+JWT+Redis 幂等键组合未成为瓶颈；
2. **内容写路径** 500~600 RPS：moments 主要耗时在写库+geo 索引，comments 含通知 fanout；
   p95 ~80-100ms 属于单机单实例 dev 栈的可接受量级，瓶颈仍是数据库往返而非 Go 层；
3. 写路径与读路径量级差异（读认证 1.3k / 写评论 0.5k RPS，同为 20 并发）说明**写多于读或需
   更高写入吞吐时应优先扩容数据库规格/连接池**，而非应用层重构；
4. 压测脚本教训：`Idempotency-Key` 若用「左对齐 + 尾部补零」生成，`i=1` 与 `i=10` 会生成**相同**
   字符串（补零吞掉位数差异），表现为随机 22002；改用 `uuid`/十六进制随机键后 0 冲突。
   幂等实现本身行为正确（同键并发 → 22002、完成后回放），该现象属脚本缺陷，已记录避免复踩。

复跑：`go run ./cmd/loadgen -url ... -method POST -body '...{{seq}}...' -body-seq -idem-unique
-token-file /tmp/tokens.txt -c 20 -d 15s`；清理按 `nickname LIKE '<marker>%'` 删除用户级联。

## WS E2E 关闭断言去时序依赖（2026-10-25，第 49 轮）

收尾复核时 `--all` 暴露 WS E2E 偶发失败：客户端发 close 帧后，`SetSubscriptions` 触发的
`SubscriptionsSnapshot` 仍可能排在 close 之前到达，旧断言「下一条必是关闭错误」在事件未读尽时
会误判。修复：关闭阶段循环读直至错误/超时，只断言「服务端最终关闭连接」，不假设 close 帧是
下一条消息；`-count=5` 重复运行通过，最终 `--all` 全绿。

## 认证读路径性能优化（2026-10-25，第 50 轮）

目标：提升认证后高频读接口（`/api/v1/users/account`、`/api/v1/system/bootstrap`）的吞吐与延迟，
降低数据库连接池压力。方法：先在 backend-test（APP_ENV=test，独立库）用 `cmd/loadgen` 建立基线，
再逐项修改、同环境复测（并发 50、20s，同机压测，数字用于相对比较）。

### 基线（提交 c6fa04aa）
| 指标 | users/account |
|---|---|
| RPS | **1,514** |
| p50 / p95 / p99 | 13.75 / 81.66 / 123.29 ms |
| pgxpool empty_acquire（20s） | **106,051**（≈6 次/请求，池持续打满） |

### 优化 1：会话校验进程内缓存（提交 4c6c2e11）
`AuthenticateRequest` 每个受保护请求都要按 session_id 查一次会话。新增
`platform/auth.CachedSessionStore`（SessionStore 装饰器）：Get 结果进程内缓存（TTL 5s、上限 4096，
错误/不存在不缓存），Save/Touch/Revoke/RevokeOthers 同步失效；admin 封禁直接 UPDATE
`auth_sessions` 绕过装饰器，新增 `Service.InvalidateSessionCache` + admin 侧失效钩子接线，保证撤销
即时生效。跨实例/直连改库的撤销最多延迟 5s（当前单实例部署；多实例需改为 Redis 广播/共享缓存，
已在代码注释与本节注明）。

### 优化 2：get_account 三次查询合并为单条 SQL（提交 c4155891）
`GetProfile` 原有 3 次往返（画像行 / 5 个 COUNT 统计 / 标签列表）。压测显示单请求约 6 次连接池
acquisition——会话 1 次 + 画像 3 次 + 其他。新增 `queryGetUserProfileDetails`：一条 SQL 返回画像
15 列 + 5 个统计标量子查询 + `array_agg(tag_name ORDER BY tag_name)` 标签，语义与旧实现一致，
单请求 DB 往返 3→1；旧查询与 helper 删除避免死代码。

### 复测结果（同环境、c=50、20s）
| 版本 | RPS | p50 / p95 / p99 | empty_acquire | 说明 |
|---|---|---|---|---|
| 基线 c6fa04aa | 1,514 | 13.75 / 81.66 / 123.29 | 106,051 | — |
| +会话缓存 4c6c2e11 | 1,533 | 13.19 / 81.07 / 215.36 | — | 单一优化被 3 次画像查询掩盖，池仍打满 |
| +单条 SQL（两项合并） | **1,691** | **11.04** / 81.99 / 210.38 | **20,498** | 吞吐 +11.7%，p50 −20%，池取空 **≈9×↓**（≈0.6 次/请求） |

补充：c=100 时 1,390 RPS / p50 66.8ms（吞吐不升反降）——同机压测进程与后端抢 CPU，
单实例吞吐在 ~1.7k 见顶，**瓶颈已不是数据库连接池**；生产多核/独立压测机需在准生产复测。

### 结论与后续
1. 读路径优化的关键是**减少每请求查询次数**（3→1 比加缓存更有效）；缓存单独使用在
   「其他查询仍打满池」时收益被掩盖，两者组合才有明显效果；
2. 剩余的池压力主要来自 bootstrap 等聚合端点（同样多查询），下一轮可沿同一思路收敛；
3. 缓存一致性边界已显式处理：同实例撤销即时生效（含 admin 封禁）；跨实例限制 5s TTL；
   多实例部署前需升级失效广播方案。

## bootstrap 并行化实测：负结果与回滚（2026-10-25，第 51 轮）

CPU profile（bootstrap c=50）显示 pgx Query 累计占 27%（3 次串行 DB 读取：公告版本 /
登录画像 / 未读摘要），据此尝试用 `sync.WaitGroup` 把三个独立读取并行化（每个 goroutine
带 recover，语义与降级行为不变），同环境 c=50、20s 复测：

| 版本 | RPS | p50 / p95 / p99 |
|---|---|---|
| 串行（基线） | 1,286 | 16.53 / 84.88 / 211.26 ms |
| 并行 | 1,267 | 17.11 / 86.46 / 102.43 ms |

**结论：并行化无吞吐收益、p50 略升，不保留该改动，已回滚。** 原因分析：c=50 时连接池
（25 连接）本就饱和，请求延迟由**排队时间**主导而非串行 RTT 之和；并行后每个请求同时占用
最多 3 个连接，反而放大队头排队（总连接秒不变，并发占用曲线更尖）。p99 改善来自调度方差，
不构成保留理由——**饱和并发下串行读取更公平、实现更简单**。

该负结果的方法论价值：性能改动必须以同环境实测为准，profile 中的「cum 27% DB 时间」
不等于并行化就能提速；只有在低并发/延迟敏感场景（排队不占主导）时并行才有意义。

后续方向改为**减少查询次数**（如公告版本缓存 + 写路径失效），而不是并行化。

## bootstrap 公告版本读缓存（2026-10-25，第 52 轮）

第 51 轮并行化负结果后改为**减少查询次数**：bootstrap 每请求都要读公告版本生成 ETag
（`queryAnnouncementsVersion` 为 `MAX(updated_at)` 扫描），但公告变更是低频事件。

### 设计
- 缓存放 `system.PostgresRepository`（DB 访问层）：5s TTL、查询失败不缓存；
- 同实例写（`Create/Update/DeleteAnnouncement` 成功后）主动失效；
- admin 模块用独立 `systemRepo` 实例写公告，bootstrap 接线
  `admin.Service.SetAnnouncementsVersionInvalidator` → `system.Service.InvalidateAnnouncementsVersion`
  显式失效（ETag 立即变化，不等 TTL）；跨实例/绕过钩子的写入以 5s TTL 兜底；
- 内存仓库不加缓存（cheap），既有「公告发布后 bootstrap_version 递增」回归测试保持通过。

### 复测（同环境 c=50、20s，backend-test）
| 版本 | RPS | p50 / p95 / p99 | 每请求 DB 读取 |
|---|---|---|---|
| 基线（无版本缓存） | 1,286 | 16.53 / 84.88 / 211.26 ms | 3 |
| +公告版本缓存 | **1,536（+19.4%）** | **12.32** / 84.26 / 215.01 ms（p50 −25.5%） | 2 |

单测覆盖：缓存命中/失效/TTL/错误不缓存/写后失效；admin 公告 create/update/delete 各触发一次
失效（含独立 systemRepo 实例场景）。

### 累计读路径优化（第 50–52 轮）
| 优化 | users/account | bootstrap |
|---|---|---|
| 基线 | 1,514 RPS / p50 13.75ms | 1,286 RPS / p50 16.53ms |
| 会话缓存 + 画像单查询 + 公告版本缓存 | **1,691 RPS（+11.7%）/ p50 11.04ms** | **1,536 RPS（+19.4%）/ p50 12.32ms** |

说明：均为同机 dev 压测（压测进程与容器共享 CPU），用于相对比较；生产容量需准生产独立压测机复测。

### 下一步
- 继续收敛其他聚合读端点（搜索/Feed 等按调用频率与查询数评估）；
- 优化后重跑 `--all` 全量验收并收口。

## 读端点延迟盘点与「搜索索引」复核（2026-10-25，第 53 轮）

继续按「调用频率 × 查询数」找优化点：先对高频读端点做**单请求延迟盘点**（无并发，
hotspot_test 现有 7,767 users / 972 moments / 243 notifications），避免用高并发压测的
失真数字下结论：

| 端点 | 单请求平均延迟（5 次采样） |
|---|---|
| `notify/fetch` | 2.5 ms |
| `notify/unread_summary` | 2.8 ms |
| `users/search`（1 字符关键词） | 28.0 ms |
| `users/search`（2/3/4 字符关键词） | 20.0 / 5.9 / 6.5 ms |
| `groups/fetch_joined` | 2.8 ms |
| `moments/fetch_following` | 7.3 ms |
| `users/account` | 3.8 ms |

### 搜索慢的真相与结论
- `users/search` 的子串匹配（`nickname/bio ILIKE '%kw%'`）**早已有 pg_trgm GIN 索引**
  （迁移 `045_query_indexes.sql`：`idx_users_nickname_trgm` / `idx_users_bio_trgm`，
  `040/052` 亦有 moments/tags 的 trgm 索引）——原计划的「新增 trgm 迁移」是**重复工作，已放弃**；
- 3 字符及以上关键词 6ms 左右（索引可生效）；**1–2 字符关键词天生无法用 trigram**
  （pg_trgm 要求 pattern ≥3 字符），7.7k 用户下顺序扫描 + top-N 排序约 14ms SQL 执行；
  这是查询语义（任意子串）决定的固有成本，未改契约/未加最小长度限制；
- 结论：当前规模不需要新的搜索索引/查询改造；若未来 users 到十万级，可评估
  「短关键词专用路径」或搜索服务拆分，而非继续加同名索引。

### 压测环境教训（重要）
本轮对 `users/search` 做 50 并发压测（每请求全表扫描、CPU-bound）时，OrbStack 虚拟机
两次出现 Docker daemon 卡死（`docker info` 超时、网关 502），恢复方式仍为
quit/reopen OrbStack + `compose up -d` + 重启 backend。此后改为**低并发单请求采样**。
结论：**本机 dev 环境不适合对 CPU-bound 端点做 ≥50 并发压测**——会先压垮虚拟机而非暴露
应用瓶颈；容量结论应以准生产独立压测机为准（与第 45 轮结论一致，本轮补充了触发条件）。

### 第 50–53 轮累计成果
| 优化 | 效果（同环境实测） |
|---|---|
| 会话校验进程内缓存（TTL 5s + 撤销即时失效 + admin 封禁钩子） | 与下两项合并计入 |
| `get_account` 画像 3 查询合并为 1 条 SQL | users/account 1,514→1,691 RPS（+11.7%），p50 13.75→11.04ms |
| bootstrap 公告版本读缓存（写路径显式失效） | bootstrap 1,286→1,536 RPS（+19.4%），p50 16.53→12.32ms |
| 读端点延迟盘点 | 除短关键词搜索外均在 2–8ms；确认无需新增索引 |

## pgx 预编译语句：运行期查询改用 extended protocol（2026-10-25，第 54 轮）

第 49 轮 CPU profile 显示 simple protocol 的客户端转义（`sanitize.rawState`）约占 2.7% CPU，
且服务端每次查询都要重复解析/规划。此前用 simple protocol 是为了让**迁移文件多语句**
能在一条消息内执行（extended protocol 的 Prepare 不允许一条语句里多条 DDL）。

### 设计：运行期预编译 + 迁移强制简单协议
- 连接池默认 `pgx.QueryExecModeCacheStatement`（extended protocol + 语句缓存）；
- 迁移执行新增 `persistence.PostgresDB.ExecSimple`：以 `QueryExecModeSimpleProtocol`
  逐文件执行（`migrate.execMigrationSQL` 优先使用它，stub/内存实现回退普通 `Exec`），
  保持「整文件单消息」语义与「禁止 CREATE INDEX CONCURRENTLY」的隐式事务约束不变；
- 若未来接 PgBouncer 事务池（不支持跨事务命名语句），改回 SimpleProtocol（代码注释已写）。

### 附带暴露并修复的潜在 SQL 类型问题
Postgres 在 prepare（extended protocol）阶段**无法从 `$n IS NULL` 推断参数类型**——
simple protocol 是把参数内联成字面量，所以旧实现没暴露。扫描全仓后仅两处缺 cast，已补：
- `share` 列表：`$2::text` / `$3::timestamptz`（`TestShare_MeList` 10600 的根因）；
- `moments` 收藏列表：`$2::timestamptz`。
其余可空参数（notify/comments/reactions/admin 等）此前已带 cast。

### 压测对比（backend-test，同机 c=30、10s，每次新建访客 token）
| 端点 | simple（RPS / p50 / p95） | prepared（RPS / p50 / p95） | 变化 |
|---|---|---|---|
| `users/account` | 2,258 / 5.14 / 66.79 ms | 3,303 / 4.90 / 39.02 ms | **+46% RPS，p95 −42%** |
| `system/bootstrap` | 2,007 / 6.16 / 67.16 ms | 2,658 / 6.33 / 39.86 ms | **+32% RPS，p95 −41%** |
| `notify/fetch` | 10,286 / 1.80 / 5.02 ms | 11,342 / 1.92 / 5.18 ms | +10% RPS（单查询、轻负载，噪声内） |
| `moments/fetch_following` | 855 / 13.25 / 88.21 ms | 2,890 / 5.63 / 50.53 ms | **+238% RPS，p50 −58%** |

说明：prepared 版本在合并后首次测量时更高（account 4,310 / bootstrap 4,035 / following 3,129），
两次测量差异来自同机 dev 环境抖动（压测进程与容器共享 CPU、OrbStack 虚拟机）；此处取**较保守的一次**。

### 验证
- 全新库迁移：49/49 迁移文件成功、45 张表、`schema_migrations`=49（验证 `ExecSimple`）；
- 工作树门禁 + 主树 `verify-backend-ci.sh --all`：**ALL CHECKS PASSED**（含真实库 IT 与 ~100 条 HTTP E2E，
  首轮曾失败于 share 列表，补 cast 后通过）；
- dev 真实库冒烟（`localhost:8888`）：guest 登录/account/bootstrap/search/notify/following/nearby/
  share 列表全部 `code=0`（share/create 因冒烟脚本未带 Idempotency-Key 返回 10000，属预期校验）。

### 发版影响
该改动已合入 main（`b11ab0ba` + `200ae1ae`），v1.4.0 的候选 HEAD 因此前移；
打 tag 前需按发布门禁**对新 HEAD 重跑**生产镜像本地构建与 OCI 指纹核对（前端无改动，browser QA 结论仍有效）。

## fetch_nearby 候选集裁剪：只对分页结果补全标签/收藏（2026-10-25，第 55 轮）

`moments/fetch_nearby` 原实现拉取最多 2000 个候选行后，对每个候选行执行 6 个相关子查询，
再对**全部候选**加载标签，最后才分页出 ≤20 条。梳理发现：

- `is_liked` EXISTS：SQL 扫描后在 Go 里一直 `_ = isLiked`（契约已无该字段，reactions 聚合另行填充）——纯浪费；
- `is_favorited` EXISTS：响应需要，但只需分页后 ≤21 条，不需要对 2000 个候选逐行判断；
- 标签同理：排序（距离/热度/时间/ID）不依赖标签，无需全量加载。

### 改动
- `queryNearbyMoments`：删除 is_liked、is_favorited 两个 EXISTS；保留排序所需的 is_following、
  点赞实时 COUNT 与双向拉黑过滤；
- `queryFollowingMoments`：同步删除未使用的 is_liked EXISTS；
- 分页后用页面 ID 批量加载 tags（原有 `loadMomentTags`）与 favorites（新增 `loadMomentFavorites` +
  `queryMomentFavoritesByIDs`），响应字段语义不变。

### SQL 层 A/B（同库 1032 条候选、同参数，EXPLAIN ANALYZE）
| 版本 | Execution Time | Planning Time | Buffers |
|---|---|---|---|
| 旧（6 个逐行子查询） | 12.500 ms | 5.929 ms | 838 |
| 新（4 个逐行子查询，标签/收藏页面级） | **4.997 ms** | **2.256 ms** | 782 |

### 端到端压测（backend-test 同机 c=10、10s）——注意测法坑
- **默认请求（不传 reference_point）** 走存储位置/default_location 回退，
  hotspot_test 的 1032 条种子瞬间集中在同一点，回退点命中的候选集很小：
  改前 739 RPS / p50 5.95 / p95 64.24ms，改后 686 RPS / p50 6.16 / p95 65.62ms——**在噪声内，无差异**；
- **显式 reference_point 指向种子聚类点**（候选集 1032 条）：改后 257 RPS / p50 32.49 / p95 72.82ms；
  旧二进制未单独构建，故该场景以 SQL 层 A/B 为准（旧 SQL 每次多 ~7.5ms）；
- 结论：本次收益在候选集稠密时生效；候选集稀疏时端点被 Go 排序/批量补全主导，SQL 收益被摊薄。

### 语义验证（真实库）
- 新建带标签瞬间 → `set_favorite(active=true)` → `fetch_nearby`（reference_point 指向该点）：
  页面内能查到该条，`tags=["sem3-tag"]`、`is_favorited=true`，与分页后补全逻辑一致；
- 工作树门禁 + 主树 `verify-backend-ci.sh --all`：**ALL CHECKS PASSED**（真实库 IT + ~100 条 E2E）。

### 教训
HTTP 聚合压测会掩盖 SQL 层收益：必须用代表性数据/参数测量，SQL 改动以 EXPLAIN ANALYZE 做 A/B，
再用端到端压测确认没有退化（本轮默认参数下端点数字不变，属预期而非未生效）。

## 搜索/关注流查询收敛：标签与可见性并入主查询（2026-10-25，第 56 轮）

`users/search` 原 4 次数据库往返（主查询 + 标签批量 + settings 批量 + 关系批量），
`moments/fetch_following` 原 5 次（主查询 + 标签批量 + 图片 + 反应 + 作者关系）。

### 改动
- `users/search`：主查询（本就 JOIN `user_settings`）顺带 SELECT 标签数组与
  `presence_visible/location_visible/stats_visible` 三列；新增 `scanSearchProfileRow` 一次扫描；
  `PaginatedProfiles.Settings` 为可选快路径，Service 命中时跳过 `GetSettingsBatch`；
  内存仓库保持 nil 走原回退（行为不变）→ 4→3 次往返；
- `moments/fetch_following`：主查询（≤limit+1 行）顺带 `array_agg` 标签，删除对 ≤21 行的
  标签批量查询 → 5→4 次往返；响应字段语义不变；
- 单测：搜索内联 settings 快路径断言「不调用 GetSettingsBatch」；stub 行补标签/可见性列；
  users 覆盖率 69.8%→70.3%（门禁 70%）。

### 压测（backend-test 同机 c=30、10s，同一访客 token 每轮新建）
| 端点 | 改前基线 | 改后（两次测量） |
|---|---|---|
| `moments/fetch_following` | 3,027 RPS / p50 5.10 / p95 47.29ms | 3,595 / 4.05 / 48.88；3,270 / 4.60 / 50.09 |
| `users/search`（keyword=abc，0 命中） | 1,637 / 8.26 / 67.48 | 1,489（首轮噪声）/ 1,838（复测）/ 7.64 / 65.52 |
| `users/search`（keyword=游客，密集命中） | 未测（本轮首次测） | **174 RPS / p50 185.45 / p95 287.93ms** |

结论：关注流往返减少带来约 +8~19% 吞吐（同机噪声较大，取区间）；搜索零命中场景无退化。

### 新发现的遗留问题（下一轮目标）
1–2 字符关键词（如「游客」）无法用 pg_trgm（要求 ≥3 字符），7.7k 用户下顺序扫描 +
候选 500 行的逐行标签聚合，使密集短关键词搜索在 c=30 时 p50 达 185ms；
下一步可把标签聚合推迟到分页后（keyword-only 搜索排序不依赖标签），并复核候选上限策略。

### 验证
- 工作树门禁 `ALL CHECKS PASSED`；主树 `verify-backend-ci.sh --all`：**ALL CHECKS PASSED**
  （真实库 IT + ~100 条 HTTP E2E）；
- 前端无改动；本轮后端提交 `4efd84ad`（HEAD 前移，v1.4.0 打 tag 前仍需对新 HEAD 重跑镜像构建与指纹核对）。

## 密集短关键词搜索：标签聚合延后 + 根因量化（2026-10-25，第 57 轮）

第 56 轮发现「游客」等 1–2 字符关键词搜索在 c=30 时 p50 185ms。本轮修正了其中最浪费的一段：
keyword-only 搜索不再对最多 500 个候选行逐行 `array_agg` 标签，改为分页后按页面 ID 批量补全
（tag 过滤路径仍需候选全量标签，保留内联；settings 可见性仍随主查询）。

### 效果
| 指标 | 改前 | 改后 |
|---|---|---|
| 单请求延迟（游客，5 次平均） | 18.3 ms | **13.4 ms（−27%）** |
| 单请求延迟（游，1 字符） | 19.3 ms | 19.3 ms（无变化） |
| c=30 密集（游客） | 174 RPS / p50 185.45ms | 160 RPS / p50 190.85ms（**CPU 饱和，无改善**） |
| c=30 零命中（abc） | 1,838 RPS（复测） | **2,043 RPS / p50 6.50ms** |
| c=30 tag 过滤（美食） | 未测 | **3,935 RPS / p50 4.17ms** |

### 根因量化（EXPLAIN ANALYZE，去标签列与带标签列均 ~8.4ms）
- 执行时间 8.407ms / Planning 2.067ms，主要成本是 `users ⋈ user_settings` 对 7,767 行做
  `ILIKE '%游客%'` 顺序扫描（标签聚合并非瓶颈，只有 26 次 index search）；
- 单请求 13.4ms ≈ SQL 8.4ms + Go/JSON ~5ms；c=30 时 30 并发把单核 CPU 打满，
  p50≈单请求 CPU×并发/可用核心，吞吐不再随优化线性提升；
- **2 字符是硬限制**：pg_trgm 要求 pattern ≥3 字符才能用 GIN 索引；中文两字词（游客/美食）
  无法命中 trigram，只能顺序扫描。1 字符同样如此。

### 结论与建议
- 本轮改动保留：单请求 CPU 降 ~27%（对 3+ 字符与 tag 路径无退化），是纯收益；
- 密集短关键词的吞吐上限由「顺序扫描 CPU」决定，属该索引方案固有成本；
- 若产品上需要短关键词搜索体验，候选方向（需产品/架构决策，不擅自改契约）：
  1) 客户端搜索框最小输入长度 3 字符（UX 约束，最省成本）；
  2) 评估 `pg_bigm`（中文 2-gram）扩展或独立搜索服务（ES/Meilisearch）；
  3) 对 1–2 字符关键词改前缀匹配（`ILIKE 'kw%'`，可走 btree）——属于契约语义变更，需 spec 先行。

### 验证
- 工作树门禁 + 主树 `verify-backend-ci.sh --all`：**ALL CHECKS PASSED**（真实库 IT + E2E）；
- 单测：keyword-only 页面级标签加载、带标签列扫描、inline settings 快路径；users 覆盖率 71.0%；
- 前端无改动。

## backend-test pprof 接入 + fetch_nearby 紧凑候选（2026-10-25，第 58 轮）

### 1) 测试实例暴露 pprof（可复现剖析入口）
`backend-test` 增加 `PPROF_ADDR=0.0.0.0:6060` 与宿主机 `127.0.0.1:6060` 映射（仅测试 profile，
不进入生产 compose/镜像）：
```bash
curl -o cpu.prof 'http://127.0.0.1:6060/debug/pprof/profile?seconds=20'
go tool pprof -top cpu.prof
```

### 2) 剖析发现
对 `fetch_nearby`（c=10、密集候选 1032 行）采样 20s：
- `runtime.mallocgc` cum 27%、`tryDeferToSpanScan` 12%、`wbBufFlush` 14.6%、
  `bulkBarrierPreWrite` 19.6% —— **GC 相关开销合计约 40%**；
- 根因：每个候选行都构造 `MomentItem/Location/UserInfo`（指针 + 字符串指针），
  实际分页只返回 ≤20 条，却为上千候选分配短命对象。

### 3) 改动（`39f48ff8`）
候选扫描改为值类型 `nearbyCandidate`（无指针/无嵌套响应结构，仅保留排序字段）；
排序后只为页面内 ≤limit 条构造 `MomentItem`；游标定位新增 `findNearbyCandidateStart`/
`compareNearbyCandidate`（与 `compareNearbyCandidates` 同序），NextCursor 语义一致；
tags/is_favorited 仍分页后批量补全，响应字段不变。

### 4) 复核（同一 pprof 口径，c=10、30s）
| 指标 | 改前 | 改后 |
|---|---|---|
| 20s 窗口 CPU 采样总量 | 31.22s（156% 核） | **19.72s（98.6% 核，−37%）** |
| 请求 p50 | 21.55 ms | **16.83 ms（−22%）** |
| p95 | 57.39 ms | 69.84 ms（同机噪声，另一轮 57ms 量级） |
| RPS | 365 | 352（±噪声，CPU 不再是唯一瓶颈） |
| GC 头部函数 | mallocgc 27% / wbBufFlush 14.6% / tryDefer 12% | tryDefer 14.9% / wbBufFlush 19.7%，mallocgc 跌出前 12 |

语义验证：带标签瞬间创建 → 收藏 → `fetch_nearby` 返回 `tags=["cmp-tag"]`、`is_favorited=true`、
`distance=0` ✓；工作树门禁 + 主树 `--all`（DB IT + E2E）**ALL CHECKS PASSED**。

### 5) 下一步（数据驱动）
剩余 CPU 大头是 `pgx.baseRows.Scan`（12.2%）与行解码（DataRow.Decode 3.9%）：候选仍读取
昵称/头像/描述/位置名等仅页面需要的字符串列。下一步可让候选扫描只取排序所需列
（id/时间/坐标/likes/comments/shares/is_following），分页后按页面 ID 一次补全展示字段，
进一步降低行解码与字符串分配。

## fetch_nearby 候选列裁剪：只扫排序列 + 页面详情批量补全（2026-10-25，第 59 轮）

第 58 轮把候选改为值类型后，pprof 显示剩余大头是 `pgx.baseRows.Scan`（12.2%）与
`DataRow.Decode`（3.9%）：候选行仍在读取昵称/头像/描述/位置名等仅页面需要的字符串列。

### 改动（`a5708b10`）
- `queryNearbyMoments` 只保留排序所需列：`moment_id/created_at/likes COUNT/comments/shares/
  latitude/longitude/is_following`；
- 分页后对 ≤limit 条结果发一次 `queryNearbyMomentDetails`（JOIN users）补全
  `user_id/昵称/头像/描述/位置名`；
- `nearbyCandidate` 进一步瘦身为排序字段；响应构造只在页面内条目上发生；
- tags/is_favorited 仍按同一批页面 ID 批量补全；响应字段与排序规则不变。

### 复核（同一 pprof 口径，c=10、30s 密集候选 1032 行）
| 指标 | 原始（第 57 轮） | 紧凑候选（第 58 轮） | 列裁剪（本轮） |
|---|---|---|---|
| 20s CPU 采样 | 31.22s（156% 核） | 19.72s（98.6% 核） | **14.89s（74.4% 核，再 −24.5%）** |
| p50 | 21.55 ms | 16.83 ms | 16.2–19.1 ms（噪声内持平） |
| RPS | 365 | 352 | 289–336（±噪声，CPU/请求已降至原 74.4%） |
| 头部 CPU | mallocgc 27% | tryDefer 14.9%/wbBuf 19.7% | syscall 8.5% / wbBuf 12.3% / bulkBarrier 13.0% / Scan 7.9% |

两轮重复测量（20s、无 profile）：300 RPS / p50 18.12、336 RPS / p50 16.21，与上一轮持平；
累计 CPU 采样从 31.22s 降至 14.89s（**−52%**），说明候选扫描/解码成本已大幅下降，
当前吞吐受同机 CPU 与 DB 往返共同限制。

语义验证：新建带标签瞬间 → 收藏 → `fetch_nearby` 返回
`description='cols nearby'`、`tags=['col-tag']`、`is_favorited=true`、`nickname='游客·fliz'`、
`location_name='Cols City'`、`distance=0` ✓；工作树门禁 + 主树 `--all` **ALL CHECKS PASSED**。

### 下一步
候选 SQL 仍有 4 个逐行子查询（reactions COUNT、is_following EXISTS、双向拉黑 NOT EXISTS ×2），
其中热度与关注只用于同距离时的次级排序。下一步候选：把热度与关注改为「候选集批量聚合查询」
（GROUP BY / ANY），SQL 只保留 id/时间/坐标 + 拉黑过滤，可进一步降低 DB 扫描成本。

## fetch_nearby 热度/关注改候选集批量聚合（2026-10-25，第 60 轮）

第 59 轮后候选 SQL 仍对最多 2000 行做 4 个逐行子查询（reactions COUNT、is_following EXISTS、
双向拉黑 NOT EXISTS×2）。本轮把热度与关注改为候选集批量聚合：

### 改动（`3eb4ee6e`）
- `queryNearbyMoments` 只保留 `moment_id/user_id/created_at/comments_count/shares_count/lat/lng`
  + 双向拉黑过滤（正确性所需），去掉 users join、热度 COUNT 与关注 EXISTS；
- 新增 `queryNearbyMomentHeat`（`GROUP BY target_id` 的 reactions 计数）与
  `queryNearbyFollowingUserIDs`（`ANY(authorIDs)` 关注集合），Go 层合并进排序分数；
- score 公式不变（likes+comments+shares + 1000×isFollowing + max(0, radius-distance)）；
- 匿名视角跳过关注批量查询；候选为空提前返回；响应 likes 语义由同一批量结果填充。

### 复核（c=10、密集候选 1032 行）
| 指标 | 列裁剪后（第 59 轮） | 批量聚合后（本轮） |
|---|---|---|
| RPS | 289–336 | **498–502（+50%+）** |
| p50 | 16.2–19.1 ms | 18.4–18.5 ms（持平） |
| p95 | 70–77 ms | **32.5–34.0 ms（−55%）** |
| p99 | 77–88 ms | 41 ms |
| 候选逐行子查询 | 4 个/行 | 0（仅拉黑过滤） |

同轮 pprof（c=10、30s）显示吞吐提升后 CPU 采样随之上升（请求量 +50%），但 p95/p99 明显收敛；
`fetch_following` 无回归（c=30 2,974 RPS，历史噪声区间 2.9–4.1k）。

### 语义验证（真实库）
新建瞬间 → `reactions/set(emoji=👍, active=true)`（code=0）→ 收藏 → `fetch_nearby`：
该条 `reactions=[{emoji:👍,count:1,reacted_by_me:true}]`、`tags=["bat-tag"]`、`is_favorited=true` ✓。
（`likes_count` 已从契约删除，JSON 不返回该字段，属预期。）
工作树门禁 + 主树 `--all`（DB IT + E2E）**ALL CHECKS PASSED**。

### 下一步
附近流已从 365 RPS / p95 57ms 优化到 ~500 RPS / p95 33ms。剩余 CPU 主要是 GC
（tryDeferToSpanScan/wbBufFlush/bulkBarrierPreWrite 合计约 30%），可继续减少候选集
临时 map/切片分配；或把同款「候选批量聚合」模式复用到其他发现流（moments/search 等）。

## 附近流页面补全合并 + 测试库污染教训（2026-10-25，第 61 轮）

### 1) 页面补全 3 查询合并为 1（`60ec4dfb`）
附近流分页后原本要发 3 次查询：展示字段、标签数组、收藏态（均只针对 ≤limit 行）。
合并为一条 `queryNearbyMomentDetails`：
- LATERAL `array_agg(mt.tag_name ORDER BY mt.tag_name)` 内联标签；
- 参数化 `EXISTS(moment_favorites)` 内联收藏态（匿名传空串，恒 false）；
- 附近流每次请求的 DB 往返从 8 次降到 6 次（候选/热度/关注/页面补全 + service 侧图片/反应/作者关系）。

压测（重置后重建的 1032 条密集候选数据集，c=10、20s×2）：
| 指标 | 第 60 轮（环境窗口 A） | 本轮（环境窗口 B） |
|---|---|---|
| RPS | 498–502 | 421–442 |
| p50 | 18.4 ms | **16.2–16.9 ms** |
| p95 | 32–34 ms | 53.6–55.6 ms |

两轮不在同一环境窗口（OrbStack 抖动 + 数据集/作者分布不同），p95 差异主要是噪声；
该改动是**结构性减少往返与 map 分配**，语义验证通过，故保留并如实记录「未观察到明确吞吐提升」。

### 2) 测试库污染导致 EXPLAIN 计划测试误报（重要教训）
首次 `--all` 失败于 `TestExplainUsesExpectedS2Indexes`（期望 S2 部分索引、实际走了别的索引）。
根因不是代码改动，而是 **hotspot_test 被本轮压测数据污染**（此前基准累计写入上万条瞬间/用户），
规划器在数据量变化后选择不同索引，导致依赖计划形状的测试误报。

处置与规范：`hotspot_test` 是测试库，可随时重建：
```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml --profile test stop backend-test
docker exec -it hotspot-postgres-1 psql -U postgres -c 'DROP DATABASE IF EXISTS hotspot_test WITH (FORCE)'
docker exec -it hotspot-postgres-1 psql -U postgres -c 'CREATE DATABASE hotspot_test'
# 再启动 backend-test 自动建表/迁移，然后重跑 --all
```
重建后 `--all` 恢复 **ALL CHECKS PASSED**；建议在发版门禁/大改动验证前先重建一次测试库。

### 3) 语义验证（真实库）
重建后重新造 1032 条密集候选数据，创建带标签瞬间 → 收藏 → `fetch_nearby`：
`description='page hydration'`、`tags=['pg-tag']`、`is_favorited=true` ✓。

## 只读性能冒烟脚本 + 当前热点盘点（2026-10-25，第 62 轮）

### 1) `scripts/perf-smoke.sh`（新增）
固定读端点（account/bootstrap/notify/following/nearby/users-search）一键压测，输出
rps 与 p50/p95/p99，用于**同一会话内的前后对比**；非测试环境要求显式传 `TOKEN`（不新建访客）。
用法与注意事项已写入 `backend-go/AGENTS.md`「只读性能冒烟」。

### 2) 同会话端点盘点（c=10、10s，当前 HEAD）
| 端点 | RPS | p50 | p95 |
|---|---|---|---|
| users/account | 3,672 | 1.73 ms | 4.29 ms |
| system/bootstrap | 2,636 | 2.48 ms | 10.22 ms |
| notify/fetch | — | — | —（同批测量中约 0.5–1 ms 量级） |
| moments/fetch_following | 2,817 | 2.30 ms | 6.55 ms |
| moments/fetch_nearby（1032 候选） | 651 | 14.46 ms | 23.75 ms |
| moments/search（keyword=seed，全部命中） | 859 | 6.41 ms | 49.60 ms |
| users/search（密集短关键词） | 2,148 | 3.50 ms | 11.13 ms |

### 3) 关键结论
- **account/bootstrap 已接近 I/O 极限**：account c=20 剖析总 CPU 仅 52%（syscall 占 21%），
  p50 4.09ms、p95 52ms 的长尾来自同机 VM 调度/DB 往返抖动，继续改 Go 代码收益很小；
- **moments/search 的 p95 高是合成数据最坏情况**：`LIKE '%seed%'` 命中全部 1020 行，
  SQL 需要全量过滤+排序；真实低选择性关键词会走 pg_trgm 索引，不代表线上形态；
- **nearby 仍是真实最重读路径**（已从 257 RPS/p95 73ms 优化到 651 RPS/p95 24ms 量级）；
- 本机 OrbStack 不同时间窗吞吐波动可达 ±2 倍，**跨时间绝对值不可比**；
  第 7 轮实测同一代码同一数据在 498–502 RPS 与 134–374 RPS 之间波动，即属此类抖动。

### 4) 下一步建议
- 应用层热点已基本收敛：继续微优化收益低且易被 VM 抖动掩盖；
- 若要再上一个量级，应转向架构：多实例部署（会话/公告缓存跨实例广播）、
  独立压测机（准生产环境）验证真实容量，或搜索服务化（解决 1–2 字符中文关键词只能顺序扫描的问题）。

## 架构审计启动 + 实时/精度 P0 修复（2026-11-06，第 64 轮）

按用户要求转入**架构级改进**阶段（明确不考虑向后兼容；spec 亦可按更优方案修订），
先完成 6 个只读深审（持久化/DB、平台基础设施、实时/聊天、业务模块架构、spec 契约、
工程效能），综合结论与分批执行计划见会话工作区 `.workwork/design/backend-improvement-plan.md`
（审计原文在 `.workwork/audit/`，共约 3800 行，按域列 P0/P1/P2 与 file:line 证据）。

本轮已修复（3 个提交）：

1. **WS resume 回放丢事件**：断线积压超过连接出站队列（128）时，旧实现用非阻塞
   `Send` 灌回放事件，队列满即丢且丢的是最新事件（恰是重连要补的）。改为带总时限的
   阻塞入队，新增 200 条积压回归测试。
2. **fanout 每连接重复编码**：广播改用 `websocket.PreparedMessage`，事件 JSON 与
   permessage-deflate 结果按（服务端/压缩级别）只准备一次。
3. **resume 基线取全局 LastSeq**：改用连接真正写入 TCP 的最大 `ServerSeq`
   （全局 seq 会被其它用户事件推进，会把本用户未送达事件永久跳过）。
4. **Hub 锁内 Close 慢连接**：逐出连接改为锁外关闭，消除单次最长 5s 的全 Hub 停顿；
   `BroadcastToAll` 移除 1000 条 drop-tail 截断（万连接下公告只覆盖 10%）。
5. **写路径时间戳秒级截断**：moments/comments 创建改为直接写 `time.Time`（timestamptz
   微秒），对外展示仍按秒；相关 keyset 游标改为 RFC3339Nano。真实库契约测试当场复现
   并确认了修复前「同秒评论翻页重复」缺陷（comments / notify 游标改为从行内真实时间编码）。
6. 加固：block 关系缓存值解析防越界 panic；`verify-backend-ci.sh` 与
   reception-bot 脚本在 macOS bash 3.2 下「全角括号紧邻 `$VAR`」被并入变量名的
   展开错误（Linux CI 不复现，属本地/CI 行为分叉）。

验证：`./scripts/verify-backend-ci.sh --with-db` → ALL CHECKS PASSED（含 race/覆盖率/
真实库 IT/契约测试/迁移索引一致性）。

后续批次（数据模型与迁移地基、架构收敛、性能体系、可观测性/契约/spec）见上述计划文件。


## Batch B1：迁移体系加固（2026-11-06）

持久化审计 P0-3/P1-7 落地：旧 runner 的「存量库启发式基线（只登记不执行）」已导致线上
索引被永久吞掉（058 迁移自述）。新实现（提交见 git log）：

- `schema_migrations` 增加 `checksum`（sha256）/ `execution_ms`：已应用迁移文件改动或删除
  会在启动期 fail-fast；旧版本行首次启动自动补写 checksum（一次性过渡）。
- **删除启发式基线**：库内有业务表但版本表为空时启动失败，必须
  `go run ./cmd/migrate -adopt-through=<file>` 显式接管；adopt 后立即 manifest 校验。
- **每文件与版本登记同事务提交**（原子）：失败/崩溃不会留下「已执行未登记」窗口或半应用状态。
- **schema manifest**：`migrations/schema_manifest.json`（43 表 / 391 列 / 128 索引）作为
  「迁移集合应产生的 schema」快照，启动与 `cmd/migrate -verify-only` 校验期望对象齐全；
  `verify-backend-ci.sh --with-db` 新增「临时库重放 + manifest diff」门禁，迁移改了忘重建 manifest 即失败。
- `cmd/migrate` 扩展：默认应用、`-verify-only`、`-adopt-through`、`-write-manifest`。
- 迁移纪律调整：破坏性 DDL 允许（用户明确不考虑向后兼容），但迁移文件必须声明
  `-- allow-destructive: <理由>`（scripts/check-migrations.sh 校验）；CONCURRENTLY 仍禁止（事务内执行）。
- 验证：`verify-backend-ci.sh --with-db` ALL CHECKS PASSED（含 legacy 拒绝/adopt/checksum 篡改检测的手工实测）。


## Batch B2：删除死表死列与死抽象（2026-11-06）

持久化审计 P1-2 落地：

- 迁移 060（带 \`-- allow-destructive\` 声明）删除 0 引用的历史死表
  `moment_likes` / `comment_likes` / `feed_dismissals`，以及只写 0 的
  `comments.likes_count` 列（reactions 已完全取代前两者语义；契约 CommentItem 不含
  likes_count，仅 repo 内部扫描）。
- Go 侧：comments repository/类型/测试移除 likes_count 读写；seed 数据构建删掉三张死表
  的写入与断言（种子规模 38 表 396 行 → 36 表 358 行）；dead abstraction
  `persistence.Repository[T]` / `ErrNoTransaction` / 未使用的软删 SQL 常量删除。
- `scripts/check-migration-indexes.sh` 的「期望索引」来源从解析历史 SQL 改为
  `migrations/schema_manifest.json`（自动反映 DROP TABLE/INDEX 后的净结果）。
- `cmd/migrate -write-manifest` 使用跳过 manifest 校验的内部路径（新迁移改变期望
  schema 时先应用再生成新 manifest），生产启动路径仍严格校验。
- schema manifest 重建：43 表/391 列/128 索引 → 40 表/380 列/121 索引。
- 验证：`verify-backend-ci.sh --with-db` ALL CHECKS PASSED；在全新库执行
  `cmd/migrate` + 真实 `cmd/seed`（if-missing 与 reset 两种模式）均成功，
  死表确认不存在、行数与 dry-run 一致。


## Batch B3b：moments 计数去冗余列 + 内部互动热度列（2026-11-06）

持久化审计 P0-1/P0-2 收口：

- 迁移 062（-- allow-destructive）：删除 moments.likes_count / comments_count /
  shares_count；新增内部列 interaction_count = reactions + 未删评论 + 未撤销分享，
  仅用于 nearby/popular 排序（不对外序列化）；新增 shares 的
  `((payload->>'shared_moment_id')) WHERE share_type='moment' AND is_revoked=FALSE`
  表达式索引支撑 shares_count 读时聚合；一次性把存量数据回填进 interaction_count。
- 契约字段 comments_count / shares_count 改为读时聚合（列表/详情 SQL 内标量子查询；
  nearby 页后批量查询补全），同一对象跨端点数值一致，share 撤销后 shares_count 自然回减。
- 写路径：删除「评论 +1 / -1、reaction +1、分享 +1」三个计数列维护；改为统一
  Service.AdjustInteractionCount（comments/reactions/share 三方注入；失败仅 WARN，
  只影响排序启发式）。删除 moments 启动期全表自愈 UPDATE（P0-2）。
- nearby：候选查询由 comments_count+shares_count+单独 reactions 批量聚合改为直接读
  interaction_count（少一次 DB 往返）；popular 排序由「两列相加」改为 interaction_count。
- share 模块不再私建 moments 服务实例（审计指出的隐藏副本之一），改由 bootstrap 注入。
- 全局搜索 suggest moments 的 popularity 由 reactions LATERAL+两列改为 interaction_count。
- seed：moments 写 interaction_count（reactions+评论+分享行数），同步单测。
- schema manifest：40 表 / 377 列 / 123 索引。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED
（真实库契约测试覆盖计数聚合/分页/软删；迁移回填后 dev 库 interaction_count 0–16 与业务量一致）。


## Batch B4：keyset 游标精度全场收尾（2026-11-06）

持久化审计 P1-1 的剩余面：写路径已统一 time.Time（微秒），但部分列表的游标仍从
**契约展示字段**（按秒格式化的字符串）构造，同秒行翻页会重复/丢行。

- admin：audit logs / users / groups / reports 四个列表改为与 items 平行保存行内
  真实 CreatedAt（新增 scan*WithCreatedAt 变体；reportHandleRecord 增加
  CreatedAtRaw），游标取真实微秒时间。admin tags/rbac/stats/geo 原本已用 time.Time。
- share：列表 SQL 追加原始 created_at 列，游标用 RFC3339Nano；展示字段保持秒级。
- 复核其余模块：moments/comments/notify/users/groups/privatechats 已用真实时间；
  privatemessages/groupmessages 的 TO_CHAR 已含 .US 微秒；system announcements、
  admin moderation tag 均直接持有 time.Time；内存实现仅测试使用（保持展示字符串口径）。
- 验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。


## Batch B5：审计日志/终态举报保留期（2026-11-06）

持久化审计 P1-9 的第一批（成本最低、合规收益最高）：admin_audit_logs 与 reports
长期只增不减。

- 新增 admin 后台保留期清理（RETENTION_SWEEP_ENABLED 管控）：
  - 审计日志默认保留 365 天（ADMIN_AUDIT_RETENTION_DAYS，<=0 关闭）；
  - 只清已处理终态举报、默认 365 天（REPORT_RETENTION_DAYS，<=0 关闭），
    pending 举报永不自动删除。
  - 删除走 CTE + RETURNING 统计行数（persistence.Queryable 不暴露 affected rows），
    单轮 5 分钟超时，失败 WARN 不阻断；worker.GoRestartable 崩溃自愈。
- config 新增允许显式 0 的 envNonNegativeIntOrDefault（既有 envIntOrDefault 把 0
  当非法回退默认，无法表达「关闭」语义），单测覆盖默认/覆盖/0。
- 双实现契约测试补「保留期只删 cutoff 前数据」；PG IT 覆盖「pending 保留、
  终态超期删除、终态未超期保留」。
- compose/.env.example 注入两个新变量并更新保留期说明。

未做（设计文档已记）：messages/reactions 分区与清理、软删内容 purge（需先统一
deleted_at 模型）、周期任务多实例 jitter+分布式锁。验证：--with-db ALL CHECKS PASSED。


## Batch C（第一批）：模块边界一等化 + 可执行架构约束（2026-11-06）

架构审计 P1-6（模块边界虚化）与 P0-4（数据所有权）的第一步：

- **comments 升级为一等模块**：新增 `comments/module.go`（NewModule/Mount/
  MountPublic），评论路由与服务由 bootstrap 显式装配；moments 模块不再在
  NewModule 内部构造评论服务、不再挂载 /comments 路由。
- **privatemessages 升级为一等模块**：新增 `privatemessages/module.go`，消息
  路由/运行时由 bootstrap 装配；privatechats 只消费 `ReadService` 最小接口，
  不再内部构造消息模块。
- **新增 `internal/archtest` 可执行架构约束**（负例已验证会失败）：
  1) 模块间 import 边 allowlist：新增未登记 import 立即失败；
  2) 表归属登记必须覆盖 schema_manifest 全部 40 张表；
  3) SQL 写表扫描：模块写非归属表必须命中带原因的 allowlist（23 条既有债务
     已显式登记，清理后可直接删除条目）。
- 路由清单不变（E2E 全量契约测试覆盖）；模块列表从 15 个增至 17 个。

未做（后续）：admin→system 公告仓储隐藏副本、admin 10 处跨模块写改走归属模块
管理接口、错误模型统一（P0-3）、全局单例 DI 收敛（P0-1）、repo_memory 迁测试。


## Batch C（第二批）：公告归属 system + 跨模块仓储构造门禁（2026-11-06）

- **admin 不再自建 system.Repository 隐藏副本**：公告 CRUD 经 system 模块新增的
  `AnnouncementAdminStore` 最小接口（`system.Service` 实现）转发到唯一仓储，
  Repository 内部缓存失效天然同实例；删除 admin 的独立 systemRepo 构造、
  `SetAnnouncementsVersionInvalidator` 钩子及其过时测试。管理端公告测试改为
  断言「写入确实委托给注入的 system store」。
- **生产代码去除测试便捷构造函数里的跨模块内存仓储**：
  - privatemessages：`NewService`/`NewServiceWithRepo`/`NewRuntimeWithRepo` 移到
    `service_testhelpers_test.go`（生产只用显式依赖的 NewServiceWithDeps/NewRuntime）；
  - comments：删除内部 `notify.NewService()` 的 `NewService` 便捷构造，调用方改
    显式 NewServiceWithDeps；
  - moments：删除 `NewService()`（内部构造 notify 内存服务）。
- **archtest 新增规则**：模块生产代码不得构造其它模块的 Repository
  （`NewPostgresRepository`/`NewInMemoryRepository`），并带检测器自测样例，
  防止扫描空转；隐藏副本类问题从「靠审计发现」变为「提交即失败」。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。


## Batch C（第三批）：举报裁决的内容删除走归属模块（2026-11-06）

架构审计 P0-4（数据所有权）：admin 直接 UPDATE moments/comments 表的两条路径收敛：

- **comments 模块**新增 `DeleteAsAdmin`：无归属校验、目标不存在/已删除幂等成功；
  和自助删除共用 `softDelete`（软删 + 内部互动热度递减同一事务）。
- **moments 模块**新增 `DeleteAsAdmin`：无归属校验；软删 + MomentDeleted 广播
  （作者其他设备 + 收藏者占位更新）抽成 `softDeleteAndBroadcast`，与自助删除共用。
- **admin**：举报裁决（reports/judge）的 moment/comment 目标改为调用注入的内容
  审核最小接口（bootstrap 注入 moments/comments Service），删除 admin 仓库的
  DeleteMoment/DeleteComment 方法与 SQL；archtest allowlist 移除 admin:moments、
  admin:comments 两条债务。
- 顺带修复潜在缺陷：admin 原 comment 软删 SQL 仍在更新 B3a 已删除的
  `comments.replies_count` 列，举报裁决删除评论会运行时报错；改走 comments
  模块后该问题消失。
- 测试：comments/moments 各补 DeleteAsAdmin 幂等/无归属校验/广播路径用例；
  admin handler 测试改用内存内容审核 fake。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。


## Batch C（第四批）：举报裁决的消息撤回走归属模块（2026-11-06）

架构审计 P0-4：admin 直接 UPDATE group_messages/private_messages 的两条路径收敛。

- **groupmessages** 新增 `RecallMessageByIDAsAdmin`：按 message_id 查群归属后复用
  `RecallMessageAsAdmin`（无作者/时间窗限制，广播 GroupMessageRecalled）；不存在/
  已撤回返回 (false, nil) 供裁决侧 fallback。
- **privatemessages** 新增 repo `RecallAsAdmin`（PG + 内存实现、契约接口）与
  `RecallMessageByIDAsAdmin`：取双方参与者后审核撤回，向双方广播
  PrivateMessageRecalled；已撤回/不存在返回 false。
- **admin**：举报裁决消息目标改为调用注入的群/私聊消息审核接口（bootstrap 注入
  gm/pm Service），删除 admin 仓库 RecallGroupMessage/RecallPrivateMessage 及 SQL；
  archtest allowlist 移除 admin:group_messages、admin:private_messages 两条债务。
- 行为改进：此前 admin 裁决撤回消息只改库不广播，在线双方界面不会即时更新；
  现在与自助撤回同款事件下发。
- 测试：两模块补按 ID 审核撤回（命中/幂等/不存在）用例；privatemessages 断言双方
  均收到 PrivateMessageRecalled；admin handler 测试用内存消息审核 fake。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。


## Batch C（第五批）：admin 群解散走 groups 模块（2026-11-06）

架构审计 P0-4：admin 直接写 groups/group_applications/group_invites 的路径收敛。

- **groups** 新增 repo `DismissAsAdmin`（PG + 内存 + 契约接口）：软删群
  （deleted_at）+ 同事务作废该群全部 pending 申请/邀请（status=revoked，字段口径
  与 BanMember 级联一致）；与 owner 自助解散（硬删子表）语义区分。
- **groups.Service.DismissAsAdmin**：无 owner 校验、群不存在/并发已解散 →
  ErrGroupNotFound；广播 GroupDismissed(policy_violation)、摘除群订阅，并通过
  GroupDismissAuditWriter 写 group.dismiss 审计。
- **admin** 群解散/举报裁决群目标改调注入的群审核接口，删除 admin 仓库
  DismissGroup 与三条直写 SQL；archtest allowlist 移除 admin:groups、
  admin:group_applications、admin:group_invites 三条债务。
- 审计可读性保持：groups 只回传 operatorID，admin 的 AppendGroupDismissAudit
  按管理员资料补昵称（避免审计退化为裸 user_id）。
- 测试：groups 补审核解散用例（无 owner 校验/不存在/并发已解散/审计回调参数）；
  admin handler 测试用内存群审核 fake（含广播与审计模拟）。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。


## Batch C（第六批）：封禁状态/会话撤销走归属模块（2026-11-06）

架构审计 P0-4：admin 直写 users 与 auth_sessions 的两条路径收敛。

- **users** 新增 `AccountBanState` + Repository/Service `SetAccountBanState`
  （PG + 内存实现）：封禁/解封状态写入 users 表归 users 模块所有；admin 只做
  权限校验、目标校验、审计与错误映射。
- **platform/auth** 新增管理端会话能力（无需改 SessionStore 接口）：
  `ListActiveSessionIDsAsAdmin`（最近活跃优先、稳定排序）与
  `RevokeSessionAsAdmin`（不校验所有权，成功后自行失效会话校验缓存）。
- **admin**：封禁流程改调注入的 users 写入口 + auth 会话服务；删除 admin 仓库
  SetUserBanState/ListUserSessions/RevokeSession 与三条直写 SQL；删除
  `SetSessionCacheInvalidator` 钩子（缓存失效归 auth 服务内部）。
- **archtest allowlist 再减 2 条**：admin:users、admin:auth_sessions。
- 测试：auth 补管理端会话列出/撤销/幂等用例；PG ban-expiry IT 改用 users 仓储
  造数；admin 测试用内存封禁写入/会话服务 fake；bootstrap 授权测试保留内存
  封禁构造入口（仅内存实现，生产写路径已归属 users）。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。


## Batch C（第七批）：账户注销级联按归属模块拆分（2026-11-06）

架构审计 P0-4 最大一处债务（users 注销直写 8 张跨模块表）收敛：

- 7 个归属模块新增 `PurgeUserData`（仓储 SQL + Service 入口，type-assert 可选
  实现；内存/测试仓储无跨模块行时 no-op）：auth（凭证+会话）、groups（成员）、
  media（上传会话）、moments（收藏）、notify（通知）、privatechats（会话视角
  状态）、privatemessages（已读位点）。幂等，可整体重试。
- **users 仓储只处理归属行**：users 匿名化 + user_settings/user_relations/
  user_profile_tags；其余行不再由 users 直接 DELETE。
- **编排在组合根**：新增 `bootstrap.accountPurgeCoordinator`（固定顺序调用各模块，
  失败即中止）；users.Service 经 `AccountPurger` 接口注入。执行顺序为「先清
  归属模块数据、最后匿名化 users 行」——级联失败时账号仍可重试，不会出现
  已不可用却残留跨模块数据。
- 架构债务从 17 条降至 5 条（仅剩：admin:moment_tags、admin:reports、
  auth:users、auth:user_settings、privatemessages:private_chats；均为跨切面
  注册/审核/共享键表语义，已显式登记）。
- 测试：users 补「清理失败中止且不匿名化」用例；原 PG 注销 IT 改为验证
  users 仓储不再触碰外部行；新增 test/integration 逐模块真实库用例（目标用户
  行删除、peer 行保留、重复执行幂等）。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。


## Batch C（第八批）：内存仓储迁入测试构建（8/17 模块，2026-11-06）

架构审计 P0-2：「Repository 双实现 + 无 DB 优雅降级」不成立——生产装配永远只
构造 Postgres 仓储，内存实现只被测试使用却以生产代码形态承担漂移成本。

- **8 个模块**的 `repo_memory.go` 迁移为 `*_test.go`（生产代码净减约 1.5k 行）：
  auth、mapmodule、privatechats、reactions、reports、search、share、system。
- 共享声明拆分到生产文件（内存实现在测试构建中继续复用）：
  auth 的仓储行类型/节流常量 → `repo_types.go`；privatechats 会话键 → `repo_helpers.go`。
- 测试专用便捷构造函数（`NewService()` 等内部构造内存仓储者）移入
  `service_testhelpers_test.go`；生产构造函数只接受显式依赖，auth nil 仓储改为
  fail-fast panic（不再静默降级到内存实现）。
- system 覆盖率补测（模块装配/路由挂载/管理端公告方法/健康接线）：
  69.3% → 82.9%，8 个模块均满足 70% 门槛。
- 剩余 9 个模块（admin/comments/groupmessages/groups/media/moments/notify/
  privatemessages/users）被跨包测试夹具引用（bootstrap/users/comments/
  privatechats/reports 的测试直接用对方模块内存仓储），需先逐包替换为本地 fake
  或真实库 testdb 夹具，后续批次继续迁移。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。


## Batch C（第九批）：再迁 2 模块内存仓储 + 消费方接口收窄（2026-11-06）

- **groupmessages、media** 的 `repo_memory.go` 迁移为 `*_test.go`；生产代码继续
  减少内存实现（至此 10/17 模块已迁入测试构建）。
- **装配/适配器接口收窄**（架构审计 P0-1：消费方依赖窄接口而非具体 Service）：
  - reports.MessageSnapshotSource 的群消息源改为 `GroupMessageSnapshotSource`
    窄接口（GetMessageReportSnapshot + IsGroupMember）；
  - bootstrap 的举报媒体适配器改为 `completedFileMetaReader` 窄接口。
- 跨包测试夹具去内存依赖：
  - reports 快照测试改用本地群消息源 fake（不再引用 groupmessages 内存仓储）；
  - bootstrap 媒体适配器测试改用本地 FileMeta fake（媒体服务行为由 media 包测试覆盖）。
- 覆盖率补测：groupmessages 迁移后 67.5% → 75.3%（仓储剩余读/审核路径骨架用例）；
  media 72.8%；两模块均满足 70% 门槛。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。


## Batch E（第一批）：就绪探针与服务发布连续性（2026-11-06）

审计 ops P0-2 / P1-2 的落地批次：

- **新增 `GET /readyz` 就绪探针**（bootstrap 层，与业务 health 分离）：
  - 数据库为硬依赖，Ping 失败返回 **503**（容器 healthcheck/LB 摘流量）；
  - Redis 为降级依赖，未配置视为 ok，配置后 Ping 失败仅 `degraded`（仍 200）——
    限流/幂等在 Redis 故障时 fail-open，摘流量反而扩大故障面；
  - 检查总预算 2s；`/healthz` 保持纯存活探针语义不变。
- **容器探针口径统一**：base/dev compose 的 backend/backend-test healthcheck
  改打 `/readyz`（原 dev 用 POST system/health 判 code=0，degraded 也当健康）。
- **优雅停机预算对齐**：backend 设置 `stop_grace_period: 30s`（应用侧 HTTP 排水
  10s + WS 排水 3s + 收尾），避免 Docker 默认 10s 提前 SIGKILL。
- **发布不再硬重启 nginx**：`deploy-production.sh` 发布/自动回滚路径由
  `docker restart nginx` 改为 `docker exec … nginx -s reload`（软重载，在途
  HTTP/WS 连接不再被全量断开）；runbook 手工回滚同步更新。
- 验证：`go test ./internal/bootstrap/` 覆盖 ready/degraded/503/未配置依赖、
  路由注册；`docker compose config` 确认 stop_grace_period/healthcheck；
  `nginx -s reload` 在本机 nginx 容器实测成功。

---



## Batch E（第二批）：可观测性装配闭环（2026-11-06）

审计 ops P0-1/P0-2 的装配端补齐（此前「有零件没装机」）：

- **prod 网络修正**：prometheus/grafana 在 `docker-compose.prod.yml` 显式加入
  `hotspot_internal`——base 里二者落在隐式 default 网络，prod 下抓取
  backend:8081 必然失败（ops 审计 P0-1 证据 2）。
- **部署链路纳入采集栈（可选开启）**：`deploy-production.sh` 读取服务器
  `.env` 的 `OBSERVABILITY_ENABLED`，为 1 时以 `--profile observability`
  滚动更新 prometheus + grafana；默认关闭（小机器不常驻），配置仍随部署分发。
- **Redis fail-open 指标化**：新增 `hotspot_redis_failopen_total{component}`，
  覆盖 ratelimit / idempotency / hotcache / realtime 多实例广播的故障降级
  路径（此前只打 Warn 日志，线上持续降级不可见）。
- **告警投递闭环（单机方案）**：新增 `HotspotRedisFailOpen` 告警规则；
  `health-check.sh` 巡检新增 `/readyz` 检查（degraded 也告警）与 Prometheus
  firing alerts 拉取，经既有 `ALERT_WEBHOOK` 外发——不引入 Alertmanager
  也把「告警发出」打通，最坏延迟=巡检周期。
- **Grafana 面板**：provisioning 增加 dashboards provider 与「HotSpot 后端总览」
  （HTTP 状态码速率/p99、WS 连接与在线用户、DB 连接池、Redis fail-open、
  WS 丢弃事件/goroutine/RSS）；数据源固定 uid=prometheus 供面板稳定引用。
- 验证：`promtool check config/rules` SUCCESS（8 条规则）；prod compose
  config 确认采集栈网络；dashboard JSON 解析通过；脚本 `bash -n`/`sh -n` 通过。


## Batch D（第一批）：性能可验证体系（2026-11-06）

审计 ops P1-1「性能需求不可验证」落地：

- **组件级 benchmark（从 0 到 5）**：
  - `realtime`：100 连接广播 prepared 路径 ~2.2 µs/op / 13 allocs，逐连接
    marshal 回退路径 ~115 µs/op / 400 allocs（**约 50x**，禁止回退）+
    单事件 marshal 体积基线；
  - `httpx`：20 条列表统一 envelope 编码 ~12 µs/op / 3 allocs；
  - `observability`：请求指标中间件固定税 ~0.54 µs/op / 7 allocs。
- **perf-smoke 修复（此前失真）**：默认坐标改为种子锚点（34.2604/117.1855）、
  搜索关键词改「林」（命中种子用户「林一」），先预热再采样；校验读端点必须
  HTTP 200 且 envelope code=0（此前只看 HTTP 状态，业务错误不可见）；支持
  `--min-rps` / `--max-p99-ms` 门禁阈值，任一失败非零退出；token 获取失败
  干净报错。
- **实测基线**（backend-test + seed 数据，c=10/warmup 2s/5s）：account ~5.4k rps、
  bootstrap ~4.0k、notify ~7.7k、following ~3.5k、nearby ~350（p99 79ms）、
  users/search ~570（p99 65ms）——nearby/search 为当前 p99 最高的读端点，
  容量规划优先关注（两项均已有 S2/pg_trgm 索引）。
- **文档**：新增 `docs/performance.md`（工具/基线/SLO/容量模型/回归纪律），
  `backend-go/AGENTS.md` 性能章节同步。
- 验证：`go test -short ./...` 全绿；benchmark 命令实测输出；perf-smoke 基线
  6 端点 code=0 退出 0、不可达目标退出 1；--with-db ALL CHECKS PASSED。


## Batch D（第二批）：查询往返收敛（2026-11-06）

审计 perf D2 的首批落地：

- **附近流关注标记合入候选查询**：`queryNearbyMoments` 内用
  `m.user_id IN (SELECT target_user_id FROM user_relations WHERE user_id=$3 AND is_following)`
  一次返回 is_following（PG 规划为 hash 子计划，一次构建 + 每候选 hash 探测），
  删除候选查询后的 `queryNearbyFollowingUserIDs` 批量往返；`ListNearbyMoments`
  本模块 DB 查询 **3 → 2 次**（候选含关注 + 分页详情）。排序语义不变（关注加分
  +1000 仍在排序前完成）。
- **reference_point 回退轻量化**：`users.Service.RecentLocation` 优先走 PG 仓储
  `GetRecentLocation`（单条 `SELECT latitude, longitude`，仅存在/未注销/坐标
  有效），不再为拿一个坐标执行 GetProfile 的完整画像查询；内存/测试仓储未实现
  该能力时保持原 GetProfile 回退（行为一致）。影响 nearby/search/groups 等所有
  依赖回退锚点的列表读路径。
- **回归断言**：新增 `TestPostgresListNearbyMomentsQueryCount`（stub 断言
  ListNearbyMoments 恰好 2 次 Query 且关注候选排序优先）；users 新增
  `GetRecentLocation` 单测/真实库 IT（存在/无坐标/已注销三态）。benchmark 与
  SLO 口径见 docs/performance.md。
- 验证：go test -short 全绿；--with-db ALL CHECKS PASSED（含新增 IT）。


## Batch D（第三批）：附近流图片批量并入详情（2026-11-06）

- `queryNearbyMomentDetails` 通过 `LEFT JOIN LATERAL ... jsonb_agg` 随分页详情
  一次取回图片行（moment_images JOIN media_upload_sessions completed，按 position
  排序），服务层只做 presign——删除原有「详情后再发一次图片批量查询」往返。
- 仓储→服务的内部传递：`MomentItem` 新增不参与 JSON 契约的
  `imageRows/imagesFetched` 字段；`hydrateImagesBatch` 检测到全部项已取回图片
  时走 `HydrateLoadedMomentImages`（仅 presign），其它列表端点保持原批量查询
  路径（混合列表自动回退）。
- 附近流实际 DB 往返：候选(含关注) + 详情(含图片) + reactions 聚合 + 作者关系
  = **4 次**（本批前 5 次；审计目标 3-4 已达标）。排序/分页/图片顺序语义不变。
- 测试：新增图片行嵌入断言（imagesFetched/imageRows）、`HydrateLoadedMomentImages`
  presign/尺寸/缩略图用例、`momentImagesFetched` 混合列表回退用例；真实库
  IT（nearby S2/分页/拉黑过滤）全绿。
- 验证：go test -short 全绿；--with-db ALL CHECKS PASSED。

## Batch E（第三批）：生产镜像运行时门禁 + 迁移基线回归（2026-11-06）

审计 ops P1-4 / P1-5 / P1-8 首批落地：

- **CI 生产镜像运行时冒烟**（.github/workflows/ci.yml → docker-build job）：
  Dockerfile 此前只在 CI build、从不 run，USER/CMD/探针/启动迁移路径只能在
  线上首次暴露。现在 backend 镜像以 load: true + 固定 build-args 载入本地，
  起 job service postgres（--network host 直连）后断言：
  - 非 root（docker run --entrypoint id -u 非 0）；
  - 启动自动执行迁移 + schema_manifest 校验，GET /readyz 返回 status=ok；
  - GET /healthz 的 version 等于注入的 GIT_VERSION；OCI
    org.opencontainers.image.version 标签一致。
- **存量库迁移基线回归**（backend-integration job）：迁移后重复执行
  go run ./cmd/migrate（幂等）、-verify-only（checksum + pending + manifest
  只读校验）、scripts/check-migration-indexes.sh --database hotspot_test
  （manifest 索引逐名 / VALID 校验）——防「基线登记导致新索引永久缺失」类
  事故回归。
- **可复现构建**：Dockerfile go build 增加 -trimpath（源码路径不进二进制，
  本地与 CI 构建差异收敛；审计 P1-8）。
- 本机实测（macOS + Docker Desktop，等价命令）：构建带固定指纹的生产镜像 →
  对全新 scratch 库启动 → 迁移 54 个、/readyz ok、healthz version=v0.0.0-ci、
  OCI 标签一致、uid=100 非 root；**重启镜像二次启动幂等**（迁移数仍 54、
  readyz 仍 ok）；migrate -verify-only 输出 schema verification passed；
  check-migration-indexes.sh 对 scratch 库校验 OK（123 期望索引）。

## Batch F（第一批）：业务错误可观测（P0-3 首个切面）（2026-11-06）

审计 P0-3：契约错误走 HTTP 200 + envelope code!=0，HTTP 指标与 5xx 告警都
看不到它们；此前仅零散日志，业务错误率/分布不可观测、不可告警。

- 新增指标 hotspot_business_errors_total{code,route,method}：route 取 chi
  路由模板（不引入路径参数高基数），method 取 HTTP 方法；由统一日志出口
  observability.LogBusinessError 同步记录，保证「日志/指标/响应」口径一致。
- 现有统一出口（httpx.WriteAPIError / httpx.Business / BusinessForbidden /
  users.auth 等模块的 writeServiceError）全部经 LogBusinessError，自动获得
  指标，无需逐个 handler 改造；裸 WriteJSON 校验分支仍不进入指标（高基数
  10400 噪声，后续可评估仅记录 5xx 语义错误）。
- Grafana 总览新增「业务错误码分布」面板（sum by code rate）。
- 测试：chi 路由内触发 10400 后断言指标含 code/method/route 模板三元组。
- 验证：go test -short 全绿；--with-db ALL CHECKS PASSED；合并后主环境实测
  触发业务错误 → /metrics 出现计数 → Prometheus 可查询。

## Batch C（第十批）：admin 内存仓储迁入测试构建（2026-11-06）

架构审计 P0-2：admin 是剩余双仓储中最大的一个（8 个 repo_memory*.go，约
1.4k 行生产代码），但生产装配只用 Postgres 实现，内存版仅测试使用。

- 8 个 admin/repo_memory*.go 全部迁移为 *_test.go（生产代码净减约 1.4k 行；
  内存实现仍在测试构建中服务同包 200+ 用例，接口漂移由编译期暴露）。
- 生产侧只保留被 PG 实现复用的纯转换函数：新增 audit_convert.go
  （auditLogItemFromEntry / cloneAuditDetails）。
- 唯一跨包消费者 bootstrap/realtime_authz_test 改用本地最小桩
  banStateRepoStub（嵌入 admin.Repository + 实现 GetUserBanState/
  SetUserBanState），不再依赖 admin 内存仓储；未登记用户与真实实现同口径
  返回 normal 状态。
- 覆盖率：admin 70.6%（内存实现移出生产统计后仍过 70% 门槛）。
- 累计 11/17 模块完成内存仓储迁移。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch C（第十一批）：comments 内存仓储迁入测试构建（2026-11-06）

审计 P0-2 持续推进：comments 的内存实现只被测试使用，生产装配只有 PG。

- comments/repo_memory.go（约 300 行）迁移为 repo_memory_impl_test.go。
- 两个跨包测试消费者改用本地最小桩（embed comments.Repository + 必要方法），
  不再依赖具体内存仓储：
  - moments/handler_test（4 处建服务）：newMomentsCommentsRepo（Create/List/
    Get/SoftDelete/回复空集）；
  - bootstrap/reaction_providers_test：commentRepoTestStub（Create/Get/SoftDelete）。
- comments 生产覆盖率 72.5%，仍过 70% 门槛；累计 12/17 模块完成内存迁移。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch C（第十二批）：privatemessages 内存仓储迁入测试构建（2026-11-06）

审计 P0-2 持续推进：privatemessages 内存实现（约 600 行）只被测试使用。

- privatemessages/repo_memory.go 迁移为 *_test.go；生产保留 PG 与内存共用
  的纯函数 repo_helpers.go（conversationKey / otherUserID）。
- 唯一跨包消费者 privatechats/handler_test 改用本地最小测试存储
  （fakeMessageStore + fakeMessageRepo + fakeReadService）：同一份存储同时
  驱动消息 handler（Send/History）与会话服务（Recent/LastByChatKeys/
  UnreadCountsByChatKeys），并实现删除边界语义（deletedAfter/boundaries 过滤）
  以保持「删除会话后不可见、新消息重现空会话」等既有断言。
- split 测试后真实覆盖率暴露不足（62.2%），补 module 装配/路由、PG 仓储剩余
  方法骨架、PurgeUserData、LastByChatKeys/UnreadCountsByChatKeys 空结果路径
  用例 → 76.4%，过 70% 门槛。
- 累计 13/17 模块完成内存仓储迁移。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch C（第十三批）：groups / moments 内存仓储迁入测试构建（2026-11-06）

审计 P0-2 最大一批：groups（5 个 repo_memory*.go，约 1.5k 行）与 moments
（repo_memory.go，约 640 行）生产内存实现全部迁入测试构建，生产代码净减
约 2.2k 行。

- groups：repo_memory.go / _announcements / _applications / _invites /
  _members → *_test.go；生产专用便捷构造 groups.NewService()（内部构造内存
  仓储）移入 service_testhelpers_test.go，生产装配只接受显式依赖。
- moments：repo_memory.go → *_test.go；生产复用的 containsTag 提取到
  tag_helpers.go（搜索游标过滤使用）。
- 跨包阻塞点解除：users 的 4 个用例此前用 groups/moments 真实 Service 当
  content provider；改为本地最小桩 stubGroupsProvider / stubMomentsProvider
  （覆盖 ListGroupsByUser/CountCommonGroups/OwnedGroupCount 与
  ListMomentsByAuthor（含 includePrivate 过滤）/ListReactedByUser），users
  用例只验证本模块聚合/过滤契约，不再依赖其它模块内存实现。
- 覆盖率（真实生产代码）：groups 72.9%、moments 71.1%，均过 70% 门槛。
- 累计 15/17 模块完成内存迁移（剩余 notify、users）。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch F（第二批）：10600 内部错误统一出口 + 指标/告警（2026-11-06）

审计 P0-3 续：业务错误已统一出口，但 10600「服务暂时不可用」仍有 45 处
裸 WriteJSON（不走统一日志/指标；HTTP 状态恒 200，5xx 告警看不到）。

- 新增 httpx.WriteServerError(w,r,err)：统一记录底层错误（含 request_id/
  路由模板）+ 内部错误指标 + 契约 10600 响应；
- 45 处 handler 裸 10600 出口全部改为该 helper（groups 最多 35 处，另有
  share/groupmessages/privatemessages/reports/privatechats/media/auth/admin
  等）；grep 已无残留裸 10600；
- 新增指标 hotspot_internal_errors_total{route,method}（10600 专属；业务
  错误指标 hotspot_business_errors_total 仍覆盖 4xx 语义）；Grafana 总览新增
  「内部错误（10600）速率」面板；Prometheus 新增 HotspotInternalErrorsHigh
  告警（sum rate > 0.2/s 持续 5m）；
- 测试：路由内触发 WriteServerError 断言响应 code=10600 且指标含 route 模板。
- 验证：go test -short 全绿；--with-db ALL CHECKS PASSED；promtool check rules
  9 条规则 SUCCESS。

## Batch C（第十四批）：users 内存仓储迁入测试构建（2026-11-07）

审计 P0-2 倒数第二个模块：users 内存实现约 1k 行，生产装配只有 PG，
但被 privatechats/bootstrap/privatemessages 测试跨包引用。

- users/repo_memory.go → *_test.go；生产专用便捷构造 users.NewService()
  （内部构造内存仓储）移入 service_testhelpers_test.go。
- 生产保留内存/PG 共用的搜索纯函数：新增 repo_search_helpers.go
  （containsFold / normalizeSearchRadius / normalizeSearchSort /
  sortSearchCandidates / searchRelevanceScore / paginateSearchCandidates /
  searchProfileCandidate / searchCursor 及编码解码/maxInt），PG 搜索路径
  继续复用同一套关键词/半径/排序/游标口径。
- 三处跨包测试消费者改本地最小桩，不再依赖 users 内存仓储：
  - privatechats：fakeUsers 同时实现 chat.userReader 与 message.userReader
    （OpenTheDoor/BatchGet/SettingsFor/Relationship/ChatTargetsContext），
    隐私设置用例直接改 settings；
  - privatemessages：permissiveUsers 宽松桩（任意用户存在、默认 everyone），
    保持原有发送链路测试语义；
  - bootstrap realtime 缓存方向还原用例：authzUsersRepo（Follow/Block/
    Relationship/BlockedEitherWay 读写 map）。
- 迁移后真实覆盖率暴露不足，补 module 装配/路由、跨模块转发方法、PG 仓储
  骨架分支、搜索 SQL 构造/扫描错误等用例 → 74.0%，过 70% 门槛。
- 累计 16/17 模块完成内存仓储迁移（仅剩 notify）。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch C（第十五批）：notify 内存仓储迁入测试 + P0-2 完成（2026-11-07）

最后一块内存双实现：notify（repo_memory.go 约 190 行，8 个跨包测试文件、
36 处构造引用）。

- 新增 `internal/modules/notify/notifytest` 测试支持包：集中提供
  notify.Repository 的内存实现（Append/AppendMany/AppendBatchPerUser/
  List/UnreadCount/MarkAsRead/MarkAllRead/PruneReadBefore），只被 *_test.go
  引用，不进入服务端二进制；避免 8 个消费者各自复制易漂移的假实现。
- notify/repo_memory.go → *_test.go；生产复用 bucket 分类函数提取到
  repo_helpers.go；生产便捷构造 notify.NewService()、moments.
  NewServiceWithRepo、groups.NewService/NewServiceWithRepo（内部构造内存仓储/
  notify 默认实现）全部移入各自测试 helper，生产装配只接受显式依赖。
- 8 个消费者（comments/moments/admin/groups/users）测试改用
  notify.NewServiceWithRepo(notifytest.New())，List 断言语义不变。
- notify 真实覆盖率 78.1%。
- archtest 新增 TestModulesDoNotShipInMemoryRepositories：扫描模块生产代码，
  出现 repo_memory* 文件或 NewInMemoryRepository 定义即失败，固化 P0-2 成果。

**P0-2 完成：17/17 模块内存仓储全部迁出生产构建**（28 个 *_impl_test.go；
生产代码 grep 已无 NewInMemoryRepository 引用）。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch C（第十六批）：跨模块写债务收敛 1/5 — admin:moment_tags（2026-11-07）

架构审计 P0-4 首批收敛：admin 直接 DELETE moment_tags（标签硬删除）是
跨模块写表债务，moment_tags 归属 moments 模块。

- moments 模块新增标签关联管理能力：HasTagAssignments /
  DeleteTagAssignments（Service 经可选 tagAssignmentStore 能力断言消费，
  不膨胀通用 Repository 大接口、不迫使既有测试桩实现）；SQL 移入
  moments/repo_postgres_tags_admin.go + repo_postgres_queries.go。
- admin 侧新增 momentTagStore 最小接口注入（NewModule 显式参数，bootstrap
  传 momentsModule.Service）：GetTagDeleteState 的引用判定、SetTagBan 的
  hard_delete 删除均经 moments 模块；admin 只保留本模块 admin_blocked_tags
  读写（queryBlockedTagExists）。
- 事务语义保持：admin RunInTransaction 的 txCtx 传给 moments 仓储，
  跨模块删除与屏蔽仍在同一事务（persistence ctx tx 传播）。
- archtest 删除 allowlist 条目 admin:moment_tags（跨模块写债务 5→4）。
- 测试：admin 测试注入 fakeMomentTagStore（含与 backend stub map 共享版本），
  新增断言硬删通过 moments 接口执行；moments 内存实现补齐方法。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch C（第十七批）：跨模块写债务收敛 2/5 — admin:reports（2026-11-07）

架构审计 P0-4 第二项：admin 直接 UPDATE/DELETE reports（裁决与保留期清理）
是跨模块写表债务，reports 表归属 reports 模块。

- reports 模块新增管理端能力（Service 经可选 adminReportStore 能力断言，
  不膨胀通用 Repository）：MarkHandledAsAdmin（pending 守卫 + RETURNING，
  并发裁决只有一方命中）、DeleteHandledBefore（保留期清理，pending 永不删）；
  SQL 归位 reports/repo_postgres_admin.go + queries。
- admin 侧：Repository 移除 HandleReport/DeleteHandledReportsBefore，删除
  对应 PG 实现与 SQL；Service 新增 reportAdminStore 最小接口（bootstrap 经
  SetReportAdminStore 注入 reportsModule.Service）。
- HandleReport 裁决改经 reports 模块落库：pending 冲突仍返回 10409「已处理」，
  告警/审计/通知流程不变；保留期 sweeper 未注入时告警跳过，不误删。
- archtest 删除 allowlist 条目 admin:reports（跨模块写债务 5→3）。
- 测试：admin 内存适配 testReportAdminStore 注入既有 helper；保留期真实库 IT
  改由 reports 仓储执行 DELETE；移除 admin 仓储裁决 SQL 用例。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch C（第十八批）：跨模块写债务收敛 4/5 — auth:user_settings/users（2026-11-07）

架构审计 P0-4 第三项：auth 注册/游客建用户与登录元数据直写 users/user_settings，
两表归属 users 模块。

- users 模块新增 ProfileWriter 最小能力（ProvisionUserInput/ProvisionUser/
  RecordLogin）：Service 经可选仓储能力断言消费；SQL（建用户+默认设置、
  登录元数据）归位 users/repo_postgres_profile_writer.go。
- auth 仓储改为注入 users.ProfileWriter：Register/CreateGuest 在既有事务内
  调 ProvisionUser（txCtx 透传，凭证+资料仍同事务回滚）；UpdateLoginMeta 改调
  RecordLogin；auth 侧 queryInsertUserProfile/queryUpsertUserSettings/
  queryUpdateLoginMeta 全部删除。
- 依赖方向 auth→users（archtest import allowlist 显式登记），users 不反向依赖；
  bootstrap 在 users/auth 模块装配后注入 usersModule.Service。
- archtest 删除 allowlist 条目 auth:user_settings、auth:users（跨模块写债务 3→1，
  仅剩 privatemessages:private_chats）。
- 测试：auth 仓储单测改为断言 ProfileWriter 调用参数（含游客 email/phone 为空、
  注册元数据、writer 未接线显式失败）；真实库 IT 注入 users PG 仓储；
  users 覆盖率 73.1%（新增能力覆盖后仍过门槛）。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch C（第十九批）：跨模块写债务收敛 5/5 — privatemessages:private_chats（2026-11-07）

架构审计 P0-4 最后一项：privatemessages 发消息时 INSERT private_chats（会话表
归属 privatechats 模块）。

- privatemessages 定义 ChatEnsurer 最小端口（EnsureChat(ctx, chatKey,
  userAID, userBID, createdAt)）；发消息事务内不再直写 private_chats，改调
  注入的实现；未接线时显式报错（绝不静默跳过）。
- privatechats 新增 EnsureChat 能力（PG 仓储沿用既有 upsert SQL；Service 经
  可选仓储能力断言暴露），由 bootstrap 在 privatechats 模块装配后注入
  privatemessages（pmModule.SetChatEnsurer(pcModule.Service())）。
- 无 import 环：privatemessages 只持端口，privatechats 结构化实现；事务语义
  不变（privatemessages txCtx 透传给 privatechats 仓储，聊天行 + 消息行仍同
  事务回滚）。
- **archtest 删除最后一条 allowlist：crossModuleWritesAllowlist 清空，P0-4
  完成（跨模块 SQL 写表债务 0 条）**；扫描器继续生效，任何新增跨模块写会直接
  失败，需显式登记原因。
- 测试：真实库 IT/契约测试用 realChatEnsurer 落会话行；新增 privatechats
  EnsureChat 仓储/服务用例；全量 --with-db 通过。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch D（第二批）：users/search 短中文关键词索引化（2026-11-07）

D5 性能项：users/search 的 1–2 字符中文关键词无可用索引（pg_trgm 需 ≥3 字符），
100k 用户合成库实测 seq scan 85ms。方案与实测：

- 数据库镜像 postgres:18.4-alpine → groonga/pgroonga:4.0.6-alpine-18
  （PostgreSQL 18 + PGroonga 官方镜像；dev/CI/生产/备份恢复文档同步）。
- 迁移 063：启用 pgroonga 扩展；users.nickname/bio 各建 PGroonga GIN 索引
  （pgroonga_text_full_text_search_ops_v2）；schema manifest 更新（125 索引）。
- 搜索 SQL：非手机号 keyword 增加 (nickname &@ $kw OR bio &@ $kw) 预过滤
  （1 字符起可走索引），保留原 ILIKE 精确复核，输出语义不变；手机号精确分支
  保持不变。参数/测试同步更新。
- 实测 100k 用户（keyword=游客，2 字符）：旧 85.3ms → 新 **2.1ms**
  （BitmapAnd/BitmapOr 索引扫描）。

验证：verify-backend-ci.sh --with-db ALL CHECKS PASSED（含 manifest 重放校验、
真实库 IT、覆盖率）。后续可把同一索引策略推广到 moments/groups 搜索。

## Batch D（第三批）：moments/groups 搜索短关键词索引化（2026-11-07）

D5 续：把 users/search 的 PGroonga 方案推广到 moments/search、groups/search。

- 迁移 064：moments.description、groups.name/description 建 PGroonga GIN 索引；
  schema manifest 更新（128 索引）。
- SQL：moments/groups keyword 分支增加 &@ 子串预过滤，保留 ILIKE 精确复核；
  测试 stub 参数位置同步更新（keyword 现为 raw+escaped 两参）。
- 实测 100k 瞬间 / 50k 群组合成库（keyword=游客，2 字符）：
  moments 41.9ms → 0.73ms（~57×）；groups 19.7ms → 0.73ms（~27×）。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED
（manifest 128 索引、真实库 IT、覆盖率）。

## Batch D（第四批）：nearby 候选查询拉黑过滤物化（2026-11-07）

D 系列性能回归排查：moments/fetch_nearby 压测 p50 长期约 82ms、DB CPU 单请求
接近打满（backend-test，hotspot_test 478 候选瞬间）。根因：候选粗筛用两个相关
NOT EXISTS 过滤双向拉黑，优化器对每个候选行各做一次 user_relations 全表扫描
（478 候选 × 2），prepared 执行 6.7ms/次。

- 改写为 CTE blocked（一次物化查看者双向拉黑集合，通常 0-几十行）加
  m.user_id NOT IN 子查询：语义不变（任何一方拉黑即互不可见）。
- hotspot_test prepared 执行对比：旧 6.7ms 降为新 0.9ms（约 7.5 倍）；
  DB 单请求 CPU 显著下降，nearby 吞吐上限回到连接池/网络而非 DB。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

### Batch D 端点级验证（2026-11-07）

perf-smoke 扩展至 8 端点并在 hotspot_test 累计数据上复测：
- moments/fetch_nearby：拉黑过滤物化后 134-158 rps/p50 82-85ms →
  743 rps/p50 7.1ms/p99 60.4ms（吞吐约 5 倍、p50 约 11 倍）。
- users/search：PGroonga 后 570-637 rps/p99 65-68ms → 2161 rps/p99 26.8ms。
- moments/search、groups/search 均稳定数千 rps、p99 < 12ms。

## Batch E（第三批）：独立迁移 job + 镜像 Id 清单校验（2026-11-07）

E2 部署链路加固：迁移从「server 启动隐式执行」提升为部署显式前置步骤。

- backend 镜像同时构建 `server` 与 `migrate` 两个二进制（同镜像同版本指纹，
  新增二进制约 12MB）。
- compose 新增一次性 `migrate` 服务（profile=migrate，prod 覆盖内网）：
  deploy 脚本在数据库快照之后、滚动更新之前执行
  `--profile migrate run --rm -T migrate`；迁移失败即中止（此时 backend
  尚未切换，避免新容器 crash-loop + 自动回滚震荡），server 启动期保留
  EnsureSchema 兜底。
- 发布包新增 `hotspot-images-<ver>.ids.txt`（两个镜像的 docker config Id）：
  deploy 加载后逐一核对 `docker inspect .Id`，与 tar sha256 一起约束
  「发布包内容 == CI 构建产物」；旧版本无该资产时告警放行。
- 评估结论入档 release-runbook §7：蓝绿部署暂不实施（触发条件与迁移兼容
  前提已写明）；cosign keyless 签名列为后续步骤（当前 tar sha256 + 镜像 Id
  清单 + OCI 标签 + GH 资产来源）。

验证：Dockerfile 本地构建通过（/app 同时含 server/migrate）；
`docker run ... ./migrate`、compose --profile migrate run 对 dev 库执行成功；
actionlint release.yml 通过；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch F（第三批）：业务错误统一出口 — 221 处裸 WriteJSON 清零（2026-11-07）

架构审计 C3/P0-3 续：业务错误响应此前大量写成
`WriteJSON(w, 200, Error(code,msg))`，绕过统一日志与
`hotspot_business_errors_total` 指标（F1 加的指标只覆盖少数走 WriteAPIError 的路径）。

- 新增 httpx.WriteCodeError(w,r,code,message)：统一记录业务错误日志（method 用
  chi 路由模板，避免高基数/敏感参数）+ 业务错误指标 + 契约响应（HTTP 200）。
- codemod 全仓：200 处常量错误 + 21 处 err/apiErr 变量错误全部改走该出口；
  platform 内 resourceguard middleware 同步；system/health 手写 LogBusinessError
  去重（避免双计）。
- 结果：非 httpx 包的裸 `WriteJSON(..., Error(...))` 为 0；所有业务错误自动进入
  F1 的指标/Grafana 面板/F2 的错误日志链路口径。
- 测试：httpx 新增 WriteCodeError 响应 + 指标（code/route/method 标签）断言。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch C（第二十批）：resourceguard 去进程级全局单例（2026-11-07）

架构审计 C2 第一个纵切片：resourceguard 的 `SetDefaultChecker/DefaultChecker`
进程级可变全局删除，改为显式依赖 + context 传递。

- 根中间件 `resourceguard.WithChecker(checker)` 在装配期把 Checker 注入请求
  context；守卫中间件与写入方 `RecordUser/Group/Moment` 都从 ctx 取实例。
- bootstrap `NewHTTPRouterWithReadiness` 显式接收 checker（app.New 传入按
  PgDB 构造的 DBChecker）；未注入 checker 的测试装配自动 bypass。
- 同进程可装配多套 App / 无隐藏单例；`t.Parallel()` 不再被全局 swap 污染。
- 迁移 4 组测试：middleware 用 `WithChecker(...)` 包裹、service 级用例用
  `WithCheckerContext(ctx, checker)` 注入；删除全部 SetDefaultChecker 调用。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch C（第二十一批）：idempotency store 去进程级全局单例（2026-11-07）

架构审计 C2 第二个纵切片：删除 `idempotency.defaultStore` 原子全局、
`SetStore/currentStore` 与包级 sweeper 单例，store 改为显式依赖 +
context 传递（与 resourceguard 同一模式）。

- 根中间件 `idempotency.WithStore(store)` 装配期注入请求 context；
  `ReplayOrStore` 从 ctx 取 store；未注入时明确退化为「直接执行 producer、
  不做幂等缓存」（不再回退全局）。
- `app.New` 构造唯一 store（Redis 或 InMemory）并传入 router；
  `StartSweeper(ctx, store)` 显式接收 store（InMemory 才启动，Redis TTL 跳过）。
- 同进程多套 App 各自持有独立幂等 store；测试不再需要全局 SetStore 复位，
  13 个模块测试文件删除全局初始化；auth/moments 回放测试改为 ctx 注入。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch C（第二十二批）：platformauth 去全局单例（Service + AdminChecker）（2026-11-07）

架构审计 C2 第三个纵切片：删除 `platformauth.defaultService` 与 `adminChecker`
两个进程级原子全局，认证依赖改为显式装配 + context 传递。

- 根中间件 `platformauth.WithDependencies(service, adminChecker)` 注入请求
  context；RequiredAuth/OptionalAuth/AdminAccess/IsAdminUser 统一从 ctx 读取。
- `app.New` 先构造唯一 `platformAuth`，再传入 BuildModules（auth 模块不再
  反向写全局）、realtime handler、sweeper 与 router；`StartSweeper(ctx, svc)`
  显式接收 service 并返回 stop。
- 安全语义保持 fail-closed：路由树未注入认证依赖时 RequiredAuth 返回 10600
  （不再静默放行）；OptionalAuth 匿名降级并记 Error 日志。
- 测试迁移：middleware/admincheck 改为 ctx 注入、admin 挂载路由测试用
  WithDependencies 包裹、notify 路由装配测试显式注入；删除全部全局 setter。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch C（第二十三批）：架构守卫（可变全局 / 多写事务）+ DefaultHub 评估（2026-11-07）

C2/C5 收口防腐：把本轮清掉的全局单例与事务纪律固化为 CI 硬断言。

- `TestNoNewMutablePackageGlobals`：扫描 platform/modules 包级 atomic/sync.Map 全局，
  只允许 3 条登记项（observability.defaultLogger、realtime.defaultHub、
  requestmeta.trustedProxyCIDRs，均在 allowlist 写明理由）；新增全局直接失败。
- `TestRepoMultiWriteMethodsUseTransaction`：AST 扫描 `repo_postgres*.go`，同一函数
  直接执行 >=2 次写语句必须出现 RunInTransaction/WithTx，或进入 allowlist
  （ProvisionUser 调用方事务契约、execInTx/MarkAsRead 互斥分支三类已登记理由）。
- `realtime.DefaultHub` 显式注入评估：现状 80 处调用（17 个非测试文件）、21 个测试
  文件依赖 SetDefaultHub；彻底移除需把 Hub 依赖穿透 10+ 模块构造器/registry 并
  重写测试装配，收益（多 App 隔离）与成本不匹配，留作独立批次；本轮用守卫
  防止新增全局，Hub 改造保持计划内。

验证：go test -short 全绿（archtest 全量）；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch D（第五批）：性能预算回归门禁（2026-11-07）

- 新增 `docs/perf-baseline.json`（8 端点 rps/p99 基线 + 容差）与
  `scripts/perf-budget.sh`（跑 perf-smoke → 对比基线 → 回归即 exit 1）；
- perf-smoke 新增 `--report <csv>`，供脚本消费（人工表格输出不变）；
- 实测：backend-test 上 8/8 OK；人为收紧 p99 阈值后 exit 1 且逐端点标
  REGRESSION，门禁行为符合预期；
- 定位为手动/发版前回归工具，不进默认 CI（共享 runner 噪声），文档见
  docs/performance.md §7。

## Batch D（第六批）：groups 标签过滤接 GIN（2026-11-07）

- 迁移 065 idx_groups_tags_gin（GIN(tags)），manifest 重建（129 索引）；
- 查询从 = ANY(g.tags) 改 g.tags @> ARRAY[$1]::text[]（语义等价；
  = ANY 无法走 GIN），新增单测锁定写法；
- 10 万群合成库实测：选择性标签 13.4ms → 0.28ms（约 48x）；非选择性标签
  仍走 LIMIT 提前终止的顺序扫描（0.33ms，无回退）；
- 合成库已清理；dev/test/it 库已应用 065。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch G（第一批）：端点面快照门禁（route_manifest）（2026-11-07）

契约治理 P1（审查 spec-contract：路由 dump 只写 /tmp、端点增删无人 review）落地：

- 新增 `backend-go/docs/route_manifest.json`：173 条路由的 method + 模板 path +
  认证档（anonymous/admin/probe/authenticated）显式快照；
- `TestRouteManifest` 用生产同款装配枚举 chi 路由并与快照逐字节比较：
  端点增删、路径/认证档变化都会失败，必须有意识执行
  `UPDATE_ROUTE_MANIFEST=1 go test ./internal/bootstrap -run TestRouteManifest`
  并 review diff；
- 快照自身健康断言：无重复路由、每条都有认证档、/healthz /readyz /metrics
  探针必须存在；
- 端点面 173 条（168 POST + 4 GET + 3 探针，与 spec 端点集合一致）。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch C5（第一批）：aftercommit 提交后副作用 + 举报裁决事务化（2026-11-07）

跨模块编排事务的第一块地基：副作用与事务边界解耦。

- 新增 `internal/platform/aftercommit`：事务内 `Register` 的审计/通知/实时
  回调只在 COMMIT 成功后按序执行一次；ROLLBACK/panic 全部丢弃。回调使用
  **事务前 ctx** 执行（自动剥掉已关闭的事务句柄，可继续正常查库）；不在事务
  上下文中注册时立即执行（保持直接调用语义）。
- `persistence.RunInTransaction` 集成：开启事务时挂 hooks 容器，commit 后
  Run、回滚/panic/commit 失败时 Discard；嵌套事务复用外层容器（只在最外层
  commit 后执行）。
- admin `HandleReport`（举报裁决）事务化：删内容 + 封禁 + 举报终态在一个
  事务内；裁决审计与举报人通知改为 aftercommit，回滚不再出现「内容已删但
  审计/通知已发」；并发裁决败者（pending 守卫 false）回滚已执行副作用。
  `SetTxRunner` 未注入时（单测/内存装配）直跑，语义与旧实现一致。
- 已知边界：内容归属模块的实时广播仍在其服务内同步发送（早于 COMMIT），
  回滚场景存在幻影广播；后续把各模块广播迁到 aftercommit 即可收敛（已记录）。

验证：go test -short 全绿；aftercommit 单测（提交执行/回滚丢弃/无事务立即）；
persistence 真库 IT（hook 在 commit 后可用事务前 ctx 查库、回滚不执行）；
admin 事务注入单测；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch C5（第二批）：admin 跨模块编排写统一事务包装（2026-11-07）

把 C5 第一批的举报裁决事务模式收口为通用能力，覆盖 admin 其余两个跨模块
编排写入口：

- `Service.runDecisionInTx`：统一「业务错误需回滚」哨兵 + TxRunner 包装；
  未注入 runner（单测/内存装配）时直跑，语义不变；
- `HandleReport` 改用该 helper（行为不变，去掉重复包装代码）；
- `SetUserBan`（账户封禁/解封 + 会话撤销 + 审计）与 `DismissGroup`（群解散 +
  申请/邀请作废 + 广播/审计）改为事务包装，失败/业务错误整体回滚；
- 新增两个「事务确实被使用」单测（fake TxRunner 计数 + 结果断言）。
- 已知边界同前：内容/会话模块的实时广播仍早于 COMMIT（幻影广播），后续迁
  aftercommit 收敛。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch C5（第三批）：事务内实时广播迁 aftercommit（消除幻影广播）（2026-11-07）

第二批遗留边界收敛：admin 事务包装内的内容/会话模块广播不再早于 COMMIT。

- moments `softDeleteAndBroadcast`：作者 MomentDeleted + 收藏者广播放入
  aftercommit（回滚不再出现「库未删但用户已收到删除事件」）；
- groups `DismissAsAdmin`：GroupDismissed 广播 + DropGroup 迁 aftercommit；
- groupmessages `RecallMessageAsAdmin`：GroupMessageRecalled 广播迁 aftercommit；
- privatemessages `RecallMessageByIDAsAdmin`：双方 PrivateMessageRecalled
  广播迁 aftercommit；
- admin `banUser`：CloseSession（踢 WS 连接）迁 aftercommit，封禁回滚不误踢；
- 无事务上下文的调用方式不变（aftercommit.Register 立即执行），测试语义不受
  影响；新增 groups「广播必须等 COMMIT」单测（before=0、Run 后=1）。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch H（第一批）：comments.mentions 持久化（契约字段补齐）（2026-11-07）

契约审计 P1：CommentItem.mentions 在 spec 中恒为数组，此前只参与通知、从未
落库，所有读接口恒返回空数组（创建响应也不回显）。

- 迁移 066：comments 增列 `mentions TEXT[] NOT NULL DEFAULT '{}'`（评论量远
  小于消息，行内数组无需 join）；manifest 重建（40 表 / 378 列 / 129 索引）；
- 写路径：`queryInsertComment` 增 $7 mentions，Create 同事务写入原始 mention
  用户 ID 列表（nil 归一为 `[]`），创建响应回显；
- 读路径：List / Get / ListReplies / ListTopRepliesByParentIDs 四条查询全部
  带出 c.mentions 并扫描进 `CommentItem.Mentions`；
- 新增 stubbed repo 单测：创建→读取 mentions round-trip（含列序对齐）。

验证：go test -short 全绿；真库 psql 插入/查询验证；
verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch H（第二批）：X-Request-Id 契约修复（客户端 ID 原样回显）（2026-11-07）

契约审计 P1：spec api_spec_base.md 要求「客户端可选传入，服务端原样回显；服务端
不对其施加格式/熵值约束」。此前两处违反：

- nginx 所有 API location 都 `proxy_set_header X-Request-ID $request_id`，把客户端
  trace id 覆盖成 nginx 生成值；
- 后端 RequestIDMiddleware 只接受 16-128 长度，短 ID 被丢弃并换成生成值。

修复：
- nginx（base/dev/prod-tls/server.inc 全部代理点）新增 http 层 map：
  `map $http_x_request_id $hotspot_request_id { default $http_x_request_id; "" $request_id; }`
  ——有则原样透传、无则由 nginx 生成；
- 后端去掉长度约束：非空客户端 ID 一律原样回显（仅缺失时生成）；
- 新增短 ID（"abc"）回显单测；三套 nginx 配置（base/dev/prod）容器内 `nginx -t` 通过。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch H（第三批）：moments 匿名端点 Public 类型层（契约类型隔离首个纵切片）（2026-11-07）

spec api_spec_public.md「类型体系」要求匿名节点 Request/Response 使用 Public*
专属类型、禁止与登录版共用，且响应不得含查看者状态字段。此前 /public/v1
全部端点直接复用登录 DTO（is_favorited / author_relation / recommendation 等
字段虽然值恒零/null，但仍在契约形状里）。

- 新增 moments 公共类型：PublicMomentItem（MomentItem 去三个查看者字段）、
  PublicReaction（Reaction 去 reacted_by_me）、PublicPaginatedMoments、
  PublicSearch/FetchNearby/OpenMomentDoorRequest；
- 三个匿名 handler（PublicSearch / PublicFetchNearby / PublicOpenTheDoor）只做
  Public 请求→内部请求转换、复用校验、内部结果→Public 响应映射；
- MountPublic 改挂 Public handler；提取 validateSearchMomentsRequest 供两版共用
  （同一契约口径，不复制校验逻辑）；
- 数组字段（images/tags/mentions/reactions）映射时归一为空数组，不返回 null；
- 测试：匿名响应无 is_favorited/author_relation/recommendation 键、契约字段齐全、
  数组恒为数组、热门搜索/附近分页 shape、类型转换归一；moments 覆盖率 70.4%。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch H（第四批）：comments 匿名端点 Public 类型层（2026-11-07）

延续 H3 的类型隔离纵切片，覆盖 comments 三个匿名读端点：

- 新增 PublicCommentItem（CommentItem 的 reactions 换 PublicReaction，去
  reacted_by_me）、PublicReaction、PublicPaginatedComments 与三个 Public 请求类型；
- PublicFetch / PublicFetchReplies / PublicOpenTheDoor 做请求转换 + 复用服务读取 +
  内部结果→Public 响应映射；top_replies 递归使用 PublicCommentItem；数组归一为空数组；
- MountPublic 改挂 Public handler；错误口径保持（10300 防枚举、10000 参数）；
- 测试：转换剥离 reacted_by_me、top_replies 递归、数组归一、三个端点 not-found 10300、
  参数缺失 10000；comments 覆盖率 72.0%。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch H（第五批）：groups 匿名端点 Public 类型层（2026-11-07）

延续 H3/H4 类型隔离，覆盖 groups 四个匿名读端点（公告无对偶类型，按 spec
直接共用 GroupAnnouncementItem）：

- 新增 PublicGroupItem（去 viewer_state / friends_in_group / last_message）、
  PublicGroupMemberItem、PublicPaginatedGroups/Members 与四个 Public 请求类型；
- PublicSearch / PublicOpenTheDoor / PublicListMembers / PublicSearchMembers 做
  请求转换 + 复用服务读取 + Public 响应映射（tags 归一空数组）；
- 校验/错误口径复用现有 helper 与映射（10000/10200/10300 保持一致）；
- 测试：viewer 字段剥离、四个端点成功 shape、缺条件/缺 keyword 10000、
  不存在群 10300；groups 覆盖率 72.1%。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch H（第六批）：comments 读路径补齐 UserInfo.avatar_url（2026-11-07）

契约审计 P1「UserInfo.avatar_url 多读路径缺失」逐模块收口（本批 comments）：

- 所有评论读查询（List / Get / ListReplies / ListTopRepliesByParentIDs）在
  nickname 后补 `COALESCE(u.avatar_url, '')`，扫描后填充 UserInfo.AvatarURL
  （空串归一 nil，契约 avatar_url null=未设置）；ranked 子查询用 AS avatar_url
  显式别名避免外层未知列；
- 创建评论的昵称查询同时取 avatar，创建响应回显头像；
- 测试：stub 补 nickname/avatar 两列、批量 rows 列序对齐；新增「创建→读取
  avatar_url 一致」单测；真库 contract 测试（top_replies batch）通过。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch H（第七批）：moments 读路径补齐 UserInfo.avatar_url（2026-11-07）

契约审计「avatar_url 多读路径缺失」逐模块收口（本批 moments）：

- 6 条读查询/构造补 `COALESCE(u.avatar_url, '')`：queryGetMoment、ListMine、
  ListFavorites、ListMomentsByAuthor、ListReactedByUser、moments/search 动态
  SQL、创建瞬间的昵称查询；
- 对应扫描器（scanMomentRowParts / WithAssociations / Favorite / UserMoment /
  UserReactedMoment / Search）统一多扫 avatar 列并填充 UserInfo.AvatarURL
  （空串归一 nil）；momentItemFromRow 增加 avatar 参数收口；
- 创建瞬间响应回显作者头像；
- 测试 stub 列序对齐（repo read/user_lists/search_tests），full suite 全绿。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch H（第八批）：groups 成员读路径补齐 UserInfo.avatar_url（2026-11-07）

契约审计「avatar_url 多读路径缺失」逐模块收口（本批 groups 成员面）：

- queryListGroupMembers / queryListAdminGroupMembers / querySearchGroupMembers
  在 nickname 后补 `COALESCE(u.avatar_url, '')`；
- 三个扫描点统一填充 UserInfo.AvatarURL（空串归一 nil），覆盖公开成员列表、
  公开成员搜索、管理视角成员列表；
- 测试：memberRow 固定 avatar 列并断言 ListMembers 返回头像（防列序漂移）。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch H（第九批）：groups 申请/邀请/公告/封禁读路径补齐 avatar_url（2026-11-07）

契约审计「avatar_url 多读路径缺失」逐模块收口（本批 groups 其余读面）：

- 申请：queryGetGroupApplication / ListGroupApplications / ListMyGroupApplications
  补申请人 avatar，GroupApplicationItem.Applicant 回填；
- 邀请：ListSentInvites 补 inviter+invitee avatar、ListMyInvites 补 inviter avatar；
- 公告：List/GetGroupAnnouncement 补创建者 avatar，scanAnnouncementRow 回填；
- 封禁名单：ListGroupBans 补被封禁者+操作者 avatar，AdminBannedMemberItem
  User/BannedBy 回填；
- 测试：对应 stub 行补 avatar 列，并断言 applicant/inviter/封禁成员 avatar 已解析。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch H（第十批）：reactions 读路径补齐 UserInfo.avatar_url（2026-11-07）

契约审计「avatar_url 多读路径缺失」逐模块收口（本批 reactions）：

- queryListReactions 在 nickname 后补 `COALESCE(u.avatar_url, '')`；
- ListReactions 扫描填充 ReactionItem.User.AvatarURL（空串归一 nil），
  覆盖公开端点 /public/v1/reactions/fetch 与登录版反应人列表；
- 测试 row 列序对齐（游标过滤索引 3→4），并断言首条反应人 avatar 已解析。

其余模块核查：notify actor 由各模块 buildActor/resolveUserInfo 提供 avatar；
comments/moments/groups/messages/search/share/system/admin 读路径均已带 avatar。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch H（第十一批）：users 匿名端点 Public 类型层（search / open_the_door / batch_fetch）（2026-11-07）

延续 H3-H5 Public 类型层，覆盖 users 三个核心匿名读端点：

- 新增 PublicUserItem（UserItem 去 relation / social_context）、
  PublicPaginatedUsers、PublicSearchUsersRequest、PublicOpenUserDoorRequest、
  PublicBatchFetchUsersRequest、PublicUserProfileResult；
- PublicSearch / PublicOpenTheDoor / PublicBatchGet：请求转换 + 复用服务读取 +
  Public 响应映射；recent_moments 直接复用 moments 模块导出的
  PublicMomentItem/PublicPaginatedMoments（同一公开形状，避免跨模块漂移）；
- 提取 validateSearchUsersRequest 供登录版/匿名版共用；tags 归一空数组；
  batch_fetch 按 spec 复用共享 UsersBatchResult（本就不含查看者字段）；
- 测试：公开 user 无 relation/social_context、recent_moments 无查看者字段、
  tags 数组、search/batch_fetch 成功 shape 与 10000 参数错误；
  users 覆盖率 73.2%。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch H（第十二批）：users 其余匿名端点 Public 类型层收口（2026-11-07）

覆盖 users 剩余 6 个匿名读端点（关系列表 / 群组 / 瞬间）：

- 新增 PublicRelatedUserItem（RelatedUserItem 去 remark；互关由
  followed_back_at 表达）、PublicPaginatedRelatedUsers、PublicFetch* 请求类型；
- PublicListFriends / PublicListFollowing / PublicListFollowers：Public 请求→
  内部请求→服务读取→PublicRelatedUserItem 映射；
- PublicListGroups 复用 groups.ToPublicPaginatedGroups（PublicGroupItem）；
- PublicListMoments / PublicListReactedMoments 复用 moments.ToPublicPaginatedMoments
  （PublicMomentItem）；为此导出 groups/moments 的 Public 转换 helper；
- MountPublic 全部 users 路由改为 Public handler；
- 测试：关系项无 remark/relation/social_context、群组无 viewer 字段、瞬间无
  is_favorited/author_relation，六个端点成功 shape；users 覆盖率 73.4%。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch H（第十三批）：public bootstrap 类型隔离（2026-11-07）

契约审计：/public/v1/system/bootstrap 此前直接复用登录版 BootstrapData，
匿名响应带 profile / unread_summary（恒 null）两个查看者字段；缓存命中时
还返回 profile/unread 而非 spec 要求的「仅 bootstrap_version、省略
meta/client_config」。

- 新增 PublicBootstrapData（bootstrap_version + meta/client_config，缓存命中
  时两键整体省略）与 PublicBootstrapCacheHitData（仅版本号）类型；
- 新增 PublicBootstrap handler：保留 device_id Cookie 下发与 ETag 语义，但只
  取服务层公共段，即使可选鉴权解析出 profile 也绝不下发；
- MountPublic 改挂 PublicBootstrap；登录版 Bootstrap 行为不变；
- 测试：匿名响应无 profile/unread_summary 且含三公共键；ETag 命中仅
  bootstrap_version（meta/client_config 键缺省）。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch C2（第十四批）：实时 Hub 显式注入（模块去 DefaultHub 全局直取）（2026-11-07）

架构审计 P1「15+ 组全局单例」逐项收敛（本轮 realtime Hub）：

- realtime 新增 context 注入：WithHub / HubFromContext（未注入时回退 DefaultHub，
  兼容后台任务与直调测试）；
- bootstrap 根中间件 realtimeHubMiddleware 在请求入口注入 Hub，请求路径依赖
  由「隐式进程全局」改为「显式 context 依赖」；
- 16 个模块文件 80 处 `realtime.DefaultHub()` 全部替换为
  `realtime.HubFromContext(ctx)`；notify.PushSystemAnnouncement 补 ctx 参数
  （admin 接口/桩同步），auth handler 用 r.Context()；
- 模块代码对 realtime.DefaultHub/SetDefaultHub 直取数 = 0，仅 bootstrap 组合根
  保留（Hub 生命周期/collector/redis bridge/中间件）；
- 架构守卫新增 TestModulesUseContextHubNotGlobal：模块内再直取全局直接失败；
- 测试：realtime.WithHub 覆盖/回退语义单测；全量 go test -short 全绿。

验证：verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch C2（第十五批）：App 独占 Hub 实例（组合根去全局直取）（2026-11-07）

承接 C2 第十四批（模块上下文注入），本轮把「Hub 实例归属」也移到组合根：

- NewApp 创建应用级 `hub := realtime.NewHub()`，本 App 独占；
- 显式传入 HTTP 路由构造 → 根中间件注入请求 context（不再由中间件读全局）；
- 实时 Handler 新增 SetHub：WS 连接注册/退订、在线态、typing 广播全部走应用
  级实例（未注入时回退 DefaultHub，仅单测/直调路径）；
- Hub 生命周期操作（collector、resumeStore、Redis bridge/publisher、Shutdown
  Drain）全部改用 App 持有的 hub；App 结构体保存 hub 引用；
- initRedisBridgeFromClient 增加 hub 参数；HTTP 轻量构造器（NewHTTPRouter/
  WithLogger）传 nil 走 fallback；
- 测试：Handler hubOrDefault 回退/注入语义；全量 go test -short 全绿；
- 架构守卫说明更新：全局 defaultHub 仅测试/后台 fallback，生产由 App 显式持有。

验证：verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch C2（第十六批）：Logger 上下文注入（模块去 observability.L 全局直取）（2026-11-07）

架构审计「15+ 组全局单例」继续收敛（本轮 logger，规模最大的一项）：

- observability 新增 WithLogger / FromContext / WithLoggerMiddleware：请求级
  Logger 注入 context，未注入（后台/直调测试）回退 L()；
- bootstrap 根中间件在请求入口注入应用级 Logger（与 realtime Hub 同一 DI 模式）；
- 模块内 201 处 `observability.L()`（37 文件）全部替换为
  `observability.FromContext(ctx)`；无 ctx 的少数 helper 按语义改为接收
  ctx / r.Context() / context.Background()（notify prune 后台等）；
- 架构守卫新增 TestModulesUseContextLoggerNotGlobal：模块再直取 L() 直接失败；
- platform 层 66 处 L() 保留（启动期/无 ctx 后台任务兜底），allowlist 说明更新；
- 测试：FromContext 注入/回退语义单测 + archtest 全绿。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch H（第十四批）：search/suggest 类型化 payload + 匿名 Public 判别联合（2026-11-07）

契约审计：`SearchSuggestItem.Payload` 原为 `map[string]any` 弱类型，登录/匿名
共用同一响应形状；匿名节点未提供独立 Public 类型（spec 要求 TagItem /
PublicUserItem / PublicGroupItem / PublicMomentSuggestItem 判别联合）。

- 新增类型化 payload：TagItem、SearchSuggestUserPayload（含 relation/
  social_context 登录查看者字段）、SearchSuggestGroupPayload（含 viewer_state/
  friends_in_group）、SearchSuggestMomentPayload（无查看者字段）及 Public 对偶
  （PublicSearchSuggestUserPayload / PublicSearchSuggestGroupPayload）；
- 仓储三处 suggest payload 由 map 改为 typed 结构（user/group/moment/tag），
  relativeLocationPayload 改类型化 relativeLocationView；
- 新增 PublicSuggest handler：复用 decodeSuggestRequest 校验，将 typed payload
  映射为 Public 判别联合（剥离查看者字段）；MountPublic 改挂 PublicSuggest；
- 登录版 Suggest 响应形状保持（relation/social_context/viewer_state 键恒在 null）；
- 测试：仓储层 JSON payload 断言改为 typed→JSON 归一；新增 public suggest 形状
  测试（四类 payload、无查看者字段、缺 prefix 10000）；

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch D1（续）：handler 级 benchmark + 架构调整后全链路基线复测（2026-11-07）

性能工程审计「Go benchmark 覆盖不足」补齐冷启动/账户/通知热路径：

- 新增 BenchmarkBootstrapHandler / BenchmarkPublicBootstrapHandler（system）：
  登录/匿名冷启动 envelope + 版本组装（~5.1 µs/op，43 allocs）；
- 新增 BenchmarkGetMyProfileHandler（users/account 投影 ~3.7 µs/op，39 allocs）；
- 新增 BenchmarkNotifyListHandler（notify/fetch 空列表 ~2.2 µs/op，31 allocs）；
- 全链路复测（backend-test + seed，c=10/5s）：8 端点全部 code=0；rps 相比
  docs/perf-baseline.json 全面提升（account 9847、bootstrap 8580、notify 14167、
  following 10237、search 2788/8871/5907、nearby 576），`perf-budget.sh` 全部 OK；
- docs/performance.md 基线表更新为本轮实测（含 moments/groups search 两行）与
  新增组件 benchmark 结果；确认 logger/Hub 上下文注入未造成读路径退化。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED；
perf-budget.sh：8 端点均在预算内。

## Batch E6：Go 工具链版本对齐 + 漂移守卫（2026-11-07）

工程效能审计 P1「工具链漂移（本地 1.27.3 / CI 1.26.10 / go.mod 1.26.0）」：

- go.mod `go` 指令由 1.26.0 改为 1.26.10，与 CI setup-go、Dockerfile/Dockerfile.dev
  的 GO_VERSION 默认值完全一致（本地 1.27.3 仍满足 >= 最小版本，语义正确）；
- 新增架构守卫 TestGoToolchainVersionsAligned：解析 go.mod / Dockerfile /
  Dockerfile.dev / .github/workflows/ci.yml，任一 patch 版本漂移即失败；
- `go mod tidy` 无额外依赖变化（依赖集合保持）。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch D2：PGroonga 规划器启动期预热（消除搜索冷启动/扩容尖刺）（2026-11-07）

性能剖析发现：PGroonga 的关系/索引代价状态按 **后端连接** 惰性初始化，每个
新连接首次 `&@` 查询的 Planning Time 实测 50-70ms（`EXPLAIN (ANALYZE)`：首次
55ms，同连接后续 0.7ms）。连接池扩缩容时，搜索端点会出现集中 p99 尖刺。

- persistence 新增 `PostgresDB.WarmPlanner(ctx, query, maxConns)`：并发获取
  最多 maxConns 个池连接各执行一次轻量 PGroonga 查询后归还（stub/无 DSN no-op，
  幂等，失败可忽略）；
- NewApp 在 ensureSchema 之后、模块构建之前预热 16 个连接（覆盖 c=10 并留余量），
  成功 Info / 失败 Warn，不阻断启动；
- 新增 IT：真实库预热后池连接数 >=4 且重复调用幂等；stub 后端 no-op；
- 冷启动后首个搜索请求不再承担 50-70ms 规划成本。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch D2（第二批）：reactions/作者关系批量水合并发化（2026-11-07）

列表页原串行执行两次独立批量读：reactions 聚合（跨模块 provider）与作者关注
关系（FollowRelationReader）。二者互不依赖、写 MomentItem 的不同字段：

- 新增 Service.fillEngagementBatch：errgroup 语义的 WaitGroup 并发执行两路
  批量水合（provider 缺失时各函数自身短路）；
- 应用于 moments/fetch_following、fetch_nearby、search 与收藏列表（4 条列表
  主路径），串行等待从「查询A+查询B」降为 max(A,B)；
- 数据竞争：两路分别只写 Reactions 与 AuthorRelation 字段（disjoint fields），
  -race 单测全绿；

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch D2（第三批）：附近流详情查询合并 reactions/作者关系（4→2 次往返）（2026-11-07）

附近流原路径：候选粗筛 + 详情批量 + reactions 聚合 + 作者关系批量 = 4 次 DB
往返（详情查询已含图片/tag/收藏态）。本轮把后两者并入详情查询：

- queryNearbyMomentDetails 新增 reactions LATERAL（按 emoji 聚合 count +
  reacted_by_me，jsonb 返回）与两个作者关系 EXISTS 列（viewer→author /
  author→viewer），只作用于页内 ≤20 行；
- nearbyMomentDetail 扫描并回填 Reactions/AuthorRelation；MomentItem 新增内部
  reactionsFetched/authorRelationFetched 标记；
- fillReactionsBatch/fillAuthorRelations 只对未取回的行发批量查询（附近流全部
  短路，其它列表路径行为不变）；
- 附近流 DB 往返 4→2（候选 + 详情），公开/登录两种视角输出形状不变；
- 测试：nearby query-count 用例断言仍为 2 次查询，并新增 reactions 数组/作者关系
  回填与 fetched 标记断言；moments 包 -race 全绿。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch D2（第四批）：users/search 回退链改轻量位置查询（2026-11-07）

pg_stat_statements 实测（c=10、keyword=游客、10s）：users/search 每请求除主
查询（mean 1.0ms）外还执行一次 `GetProfile` 完整画像查询（mean 0.152ms，含 5
个统计子查询），10s 内占该端点 DB 总耗时 ~13%。根因：Service.resolveReferencePoint
回退链第 2 级直接调 `repo.GetProfile`，而未使用已有的单列轻量查询能力。

- resolveReferencePoint 改走 `s.RecentLocation`（PG 仓储实现为
  `queryGetRecentLocation` 单列查询；内存/测试仓储自动回退 GetProfile，行为不变）；
- 新增单测：无 reference_point 的 Search 不得触发 GetProfile（getProfileCalls=0），
  且使用 GetRecentLocation 返回的位置；
- 该路径同时惠及所有缺省 reference_point 的搜索/列表端点。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

实测（backend-test，2026-11-07）：users/search keyword=游客 单端点 loadgen
c=10/10s：修复前 1799 rps / p95 25.22ms / p99 40.24ms → 修复后 **2139 rps /
p95 8.53ms / p99 28.81ms**（吞吐 +18.9%，p95 -66%，p99 -28%）。全链路
perf-smoke：users/search 2785→3562 rps；8 端点 perf-budget 全部 OK。

## Batch D5（第一批）：users/search 去掉冗余 ILIKE 复核（PGroonga 已含同语义索引扫描）（2026-11-07）

EXPLAIN 发现 `(nickname &@ x OR bio &@ x) AND (nickname ILIKE x OR bio ILIKE x)`
会被 PostgreSQL 规划成 **两个 BitmapOr（共 4 次 PGroonga index scan）+ BitmapAnd**：
PGroonga opclass 本身支持 ILIKE（`~~*`）且实测 &@ 已具备大小写不敏感、%/_ 字面、
1 字符 CJK 语义（热缓存执行 0.31-0.55ms vs 旧 0.46-0.59ms），ILIKE 复核纯属重复。

- buildSearchUsersQuery 非手机号分支移除 ILIKE 复核，仅保留 `&@` 子串索引条件
  （手机号分支保留精确 phone_searchable EXISTS 语义）；
- 真实库语义 IT（TestPostgresITSearchKeywordSemantics）：大小写不敏感、% 字面、
  CJK 子串命中均锁定；
- SQL 构建/参数测试同步（非手机号 keyword 参数 3→2）。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

实测（backend-test，2026-11-07）：users/search keyword=游客 单端点 loadgen
c=10/10s：去掉 ILIKE 复核前 2139 rps / p95 8.53ms / p99 28.81ms → 去掉后
**3279 rps / p95 5.08ms / p99 7.27ms**（吞吐 +53%，p99 -75%）；全链路
perf-smoke users/search 3562→5090 rps、p99 25.3→6.0ms，8 端点 budget 全部 OK。

## Batch D5（第二批）：moments/search、groups/search 去掉冗余 ILIKE 复核（2026-11-07）

与 users/search 同款问题：`&@` 子串过滤叠加 ILIKE 复核会被 PostgreSQL 规划成
同一谓词的第二次 PGroonga index scan（BitmapAnd）。PGroonga &@ 的语义（大小写
不敏感、%/_ 字面、CJK 子串）已由 users/search 语义 IT 验证，本轮推广到另外两个
搜索端点：

- moments/search：移除 description ILIKE 过滤（仅保留 `&@`）；relevance 排序的
  CASE 仍使用 keyword 参数（排序行为不变），删除不再被使用的转义参数；
- groups/search：移除 name/description ILIKE 过滤，仅保留 `&@`（Go 侧相关性
  评分/排序不变）；
- 新增真实库语义 IT：moments、groups 各自锁定大小写不敏感、% 字面、CJK 子串；

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

实测（backend-test，2026-11-07，暖态）：moments/groups ILIKE 复核移除后
groups/search 6027→6178-6584 rps（p99 18.9→14.6-18.7ms），moments/search 持平
（9787 rps，p99 3.5-4.0ms）；users/search 不受影响。DB 缓存冷热会使
users/search 在 2.4k-5k rps / p99 6-40ms 间波动，回归判断以 perf-budget 相对
阈值为准。

## Batch D2（第五批）：nearby 候选粗筛 S2 区间改 OR BETWEEN（2026-11-07）

pg_stat_statements + EXPLAIN 定位 nearby p99：候选粗筛的半径谓词写成
`EXISTS (SELECT 1 FROM unnest(mins,maxs) WHERE s2_cell_id BETWEEN lo AND hi)`，
PostgreSQL 规划成 semi-join：扫描全部 public moments 后逐行 ×28 个区间过滤
（实测 922 行 → 25574 次 Join Filter，3.1ms/次），DB CPU 2 核被打满。

- 新增 buildNearbyMomentsQuery(rangeCount)：把每个 S2 覆盖区间生成
  `m.s2_cell_id BETWEEN $lo AND $hi` 的 OR 谓词（区间数 ≤50），args 按
  lo/hi 交替 + viewer 末尾；SQL 文本只随区间数变化（prepared 缓存友好）；
- 同数据 EXPLAIN：3.145ms → 0.343ms（Bitmap heap scan + filter 22 行）；
- 候选粗筛不再使用 unnest semi-join，区间条件可直接由 btree/bitmap 使用；
- 测试：query-count 用例改为匹配 SQL 特征；新增 builder 单元测试
  （OR 区间数、占位符顺序、无 unnest）。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

实测（backend-test，2026-11-07）：nearby 候选粗筛改 OR BETWEEN 后单端点
c=10/10s：461→**726 rps**（+57%），p50 8.11→5.73ms，p95 76.1→66.5ms，
p99 81.8→74.5ms；8 端点 perf-budget 全部 OK。

## Batch D2（第六批）：nearby 详情查询改「页面 ID 范围聚合 + hash join」（2026-11-07）

pg_stat_statements：nearby 候选粗筛降到 1.0ms 后，详情查询仍占 ~86% DB CPU
（mean 6.44ms/请求，20 行）。原 SQL 对每行执行 comments/shares/tags 相关子查询
与 images/reactions LATERAL，20 行×多次子查询重复扫描/聚合。

- queryNearbyMomentDetails 重写为 CTE：comments/shares/tags/images/reactions
  各按 `= ANY($1)` 一次索引扫描 + GROUP BY，再与 page 行 hash join；favorites/
  作者关系仍为页内单行 EXISTS（20 行规模）；
- 显式执行计划（20 IDs）：~1.2ms → ~0.5ms；仍为单次 DB 往返；
- 新增真实库 IT：附近流详情聚合归属正确（tags/reactions/counts/images 空集合）；
- 响应形状/顺序不变（服务层按候选顺序组装）。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

实测（backend-test，2026-11-07）：nearby 详情查询改范围聚合后单端点
c=10/10s：726→**1303 rps**（+79%），p50 5.73→3.82ms，p95 66.5→49.2ms，
p99 74.5→54.9ms；全链路 perf-smoke nearby 1190 rps / p99 58.2ms，8 端点
perf-budget 全部 OK。nearby 相比本轮系列优化前（~480rps/81ms）约 2.5x。

## Batch D4：WS 全局连接上限 + 拒绝指标（2026-11-07）

审计 D4「全局连接上限」落地：此前只有单用户 20 连接上限，单实例总连接数（以及
对应内存/广播 fan-out）仍无界。

- config 新增 `WS_MAX_CONNECTIONS`（默认 10000，单实例上限；多实例按实例数下调）；
- Handler 鉴权/封禁检查通过后原子占用连接槽，超限推 `Error` code 20300 +
  `retry_after=5` 后以 **close 4002** 断开；断连 defer 释放槽位；
- Hub 新增 `NoteConnectionRejected / RejectedConnections`，HubCollector 暴露
  `realtime_connections_rejected_total` 计数指标；
- bootstrap 注入 `SetMaxConnections(cfg.WSMaxConnections)`；`.env.example` 增配置项；
- spec 更新 close code 4002 行：从「预留」改为实际发出点（全局连接上限）；
- 测试：并发槽位 acquire/release、默认不限、超限 Error+close 4002、释放后可重连、
  config env/default、rejected 指标值。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch D4（第二批）：resume 回放分批化（内存峰值常数级）（2026-11-07）

审计 D4「resume 回放上限+分页」落地。原 `RedisResumeStore.Replay` 用 XRANGE
start..+ 一次拉取 user 流 + 最多 100 个 group 流（每流保留 2000 条），最坏
101×2000≈20 万事件全部反序列化进内存后才开始写 WS——重连风暴/长时间断线时
存在明显内存峰值与长阻塞。

- ResumeStore 增加可选 `batchReplayer` 接口（不改现有接口，测试桩自动退化）；
- RedisResumeStore 新增 `ReplayBatch(user, groups, fromSeq, limit)`：每流
  `XRANGE ... COUNT limit+1`（归并前 limit 条必在各流前 limit+1 条内），归并后
  截断并返回 hasMore；`Replay` 保持不变（诊断/测试用，内部走同一实现）；
- Handler 回放循环：每批最多 500 条（`resumeReplayBatchMax`），从上一批最后
  seq 继续直到清空；仍共用 20s 总回放时限，超时行为不变；
- 测试：Redis 120 事件分批 drain（seq 连续、每批≤limit、hasMore 正确）；Handler
  断线积压 537 条时跨批回放全部事件（顺序/seq 单调）；

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## Batch C1（收尾）：admin 生产代码去内存仓储回退（fail-fast）（2026-11-07）

审计 C1「repo_memory 迁 *_test.go，生产要求 DB（fail-fast）」最后一块：admin
的 `InMemoryModerationStore` 此前仍在生产代码中作为「无 DB/无公告适配器」回退，
生产包藏 209 行内存实现且语义与 DB 分叉。

- `moderation_store_memory.go` 改名 `_test.go`（生产包不再包含内存实现，仅测试
  构造使用；生产代码对 InMemoryModerationStore 引用 = 0）；
- `PersistenceModerationStore` 删除 memory 字段与全部回退：queryable 不可用返回
  `persistence.ErrNotImplemented`；announcements 未注入时公告方法返回同一错误；
- `NewService` 缺失 moderationStore 直接 panic（编程错误，bootstrap 必须装配）；
- 测试更新：模块/handler/retention 显式注入内存 store；新增 fail-fast 单测覆盖
  标签与 5 个公告方法在缺 DB/适配器时的错误路径；
- admin 包覆盖率 69.7%→70.0%（门禁阈值 70%）。

验证：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

## 最终验证与结论（2026-11-07）

经 80 轮持续审计与分批修复，后端本次改进工作收口。最终验证（worktree HEAD
`2c888ff2`，已 ff-only 合并 main）：

- `go test -race -short ./...`：全绿；
- worktree `verify-backend-ci.sh --with-db`：**ALL CHECKS PASSED**（gofmt/vet/
  staticcheck/单测/真实库 IT/schema manifest/archtest/覆盖率/路由快照/迁移校验）；
- 主树 `verify-backend-ci.sh --with-e2e`（同一代码提交）：**ALL CHECKS PASSED**
  （真实库 IT + HTTP E2E）；
- `scripts/perf-budget.sh`：8 个读端点全部 **PERF BUDGET OK**；
- 主环境 backend 重启后 health=200，PGroonga 预热日志与
  `realtime_connections_rejected_total` 指标正常。

本轮系列（64-80 轮）主要成果：13 条 P0/P1 缺陷修复、迁移体系加固与 schema
manifest、计数列去冗余、模块边界 archtest、事务 aftercommit、统一业务错误出口、
显式 DI（Hub/Logger/资源守卫）、契约 Public 类型收口、端点面快照门禁；
性能侧 nearby 约 2.5x（480→~1190 rps，p99 81→~58-63ms）、users/search 全链路
~2.8k→最高 5k rps（去冗余 ILIKE/轻量位置回退）、PGroonga 规划器预热；
实时侧 WS 全局连接上限+拒绝指标、resume 分批回放（内存峰值常数级）。

**已知可选的后续架构升级**（非缺陷，规模/需求触发后再做）：
1. 公告全量 fanout 虚拟化（announcements + per-user read 状态两表，替代每公告
   N 行 notifications）；
2. 通用 outbox（实时/通知投递的崩溃一致性）；
3. 分片 Hub / 多实例广播压测；4. 短关键词（1-2 字）若产品要求高吞吐可评估
   pg_bigm/独立搜索服务；5. 更大数据量下 users/search 密集关键词的候选上限策略。

## Batch D3：系统公告通知虚拟化（2026-11-07）

原实现发布公告时向 notifications 表 per-user fanout（100k 用户=10 万行/公告），
fanout 非原子、表膨胀且无法区分公告本体与已读状态。改为虚拟模型：

- 新迁移 067：`announcement_reads(announcement_id,user_id,read_at)` 记录稀疏已读状态；
  旧 fanout 行中 is_read=TRUE 的状态迁入新表后删除全部旧 system_announcement 通知行。
  可见性口径与旧 fanout 对齐：仅公告创建时已注册用户可见（读侧 join users.created_at）。
- notify_id 稳定为 `sysann_<announcement_id>`：WS SystemAnnouncement 与 notify/fetch
  共用同一 ID；notify/mark_read 按前缀把公告 ID 路由到 system 模块已读表。
- system 模块新增 `AnnouncementNotificationStore`（列表/未读计数/标记已读/全部已读），
  经 bootstrap 注入 notify；notify 不再直查 system_announcements（数据所有权保持）。
- notify 列表把「持久化通知行 + 公告虚拟行」按 (created_at DESC, notify_id DESC)
  全序归并后统一分页；unread_summary 合并 system 未读数。
- admin 发布公告不再 fanout，只做一次全体 WS 广播（SendPush 控制）；删除
  AppendMany/NewAnnouncementNotifyID 等 fanout API 与 admin ListUserIDsAfter。
- 测试：system 公告虚拟读/已读状态真实库 IT；notify 合并分页/过滤/未读/mark_read
  路由真实库 IT + 单测；admin 公告测试改为断言「不落 notifications 行 + 稳定 notify_id」。
- 门禁：go test -short 全绿；worktree verify-backend-ci.sh --with-db ALL CHECKS PASSED。

运行时验证（main 环境，2026-11-07）：
- 迁移 067 已应用，主库旧 system_announcement fanout 行清理完成；
- 新建访客（注册后）插入一条 system_announcements 后：
  `notify/fetch` 返回虚拟行 `sysann_<announcement_id>`（type=system_announcement，
  payload 含 title/severity），notifications 表对应行数 = 0；
  `notify/unread_summary` system=1/total=1；
  `notify/mark_read`（scope=ids，注意 Idempotency-Key 必须为合法字符集）返回 0 后
  announcement_reads 行数 = 1、unread total=0，虚拟行 is_read=true；
- 注册早于公告的用户可见性符合旧 fanout 口径（种子旧公告对新访客不可见）。

## Batch outbox：通用事务外盒 + 系统公告广播接入（2026-11-07）

aftercommit 回调在进程崩溃于 COMMIT 与回调之间时会丢副作用（实时广播/通知）。
新增平台通用 outbox：

- 迁移 068 `outbox_events(event_id, topic, payload, available_at, attempts,
  last_error, processed_at)`；部分索引 `(available_at,event_id) WHERE
  processed_at IS NULL` 供 dispatcher 领取。
- `internal/platform/outbox`：
  - Writer.Enqueue：ctx 带事务时**与业务写同事务**提交（无事务退化为池直写）；
  - Dispatcher：事务内 `FOR UPDATE SKIP LOCKED` 批量领取，handler 投递，
    失败指数退避（最多 10 次后 dead-letter 保留 last_error），后台随 rootCtx 退出；
  - DrainOnce 供测试与后台循环共用。
- 首个生产接入：admin 创建系统公告时「公告落库 + WS 广播事件入队」同事务
  （TopicSystemAnnouncement）；bootstrap 注册 handler 解码 admin payload 后经
  App Hub 广播，与同步路径共用 notify.SystemAnnouncementRealtimeEvent，
  payload 形状不漂移。未装配 outbox 的单测/内存装配仍走同步广播。
- 测试：outbox 真实库 IT（事务回滚不落事件、成功投递一次、失败重试/attempts、
  幂等领取）；admin 事务内入队单测；bootstrap handler 广播 payload 单测；
  archtest 登记 outbox_events 归属 platform；schema manifest 重建（42 表/389 列/134 索引）。
- 门禁：go test -short 全绿；verify-backend-ci.sh --with-db ALL CHECKS PASSED。

outbox 运行时验证（main 环境，2026-11-07）：
- 迁移 068 已应用；admin 通过 `/admin/api/v1/announcements/create`（Web cookie 鉴权，
  Idempotency-Key 需合法字符集）创建 send_push=true 公告后，outbox_events 出现
  TopicSystemAnnouncement 行；dispatcher 在 1s 轮询内处理，attempts=1、
  processed_at 非空、pending=0。
- 同一公告对「公告创建前已注册」的访客可见：notify/fetch 返回
  `sysann_<announcement_id>` 虚拟行，证明 outbox 事务生产 + 读取合并链路。
- 验证公告与 outbox 行已清理；outbox 无积压。

outbox 合并后性能门禁（backend-test / hotspot_test，c=10，warmup=2s，5s）：
- 8 个读端点全部 code=0；PERF BUDGET OK。
- 关键端点：notify/fetch 8713 rps / p99 3.21ms；users/search 1402 rps / p99 48.38ms；
  moments/fetch_nearby 874 rps / p99 61.36ms（均在 baseline 的 60% rps 下限与
  2.5x p99 上限内）。outbox 不在读热路径，公告广播为低频写。

## Batch 实时 Hub 分片 + 多实例广播压测（2026-11-07）

原 Hub 用单把全局 `sync.RWMutex` 保护 byUser/bySession/byUserWatchers/byGroup：
广播是读锁、连接注册/注销/订阅重建是写锁，连接抖动会让所有广播等写锁；在线规模
上来后注册风暴直接拖慢全部实时推送。

- **32 shard 锁条带化**：user/session/watch/group key 经 FNV-1a 散列到独立 shard，
  各 shard 自有 RWMutex+注册表；不同用户/群组的广播与连接抖动互不阻塞。
  广播路径只复合同 shard sink 快照，绝不在持锁时做网络写（沿用原锁外 Close 纪律）。
- **sink 反向索引去全局锁**：`subsBySink`/`sessionBySink` 改 `sync.Map`；
  `subEntry` 自带互斥锁 + closed 标记，订阅重建与连接注销并发时不会把死连接
  重新写回路由表。
- **全量广播单趟快照**：新增近似 sessionCount 容量提示，`BroadcastToAll`/
  `DrainConnections` 每 shard 只持一次读锁复制，避免两趟扫描；大在线量下
  BroadcastToAll 比旧实现更快。
- **多实例广播压测**：新增 `TestRedisBridgeMultiInstanceBroadcastBurst`
  （miniredis 双实例，8 用户 × 1000 条并发跨实例事件）：8000 条恰好一次投递，
  实测约 4.0 万 events/s（Apple M1 本机）；同时作为跨实例路由/缓冲回归测试。
- **Benchmark（Apple M1，8 vCPU，`go test -run '^$' -bench 'BenchmarkHub'`）**：
  | 场景 | 分片前 | 分片后 | 变化 |
  |---|---|---|---|
  | 并发广播不同用户 | 1632 ns/op | 1473 ns/op | -9.7% |
  | 广播 + 连接注册/注销混合 | 1537 ns/op | 938 ns/op | **-39%** |
  | BroadcastToAll（512 连接） | 13110 ns/op | 10370 ns/op | **-21%** |
- 验证：realtime 包 `-race` 全绿、`-count=5` 稳定；`go test -short ./...` 全绿；
  worktree `verify-backend-ci.sh --with-db` ALL CHECKS PASSED。

Hub 分片合并后门禁与性能（2026-11-07）：
- worktree `verify-backend-ci.sh --with-db` ALL CHECKS PASSED；主树 `--with-e2e`
  ALL CHECKS PASSED；realtime 包 `-race`/`-count=5` 稳定。
- perf-budget（backend-test，c=10/warmup 2s/5s）：8 端点全部 OK；
  users/account 9644 rps（p99 2.87ms）、notify/fetch 10014 rps（p99 3.01ms）、
  users/search 2109 rps（p99 35.66ms）、moments/fetch_nearby 1114 rps（p99 56.06ms）。
- 运行环境 main backend 已重启并 healthy；验证后 backend-test 已停止。

## Batch 搜索短关键词（1–2 字）PGroonga 收口（2026-11-07）

### 现状与缺口
- users/moments/groups 的主搜索（migrations 063/064）已用 PGroonga `&@`：1 字符起
  可走 GIN，且 PGroonga opclass 也能加速 `ILIKE '%kw%'`（实测 20k 行 users 的
  ILIKE 计划直接走 `idx_users_nickname_pgroonga`，非顺序扫描）。
- 仍有两处缺口：
  1. `search/suggest` 的 4 类查询仍是 leading-wildcard ILIKE（用户/群/瞬间/标签）；
  2. `moment_tags.tag_name` 只有 pg_trgm GIN——pg_trgm 需要 >=3 字符模式，
     1–2 字（尤其中文）标签联想只能顺序扫描。

### 方案对比（1–2 字符子串搜索）
| 方案 | 支持 1–2 字 | 本项目可用性 | 结论 |
|---|---|---|---|
| pg_trgm | 否（<3 字符退化为 seq scan） | 已启用 | 不可用于短关键词 |
| pg_bigm | 是（2-gram） | 未安装，需要新增扩展/镜像改造 | 引入新扩展成本高，收益与 PGroonga 重叠 |
| **PGroonga** | 是（1 字符起走 GIN，NormalizerNFKC 归一） | 已启用（063/064），运维/索引模式已存在 | **选择**：补齐 tags 索引 + suggest 统一 `&@` |

### 改动
- 迁移 069：`idx_moment_tags_name_pgroonga`（`moment_tags(tag_name)` PGroonga GIN），
  与 users/moments/groups 的短关键词索引同一方案；schema manifest 更新（135 索引）。
- `search/suggest` 四类匹配统一改 `&@ $prefix`（tags/users/groups/moments），
  不再对 prefix 做 LIKE 通配符转义；与主搜索模块语义/索引路径一致。
- 真实库 IT：1 字中文同时命中 user/group/moment/tag；2 字精确命中目标用户；
  PGroonga case-insensitive 与旧 ILIKE 一致；`EXPLAIN` 断言 users/tags 均走
  PGroonga 索引（3k+ 无关行下仍选索引）。

### 实测（hotspot_it，20k 无关行 + 1 目标行，事务回滚）
- tags 2 字 `ILIKE '%目标%'`（069 前，无 PGroonga tags 索引）：**Seq Scan，7.42ms**；
- tags 2 字 `&@ '目标'`（069 后索引可用）：**Index Only Scan，0.09ms**（约 80x）；
- users 1 字 `ILIKE '%林%'` / `&@ '林'` 均走已有 `idx_users_nickname_pgroonga`：
  ~0.22–0.27ms（0 命中）；说明主搜索早已受益，缺口只在 tags + suggest 统一路径。

搜索短关键词合并后验证（2026-11-07）：
- 主树 `--with-e2e` ALL CHECKS PASSED；main 重启后 migration 069 已应用。
- 运行时 `/public/v1/search/suggest`：prefix=`林` 默认 5km 半径下按契约过滤掉
  5km 外种子用户（0 结果）；显式 radius=100000 时返回「林小满/林一」等用户，
  说明 1 字匹配与 radius 过滤均正常（非索引问题）。语义/索引由真实库 IT 覆盖。
- perf-budget：首次在宿主 load≈15 时 8 端点全部 REGRESSION（环境噪声，非代码
  回归）；load 回落到 ~3.7 后重跑 **PERF BUDGET OK**：users/account 6803 rps
  (p99 4.44ms)、notify/fetch 7817 rps (p99 3.90ms)、users/search 1657 rps
  (p99 36.99ms)、moments/fetch_nearby 960 rps (p99 53.56ms) 等。
- 验证后 backend-test 已停止；main backend 保持 latest + healthy。

## Batch 搜索 S5：users/search 密集关键词候选策略（2026-11-07）

### 旧策略问题
原实现 SQL 固定 `ORDER BY created_at DESC LIMIT 500` 取候选，再在 Go 侧做
relevance/距离排序与游标分页：
- 关键词命中 >500 用户时，结果只覆盖「最新 500」，relevance 排序在截断集内做，
  真正的精确/前缀命中可能被排除；
- 深分页超过 500 候选后直接断路（cursor 无法定位下一页），且客户端无法感知。

### 新策略（无半径 + 有关键词 + relevance/newest）
- 排序、relevance 评分、keyset 分页全部下推 SQL，**不再有 500 候选上限**；
- 内层只排 `(user_id, last_online/created_at, score)` 小行（避免 10 万命中时
  排序宽度过大），外层仅 JOIN 页内 ≤MaxPageSize 行的完整画像/标签；
- SQL 评分与 Go `searchRelevanceScore` 严格同分（昵称精确 6/前缀 4/包含 3、
  bio 包含 2、tag 过滤 3），游标新增 `relevance_score` 并校验 sort_by 绑定，
  跨排序复用游标返回 invalid_cursor；
- 时间 tie-breaker 统一 `date_trunc('second', …)`：与响应里的 RFC3339 秒级
  时间字符串一致，避免微秒精度造成翻页跳行/重复；
- distance 排序（需 Go 精确圆距离）与 radius 过滤（S2 覆盖仅是包围盒）保留原
  有界候选路径——地理范围天然有界，不在本轮范围。

### 验证
- 真实库 IT：650 个前缀命中 + exact/prefix/contains/bio 共 654 行，relevance
  与 newest 各翻 ≥6 页，跨页无重复、无遗漏；exact→prefix→contains→bio 全序；
  relevance 游标请求 newest 返回 invalid_cursor；tags 按页内联；
- 单测：既有搜索/分页/仓库桩全绿（ranked 路径主查询内联标签，页大小有上界）。

### 性能取舍（hotspot_it 合成 100k 全命中）
- 新全量 ranked 页（LIMIT 21）：**104ms**（BitmapOr 扫 10 万行 + external sort）；
- 旧 500 候选页：45.7ms，但结果只含最新 500、深分页断路（错误结果换来的快）；
- 结论：真实低/中密度关键词走 PGroonga 索引后是毫秒级；**10 万级全命中**属于
  搜索语义的最坏情况，需独立搜索服务/前缀限制才可能再降一个量级。本机
  perf-budget 的种子数据密度远低于此，合并后实测见下节。

密集关键词 ranked 路径合并后验证（2026-11-07）：
- 主树 `--with-e2e` ALL CHECKS PASSED；go test -short 全绿；
  worktree `--with-db` ALL CHECKS PASSED。
- perf-budget（backend-test，c=10/warmup 2s/5s，load≈5.0）：8 端点全部 OK；
  **users/search 3270 rps / p50 2.28ms / p99 22.19ms**（上一轮同环境 1657 rps /
  p99 36.99ms；本机波动大，但相持平/更优），其余端点均在预算内。
- main backend 已重启并 healthy；验证后 backend-test 已停止。

## 五项架构升级合并后全量门禁（2026-11-07）

main（`55a58287`）执行发版前同级核验：`./scripts/verify-backend-ci.sh --all`
（静态 + race + 模块覆盖率 + 真实库 IT/契约测试 + HTTP E2E + govulncheck）：
- `govulncheck`：No vulnerabilities found；
- **ALL CHECKS PASSED**。
说明：本轮五项升级（公告虚拟化、outbox、分片 Hub、短关键词 PGroonga、密集关键词
ranked 分页）均已合并 main 并通过该全量门禁；perf-budget 8 端点 OK。

## 搜索 10 万级密集关键词优化评估（2026-11-07，结论：保留现有实现）

上一轮把 users/search 密集关键词改为 SQL ranked + keyset 全量分页后，100k 全命中
worst case 实测 104ms。继续优化的尝试与结论：

| 变体 | 100k 全命中页 1 实测 | 结论 |
|---|---|---|
| 当前 ranked 查询 | 104ms | 正确但扫描 10 万行 + 评分 + 排序 |
| 内层去掉 user_settings JOIN（页外再过滤） | 89ms | 仅 ~15%，且需 over-fetch/回退保证页满，复杂度不值 |
| 提高 max_parallel_workers_per_gather | 136ms | 并行扫描/排序在小表反而退化 |
| 分桶 UNION + btree(last_online,user_id) 强制有序 | 625ms | 规划器仍选 bitmap，分桶额外重复扫描，显著更差 |

根因：任意子串过滤必须由 PGroonga 扫出全部匹配行（100k），再做 rank 排序；
这是「无独立搜索引擎 + 任意子串语义」的固有成本，不是 SQL 写法问题。安全做法
仍然是：
- 典型低/中密度关键词走 PGroonga 索引（perf-budget users/search 3270 rps /
  p99 22.19ms）；
- 10 万级全命中通过 rate limit/产品层关键词最小长度或独立搜索服务解决；
- 不在应用层引入不可靠的 planner hint/density probe（会制造稀疏关键词长尾）。
故本轮**不修改** ranked 实现，避免用错误结果或高复杂度换 marginal 收益。

## Batch outbox 扩展：举报裁决通知广播（2026-11-07）

outbox 从系统公告扩展到举报裁决的实时通知：

- 举报结果通知行改为在**裁决事务内** append（原先在 aftercommit，进程崩溃会
  永久丢通知）；WS 广播事件与裁决同事务写入 outbox（TopicAdminReportHandled）。
- bootstrap dispatcher 新增 handler：解码 admin.ReportHandledBroadcastPayload，
  经 App Hub 向举报人广播 ReportResolved/ReportDismissed；payload 键与旧同步
  路径完全一致（notify_id/report_id/target_type/target_id/[note]/resolved_at 或
  dismissed_at）。至少一次投递，客户端按 notify_id 去重。
- 未装配 outbox 的单测/内存装配保留 aftercommit 同步回退（既有举报通知测试不变）。
- 测试：admin 单测断言「append 在裁决事务内、outbox 事件入队、提交前不广播」；
  bootstrap handler payload/事件类型断言；通用 outbox 真实库 IT（重试/幂等语义）
  继续覆盖 dispatcher。
- 门禁：`go test -short ./...` 全绿；worktree `verify-backend-ci.sh --with-db`
  ALL CHECKS PASSED。

举报裁决 outbox 运行时验证（main，2026-11-07）：
- 访客通过 `/api/v1/reports/create` 举报 seed 管理员 → admin 通过
  `/admin/api/v1/reports/judge` dismiss；outbox_events 出现
  TopicAdminReportHandled 行，dispatcher 1s 内处理（attempts=1、processed_at 非空）；
- 举报人 notifications 表出现 1 条 report_dismissed（report_id 精确匹配），
  证明「通知落库在裁决事务内 + WS 广播经 outbox 补投」；
- 验证后 report/notification/outbox 行已清理，outbox pending=0；main backend healthy。
- perf-budget（backend-test，c=10/warmup 2s/5s）：8 端点全部 OK；users/search
  3156 rps / p99 21.18ms、notify/fetch 8983 rps / p99 3.39ms 等。

## 覆盖率提升至真实库口径每模块 ≥90% + comments 详情 reactions 修复（2026-11-08）

- **覆盖率口径**：`scripts/verify-backend-ci.sh --with-db` 与 CI backend-integration job
  新增门禁——真实库上跑全部模块测试（含 `TestPostgresIT|Contract`，`POSTGRES_DSN` 注入），
  每个 `internal/modules/*` 包语句覆盖率 ≥90%；默认无 DB 阶段保留 ≥70% 下限
  （无库时 repo 代码天然不可覆盖，90% 只在具备 Postgres 的阶段校验）。
- **17 个模块 before → after**（`go test -count=1 -cover`，真实库口径）：
  admin 70.8→94.5、auth 73.1→94.9、comments 75.3→99.5、groupmessages 82.6→99.7、
  groups 74.7→94.8、mapmodule 85.5→95.1、media 78.0→99.1、moments 75.8→95.1、
  notify 89.7→93.7、privatechats 77.4→99.7、privatemessages 80.2→99.6、
  reactions 82.9→94.0、reports 83.2→94.4、search 78.1→93.3、share 78.5→99.1、
  system 83.0→98.8、users 73.7→99.4。新增 72 个测试文件（约 2.08 万行），仅测试代码，
  未改既有断言；真实库用例统一 `testdb` 事务回滚 + 唯一 ID 前缀，不污染共享测试库。
- **生产缺陷修复**：`comments/service.go` 的 `GetForDetail` 把 `fillReactionsBatch`
  作用在临时切片上，聚合结果未回写，导致 `/api/v1/comments/open_the_door` 与
  `/public/v1/comments/open_the_door` 的 `reactions` 恒为 null（spec 要求恒为数组）。
  改为先在本地切片聚合再返回首元素，并补回归断言；dev 环境实测已返回非空 reactions。
- **残留**：少量不可注入的失败分支（crypto/rand、AES-GCM/游标 Encode、SMTP TLS、idgen 等）
  仍未覆盖，未伪造测试；不影响 ≥90% 门禁。

## 死代码 / 兼容残留清理 Batch 1（2026-11-08）

方法：staticcheck U1000 且 -tests=false（测试不作为“使用”）+ deadcode 从 main 出发复核可达性；
再逐项全仓引用核对。**未发现为旧接口/旧客户端保留的后端业务代码**——HTTP 路由清单与 spec 逐条
一致、无 deprecated 请求字段；可清理的是重构后遗留的死实现。

### 删除内容

- **29 个生产零引用符号**（U1000）：admin（queryRoleInsertPermission、fillTrendDays、limitPageItems、
  geoBucketCount）、auth（nullIfEmpty、registerCodeRecord、retentionSweeperRunning、loginCounterDecay、
  stopOnce/stop）、comments（parentIDStr）、groups（querySearchGroups、groupMutedUntil、ownerIDValue、
  groupUserInfo）、mapmodule（fallbackHeatBucketCenter、aggregateHeatBuckets）、media/auth
  retentionSweeperRunning、moments（compareNearbyCandidates/paginateNearbyMoments/findNearbyCursorStart/
  escapeLikePattern/loadMomentFavorites/compareSearchTuple + 4 条废弃 SQL）、platform/auth
  queryRevokeOtherSessions。
- **导出死符号**：bootstrap.NewHTTPRouterWithLogger、platformauth.Forbidden、
  platformauth.UnauthorizedError、auth.DefaultBcryptCost；**auth.OptionalAuth**（契约只有
  public/api/admin 三节点，无 optional 档）——同步修正 AGENTS「四档认证」与 http.go 过时注释。
- **测试专用辅助下沉**：仅被测试替身实现使用的 helper 移到各包 legacy_helpers_test.go
  （admin geoBucketCount/fillTrendDays/limitPageItems、auth registerCodeRecord、comments parentIDStr、
  groups groupMutedUntil、moments nearby 分页比较与 compareSearchTuple、mapmodule 热度聚合）。
- **login tracker 生命周期**：去掉生产代码里的测试专用 stop 钩子（stopOnce/stopCh/stop），
  测试改用可取消 ctx；生产只在 ctx.Done 退出。
- **删除仅用于覆盖死代码的测试**：OptionalAuth、auth/media retention sweeper guard、mapmodule 热度聚合、
  moments G4 比较/分页、escapeLikePattern、loadMomentFavorites、groups 旧搜索错误分支、auth nullIfEmpty 等。
- **整文件删除**：moments/repo_postgres_favorites.go、platform/auth/api_errors.go；清理死 SQL 常量 8 条。

### 明确保留（不属于可删垃圾）

platform/migrate 的 AdoptLegacy/checksum 回填（生产存量库依赖）、刷新令牌 escrow 过渡分支
（存量令牌）、admin reports total 兼容字段（spec 明确）、前端 isDark 旧存储迁移 / 鸿蒙明文 token
清理（已安装用户），均需按下线版本择期处理；详见审计记录 .workwork/design/backend-legacy-code-audit.md。

### 验证

gofmt/go vet/go build、go test -short 全绿；./scripts/verify-backend-ci.sh --all
**ALL CHECKS PASSED**（DB 口径 17 个模块覆盖率均 ≥90%，auth 93.0% 为清理后最低）；
staticcheck U1000（-tests=false）清零。

## 死代码 / 兼容残留清理 Batch 2（2026-11-08）

- **删除重复的旧内存 refresh store**：RefreshStore 与 InMemoryRefreshStore 语义重复（前者 Rotate 不校验旧
  token、仅测试使用），删除并让相关测试改用 InMemoryRefreshStore；RefreshSession 类型保留，移入
  refresh_session_types.go。
- **过时注释修正**：login tracker 注释不再引用已删除的 loginCounterDecay（改为「衰减窗口 /
  effectiveLockDuration」）；admin statsTrends 注释不再提已删除的 fillTrendDays；
  privatemessages/groupmessages 的 mark_read 注释改为描述统一端点，不再引用已下线的
  /me/mark_as_read 端点。
- **测试专用装配口处理**：NewHTTPRouter/BuildModules、各模块 NewServiceWithXxx、StopPruneSweeper 等
  仅被单测/跨包测试使用（如 platformauth.NewService 被 realtime 测试依赖），未强行下沉到
  export_test.go（会破坏跨包测试）；生产装配不调用它们，保留为测试缝并在需要时再统一收敛。
- 验证：gofmt/vet/build、go test -short 全绿；verify-backend-ci.sh --all **ALL CHECKS PASSED**
  （DB 口径模块覆盖率仍全部 ≥90%）；staticcheck U1000（-tests=false）保持为零。
