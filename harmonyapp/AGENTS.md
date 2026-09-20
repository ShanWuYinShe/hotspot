# HotSpot HarmonyOS App

## 功能边界

HarmonyOS ArkTS 客户端。覆盖认证(含多账户切换)/发现与搜索/信息流/生活瞬间与评论/媒体上传/聊天(私聊+群聊)/群组/通知与公告/举报/地图(原生组件+位置选择器)/附近/个人中心/管理后台/订阅管理。支付与交易、端侧离线同步不在范围内。

管理后台与订阅管理的端内页面已存在（`AdminOverviewPage`/`SubscriptionsPage`，路由常量见 `AppConstants.ets`）；管理后台契约见 `spec/api_spec_admin.md`，其余以 `spec/` 为准。

API 契约见 `spec/api_spec_*.md`（仓库根 `spec/` 目录，只读权威）。

## 版本号管理

App 版本唯一来源是 `AppScope/app.json5`（`versionName` + `versionCode`），EntryAbility 启动时经 bundleManager 读取并打日志。**各端版本独立**：Web/后端用仓库 git tag（`vX.Y.Z`）发版，鸿蒙不跟 tag 走——Web 发版不需要 bump `app.json5`，只有鸿蒙发版才 bump：

- `versionName` 取本次鸿蒙发版的 `X.Y.Z`，`versionCode` 设为 `X*1000000 + Y*1000 + Z`（如 `1.2.3` → `1002003`）；
- 发版流程：commit 在对应 Worktree 内完成；合并进主分支后推送（见全局 Git 准则）。

排查用户反馈时以 `versionName` 定位到发版 commit。未 bump 版本即发安装包 = 无法溯源，视为发布事故。

## 构建 & 测试

DevEco Studio 工具路径（node/hvigor/ohpm 以本机 `/Applications/DevEco-Studio.app/Contents/tools/` 下实际目录为准，使用前 `ls` 验证；JAVA_HOME 见全局「本机环境」Android Studio JBR 条目，变动时用 `mdfind` 定位）：

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
- `$NODE $HVIGOR --mode project -p product=default -p buildMode=debug assembleApp --analyze=normal --parallel --incremental --no-daemon` — 编译整个应用（含 CompileArkTS + 打包 + 签名，约 45s）。**`-p buildMode=debug` 不可省**：缺省按 release 构建，`BuildProfile.DEBUG=false` 会让 `SERVER_BASE_URL` 指向生产域名（`api.chaoset.com`），本地调试环境永远连不上后端且不易察觉
- `$NODE $HVIGOR clean --no-daemon` — 清理构建产物

> 当前 hvigor 版本无 `assembleHap` 任务（旧文档废弃，调用报 `Task 'assembleHap' was not found`）；类型检查在 assembleApp 启动时自动执行（首行 `TYPE CHECK SUCCESSFUL`），完整验证统一用 assembleApp。

如果修正 `DEVECO_SDK_HOME` / `JAVA_HOME` 后仍复用旧环境，先运行 `$NODE $HVIGOR --stop-daemon`，再重新构建。

Lint & Format: 依赖 DevEco Studio IDE 内置检查；仓库根 `code-linter.json5` 是 codelinter 的规则配置（性能/TS 语法/安全规则集），命令行批量检查可配合 codelinter CLI 使用。

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
- Token 自动携带：从 `PreferencesUtil` 读取，收到 `code=10100`（认证失败）时自动 refresh 并重试
- 幂等键：MUST 幂等接口以 `spec/api_spec_base.md`「必须使用幂等键的接口（MUST）」列表为准；实现中 `MUST_IDEMPOTENT_ENDPOINTS` 须与该列表同步（增删端点时需更新），自动注入 `Idempotency-Key`
- Cursor 不透明：请求时透传 `cursor`，收到响应后将 `next_cursor` 作为下一页的 `cursor` 透传，不解析、不构造
- RPC + JSON：仅 POST，不实现 REST 风格

## Token 与安全

- Token 通过 `PreferencesUtil` 存取
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

- 颜色/字号/间距/圆角等一律走 `DesignConstants` / `AppThemeManager` token，禁止硬编码——强制细则见下方「UI/UX 设计规范」与 `docs/design-system.md`。
- 核心服务使用单例模式（`GlobalStateManager.getInstance()`）

