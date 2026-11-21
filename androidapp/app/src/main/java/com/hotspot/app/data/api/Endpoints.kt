package com.hotspot.app.data.api

/**
 * API 端点表：路径严格对齐 spec/api_spec_index.md（POST /{node}/{version}/{resource}/{action}）。
 * 基址来自 BuildConfig.API_BASE_URL（debug=模拟器回环，release=线上站点，只含 origin）。
 */
object ApiEndpoint {
    // /public/v1（匿名节点；App 端也用它做登录/注册）
    const val SIGNIN = "/public/v1/auth/signin"
    const val REGISTER = "/public/v1/auth/register"
    const val GUEST = "/public/v1/auth/guest"
    const val REFRESH_TOKEN = "/public/v1/auth/refresh_token"

    // /public/v1/system（匿名：客户端公开配置，含 defaults.default_location 浏览锚点）
    const val SYSTEM_CLIENT_CONFIG = "/public/v1/system/client_config"

    // /api/v1（必须 Bearer Token）
    const val SIGNOUT = "/api/v1/auth/signout"

    const val MOMENTS_SEARCH = "/api/v1/moments/search"
    const val MOMENTS_OPEN = "/api/v1/moments/open_the_door"
    const val MOMENTS_FETCH_FOLLOWING = "/api/v1/moments/fetch_following"
    const val MOMENTS_FETCH_NEARBY = "/api/v1/moments/fetch_nearby"
    const val MOMENTS_FETCH_CREATED = "/api/v1/moments/fetch_created"
    const val MOMENTS_FETCH_FAVORITES = "/api/v1/moments/fetch_favorites"
    const val MOMENTS_SET_FAVORITE = "/api/v1/moments/set_favorite"

    const val USERS_SEARCH = "/api/v1/users/search"
    const val USERS_ACCOUNT = "/api/v1/users/account"
    // 关注/取关（spec/api_spec_user.md → users/set_follow，两态统一）
    const val USERS_SET_FOLLOW = "/api/v1/users/set_follow"

    const val GROUPS_SEARCH = "/api/v1/groups/search"
    const val GROUPS_FETCH_JOINED = "/api/v1/groups/fetch_joined"
    // 群详情（spec/api_spec_group.md）：群组详情 / 加入 / 退出 / 成员列表（登录版）。
    const val GROUPS_OPEN_THE_DOOR = "/api/v1/groups/open_the_door"
    const val GROUPS_ENTER = "/api/v1/groups/enter"
    const val GROUPS_EXIT = "/api/v1/groups/exit"
    const val GROUPS_MEMBERS_FETCH = "/api/v1/groups/members/fetch"
    // 群公告列表仅 public 版（spec/api_spec_public.md：无查看者字段、不设登录镜像）。
    const val GROUPS_ANNOUNCEMENTS_FETCH = "/public/v1/groups/announcements/fetch"

    const val PRIVATE_CHATS_FETCH = "/api/v1/private_chats/fetch"
    const val PRIVATE_MESSAGES_FETCH = "/api/v1/private_messages/fetch"
    const val PRIVATE_MESSAGES_SEND = "/api/v1/private_messages/send"
    const val PRIVATE_MESSAGES_MARK_READ = "/api/v1/private_messages/mark_read"

    const val COMMENTS_FETCH = "/api/v1/comments/fetch"
    const val COMMENTS_CREATE = "/api/v1/comments/create"

    const val REACTIONS_SET = "/api/v1/reactions/set"

    const val NOTIFY_FETCH = "/api/v1/notify/fetch"
    const val NOTIFY_UNREAD_SUMMARY = "/api/v1/notify/unread_summary"
    const val NOTIFY_MARK_READ = "/api/v1/notify/mark_read"
}
