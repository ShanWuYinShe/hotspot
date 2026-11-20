package com.hotspot.app.data.repo

import com.hotspot.app.data.api.ApiClient
import com.hotspot.app.data.api.ApiEndpoint
import com.hotspot.app.data.api.ApiError
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.auth.AuthSession
import com.hotspot.app.data.auth.TokenStore
import com.hotspot.app.data.model.AccountData
import com.hotspot.app.data.model.AuthTokenPair
import com.hotspot.app.data.model.CommentItem
import com.hotspot.app.data.model.EnterGroupResult
import com.hotspot.app.data.model.GroupAnnouncementItem
import com.hotspot.app.data.model.GroupAnnouncementList
import com.hotspot.app.data.model.GroupItem
import com.hotspot.app.data.model.GroupMemberItem
import com.hotspot.app.data.model.MessageSentResult
import com.hotspot.app.data.model.MomentItem
import com.hotspot.app.data.model.NotifyItem
import com.hotspot.app.data.model.Page
import com.hotspot.app.data.model.PrivateChatItem
import com.hotspot.app.data.model.PrivateMessageItem
import com.hotspot.app.data.model.UnreadSummary
import com.hotspot.app.data.model.UserItem
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import java.util.UUID

/** 分页参数（spec/api_spec_types.md PaginationParams：limit 默认 20 / cursor 不透明透传）。 */
internal fun pagination(cursor: String?, limit: Int = 20): JsonObject = buildJsonObject {
    put("limit", limit)
    if (cursor != null) put("cursor", cursor)
}

internal fun emptyObject(): JsonObject = buildJsonObject { }

/** 认证：登录 / 注册（用户名型，免验证码）/ 登出。字段严格按 spec/api_spec_public.md。 */
class AuthRepository(
    private val api: ApiClient,
    private val tokens: TokenStore,
) {
    suspend fun signin(loginId: String, password: String): ApiResult<AuthSession> {
        val body = buildJsonObject {
            put("login_id", loginId)
            put("password", password)
            put("client_type", "app")
            put("device_id", tokens.deviceId())
        }
        return toSession(api.post(ApiEndpoint.SIGNIN, body, AuthTokenPair.serializer()))
    }

    suspend fun register(loginId: String, nickname: String, password: String): ApiResult<AuthSession> {
        val body = buildJsonObject {
            put("login_id", loginId)
            // 用户名型 login_id 必须为 null（同传验证码 = 参数冲突）；手机号/邮箱型在验证码服务接入前不可用。
            put("verification_code", JsonNull)
            put("nickname", nickname)
            put("password", password)
            put("client_type", "app")
            put("device_id", tokens.deviceId())
        }
        return toSession(
            api.post(
                ApiEndpoint.REGISTER,
                body,
                AuthTokenPair.serializer(),
                idempotencyKey = UUID.randomUUID().toString(),
            ),
        )
    }

    /**
     * 访客模式（spec api_spec_public.md → POST /public/v1/auth/guest，MUST 幂等）：
     * 每次调用创建独立访客账号（user_type=temp）；平台开关关闭时返回 10400
     * （message「当前未开放访客模式」），由 ApiClient.businessErrorMessage 透传服务端 message。
     */
    suspend fun guest(nickname: String? = null): ApiResult<AuthSession> {
        val body = buildJsonObject {
            put("nickname", nickname)
            put("client_type", "app")
            put("device_id", tokens.deviceId())
        }
        return toSession(
            api.post(
                ApiEndpoint.GUEST,
                body,
                AuthTokenPair.serializer(),
                idempotencyKey = UUID.randomUUID().toString(),
            ),
        )
    }

    suspend fun signout(): ApiResult<Unit> {
        val result = api.postVoid(ApiEndpoint.SIGNOUT)
        tokens.clear()
        return result
    }

    private fun toSession(result: ApiResult<AuthTokenPair>): ApiResult<AuthSession> = when (result) {
        is ApiResult.Ok -> {
            val pair = result.data
            val userId = pair.userId
            val accessToken = pair.accessToken
            val refreshToken = pair.refreshToken
            if (userId.isNullOrBlank() || accessToken.isNullOrBlank() || refreshToken.isNullOrBlank()) {
                ApiResult.Err(ApiError.Network("登录响应缺少凭证字段"))
            } else {
                tokens.save(userId, accessToken, refreshToken)
                ApiResult.Ok(AuthSession(userId, accessToken, refreshToken))
            }
        }
        is ApiResult.Err -> result
    }
}

/**
 * 首页信息流：
 * - 为你推荐 = moments/search 热门模式（spec：无输入条件的热门瞬间，承接发现页首屏）
 * - 正在关注 = moments/fetch_following
 */
