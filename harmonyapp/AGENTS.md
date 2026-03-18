# HarmonyOS App - Agent Guide

> HarmonyOS ArkTS (.ets) 鸿蒙客户端的 AI Agent 操作指南。

## 使用范围

本文件只补充鸿蒙端特有的硬约束（API 调用、图标、安全区等），其他页面结构和组件划分请 Agent 通过代码与 `spec/` 自行探索。

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

- **业务成功判断唯一依据**：`code === 0`，禁止依赖 HTTP 状态码或 `res.message`。

## 状态与持久化规则

- 持久化键名必须有意义、自描述（如 `currentUser`、`isLoggedIn`），禁止使用 `key1`、`temp` 等模糊命名。

## 错误处理

- 错误码 `10201`（账号封禁）须强制登出。
- 错误码 `10100`（认证失败）须尝试刷新 Token 或跳转登录。

## 图标使用规范

### 官方资源

- **HarmonyOS 主题图标库**：https://developer.huawei.com/consumer/cn/design/harmonyos-symbol/
- 所有系统图标必须从此库中选取，禁止使用不存在的符号名。

### 使用规则

- **统一使用 `SymbolSpan` / `SymbolGlyph`**：系统图标优先使用 `sys.symbol.*` 符号字体，而非 `Image($r('sys.media.ohos_ic_public_*'))`。符号字体支持动态字号、主题适配、矢量缩放，优于位图。
- **图标集中定义**：所有图标名称在 `components/SystemIcon.ets` 的 `AppIcon` 枚举中统一管理。新增图标时必须先在官方图标库中确认符号名存在，再添加到枚举。
- **系统图标引用方式**：禁止在业务代码中直接写 `$r('sys.symbol.xxx')` 字符串，必须通过 `AppIcon` 枚举引用。
- **业务图形例外**：地图 marker、自定义品牌图标等业务相关图形可以使用自定义图片资源（例如 SVG 或 `Image($r('app.media.*'))`），不强制使用 `sys.symbol.*`。

> 具体符号名是否存在，请以官方图标库为准（以实际文档为准），本文件不维护完整列表。

## WebSocket

- 事件类型必须与 `spec/api_spec_websocket.md` 一致，禁止扩展 spec 未定义的事件。

## 位置与距离

- 请求参数使用 `Location`（point + location_name）。
- 响应使用 `RelativeLocation`（含 distance、bearing_degrees）。
- **禁止在客户端计算距离或角度**，一律由后端完成（具体实现以 `spec/` 和后端代码为准）。

## 顶部/底部安全区（Safe Area）规则

- 使用系统提供的 avoid area 计算 `safeAreaTop` / `safeAreaBottom`，所有可交互组件和正文内容必须布局在 Safe Area 内。
- 纯背景层可以扩展到状态栏 / 导航栏区域，但这些区域内不得放置可点击组件或正文文本。
- 只有当 `safeAreaTop` 或 `safeAreaBottom` 为 0（系统栏实际不存在）时，才允许交互内容延伸到对应屏幕边缘，禁止自行假设“某些设备没有导航栏”。
- 头部和主壳等通用组件必须复用统一的 Safe Area 处理，不得在局部硬编码固定高度。

## 额外禁止

除根目录 `AGENTS.md` 中的全局禁止项外，鸿蒙端额外禁止：
- 硬编码 API 路径 `"/api/v1/..."`，或绕过 `ApiEndpoints` / `HttpUtil` / `ApiClient` 自行封装 HTTP。
- 在**需要系统图标的场景**使用未在官方图标库中验证的 `sys.symbol.*` 名称，或在业务代码中直接写 `$r('sys.symbol.xxx')` / 使用 `Image($r('sys.media.*'))` 替代 `SymbolSpan` / `SymbolGlyph`（地图 marker 等业务自定义图形不在此限）。
- 在客户端计算距离/角度，或解析/构造游标（只能透传），或将交互组件放在系统状态栏 / 导航栏覆盖区域内。

## 安全规则

文件删除、批量变更、Git 操作规范等全局规则见根目录 `AGENTS.md` 的「权限边界」和「Git 操作规范」章节，此处不重复。

> **⚠️ `tmp/` 路径规则**：所有临时文件必须存放在项目根目录的 `tmp/` 下。详见根目录 `AGENTS.md` 的「Agent 行为准则」章节。

**HarmonyOS 额外约束**：

- **文件修改安全措施**：继承根目录 `AGENTS.md` 的统一要求，仅补充一点：`entry/src/main/ets/` 目录下的任何文件修改或删除，务必严格按照该流程执行，避免破坏 ArkTS 关键代码。

## 编译与检查命令

编译和代码检查是两个独立的工具，**必须分别运行**：
- **编译（hvigorw）**：检查编译错误，生成 HAP 包。
- **代码检查（codelinter）**：检查 DevEco Studio IDE 中显示的 linter 警告（性能建议、代码规范等）。

> **编译命令限制**：`hvigorw` 编译命令**禁止 Agent 执行**，必须由用户手动运行。Agent 在完成代码改动后会自动执行 `git add` 暂存改动，用户可直接运行 `hvigorw` 验证编译。`codelinter` 可由 Agent 直接执行。

```bash
# macOS 默认路径
DEVECO_SDK_HOME="/Applications/DevEco-Studio.app/Contents/sdk"
DEVECO_SDK_NODE="/Applications/DevEco-Studio.app/Contents/tools/node/bin/node"
OHPM="/Applications/DevEco-Studio.app/Contents/tools/ohpm/bin/ohpm"
HVIGORW="/Applications/DevEco-Studio.app/Contents/tools/hvigor/bin/hvigorw.js"
CODELINTER="/Applications/DevEco-Studio.app/Contents/plugins/codelinter/run/index.js"

# 安装依赖（编译前必须先执行）
"$OHPM" install --all \
  --registry https://ohpm.openharmony.cn/ohpm/ --strict_ssl true

# 编译【🚫 禁止 Agent 执行，必须由用户手动运行；Agent 已在任务结束时自动 git add】
"$DEVECO_SDK_NODE" "$HVIGORW" \
  --mode module -p product=default assembleHap \
  --info --analyze=ultrafine --parallel --incremental \
  --daemon

# 代码检查（检查 IDE 中的 linter 警告，基于 code-linter.json5 规则配置）【✅ Agent 可直接执行】
DEVECO_SDK_HOME="$DEVECO_SDK_HOME" "$DEVECO_SDK_NODE" "$CODELINTER" \
  -c code-linter.json5 entry/src/main/ets/

# 禁止模式扫描【✅ Agent 可直接执行】
rg '"/api/v1/' entry/src/main/ets/
```
