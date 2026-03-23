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

## 与根流程的关系
- 主目录保持 `main` + clean，所有实际改动都在 worktree 中完成；本地分支、本地集成 worktree、本地验证入口和风险边界，以仓库根 `CLAUDE.md` 为准。
- 默认验证入口是 `verify.sh`，默认验证的是当前 worktree 快照；涉及 API、WebSocket 或地图行为时，再对照 `spec/`。

## 关键约束
- 禁止硬编码 `/api/v1/...`，必须通过 `src/lib/api/endpoints.ts`。
- 所有请求统一走 `src/lib/api/client.ts` 和 `src/lib/api/services/*`；不要另起一套 HTTP 封装。
- 所有请求都是 `POST`，业务成功只看 `code === 0`。
- `cursor` 只透传，不解析、不构造。
- Web 端只用 HttpOnly Cookie，不在 localStorage / sessionStorage 放 refresh token，也不解析 JWT。
- TypeScript 禁止 `any`。
- WebSocket 事件类型必须和 spec 对齐。
- 地图、位置和列表展示中不在前端计算距离或角度。

## 地图与坐标边界
- 只在“业务坐标 ↔ 地图坐标”的单一边界层做一次转换；如果输入已经是地图坐标系，不要重复转换。
- marker、overlay、位置选择器中心点必须使用地图 SDK 原生能力，禁止 DOM 伪 marker。
- center pin 必须绑定地图相机中心；回写业务状态时直接使用地图 SDK 返回的中心坐标。

## 常用命令
- 默认验证入口：`./deploy/scripts/verify.sh --env-name <name>`
- 需要预演候选结果时：`./deploy/scripts/preview-build.sh` + `./deploy/scripts/preview-up.sh`
- `pnpm check`、`pnpm build`、`pnpm dev` 仅作为本地调试参考，不作为默认最终验证入口。
- 当前仓库未配置独立前端单测脚本，最终验证仍以 `deploy/` 下的 Docker 流程为准。
