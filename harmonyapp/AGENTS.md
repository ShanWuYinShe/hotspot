# HotSpot HarmonyOS App

## 功能边界

HarmonyOS **ArkTS 客户端**。覆盖认证(含多账户切换)/发现与搜索/信息流/生活瞬间与评论/媒体上传/聊天(私聊+群聊)/群组/通知与公告/举报/地图(原生组件+位置选择器)/附近/个人中心。管理后台功能仅 Web 端提供,本端不含。支付与交易、端侧离线同步不在范围内。

> 订阅管理:契约待定,当前 spec 未定义订阅相关 API,待后续版本补充。

API 契约见 `spec/api_spec_*.md`（仓库根 `spec/` 目录，只读权威）。

## 构建 & 测试

DevEco Studio 工具路径：

```
NODE=/Applications/DevEco-Studio.app/Contents/tools/node/bin/node
HVIGORW=/Applications/DevEco-Studio.app/Contents/tools/hvigor/bin/hvigorw.js
OHPM=/Applications/DevEco-Studio.app/Contents/tools/ohpm/bin/ohpm
DEVECO_SDK_HOME=/Applications/DevEco-Studio.app/Contents/sdk
JAVA_HOME=/Applications/DevEco-Studio.app/Contents/jbr/Contents/Home
PATH=$JAVA_HOME/bin:$PATH
```

- `$OHPM install --all --registry https://ohpm.openharmony.cn/ohpm/ --strict_ssl true` — 安装依赖
- `$NODE $HVIGORW --mode project -p product=default assembleApp --analyze=normal --parallel --incremental --daemon` — 编译整个应用
- `$NODE $HVIGORW assembleHap` — 仅编译 HAP 包（快速检查用）
- `$NODE $HVIGORW clean` — 清理构建产物

> 如果修正 `DEVECO_SDK_HOME` / `JAVA_HOME` 后仍复用旧环境，先运行 `$NODE $HVIGORW --stop-daemon`，再重新构建。

> **Lint & Format**: DevEco Studio 内置，无独立 CLI。代码检查由 IDE 在编辑时自动执行。

## 技术栈（名称级）

- HarmonyOS ArkTS (.ets)、ArkUI 原生组件、hvigor 构建工具
- `@ohos.net.http`（网络）、`@ohos.data.preferences`（持久化）

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
  - `utils/`：工具函数
- `entry/src/main/ets/features/`：按业务域划分的页面（auth/、chat/、group/、moment/ 等）

## API 调用规范

- **禁止硬编码路径**：必须通过 `core/config/ApiEndpoints.ets`
- **统一入口**：通过 `core/network/HttpUtil.ets` 的 `post()` 方法（`HttpUtil.post()` / `ApiClient`）发起请求
- **成功判定**：仅使用 `code === 0`，禁止依赖 HTTP 状态码或 `message` 字段
- **Token 自动携带**：从 `PreferencesUtil` 读取，收到 `code=10100`（认证失败）时自动 refresh 并重试
- **幂等键**：MUST 幂等接口以 `spec/api_spec_base.md`「必须使用幂等键的接口（MUST）」列表为准；实现中 `MUST_IDEMPOTENT_ENDPOINTS` 须与该列表同步（增删端点时需更新），自动注入 `Idempotency-Key`
- **Cursor 不透明**：请求时透传 `cursor`，收到响应后将 `next_cursor` 作为下一页的 `cursor` 透传，不解析、不构造
- **RPC + JSON**：仅 POST，不实现 REST 风格

## Token 与安全

- Token 通过 `PreferencesUtil` 存取
- 使用 `Authorization: Bearer` 方式携带
- 禁止解析 JWT，登录态按业务接口返回值判断
- 错误码 `10100`（认证失败）须尝试刷新 Token 或跳转登录
- 错误码 `10201`（账号封禁）须强制登出并展示封禁页

## 状态管理

- **AppStorage + PersistentStorage**：HarmonyOS 原生状态管理
- **GlobalStateManager**：单例，管理全局状态（currentUser、isLoggedIn、unreadCount 等）
- **AuthAccountManager**：多账户支持，持久化 accounts 和 current_user_id
- UI 绑定使用 `@StorageLink` 等装饰器
- 持久化键使用有意义的名称（currentUser、isLoggedIn 等）

## WebSocket 规范

