# CLAUDE.md

## 架构
Rust 主后端，基于 Axum 框架。

- `src/main.rs`：入口，装配 AppState、路由、tracing
- `src/routes/v1/`：按业务域拆分路由
- `src/handlers/`：HTTP 边界，参数提取，调用 service
- `src/services/`：业务逻辑、事务、事件发布
- `src/repositories/`：SQLx 仓库层
- `src/events/`、`src/websocket/`：事件总线与实时推送

## 约束
- **分层**：Handler → Service → Repository，不跨层访问。
- **HTTP 方法**：统一 POST + JSON。
- **响应格式**：`HTTP 200 + ApiResponse<T>`，业务成功唯一依据 `code == 0`。
- **SQL**：使用 `sqlx` 宏编写查询。
- **分页**：使用 cursor / keyset，cursor 编解码在 repo 层完成。
- **地理计算**：距离和空间排序在 PostGIS 层完成。
- **日志**：携带 request id，敏感字段（密码、Token）不写入日志。
- **契约对齐**：修改接口前查阅对应 `spec/` 文档。
