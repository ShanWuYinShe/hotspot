# HotSpot

基于地理位置的社交应用。

## 技术栈

| 子项目 | 技术栈 |
|--------|--------|
| `backend/` | Rust / Axum / SQLx / PostGIS |
| `backend-go/` | Go / Chi / pgx / PostgreSQL / S2 |
| `frontend_svelte/` | SvelteKit / Svelte 5 / TypeScript |
| `harmonyapp/` | HarmonyOS ArkTS |
| `deploy/` | Docker / Nginx |
| `spec/` | API 规范 |

## 开发

格式化 / 语法检查 / 编译通过受控脚本执行（必须在 feature worktree 中）：

```bash
# Rust 后端
python3 .claude/scripts/rust_dev.py --task all

# Go 后端
python3 .claude/scripts/go_dev.py --task all

# Web 前端
python3 .claude/scripts/frontend_dev.py --task all

# Harmony 客户端
python3 .claude/scripts/harmony_worktree.py --task all
```

测试与最终验证通过 `deploy/` 隔离执行：

```bash
./deploy/scripts/verify.sh --env-name <name>
```

## 入口

- 协作与实现说明先读 [CLAUDE.md](CLAUDE.md)，进入子项目后再读对应目录下的 `CLAUDE.md`。
- [spec/](spec/) - API 规范
- [deploy/](deploy/) - 本地验证与运行入口
