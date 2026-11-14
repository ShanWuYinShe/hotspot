# HotSpot Backend (Go)

## 功能边界

Go 主后端,模块化单体 + S2 地理索引。提供认证/用户/群组/聊天/瞬间/Feed/搜索/地图/分享/媒体/举报/通知/系统/管理后台等全部 HTTP API 与 WebSocket 实时能力。支付/交易、端侧离线同步、消息全文搜索不在范围内。

API 契约见 `spec/api_spec_*.md`（仓库根 `spec/` 目录，只读权威）。

## 构建 & 测试

- `go build ./...` — 编译
- `go test -short ./...` — 单元测试（`-short` 跳过需要 DB 的集成测试）
- `go vet ./...` — 静态检查
- `gofmt -w .` — 格式化
- 集成测试（E2E，需要 Docker）：
  ```bash
  docker compose -f docker-compose.yml -f docker-compose.dev.yml --profile test up -d backend-test
  BACKEND_URL=http://localhost:8082 go test -count=1 ./test/integration/
  ```
  `backend-test` 为独立测试后端（`RATE_LIMIT_BYPASS=1` 显式跳过限流 + 独立 `hotspot_test` 数据库，见 `docker-compose.dev.yml`），不污染 dev 主库。

- **dev 环境限流默认关闭**：`docker-compose.dev.yml` 的 backend 注入 `RATE_LIMIT_BYPASS: ${RATE_LIMIT_BYPASS:-1}`，
  联调 / UI 审计 / 自动化测试不会被 `user_per_hour`、`auth_signin_per_10minutes` 等桶打出 429。
  需要验证限流行为时显式打开（env 变更必须 recreate 容器）：
  `RATE_LIMIT_BYPASS=0 docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d --force-recreate backend`；
  限流阈值本身在 DB `AppConfig.rate_limits`（管理员配置，变更即时生效），可配合调小验证。
  生产 compose 不注入该变量，始终启用限流。

### 本地复现 CI（按需）：`./scripts/verify-backend-ci.sh`

> 路径说明：本文所有 `scripts/xxx.sh` 都在**仓库根 `hotspot/scripts/`**，不是 `backend-go/scripts/`（该目录不存在）。
> 从 `backend-go/` 里执行要写 `../scripts/verify-backend-ci.sh`，或先 `cd ..` 到仓库根。同理 `docs/` 指的是
> 仓库根 `hotspot/docs/`（如 `docs/backend-audit.md` = `../docs/backend-audit.md`）。

日常推送**不要求**本地跑全量验证：全量测试、`-race`、覆盖率门禁、真实库 IT 与 HTTP E2E 一律由
GitHub Actions 承担，CI 全绿为准（流程见根 `AGENTS.md`「CI 门禁与本地验证分工」）。

`go test -short` **不覆盖** CI 的以下检查；需要本地复现 CI 失败时按范围用仓库根脚本执行
（真实库 SQL/参数错误会造成红灯）：

- `-race` 数据竞争 + `internal/modules` 每包覆盖率 **≥70%** 门禁（无 DB 单测口径下限）；
- 真实库集成测试（会暴露 SQL 列歧义、参数个数不符等 stub 单测无法发现的问题；
  CI backend-integration job 已覆盖）：`internal/modules` 的 `TestPostgresIT|Contract`
  仓库/契约测试，以及 `internal/platform` 的 outbox 事务回滚与派发、RunInTransaction
  aftercommit、WarmPlanner、auth session/refresh token 契约等真实库用例；
- HTTP 全链路集成（`test/integration`，~100 用例，需 compose dev 栈）。

```bash
./scripts/verify-backend-ci.sh              # 静态 + race + 覆盖率（默认）
./scripts/verify-backend-ci.sh --with-db    # 追加真实库 IT + 模块覆盖率 ≥90% 门禁
./scripts/verify-backend-ci.sh --all        # 全量（含 E2E 与 govulncheck）
```

脚本与 `.github/workflows/ci.yml` 的 backend / backend-integration job 逐条对齐；
worktree 内执行时 DB/E2E 阶段自动回落到主工作树 compose（验证主树源码）。

**覆盖率口径**：无 DB 的默认阶段要求 `internal/modules` 每包 ≥70%（repo 代码在无库时天然不可覆盖）；
`--with-db` 与 CI backend-integration job 在真实库上跑全部模块测试（含 `TestPostgresIT|Contract`）
与 `internal/platform` 全部测试（含 platform 层真实库 IT），模块覆盖率要求每包 **≥90%**。

## 潜在问题审计

静态审计清单与结论（goroutine panic、context 泄漏、连接/事务、忽略错误、SQL 注入、无界查询）
见 `docs/backend-audit.md`。**已确认无问题的项不必重复排查**；新发现请更新该文档。

## 性能：PGO（Profile-Guided Optimization）

Go 1.21+ 支持用真实生产 profile 做编译器优化（官方口径 **2–14%**，零代码改动）。
本仓库的采集方式：

```bash
# 前提：线上/准生产实例开启 PPROF_ADDR（localhost 绑定）且有真实流量
./scripts/collect-pgo.sh            # 采 60s CPU profile → backend-go/cmd/server/default.pgo
git add backend-go/cmd/server/default.pgo
go build ./cmd/server               # Go 自动发现同目录 default.pgo
```

**纪律**

- **必须用生产/准生产流量采集**：dev 环境热点分布不同，dev profile 编译可能反而变慢，
  本仓库不提交 dev 采集的 profile；
- 生效验证用对照构建（`go build -pgo=off` vs 默认）+ 线上 P95/QPS 对比，或 benchmark + benchstat；
- profile 会随版本漂移，大版本前重采一次。

### 只读性能冒烟（`scripts/perf-smoke.sh`）