class FeedRepository(private val api: ApiClient) {
    suspend fun recommended(cursor: String?): ApiResult<Page<MomentItem>> = api.post(
        ApiEndpoint.MOMENTS_SEARCH,
        buildJsonObject {
            put("keyword", JsonNull)
            put("tag", JsonNull)
            put("sort_by", "popular")
            put("pagination", pagination(cursor))
        },
        Page.serializer(MomentItem.serializer()),
    )

    /** 瞬间详情（登录视角 MomentItem，含 is_favorited 查看者状态）。 */
    suspend fun openMoment(momentId: String): ApiResult<MomentItem> = api.post(
        ApiEndpoint.MOMENTS_OPEN,
        buildJsonObject { put("moment_id", momentId) },
        MomentItem.serializer(),
    )

    /** 收藏 / 取消收藏（spec/api_spec_moment.md → moments/set_favorite，两态统一端点）。 */
    suspend fun setFavorite(momentId: String, active: Boolean): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.MOMENTS_SET_FAVORITE,
        buildJsonObject {
            put("moment_id", momentId)
            put("active", active)
        },
    )

    suspend fun following(cursor: String?): ApiResult<Page<MomentItem>> = api.post(
        ApiEndpoint.MOMENTS_FETCH_FOLLOWING,
        buildJsonObject {
            put("reference_point", JsonNull)
            put("pagination", pagination(cursor))
        },
        Page.serializer(MomentItem.serializer()),
    )
}

/** 附近枢纽：附近的人 / 附近群组 / 附近瞬间（距离排序由后端计算，客户端只展示）。 */
class NearbyRepository(private val api: ApiClient) {
    suspend fun nearbyPeople(cursor: String?): ApiResult<Page<UserItem>> = api.post(
        ApiEndpoint.USERS_SEARCH,
        buildJsonObject {
            put("keyword", JsonNull)
            put("tag", JsonNull)
            put("radius", 5000)
            put("sort_by", "distance")
            put("reference_point", JsonNull)
            put("pagination", pagination(cursor))
        },
        Page.serializer(UserItem.serializer()),
    )

    suspend fun nearbyGroups(cursor: String?): ApiResult<Page<GroupItem>> = api.post(
        ApiEndpoint.GROUPS_SEARCH,
        buildJsonObject {
            put("keyword", JsonNull)
            put("tag", JsonNull)
            put("radius", 5000)
            put("sort_by", "distance")
            put("reference_point", JsonNull)
            put("pagination", pagination(cursor))
        },
        Page.serializer(GroupItem.serializer()),
    )

    suspend fun nearbyMoments(cursor: String?): ApiResult<Page<MomentItem>> = api.post(
        ApiEndpoint.MOMENTS_FETCH_NEARBY,
        buildJsonObject {
            put("radius", 5000)
            put("reference_point", JsonNull)
            put("pagination", pagination(cursor))
        },
        Page.serializer(MomentItem.serializer()),
    )
}

/** 全局搜索：用户 / 瞬间 / 群组各自独立分页（spec/api_spec_discovery.md：不提供聚合端点）。 */
class SearchRepository(private val api: ApiClient) {
    suspend fun searchUsers(keyword: String, cursor: String?): ApiResult<Page<UserItem>> = api.post(
        ApiEndpoint.USERS_SEARCH,
        buildJsonObject {
            put("keyword", keyword)
            put("tag", JsonNull)
            put("radius", JsonNull)
            put("sort_by", "relevance")
            put("reference_point", JsonNull)
            put("pagination", pagination(cursor))
        },
        Page.serializer(UserItem.serializer()),
    )

    suspend fun searchMoments(keyword: String, cursor: String?): ApiResult<Page<MomentItem>> = api.post(
        ApiEndpoint.MOMENTS_SEARCH,
        buildJsonObject {
            put("keyword", keyword)
            put("tag", JsonNull)
            put("sort_by", "relevance")
            put("pagination", pagination(cursor))
        },
        Page.serializer(MomentItem.serializer()),
    )

    suspend fun searchGroups(keyword: String, cursor: String?): ApiResult<Page<GroupItem>> = api.post(
        ApiEndpoint.GROUPS_SEARCH,
        buildJsonObject {
            put("keyword", keyword)
            put("tag", JsonNull)
            put("radius", JsonNull)
            put("sort_by", "relevance")
            put("reference_point", JsonNull)
            put("pagination", pagination(cursor))
        },
        Page.serializer(GroupItem.serializer()),
    )
}

