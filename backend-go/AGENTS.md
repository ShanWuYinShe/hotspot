# HotSpot Backend (Go)

## 功能边界

Go 主后端,模块化单体 + S2 地理索引。提供认证/用户/群组/聊天/瞬间/Feed/搜索/地图/分享/媒体/举报/通知/系统/管理后台等全部 HTTP API 与 WebSocket 实时能力。支付/交易、端侧离线同步、消息全文搜索不在范围内。

历史 `backend/`（Rust/Axum）已删除（2026-08 清理），git 历史可找回。

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
  `backend-test` 为独立测试后端（`APP_ENV=test` 跳过限流 + 独立 `hotspot_test` 数据库，见 `docker-compose.dev.yml`），不污染 dev 主库。

## Dev 容器（热加载）

- dev 模式后端跑 `Dockerfile.dev`（基于 `golang:1.26.5-alpine` + air），源码挂载 + air 监听 `.go`/`.toml` 变化自动重编译重启，改 Go 代码无需重启容器；`.air.toml` 配置编译/输出路径（`tmp/` 由 compose 卷隔离）。
- 首次构建/重建镜像：`docker compose -f docker-compose.yml -f docker-compose.dev.yml build backend`（`GOPROXY` 默认 `goproxy.cn,direct`，依赖经 BuildKit cache mount 缓存）。
- 注意：`Dockerfile.dev` 的 `ARG GO_VERSION`/`AIR_VERSION` 必须带精确 patch 版本（如 `1.26.5`），浮动 tag（`1.26-alpine`）拉取易超时。

## 技术栈（名称级）

- Go（见 `go.mod`）、chi router、PostgreSQL + pgx/v5、Redis（限流/幂等缓存/WS 跨实例 fanout，可选）、google/s2（地理网格索引）、gorilla/websocket、zap（结构化日志）

## 目录结构（强制）

