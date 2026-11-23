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
    // 地图视口渲染（spec/api_spec_public.md → map/canvas；仅 public 版，无登录镜像）
    const val PUBLIC_MAP_CANVAS = "/public/v1/map/canvas"

    // /api/v1（必须 Bearer Token）
    const val SIGNOUT = "/api/v1/auth/signout"

    const val MOMENTS_SEARCH = "/api/v1/moments/search"
    const val MOMENTS_OPEN = "/api/v1/moments/open_the_door"
    const val MOMENTS_UPDATE = "/api/v1/moments/update"
    const val MOMENTS_DELETE = "/api/v1/moments/delete"
    const val MOMENTS_FETCH_FOLLOWING = "/api/v1/moments/fetch_following"
    const val MOMENTS_FETCH_NEARBY = "/api/v1/moments/fetch_nearby"
    const val MOMENTS_FETCH_CREATED = "/api/v1/moments/fetch_created"
    const val MOMENTS_FETCH_FAVORITES = "/api/v1/moments/fetch_favorites"
    const val MOMENTS_SET_FAVORITE = "/api/v1/moments/set_favorite"

    const val USERS_SEARCH = "/api/v1/users/search"
    const val USERS_ACCOUNT = "/api/v1/users/account"
    // 关注/取关（spec/api_spec_user.md → users/set_follow，两态统一）
    const val USERS_SET_FOLLOW = "/api/v1/users/set_follow"
    // 我的账户资料更新（spec/api_spec_user.md → users/update_account，patch 语义）
    const val USERS_UPDATE_ACCOUNT = "/api/v1/users/update_account"
    // 关系列表（user_id 必传；好友/关注/粉丝各自分页）
    const val USERS_FETCH_FRIENDS = "/api/v1/users/fetch_friends"
    const val USERS_FETCH_FOLLOWING = "/api/v1/users/fetch_following"
    const val USERS_FETCH_FOLLOWERS = "/api/v1/users/fetch_followers"
    // 用户资料详情（spec → users/open_the_door；include_recent_moments 聚合主页首屏）
    const val USERS_OPEN_THE_DOOR = "/api/v1/users/open_the_door"
    // 用户瞬间历史（公开视角，spec → users/fetch_moments）
    const val USERS_FETCH_MOMENTS = "/api/v1/users/fetch_moments"
    const val USERS_SET_REMARK = "/api/v1/users/set_remark"
    const val USERS_SET_BLOCK = "/api/v1/users/set_block"
    const val USERS_FETCH_BLOCKED = "/api/v1/users/fetch_blocked"
    // 我的举报记录（spec/api_spec_report.md → reports/fetch，命名空间天生私有）
    const val REPORTS_FETCH = "/api/v1/reports/fetch"
    const val REPORTS_CREATE = "/api/v1/reports/create"
    // 媒体上传三步链路（spec/api_spec_media.md）：presign_upload → PUT 直传 → complete_upload
    const val MEDIA_PRESIGN_UPLOAD = "/api/v1/media/presign_upload"
    const val MEDIA_COMPLETE_UPLOAD = "/api/v1/media/complete_upload"
    // 发布瞬间（spec/api_spec_moment.md → moments/create，MUST 幂等）
    const val MOMENTS_CREATE = "/api/v1/moments/create"

    const val GROUPS_SEARCH = "/api/v1/groups/search"
    const val GROUPS_FETCH_JOINED = "/api/v1/groups/fetch_joined"
    // 群详情（spec/api_spec_group.md）：群组详情 / 加入 / 退出 / 成员列表（登录版）。
    const val GROUPS_OPEN_THE_DOOR = "/api/v1/groups/open_the_door"
    const val GROUPS_ENTER = "/api/v1/groups/enter"
    const val GROUPS_EXIT = "/api/v1/groups/exit"
    const val GROUPS_MEMBERS_FETCH = "/api/v1/groups/members/fetch"
    // 群公告列表仅 public 版（spec/api_spec_public.md：无查看者字段、不设登录镜像）。
    const val GROUPS_ANNOUNCEMENTS_FETCH = "/public/v1/groups/announcements/fetch"
    const val GROUPS_ADMIN_UPDATE = "/api/v1/groups/admin/update"
    const val GROUPS_ADMIN_SETTINGS = "/api/v1/groups/admin/settings"
    const val GROUPS_ADMIN_APPLICATIONS_FETCH = "/api/v1/groups/admin/applications/fetch"
    const val GROUPS_ADMIN_APPLICATIONS_APPROVE = "/api/v1/groups/admin/applications/approve"
    const val GROUPS_ADMIN_APPLICATIONS_REJECT = "/api/v1/groups/admin/applications/reject"
    const val GROUPS_ADMIN_INVITES_CREATE = "/api/v1/groups/admin/invites/create"
    const val GROUPS_ADMIN_INVITES_FETCH_SENT = "/api/v1/groups/admin/invites/fetch_sent"
    const val GROUPS_ADMIN_INVITES_REVOKE = "/api/v1/groups/admin/invites/revoke"
    const val GROUPS_ADMIN_MEMBERS_SET_BAN = "/api/v1/groups/admin/members/set_ban"
    const val GROUPS_ADMIN_MEMBERS_FETCH_BANNED = "/api/v1/groups/admin/members/fetch_banned"
    const val GROUPS_ADMIN_MEMBERS_SET_ROLE = "/api/v1/groups/admin/members/set_role"
    const val GROUPS_ADMIN_MEMBERS_SET_MUTE = "/api/v1/groups/admin/members/set_mute"
    const val GROUPS_ADMIN_MEMBERS_REMOVE = "/api/v1/groups/admin/members/remove"
    const val GROUPS_ADMIN_TRANSFER_OWNERSHIP = "/api/v1/groups/admin/transfer_ownership"
    const val GROUPS_ADMIN_ANNOUNCEMENTS_CREATE = "/api/v1/groups/admin/announcements/create"
    const val GROUPS_ADMIN_ANNOUNCEMENTS_UPDATE = "/api/v1/groups/admin/announcements/update"
    const val GROUPS_ADMIN_ANNOUNCEMENTS_DELETE = "/api/v1/groups/admin/announcements/delete"

    const val PRIVATE_CHATS_FETCH = "/api/v1/private_chats/fetch"
    const val PRIVATE_MESSAGES_FETCH = "/api/v1/private_messages/fetch"
    const val PRIVATE_MESSAGES_SEND = "/api/v1/private_messages/send"
    const val PRIVATE_MESSAGES_RECALL = "/api/v1/private_messages/recall"
    const val PRIVATE_MESSAGES_MARK_READ = "/api/v1/private_messages/mark_read"
    // 群聊消息（spec/api_spec_chat.md → group_messages/fetch | send；群成员可见/可发）
    const val GROUP_MESSAGES_FETCH = "/api/v1/group_messages/fetch"
    const val GROUP_MESSAGES_SEND = "/api/v1/group_messages/send"
    const val GROUP_MESSAGES_RECALL = "/api/v1/group_messages/recall"
    const val GROUP_MESSAGES_MARK_READ = "/api/v1/group_messages/mark_read"

    const val COMMENTS_FETCH = "/api/v1/comments/fetch"
    const val COMMENTS_CREATE = "/api/v1/comments/create"
    const val COMMENTS_DELETE = "/api/v1/comments/delete"
    const val COMMENTS_FETCH_REPLIES = "/api/v1/comments/fetch_replies"

    const val REACTIONS_SET = "/api/v1/reactions/set"

    const val NOTIFY_FETCH = "/api/v1/notify/fetch"
    const val NOTIFY_UNREAD_SUMMARY = "/api/v1/notify/unread_summary"
    // 平台管理端（admin 命名空间，权限由后端 RBAC 校验）
    const val ADMIN_RBAC_ME = "/admin/api/v1/rbac/me"
    const val ADMIN_REPORTS_FETCH = "/admin/api/v1/reports/fetch"
    const val ADMIN_REPORTS_JUDGE = "/admin/api/v1/reports/judge"

    const val NOTIFY_MARK_READ = "/api/v1/notify/mark_read"
}
