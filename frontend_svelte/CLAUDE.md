# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 这个前端做什么
这是 HotSpot 的 SvelteKit Web 前端，使用 Svelte 5 + TypeScript。`src/routes` 负责页面与路由分组，`src/lib/api` 负责所有 API 调用，`src/lib/stores` 管全局状态，`src/lib/websocket` 管实时消息，`src/lib/components` / `hooks` / `attachments` 提供复用能力。

## 代码组织
- `src/routes/(public)/(protected)/(admin)/(auth)`：按访问级别分组的页面。
- `src/routes/**/+page.ts`、`+layout.ts`：数据加载和路由守卫。
- `src/lib/api/endpoints.ts`：唯一 API 路径常量。
- `src/lib/api/client.ts`：统一请求封装，处理 Token refresh、Idempotency-Key、媒体 URL 归一化。
- `src/lib/api/services/*.ts`：按业务域封装接口调用。
- `src/lib/stores/`：UI、通知、消息、presence 等全局状态。
- `src/lib/websocket/`：WebSocket 协议、处理器和 store。
- `src/hooks.server.ts`：服务端会话/认证钩子。

## 关键约束
- 禁止硬编码 `/api/v1/...`，必须通过 `src/lib/api/endpoints.ts`。
- 所有请求都是 `POST`，业务成功只看 `code === 0`。
- `cursor` 只透传，不解析、不构造。
- Web 端只用 HttpOnly Cookie，不在 localStorage / sessionStorage 放 refresh token，也不解析 JWT。
- TypeScript 禁止 `any`。
- WebSocket 事件类型必须和 spec 对齐。

## 常用命令
- `pnpm check`
- `pnpm build`
- `pnpm dev`
- 当前仓库未配置独立前端单测脚本，主要用 `pnpm check` / `pnpm build` 验证。
