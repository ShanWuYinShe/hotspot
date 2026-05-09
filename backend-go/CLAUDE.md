# CLAUDE.md

- 本文件只定义本子项目的技术约束与本地命令；仓库级 workflow、任务分级与验证边界以根 `CLAUDE.md` 为准。

## 构建 & 测试 & Lint
- `go build ./...` — 编译
- `go test -short ./...` — 单元测试（`-short` 跳过需要 DB 的集成测试）
- `go vet ./...` — 静态检查
- `gofmt -w .` — 格式化

## 约束
- **格式化**：修改代码后必须运行 `gofmt -w .`，具体规则由 Go 工具链定义。
- **分层**：Handler → Service → Repository。SQL 仅在 repo 层编写。
- **存储**：PostgreSQL 为权威存储，Redis 用于热点缓存。
- **分页**：使用 cursor / keyset，cursor 对上层保持不透明。
- **地理**：所有地理查询与计算统一走 `internal/platform/geo`（基于 S2）。
- **SQL 构造**：动态 SQL 通过白名单映射生成。
- **幂等**：复用平台层 `idempotency` 能力。
- **错误响应**：HTTP 200 + `ApiResponse<T>`，业务成功唯一依据 `code == 0`。
- **WebSocket 认证**：与 HTTP 共享同一套 auth 机制。
- **契约对齐**：修改 API、字段或错误码前先查阅 `spec/`。
