# HotSpot Android 客户端（Kotlin + Jetpack Compose）

本文件是 androidapp 专属约束；通用原则（Git/CI/Docker/设计系统）以根 `AGENTS.md` 为准。

## 定位与范围

按 Figma 设计稿（多端 UI Design Kit 移动端 390 屏）实现的 Android 客户端，与 `frontend` / `harmonyapp`
同源同设计系统。**首版为可构建、可联调的骨架**：设计 token、原语、应用壳（底部 5 tab + 登录态门禁）、
登录/注册/访客模式（390 设计稿 87:1818 版式；忘记密码：邮箱/手机号验证码重置，成功后自动登录，dev 验证码走 Mailpit）、首页信息流（整卡列表；卡片爱心可直接点赞，`reactions/set target_type=moment` 乐观更新；点卡进详情、点作者进主页）/ 附近 / 消息（私聊 + 群聊会话混合列表；长按会话置顶/免打扰，私聊可删除会话，`private_chats/update_preferences|delete`、`groups/update_preferences`）/ 我的 / 搜索（用户/瞬间/群组分页 + `search/suggest` 300ms 防抖联想，点击直达）/ 通知 / 设置 /
我的瞬间（发布 + 收藏，含 private 徽章与软删占位）/ 我的群组（成员视角 + 本地过滤 + 页头创建群组；顶部展示待处理入群邀请（接受/拒绝）与我提交的审核中申请（撤回），`groups/invites/fetch_received|accept|reject`、`groups/applications/fetch_submitted|cancel`）/ 创建群组（groups/create：可选封面/名称/简介/分类/公开或仅邀请/新人审核/默认位置）/
瞬间详情（点赞/取消点赞，reactions/set target_type=moment 乐观更新；含评论楼中楼列表、常驻贴底评论输入条、收藏与系统分享（分享走 `share/create` 生成可撤销链接，失败降级原始详情链接）；hero 支持多图横滑与点击全屏查看；作者可编辑文案/删除瞬间，非作者可举报）/ 群详情（设计稿 87:1696：封面卡 + 简介/成员/公告
三 tab（公告 tab 对 owner/admin 提供发布/编辑/删除，其余角色只读）+ 加入/退出群组；
更多菜单「分享群组」走 `share/create` 生成 /share/<code>，失败降级原始群组链接，
入口接我的群组行、附近群组卡、搜索群组结果；成员 tab 支持按昵称搜索 `groups/members/search`，350ms 防抖）/ 群聊会话（设计稿 87:3529：
群头像/群名/成员数头部 + 多人群消息（他人头像+昵称、自己右对齐、系统与撤回消息居中）+ 文本发送，
入口 = 群详情「打开群聊」）/
个人中心（设计稿 87:3393：资料卡 + 瞬间/粉丝/群组指标 + 编辑资料/设置 + 我的内容 5 行计数入口）/
关系页（87:3300：好友/关注/粉丝/黑名单四 tab + 搜索；好友发消息、关注两态、黑名单解除屏蔽）/
编辑资料（87:3128：昵称/简介/性别/兴趣标签 ≤5，users/update_account patch）/
设置（87:4864：账号卡 / 账号安全（访客升级 / 修改密码 / 登录设备 / 删除账号；访客账号在设置内设置用户名与密码升级为正式账号并保留内容）/ 隐私与安全（账号可见性 / 位置共享 / 附近新瞬间通知）+ 互动与发现（允许被搜索 / 允许关注 / 允许私信，均走 `users/update_settings`）/ 三主题预览卡 × 明暗三档 / 地图与位置区块 / 系统与关于（版本 / 检查更新（GitHub Releases）/ 我的分享 / 我的举报 / 屏蔽名单 / 退出登录）；
平台管理员额外显示「管理后台」入口）/
我的分享（share/fetch_created 按类型筛选 + 分页，复制链接 / 撤销；入口：设置 →「我的分享」）/ 我的举报（87:3350：状态筛选 + 处理结果，reports/fetch 分页；支持按对象 ID/原因/描述搜索，契约无关键词参数时先加载完全部分页再本地筛选）/ 屏蔽名单（入口：设置 →「屏蔽名单」，users/fetch_blocked 列表 + 解除屏蔽）/ 通知中心（平台公告（system/announcements/fetch，最多 3 条，按 severity 显示通知/警告/重要）+ 列表 + 分页加载更多 + 未读标记 + 全部已读；支持按通知内容/用户 ID/类型搜索，同样先加载完全部分页再本地筛选，
点击按 payload 跳转：用户主页/瞬间详情/群详情/群聊/我的举报，并单条标记已读）/
他人主页（87:5163：资料卡 + 瞬间/粉丝/关注 + 关注/私信 + TA 的群组（users/fetch_groups 首屏最多 5 个，点击进群详情）+ TA 反应过的瞬间（users/fetch_reacted_moments 首屏最多 4 个）+ 最近瞬间带图画廊 + 分享/备注/屏蔽/举报；
分享资料优先 `share/create` 生成可撤销链接，失败降级原始主页链接；
入口：枢纽用户行、附近的人卡片、附近瞬间作者、瞬间详情作者、搜索用户结果）/
发布瞬间（无独立设计画板，结构对齐 Web CreateSheet：文案 + 相册选图 + 氛围标签 + 可见范围 + 地图选点位置（默认回退当前浏览位置）；
media/presign_upload → PUT → complete_upload → moments/create）已实现；
地图选点页已接入（Figma 移动端 `87:1866`「地图选点 · /map」：全屏底图 + 顶部返回/筛选浮动条 +
右侧缩放/回中/图层控件 + 中心定位针 + 底部搜索/最近使用/「设为浏览锚点」面板；锚点持久化到本地
并驱动附近页 reference_point；同一页支持 `pickMode` 供瞬间详情修改位置复用；不接第三方地图 SDK，底图暂为示意底色）；
管理后台（设置页对平台管理员显示入口，`rbac/me` 鉴权）已接入：
- **举报审核**：待处理举报列表 → 通过/驳回 + 处理备注（`reports/fetch|judge`）；
  支持按举报人/对象 ID/原因关键词搜索（契约无关键词参数：搜索时先加载完全部分页，再本地筛选）；
