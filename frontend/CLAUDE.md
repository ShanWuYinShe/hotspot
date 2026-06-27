# CLAUDE.md

> **注意**：本项目为 **主前端**（SvelteKit），当前所有前端开发在此进行，**同时是 UI/UX 与设计系统的唯一事实源**。改前端样式前先阅读 `.claude/rules/` 下的设计规范。

## 功能边界

SvelteKit **主前端**,UI/UX 与设计系统唯一事实源。覆盖认证/发现与搜索/信息流/生活瞬间与评论/媒体上传/聊天(私聊+群聊)/群组/通知与公告/举报/分享/地图/个人中心/管理后台。支付与交易、端侧离线同步不在范围内。

API 契约见 `spec/api_spec_*.md`（仓库根 `spec/` 目录，只读权威）。

## 构建 & 测试

- `bun run build` — 构建
- `bun check` — 类型检查（svelte-check）
- `bun lint` — 格式检查（prettier --check）
- `bun run format` — 格式化（prettier --write）

> **测试**：package.json 无 `test` 脚本，暂无自动化测试。

## 技术栈

- SvelteKit（Svelte 5）
- TypeScript（严格模式）
- Vite

## 目录与分层（必须）

- `src/routes/`：页面与路由（`+page.svelte` / `+page.ts` / `+layout.svelte`）
- `src/routes/(public)/(protected)/(admin)/(auth)`：按访问级别分组
- `src/lib/api/`：HTTP 客户端与服务层
  - `client.ts`：统一请求封装
  - `endpoints.ts`：API 路径常量
  - `services/*.ts`：业务 API 调用
- `src/lib/stores/`：全局状态（Svelte stores）
- `src/lib/components/`：可复用组件
- `src/lib/hooks/`：可复用逻辑封装
- `src/lib/attachments/`：Svelte actions

## API 调用规范

- **禁止硬编码路径**：必须通过 `src/lib/api/endpoints.ts`
- **统一请求封装**：所有请求走 `client.ts` 和 `services/*`
- **HTTP 方法**：统一 POST（RPC + JSON，不实现 REST 风格）
- **成功判定**：仅使用 `ApiResponse.code === 0`
- **HTTP 层错误**：通过 try/catch 处理网络/超时/非 200（见通用契约）
- **仅在 HTTP 200 时解析**：非 200 直接视为请求失败，不解析 `ApiResponse`
- **Cursor 不透明**：请求时透传 `cursor`，收到响应后将 `next_cursor` 作为下一页的 `cursor` 透传，不解析、不构造

## Token 与安全

- Web 端仅使用 HttpOnly Cookie，禁止在 localStorage/sessionStorage 存储 Refresh Token
- App 端使用 `Authorization: Bearer`（如有跨端场景）
- 前端不解析 JWT，只按业务接口返回值处理登录态

## WebSocket 规范

- 事件名与 payload 结构以 `spec/api_spec_websocket.md` 为准
- 可在前端定义**镜像类型**（仅映射契约，不得扩展）
- 消息最终态以 WebSocket 推送为准

## 位置与距离

- 位置请求使用 `Location`，响应使用 `RelativeLocation`
- 禁止前端自行计算距离/角度（由后端 PostGIS 完成）

## 数据格式（契约级）

- 所有时间字段为 UTC ISO 8601，前端负责时区转换
- 所有 `xxx_id` 字段按 `string` 处理，不做数值运算

## TypeScript 约束

- 禁止 `any`
- 访问可选字段必须使用 `?.`/`??`

## UI/UX 设计规范

见 `.claude/rules/` 下按文件类型懒加载的规则：

| 文件               | 适用范围                                  | 内容                                                                                                                     |
| ------------------ | ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `ui-layout.md`     | `src/**/*.svelte`                         | 页面布局系统（`.page-shell`、**容器查询原语** `.density-grid`/`.sidebar-layout`/`.switcher`、**platformStore 语义化派生量**、红线） |
| `design-tokens.md` | `src/**/*.{svelte,css}`                   | 设计 token（颜色、表面质感、圆角、字号、间距、动效、图标、**对比度/WCAG**）                                                               |
| `navigation.md`    | `src/**/*.svelte`, `config/navigation.ts` | 导航框架（设备导航模型、返回按钮、入口去重、路由收敛、审查清单）                                                         |

**响应式适配原则**：组件层用容器查询原语（`.density-grid` 等，组件按自身容器宽度自适应），视口层判断用语义化派生量（`useMasterDetail`/`useSidebarChrome`/`hideRedundantEntry`）。业务页禁止裸用 `isMobile/isTablet/isDesktop/isWide` 切换纯布局密度。

改代码前花 10 秒看对应规则，避免不一致。
