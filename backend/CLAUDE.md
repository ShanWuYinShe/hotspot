# CLAUDE.md

- 本文件只定义本子项目的技术约束与本地命令；仓库级 workflow、任务分级与验证边界以根 `CLAUDE.md` 为准。

## 构建 & 测试 & Lint
- `cargo test` — 单元测试
- `cargo clippy` — lint
- `cargo fmt` — 格式化

## 约束
- **分层**：Handler → Service → Repository，不跨层访问。
- **HTTP 方法**：统一 POST + JSON。
- **响应格式**：`HTTP 200 + ApiResponse<T>`，业务成功唯一依据 `code == 0`。
- **SQL**：使用 `sqlx` 宏编写查询。
- **分页**：使用 cursor / keyset，cursor 编解码在 repo 层完成。
- **地理计算**：距离和空间排序在 PostGIS 层完成。
- **日志**：携带 request id，敏感字段（密码、Token）不写入日志。
- **契约对齐**：修改接口前查阅对应 `spec/` 文档。
