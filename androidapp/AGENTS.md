# HotSpot Android 客户端（Kotlin + Jetpack Compose）

本文件是 androidapp 专属约束；通用原则（Git/CI/Docker/设计系统）以根 `AGENTS.md` 为准。

## 定位与范围

按 Figma 设计稿（多端 UI Design Kit 移动端 390 屏）实现的 Android 客户端，与 `frontend` / `harmonyapp`
同源同设计系统。**首版为可构建、可联调的骨架**：设计 token、原语、应用壳（底部 5 tab + 登录态门禁）、
登录/注册/访客模式（390 设计稿 87:1818 版式）、首页信息流 / 附近 / 消息（私聊 + 群聊会话混合列表）/ 我的 / 搜索 / 通知 / 设置 /
我的瞬间（发布 + 收藏，含 private 徽章与软删占位）/ 我的群组（成员视角 + 本地过滤）/
瞬间详情（含评论楼中楼列表、常驻贴底评论输入条、收藏与系统分享）/ 群详情（设计稿 87:1696：封面卡 + 简介/成员/公告
三 tab + 加入/退出群组，入口接我的群组行、附近群组卡、搜索群组结果）/ 群聊会话（设计稿 87:3529：
群头像/群名/成员数头部 + 多人群消息（他人头像+昵称、自己右对齐、系统与撤回消息居中）+ 文本发送，
入口 = 群详情「打开群聊」）/
个人中心（设计稿 87:3393：资料卡 + 瞬间/粉丝/群组指标 + 编辑资料/设置 + 我的内容 5 行计数入口）/
关系页（87:3300：好友/关注/粉丝三 tab + 搜索；好友发消息、关注两态）/
编辑资料（87:3128：昵称/简介/性别/兴趣标签 ≤5，users/update_account patch）/
设置（87:4864：账号卡 / 三主题预览卡 × 明暗三档 / 系统与关于）/
我的举报（87:3350：状态筛选 + 处理结果，reports/fetch 分页）/
他人主页（87:5163：资料卡 + 瞬间/粉丝/关注 + 关注/私信 + 最近瞬间带图画廊 + 分享/备注/屏蔽/举报；
入口：枢纽用户行、附近的人卡片、附近瞬间作者、瞬间详情作者、搜索用户结果）/
发布瞬间（无独立设计画板，结构对齐 Web CreateSheet：文案 + 相册选图 + 氛围标签 + 可见范围 + 默认浏览位置；
media/presign_upload → PUT → complete_upload → moments/create）已实现；
地图原生渲染、WebSocket 实时、管理后台为**代码内登记 TODO 占位**
（展示真实空态，禁止造假数据）。

API 契约见根 `spec/`（只读权威）。距离/位置由后端计算，客户端只展示；所有 `xxx_id` 按 `string` 处理；
- **发现类请求显式传浏览锚点**：附近/发现接口（`users/search`、`groups/search`、`moments/fetch_nearby`）必须显式传
  `reference_point`（取 `client_config.defaults.default_location.point`，见 `AppContainer.defaultLocation()`）——
  服务端回退链在 groups/moments 上实测不一致（不传锚点时 groups 返回空），客户端显式传参规避；未接设备定位前，
  导航胶囊位置文案也用该默认位置的 `location_name`，不写死地名。
- 附近的人列表需排除当前登录用户（对齐 Web/鸿蒙；`users/set_follow` 对自身返回 10400）。
后端 int64 字段按 spec 的 string 处理，不做数值运算。

## 设计 token 链路（禁止手改生成物）

```
frontend/src/styles/theme/**  （Web CSS = 唯一事实源）
  → node scripts/design-tokens/gen-android.mjs            # 生成
      → app/src/main/java/com/hotspot/app/ui/theme/Tokens.kt
      → app/src/main/assets/design-tokens.json
  → node frontend/scripts/check-cross-platform-android.mjs # 逐值校验（挂 bun run check:theme + CI）
```