- **用户管理**：昵称搜索 + 正常/已封禁筛选，封禁（永久/1/7/30 天 + 原因）与解封（`users/search`、`users/set_ban`）；
  点击用户行打开详情（`users/open_the_door`：账号/隐私/活跃时间，IP 与注册地等 PII 仅 `admin.users.read_ip` 可见，权限不足显示占位）；
- **群组管理**：群名搜索，展示群主/成员数/创建时间，解散群组（`groups/search`、`groups/dismiss`；
  原因仅进审计日志，成员收到 GroupDismissed 事件）；
  点击群组行打开详情（`groups/open_the_door`：群主/成员数/标签/创建与最近活跃/简介；解散仍在列表操作）；
  详情内展示只读成员列表（`groups/members/fetch`，按加入时间正序 + 加载更多；成员可见性受限时显示错误与重试）；
  成员列表支持按昵称搜索（`groups/members/search`，350ms 防抖，搜索模式独立分页与空态）；
- **系统公告**：按级别（全部/通知/警告/错误）筛选，发布/编辑/删除公告，可选同时推送（`announcements/fetch|create|update|delete`）；
  支持按标题/正文/发布人搜索（同样先加载完全部分页再本地筛选）；
- **统计分析**：全站存量（用户/群组/瞬间/合计）、近 7 日新增柱状图、热门地点与 S2 区域 Top（`stats/geo`）；
  区域 Top 可下钻查看用户/群组/瞬间明细（`stats/geo/drilldown`，按实体类型切换 + keyset 分页；
  支持按标题/对象 ID/位置搜索，契约无关键词参数时先加载完全部分页再本地筛选；
  用户/群组行可继续打开对应管理详情，瞬间行只读）；
