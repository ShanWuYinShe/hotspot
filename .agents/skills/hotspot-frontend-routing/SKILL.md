---
name: hotspot-frontend-routing
description: hotspot 前端路由与目录组织：文件路由表、设备分目录、私有组件放置、SPA 模式约束。触发：新增路由或页面；决定组件放哪（desktop/mobile/shared）；目录结构问题。
---

# hotspot-frontend-routing

## 路由系统

基于 SvelteKit 文件路由（SPA fallback 服务一切路径，动态路由无需占位枚举）：

```
src/routes/
  +layout.ts               # export const ssr = false（SPA 模式开关）
  +layout.svelte           # 全局壳（styles + stores + AuthGate；MSW 注册在 hooks.client.ts）
  +error.svelte            # 404
  (auth)/login/+page.svelte    # 登录页（无 MainLayout 外壳）
  share/[code]/+page.svelte    # 分享落地页（真实用户走 SPA；爬虫 UA 由 nginx 分流到后端 OG，
                               # 见 backend-go/AGENTS.md「路由与协议」的 /share/{code} 登记例外）
  (app)/
    +layout.svelte         # MainLayout 外壳（桌面/平板/移动/横屏四端分派）
    +page.svelte           # 首页 = 发现（moments/search 热门模式，spec 定位“发现页首屏”）
    nearby/+page.svelte    # 「附近」枢纽页（地理轴唯一归属：Hero 统计 + 人/瞬间/群组三段分区，
                           # 各段“查看全部”进下方三个叶子页；2026-10-19 起不再是 307 跳转）
    nearby/shared/NearbyHubSections.svelte  # 枢纽三段分区（纯展示，用例锁预览条数与入口）
    nearby/people/         # 附近的人
    nearby/groups/         # 附近的组
    nearby/moments/        # 附近时刻（页面样板结构权威基准）
    moments/[momentId]/    # 时刻详情
    messages/              # 消息列表
    messages/[peerId]/     # 私聊详情（peerId 即对端 user_id）
    messages/group/[groupId]/  # 群聊详情
    admin/                 # 平台管理后台（/admin 布局做权限守卫 + 模块 Tab；概览 + stats/users/groups/
                           # reports/tags/announcements/rbac/audit/config 九模块；spec api_spec_admin.md。
                           # 桌面侧栏对平台管理员条件渲染入口；/admin/* 页面路由与 /admin/api/v1 API 节点
                           # 由 nginx 分流：仅 /admin/api/v1/ 代理后端（见 docker/nginx*.conf））
    me/                    # 个人主页总览（mobile→ProfileMobile，tablet/desktop→ProfilePage）
    me/moments/            # 个人主页固定分区（moments / groups 两个独立固定路由，
    me/groups/             # 任意设备走 ProfilePage 分区模式，不走动态段）
    me/friends/            # ⚠️ 旧路径：仅 +page.ts 做 308 重定向到
    me/following/          #    /me/relationships?tab=…（单一事实源 lib/relationships.ts），
    me/fans/               #    已无 +page.svelte，不得再在此挂任何 UI
    me/relationships/      # 关系聚合页（好友/关注/粉丝三 tab，colocate RelationshipsPage）
    me/reports/            # 我的举报（状态 Tab 筛选；colocate ReportsPage；入口在设置页「隐私与安全」）
    me/edit/               # 编辑资料
    users/[userId]/        # 用户详情
    groups/[groupId]/      # 群组详情
    settings/              # 设置
    map/                   # 地图（+page.svelte 设备分派；desktop/ mobile/ shared/ 子目录）
    notify/               # 通知
    search/                # 搜索
```

> 分栏页（`notify/`、`moments/[momentId]/`、`users/[userId]/`）
> 使用 `SplitPageLayout`；其余 `(app)` 页面（含首页、群组详情、个人主页）使用 `StandardPageLayout`。

路由常量见 `@/lib/routes`（个人主页分区路径用 `ME_SECTION_PATHS` 静态映射，不走动态段）。

页面私有组件与路由同目录，**按设备分目录**：每个路由目录下，`+page.svelte` 做设备分派（`platform.platform`），设备专属组件放 `desktop/` / `mobile/` 子目录，跨设备共享的组件与逻辑放 `shared/` 子目录（如 `map/desktop/LocationDesktop.svelte` + `map/mobile/LocationMobile.svelte` + `map/shared/location-data.ts`）。**多组件路由的私有组件一律拆 `shared/`**（如 `nearby/groups/shared/`、`notify/shared/`、`search/shared/`）；单组件路由（如 `nearby/people/UserCard.svelte`）保持平铺。跨路由复用的页面组件（如 `components/profile/SectionFriends.svelte` 同时被 `me/moments` 分区页与 `me/relationships` 聚合页复用）放在 `components/` 对应域目录；共享组件统一在 `components/`，设备专属共享组件按 `components/<域>/{desktop,tablet,mobile}/` 分目录（如 `layout/mobile/MobileLandscape.svelte`、`layout/desktop/DesktopLayout.svelte`）。**设备分派规则（页面内布局分支）**：
- **默认两分支**：`isDesktop = platform.platform !== 'mobile'`（tablet 与 desktop 同分支），适用于大多数页面（tablet 在字号/边距等细节差异由组件内 `platform` 派生，不需要独立 if 分支）。
- **按需三分支**：当 tablet 在**列数、栏宽、整段布局**上确实与 desktop 表现不同时（如 `nearby/moments` 桌面 3 列 / 平板 2 列、`nearby/people`、`nearby/groups`），显式写 `isDesktop` + `isTablet` + mobile 三分支是**合规**的，不算"逐页发明"。判断标准是"tablet 是否有独立的**布局结构**"，而非"字号/间距不同"。
- **反模式**：仅为字号/颜色/间距等可在组件内用 `platform` 单值派生的差异而开三分支——这种情况一律用两分支 + 组件内派生。
## 搜索页 URL 状态契约

- `/search` 支持 `?query=<q>&tab=<all|people|posts|groups|topics>&tag=<tag>`（`people/posts` 为内部 Tab 名，
  外部兼容 `users→people`、`moments→posts`；非法 tab→all；空 query 清空全部参数回发现态）。
- 初始化从 URL 恢复（有 query 即搜索，只 1 组请求）；同步用 `history.replaceState`（不新增历史项），
  `popstate` 前进/后退恢复；输入防抖/联想不写 URL，只在 Enter/选最近·热门/tag/Tab 切换/Esc 时同步。
- 解析/序列化统一走 `frontend/src/lib/search-url.ts`（含单测）；新增 URL 态页面按同模式（parse/build + replaceState + popstate）。
