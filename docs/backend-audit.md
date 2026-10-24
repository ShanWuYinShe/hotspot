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