- **权限与角色**：查看平台管理员列表（角色/权限摘要/授权时间）、RBAC 角色与权限字典，
  支持授权（按角色授权或 superadmin 授予全部权限）与撤销（superadmin 行不显示撤销入口，避免误撤自己）
  （`rbac/admins/fetch|award|revoke`、`rbac/roles/fetch`、`rbac/permissions/fetch`）；
  角色支持新建/编辑名称描述/删除/按权限字典勾选 set_permissions（`'*'` 全量权限仅 superadmin，未在 App 暴露）；
- **标签管理**：标签名搜索（含已屏蔽项）、使用次数排序、软屏蔽/解屏蔽（`tags/search|set_ban`；
  物理删除 hard_delete 不在 UI 暴露）；
- **审计日志**：按目标类型（用户/群组/公告/角色/管理员/举报/标签/配置）、动作（18 种）与时间范围（全部/近 7 天/近 30 天/自定义起止日期）筛选，展示操作者、动作、
  对象、详情、IP 归属与设备摘要（`audit/fetch`，`ip_address` 需 `admin.audit.read_ip`）；
  支持按操作者/动作/对象/详情搜索（契约无 keyword：搜索时先加载完全部分页再本地筛选）；
  点击行打开详情弹窗（操作者/对象/绝对时间/IP 归属与地址/设备/详情键值/日志 ID），
  用户/群组对象可一键跳转到对应管理列表并按名称预填搜索（`admin/users/filter`、`admin/groups/filter`）；
管理后台模块（举报/用户/群组/公告/统计/权限角色/标签/审计日志）已接入；其余扩展仍为**代码内登记 TODO 占位**（展示真实空态，禁止造假数据）；
WebSocket 实时已接入（`data/realtime/`：连接/Auth/动态心跳/指数退避重连/resume 或全量订阅恢复，
私聊、群聊与消息列表消费增量事件）。

API 契约见根 `spec/`（只读权威）。距离/位置由后端计算，客户端只展示；所有 `xxx_id` 按 `string` 处理；
- **发现类请求显式传浏览锚点**：附近/发现接口（`users/search`、`groups/search`、`moments/fetch_nearby`）必须显式传
  `reference_point`（优先取用户在地图选点页设置的本地锚点，未设置时回退
  `client_config.defaults.default_location.point`，见 `AppContainer.defaultLocation()`）——
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
  data/auth/       # TokenStore（Bearer 持久化，Keystore AES/GCM 加密）
  data/model/      # spec 字段 DTO
  data/repo/       # 按业务域的仓库（认证/信息流/附近/搜索/私聊/评论/资料/通知/群组详情）
  data/location/   # 浏览锚点本地存储（地图选点页写入；附近页 reference_point 优先用它）
  data/media/      # 聊天媒体上传（chat_image 10MB；chat_video 100MB MP4/MOV；chat_file 100MB + 黑名单）
  data/audio/      # 应用级语音播放控制器（全局单播放器，聊天页退出停止）
  data/realtime/   # WebSocket 实时通道：HotSpotRealtimeClient（OkHttp WS）+ RealtimeProtocol（纯函数解析/退避/订阅集合）
  navigation/      # Routes / AppNav（登录态门禁 + 底部 5 tab）
  ui/theme/        # Tokens（生成物）/ Theme / ThemeController / Type / Shape / Dimens / Motion
  ui/components/   # 原语：SystemButton/SystemInput/SystemTextArea/SearchInput/Badge/Tabs/Surfaces/
                   #       Avatar/States/PageHeader/BottomNav/AmbientCanvas/MomentCard/Focus
                   # 评论区：CommentThread（楼中楼+Canvas 导引线）/ CommentInputBar（docked+IME）
                   #       / CommentAuthorBadge（作者微徽章）
  ui/screens/      # 按业务页面（Home/Nearby/Map/Messages/Chat/Me/Search/Notify/Settings/Login/...）
  util/            # 格式化等纯函数；CommentThreading（评论归组/分页去重/乐观点赞）
