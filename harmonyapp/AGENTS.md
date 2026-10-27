# HotSpot HarmonyOS App

## 功能边界

HarmonyOS ArkTS 客户端。覆盖认证(含多账户切换)/发现与搜索/信息流/生活瞬间与评论/媒体上传/聊天(私聊+群聊)/群组/通知与公告/举报/地图(原生组件+位置选择器)/附近/个人中心/管理后台/订阅管理。支付与交易、端侧离线同步不在范围内。

管理后台与订阅管理的端内页面已存在（`AdminOverviewPage`/`SubscriptionsPage`，路由常量见 `AppConstants.ets`）；管理后台契约见 `spec/api_spec_admin.md`，其余以 `spec/` 为准。

API 契约见 `spec/api_spec_*.md`（仓库根 `spec/` 目录，只读权威）。

## 版本号管理

App 版本唯一来源是 `AppScope/app.json5`（`versionName` + `versionCode`），EntryAbility 启动时经 bundleManager 读取并打日志。**各端版本独立**：Web/后端用仓库 git tag（`vX.Y.Z`）发版，鸿蒙不跟 tag 走——Web 发版不需要 bump `app.json5`，只有鸿蒙发版才 bump：

- `versionName` 取本次鸿蒙发版的 `X.Y.Z`，`versionCode` 设为 `X*1000000 + Y*1000 + Z`（如 `1.2.3` → `1002003`）；
- 发版流程：commit 在对应 Worktree 内完成；合并进主分支后推送（见 skill `git-workflow`）。

排查用户反馈时以 `versionName` 定位到发版 commit。未 bump 版本即发安装包 = 无法溯源，视为发布事故。

## 构建 & 测试

DevEco Studio 工具路径（node/hvigor/ohpm 以本机 `/Applications/DevEco-Studio.app/Contents/tools/` 下实际目录为准，使用前 `ls` 验证；JAVA_HOME 见 skill `local-environment` 的 Android Studio JBR 条目，变动时用 `mdfind` 定位）：

```
NODE=/Applications/DevEco-Studio.app/Contents/tools/node/bin/node
HVIGOR=/Applications/DevEco-Studio.app/Contents/tools/hvigor/bin/hvigorw.js
OHPM=/Applications/DevEco-Studio.app/Contents/tools/ohpm/bin/ohpm
DEVECO_SDK_HOME=/Applications/DevEco-Studio.app/Contents/sdk
JAVA_HOME=/Applications/Android Studio.app/Contents/jbr/Contents/Home
PATH=$JAVA_HOME/bin:$PATH
```

- **所有 hvigor 命令一律 `--no-daemon`**：构建/check 完进程立即退出，禁止 `--daemon`（会常驻 Java 进程占资源）。历史残留 daemon 用 `$NODE $HVIGOR --stop-daemon` 清理；游离进程（`--stop-daemon` 报 0 但 `ps` 仍见 `hvigor-java-daemon`）用 `pkill -f hvigor-java-daemon`。
- `$OHPM install --all --registry https://ohpm.openharmony.cn/ohpm/ --strict_ssl true` — 安装依赖
- `$NODE $HVIGOR --mode project -p product=default -p buildMode=debug assembleApp --analyze=normal --parallel --incremental --no-daemon` — 编译整个应用（含 CompileArkTS + 打包 + 签名，约 45s）。**`-p buildMode=debug` 不可省**：缺省按 release 构建，`BuildProfile.DEBUG=false` 会让 `SERVER_BASE_URL` 指向生产域名（`chaoset.com`，与 Web 网关同域；api 子域未建 DNS），本地调试环境永远连不上后端且不易察觉
- `$NODE $HVIGOR clean --no-daemon` — 清理构建产物

> 当前 hvigor 版本无 `assembleHap` 任务（旧文档废弃，调用报 `Task 'assembleHap' was not found`）；类型检查在 assembleApp 启动时自动执行（首行 `TYPE CHECK SUCCESSFUL`），完整验证统一用 assembleApp。

