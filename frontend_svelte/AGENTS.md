# Frontend (SvelteKit) - Agent Guide

> SvelteKit / Svelte 5 / TypeScript Web 前端的 AI Agent 操作指南。

## 使用范围

本文件只补充 Web 前端特有的硬约束（API 调用、类型、安全等），其他页面结构和组件划分请 Agent 通过代码与 `spec/` 自行探索。

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
- **禁止在前端计算距离或角度**，一律由后端完成（具体实现以 `spec/` 和后端代码为准）。
- 地图上的 marker 与中心点（包括位置选择器的 center pin）必须遵守根目录 `AGENTS.md` 中「位置与距离 / 地图 SDK 与中心点」小节的全局约束，不在本文件重复描述实现细节。

## 额外禁止

除根目录 `AGENTS.md` 中的全局禁止项外，前端额外禁止：
- 直接在代码中写 `"/api/v1/..."`，或绕过 `endpoints.ts` / `post<T>()` 自行封装 HTTP。
- 使用 `any` 类型、在本地存储 Refresh Token、解析 JWT 判断登录态。
- 在前端计算距离/角度，或对游标做任何解析/构造（只能透传）。

## 安全规则

文件删除、批量变更、Git 操作规范等全局规则见根目录 `AGENTS.md` 的「权限边界」和「Git 操作规范」章节，此处不重复。

> **⚠️ `tmp/` 路径规则**：所有临时文件必须存放在项目根目录的 `tmp/` 下。详见根目录 `AGENTS.md` 的「Agent 行为准则」章节。

## 验证命令

```bash
# 类型检查
pnpm check

# 禁止模式扫描
rg '"/api/v1/' src/

# 构建
pnpm build
```
