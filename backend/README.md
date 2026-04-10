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

## 验证与协作

- 仓库级 workflow、skill 入口和风险边界以根目录 [CLAUDE.md](../CLAUDE.md) 为准。
- 普通分析、代码阅读、局部低副作用开发动作通常可以直接完成。
- 最终验证与隔离运行优先走 `deploy/` 标准入口：

```bash
./deploy/scripts/verify.sh --env-name <name>
```

- 若需保存日志片段、验证截图或中间分析文件，默认写到当前 worktree 的 `tmp/`。

## 规范与文档

- 完整协作规范见 [CLAUDE.md](CLAUDE.md)
- 部署配置见 [deploy/](../deploy/)
- API 规范见 [spec/](../spec/)