**提交时的 hvigor 门禁（2026-10-19 实测）**：repo root 的 pre-commit 钩子会对 `harmonyapp/` 改动跑 hvigor
`assembleApp`（工具链缺失时告警跳过）。沙盒内缓存目录（`~/.hvigor`、`~/.npm`、`~/.ohpm`）已由
sandbox-extra-roots 放行，直接 `git commit` 即可，**不要 export `HVIGOR_USER_HOME`/`npm_config_cache` 重定向**。
若报 `EPERM: ... mkdir '/Users/<me>/.hvigor/project_caches/...'`（不是 Git 问题），把该目录加进插件配置即可
（见 `sandbox-adaptation` skill）。`JAVA_HOME`/`DEVECO_SDK_HOME` 按上文「构建 & 测试」设置（工具链路径，非沙盒重定向）。

如果修正 `DEVECO_SDK_HOME` / `JAVA_HOME` 后仍复用旧环境，先运行 `$NODE $HVIGOR --stop-daemon`，再重新构建。

Lint & Format: 依赖 DevEco Studio IDE 内置检查；`harmonyapp/code-linter.json5`（本模块根）是 codelinter 的规则配置（性能/TS 语法/安全规则集），命令行批量检查可配合 codelinter CLI 使用。

## 调试签名（本机绑定）

- `build-profile.json5` 的 `signingConfigs.default` 是**本机调试签名**：材料在各人 `~/.ohos/config/` 下，路径与口令均为单机有效；提交的版本只对提交者本机可用，对其他机器无意义。
- DevEco 会在材料失配时自动重生并重写该文件（`git status` 见 M 属正常）：本机能编能装就不用管；**不要把轮换后的口令反复提交**（纯噪音）。
- 各 checkout 的隐藏由 `.husky/post-checkout` 自动完成（新建 worktree / 切分支即挂 `skip-worktree`，防切分支把文件还原成旧口令导致签名失配），一般不用手动执行；若标记丢失再手动补一次：
  `git update-index --skip-worktree harmonyapp/build-profile.json5`
- 新 worktree 检出即带可用签名，开箱即用；若将来 DevEco 整体重生材料，各 checkout 打开一次即自动修复，互不干扰。
- 生产/发布签名走正式证书流程，不在此文件里配明文。

## 技术栈（名称级）

- HarmonyOS ArkTS (.ets)、ArkUI 原生组件、hvigor 构建工具
- `@ohos.net.http`（网络）、`@ohos.data.preferences`（持久化）

## 目录与分层（必须）

- `entry/src/main/ets/entryability/`：应用入口 Ability
- `entry/src/main/ets/components/`：公共 ArkUI 组件
- `entry/src/main/ets/constants/`：常量定义（AppConstants、DesignConstants、SystemConstants）
- `entry/src/main/ets/core/`：核心逻辑层（按用途分子目录，不逐一举出）
  - `api/ApiClient.ets`：按业务域封装的 API 调用静态方法
  - `config/ApiEndpoints.ets`：API 路径常量（唯一路径定义处）
  - `network/HttpUtil.ets`：统一 HTTP 客户端（POST、Token 刷新、幂等键）
  - `network/WebSocketService.ets`：WebSocket 单例
  - `state/`：全局状态管理（GlobalStateManager、AuthAccountManager、PersistentStateManager）
  - `types/`：API 类型定义（对齐 spec）、WebSocket 事件类型
- `entry/src/main/ets/features/`：按业务域划分的页面（每域一个目录，新增页面按域归类，禁止在 core/ 里放页面）

## API 调用规范

- 禁止硬编码路径：必须通过 `core/config/ApiEndpoints.ets`
- 统一入口：通过 `core/network/HttpUtil.ets` 的 `post()` 方法（`HttpUtil.post()` / `ApiClient`）发起请求
- 成功判定：仅使用 `code === 0`，禁止依赖 HTTP 状态码或 `message` 字段
- Token 自动携带：经 `HttpUtil.getToken()`（Asset 存储）读取，收到 `code=10100`（认证失败）时自动 refresh 并重试
- 幂等键：MUST 幂等接口以 `spec/api_spec_base.md`「必须使用幂等键的接口（MUST）」列表为准；实现中 `MUST_IDEMPOTENT_ENDPOINTS` 须与该列表同步（增删端点时需更新），自动注入 `Idempotency-Key`
- Cursor 不透明：请求时透传 `cursor`，收到响应后将 `next_cursor` 作为下一页的 `cursor` 透传，不解析、不构造
- RPC + JSON：仅 POST，不实现 REST 风格

