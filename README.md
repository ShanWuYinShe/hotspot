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

## 规范

- [AGENTS.md](AGENTS.md) - AI Agent 操作指南
- [spec/](spec/) - API 规范文档
- [deploy/](deploy/) - 部署配置
