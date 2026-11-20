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
    const val SEARCH = "search"
    const val NOTIFY = "notify"
    const val SETTINGS = "settings"
    const val CHAT = "chat/{userId}"
    const val MOMENT_DETAIL = "moment/{momentId}"

    /** 底部导航 5 tab 的路由集合（仅这些路由显示底栏）。 */
    val tabRoutes: Set<String> = setOf(HOME, NEARBY, PUBLISH, MESSAGES, ME)

    fun chat(userId: String): String = "chat/" + userId

    fun moment(momentId: String): String = "moment/" + momentId
}
