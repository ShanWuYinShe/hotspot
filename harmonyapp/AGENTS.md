# HarmonyOS App - Agent Guide

> HarmonyOS ArkTS (.ets) 鸿蒙客户端的 AI Agent 操作指南。

## 技术栈

- **框架**：HarmonyOS ArkTS (.ets)
- **构建工具**：hvigor
- **网络**：`@ohos.net.http`
- **持久化**：`@ohos.data.preferences`（AppStorage + PersistentStorage）
- **UI**：ArkUI 原生组件

> API 版本以 `build-profile.json5` 为准。

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
- **禁止在业务代码中直接写 `$r('sys.symbol.xxx')` 字符串**，必须通过 `AppIcon` 枚举引用。
- **验证流程**：修改 `AppIcon` 枚举时，须在 https://developer.huawei.com/consumer/cn/design/harmonyos-symbol/ 中搜索确认符号名存在。

### 常见错误

- `sys.symbol.location` 不存在，正确名称是 `sys.symbol.position`。
- 并非所有 SF Symbols 名称都有 HarmonyOS 对应项，如 `pencil`、`gear`、`brain` 等在 HarmonyOS 中不存在。
- `Image($r('sys.media.ohos_ic_public_*'))` 可用但不推荐，应优先查找对应的 `sys.symbol.*` 替代。

## WebSocket

- 事件类型必须与 `spec/api_spec_websocket.md` 一致，禁止扩展 spec 未定义的事件。

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
- [ ] 新增图标是否在 [HarmonyOS 主题图标库](https://developer.huawei.com/consumer/cn/design/harmonyos-symbol/) 中验证了符号名存在？
- [ ] 是否使用了 `Image($r('sys.media.*'))` 而可以改用 `SymbolSpan($r(AppIcon.XXX))`？
- [ ] 颜色/字体/间距是否使用了 `$r('app.color.*')` / `DesignConstants` 常量而非硬编码值？

## 安全规则

文件删除、批量变更、Git 操作规范等全局规则见根目录 `AGENTS.md` 的「权限边界」和「Git 操作规范」章节，此处不重复。

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
DEVECO_SDK_HOME="$DEVECO_SDK_HOME" "$DEVECO_SDK_NODE" "$HVIGORW" \
  --mode module -p product=default assembleHap \
  --info --analyze=ultrafine --parallel --incremental \
  --daemon

# 代码检查（检查 IDE 中的 linter 警告，基于 code-linter.json5 规则配置）【✅ Agent 可直接执行】
DEVECO_SDK_HOME="$DEVECO_SDK_HOME" "$DEVECO_SDK_NODE" "$CODELINTER" \
  -c code-linter.json5 entry/src/main/ets/

# 禁止模式扫描【✅ Agent 可直接执行】
rg '"/api/v1/' entry/src/main/ets/
```