/** 消息：私聊会话列表 / 历史消息 / 发送文本消息。 */
class ChatRepository(private val api: ApiClient) {
    suspend fun conversations(cursor: String?): ApiResult<Page<PrivateChatItem>> = api.post(
        ApiEndpoint.PRIVATE_CHATS_FETCH,
        buildJsonObject { put("pagination", pagination(cursor)) },
        Page.serializer(PrivateChatItem.serializer()),
    )

    suspend fun messages(targetUserId: String, cursor: String?): ApiResult<Page<PrivateMessageItem>> = api.post(
        ApiEndpoint.PRIVATE_MESSAGES_FETCH,
        buildJsonObject {
            put("target_user_id", targetUserId)
            put("pagination", pagination(cursor))
        },
        Page.serializer(PrivateMessageItem.serializer()),
    )

    suspend fun sendTextMessage(targetUserId: String, content: String): ApiResult<MessageSentResult> = api.post(
        ApiEndpoint.PRIVATE_MESSAGES_SEND,
        buildJsonObject {
            put("target_user_id", targetUserId)
            put("type", "text")
            put("content", content)
            put("file_id", JsonNull)
            put("share_code", JsonNull)
            put("reply_to_message_id", JsonNull)
        },
        MessageSentResult.serializer(),
        idempotencyKey = UUID.randomUUID().toString(),
    )
}

/**
 * 评论：顶层列表（首屏 limit 50，cursor 透传）/ 创建（MUST 幂等）/ 评论点赞。
 * 契约见 spec/api_spec_comment.md 与 spec/api_spec_reaction.md；回复归组在 UI 层按 parent_id 完成。
 */
class CommentRepository(private val api: ApiClient) {
    suspend fun fetchComments(
        momentId: String,
        cursor: String?,
        limit: Int = COMMENT_PAGE_SIZE,
    ): ApiResult<Page<CommentItem>> = api.post(
        ApiEndpoint.COMMENTS_FETCH,
        buildJsonObject {
            put("moment_id", momentId)
            put("pagination", pagination(cursor, limit))
        },
        Page.serializer(CommentItem.serializer()),
    )

    /**
     * 创建评论/回复（spec：客户端必须携带 Idempotency-Key）。
     * parent_id 指向顶层评论（若指向二级回复，服务端自动挂到所属顶层评论）；mentions 为被 @ 的 user_id。
     */
    suspend fun createComment(
        momentId: String,
        content: String,
        parentId: String?,
        mentions: List<String>,
        idempotencyKey: String,
    ): ApiResult<CommentItem> = api.post(
        ApiEndpoint.COMMENTS_CREATE,
        buildJsonObject {
            put("moment_id", momentId)
            put("content", content)
            put("parent_id", parentId)
            put("mentions", buildJsonArray { mentions.forEach { add(JsonPrimitive(it)) } })
        },
        CommentItem.serializer(),
        idempotencyKey = idempotencyKey,
    )

    /** 评论点赞/取消（reactions/set target_type=comment；emoji 须在服务端白名单内）。 */
    suspend fun setCommentReaction(commentId: String, active: Boolean): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.REACTIONS_SET,
        buildJsonObject {
            put("target_type", "comment")
            put("target_id", commentId)
            put("emoji", LIKE_EMOJI)
            put("active", active)
        },
    )

    companion object {
        /** 首屏评论页大小（任务契约：limit 50）。 */
        const val COMMENT_PAGE_SIZE = 50

        /** 点赞 emoji：服务端白名单默认含 👍（与 Web / 鸿蒙同口径）。 */
        const val LIKE_EMOJI = "👍"
    }
}

/**
 * 个人中心：账户资料（含仅自己可见的 email/phone/stats）+ 「我的内容」集合。
 * 我的瞬间/收藏/群组均为成员视角端点（spec：fetch_created 含 private 与软删占位、
 * fetch_favorites 含收藏时间、fetch_joined 含 viewer_state/last_message），
 * 不属于公共发现域，故统一收敛在资料仓库。
 */
class ProfileRepository(private val api: ApiClient) {
    suspend fun account(): ApiResult<AccountData> =
        api.post(ApiEndpoint.USERS_ACCOUNT, emptyObject(), AccountData.serializer())

    /** 我创建的瞬间（拥有者视角，spec/api_spec_moment.md → moments/fetch_created）。 */
    suspend fun createdMoments(cursor: String?): ApiResult<Page<MomentItem>> = api.post(
        ApiEndpoint.MOMENTS_FETCH_CREATED,
        buildJsonObject { put("pagination", pagination(cursor)) },
        Page.serializer(MomentItem.serializer()),
    )

