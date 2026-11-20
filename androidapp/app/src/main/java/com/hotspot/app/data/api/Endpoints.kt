package com.hotspot.app.data.api

/**
 * API 端点表：路径严格对齐 spec/api_spec_index.md（POST /{node}/{version}/{resource}/{action}）。
 * 基址来自 BuildConfig.API_BASE_URL（debug=模拟器回环，release=线上站点，只含 origin）。
 */
object ApiEndpoint {
    // /public/v1（匿名节点；App 端也用它做登录/注册）
    const val SIGNIN = "/public/v1/auth/signin"
    const val REGISTER = "/public/v1/auth/register"
    const val REFRESH_TOKEN = "/public/v1/auth/refresh_token"

    // /api/v1（必须 Bearer Token）
    const val SIGNOUT = "/api/v1/auth/signout"

    const val MOMENTS_SEARCH = "/api/v1/moments/search"
    const val MOMENTS_OPEN = "/api/v1/moments/open_the_door"
    const val MOMENTS_FETCH_FOLLOWING = "/api/v1/moments/fetch_following"
    const val MOMENTS_FETCH_NEARBY = "/api/v1/moments/fetch_nearby"

    const val USERS_SEARCH = "/api/v1/users/search"
    const val USERS_ACCOUNT = "/api/v1/users/account"

    const val GROUPS_SEARCH = "/api/v1/groups/search"

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