```

## API 与网络

- 仅 POST + JSON（RPC 风格），路径必须经 `data/api/Endpoints.kt`（禁止页面硬编码路径）。
- 成功判定只认 `code === 0`；`10100` 清会话回登录；MUST 幂等接口带 `Idempotency-Key`。
- **Token 安全**：access/refresh token 使用 Android Keystore AES/GCM 加密后写入 SharedPreferences
  （密钥不可导出、每条密文带随机 IV）；旧版本明文 token 首次读取时自动加密迁移并删除明文键；
  Keystore 解密失败按未登录处理；`device_id`/`user_id` 非敏感保持明文。
- Base URL：默认 `http://10.0.2.2:8888`（指向宿主 nginx 网关）；模拟器/真机一律
  `adb reverse tcp:8888 tcp:8888` + `-PHOTSPOT_DEBUG_BASE_URL=http://127.0.0.1:8888` 覆盖构建；
  release 默认线上站点且强制 HTTPS；debug 明文仅放行本机地址（network_security_config）。
- 端上走查设备真机优先（`adb devices -l` 中状态 `device` 且 serial 非 `emulator-*`），无合格真机才用模拟器；
  选中后所有 `adb` 命令固定 `-s <serial>`，判据与流程见 skill `hotspot-android-device-verify`「设备选择」。
- **WebSocket**：端点 `/ws/v1`（spec/api_spec_websocket.md），App 端连接后 5 秒内发 `Auth`
  （access_token + device_id + resume_token）；心跳参数由 `ConnectionEstablished.config` 下发，禁止硬编码；
  断线指数退避重连（4002 按 `Error.retry_after` 等待），4001 先 refresh token（仅 10100/会话失效才清会话回登录，
  网络失败保留会话退避重试；fatal Error 不直接登出，登出以 close code 收尾为准），1000/1008 不重连；
  聊天页进入时订阅对端 user / 群 group 频道，离开取消，重连后按本地集合全量 `set` 恢复；
  输入中状态：输入节流 2s 发 `SendPrivateTyping/SendGroupTyping`，停止 3s 发 false；接收 6s 无新事件自动清除
  （规则与文案见 `util/ChatTypingRules.kt`，群聊多人折叠为「A、B、C 等 N 人正在输入…」）。
- **群聊 @ 提及**：输入 `@` 弹出群成员候选（缺省拉取 `groups/members/fetch`），插入「@昵称 」；
  发送时 `group_messages/send` 的 `mentions` 只提交最终文案里仍存在的被 @ 成员 user_id
  （`util/MentionRules.kt`）；消息气泡内 `@昵称` 加粗高亮。
- **聊天语音**：输入条麦克风按钮按住说话（按下开始录音，上滑超过 72dp 进入「松开取消」，
  松手发送；录音中显示红点/计时与操作提示）→ `RECORD_AUDIO` 权限 →
  `ChatAudioUploader`（chat_audio，20MB / audio/aac，AAC-ADTS，流式上传）→ `type=audio`；
  无权限时首次点按走系统授权，授权后沿用带取消/发送按钮的录音栏；
  语音气泡点击由应用级 `AudioPlaybackController` 播放/停止（全局单播放器，切换语音先停上一条），
  播放中显示进度条与「已播放 / 总时长」（每 200ms 轮询），播完/停止后显示实际总时长；
  服务端无 duration 时以播放器实际时长为准；离开聊天页自动停止。
- **聊天图片/视频**：输入条相册按钮 → 系统 Photo Picker（ImageAndVideo）→ 按 MIME 分流：
  `ChatImageUploader`（chat_image，10MB / JPG·PNG·WebP·GIF）或 `ChatVideoUploader`
  （chat_video，100MB / video/mp4·video/quicktime，流式上传）→ `private_messages/send` /
  `group_messages/send` 的 `type=image|video` + `file_id`；图片气泡用 Coil 加载 `media.url`，
  加载失败回落「[图片]」；点击图片打开全屏查看器（黑底、双指缩放/拖拽、双击放大/复位、关闭）；
  视频气泡优先展示 `media.thumbnail_url` + 播放角标，无缩略图用深色占位，点击用
  系统 ACTION_VIEW 打开签名 URL。
