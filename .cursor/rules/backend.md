---
globs: backend/**
alwaysApply: false
---

# 后端开发规则路由

编辑后端代码时，须同时遵守以下规范（按需阅读）：

- **实现规范**：`spec/backend_rule.md` -- Rust/Axum 编码约束、分层架构、SQL 规则
- **功能边界**：`spec/backend_doc.md` -- 后端提供的能力与非目标
- **Agent 操作指南**：`backend/AGENTS.md` -- 架构分层、SQL 规则、验证命令

新增 API 端点时的文件修改清单：

- `spec/api_spec_*.md` -- 确认端点定义存在
- `backend/src/dto/request/<module>.rs` -- 添加请求 DTO
- `backend/src/dto/response/<module>.rs` -- 添加响应 DTO
- `backend/src/handlers/<module>.rs` -- 添加 Handler
- `backend/src/services/<module>_service.rs` -- 添加 Service 方法
- `backend/src/repositories/<domain>/<module>_repo.rs` -- 添加 Repository 方法
- `backend/src/routes/v1/<module>.rs` -- 注册路由
- 如涉及新模块：更新对应 `mod.rs` 的 pub mod 声明
