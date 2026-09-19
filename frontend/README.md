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
- 管理后台（users/groups/reports/tags/announcements/rbac/audit/config）

## 运行

前端必须与后端服务一起通过 Docker Compose 运行（API、WebSocket、S3 均经 nginx 网关，与生产一致）：

```bash
# From the repository root:
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# Then open: http://localhost:8888
```

**Do not** run `bun run dev` directly on the host and attempt to access the Vite dev server via its port — API calls will fail because they must go through the nginx gateway. See `AGENTS.md` in this directory for details.

## 浏览器巡检

前置：dev 环境已启动、宿主机已安装 `agent-browser`。

```bash
bun run browser:qa         # 桌面 20 页
bun run browser:qa:mobile  # 桌面 + 移动 5 页（390×844）
```

自动以访客身份巡检全部主要页面（横向溢出 / 坏图 / h1 / TabBar / 控制台错误），
截图与报告输出到仓库根 `.workwork/browser-check/qa-<时间戳>/`。
