# CLAUDE.md

本文件只定义本子项目的技术约束与本地命令；仓库级 workflow 以根 `CLAUDE.md` 为准。

## 构建 & 测试

- `go build ./...` — 编译
- `go test -short ./...` — 单元测试（`-short` 跳过需要 DB 的集成测试）
- `go vet ./...` — 静态检查
- `gofmt -w .` — 格式化

## 约束

- **模块架构**：每个功能模块在 `internal/modules/{模块名}/` 下，实现 `modulekit.Module` 接口的 `Mount(r chi.Router)` 方法，然后在 `internal/bootstrap/module_registry.go` 的 `BuildModules()` 中注册。
- **模块目录结构**：`module.go`（注册路由）、`handler.go`（HTTP handler）、`service.go`（业务逻辑）、`repo.go`（接口定义）、`types.go`（DTO）、`repo_postgres.go` / `repo_memory.go`（实现）。
- **数据库迁移**：`migrations/` 下 `NNN_描述.sql` 格式，通过 `embed.go` 嵌入二进制，由 `internal/platform/migrate` 自动执行。新增 migration 使用下一个可用序号。
- **地理查询**：所有地理查询与计算统一走 `internal/platform/geo`（基于 S2）。
- **SQL 构造**：动态 SQL 通过白名单映射生成。
- **幂等**：复用平台层 `idempotency` 能力。
- **响应格式**：HTTP 200 + `ApiResponse<T>`，业务成功唯一依据 `code == 0`。
- **WebSocket 认证**：与 HTTP 共享同一套 auth 机制。