    /** 我的收藏（spec → moments/fetch_favorites）。 */
    suspend fun favoriteMoments(cursor: String?): ApiResult<Page<MomentItem>> = api.post(
        ApiEndpoint.MOMENTS_FETCH_FAVORITES,
        buildJsonObject { put("pagination", pagination(cursor)) },
        Page.serializer(MomentItem.serializer()),
    )

    /** 我加入的群组（成员视角，spec/api_spec_group.md → groups/fetch_joined）。 */
    suspend fun joinedGroups(cursor: String?): ApiResult<Page<GroupItem>> = api.post(
        ApiEndpoint.GROUPS_FETCH_JOINED,
        buildJsonObject { put("pagination", pagination(cursor)) },
        Page.serializer(GroupItem.serializer()),
    )
}

/**
 * 群组详情域：详情 / 成员 / 公告 / 加入 / 退出。
 * 契约见 spec/api_spec_group.md（open_the_door、members/fetch、enter、exit）与
 * spec/api_spec_public.md（announcements/fetch：仅 public 版、请求只有 group_id、响应无分页）。
 * 距离/位置由后端计算，客户端只展示；加入/退出结果以服务端 status/错误码为准。
 */
class GroupRepository(private val api: ApiClient) {
    /** 群组详情（登录视角 GroupItem，含 viewer_state：null = 非成员）。 */
    suspend fun open(groupId: String): ApiResult<GroupItem> = api.post(
        ApiEndpoint.GROUPS_OPEN_THE_DOOR,
        buildJsonObject { put("group_id", groupId) },
        GroupItem.serializer(),
    )

    /**
     * 加入群组（智能路由：direct/password + requires_approval 由服务端判定）。
     * password 群需先收集口令；message 为审批群可选附言（本页不做输入，传 null）。
     * 错误码 10200/10300/10400 由 ApiClient 透传服务端 message。
     */
    suspend fun enter(
        groupId: String,
        password: String? = null,
        message: String? = null,
    ): ApiResult<EnterGroupResult> = api.post(
        ApiEndpoint.GROUPS_ENTER,
        buildJsonObject {
            put("group_id", groupId)
            put("password", password)
            put("message", message)
        },
        EnterGroupResult.serializer(),
    )

    /** 退出群组（群主调用会被服务端 10400 拒绝，客户端只透传 message）。 */
    suspend fun exit(groupId: String): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.GROUPS_EXIT,
        buildJsonObject { put("group_id", groupId) },
    )

    /** 成员列表（成员视角公开字段；按 joined_at 正序，客户端只做分页透传）。 */
    suspend fun members(
        groupId: String,
        cursor: String?,
        limit: Int = GROUP_MEMBERS_PAGE_SIZE,
    ): ApiResult<Page<GroupMemberItem>> = api.post(
        ApiEndpoint.GROUPS_MEMBERS_FETCH,
        buildJsonObject {
            put("group_id", groupId)
            put("role", JsonNull)
            put("pagination", pagination(cursor, limit))
        },
        Page.serializer(GroupMemberItem.serializer()),
    )

    /** 群公告列表（单次全量返回，无 cursor；置顶优先由服务端排序）。 */
    suspend fun announcements(groupId: String): ApiResult<List<GroupAnnouncementItem>> {
        val result = api.post(
            ApiEndpoint.GROUPS_ANNOUNCEMENTS_FETCH,
            buildJsonObject { put("group_id", groupId) },
            GroupAnnouncementList.serializer(),
        )
        return when (result) {
            is ApiResult.Ok -> ApiResult.Ok(result.data.items)
            is ApiResult.Err -> result
        }
    }

    companion object {
        /** 成员列表首屏页大小（任务契约：首屏 30）。 */
        const val GROUP_MEMBERS_PAGE_SIZE = 30
    }
}

/** 通知中心：列表 / 未读摘要 / 全部已读。 */
class NotifyRepository(private val api: ApiClient) {
    suspend fun fetch(cursor: String?): ApiResult<Page<NotifyItem>> = api.post(
        ApiEndpoint.NOTIFY_FETCH,
        buildJsonObject {
            put("type", JsonNull)
            put("is_read", JsonNull)
            put("pagination", pagination(cursor))
        },
        Page.serializer(NotifyItem.serializer()),
    )

    suspend fun unreadSummary(): ApiResult<UnreadSummary> =
        api.post(ApiEndpoint.NOTIFY_UNREAD_SUMMARY, emptyObject(), UnreadSummary.serializer())

    suspend fun markAllRead(): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.NOTIFY_MARK_READ,
        buildJsonObject {
            put("scope", "all")
            put("notify_ids", JsonNull)
        },
    )
}