## 组件原语强制（对齐前端 button/input 体系）

- 通用文本操作按钮必须用 `SystemButton`（variant=primary/secondary/outline/ghost/danger，
  surface/surface-danger，size=small/default/large/sheet，支持 loading/disabled/fullWidth）；禁止手写
  `Button('文案') + backgroundColor/fontColor/height/borderRadius` 重复样式。
- 通用单行输入框必须用 `SystemInput`（前缀图标/清除按钮/错误态）；登录/表单等
  密码框可用 `SystemInput({ passwordMode: true })`。
- 已登记例外（可保留手写原生控件）：头像/图形按钮、地图全屏控件、搜索建议行、
  动态卡片内嵌操作、主题色板 swatch、扫描/贴纸等特殊控件。存量页面中的手写
  Button/TextInput 属遗留债务，改到该文件时优先迁移为原语。当前存量例外清单：
  - ProfilePage 外部主页/升级/编辑流程按钮（走 `getProfileVisualStyle()` 主题样式对象，
    已 token 化，不套 SystemButton）；
  - 创建页「移除」等 48vp 大 chip、群组详情「恢复原头像」ghost 小按钮；
  - SettingsPage「退出登录」（50vp 透明底定制）、DebugPanel 调试按钮；
  - ChatPage 隐藏键盘失焦占位 `Button('')`、LocationPicker 搜索栏内置 searchButton；
  - ProfileEditPage 多行简介输入辅助、ChatPage 输入栏（含发送钮 `Button('发送')` 属已登记例外 — 需 ChatTheme/getChatVisualStyle 定制动效与 isSendInteractive 动态态，见 ChatPage.ets 顶部“聊天发送钮属已登记例外”注释）、CreateGroupPage 人数步进输入。

## UI/UX 设计规范

设计 token 的**全部数值档位**（颜色/字号/间距/圆角/边框/动效/交互反馈/透明度/尺寸/阴影/模糊）、双端共享组件映射表与跨端投影契约速查的唯一权威是 [docs/design-system.md](../docs/design-system.md)（仓库根 docs/），改 UI 前必读。本节只保留强制规则与指针。

- **禁止硬编码**：颜色一律使用 `AppThemeManager.getPaletteById(resolvedThemeId)` 的 palette token（或 `DesignConstants` 系统/图片固定色），禁止硬编码色值；字号/间距/圆角/边框/动效/透明度/尺寸一律用 `DesignConstants` 对应 token 档位，禁止裸 `fontSize(N)` / `padding({left:N})` / `width(N)` / `duration: N` / `scale(...)`；`$r('app.color.*')` 资源仅用于启动窗口等系统场景（当前 color.json 只保留 start_window_background）。
- **跨端同步机制（强制）**：前端与鸿蒙共享同一套设计 token 刻度，同步由 CI 脚本强制拦截，**改两端任一 token 必须同步改另一端**：

  | 校验 | 脚本 | 覆盖范围 |
  |---|---|---|
  | 跨端色彩 | `frontend/scripts/check-cross-platform-colors.mjs` | `themes/*.css --cst-palette-*` ↔ `ThemeConstants.ets ColorConfig` 逐值一致（3 主题 × 2 明暗 × 14 组映射）+ 主题 swatch |
  | 跨端刻度 | `frontend/scripts/check-cross-platform-scale.mjs` | 字号 9 档（`--text-*` ↔ `FONT.SIZE`）+ 圆角 5 档（`--radius-*` ↔ `RADIUS`）逐值一致；单端加近邻档即拦截。已登记豁免：`FONT.SIZE.LG=16`（HarmonyOS 平台适配档） |

  两脚本均挂入 `bun run check:theme`（随 `bun run check` 与 CI 生效），漂移即红。