- 事件名与 payload 结构以 `spec/api_spec_websocket.md` 为准
- `WebSocketService` 单例管理连接，支持鉴权、心跳、自动重连
- 事件类型定义在 `core/types/WebSocketTypes.ets`，必须与 spec 一致
- 可定义镜像类型，不得扩展 spec 中未定义的事件

## 位置与距离

- 位置请求使用 `Location`（point + location_name），响应使用 `RelativeLocation`
- 禁止在客户端计算距离或角度，由后端 PostGIS 完成
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

- 颜色使用 `$r('app.color.*')` 资源引用或 `DesignConstants` 常量，禁止硬编码色值
- 字体大小、间距等使用 `DesignConstants` 常量
- 核心服务使用单例模式（`GlobalStateManager.getInstance()`）

## UI/UX 设计规范

以下为内联设计规范（原 `.claude/rules/design-tokens.md` 已并入本文件），改 UI 前必读，避免硬编码色值/字号/间距、避免手搓页面头部。

### 设计系统（design-tokens）

所有颜色、字号、间距、圆角、阴影**只用 DesignConstants / AppThemeManager token**，对齐前端 forest 设计系统（单一事实源）。**禁止**在页面/组件里硬编码十六进制色值或裸数字 `fontSize(N)` / `padding({left:N})`。

**颜色**：
- 画布：`COLOR.PRIMARY_BG_*` / `SECONDARY_BG_*`
- 表面：`COLOR.SURFACE_NEUTRAL_SOFT` / `SURFACE_INFO_SOFT` 等
- 文字：`COLOR.TEXT_PRIMARY_*` / `TEXT_SECONDARY_*` / `TEXT_TERTIARY_*`
- 品牌：`COLOR.BRAND_PRIMARY`（forest `#3D7A55`，与前端 `--theme-accent-primary` 一致）
- 语义：`SUCCESS` / `WARNING` / `ERROR` / `INFO`
- 主题解析：`AppThemeManager.getPaletteById(resolvedThemeId)` → `@StorageProp(KEY_RESOLVED_THEME_ID)`

**字号（FONT.SIZE，禁用裸 `fontSize(N)`）**：`XS_MINUS 9` / `XS2 10` / `XS 11` / `SM 12` / `SM_PLUS 13` / `MD 14` / `MD_PLUS 15` / `LG 16` / `LG_PLUS 18` / `XL 20` / `XL_PLUS 22` / `XXL 24` / `XXXL 32` / `HERO 48`。字重 `FONT.WEIGHT.{LIGHT,REGULAR,MEDIUM,SEMIBOLD,BOLD}`。

**间距（SPACING，4px 网格）**：`XS 4` / `SM 8` / `MD 12` / `LG 16` / `XL 24` / `XXL 32` / `XXXL 48`。**禁止**组件内 `padding({left:16})` 这类裸数——用 `SPACING.LG`。

**圆角（RADIUS / SHAPE）**：`SM 8` / `MD 12` / `LG 16` / `XL 18` / `XXL 22` / `FULL 999`。语义形状 `SHAPE.{CARD 22, HERO_CARD 24, INPUT 18, BUTTON 28, CHIP FULL, TAB_BAR 24}`。与前端 `--radius-*` 完全一致。

**页面头部（强制 SystemPageHeader）**：
- **所有页面头部必须用 `SystemPageHeader`**（或专用组件 `SearchPageHeader` / `ChatPageHeader`）。
- 禁止手搓 `Row { Text(...).fontSize(XL)... Button(...) }` 头部。
- SystemPageHeader API：`title` / `subtitle` / `showBackButton`+`onBack` / `action`(文本按钮) / `leftAction`+`rightAction`(IconAction) / `trailing`(@BuilderParam 任意尾随) / `titleAlign` / `titleSize` / `titleWeight` / `backgroundVariant`('bar'|'transparent'|'card')。
- 安全区由 SystemPageHeader 内聚（自带 `@StorageLink('safeAreaTop')` + `expandSafeArea`），页面无需重复写。
- 内联搜索框头部 → `SearchPageHeader`；浮动卡片+头像头部 → `ChatPageHeader`。

**页面内容栏（PageShellContainer）**：档位（对齐前端 `.page-shell`）：`form 560` / `narrow 720` / `default 1180` / `wide 1320`；feed：`feed-narrow 640` / `feed-medium 960`。
