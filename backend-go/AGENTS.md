# HotSpot Backend (Go)

## 功能边界

Go **主后端**,模块化单体 + S2 地理索引。提供认证/用户/群组/聊天/瞬间/Feed/搜索/地图/分享/媒体/举报/通知/系统/管理后台等全部 HTTP API 与 WebSocket 实时能力。支付/交易、端侧离线同步、消息全文搜索不在范围内。

> 历史 `backend/`（Rust/Axum）已暂停，不在本规范约束范围内。

API 契约见 `spec/api_spec_*.md`（仓库根 `spec/` 目录，只读权威）。

## 构建 & 测试

- `go build ./...` — 编译
- `go test -short ./...` — 单元测试（`-short` 跳过需要 DB 的集成测试）
- `go vet ./...` — 静态检查
- `gofmt -w .` — 格式化

## Dev 容器（热加载）

- dev 模式后端跑 `Dockerfile.dev`（基于 `golang:1.26.2-alpine` + air），源码挂载 + air 监听 `.go`/`.toml` 变化自动重编译重启，**改 Go 代码无需重启容器**；`.air.toml` 配置编译/输出路径（`tmp/` 由 compose 卷隔离）。
- 首次构建/重建镜像：`docker compose -f docker-compose.yml -f docker-compose.dev.yml build backend`（`GOPROXY` 默认 `goproxy.cn,direct`，依赖经 BuildKit cache mount 缓存）。
- 注意：`Dockerfile.dev` 的 `ARG GO_VERSION`/`AIR_VERSION` 必须带精确 patch 版本（如 `1.26.2`），浮动 tag（`1.26-alpine`）拉取易超时。

## 技术栈（名称级）

- Go（见 `go.mod`）、chi router、PostgreSQL + pgx/v5、Redis（限流/幂等缓存/WS 跨实例 fanout，可选）、google/s2（地理网格索引）、gorilla/websocket、zap（结构化日志）

## 目录结构（强制）

```
backend-go/
├── cmd/                    # 入口（server / migrate）
├── migrations/             # SQL 迁移（embed 进二进制）
└── internal/
    ├── bootstrap/          # 装配：路由聚合、模块注册、worker、WS 鉴权
    ├── modules/            # 业务模块（每模块一个 package）
    └── platform/           # 横切基础设施（auth/realtime/httpx/...）
```

**规则**：
- `internal/modules/*` 只能依赖 `internal/platform/*` 和同层其它模块的 Service，禁止反向依赖 `bootstrap`
- `internal/platform/*` 不含业务判断，只提供可复用基础设施
- 业务模块之间通过 **Service 接口**解耦（如 `groups` 注入 `users.Service`），不直接 import 对方 repo

## 架构分层（强制）

```
Handler -> Service -> Repository
```

- **Handler**（`handler*.go`）：仅做参数提取/校验、调用 Service、返回 `httpx.ApiResponse`。不写业务逻辑、不直接查库
- **Service**（`service*.go`）：业务规则/权限/事务/事件发布，不写 SQL
- **Repository**（`repo*.go`）：只封装 SQL 与行映射，不含业务判断

**每模块的文件组织约定**：
- `module.go`：`Mount(r chi.Router)` 挂载路由；在 `internal/bootstrap/module_registry.go` 的 `BuildModules()` 中注册（实现 `modulekit.Module`）
- `handler.go`（必要时 `handler_xxx.go` 按子域拆分）
- `service.go`（必要时 `service_xxx.go`）
- `repo.go` + `repo_postgres.go` + `repo_memory.go`（内存实现用于测试）
- `types.go`：请求/响应/内部模型（DTO）
- `errors.go`：模块级 sentinel error

## 数据库迁移

`migrations/` 下 `NNN_描述.sql` 格式，通过 `embed.go` 嵌入二进制，由 `internal/platform/migrate` 自动执行。新增 migration 使用下一个可用序号。

## 路由与协议

- 仅 POST，RPC + JSON
- 所有 API 必须是明确动作子路径，禁止裸资源路径（如禁止 `/api/v1/users`，须使用 `/api/v1/users/search` 等动作路径）
- 路由通过 chi 的 `r.Route("/api/v1/<ns>", ...)` + `group.Post("/<action>", handler)` 注册
- **认证分组**：每模块用两个 `chi.Router.Group` 区分——公开/可选组挂 `platformauth.OptionalAuth`，鉴权组挂 `platformauth.RequiredAuth`；`/admin/*` 额外挂 `platformauth.AdminAuth`
- **统一中间件链**（顺序）：`OptionalAuth|RequiredAuth` → `ratelimit.Middleware` → `resourceguard.Middleware` → `idempotency.Middleware`

## 统一响应与错误处理

- 所有业务响应通过 `httpx.WriteJSON(w, http.StatusOK, ...)` 返回 `httpx.ApiResponse[T]`
- 业务层必须返回 HTTP 200 + `ApiResponse<t>`；HTTP 非 200 仅用于网关/框架/不可预期异常
- 业务成功仅以 `code == 0` 判断（`httpx.Success(data)` / `httpx.SuccessEmpty()`）
- 错误码只能取自 `spec/api_spec_errors.md`，禁止自创
- 禁止向客户端暴露内部错误细节

**错误传播模式**：
- Service 返回模块级 sentinel error（如 `ErrGroupNotFound`、`ErrAccessRestricted`）
- Handler 负责把 sentinel error 映射为 `httpx.Error(code, message)`
- 仅 `10600`（服务不可用）才使用 `observability.L().Error` 记录；其它业务错误用 `Warn`

## 请求解码（强制）

