# HarmonyOS App - Agent Guide

> HarmonyOS ArkTS (.ets) 鸿蒙客户端的 AI Agent 操作指南。

## 技术栈

- **框架**：HarmonyOS ArkTS (.ets)
- **API 版本**：API 11 (SDK 5.1.1)
- **构建工具**：hvigor
- **网络**：`@ohos.net.http`
- **持久化**：`@ohos.data.preferences`（AppStorage + PersistentStorage）
- **UI**：ArkUI 原生组件

## 目录结构

```
entry/src/main/ets/
├── entryability/            # 应用入口 Ability
│   └── EntryAbility.ets
├── components/              # 公共 ArkUI 组件
│   ├── LocationBar.ets
│   ├── SystemAvatar.ets
│   ├── SystemCard.ets
│   └── ...
├── constants/               # 常量定义
│   ├── AppConstants.ets     # API_BASE_URL、WS_BASE_URL、调试开关
│   ├── DesignConstants.ets  # 设计常量
│   └── SystemConstants.ets  # 系统常量
├── core/                    # 核心逻辑层
│   ├── api/
│   │   └── ApiClient.ets    # API 客户端（按业务域封装静态方法）
│   ├── config/
│   │   └── ApiEndpoints.ets # API 路径常量（唯一路径定义处）
│   ├── network/
│   │   ├── HttpUtil.ets     # HTTP 工具（POST、Token 刷新、幂等键）
│   │   ├── WebSocketService.ets  # WebSocket 单例
│   │   ├── WebSocketUtil.ets
│   │   └── ApiErrorUtil.ets # 错误处理工具
│   ├── state/
│   │   ├── GlobalStateManager.ets    # 全局状态单例
│   │   ├── AuthAccountManager.ets    # 多账户管理
│   │   └── PersistentStateManager.ets
│   ├── storage/
│   │   └── PreferencesUtil.ets       # 持久化工具
│   ├── types/
│   │   ├── api.ets                   # API 类型定义（对齐 spec）
│   │   ├── CommonTypes.ets
│   │   └── WebSocketTypes.ets        # WebSocket 事件类型
│   ├── di/
│   │   └── Container.ets             # 依赖注入容器
│   ├── services/                     # 各类服务（通知、分布式等）
│   └── utils/                        # 工具函数
└── features/                         # 按业务域划分的页面
    ├── auth/pages/
    ├── chat/pages/
    ├── discover/pages/
    ├── group/pages/
    ├── map/pages/
    ├── message/pages/
    ├── moment/pages/
    ├── nearby/pages/
    ├── profile/pages/
    ├── search/pages/
    ├── settings/pages/
    └── subscriptions/pages/
```

## API 调用规则（严格执行）

### 路径管理

**所有 API 路径必须定义在 `core/config/ApiEndpoints.ets` 中。**

```typescript
// ✅ 正确：使用 ApiEndpoints 类层次常量
const result = await HttpUtil.post<ApiResponse<AuthData>>(
  ApiEndpoints.AUTH.LOGIN, data
);

// ❌ 禁止：硬编码路径
const result = await HttpUtil.post<ApiResponse<AuthData>>(
  '/api/v1/auth/login', data
);
```

**禁止模式**（会被 rg 扫描拦截）：
- 代码中出现 `"/api/v1/` 字符串。

### HTTP 客户端

- 统一通过 `core/network/HttpUtil.ets` 的 `post()` 方法发起请求。
- 所有请求自动携带 Token（从 `PreferencesUtil` 读取）。
- 401 时自动 refresh token 并重试。
- `MUST_IDEMPOTENT_ENDPOINTS` 列表中的接口自动注入 `Idempotency-Key`。

### API 客户端封装

- `core/api/ApiClient.ets` 按业务域提供静态方法，封装 `HttpUtil.post()`。
- 新增 API 调用时，优先在 `ApiClient` 中添加对应方法。

### 响应处理

```typescript
// ✅ 正确：只看 code === 0
const res = await ApiClient.getUserProfile(userId);
if (res.code === 0 && res.data) {
  // 使用 res.data
}

// ❌ 错误：依赖 HTTP 状态码
// ❌ 错误：依赖 res.message 做逻辑分支
```

## 状态管理

### 全局状态

