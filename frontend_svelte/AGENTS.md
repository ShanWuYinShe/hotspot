# Frontend (SvelteKit) - Agent Guide

> SvelteKit / Svelte 5 / TypeScript Web 前端的 AI Agent 操作指南。

## 技术栈

- **框架**：SvelteKit + Svelte 5（使用 runes：`$state`、`$derived`、`$effect`）
- **构建**：Vite
- **样式**：Tailwind CSS
- **UI 组件库**：bits-ui + lucide-svelte
- **通知**：svelte-sonner

> 具体版本以 `package.json` 为准。

## API 调用规则（严格执行）

### 路径管理

**所有 API 路径必须定义在 `src/lib/api/endpoints.ts` 中。**

```typescript
// ✅ 正确：使用 endpoints 常量
import { AUTH } from '$lib/api/endpoints';
const result = await post<AuthData>(AUTH.LOGIN, data);

// ❌ 禁止：硬编码路径
const result = await post<AuthData>('/api/v1/auth/login', data);
```

**禁止模式**（会被 rg 扫描拦截）：
- 代码中出现 `"/api/v1/` 字符串。

### HTTP 客户端

- 统一使用 `src/lib/api/client.ts` 中的 `post<T>()` 函数。
- 所有请求自动携带 `credentials: 'include'`。
- 401 时自动尝试 refresh token 并重试。
- 幂等接口自动注入 `Idempotency-Key`。

### 响应处理

- **业务成功判断唯一依据**：`code === 0`，禁止依赖 HTTP 状态码或 `res.message`。

### 分页

- 透传 `cursor`，不解析、不构造。下一页使用 `res.data.next_cursor`。

## 状态规则

- Token 存储：Web 端只用 HttpOnly Cookie，**禁止在 localStorage/sessionStorage 存 Refresh Token**。
- 认证持久化：`localStorage`（key: `hotspot_auth`）。
- **禁止解析 JWT**，登录态只按业务接口返回值判断。

## TypeScript 规则

- **禁止 `any`**，使用 `unknown` + 类型守卫替代。
- 访问可选字段必须用 `?.` / `??`。
- 类型定义应与 `spec/api_spec_types.md` 对齐。

## WebSocket

- 事件类型必须与 `spec/api_spec_websocket.md` 一致，禁止扩展 spec 未定义的事件。

## 位置与距离

- 请求参数使用 `Location`（point + location_name）。
- 响应使用 `RelativeLocation`（含 distance、bearing_degrees）。
- **禁止在前端计算距离或角度**，由后端 PostGIS 完成。
- 所有地图上的定位点 / 标记（marker）必须使用地图 SDK 提供的原生 marker/overlay 来渲染，**禁止在地图容器上用绝对定位的 DOM 自己绘制 marker**。位置选择页面（如 `location-picker-modal` 等）中的选点标记必须跟随地图坐标系的平移与缩放，否则会在缩放时产生位置偏差，这是前端实现必须避免的根本性错误。

## 实现检查清单

- [ ] 架构一致性：有无类似页面/组件？遵循相同结构？
- [ ] API 客户端复用：`src/lib/api/services/` 中有现有方法？
- [ ] 状态管理复用：使用现有 Svelte stores 而非新方式？
- [ ] 组件复用：`src/lib/components/` 中有可用组件？
- [ ] 类型复用：对齐 `spec/api_spec_types.md`？复用 `src/lib/types/api.ts`？
- [ ] 错误处理复用：复用现有错误处理模式？

**禁止**：硬编码 API 路径、绕过统一层、重复实现、引入不一致状态管理

## 常见错误

- [ ] 硬编码 API 路径 `"/api/v1/..."`？
- [ ] 依赖 HTTP 状态码而非 `code === 0`？
- [ ] 使用 `any` 类型？
- [ ] localStorage 存 Refresh Token？
- [ ] 解析 JWT？
- [ ] 前端计算距离/角度？
- [ ] 遗漏幂等键？
- [ ] cursor 解析而非透传？
- [ ] 绕过统一层或重复实现？

## 安全规则

文件删除、批量变更、Git 操作规范等全局规则见根目录 `AGENTS.md` 的「权限边界」和「Git 操作规范」章节，此处不重复。

> **⚠️ `tmp/` 路径规则（强制）**：所有临时文件、会话文件（`session-context-*.md`、`session-progress-*.md`）、设计文档等必须存放在**项目根目录（Monorepo 根目录）的 `tmp/`** 下，即 `frontend_svelte/` 的上级目录中的 `tmp/`。**绝对禁止在 `frontend_svelte/` 目录下创建 `tmp/` 目录**（如 `frontend_svelte/tmp/`）。详见根目录 `AGENTS.md` 的「Agent 行为准则」章节。

## 验证命令

```bash
# 类型检查
pnpm check

# 禁止模式扫描
rg '"/api/v1/' src/

# 构建
pnpm build
```