对指定实例跑一组固定读端点（account/bootstrap/notify/following/nearby/users-search），
先预热再采样输出 rps 与 p50/p95/p99；默认坐标/搜索关键词对齐 dev 种子数据，且
**读端点出现非 200/code!=0 或低于阈值时退出 1**（可用于发版前冒烟）：

```bash
./scripts/perf-smoke.sh                                  # 默认 backend-test :8082，c=10、warmup 2s、5s+2s
./scripts/perf-smoke.sh --base http://localhost:8888 --concurrency 20 --duration 10s
./scripts/perf-smoke.sh --min-rps 100 --max-p99-ms 200   # 回归门禁模式（阈值按机器保守填）
TOKEN=<既有 token> ./scripts/perf-smoke.sh               # 非测试环境必须显式传 TOKEN（不新建访客）
```

基线数字、SLO 与容量模型见 `docs/performance.md`；组件级热路径用 Go benchmark
（`go test -run '^

注意：本机 OrbStack 虚拟机在不同时间窗口的绝对吞吐波动可达 ±2 倍（CPU 节流/后台负载），
**只有同一会话内的前后对比可信**；跨时间比较请同时记录时间与数据集（如 hotspot_test 是否被压测数据污染）。

**其他性能基线**（避免重复排查）：pprof 端点存在、pgx 连接池
MaxConns 25/MinConns 2 可配、HTTP 超时 15s/idle 60s、Redis 命令超时 500ms/1s 为有意 fail-slow；
本地热读接口耗时以 `./scripts/perf-smoke.sh` 输出的 p50 为准（同机同会话内前后对比）——当前量级不需要读缓存。

## Dev 容器（热加载）

- dev 模式后端跑 `Dockerfile.dev`（基于 `golang:1.26.11-alpine` + air），源码挂载 + air 监听 `.go`/`.toml` 变化自动重编译重启，改 Go 代码无需重启容器；`.air.toml` 配置编译/输出路径（`tmp/` 由 compose 卷隔离）。
- 首次构建/重建镜像：`docker compose -f docker-compose.yml -f docker-compose.dev.yml build backend`（`GOPROXY` 默认 `goproxy.cn,direct`，依赖经 BuildKit cache mount 缓存）。
- 注意：`Dockerfile.dev` 的 `ARG GO_VERSION`/`AIR_VERSION` 必须带精确 patch 版本（如 `1.26.11`），浮动 tag（`1.26-alpine`）拉取易超时；跨仓版本主本见 `~/work/workwork/versions.env`，改版本走 `~/work/workwork/scripts/sync-versions.sh`。

## 开发种子数据（dev 环境）

前端已无任何 API mock，dev 下全部走真实后端。首次启动或数据被改乱后，用种子命令注入
一套可复现数据（Postgres + SeaweedFS 双写；幂等，重复执行不产生重复数据）：

```bash
# 仓库根执行（worktree 内同样适用；自动走 dev compose 的 backend 容器）
./scripts/seed-dev.sh            # 默认 --if-missing：缺失则写入，已存在则跳过（保留开发中产生的互动）
./scripts/seed-dev.sh --reset    # 先按固定前缀清理 seed 数据（含 S3 对象）再重建
./scripts/seed-dev.sh --dry-run  # 只打印计划（账号/媒体数量/各表行数），不写数据
```

前置：dev compose 已启动（`docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d`）。
脚本在 backend 容器内执行 `go run ./cmd/seed`（继承容器内 POSTGRES_DSN / S3_* env），
无需新增 compose 服务；dev compose 已把仓库根 `devdata/` 只读挂载到 `/devdata` 供 seed 读取素材
（脚本显式传 `-assets-dir /devdata/seed/media`）；也可在容器内直接执行
`go run ./cmd/seed [--reset|--dry-run]`。
全新 SeaweedFS 卷默认没有 S3 bucket（PUT 会 404），脚本会在真实写入前尽力自动创建
（`weed shell s3.bucket.create`；现有 dev 卷已存在时为空操作）。

| 账号 | login_id | 口令（env 可覆盖） | 角色 |
|------|----------|--------------------|------|
| 鹿鸣 | `qa@hotspot.local` | `SEED_QA_PASSWORD`（默认 `Qa12345678`） | 普通用户（QA 主账号，`user_seed_lumos`） |
| 巡检员 | `admin@hotspot.local` | `SEED_ADMIN_PASSWORD`（默认 `Admin12345678`） | 管理员（`grant_all`，同时是完整普通用户） |

其余 8 个开发账号（`sophia/marco/mika/chen/lin/xiaoman/privacy/banned@hotspot.local`）口令统一
`Seed123456`；仅本地开发使用，生产环境 `cmd/seed` 会直接拒绝执行（APP_ENV 判定）。

- **实现位置**：`internal/seed/`（数据规格 `specs_*.go`、媒体计划 `mediaplan.go`、写入层
  `tables.go`、清理 `cleanup.go`、dry-run `plan.go`、执行 `run.go`）；入口 `cmd/seed/main.go`。
- **素材**：仓库根 `devdata/seed/media/*.jpg`（外置开发数据配套设施，**不 embed、不进生产二进制**；
  详见 `devdata/seed/README.md`）：9 张 400x400 头像、4 张 800x800 群头像、12 张 1200x800 瞬间图；
  目录由 `-assets-dir` / `SEED_ASSETS_DIR` 指定（默认候选 `../devdata/seed/media`、`/devdata/seed/media`；
  dev compose 把仓库根 `devdata/` 挂载到 `/devdata`，`scripts/seed-dev.sh` 显式传
  `/devdata/seed/media`）；替换素材须保持文件名与尺寸，缺失直接报错并列清单。
  `file_id/storage_key` 复写 media 模块规则（`file_+md5(userID:uploadID)` /
  `uploads/<userID>/<uploadID>.jpg`，单测锁定）。
- **固定 ID 前缀**（`--reset` 按前缀清理，也是 QA 脚本的稳定 ID）：`user_seed_*`、
  `moment_seed_01..24`、`comment_seed_*`、`group_seed_1..4`、`gmsg_seed_*`、`chat_seed_*`、
  `notify_seed_*`、`share_seed_*`、`report_seed_*`、`upload_seed_*`、`role_seed_*`、`audit_seed_*` 等。
- **数据规模**（dry-run 可见）：36 张表 358 行 + 38 个媒体对象；覆盖 10 用户/24 瞬间/4 群组/
  3 私聊/9 通知/2 分享/4 举报/2 管理角色/6 审计日志/2 系统公告/1 份 AppConfig。
- **S2 地理索引**：users/moments/groups 的 `s2_cell_id` 用 `platform/geo.LeafCellID`、
  groups 的 `s2_cell_token` 用 `geo.CellTokenAtLevel(point, 14)`（与各模块写入口径一致），
  否则附近/地图查询为空。
- **分享码**：shares 表契约要求 12 位、字符集 `[A-HJ-NP-Z2-9]`；seed 使用
  `SEEDSUNSET23`（moment_seed_01）与 `SEEDUSERPAGE`（user_seed_lumos），QA 走
  `/share/SEEDSUNSET23` 即有内容（seed-spec 里 3 位的 `abc/def` 会被参数校验拒绝）。
- **与 seed-spec 的差异（以代码/迁移为准）**：reports 状态为 `pending/resolved/dismissed`
  （非 handled/rejected）；瞬间软删列为 `is_deleted`（迁移 037，无 `deleted_at` 列）；
  shares 表无 `view_count` 列；管理端权限 key 为 `admin.reports.read` 形式（非 `reports:read`）；
  反应 emoji 取 AppConfig 白名单（👍 ❤️ 😂 😮 😢 🙏，无 🔥）。
- **默认浏览锚点**：`app_configs.defaults.default_location` 与 `default_group_settings.location`
  写入徐州云龙湖（`specs.go` 的 `seedAnchor*`，与 `user_seed_lumos` 一致），否则服务端默认
  南天门与徐州种子数据相距 400+km、附近/地图页开箱即空。**已有 app_configs 配置不会被覆盖**：
  seed 只写自己创建的行（`--reset` 同样只删 `updated_by_user_id LIKE 'user_seed_%'` 的行），
  已有环境（如主开发库）的既有默认位置保持原样；全新库（CI/新机器）上 seed 写入即生效。
  需要强制对齐时用管理端「应用配置」页修改，或先删除该 `config_id='default'` 行再重跑 seed。
- **测试**：`go test ./internal/seed/...`（不依赖 DB/网络/devdata：素材由单测在临时目录现场生成）。
  验证口径：真实 Postgres + SeaweedFS 下连跑两次 `./scripts/seed-dev.sh` 应幂等（无重复数据），`--reset` 后重建结果一致。

## 技术栈（名称级）

- Go（见 `go.mod`）、chi router、PostgreSQL + pgx/v5、Redis（限流/幂等缓存/WS 跨实例 fanout，可选）、google/s2（地理网格索引）、gorilla/websocket、zap（结构化日志）

## 目录结构（强制）

```
backend-go/
├── cmd/                    # 入口（server / migrate / seed）
├── migrations/             # SQL 迁移（embed 进二进制）
├── test/                   # 集成测试（E2E，需要 Docker）
└── internal/
    ├── bootstrap/          # 装配：路由聚合、模块注册、worker、WS 鉴权
    ├── modules/            # 业务模块（每模块一个 package）
    └── platform/           # 横切基础设施（auth/realtime/httpx/...）
```

规则：
- `internal/modules/*` 只能依赖 `internal/platform/*` 和同层其它模块的 Service，禁止反向依赖 `bootstrap`
- `internal/platform/*` 不含业务判断，只提供可复用基础设施
- 业务模块之间通过 Service 接口解耦（如 `groups` 注入 `users.Service`），不直接 import 对方 repo

## 架构分层（强制）

```
Handler -> Service -> Repository
```

- Handler（`handler*.go`）：仅做参数提取/校验、调用 Service、返回统一响应。不写业务逻辑、不直接查库
- Service（`service*.go`）：业务规则/权限/事务/事件发布，不写 SQL
- Repository（`repo*.go`）：只封装 SQL 与行映射，不含业务判断

每模块的文件组织约定：
- `module.go`：挂载路由；在 `internal/bootstrap/module_registry.go` 的 `BuildModules()` 中注册
- `handler.go`（必要时 `handler_xxx.go` 按子域拆分）
- `service.go`（必要时 `service_xxx.go`）
- `repo.go`（接口）为入口；双实现与 SQL 组织见「持久化（Repository 双实现）」节；文件允许按子域继续拆分（`repo_postgres_<子域>.go` / `handler_<子域>.go` / `service_<子域>.go`，如 moments 现有 `repo_postgres_read.go` / `repo_postgres_images.go`）
- `types.go`：请求/响应/内部模型（DTO）
- `errors.go`：模块级 sentinel error

## 数据库迁移

**完整纪律见 skill hotspot-backend-migrations**。要点速记：

- `migrations/NNN_描述.sql` 字典序执行；`schema_migrations` 版本表记录 `name + checksum + execution_ms`。
  每个文件与其版本登记在**同一事务**内提交（原子）：文件执行失败/进程崩溃不会留下
  「已执行未登记」的重放窗口，失败文件也不会被标记为已应用。
- **不做基线、不做存量库接管**：库内已有业务表但版本表为空时启动直接失败；必须从空库按文件顺序执行全量迁移。
- **checksum 冻结历史**：已应用迁移文件的任何改动/删除都会在启动期 fail-fast；
  要改 schema 一律新增迁移文件（不允许改写历史，即使已不考虑二进制向后兼容）。
- **schema manifest**：`migrations/schema_manifest.json` 是「全部迁移执行后的期望 schema」
  快照（表/列/索引名）。启动与 `cmd/migrate -verify-only` 校验期望对象是否齐全；
  新增/修改迁移后必须重建：在全新库执行
  `go run ./cmd/migrate -write-manifest=migrations/schema_manifest.json`，
  `verify-backend-ci.sh --with-db` 会用「临时库重放 + diff」保障 manifest 不落后。
- 破坏性 DDL（DROP/RENAME/TYPE 变更）不再被静态禁止（用户明确不考虑向后兼容），
  但迁移文件必须以 `-- allow-destructive: <理由>` 显式声明；CI 校验
  `scripts/check-migrations.sh`。
- **禁止 CREATE INDEX CONCURRENTLY**（迁移整文件在显式事务内执行，CONCURRENTLY 报 25001；
  需在线建索引时拆成单语句文件并在 runner 侧支持单独执行模式，尚属未支持项）。
- 无 down 迁移；回滚 = 旧镜像 + 保留新 schema。

## 路由与协议

路由与协议契约以 `spec/api_spec_base.md` 为权威来源，本节仅列出实现侧约束：

- 仅 POST，RPC + JSON；所有 API 必须是明确动作子路径，禁止裸资源路径（如禁止 `/api/v1/users`，须使用 `/api/v1/users/search` 等动作路径）
- 登记例外：`GET /share/{code}`（`share/og.go`）——面向链接预览爬虫的 HTML 落地页（og 标签），nginx 按 `$share_crawler` UA 分流（见 docker/nginx*.conf），不属于 `/api` RPC 面；真实用户仍走 SPA `/share/[code]`
- 认证分公开/必须/管理员三档（对应 spec 的 public/api/admin 三节点），每模块按端点的认证要求分组挂载对应中间件
- 统一中间件链（顺序）：鉴权 → 限流 → 资源守卫 → 幂等

## 统一响应与错误处理

响应与错误码契约以 `spec/api_spec_base.md` + `spec/api_spec_errors.md` 为权威来源，本节仅列出实现侧约束：

- 所有业务响应 HTTP 200 + 统一 envelope；HTTP 非 200 仅用于网关/框架/不可预期异常
- 业务成功以 `code == 0` 判断；错误码只能取自 spec，禁止自创
- 禁止向客户端暴露内部错误细节（SQL、堆栈、文件路径）
- Service 返回模块级 sentinel error，Handler 负责映射为错误码；仅 `10600` 用 Error 级日志，其它业务错误用 Warn

## 请求解码

- 请求体必须严格解码：拒绝尾随 JSON；未知字段默认拒绝（`{}` 端点例外，见下条）；禁止裸用标准库 `json.NewDecoder`
- 端点声明 Request 为 `{}` 时，必须经 `httpx.RequireEmptyObject` 校验：缺失请求体/非对象/非法 JSON 返回 `10000`；未知字段容忍（spec 已放宽，SDK/网关注入字段不硬失败）

## 认证与会话

认证与会话契约以 `spec/api_spec_auth.md` 为权威来源，本节仅列出实现侧约束：

- JWT 仅含 user_id 与不可变的 session_id，禁止写入可变字段
- Web 端通过 HttpOnly Cookie，App 端通过 Bearer Token（详见 spec）
- 鉴权分公开/必须（未登录拒绝）/管理员（额外 RBAC）三档；不存在 optional 档，需要「匿名可读、登录增强」的端点按 spec 拆分到 public 与 api 两个节点
- CSRF 防护完全依赖 auth Cookie 的 `SameSite=Strict`，不使用 CSRF Token
- 失效机制为 session 撤销：登出/撤销时标记 session 失效，后续请求校验 session 状态即拒绝；不需要独立的 token 黑名单
- 刷新 Token 必须轮转，旧 token 进入 30 秒宽限期后拒绝（详见 spec 并发刷新策略）
- 封禁账号在鉴权层拦截并返回 `10201`（强制登出 + 封禁页）

## 幂等性

幂等契约以 `spec/api_spec_base.md` 为权威来源，本节仅列出实现侧约束：

- 需要实现幂等的端点以 spec 的清单为准（当前 10 个：moments/create、comments/create、groups/create、groups/admin/announcements/create、/admin/api/v1/announcements/create、share/create、private_messages/send、group_messages/send、auth/register、auth/guest）；不需要的端点不得强制要求 `Idempotency-Key`
- 服务端按 `(user_id + endpoint + key)` 去重，仅缓存成功响应（`code == 0`）

## 分页

- 禁止 OFFSET，必须使用 cursor/keyset 分页
- cursor 不透明，客户端只透传；编码/解码仅在 Repository 层
- 时间 keyset 的精度必须对齐：游标携带的时间精度不得粗于排序列的写入精度（如列以 `time.Now().UTC()` 全精度写入，游标必须用 `RFC3339Nano` 编码；秒截断游标配全精度列会导致同秒行在翻页中静默丢失，如 `moments/fetch_favorites` 的游标实现）。新增时间 keyset 列表时，复核「写入精度 × 游标精度 × 谓词」三者一致
- limit 默认 20、上限 100，由 Service 层兜底裁剪
- 无更多数据时 `next_cursor` 为 `null`（恒定 nullable，见 `spec/api_spec_types.md`），禁止省略字段

## 数据格式

数据格式契约以 `spec/api_spec_base.md` 为权威来源，本节仅列出实现侧需注意的约束：

- 时间字段统一 UTC；`xxx_id` 字段在 API 层统一为 `string`（DB 主键可为数值，但对外必须字符串化）

## 地理位置计算

- S2 cell 预过滤必须在 PostgreSQL 完成（`s2_cell_id` B-tree 范围扫描 + `EXISTS unnest`），将候选集从全表压缩到小范围
- 精确距离（Haversine）计算与排序在 Go 层完成——S2 已将候选集压小，Go 层计算代价可忽略；保持 SQL 简单，避免在 SQL 里内联冗长的三角函数表达式

## SQL 构造

动态 SQL 必须参数化（`$N` 占位符），列名/排序方向通过白名单映射，禁止字符串拼接注入。

## 配置

运行参数（限流阈值、响应默认值、应用限制等）从 DB 持久化的 AppConfig 读取，变更即时生效无需重启。模块通过注入的配置源获取参数，禁止直接读环境变量或硬编码阈值。

## 持久化（Repository 双实现）

- 每模块 Repository 至少提供两套实现：`repo_postgres.go`（生产）+ `repo_memory.go`（内存，用于单测与无 DB 场景）
- DB 未配置时优雅降级到内存实现
- SQL 写在 `repo_postgres_queries.go`，保持查询与映射分离

## WebSocket

WebSocket 契约以 `spec/api_spec_websocket.md` 为权威来源，本节仅列出实现侧约束：

- 与 HTTP 共享同一套 auth 机制
- 连接建立后 5 秒内必须收到 Auth，否则发送 `Error(code=20000, severity=fatal)` 并断开
- 原生客户端（鸿蒙等）不受浏览器限制，实测会出现两种非浏览器行为：`Origin: null`，或同时发送 OS 默认 `Origin: <scheme>://<host>`（丢非默认端口）与客户端补充的同源 Origin；`sameOrigin` 在**无 Cookie** 时放行 null Origin，并对多 Origin 请求要求所有主机名与请求 Host 一致（浏览器禁止脚本设置 Origin，多值只可能来自原生客户端）。带 Cookie 的上述请求仍按跨站拒绝（验证方式：真机实测；见 `internal/platform/realtime/handler.go`）
- 心跳参数由服务端下发，客户端不得硬编码
- 多实例部署时必须配置跨实例 fanout（RedisBridge），否则跨实例事件丢失

## 未读计数（聊天）

- 未读计数由客户端基于 WebSocket 增量维护
- 服务端仅提供冷启动校准（如历史消息基线）
- 服务端不维护独立的未读计数字段，通过已读游标（如 `last_read_message_id`）在查询时动态计算

## 日志与安全

- 日志必须携带 `X-Request-Id`，使用结构化日志，禁止 `fmt.Println`
- 禁止记录密码、Token、隐私内容；IP 在日志中可记录，但对外响应需脱敏

## 指标（/metrics）

- `GET /metrics` 暴露 Prometheus 文本指标；仅 compose 网络内可达（nginx 不反代该路径，外部访问落到 SPA 兜底）
- registry 统一用 `observability.Registry()`，禁止 `prometheus.DefaultRegisterer`（避免把无关全局采集器带进抓取面）；自定义采集器经 `observability.Register()` 在 bootstrap 装配
- 内置采集面：Go 运行时/进程、HTTP 请求计数与耗时（`RequestMetricsMiddleware`，route 维度用 chi 路由模板，未匹配归 `unmatched`——禁止把请求路径直接当标签值，基数爆炸）、WS 连接规模（`realtime.NewHubCollector`）、pgxpool 统计（`NewDBPoolCollector`）
- 采集栈（可选，默认不启动）：`docker compose --profile observability up -d` → prometheus :9090 / grafana :3001（数据源已 provisioning；镜像 pin 见 `.env.example`） -bench . -benchmem ./internal/platform/...`）做同机前后对比。

注意：本机 OrbStack 虚拟机在不同时间窗口的绝对吞吐波动可达 ±2 倍（CPU 节流/后台负载），
**只有同一会话内的前后对比可信**；跨时间比较请同时记录时间与数据集（如 hotspot_test 是否被压测数据污染）。

**其他性能基线**（避免重复排查）：pprof 端点存在、pgx 连接池
MaxConns 25/MinConns 2 可配、HTTP 超时 15s/idle 60s、Redis 命令超时 500ms/1s 为有意 fail-slow；
本地热读接口耗时以 `./scripts/perf-smoke.sh` 输出的 p50 为准（同机同会话内前后对比）——当前量级不需要读缓存。

## Dev 容器（热加载）

- dev 模式后端跑 `Dockerfile.dev`（基于 `golang:1.26.11-alpine` + air），源码挂载 + air 监听 `.go`/`.toml` 变化自动重编译重启，改 Go 代码无需重启容器；`.air.toml` 配置编译/输出路径（`tmp/` 由 compose 卷隔离）。
- 首次构建/重建镜像：`docker compose -f docker-compose.yml -f docker-compose.dev.yml build backend`（`GOPROXY` 默认 `goproxy.cn,direct`，依赖经 BuildKit cache mount 缓存）。
- 注意：`Dockerfile.dev` 的 `ARG GO_VERSION`/`AIR_VERSION` 必须带精确 patch 版本（如 `1.26.11`），浮动 tag（`1.26-alpine`）拉取易超时；跨仓版本主本见 `~/work/workwork/versions.env`，改版本走 `~/work/workwork/scripts/sync-versions.sh`。

## 开发种子数据（dev 环境）

前端已无任何 API mock，dev 下全部走真实后端。首次启动或数据被改乱后，用种子命令注入
一套可复现数据（Postgres + SeaweedFS 双写；幂等，重复执行不产生重复数据）：

```bash
# 仓库根执行（worktree 内同样适用；自动走 dev compose 的 backend 容器）
./scripts/seed-dev.sh            # 默认 --if-missing：缺失则写入，已存在则跳过（保留开发中产生的互动）
./scripts/seed-dev.sh --reset    # 先按固定前缀清理 seed 数据（含 S3 对象）再重建
./scripts/seed-dev.sh --dry-run  # 只打印计划（账号/媒体数量/各表行数），不写数据
```

前置：dev compose 已启动（`docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d`）。
脚本在 backend 容器内执行 `go run ./cmd/seed`（继承容器内 POSTGRES_DSN / S3_* env），
无需新增 compose 服务；dev compose 已把仓库根 `devdata/` 只读挂载到 `/devdata` 供 seed 读取素材
（脚本显式传 `-assets-dir /devdata/seed/media`）；也可在容器内直接执行
`go run ./cmd/seed [--reset|--dry-run]`。
全新 SeaweedFS 卷默认没有 S3 bucket（PUT 会 404），脚本会在真实写入前尽力自动创建
（`weed shell s3.bucket.create`；现有 dev 卷已存在时为空操作）。

| 账号 | login_id | 口令（env 可覆盖） | 角色 |
|------|----------|--------------------|------|
| 鹿鸣 | `qa@hotspot.local` | `SEED_QA_PASSWORD`（默认 `Qa12345678`） | 普通用户（QA 主账号，`user_seed_lumos`） |
| 巡检员 | `admin@hotspot.local` | `SEED_ADMIN_PASSWORD`（默认 `Admin12345678`） | 管理员（`grant_all`，同时是完整普通用户） |

其余 8 个开发账号（`sophia/marco/mika/chen/lin/xiaoman/privacy/banned@hotspot.local`）口令统一
`Seed123456`；仅本地开发使用，生产环境 `cmd/seed` 会直接拒绝执行（APP_ENV 判定）。

- **实现位置**：`internal/seed/`（数据规格 `specs_*.go`、媒体计划 `mediaplan.go`、写入层
  `tables.go`、清理 `cleanup.go`、dry-run `plan.go`、执行 `run.go`）；入口 `cmd/seed/main.go`。
- **素材**：仓库根 `devdata/seed/media/*.jpg`（外置开发数据配套设施，**不 embed、不进生产二进制**；
  详见 `devdata/seed/README.md`）：9 张 400x400 头像、4 张 800x800 群头像、12 张 1200x800 瞬间图；
  目录由 `-assets-dir` / `SEED_ASSETS_DIR` 指定（默认候选 `../devdata/seed/media`、`/devdata/seed/media`；
  dev compose 把仓库根 `devdata/` 挂载到 `/devdata`，`scripts/seed-dev.sh` 显式传
  `/devdata/seed/media`）；替换素材须保持文件名与尺寸，缺失直接报错并列清单。
  `file_id/storage_key` 复写 media 模块规则（`file_+md5(userID:uploadID)` /
  `uploads/<userID>/<uploadID>.jpg`，单测锁定）。
- **固定 ID 前缀**（`--reset` 按前缀清理，也是 QA 脚本的稳定 ID）：`user_seed_*`、
  `moment_seed_01..24`、`comment_seed_*`、`group_seed_1..4`、`gmsg_seed_*`、`chat_seed_*`、
  `notify_seed_*`、`share_seed_*`、`report_seed_*`、`upload_seed_*`、`role_seed_*`、`audit_seed_*` 等。
- **数据规模**（dry-run 可见）：36 张表 358 行 + 38 个媒体对象；覆盖 10 用户/24 瞬间/4 群组/
  3 私聊/9 通知/2 分享/4 举报/2 管理角色/6 审计日志/2 系统公告/1 份 AppConfig。
- **S2 地理索引**：users/moments/groups 的 `s2_cell_id` 用 `platform/geo.LeafCellID`、
  groups 的 `s2_cell_token` 用 `geo.CellTokenAtLevel(point, 14)`（与各模块写入口径一致），
  否则附近/地图查询为空。
- **分享码**：shares 表契约要求 12 位、字符集 `[A-HJ-NP-Z2-9]`；seed 使用
  `SEEDSUNSET23`（moment_seed_01）与 `SEEDUSERPAGE`（user_seed_lumos），QA 走
  `/share/SEEDSUNSET23` 即有内容（seed-spec 里 3 位的 `abc/def` 会被参数校验拒绝）。
- **与 seed-spec 的差异（以代码/迁移为准）**：reports 状态为 `pending/resolved/dismissed`
  （非 handled/rejected）；瞬间软删列为 `is_deleted`（迁移 037，无 `deleted_at` 列）；
  shares 表无 `view_count` 列；管理端权限 key 为 `admin.reports.read` 形式（非 `reports:read`）；
  反应 emoji 取 AppConfig 白名单（👍 ❤️ 😂 😮 😢 🙏，无 🔥）。
- **默认浏览锚点**：`app_configs.defaults.default_location` 与 `default_group_settings.location`
  写入徐州云龙湖（`specs.go` 的 `seedAnchor*`，与 `user_seed_lumos` 一致），否则服务端默认
  南天门与徐州种子数据相距 400+km、附近/地图页开箱即空。**已有 app_configs 配置不会被覆盖**：
  seed 只写自己创建的行（`--reset` 同样只删 `updated_by_user_id LIKE 'user_seed_%'` 的行），
  已有环境（如主开发库）的既有默认位置保持原样；全新库（CI/新机器）上 seed 写入即生效。
  需要强制对齐时用管理端「应用配置」页修改，或先删除该 `config_id='default'` 行再重跑 seed。
- **测试**：`go test ./internal/seed/...`（不依赖 DB/网络/devdata：素材由单测在临时目录现场生成）。
  验证口径：真实 Postgres + SeaweedFS 下连跑两次 `./scripts/seed-dev.sh` 应幂等（无重复数据），`--reset` 后重建结果一致。

## 技术栈（名称级）

- Go（见 `go.mod`）、chi router、PostgreSQL + pgx/v5、Redis（限流/幂等缓存/WS 跨实例 fanout，可选）、google/s2（地理网格索引）、gorilla/websocket、zap（结构化日志）

## 目录结构（强制）

```
backend-go/
├── cmd/                    # 入口（server / migrate / seed）
├── migrations/             # SQL 迁移（embed 进二进制）
├── test/                   # 集成测试（E2E，需要 Docker）
└── internal/
    ├── bootstrap/          # 装配：路由聚合、模块注册、worker、WS 鉴权
    ├── modules/            # 业务模块（每模块一个 package）
    └── platform/           # 横切基础设施（auth/realtime/httpx/...）
```

规则：
- `internal/modules/*` 只能依赖 `internal/platform/*` 和同层其它模块的 Service，禁止反向依赖 `bootstrap`
- `internal/platform/*` 不含业务判断，只提供可复用基础设施
- 业务模块之间通过 Service 接口解耦（如 `groups` 注入 `users.Service`），不直接 import 对方 repo

## 架构分层（强制）

```
Handler -> Service -> Repository
```

- Handler（`handler*.go`）：仅做参数提取/校验、调用 Service、返回统一响应。不写业务逻辑、不直接查库
- Service（`service*.go`）：业务规则/权限/事务/事件发布，不写 SQL
- Repository（`repo*.go`）：只封装 SQL 与行映射，不含业务判断

每模块的文件组织约定：
- `module.go`：挂载路由；在 `internal/bootstrap/module_registry.go` 的 `BuildModules()` 中注册
- `handler.go`（必要时 `handler_xxx.go` 按子域拆分）
- `service.go`（必要时 `service_xxx.go`）
- `repo.go`（接口）为入口；双实现与 SQL 组织见「持久化（Repository 双实现）」节；文件允许按子域继续拆分（`repo_postgres_<子域>.go` / `handler_<子域>.go` / `service_<子域>.go`，如 moments 现有 `repo_postgres_read.go` / `repo_postgres_images.go`）
- `types.go`：请求/响应/内部模型（DTO）
- `errors.go`：模块级 sentinel error

## 数据库迁移

**完整纪律见 skill hotspot-backend-migrations**。要点速记：

- `migrations/NNN_描述.sql` 字典序执行；`schema_migrations` 版本表记录 `name + checksum + execution_ms`。
  每个文件与其版本登记在**同一事务**内提交（原子）：文件执行失败/进程崩溃不会留下
  「已执行未登记」的重放窗口，失败文件也不会被标记为已应用。
- **不做基线、不做存量库接管**：库内已有业务表但版本表为空时启动直接失败；必须从空库按文件顺序执行全量迁移。
- **checksum 冻结历史**：已应用迁移文件的任何改动/删除都会在启动期 fail-fast；
  要改 schema 一律新增迁移文件（不允许改写历史，即使已不考虑二进制向后兼容）。
- **schema manifest**：`migrations/schema_manifest.json` 是「全部迁移执行后的期望 schema」
  快照（表/列/索引名）。启动与 `cmd/migrate -verify-only` 校验期望对象是否齐全；
  新增/修改迁移后必须重建：在全新库执行
  `go run ./cmd/migrate -write-manifest=migrations/schema_manifest.json`，
  `verify-backend-ci.sh --with-db` 会用「临时库重放 + diff」保障 manifest 不落后。
- 破坏性 DDL（DROP/RENAME/TYPE 变更）不再被静态禁止（用户明确不考虑向后兼容），
  但迁移文件必须以 `-- allow-destructive: <理由>` 显式声明；CI 校验
  `scripts/check-migrations.sh`。
- **禁止 CREATE INDEX CONCURRENTLY**（迁移整文件在显式事务内执行，CONCURRENTLY 报 25001；
  需在线建索引时拆成单语句文件并在 runner 侧支持单独执行模式，尚属未支持项）。
- 无 down 迁移；回滚 = 旧镜像 + 保留新 schema。

## 路由与协议

路由与协议契约以 `spec/api_spec_base.md` 为权威来源，本节仅列出实现侧约束：

- 仅 POST，RPC + JSON；所有 API 必须是明确动作子路径，禁止裸资源路径（如禁止 `/api/v1/users`，须使用 `/api/v1/users/search` 等动作路径）
- 登记例外：`GET /share/{code}`（`share/og.go`）——面向链接预览爬虫的 HTML 落地页（og 标签），nginx 按 `$share_crawler` UA 分流（见 docker/nginx*.conf），不属于 `/api` RPC 面；真实用户仍走 SPA `/share/[code]`
- 认证分公开/必须/管理员三档（对应 spec 的 public/api/admin 三节点），每模块按端点的认证要求分组挂载对应中间件
- 统一中间件链（顺序）：鉴权 → 限流 → 资源守卫 → 幂等

## 统一响应与错误处理

响应与错误码契约以 `spec/api_spec_base.md` + `spec/api_spec_errors.md` 为权威来源，本节仅列出实现侧约束：

- 所有业务响应 HTTP 200 + 统一 envelope；HTTP 非 200 仅用于网关/框架/不可预期异常
- 业务成功以 `code == 0` 判断；错误码只能取自 spec，禁止自创
- 禁止向客户端暴露内部错误细节（SQL、堆栈、文件路径）
- Service 返回模块级 sentinel error，Handler 负责映射为错误码；仅 `10600` 用 Error 级日志，其它业务错误用 Warn

## 请求解码

- 请求体必须严格解码：拒绝尾随 JSON；未知字段默认拒绝（`{}` 端点例外，见下条）；禁止裸用标准库 `json.NewDecoder`
- 端点声明 Request 为 `{}` 时，必须经 `httpx.RequireEmptyObject` 校验：缺失请求体/非对象/非法 JSON 返回 `10000`；未知字段容忍（spec 已放宽，SDK/网关注入字段不硬失败）

## 认证与会话

认证与会话契约以 `spec/api_spec_auth.md` 为权威来源，本节仅列出实现侧约束：

- JWT 仅含 user_id 与不可变的 session_id，禁止写入可变字段
- Web 端通过 HttpOnly Cookie，App 端通过 Bearer Token（详见 spec）
- 鉴权分公开/必须（未登录拒绝）/管理员（额外 RBAC）三档；不存在 optional 档，需要「匿名可读、登录增强」的端点按 spec 拆分到 public 与 api 两个节点
- CSRF 防护完全依赖 auth Cookie 的 `SameSite=Strict`，不使用 CSRF Token
- 失效机制为 session 撤销：登出/撤销时标记 session 失效，后续请求校验 session 状态即拒绝；不需要独立的 token 黑名单
- 刷新 Token 必须轮转，旧 token 进入 30 秒宽限期后拒绝（详见 spec 并发刷新策略）
- 封禁账号在鉴权层拦截并返回 `10201`（强制登出 + 封禁页）

## 幂等性

幂等契约以 `spec/api_spec_base.md` 为权威来源，本节仅列出实现侧约束：

- 需要实现幂等的端点以 spec 的清单为准（当前 10 个：moments/create、comments/create、groups/create、groups/admin/announcements/create、/admin/api/v1/announcements/create、share/create、private_messages/send、group_messages/send、auth/register、auth/guest）；不需要的端点不得强制要求 `Idempotency-Key`
- 服务端按 `(user_id + endpoint + key)` 去重，仅缓存成功响应（`code == 0`）

## 分页

- 禁止 OFFSET，必须使用 cursor/keyset 分页
- cursor 不透明，客户端只透传；编码/解码仅在 Repository 层
- 时间 keyset 的精度必须对齐：游标携带的时间精度不得粗于排序列的写入精度（如列以 `time.Now().UTC()` 全精度写入，游标必须用 `RFC3339Nano` 编码；秒截断游标配全精度列会导致同秒行在翻页中静默丢失，如 `moments/fetch_favorites` 的游标实现）。新增时间 keyset 列表时，复核「写入精度 × 游标精度 × 谓词」三者一致
- limit 默认 20、上限 100，由 Service 层兜底裁剪
- 无更多数据时 `next_cursor` 为 `null`（恒定 nullable，见 `spec/api_spec_types.md`），禁止省略字段

## 数据格式

数据格式契约以 `spec/api_spec_base.md` 为权威来源，本节仅列出实现侧需注意的约束：

- 时间字段统一 UTC；`xxx_id` 字段在 API 层统一为 `string`（DB 主键可为数值，但对外必须字符串化）

## 地理位置计算

- S2 cell 预过滤必须在 PostgreSQL 完成（`s2_cell_id` B-tree 范围扫描 + `EXISTS unnest`），将候选集从全表压缩到小范围
- 精确距离（Haversine）计算与排序在 Go 层完成——S2 已将候选集压小，Go 层计算代价可忽略；保持 SQL 简单，避免在 SQL 里内联冗长的三角函数表达式

## SQL 构造

动态 SQL 必须参数化（`$N` 占位符），列名/排序方向通过白名单映射，禁止字符串拼接注入。

## 配置

运行参数（限流阈值、响应默认值、应用限制等）从 DB 持久化的 AppConfig 读取，变更即时生效无需重启。模块通过注入的配置源获取参数，禁止直接读环境变量或硬编码阈值。

## 持久化（Repository 双实现）

- 每模块 Repository 至少提供两套实现：`repo_postgres.go`（生产）+ `repo_memory.go`（内存，用于单测与无 DB 场景）
- DB 未配置时优雅降级到内存实现
- SQL 写在 `repo_postgres_queries.go`，保持查询与映射分离

## WebSocket

WebSocket 契约以 `spec/api_spec_websocket.md` 为权威来源，本节仅列出实现侧约束：

- 与 HTTP 共享同一套 auth 机制
- 连接建立后 5 秒内必须收到 Auth，否则发送 `Error(code=20000, severity=fatal)` 并断开
- 原生客户端（鸿蒙等）不受浏览器限制，实测会出现两种非浏览器行为：`Origin: null`，或同时发送 OS 默认 `Origin: <scheme>://<host>`（丢非默认端口）与客户端补充的同源 Origin；`sameOrigin` 在**无 Cookie** 时放行 null Origin，并对多 Origin 请求要求所有主机名与请求 Host 一致（浏览器禁止脚本设置 Origin，多值只可能来自原生客户端）。带 Cookie 的上述请求仍按跨站拒绝（验证方式：真机实测；见 `internal/platform/realtime/handler.go`）
- 心跳参数由服务端下发，客户端不得硬编码
- 多实例部署时必须配置跨实例 fanout（RedisBridge），否则跨实例事件丢失

## 未读计数（聊天）

- 未读计数由客户端基于 WebSocket 增量维护
- 服务端仅提供冷启动校准（如历史消息基线）
- 服务端不维护独立的未读计数字段，通过已读游标（如 `last_read_message_id`）在查询时动态计算

## 日志与安全

- 日志必须携带 `X-Request-Id`，使用结构化日志，禁止 `fmt.Println`
- 禁止记录密码、Token、隐私内容；IP 在日志中可记录，但对外响应需脱敏

## 指标（/metrics）

- `GET /metrics` 暴露 Prometheus 文本指标；仅 compose 网络内可达（nginx 不反代该路径，外部访问落到 SPA 兜底）
- registry 统一用 `observability.Registry()`，禁止 `prometheus.DefaultRegisterer`（避免把无关全局采集器带进抓取面）；自定义采集器经 `observability.Register()` 在 bootstrap 装配
- 内置采集面：Go 运行时/进程、HTTP 请求计数与耗时（`RequestMetricsMiddleware`，route 维度用 chi 路由模板，未匹配归 `unmatched`——禁止把请求路径直接当标签值，基数爆炸）、WS 连接规模（`realtime.NewHubCollector`）、pgxpool 统计（`NewDBPoolCollector`）
- 采集栈（可选，默认不启动）：`docker compose --profile observability up -d` → prometheus :9090 / grafana :3001（数据源已 provisioning；镜像 pin 见 `.env.example`）