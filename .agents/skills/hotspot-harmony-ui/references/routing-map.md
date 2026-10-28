# web → harmony 路由与信息架构对齐（hotspot）

本文件是 skill `hotspot-harmony-ui` 的路由对齐明细（改导航/页面结构前先对齐）。

## 目录命名

对齐前端路由域（复数）：`features/messages`（消息列表）/ `features/moments`（瞬间）/ `features/groups`（群组）/ `features/me`（个人中心，同时承载前端 `/users/[userId]` 他人主页查看——同一 ProfilePage 复用，属平台例外）/ `features/nearby` / `features/settings` / `features/search` / `features/map` / `features/auth` / `features/chat`（聊天窗口独立域）。

## 导航层级

- 底栏 4 tab：首页 / 附近 / 消息 / 我的（`PhoneLayout` TAB_ITEMS + `AppNavigation.BOTTOM_TABS`，与前端 `routes/(app)/+page.svelte`、`/nearby` 聚合页对齐）。
- 桌面侧边栏：首页 / 附近瞬间（moments tab）/ 附近的人 / 附近群组 + 消息 / 通知 overlay / 我的瞬间 / 我的群组 / 关系（对齐前端 desktop nav：home / nearby-people / nearby-groups / nearby-moments / messages / moments / my-groups / relationships；web 通知走顶栏铃铛，不在侧栏）。

## 页面行为对齐

- 消息列表：排序 = 置顶优先 → 最新一条消息时间倒序（`last_message.sent_at` 优先，回退 `last_activity_at`），已读/未读不参与排序；列表头部有「全部 / 未读 / 已读」过滤（`TabsPillGroup`，对齐前端 messages 页 TabBar）；点击未读会话后徽章立即清零。
- 聊天记录：消息按 `sent_at` 正序渲染（越新越靠下，对齐前端 useChat 翻转）；跨天自动插入日期胶囊（`FormatUtil.formatDateLabel` → 今天/昨天/M月D日，对齐前端 groupMessagesByDay/formatDateLabel）。
- 动态详情：点赞/收藏/分享/评论（评论列表 + 乐观发布，对齐前端 MomentDetailPage）。
- 附近聚合页 `NearbyPage` 三区块「查看全部」分别 push `NEARBY_USERS` / `NEARBY_GROUPS` / `NEARBY_MOMENTS`（MomentsFeedPage 已注册为路由页，push 场景带返回按钮，对齐前端 `/nearby/people|groups|moments`）。
- 搜索页 Tab：全部 / 用户 / 动态 / 话题（话题从动态标签聚合，对齐前端搜索 全部/用户/动态/话题）；群组 Tab 为鸿蒙端扩展（前端搜索无群组 Tab）。空查询显示发现态（最近搜索 / 正在关注 / 推荐话题，对齐前端 SearchDiscovery）；最近搜索持久化在 PreferencesUtil（键 `search_recent`，最多 5 条）。
- WS 事件类型与 spec/前端对齐：含 `GroupMessageReadCountUpdated` / `GroupMemberRoleChanged`（已登记类型与订阅，展示由群详情/聊天页按需刷新）。