- **封禁成员**：成员操作面板「封禁成员」→ 二次确认，走 `groups/admin/members/set_ban`（active=true
  移出群组并禁止再加入）；更多菜单「封禁列表」→ `fetch_banned` 展示封禁原因/时间并支持解除封禁，
  解除后仅移除封禁状态、不自动拉回群组。
- **邀请成员**：更多菜单（owner/admin）→「邀请成员」→ 搜索昵称/账号 → 邀请；
  走 `groups/admin/invites/create`，被邀请者已是成员时服务端 no-op（invite_id 为 null），客户端提示「对方已是群成员」。
- **邀请记录**：更多菜单（owner/admin）→「邀请记录」展示全部发出邀请（待接受/已加入/已拒绝/已撤销，含邀请人与消息），
  待接受可撤销（`groups/admin/invites/fetch_sent` status=all + `invites/revoke`，二次确认）。
- **入群申请审批**：更多菜单（owner/admin）→「入群申请」列表，逐条同意/拒绝（`groups/admin/applications/fetch|approve|reject`）；
  同意后刷新成员列表与详情，拒绝后仅从收件箱移除。
- **转让群主**：成员操作面板中仅 owner 显示「转让群主」→ 二次确认 → `groups/admin/transfer_ownership`；
  成功后刷新详情与成员列表，原 owner 变管理员、新 owner 徽章即时更新。
- **群设置**：更多菜单（owner/admin）→ 搜索可见性、成员列表可见性、入群审批、人数上限、
  加入方式（公开/密码/仅限邀请）与群密码（8-32 位；留空保持原密码，可清除）；
  读取 `groups/admin/settings`，保存走 `groups/admin/update`（只提交改动字段，其余 null 不改；
  password 与 clear_password 互斥）。
- **群成员管理**：成员 tab 中 owner/admin 对非群主、非自己的成员显示「管理」→ 操作面板
  （设为/取消管理员、禁言 1 小时/解除禁言、移除成员）；分别走 `groups/admin/members/set_role`、
  `set_mute`、`remove`，群主/自己不可被操作，服务端 10400 兜底；成功后刷新成员列表与群详情。
- **创建群组**：我的群组/附近群组页头「+」→ 可选封面（`group_avatar` 上传）/名称（≥2 字）/简介/分类/公开或仅邀请（公开时可选「直接加入 / 申请需审核」）+ ClientConfig 默认位置 → `groups/create`（幂等键、owner=当前用户，提交 `avatar_file_id` 与 `requires_approval`），成功直达群详情。
- **群信息编辑**：群详情更多菜单（owner/admin）→ 群头像/名称/简介弹窗（群头像 Photo Picker → `AvatarUploader(category=group_avatar)` → 保存时提交 `avatar_file_id`），走 `groups/admin/update`；
  其余字段传 null 不改，成功用返回的 GroupItem 直接刷新头部。
- **瞬间操作**：详情右上角「更多操作」按角色分流——自己的瞬间：编辑文案、氛围标签与可见范围
  （`moments/update` patch 语义，只提交 description/tags/privacy_level；标签预设与发布页共用 `MoodChips`，
  并额外展示瞬间已有自定义标签；可见范围初值需先从 `moments/fetch_created` 的 owner 视角取 privacy_level，
  `open_the_door` 不含该字段）/ 修改位置（复用地图选点页的 pick 模式，确认后 `moments/update` 只提交 location；
  坐标输入时位置名回退「自定义位置」）/ 删除瞬间（`moments/delete` 软删除）；
  他人瞬间：举报瞬间（`reports/create target_type=moment`，复用 `ReportDialog`）。
  删除成功后返回上一页并保留列表软删占位规则。
- **瞬间多图**：详情 hero 使用 `HorizontalPager` 横滑（多图时底部圆点指示器），点击当前图打开
  `FullscreenImageViewer`；多图查看器可左右滑翻页（底部「当前/总数」、双击放大复位），
  单图查看器保留双指缩放/拖拽；搜索结果中的瞬间行可点击进入详情。
