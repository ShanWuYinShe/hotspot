---
globs: harmonyapp/**
alwaysApply: false
---

# HotSpot 鸿蒙客户端实现规范（ArkTS / How）

## 文档定位（权威边界）

本文件只回答"如何落地"：
- ArkTS 的实现约束与代码模式
- 目录结构、状态管理、网络层、组件规范

权威引用：
- **API 契约**：`@spec/api_spec_index.md`
- **基础规范**：`@spec/api_spec_base.md`
- **通用类型**：`@spec/api_spec_types.md`
- **错误码**：`@spec/api_spec_errors.md`
- **功能边界**：`@spec/harmonyapp_doc.md`

---

## 技术栈（名称级，不写具体版本）

- HarmonyOS ArkTS (.ets)
- ArkUI 原生组件
- hvigor 构建工具
- `@ohos.net.http`（网络）
- `@ohos.data.preferences`（持久化）

---

## 目录与分层（必须）

- `entry/src/main/ets/entryability/`：应用入口 Ability
- `entry/src/main/ets/components/`：公共 ArkUI 组件
- `entry/src/main/ets/constants/`：常量定义（AppConstants、DesignConstants、SystemConstants）
- `entry/src/main/ets/core/`：核心逻辑层
  - `api/ApiClient.ets`：按业务域封装的 API 调用静态方法
  - `config/ApiEndpoints.ets`：API 路径常量（唯一路径定义处）
  - `network/HttpUtil.ets`：统一 HTTP 客户端（POST、Token 刷新、幂等键）
  - `network/WebSocketService.ets`：WebSocket 单例
  - `state/`：全局状态管理（GlobalStateManager、AuthAccountManager、PersistentStateManager）
  - `types/`：API 类型定义（对齐 spec）、WebSocket 事件类型
  - `di/Container.ets`：依赖注入容器
  - `utils/`：工具函数
- `entry/src/main/ets/features/`：按业务域划分的页面（auth/、chat/、group/、moment/ 等）

---

## API 调用规范

- **禁止硬编码路径**：必须通过 `core/config/ApiEndpoints.ets`
- **统一入口**：通过 `core/network/HttpUtil.ets` 的 `post()` 方法发起请求
- **成功判定**：仅使用 `code === 0`，禁止依赖 HTTP 状态码或 `message` 字段
- **Token 自动携带**：从 `PreferencesUtil` 读取，收到 `code=10100`（认证失败）时自动 refresh 并重试
- **幂等键**：MUST 幂等接口以 `@spec/api_spec_base.md`「必须使用幂等键的接口（MUST）」列表为准；实现中 `MUST_IDEMPOTENT_ENDPOINTS` 须与该列表同步（增删端点时需更新），自动注入 `Idempotency-Key`
- **Cursor 不透明**：请求时透传 `cursor`，收到响应后将 `next_cursor` 作为下一页的 `cursor` 透传，不解析、不构造
- **RPC + JSON**：仅 POST，不实现 REST 风格

---

## Token 与安全

- Token 通过 `PreferencesUtil` 存取
- 使用 `Authorization: Bearer` 方式携带
- 禁止解析 JWT，登录态按业务接口返回值判断
- 错误码 `10100`（认证失败）须尝试刷新 Token 或跳转登录
- 错误码 `10201`（账号封禁）须强制登出并展示封禁页

---

## 状态管理

- **AppStorage + PersistentStorage**：HarmonyOS 原生状态管理
- **GlobalStateManager**：单例，管理全局状态（currentUser、isLoggedIn、unreadCount 等）
- **AuthAccountManager**：多账户支持，持久化 accounts 和 current_user_id
- UI 绑定使用 `@StorageLink` 等装饰器
- 持久化键使用有意义的名称（currentUser、isLoggedIn 等）

---

## WebSocket 规范

- 事件名与 payload 结构以 `@spec/api_spec_websocket.md` 为准
- `WebSocketService` 单例管理连接，支持鉴权、心跳、自动重连
- 事件类型定义在 `core/types/WebSocketTypes.ets`，必须与 spec 一致
- 可定义镜像类型，不得扩展 spec 中未定义的事件

---

## 位置与距离

- 位置请求使用 `Location`（point + location_name），响应使用 `RelativeLocation`
- 禁止在客户端计算距离或角度，由后端 PostGIS 完成

---

## 数据格式（契约级）

- 所有时间字段为 UTC ISO 8601，客户端负责时区转换
- 所有 `xxx_id` 字段按 `string` 处理，不做数值运算

---

## 图标规范

- 统一使用 `SymbolSpan` / `SymbolGlyph`，优先使用 `sys.symbol.*` 符号字体
- 所有图标名称在 `components/SystemIcon.ets` 的 `AppIcon` 枚举中集中管理
- 禁止在业务代码中直接写 `$r('sys.symbol.xxx')`，必须通过 `AppIcon` 枚举引用
- 新增图标前须在 HarmonyOS 主题图标库中确认符号名存在

---

## 组件与样式

- 颜色使用 `$r('app.color.*')` 资源引用或 `DesignConstants` 常量，禁止硬编码色值
- 字体大小、间距等使用 `DesignConstants` 常量
- 核心服务使用单例模式（`GlobalStateManager.getInstance()`）
- 页面通过 `core/di/Container.ets` 获取服务实例
