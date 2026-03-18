# HotSpot Backend

基于 Rust 的高性能、分布式社交应用后端服务。

## 技术栈

- **语言**：Rust
- **Web 框架**：Axum
- **运行时**：Tokio
- **数据库**：PostgreSQL + PostGIS
- **缓存**：Redis + Moka
- **认证**：JWT + bcrypt
- **存储**：S3/MinIO
- **日志**：tracing

## 架构分层

```
Handler → Service → Repository
```

## 验证命令

```bash
cargo check
cargo clippy
cargo test
```

## 规范与文档

- 完整规范见 [AGENTS.md](AGENTS.md)
- 部署配置见 [deploy/](../deploy/)
- API 规范见 [spec/](../spec/)
