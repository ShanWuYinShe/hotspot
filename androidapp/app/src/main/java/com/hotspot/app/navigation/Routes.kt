package com.hotspot.app.navigation

/** 路由常量（navigation-compose 字符串路由）。底部 5 tab + 推入页。 */
object Routes {
    const val LOGIN = "login"
    const val HOME = "home"
    const val NEARBY = "nearby"
    const val PUBLISH = "publish"
    const val MESSAGES = "messages"
    const val ME = "me"
    const val ME_MOMENTS = "me/moments"
    const val ME_GROUPS = "me/groups"
    // 编辑资料（设计稿 87:3128）与关系页（设计稿 87:3300，tab = 好友/关注/粉丝）
    const val ME_EDIT = "me/edit"
    // 我的举报（设计稿 87:3350，入口：设置 → 系统与关于）
    const val ME_REPORTS = "me/reports"
    const val ME_SHARES = "me/shares"
    // 屏蔽名单（入口：设置 → 系统与关于）
    const val ME_BLOCKED = "me/blocked"
    const val SETTINGS_SESSIONS = "settings/sessions"
    // 管理后台（仅平台管理员可从设置进入）
    const val ADMIN_HOME = "admin"
    const val ADMIN_REPORTS = "admin/reports"
    const val ADMIN_USERS = "admin/users"
    const val ADMIN_GROUPS = "admin/groups"
    const val ADMIN_ANNOUNCEMENTS = "admin/announcements"
    const val ADMIN_STATS = "admin/stats"
    const val ADMIN_RBAC = "admin/rbac"
    const val ADMIN_TAGS = "admin/tags"
    const val ADMIN_AUDIT = "admin/audit"
    const val ADMIN_USERS_FILTER = "admin/users/filter?query={query}"
    const val ADMIN_GROUPS_FILTER = "admin/groups/filter?query={query}"
    const val RELATIONSHIPS = "me/relationships?tab={tab}"
    // 附近三个叶子页（设计稿 87:3798 / 87:3995 / 87:4192；入口 = 附近枢纽各段「查看全部」）
    const val NEARBY_PEOPLE = "nearby/people"
    const val NEARBY_GROUPS = "nearby/groups"
    const val NEARBY_MOMENTS = "nearby/moments"
    // 附近地图（原生 Canvas 渲染；全屏推入页，不显示底栏）
    const val NEARBY_MAP = "nearby/map"
    const val SEARCH = "search"
    const val NOTIFY = "notify"
    const val SETTINGS = "settings"
    const val CHAT = "chat/{userId}"
    // 瞬间详情可携带 commentId（评论通知/提及），进入后滚动定位到对应评论线程
    const val MOMENT_DETAIL = "moment/{momentId}?commentId={commentId}"
    // 群详情（推入页；设计稿 87:1696，入口：我的群组 / 附近群组 / 搜索群组结果）
    const val GROUP = "group/{groupId}"
    // 创建群组（无独立设计画板；入口：我的群组 / 附近群组页头）
    const val GROUP_CREATE = "create_group"
    // 群聊会话（设计稿 87:3529；入口：群详情「打开群聊」，群成员可见）
    const val GROUP_CHAT = "group_chat/{groupId}"
    // 他人主页（设计稿 87:5163，users/open_the_door 聚合资料 + 最近瞬间）
    const val USER = "user/{userId}"

    fun user(userId: String): String = "user/" + userId

    fun groupChat(groupId: String): String = "group_chat/" + groupId

    /** 底部导航 5 tab 的路由集合（仅这些路由显示底栏）。 */
    val tabRoutes: Set<String> = setOf(HOME, NEARBY, PUBLISH, MESSAGES, ME)

    /** 附近子页：仍显示底栏（设计稿子页保留悬浮岛导航），且「附近」tab 保持选中态。 */
    val nearbyRoutes: Set<String> = setOf(NEARBY, NEARBY_PEOPLE, NEARBY_GROUPS, NEARBY_MOMENTS)

    /** 显示底栏的全部路由。 */
    val bottomBarRoutes: Set<String> = tabRoutes + nearbyRoutes

    /** 某 tab 在当前路由下是否处于选中态（附近子页归属「附近」tab）。 */
    fun isTabActive(route: String, currentRoute: String?): Boolean =
        if (route == NEARBY) currentRoute in nearbyRoutes else currentRoute == route

    fun chat(userId: String): String = "chat/" + userId

    fun moment(momentId: String, commentId: String? = null): String =
        if (commentId.isNullOrBlank()) "moment/" + momentId else "moment/" + momentId + "?commentId=" + commentId

    fun group(groupId: String): String = "group/" + groupId

    const val CREATE_GROUP = GROUP_CREATE

    /** 关系页（0 = 好友 / 1 = 关注 / 2 = 粉丝）。 */
    fun relationships(tab: Int): String = "me/relationships?tab=" + tab
}