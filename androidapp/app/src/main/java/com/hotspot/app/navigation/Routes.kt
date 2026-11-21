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
    const val RELATIONSHIPS = "me/relationships?tab={tab}"
    // 附近三个叶子页（设计稿 87:3798 / 87:3995 / 87:4192；入口 = 附近枢纽各段「查看全部」）
    const val NEARBY_PEOPLE = "nearby/people"
    const val NEARBY_GROUPS = "nearby/groups"
    const val NEARBY_MOMENTS = "nearby/moments"
    const val SEARCH = "search"
    const val NOTIFY = "notify"
    const val SETTINGS = "settings"
    const val CHAT = "chat/{userId}"
    const val MOMENT_DETAIL = "moment/{momentId}"
    // 群详情（推入页；设计稿 87:1696，入口：我的群组 / 附近群组 / 搜索群组结果）
    const val GROUP = "group/{groupId}"
    // 他人主页（设计稿 87:5163，users/open_the_door 聚合资料 + 最近瞬间）
    const val USER = "user/{userId}"

    fun user(userId: String): String = "user/" + userId

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

    fun moment(momentId: String): String = "moment/" + momentId

    fun group(groupId: String): String = "group/" + groupId

    /** 关系页（0 = 好友 / 1 = 关注 / 2 = 粉丝）。 */
    fun relationships(tab: Int): String = "me/relationships?tab=" + tab
}