- **屏蔽与名单**：他人主页「屏蔽」走 `users/set_block`（字段为 `active`，拉黑会解除双方关注）；
  设置 →「屏蔽名单」展示 `users/fetch_blocked` 列表（昵称/屏蔽时间/屏蔽原因），
  分页拉取（底部「加载更多」）并支持逐条解除屏蔽；
  支持按昵称/用户 ID/屏蔽原因搜索（契约无关键词参数：搜索时先加载完全部分页再本地筛选）；
  解除只清拉黑关系，不自动恢复关注。
- **通知跳转**：通知行点击后先 `notify/mark_read`（scope=ids）再按类型跳转：
  `new_follower`/`mutual_follow` → 用户主页；`moment_commented`/`comment_reply`/`mentioned`
  → 瞬间详情并滚动定位到对应评论线程（路由 `moment/{momentId}?commentId=`，取 `comment_id`/`reply_id`；
  目标评论不在首屏时自动续页查找，直到命中或评论到底）；
  `nearby_new_moment` → 瞬间详情；`group_mentioned` → 群聊；群邀请/入群申请 → 群详情；举报处理 → 我的举报；
  无可跳转目标（系统公告等）只标记已读。
- **消息操作**：长按消息（私聊/群聊共用 `MessageActionSheet`）→ 复制（文本）/ 撤回（自己的消息）/
  举报（他人的消息，复用 `ReportDialog`）；撤回走 `private_messages/recall` / `group_messages/recall`，
  服务端时间窗超时返回 10400 时 toast 提示；操作可用性由 `util/MessageActionRules.kt` 统一计算。
- **聊天文件**：输入条相册右侧文件按钮 → 系统 DocumentsUI → `data/media/ChatFileUploader`
  （chat_file 分类，100MB 上限 + 扩展名/MIME 双黑名单，PUT 走流式 `putObjectStream` 不整份读内存）→
  `private_messages/send` / `group_messages/send` 的 `type=file` + `file_id`；气泡展示文件名/大小，点击 `ACTION_VIEW` 打开签名 URL。
- **设置·地图与位置**：展示当前浏览锚点（自定义地点 + 坐标；未设置时回退默认位置）；
  「去地图选点」跳转 `/map`，选点保存后回到设置并刷新；「恢复默认」清除本地锚点；「同步当前位置」把锚点上报为真实位置（`users/update_location`，影响附近发现与主页大致位置）。
- **模拟器网络判据（模拟器基线；真机同样走 adb reverse）**：本机 API 36 AVD 上端内请求 `10.0.2.2:8888`
  会连接超时（nginx 无访问日志），而 `adb shell nc 10.0.2.2 8888` 可连通——**shell 连通性不能证明端内可达**。
  验证端内可达看 `docker logs hotspot-nginx-1` 是否有对应 POST；走查流程见 skill `hotspot-android-device-verify`。

## UI 契约

- 图标族：导航/功能图标统一**线框矢量**（对齐设计稿与 Web Lucide 语义），底部导航用 `res/drawable/ic_nav_*`
  （home/compass/plus/message-circle/circle-user）；**禁止用 `Icons.Filled.*` 实心 Material 图标充当导航图标**。
- **禁止 Unicode 字符当图标**（`›`、`⌄`、`❝❝`、`✓`、`▶` 等）：用 vector drawable（`ic_*`，路径对齐 Lucide 同名图标）
  + `Icon(painterResource(...), contentDescription = …)`；emoji 仅可作为反应/内容数据，不得进图标位。
- 原语优先：按钮 `SystemButton`、输入 `SystemInput`（单行）/ `SystemTextArea`（多行）、徽章 `Badge`、Tab `SegmentedTabs/TabPills`、
  软岛 `Surfaces`、空/加载/错误态 `States`、卡墙首屏骨架 `CardGridSkeleton`（`ui/components/Skeletons.kt`，与 ``NearbyUserCard``/``NearbyGroupCard`` 同构、reduced-motion 不脉冲）、页头 `PageHeader`、底部导航 `BottomNav`。禁止页面手写重复样式。