- 改颜色/字号/圆角/动效 → **先改 Web CSS，再跑生成器并提交生成物**；直接改 `Tokens.kt` 会被 CI 打红。
- 生成物覆盖 3 主题 × 明暗：palette 23 项、语义色 76 项、氛围画布 12 项、字号 9 档、圆角 6 档、动效 16 档。
- 消费面：`HotSpotTheme.colors`（`HotSpotColors` CompositionLocal，语义 token）、`HotSpotTypography`、
  `AppShapes`、`Dimens`/`Spacing`、`Motion`。页面禁止硬编码颜色/字号/圆角/动效。

## 技术栈与构建

- Kotlin 2.4.0 / AGP 9.2.1 / Gradle 9.7.0（wrapper 已入库）/ Compose BOM 2025.12.01 + material3 /
  navigation-compose 2.9.0 / OkHttp 5.4.0 + kotlinx-serialization-json 1.11.0 / coil-compose 2.7.0
- compileSdk & targetSdk 37、minSdk 26、JVM target 17、包名 `com.hotspot.app`

```bash
cd androidapp
JAVA_HOME="/Applications/Android Studio.app/Contents/jbr/Contents/Home" \
ANDROID_HOME="$HOME/Library/Android/sdk" \
./gradlew --no-daemon assembleDebug          # APK: app/build/outputs/apk/debug/app-debug.apk
JAVA_HOME=... ANDROID_HOME=... ./gradlew --no-daemon testDebugUnitTest
JAVA_HOME=... ANDROID_HOME=... ./gradlew --no-daemon assembleRelease   # R8 + 资源压缩（产物未签名）
# token：仓库根 node scripts/design-tokens/gen-android.mjs [--check]
```

- **一律 `--no-daemon`**，收尾确认 `pgrep -fl 'GradleDaemon|KotlinCompileDaemon'` 为空。
- **发布签名**：签名为本机行为（material 不入库、CI 不持密钥）。提供 `HOTSPOT_KEYSTORE_FILE` /
  `HOTSPOT_KEYSTORE_PASSWORD` / `HOTSPOT_KEY_ALIAS` / `HOTSPOT_KEY_PASSWORD` 四个环境变量时
  `assembleRelease` 产出已签名 APK，否则保持未签名；版本号用 `HOTSPOT_VERSION_NAME`/`HOTSPOT_VERSION_CODE`
  覆盖（默认与当前发布对齐）。完整命令与 keystore 位置见 `docs/release-runbook.md`「Android APK 分发」。
- 本机不要提交 `local.properties`（已在 .gitignore）；`ANDROID_HOME` 走环境变量。
- CI 覆盖：`.github/workflows/ci.yml` 的 `Android (Compose Build)`（JDK 21 + SDK 37 + token 校验 +
  assembleDebug/testDebugUnitTest）；本地不必重复跑全量构建，但改 UI 后至少跑一次 assembleDebug。

## 目录与分层（与 harmonyapp 对齐）

```
app/src/main/java/com/hotspot/app/
  MainActivity.kt / HotSpotApplication.kt / AppContainer.kt   # 入口与依赖装配
  data/api/        # ApiClient（OkHttp + 信封解析）、Endpoints（唯一路径定义处）
  data/auth/       # TokenStore（Bearer 持久化）
  data/model/      # spec 字段 DTO
  data/repo/       # 按业务域的仓库（认证/信息流/附近/搜索/私聊/评论/资料/通知/群组详情）
  navigation/      # Routes / AppNav（登录态门禁 + 底部 5 tab）
  ui/theme/        # Tokens（生成物）/ Theme / ThemeController / Type / Shape / Dimens / Motion
  ui/components/   # 原语：SystemButton/SystemInput/SearchInput/Badge/Tabs/Surfaces/Avatar/States/
                   #       PageHeader/BottomNav/AmbientCanvas/MomentCard/Focus
                   # 评论区：CommentThread（楼中楼+Canvas 导引线）/ CommentInputBar（docked+IME）
                   #       / CommentAuthorBadge（作者微徽章）
  ui/screens/      # 按业务页面（Home/Nearby/Messages/Chat/Me/Search/Notify/Settings/Login/...）
  util/            # 格式化等纯函数；CommentThreading（评论归组/分页去重/乐观点赞）
```

