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
- worktree / integration / sync-back / quality gate / 风险边界等仓库级流程，以仓库根 `CLAUDE.md` 为唯一权威来源；本文件只补充前端的局部架构、实现约束与验证提示。
- skill 入口、受控脚本边界、`tmp/` 使用原则都以根 `CLAUDE.md` 为准；这里不重复复制仓库级规则。
- 最终验证入口默认是 `verify.sh`；涉及页面/UI/交互时，通常还需要 `verify-inspect.sh` 提供隔离页面入口。

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

## 常用能力
- 普通分析、代码阅读、局部低副作用开发动作可直接进行。
- 当仓库已经提供 skill 时，优先通过 skill 发现并使用标准 workflow 入口。
- 页面/UI/交互的最终确认仍优先走 `verify` + `verify-inspect` 提供的隔离入口。
- 验证截图、调试输出和中间文件默认写到当前 worktree 的 `tmp/`，不要污染 worktree 根目录。
