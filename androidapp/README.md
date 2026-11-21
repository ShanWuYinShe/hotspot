# androidapp — HotSpot Android 客户端（Kotlin + Jetpack Compose）

按 Figma 设计稿移动端 390 屏幕实现的首版客户端。设计 token **不手写**：由
`scripts/design-tokens/gen-android.mjs` 从 Web CSS（`frontend/src/styles/theme/`）生成，
`frontend/scripts/check-cross-platform-android.mjs` 逐值校验（挂 `bun run check:theme` 与 CI）。

## 技术栈

- Kotlin 2.4.0 / AGP 9.2.1 / Gradle 9.7.0（wrapper 已提交）
- Jetpack Compose（BOM 2025.12.01）+ material3 + navigation-compose 2.9.0
- OkHttp 5.4.0 + kotlinx-serialization-json 1.11.0（API 层）
- coil-compose 2.7.0（图片，渐显）
- compileSdk / targetSdk 37，minSdk 26，JVM target 17，包名 `com.hotspot.app`

## 构建与验证

```bash
cd androidapp
JAVA_HOME="/Applications/Android Studio.app/Contents/jbr/Contents/Home" \
ANDROID_HOME="$HOME/Library/Android/sdk" \
./gradlew --no-daemon assembleDebug        # 产物：app/build/outputs/apk/debug/app-debug.apk
./gradlew --no-daemon testDebugUnitTest
./gradlew --no-daemon assembleRelease      # R8 + 资源压缩（未签名，产物 app-release-unsigned.apk）

# 设计 token 链路（仓库根执行）
node scripts/design-tokens/gen-android.mjs          # 重新生成 Tokens.kt + design-tokens.json
node scripts/design-tokens/gen-android.mjs --check  # 只校验产物是否最新
cd frontend && bun run check:theme                  # CSS ↔ JSON ↔ Tokens.kt 逐值校验
```

构建后不留常驻守护进程：`pgrep -fl 'GradleDaemon|KotlinCompileDaemon'` 应为空。

## 调试地址

- 默认 `http://10.0.2.2:8888`（指向宿主 nginx 网关）
- 模拟器/真机联调：`adb reverse tcp:8888 tcp:8888`，并通过 Gradle 属性覆盖
  `-PHOTSPOT_DEBUG_BASE_URL=http://127.0.0.1:8888`
  （本机 API 36 AVD 上直接访问 `10.0.2.2` 会连接超时，且 `adb shell nc` 可连通不代表端内可达，
  判据与走查流程见 `.agents/skills/hotspot-android-device-verify/SKILL.md`）
- release：`-PHOTSPOT_BASE_URL=https://chaoset.com`（默认线上站点）
- debug 允许明文 HTTP（network security config，仅本机地址）；release 禁止明文

## 目录

```
app/src/main/java/com/hotspot/app/
  MainActivity.kt              # edge-to-edge + 主题/系统栏
  AppContainer.kt              # 依赖容器（无 DI 框架）
  data/api/                    # OkHttp + kotlinx-serialization，端点表严格对齐 spec/
  data/auth/TokenStore.kt      # Bearer 会话持久化（见「安全待办」）
  data/model/                  # 与 spec/ 字段一一对应的 DTO
  data/repo/                   # 认证 / 信息流 / 附近 / 搜索 / 消息 / 资料 / 通知
  ui/theme/                    # Tokens.kt（自动生成）、Theme/Type/Shape/Motion/Dimens
  ui/components/               # SystemButton / SystemInput / SearchInput / Badge / Tabs / ListGroup /
                               # ListRow / SystemCard / Avatar / 三态 / PageHeader / BottomNav /
                               # AmbientCanvas / MomentCard / CommentThread / CommentInputBar /
                               # CommentAuthorBadge
  ui/screens/                  # 登录注册访客 / 首页 / 附近 / 消息 / 私聊 / 我的 / 我的瞬间 /
                               # 我的群组 / 群详情 / 搜索 / 通知 / 设置 / 发布占位
  navigation/                  # 路由 + 登录态门禁 + 底部 5 tab
  util/                        # 媒体 URL 归一 / 时间格式化
app/src/test/                  # token 结构不变量 + 业务错误码文案单测
```