## API 与网络

- 仅 POST + JSON（RPC 风格），路径必须经 `data/api/Endpoints.kt`（禁止页面硬编码路径）。
- 成功判定只认 `code === 0`；`10100` 清会话回登录；MUST 幂等接口带 `Idempotency-Key`。
- Base URL：默认 `http://10.0.2.2:8888`（指向宿主 nginx 网关）；模拟器/真机一律
  `adb reverse tcp:8888 tcp:8888` + `-PHOTSPOT_DEBUG_BASE_URL=http://127.0.0.1:8888` 覆盖构建；
  release 默认线上站点且强制 HTTPS；debug 明文仅放行本机地址（network_security_config）。
- **模拟器网络判据**：本机 API 36 AVD 上端内请求 `10.0.2.2:8888` 会连接超时（nginx 无访问日志），
  而 `adb shell nc 10.0.2.2 8888` 可连通——**shell 连通性不能证明端内可达**。验证端内可达看
  `docker logs hotspot-nginx-1` 是否有对应 POST；走查流程见 skill `hotspot-android-device-verify`。

## UI 契约

- 图标族：导航/功能图标统一**线框矢量**（对齐设计稿与 Web Lucide 语义），底部导航用 `res/drawable/ic_nav_*`
  （home/compass/plus/message-circle/circle-user）；**禁止用 `Icons.Filled.*` 实心 Material 图标充当导航图标**。
- **禁止 Unicode 字符当图标**（`›`、`⌄`、`❝❝`、`✓`、`▶` 等）：用 vector drawable（`ic_*`，路径对齐 Lucide 同名图标）
  + `Icon(painterResource(...), contentDescription = …)`；emoji 仅可作为反应/内容数据，不得进图标位。
- 原语优先：按钮 `SystemButton`、输入 `SystemInput`、徽章 `Badge`、Tab `SegmentedTabs/TabPills`、
  软岛 `Surfaces`、空/加载/错误态 `States`、页头 `PageHeader`、底部导航 `BottomNav`。禁止页面手写重复样式。
- 触控热区 ≥44dp；焦点可见；尊重系统动画缩放（reduced-motion 直接终态）；图片统一 `Avatar`/coil 渐显。
- **安全区一律取系统 `WindowInsets`**：根 `MainActivity` 统一注入 `safeDrawing` 的 Top/Horizontal；底部贴底组件
  （BottomNav / CommentInputBar / GroupActionDock）自行使用 `navigationBars`/`ime`。**禁止在页面写死顶/底部安全区 dp 值**——
  状态栏高度、挖孔、手势条因设备而异，写死会顶出多余留白或压到系统栏。
- 列表必须三态（loading/empty/error）；分页列表有到底提示；不得用假数据填充空态。
- 主题：3 主题（vibrant/forest/inkwash）× 明暗三档（跟随系统/浅/深），系统栏颜色随主题。
- 新增页面前先对照设计稿（Figma）移动端对应画板与 `docs/ui-components.md` 的同类契约。

## 已知待办（首版登记）

1. token 存储从 SharedPreferences 迁移到 EncryptedSharedPreferences/Keystore（上线前必须）。
2. 地图原生渲染、WebSocket 实时、管理后台；
   群聊会话页仅首版键盘输入（WebSocket 实时、@ 提及、已读人数、撤回/编辑、富媒体消息未做）；
   群管理（改群信息 / 公告与成员管理 / 转让群主）与举报群组未做；
   评论首版未做：删除自己的评论、举报评论、点赞回复、查看全部回复（top_replies 预览之外的回复不展开）。
3. 真机走查（模拟器走查流程见 skill `hotspot-android-device-verify`；`assembleRelease` 已在本地
   验证通过，CI 仍只跑 debug）。