```
backend-go/
├── cmd/                    # 入口（server / migrate）
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
- `repo.go`（接口） + `repo_postgres.go`（生产） + `repo_memory.go`（内存，用于单测与无 DB 场景）为最小集合；SQL 集中 `repo_postgres_queries.go`，允许按子域继续拆分（`repo_postgres_<子域>.go` / `handler_<子域>.go` / `service_<子域>.go`，如 moments 现有 `repo_postgres_read.go` / `repo_postgres_images.go`）
- `types.go`：请求/响应/内部模型（DTO）
- `errors.go`：模块级 sentinel error

## 数据库迁移

`migrations/` 下 `NNN_描述.sql` 格式，嵌入二进制由启动时自动执行（字典序全量重跑，幂等靠各自 `IF NOT EXISTS`）。新增 migration 使用下一个可用序号（CI 校验序号唯一且无重复——历史 009 重号在脚本白名单放行，见仓库根 `scripts/check-migrations.sh`）。

**向后兼容纪律（硬性）**：迁移必须兼容「上一个已发布版本」的二进制——发布是无停机产物替换，旧版后端会短暂与新 schema 共存：

- 只加不删：新迁移内不 DROP/RENAME 仍在被上一版本读写的列与表；废弃列先停止写入，跨一个版本再清理（expand-contract 两阶段）。
- 改列语义/回填数据用 `DO $$` 自检式幂等写法（参照 `050`），不依赖「只跑一次」。
- 数据性一次性迁移（无法按格式判幂等的）必须用 `data_migrations` 旗标表包裹（参照 `053`）。
- 无 down 迁移：回滚版本 = 旧镜像 + 保留新 schema，因此 schema 变更必须对旧代码向后兼容，这是唯一防线。
- **迁移内禁止 `CREATE INDEX CONCURRENTLY`**：迁移经 pgx 简单协议（`QueryExecModeSimpleProtocol`）把整个文件作为一条消息执行，多语句文件落入 PostgreSQL 隐式事务块，CONCURRENTLY 以 25001 拒绝（2026-09-20 实测，`cmd/migrate` 与服务端 `ensureSchema` 两条路径均如此，会导致启动失败）。普通 CREATE INDEX 的 SHARE 锁会短暂阻塞读写是已知代价；要消除它需先建 `schema_migrations` 版本表并把建索引拆成单语句迁移文件（遗留项）。并发构建失败会留下 INVALID 索引且 `IF NOT EXISTS` 不会重建——启动日志见索引报错时先 DROP INVALID 索引再重启。

## 路由与协议

路由与协议契约以 `spec/api_spec_base.md` 为权威来源，本节仅列出实现侧约束：

- 仅 POST，RPC + JSON；所有 API 必须是明确动作子路径，禁止裸资源路径（如禁止 `/api/v1/users`，须使用 `/api/v1/users/search` 等动作路径）
- 认证分公开/可选/必须/管理员四档，每模块按端点的认证要求分组挂载对应中间件
- 统一中间件链（顺序）：鉴权 → 限流 → 资源守卫 → 幂等

## 统一响应与错误处理

响应与错误码契约以 `spec/api_spec_base.md` + `spec/api_spec_errors.md` 为权威来源，本节仅列出实现侧约束：

- 所有业务响应 HTTP 200 + 统一 envelope；HTTP 非 200 仅用于网关/框架/不可预期异常
- 业务成功以 `code == 0` 判断；错误码只能取自 spec，禁止自创
- 禁止向客户端暴露内部错误细节（SQL、堆栈、文件路径）
- Service 返回模块级 sentinel error，Handler 负责映射为错误码；仅 `10600` 用 Error 级日志，其它业务错误用 Warn

## 请求解码

- 请求体必须严格解码：禁止未知字段、拒绝尾随 JSON；禁止裸用标准库 `json.NewDecoder`
- 端点声明 Request 为 `{}` 时，必须校验空对象（缺失/非对象/含任意字段统一返回 `10000`）

## 认证与会话

认证与会话契约以 `spec/api_spec_auth.md` 为权威来源，本节仅列出实现侧约束：

- JWT 仅含 user_id 与不可变的 session_id，禁止写入可变字段
- Web 端通过 HttpOnly Cookie，App 端通过 Bearer Token（详见 spec）
- 鉴权分公开/可选（匿名降级）/必须（未登录拒绝）/管理员（额外 RBAC）四档
- CSRF 防护完全依赖 auth Cookie 的 `SameSite=Strict`，不使用 CSRF Token
- 失效机制为 session 撤销：登出/撤销时标记 session 失效，后续请求校验 session 状态即拒绝；不需要独立的 token 黑名单
- 刷新 Token 必须轮转，旧 token 进入 30 秒宽限期后拒绝（详见 spec 并发刷新策略）
- 封禁账号在鉴权层拦截并返回 `10201`（强制登出 + 封禁页）

## 幂等性

幂等契约以 `spec/api_spec_base.md` 为权威来源，本节仅列出实现侧约束：

- 需要实现幂等的端点以 spec 的清单为准（当前 10 个：moments/create、comments/create、groups/create、groups/admin/announcements/create、/admin/v1/announcements/create、share/create、private_messages/send、group_messages/send、auth/register、auth/guest）；不需要的端点不得强制要求 `Idempotency-Key`
- 服务端按 `(user_id + endpoint + key)` 去重，仅缓存成功响应（`code == 0`）

## 分页

- 禁止 OFFSET，必须使用 cursor/keyset 分页
- cursor 不透明，客户端只透传；编码/解码仅在 Repository 层
- 时间 keyset 的精度必须对齐：游标携带的时间精度不得粗于排序列的写入精度（如列以 `time.Now().UTC()` 全精度写入，游标必须用 `RFC3339Nano` 编码；秒截断游标配全精度列会导致同秒行在翻页中静默丢失，见 `moments/fetch_favorites` 修复先例）。新增时间 keyset 列表时，复核「写入精度 × 游标精度 × 谓词」三者一致
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
- 心跳参数由服务端下发，客户端不得硬编码
- 多实例部署时必须配置跨实例 fanout（RedisBridge），否则跨实例事件丢失

## 未读计数（聊天）

- 未读计数由客户端基于 WebSocket 增量维护
- 服务端仅提供冷启动校准（如历史消息基线）
- 服务端不维护独立的未读计数字段，通过已读游标（如 `last_read_message_id`）在查询时动态计算

## 日志与安全

- 日志必须携带 `X-Request-Id`，使用结构化日志，禁止 `fmt.Println`
- 禁止记录密码、Token、隐私内容；IP 在日志中可记录，但对外响应需脱敏
