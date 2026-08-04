# HotSpot Frontend

HotSpot 的 Web 前端，基于 SvelteKit 的纯 UI 层，所有数据来自后端 API。

## 功能

- 认证与注册
- 附近的人、群组、生活瞬间
- 信息流
- 私聊与群聊
- 地图浏览
- 发现与搜索
- 个人中心
- 管理后台（待建）

## 运行

前端必须与后端服务一起通过 Docker Compose 运行（API、WebSocket、S3 均经 nginx 网关，与生产一致）：

```bash
# From the repository root:
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# Then open: http://localhost:8888
```

**Do not** run `bun run dev` directly on the host and attempt to access the Vite dev server via its port — API calls will fail because they must go through the nginx gateway. See `AGENTS.md` in this directory for details.