## 首版范围与 TODO（明确占位，不造假数据）

已实现（真实 API、loading / empty / error 三态齐全）：
登录 / 注册 / 访客模式（按 390 设计稿重排：品牌行 + 表单卡 + 协议/链接行；用户名型注册免验证码，
访客失败透传服务端 message）、首页信息流（为你推荐 = `moments/search` 热门模式；
正在关注 = `moments/fetch_following`；下拉刷新 + 无限滚动 + 到底提示）、附近 4 屏（枢纽 =
实时附近统计卡 + 附近的人/瞬间/群组预览；叶子页 = 附近的人（关注）/附近群组（加入）/附近瞬间，
均带兴趣筛选与排序，显式传 client_config 默认位置作浏览锚点）、消息列表、私聊（只读历史 + 简单文本发送）、个人中心（设计稿 87:3393：资料卡 + 瞬间/粉丝/群组
三指标 + 编辑资料/设置 + 我的内容 5 行计数入口）、关系页（设计稿 87:3300：好友/关注/粉丝三 tab +
计数下划线 + 搜索 + 好友「发消息」/关注与粉丝「关注两态」）、编辑资料（设计稿 87:3128：昵称 / 简介 150 字 /
性别 / 兴趣标签 ≤5，保存走 users/update_account patch，头像上传待接入）、我的瞬间（`moments/fetch_created` 发布 / `moments/fetch_favorites` 收藏
两个 tab，2 列网格、private 徽章与软删占位、网格分页）、我的群组（`groups/fetch_joined`：
置顶/最近消息预览/在线数，按群名与标签本地过滤，行点击进群详情）、群详情（设计稿 87:1696：
封面卡 + 简介/成员/公告三 tab；简介页指标 4 列 / 活跃成员头像堆叠 / 公告预告；成员与公告真实分页与
三态；`groups/enter` 加入（password 群口令弹窗，approval 群落「已申请，等待审核」）、
`groups/exit` 退出（更多菜单 + 二次确认）；入口接我的群组行 / 附近群组卡 / 搜索群组结果）、
搜索（用户/瞬间/群组）、通知（列表 + 全部已读）、设置（设计稿 87:4864：账号卡 + 三主题预览卡 ×
明暗三档 + 系统与关于：版本 / 我的举报 / 退出登录）、我的举报（设计稿 87:3350：全部/待处理/已处理/已驳回
筛选 + 举报卡片状态与处理结果，`reports/fetch` 分页）、
瞬间详情（含评论楼中楼列表 + docked 评论输入条：发表/回复/点赞 + 收藏两态 + 系统分享，方案 B 视觉与
Web/鸿蒙同口径；收藏落 `moments/set_favorite`，与「我的瞬间-收藏」tab 同源）。

已登记 TODO（首版占位，UI 显式标注）：
- 地图原生渲染（附近页占位卡）
- 图片上传 / 发布（`media/presign_upload` → `moments/create`）
- WebSocket 实时消息（当前为懒加载 + 手动刷新）
- 管理后台（不在移动端首版范围）
- 群聊会话页（群详情已实现，但首版无群聊页，故不放入口，也不伪造「打开群聊」按钮）
- 群管理（改群信息 / 公告与成员管理 / 转让群主）与举报群组
- 头像上传（media/complete_upload → users/update_account.avatar_file_id；编辑资料页点头像仅提示占位）
- 关系页「黑名单」tab（依赖拉黑 users/set_block，首版未实现）
- 设置页「地图与位置」区块（高德/OpenStreetMap 切换）暂不渲染：Android 原生地图仍为 TODO，提供无效开关属造假入口
- 我的瞬间「喜欢」tab（后端无列表端点，按设计稿登记不渲染）

## 安全待办

`TokenStore` 当前用普通 SharedPreferences 持久化 access/refresh token（依赖清单未含
androidx.security:security-crypto）。上线前必须迁移 EncryptedSharedPreferences 或 Android
Keystore 加密实现（`TokenStore` 文件内已登记 TODO，接口保持不变）。
