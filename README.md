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

```bash
# 后端
cd backend && cargo run

# 前端
cd frontend_svelte && pnpm dev
```

## 入口

- 协作与实现说明先读 [CLAUDE.md](CLAUDE.md)，进入子项目后再读对应目录下的 `CLAUDE.md`。
- [spec/](spec/) - API 规范
- [deploy/](deploy/) - 本地验证与运行入口
