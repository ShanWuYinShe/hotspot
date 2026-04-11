# CLAUDE.md

## 架构
SvelteKit Web 前端，Svelte 5 + TypeScript。

- `src/routes/`：按访问级别分组的页面与路由守卫
- `src/lib/api/`：API 调用封装（endpoints、client、services）
- `src/lib/stores/`：全局状态（UI、通知、消息、presence）
- `src/lib/websocket/`：实时消息处理
- `src/hooks.server.ts`：服务端会话钩子

## 约束
- **API 路径**：统一从 `endpoints.ts` 引用。
- **请求封装**：所有请求走 `client.ts` 和 `services/*`。
- **HTTP 方法**：统一 POST。
- **响应判断**：业务成功唯一依据 `code === 0`。
- **分页**：cursor 只透传，不解析、不构造。
- **认证**：使用 HttpOnly Cookie，不在前端存储或解析 Token。
- **TypeScript**：类型明确，避免 `any`。
- **地图交互**：距离和角度由后端计算，前端仅展示；坐标转换仅在“业务坐标 ↔ 地图坐标”单一边界层执行一次；marker、overlay、中心点使用地图 SDK 原生能力，center pin 绑定地图相机中心。
