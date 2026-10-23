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
**遗留依赖**：`spec/api_spec_admin.md` 的 `app_config/update` 未定义 auth section，
按「spec 为契约唯一来源」纪律本轮不改管理端接口——后端已支持 DB AppConfig 读取，
后续在 spec 增补该 section 后接管理页即可（需产品确认默认值与可调范围）。

**遗留观察项**（不紧急，记录在案）：
- 契约测试基建已就位，但仅 privatechats 接入；后续对 notify/groups/users 等高频
  漂移模块逐步补契约用例（优先覆盖分页/错误映射/状态流转三类）；
- 双实现对比缺口示例：privatechats 内存实现无法复现「最近消息时间」排序（消息在
  privatemessages 模块，内存实现无跨模块存储），后续若引入共享内存后端可解决；
- **未修的已知限制**（需产品/架构决策，非缺陷）：
  1. auth 的验证码 TTL、登录锁定阈值已支持 AppConfig 读取，但管理后台入口依赖
     spec 增补 auth section（见上）；
- P2 剩余项（各域报告里未修的约定/架构类）与 OBS 项存档于 `.workwork/backend-review/`。
