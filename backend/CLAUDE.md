# CLAUDE.md

**本项目已暂停开发**，仅保留代码。当前所有后端开发在 `backend-go/` 进行。

## 构建 & 测试

- `cargo build` — 编译
- `cargo check` — 快速编译检查
- `cargo test` — 单元测试
- `cargo clippy` — lint
- `cargo fmt` — 格式化

## 约束

- **响应格式**：HTTP 200 + `ApiResponse<T>`，业务成功唯一依据 `code == 0`。
- **契约对齐**：修改接口前查阅对应 `spec/` 文档。
