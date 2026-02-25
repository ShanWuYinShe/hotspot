---
globs: frontend_svelte/**
alwaysApply: false
---

# HotSpot 前端实现规范（SvelteKit / How）

## 文档定位（权威边界）

本文件只描述工程实现规范与目录结构，不复述 API 契约。

权威引用：
- **API 契约**：`@spec/api_spec_index.md`
- **基础规范**：`@spec/api_spec_base.md`
- **通用类型**：`@spec/api_spec_types.md`
- **错误码**：`@spec/api_spec_errors.md`
- **功能边界**：`@spec/frontend_doc.md`

---

## 技术栈

- SvelteKit（Svelte 5）
- TypeScript（严格模式）
- Vite

---

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

---

## API 调用规范

- **禁止硬编码路径**：必须通过 `src/lib/api/endpoints.ts`
- **成功判定**：仅使用 `ApiResponse.code === 0`
- **HTTP 层错误**：通过 try/catch 处理网络/超时/非 200（见通用契约）
- **仅在 HTTP 200 时解析**：非 200 直接视为请求失败，不解析 `ApiResponse`
- **Cursor 不透明**：前端只透传 `next_cursor`
- **RPC + JSON**：仅 POST，不实现 REST 风格

---

## Token 与安全

- Web 端仅使用 HttpOnly Cookie，禁止在 localStorage/sessionStorage 存储 Refresh Token
- App 端使用 `Authorization: Bearer`（如有跨端场景）
- 前端不解析 JWT，只按业务接口返回值处理登录态

---

## WebSocket 规范

- 事件名与 payload 结构以 `@spec/api_spec_websocket.md` 为准
- 可在前端定义**镜像类型**（仅映射契约，不得扩展）
- 消息最终态以 WebSocket 推送为准

---

## 位置与距离

- 位置请求使用 `Location`，响应使用 `RelativeLocation`
- 禁止前端自行计算距离/角度

---

## 数据格式（契约级）

- 所有时间字段为 UTC ISO 8601，前端负责时区转换
- 所有 `xxx_id` 字段按 `string` 处理，不做数值运算

---

## TypeScript 约束

- 禁止 `any`
- 访问可选字段必须使用 `?.`/`??`