## Token 与安全

- Token 等敏感凭据（access_token、auth_data、各账号 AuthData）通过 `TokenAssetStore`（HarmonyOS Asset Store Kit，`@ohos.security.asset`）加密存储；Preferences 只存非敏感索引（多账号 user_id 列表 + current_user_id）
- `PreferencesUtil` 仅负责非敏感键（device_id、主题、服务器地址、通知偏好等）
- 使用 `Authorization: Bearer` 方式携带
- 禁止解析 JWT，登录态按业务接口返回值判断
- 错误码 `10100`（认证失败）须尝试刷新 Token 或跳转登录
- 错误码 `10201`（账号封禁）须强制登出并展示封禁页

## 状态管理

- AppStorage + PersistentStorage：HarmonyOS 原生状态管理
- GlobalStateManager：单例，管理全局状态（currentUser、isLoggedIn、unreadCount 等）
- AuthAccountManager：多账户支持，持久化 accounts 和 current_user_id
- UI 绑定使用 `@StorageLink` 等装饰器
- 持久化键使用有意义的名称（currentUser、isLoggedIn 等）

## WebSocket 规范

- 事件名与 payload 结构以 `spec/api_spec_websocket.md` 为准
- `WebSocketService` 单例管理连接，支持鉴权、心跳、自动重连
- 事件类型定义在 `core/types/WebSocketTypes.ets`，必须与 spec 一致
- 可定义镜像类型，不得扩展 spec 中未定义的事件
- 原生 WebSocket 不受浏览器语义约束：连接时显式携带与服务端同源的 `Origin`（见 `WebSocketUtil`），后端对无 Cookie 的 `Origin: null` 也放行；用户切换/冷启动后由 `Index` 按访问令牌补连（`onLoginStateChanged` + 有限重试），避免「已登录但不建立实时通道」

## 位置与距离

- 位置请求使用 `Location`（point + location_name），响应使用 `RelativeLocation`
- 坐标转换仅在 `CoordinateConverter.ets` 单一边界层执行一次

## 数据格式（契约级）

- 所有时间字段为 UTC ISO 8601，客户端负责时区转换
- 所有 `xxx_id` 字段按 `string` 处理，不做数值运算

## 图标规范

- 统一使用 `SymbolSpan` / `SymbolGlyph`，优先使用 `sys.symbol.*` 符号字体
- 所有图标名称在 `components/SystemIcon.ets` 的 `AppIcon` 枚举中集中管理
- 禁止在业务代码中直接写 `$r('sys.symbol.xxx')`，必须通过 `AppIcon` 枚举引用
- 新增图标前须在 HarmonyOS 主题图标库中确认符号名存在

## 组件与样式

- 颜色/字号/间距/圆角等一律走 `DesignConstants` / `AppThemeManager` token，禁止硬编码——强制细则见 skill `hotspot-harmony-ui` 与 `docs/design-system.md`。
- 核心服务使用单例模式（`GlobalStateManager.getInstance()`）

## UI 与组件规范

**完整契约见 skill hotspot-harmony-ui**（设计 token 强制规则、跨端同步机制、SystemPageHeader /
PageShellContainer、路由对齐、SystemButton / SystemInput 原语与已登记例外清单）。要点速记：

- 颜色/字号/间距/圆角/边框/动效/透明度/尺寸一律走 AppThemeManager palette 与 DesignConstants token，禁止硬编码。
- 页面头部用 SystemPageHeader（专用变体 SearchPageHeader / ChatPageHeader）；内容栏用 PageShellContainer。
- 通用按钮/输入框用 SystemButton / SystemInput，禁止手写重复样式（已登记例外见 skill）。
- 跨端 token（色彩/字号/圆角）必须与 frontend 同步，check:theme 漂移即拦截。
- 状态色 on-soft（R12）：`ColorConfig.accentOnSoft/onlineOnSoft/warningOnSoft/dangerOnSoft` 必须等于 web 语义色 80% + textPrimary 20%；
  `python3 harmonyapp/scripts/check_status_parity.py` 断言双端 6 配置 × 4 语义色 hex 一致与对比度 ≥4.5（palette 值不变）。
