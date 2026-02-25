# HotSpot Frontend (SvelteKit)

基于 SvelteKit + Svelte 5 + TypeScript 的 Web 前端应用。

## 技术栈

- **框架**: SvelteKit 2.x + Svelte 5 (Runes)
- **构建**: Vite 7
- **样式**: Tailwind CSS 4
- **UI 组件**: bits-ui + lucide-svelte
- **状态管理**: Svelte 5 runes ($state, $derived, $effect)
- **HTTP 客户端**: 自定义封装（见 `src/lib/api/client.ts`）
- **WebSocket**: 自定义封装（见 `src/lib/websocket/`）

## 项目结构

```
src/
├── app.html                  # HTML 模板
├── app.css                   # 全局样式
├── hooks.client.ts           # 客户端钩子（API + WebSocket 初始化）
├── hooks.server.ts           # 服务端钩子
├── lib/
│   ├── api/
│   │   ├── client.ts         # HTTP 客户端（统一拦截器、Token 刷新）
│   │   ├── endpoints.ts      # API 路径常量（唯一路径定义处）
│   │   └── services/         # 按业务域拆分的 API 服务
│   ├── components/           # UI 组件
│   │   ├── common/           # 业务通用组件
│   │   └── ui/               # bits-ui 基础组件
│   ├── stores/               # Svelte 5 runes 状态管理
│   ├── types/                # TypeScript 类型定义
│   ├── utils/                # 工具函数
│   └── websocket/            # WebSocket 连接与事件处理
└── routes/                   # SvelteKit 路由
    ├── (admin)/              # 管理端页面
    ├── (auth)/               # 认证页面
    ├── (protected)/          # 需登录页面
    └── (public)/             # 公开页面
```

## 开发指南

### 环境配置

```bash
# 安装依赖
pnpm install

# 复制环境配置
cp .env.example .env

# 编辑 .env 配置
# 必须配置：ORIGIN（CSRF 保护需要）
```

### 开发命令

```bash
# 开发服务器
pnpm dev

# 类型检查
pnpm check

# 生产构建
pnpm build

# 预览构建
pnpm preview
```

### AI Agent 规范

前端代码规范见 [AGENTS.md](../AGENTS.md)，开发前请务必阅读。

**常见错误检查清单**：
- [ ] 是否硬编码了 API 路径 `"/api/v1/..."`？
- [ ] 是否依赖了 HTTP 状态码而非 `code === 0`？
- [ ] 是否使用了 `any` 类型？
- [ ] 是否在 localStorage 存了 Refresh Token？
- [ ] 是否解析了 JWT？
- [ ] 是否在前端计算了距离/角度？

## API 调用规则

所有 API 路径必须定义在 `src/lib/api/endpoints.ts` 中，代码中禁止出现硬编码路径。

```typescript
// 正确
import { AUTH } from '$lib/api/endpoints';
const result = await post<AuthData>(AUTH.LOGIN, data);

// 错误
const result = await post<AuthData>('/api/v1/auth/login', data);
```

响应处理只判断 `code === 0`，不依赖 HTTP 状态码。

## 相关文档

- [API 规范](../spec/api_spec_index.md)
- [前端编码规范](../spec/frontend_rule.md)
- [WebSocket 事件](../spec/api_spec_websocket.md)
