# Frontend (SvelteKit) - Agent Guide

> SvelteKit / Svelte 5 / TypeScript Web 前端的 AI Agent 操作指南。

## 技术栈

- **框架**：SvelteKit 2.x + Svelte 5（使用 runes：`$state`、`$derived`、`$effect`）
- **构建**：Vite 7
- **样式**：Tailwind CSS
- **UI 组件库**：bits-ui + lucide-svelte
- **通知**：svelte-sonner
- **工具**：clsx、tailwind-merge、tailwind-variants、mode-watcher

## 目录结构

```
src/
├── app.html                  # HTML 模板
├── app.css                   # 全局样式
├── hooks.client.ts           # 客户端钩子（API 客户端 + WebSocket 初始化）
├── hooks.server.ts           # 服务端钩子
├── lib/
│   ├── api/
│   │   ├── client.ts         # 统一 HTTP 客户端（POST、Token 刷新、幂等键）
│   │   ├── endpoints.ts      # API 路径常量（唯一路径定义处）
│   │   └── services/         # 按业务域拆分的 API 服务
│   │       ├── index.ts      # 服务导出
│   │       ├── auth.ts
│   │       ├── moments.ts
│   │       └── ...
│   ├── components/
│   │   ├── common/           # 业务通用组件
│   │   ├── ui/               # bits-ui 基础组件
│   │   └── ...               # 其他按功能分组
│   ├── hooks/                # 复用逻辑钩子
│   ├── stores/               # Svelte 5 runes 状态管理
│   ├── types/
│   │   └── api.ts            # API 类型定义（对齐 spec）
│   ├── utils/                # 工具函数
│   └── websocket/            # WebSocket 连接与事件处理
│       ├── store.svelte.ts   # 连接管理、心跳、重连
│       ├── handlers.ts       # 事件处理与订阅
│       └── types.ts          # WebSocket 事件类型
└── routes/
    ├── (admin)/              # 管理端页面（需管理员权限）
    ├── (auth)/               # 认证页面（登录/注册）
    ├── (protected)/          # 需登录的页面
    └── (public)/             # 公开页面
```

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

## 状态管理

使用 Svelte 5 runes 管理状态：

| Store 文件 | 作用 |
|------------|------|
| `auth.svelte.ts` | 认证、多账户、Token、用户资料 |
| `notification.svelte.ts` | 通知（依赖 WebSocket） |
| `message.svelte.ts` | 未读消息计数（依赖 WebSocket） |
| `presence.svelte.ts` | 用户在线状态 |
| `location.svelte.ts` | 当前位置 |
| `toast.svelte.ts` | 全局 Toast |
| `ui.svelte.ts` | UI 状态 |

### 状态规则

- Token 存储：Web 端只用 HttpOnly Cookie，**禁止在 localStorage/sessionStorage 存 Refresh Token**。
- 认证持久化：`localStorage`（key: `hotspot_auth`）。
- **禁止解析 JWT**，登录态只按业务接口返回值判断。

## TypeScript 规则

- **禁止 `any`**，使用 `unknown` + 类型守卫替代。
- 访问可选字段必须用 `?.` / `??`。
- 类型定义应与 `spec/api_spec_types.mdc` 对齐。

## WebSocket

- 在 `hooks.client.ts` 中初始化，随认证状态自动连接/断开。
- 事件类型必须与 `spec/api_spec_websocket.mdc` 一致。
- 可做「镜像类型」，不扩展 spec 中未定义的事件。

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

## 文件删除与批量变更规则

- Agent **不得自行删除**任何源代码文件。如需删除，须列出清单并说明原因，经用户确认后执行。
- **批量修改**：当一次操作可能涉及**大量文件的修改、重命名或重构**时，Agent 必须先向用户展示完整的影响范围（受影响文件清单、变更摘要），经用户确认后才可执行。**禁止静默批量操作，必须给用户一个询问提示。**
- **禁止的危险 Shell 命令**（可能导致源代码丢失）：`rm`、`rm -rf`、`mv`（批量）、`find ... -delete`、`xargs rm`。除非用户明确授权，否则一律禁止。

## Git 安全（极严格）

> **所有 Git 写操作默认禁止，防止覆盖用户未提交的更改。**

- **允许的只读查询**：`git status`、`git diff`、`git log`、`git show`、`git ls-files`。
- **严格禁止的命令**（包括但不限于）：
  `git checkout`, `git restore`, `git reset`, `git clean`, `git stash`,
  `git add`, `git commit`, `git push`, `git pull`, `git merge`,
  `git rebase`, `git cherry-pick`, `git revert`
- **无例外**：上述命令除非用户在当前对话中**逐条明确指示**执行，否则一律禁止。
- **危险参数禁令**：禁止使用 `--force`、`--hard`、`--no-verify` 等可能绕过检查的参数。
- **子 Agent 同样受约束**：在调用子 Agent 时，必须在 prompt 中明确禁止其执行任何 Git 写操作。

## 验证命令

```bash
# 类型检查
pnpm check

# 禁止模式扫描
rg '"/api/v1/' src/

# 构建
pnpm build
```