- 触控热区 ≥44dp；焦点可见；尊重系统动画缩放（reduced-motion 直接终态）；图片统一 `Avatar`/coil 渐显。
- **安全区一律取系统 `WindowInsets`**：根 `MainActivity` 统一注入 `safeDrawing` 的 Top/Horizontal；底部贴底组件
  （BottomNav / CommentInputBar / GroupActionDock）自行使用 `navigationBars`/`ime`。**禁止在页面写死顶/底部安全区 dp 值**——
  状态栏高度、挖孔、手势条因设备而异，写死会顶出多余留白或压到系统栏。
- **API 29 及以下的状态栏图标切换必须保留 edge-to-edge layout flags**：用 `systemUiVisibility` 设置
  `LIGHT_STATUS_BAR`/`LIGHT_NAVIGATION_BAR` 时不能整体覆盖，需保留 `LAYOUT_STABLE | LAYOUT_FULLSCREEN |
  LAYOUT_HIDE_NAVIGATION`；否则窗口会重新为状态栏让位，根布局的 `safeDrawing` 再叠一次，顶部多出整整一个状态栏高度。
- **通栏头图页（瞬间详情）的沉浸式规则**：根布局默认给所有页面加 Top 安全区，纯色页面状态栏与背景同色、视觉连续；
  头图页若也内缩，状态栏会露出一条背景色，与图片割裂。此类页面进入时置 `ChromeInsets.edgeToEdgeTop`
  （根不再加 Top 内边距，内容从屏幕顶开始），页面内用 `WindowInsets.statusBars` 给顶部浮层/无图占位留白；
  头图顶部加随内容滚动的压暗渐变，`ChromeInsets.lightStatusBarIcons` 按头图是否仍覆盖状态栏切换（滚出后切回主题图标），
  离开页面必须复位。**仍禁止在页面写死状态栏高度。**
- 列表必须三态（loading/empty/error）；分页列表有到底提示；不得用假数据填充空态。首屏等待按形态选：行式列表用 `LoadingState`，名片卡墙用 `CardGridSkeleton`（周边/附近类 2 列卡），骨架与加载后卡片同构、不得引起网格跳变。
- 主题：3 主题（vibrant/forest/inkwash）× 明暗三档（跟随系统/浅/深），系统栏颜色随主题。
- 新增页面前先对照设计稿（Figma）移动端对应画板与 `docs/ui-components.md` 的同类契约。

## 已知待办（首版登记）

1. 管理后台（举报审核、用户管理、群组管理、系统公告、统计分析、权限与角色、标签管理、审计日志已接入；
   RBAC 角色权限仅支持权限字典内 key，superadmin 的 `'*'` 未在 App 暴露）；
   群聊会话页已接入 WebSocket 实时新消息/编辑/撤回/已读数刷新、输入中提示与 @提及；
   私聊/群聊已接入图片消息（相册选图 → chat_image 上传 → type=image 发送 + 图片气泡展示）；
   通知中心点击已按 payload 跳转（含 @提及瞬间/群聊）；语音消息已接入（录制 + 播放，chat_audio）；
   群管理已接入（owner/admin）：群信息编辑、群设置、公告管理、成员管理（角色/禁言/移除/封禁）、
   转让群主、入群申请审批与邀请成员；
   举报已覆盖用户/瞬间/评论/群组/消息（群组举报在群详情更多菜单，消息举报走长按操作面板）；
   消息长按操作面板已接入：复制文本 / 撤回自己未超时消息 / 举报他人消息；
   评论已补：删除自己的评论（comments/delete 软删，仅作者；确认后展示「该评论已删除」占位）、
   查看全部回复（comments/fetch_replies 展开 top_replies 预览之外的回复）、
   举报评论（reports/create target_type=comment，复用 ReportDialog）与楼中楼回复点赞。
2. 端上走查：真机优先，无合格真机才用模拟器兜底（设备判定与走查流程见 skill
   `hotspot-android-device-verify`；`assembleRelease` 已在本地验证通过，CI 仍只跑 debug）。
3. 地图真实底图（高德/OSM 瓦片）、地理编码搜索与设备定位未接：底图暂为示意底色，搜索候选来自
   `map/canvas` hotspots + 「纬度,经度」坐标，版权标为「示意底图」而非地图服务商。