- **AppStorage + PersistentStorage**：HarmonyOS 原生状态管理。
- **GlobalStateManager**：单例，管理 `currentUser`、`isLoggedIn`、`unreadCount` 等。
- **AuthAccountManager**：多账户支持，`accounts`、`current_user_id` 持久化。
- UI 绑定使用 `@StorageLink` 等装饰器。

### 持久化键命名

- 使用有意义的键名：`currentUser`、`isLoggedIn`、`userType`、`currentLocation`。
- Token 通过 `PreferencesUtil` 存取。

## 编码模式

### 单例模式

核心服务使用单例：

```typescript
// GlobalStateManager、WebSocketService、Container 等
const manager = GlobalStateManager.getInstance();
```

### 依赖注入

- `core/di/Container.ets` 做服务注册与解析。
- 页面中通过 Container 获取服务实例。

### 错误处理

- `ApiErrorUtil` 统一处理 API 错误。
- 错误码 `10201`（账号封禁）须强制登出。
- 错误码 `10100`（认证失败）须尝试刷新 Token 或跳转登录。

## WebSocket

- `WebSocketService` 单例管理连接。
- 支持鉴权、心跳、自动重连、订阅。
- 事件类型定义在 `core/types/WebSocketTypes.ets`，必须与 `spec/api_spec_websocket.mdc` 一致。

## 位置与距离

- 请求参数使用 `Location`（point + location_name）。
- 响应使用 `RelativeLocation`（含 distance、bearing_degrees）。
- **禁止在客户端计算距离或角度**，由后端 PostGIS 完成。

## 常见错误清单

- [ ] 是否硬编码了 API 路径 `"/api/v1/..."`？
- [ ] 新增的写操作接口是否在 `MUST_IDEMPOTENT_ENDPOINTS` 列表中检查？
- [ ] 是否依赖了 HTTP 状态码而非 `code === 0`？
- [ ] 是否在客户端计算了距离/角度？
- [ ] 是否正确使用了 AppStorage 装饰器？
- [ ] cursor 是否做了解析而非透传？
- [ ] 新增页面是否放在了正确的 `features/` 子目录下？

## 文件删除与批量变更规则

- Agent **不得自行删除**任何源代码文件。如需删除，须列出清单并说明原因，经用户确认后执行。
- **批量修改**：当一次操作可能涉及**大量文件的修改、重命名或重构**时，Agent 必须先向用户展示完整的影响范围（受影响文件清单、变更摘要），经用户确认后才可执行。**禁止静默批量操作，必须给用户一个询问提示。**
- **禁止的危险 Shell 命令**（可能导致源代码丢失）：`rm`、`rm -rf`、`mv`（批量）、`find ... -delete`、`xargs rm`。除非用户明确授权，否则一律禁止。

## 禁止使用子任务

- **修改 harmonyapp 项目代码时，严禁使用 Task 子任务（subagent）。** 所有文件的读取、搜索、修改必须由主 Agent 直接执行，不得委托给子任务。
- 此规则的原因：子任务的执行过程不透明，曾导致源代码文件被意外删除且无法追溯根因。
- 唯一例外：只读的代码探索（`subagent_type="explore"`，且设置 `readonly=true`）可以使用。

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

## 编译与检查命令

编译和代码检查是两个独立的工具，**必须分别运行**：
- **编译（hvigorw）**：检查编译错误，生成 HAP 包。
- **代码检查（codelinter）**：检查 DevEco Studio IDE 中显示的 linter 警告（性能建议、代码规范等）。

```bash
# macOS 默认路径
DEVECO_SDK_HOME="/Applications/DevEco-Studio.app/Contents/sdk"
HVIGORW="/Applications/DevEco-Studio.app/Contents/tools/hvigor/bin/hvigorw.js"
CODELINTER="/Applications/DevEco-Studio.app/Contents/plugins/codelinter/run/index.js"

# 编译（检查编译错误）
DEVECO_SDK_HOME="$DEVECO_SDK_HOME" node "$HVIGORW" \
  clean --mode module -p product=default assembleHap \
  --info --analyze=ultrafine --parallel --incremental \
  --enable-build-script-type-check --no-daemon

# 代码检查（检查 IDE 中的 linter 警告，基于 code-linter.json5 规则配置）
DEVECO_SDK_HOME="$DEVECO_SDK_HOME" node "$CODELINTER" \
  -c code-linter.json5 entry/src/main/ets/

# 禁止模式扫描
rg '"/api/v1/' entry/src/main/ets/
```
