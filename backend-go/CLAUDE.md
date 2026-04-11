# CLAUDE.md

## 架构
模块化单体 Go 后端，使用 Chi 路由。

- `cmd/server/main.go`：入口
- `internal/bootstrap/`：依赖装配、路由注册
- `internal/platform/`：平台能力（auth、httpx、persistence、cache、realtime、geo、observability）
- `internal/modules/`：业务模块，内部维持 `handler -> service -> repo` 分层
- `migrations/`：数据库迁移

## 约束
- **分层**：Handler → Service → Repository。SQL 仅在 repo 层编写。
- **存储**：PostgreSQL 为权威存储，Redis 用于热点缓存。
- **分页**：使用 cursor / keyset，cursor 对上层保持不透明。
- **地理**：所有地理查询与计算统一走 `internal/platform/geo`（基于 S2）。
- **SQL 构造**：动态 SQL 通过白名单映射生成。
- **幂等**：复用平台层 `idempotency` 能力。
- **错误响应**：HTTP 200 + `ApiResponse<T>`，业务成功唯一依据 `code == 0`。
- **WebSocket 认证**：与 HTTP 共享同一套 auth 机制。
- **契约对齐**：修改 API、字段或错误码前先查阅 `spec/`。
