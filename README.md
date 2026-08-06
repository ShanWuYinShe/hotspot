# HotSpot

基于地理位置的社交应用。围绕位置组织附近的人、群组与生活瞬间，提供发现、聊天、地图等能力。

## 功能

- **附近发现**：按位置浏览附近的人、群组、生活瞬间
- **社交**：关注、好友、私聊、群聊、评论、点赞、举报
- **地图**：基于地理位置的浏览与位置选择
- **搜索**：用户、群组、内容搜索
- **实时消息**：WebSocket 推送的聊天与通知
- **管理后台**：用户与内容管理

## 子项目

| 子项目 | 说明 | 技术栈 |
|--------|------|--------|
| `backend-go/` | Go 主后端（当前活跃） | Go / Chi / PostgreSQL / S2 |
| `frontend/` | Web 主前端 | SvelteKit / Svelte 5 / TypeScript / Tailwind CSS |
| `harmonyapp/` | HarmonyOS 客户端 | HarmonyOS ArkTS |
| `spec/` | API 契约（只读权威） | — |

## 快速开始

```bash
cp .env.example .env
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d
```

启动后访问 `http://localhost:8888`（nginx 网关统一反代前端、API 与 WebSocket）。

## 文档

- [AGENTS.md](AGENTS.md) — 协作规范
- [spec/](spec/) — API 契约
