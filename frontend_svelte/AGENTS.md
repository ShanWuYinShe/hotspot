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

```typescript
// ✅ 正确：只看 code === 0
const res = await post<UserData>(USERS.PROFILE, { user_id });
if (res.code === 0 && res.data) {
  // 使用 res.data
}

// ❌ 错误：依赖 HTTP 状态码或假设 data 一定有值
// ❌ 错误：依赖 res.message 做逻辑分支
```

### 分页

```typescript
// ✅ 正确：透传 cursor，不解析
const res = await post<PaginatedData<MomentItem>>(MOMENTS.LIST, {
  limit: 20,
  cursor: nextCursor, // 不透明，直接透传
});
// 下一页：使用 res.data.next_cursor
```

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

## 常见错误清单

- [ ] 是否硬编码了 API 路径 `"/api/v1/..."`？
- [ ] 是否依赖了 HTTP 状态码而非 `code === 0`？
- [ ] 是否使用了 `any` 类型？
- [ ] 是否在 localStorage 存了 Refresh Token？
- [ ] 是否解析了 JWT？
- [ ] 是否在前端计算了距离/角度？
- [ ] 是否遗漏了幂等接口的幂等键？
- [ ] cursor 是否做了解析而非透传？

## 安全规则

文件删除、批量变更、Git 操作规范等全局规则见根目录 `AGENTS.md` 的「权限边界」和「Git 操作规范」章节，此处不重复。

## 验证命令

```bash
# 类型检查
pnpm check

# 禁止模式扫描
rg '"/api/v1/' src/

# 构建
pnpm build
```
