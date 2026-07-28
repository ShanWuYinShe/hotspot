# HotSpot 前端（React / Vite / TypeScript）

> 本文件是前端专属约束。根 `AGENTS.md` 中的通用原则（Git 规范、Commit 格式、Docker 等）仍然适用。

## 功能边界

React Web **主前端**。认证/注册/发现与搜索/信息流/生活瞬间与评论/群组/聊天/附近/用户页面/个人中心/地图/管理后台。用户管理、支付与交易不在范围。

API 契约见 `spec/api_spec_*.md`（`spec/` 目录只读权威）。

## 开发模式：必须通过 Docker Compose + Nginx 网关

前端是一个纯 UI 层，所有 API、WebSocket、S3 请求都依赖后端服务。**禁止**直接在宿主机启动 Vite dev server。

### 正确流程

```bash
# 1. 启动全部服务（含前端 Vite HMR + 后端 + DB + nginx）
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# 2. 唯一访问入口
open http://localhost:8888
```

Nginx 网关（`docker/nginx.dev.conf`）统一反代：
| 路径 | 上游 |
|------|------|
| `/` | Vite HMR（容器内 `frontend:5173`，不暴露到宿主机） |
| `/api/` | Go 后端（容器内 `backend:8081`） |
| `/ws/` | Go 后端 WebSocket |
| `/s3/` | SeaweedFS S3 |

### 为什么不能直接 `bun run dev`

- `vite.config.ts` **没有** `server.proxy` — API 请求无法到达后端
- 环境变量（`VITE_AMAP_*`）由 Docker Compose 注入，宿主机缺失
- 高德地图安全代理不走 nginx 会被 CSP 拦截
- 前端 `vite.config.ts` 的 `allowedHosts` 配置假设通过 nginx/orbstack 域名访问

### 宿主机上允许的调试操作

在 Docker 容器已运行的前提下，以下操作可辅助但不替代通过 nginx 的完整验证：
- 对本地文件做纯 UI 浏览/样式调试时可用 `bun run dev`，但必须清楚 API 不可用
- 类型检查：`bun run check`
- 构建验证：`bun run build`

## 构建

```bash
bun install --cwd frontend
bun run --cwd frontend build
```

## ⚠️ 硬约束

- **调试/开发/预览时，默认入口是 `http://localhost:8888`（nginx 网关），不是 Vite dev server 地址**
- 如果确实需要在宿主机关闭（docker 未运行）时查看纯 UI，使用 `bun run build` 然后用静态服务器预览，不要在 dev 模式下假装有后端
- 修改 nginx 配置后必须重启对应容器：`docker compose -f docker-compose.yml -f docker-compose.dev.yml restart nginx`