- **视觉结构同步约定**：前端 `components/home|messages|profile/` 下组件与鸿蒙 `components/`、`features/*/` 对应实现构成 design-system.md 第 13 节映射表——修改任一端的**共享组件**（改动视觉契约：布局/字号档/圆角档/投影/动效）时，必须检查映射表对端并同步，commit message 中注明「对齐 <前端组件/commit>」；仅平台适配差异（如 FAB 位置、安全区）可不同步但需在两端代码注释中互指。
- **页面头部（强制 SystemPageHeader）**：
  - 所有页面头部必须用 `SystemPageHeader`（或专用组件 `SearchPageHeader` / `ChatPageHeader`）；禁止手搓 `Row { Text(...).fontSize(XL)... Button(...) }` 头部。
  - 已登记例外（无标准页面头部）：登录页 `LoginPage`（无主壳）、主壳 `Index`（容器）、地图全屏 `LocationPickerPage`、图片全屏预览 `PhotoPreviewPage`、分享解析中间页 `ShareResolutionPage`（无内容 UI，仅 loading/错误/跳转）、调试面板 `DebugPanel`；`SettingsSupport.ets` 等仅导出页面子组件的文件不算页面。
  - SystemPageHeader API：`title` / `subtitle` / `showBackButton`+`onBack` / `action`(文本按钮) / `leftAction`+`rightAction`(IconAction) / `trailing`(@BuilderParam 任意尾随) / `titleAlign` / `titleSize` / `titleWeight` / `backgroundVariant`('bar'|'transparent'|'card')。
  - 安全区由 SystemPageHeader 内聚（自带 `@StorageLink('safeAreaTop')` + `expandSafeArea`），页面无需重复写。
  - 内联搜索框头部 → `SearchPageHeader`；浮动卡片+头像头部 → `ChatPageHeader`。
- **页面内容栏（PageShellContainer）**：强制使用；档位数值（form/narrow/default/wide + feed 两档）见 design-system.md 第 12 节。

### 页面路由对齐（web → harmony）

- 目录命名对齐前端路由域（复数）：`features/messages`（消息列表）/
  `features/moments`（瞬间）/ `features/groups`（群组）/ `features/me`（个人中心，
  同时承载前端 `/users/[userId]` 他人主页查看——同一 ProfilePage 复用，属平台例外）/
  `features/nearby` / `features/settings` / `features/search` / `features/map` /
  `features/auth` / `features/chat`（聊天窗口独立域）。
- 底栏 4 tab：首页 / 附近 / 消息 / 我的（`PhoneLayout` TAB_ITEMS + `AppNavigation.BOTTOM_TABS`，
  与前端 `routes/(app)/+page.svelte`、`/nearby` 聚合页对齐）。
- 桌面侧边栏：首页 / 附近瞬间（moments tab）/ 附近的人 / 附近群组 + 消息 / 通知 overlay /
  我的瞬间 / 我的群组 / 关系（对齐前端 desktop nav 的 home / nearby-moments /
  nearby-people / nearby-groups / messages / notifications / moments / my-groups /
  relationships）。
- 消息列表：排序 = 置顶优先 → 最新一条消息时间倒序（`last_message.sent_at` 优先，
  回退 `last_activity_at`），已读/未读不参与排序；列表头部有「全部 / 未读 / 已读」过滤
  （`TabsPillGroup`，对齐前端 messages 页 TabBar）；点击未读会话后徽章立即清零。
- 聊天记录：消息按 `sent_at` 正序渲染（越新越靠下，对齐前端 useChat 翻转）；
  跨天自动插入日期胶囊（`FormatUtil.formatDateLabel` → 今天/昨天/M月D日，
  对齐前端 groupMessagesByDay/formatDateLabel）。
- 动态详情：点赞/收藏/分享/评论（评论列表 + 乐观发布，对齐前端 MomentDetailPage）。
- 附近聚合页 `NearbyPage` 三区块「查看全部」分别 push `NEARBY_USERS` /
  `NEARBY_GROUPS` / `NEARBY_MOMENTS`（MomentsFeedPage 已注册为路由页，push 场景
  带返回按钮，对齐前端 `/nearby/people|groups|moments`）。
- 搜索页 Tab：全部 / 用户 / 动态 / 话题（话题从动态标签聚合，对齐前端搜索
  全部/用户/动态/话题）；群组 Tab 为鸿蒙端扩展（前端搜索无群组 Tab）。
  空查询显示发现态（最近搜索 / 正在关注 / 推荐话题，对齐前端 SearchDiscovery）；
  最近搜索持久化在 PreferencesUtil（键 `search_recent`，最多 5 条）。
- WS 事件类型与 spec/前端对齐：含 `GroupMessageReadCountUpdated` /
  `GroupMemberRoleChanged`（已登记类型与订阅，展示由群详情/聊天页按需刷新）。