- 使用 `httpx.DecodeStrictJSON(r, &req)`：禁止未知字段、拒绝尾随 JSON
- 端点声明 Request 为 `{}` 时，必须用 `httpx.RequireEmptyObject(r)` 校验空对象（缺失/非对象/含任意字段统一返回 `10000`）
- 禁止使用标准库 `json.NewDecoder` 裸解码

## 认证与会话

- **JWT 仅包含 user_id**，禁止写入可变字段或会话标识
- Web 端通过 HttpOnly Cookie（`access_token` / `refresh_token`）；App 端通过 `Authorization: Bearer`
- 鉴权中间件：`platformauth.OptionalAuth` / `RequiredAuth` / `AdminAuth`
- Principal 通过 `platformauth.PrincipalFromContext(ctx)` 获取
- **CSRF 防护**：完全依赖 auth Cookie 的 `SameSite=Strict` 属性，不使用 CSRF Token（Double Submit Cookie 已移除，因 SameSite=Strict 已覆盖经典 CSRF 威胁）
- **失效机制**：登出/撤销写入黑名单，校验时必须检查黑名单
- 黑名单条目 TTL 不得短于 Token 过期时间
- 刷新 Token 必须轮转；旧 refresh_token 进入宽限期（30 秒内仍可使用，返回与首次刷新相同的新 token 对），宽限期后加入黑名单（详见 `spec/api_spec_auth.md` 并发刷新策略）

## 幂等性

- 必须支持 `Idempotency-Key` 的接口以 `spec/api_spec_base.md`「必须使用幂等键的接口（MUST）」列表为准
- `idempotency.Middleware` 解析 header（正则 `^[A-Za-z0-9_-]{32,64}$`）并注入 context；真正的回放逻辑在 Handler 内用 `idempotency.ReplayOrStore(w, r, "<endpoint>", fn)` 完成（复用平台层 `idempotency` 能力）
- 服务端按 `(user_id + endpoint + key)` 去重
- 仅缓存 `code == 0` 的成功响应

**Handler 幂等模板**：
```go
if err := idempotency.RequireKey(r); err != nil { /* 10000 */ }
// ... 解码 + 校验 ...
idempotency.ReplayOrStore(w, r, "groups/create", func() httpx.ApiResponse[XxxResponse] {
    // 业务调用；成功 return httpx.Success(...)
})
```

## 分页（Cursor/Keyset）

- 禁止 OFFSET
- cursor 不透明，客户端只透传；服务端用 `internal/platform/cursor`（AES-GCM 加密 + base64url）编码/解码
- 编码/解码仅在 Repository 层，namespace 按模块隔离（如 `"groups"`）
- limit 默认 20、上限 100，由 Service 层兜底裁剪
- 无更多数据时省略 `next_cursor` 字段

## 数据格式（契约级）

- 所有时间字段使用 UTC，序列化为 RFC3339（`time.Time.UTC().Format(time.RFC3339)`）
- 所有 `xxx_id` 字段在 API 层统一为 `string`，禁止使用裸 `id`
- DB 主键可为数值，但对外必须字符串化（如 `fmt.Sprintf("group_%d", id)`）

## 地理位置计算

- 所有地理查询与计算统一走 `internal/platform/geo`（基于 S2）：`CellTokenAtLevel` / `LeafCellID`
- 距离/范围/排序必须在 PostgreSQL 完成（PostGIS 或 ST_DWithin 等）
- 禁止在 Go 层计算用于排序的距离或角度
- 地理字段必须建立 GIST 索引

## SQL 构造

动态 SQL 通过白名单映射生成。

## 配置（AppConfig）

- `internal/platform/appconfig` 提供 `Source` 接口，从 `AppConfig`（DB 持久化）读取运行参数
- `RateLimitConfig` / `ResponseDefaults` / `AppLimits` 等变更即时生效，无需重启
- 模块通过 `appconfig.Source` 注入，禁止直接读环境变量或硬编码阈值

## 持久化（Repository 双实现）

- 每模块 Repository 至少提供两套实现：
  - `repo_postgres.go`：生产实现
  - `repo_memory.go`：内存实现，用于单测与无 DB 场景
- 通过 `NewPersistenceRepository(factory)` 按配置选择；DB 未配置时优雅降级到内存实现
- SQL 写在 `repo_postgres_queries.go`，保持查询与映射分离

## WebSocket

- 与 HTTP 共享同一套 auth 机制；事件与订阅格式严格遵循 `spec/api_spec_websocket.md`
- 实现位于 `internal/platform/realtime`：`Handler`（连接与协议）、`Hub`（单实例路由）、`RedisBridge`（跨实例 fanout，可选）
- 连接建立后 5 秒内必须收到 `Auth`，否则发送 `Error(code=20000, severity=fatal)` 并断开
- 心跳参数由服务端通过 `ConnectionEstablished.config` 下发，客户端不得硬编码
- 群组解散等需要通知所有群成员的事件，通过 `realtime.DefaultHub().BroadcastToGroup(groupID, event)` 实现 fanout
- 多实例部署时必须配置 `RedisBridge`，否则跨实例事件丢失

## 未读计数（聊天）

- 未读计数由客户端基于 WebSocket 增量维护
- 服务端仅提供冷启动校准（如历史消息基线）
- 服务端不维护独立的未读计数字段，通过已读游标（如 `last_read_message_id`）在查询时动态计算

## 日志与安全

- 日志必须携带 `X-Request-Id`（`observability.RequestIDFromContext`）
- 使用 zap 结构化日志，禁止 `fmt.Println`
- 禁止记录密码、Token、隐私内容；IP 在日志中可记录，但对外响应需脱敏
