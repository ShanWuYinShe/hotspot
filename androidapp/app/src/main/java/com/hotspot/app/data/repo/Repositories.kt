package com.hotspot.app.data.repo

import com.hotspot.app.data.api.ApiClient
import com.hotspot.app.data.api.ApiEndpoint
import com.hotspot.app.data.api.ApiError
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.auth.AuthSession
import com.hotspot.app.data.auth.TokenStore
import com.hotspot.app.data.model.AccountData
import com.hotspot.app.data.model.AuthTokenPair
import com.hotspot.app.data.model.GroupItem
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

    /** 瞬间详情（登录视角 MomentItem）。 */
    suspend fun openMoment(momentId: String): ApiResult<MomentItem> = api.post(
        ApiEndpoint.MOMENTS_OPEN,
        buildJsonObject { put("moment_id", momentId) },
        MomentItem.serializer(),
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

/** 个人中心：账户资料（含仅自己可见的 email/phone/stats）。 */
class ProfileRepository(private val api: ApiClient) {
    suspend fun account(): ApiResult<AccountData> =
        api.post(ApiEndpoint.USERS_ACCOUNT, emptyObject(), AccountData.serializer())
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
