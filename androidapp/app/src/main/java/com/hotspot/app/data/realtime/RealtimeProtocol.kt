package com.hotspot.app.data.realtime

import com.hotspot.app.data.model.GroupMessageItem
import com.hotspot.app.data.model.PrivateMessageItem
import com.hotspot.app.data.model.UserInfo
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

/**
 * WebSocket 推送事件的强类型封装（spec/api_spec_websocket.md）。
 *
 * 只暴露聊天与消息列表场景需要的业务事件；连接管理事件（ConnectionEstablished / Pong /
 * Error / SessionRevoked / AccountBanned）由 [HotSpotRealtimeClient] 内部处理，
 * SessionRevoked / AccountBanned 以 [SessionRevoked] / [AccountBanned] 的形式广播给 UI。
 */
sealed interface RealtimeEvent {
    data class PrivateMessageNew(
        val message: PrivateMessageItem,
        val targetUserId: String,
    ) : RealtimeEvent

    data class PrivateMessageEdited(
        val messageId: String,
        val targetUserId: String,
        val content: String,
        val editedAt: String,
    ) : RealtimeEvent

    data class PrivateMessageRecalled(
        val messageId: String,
        val targetUserId: String,
        val recalledBy: UserInfo?,
        val recalledAt: String,
    ) : RealtimeEvent

    data class PrivateMessageRead(
        val upToMessageId: String,
        val readAt: String,
    ) : RealtimeEvent

    data class PrivateTyping(
        val targetUserId: String,
        val actor: UserInfo,
        val isTyping: Boolean,
    ) : RealtimeEvent

    data class GroupMessageNew(
        val message: GroupMessageItem,
        val groupId: String,
    ) : RealtimeEvent

    data class GroupMessageEdited(
        val messageId: String,
        val groupId: String,
        val content: String,
        val editedAt: String,
    ) : RealtimeEvent

    data class GroupMessageRecalled(
        val messageId: String,
        val groupId: String,
        val recalledBy: UserInfo?,
        val recalledAt: String,
    ) : RealtimeEvent

    data class GroupMessageReadCountUpdated(
        val groupId: String,
        val messageId: String,
        val readCount: Int,
    ) : RealtimeEvent

    data class GroupTyping(
        val groupId: String,
        val actor: UserInfo,
        val isTyping: Boolean,
    ) : RealtimeEvent

    /**
     * 表情反应新增/移除（spec api_spec_websocket.md → ReactionAdded / ReactionRemoved）。
     * 事件只带 target_type/target_id/emoji/actor，不带最新计数：聊天页按 ±1 本地合并，
     * actor 用于判定「是不是我发的反应」（对齐 Web / 鸿蒙实现）。
     */
    data class ReactionAdded(
        val targetType: String,
        val targetId: String,
        val emoji: String,
        val actor: UserInfo,
    ) : RealtimeEvent

    data class ReactionRemoved(
        val targetType: String,
        val targetId: String,
        val emoji: String,
        val actor: UserInfo,
    ) : RealtimeEvent

    data class SessionRevoked(val reason: String) : RealtimeEvent

    data class AccountBanned(val reason: String) : RealtimeEvent
}

/**
 * WebSocket 协议纯函数层：事件解析 / 重连退避 / 订阅集合。
 * 不依赖 OkHttp 与 Android，便于在 JVM 单测中直接验证契约字段。
 */
object RealtimeProtocol {
    val json: Json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
        coerceInputValues = true
    }

    private const val HEARTBEAT_INTERVAL_FALLBACK_MS = 30_000L
    private const val HEARTBEAT_TIMEOUT_FALLBACK_MS = 10_000L
    private const val RECONNECT_BASE_MS = 1_000L
    private const val RECONNECT_MAX_MS = 60_000L

    /** ConnectionEstablished.config 下发的动态心跳参数（禁止硬编码，fallback 仅用于解析异常）。 */
    fun parseHeartbeat(payload: JsonObject): Pair<Long, Long> {
        val config = payload["config"]?.jsonObject
        val intervalSeconds = config?.get("ping_interval")?.jsonPrimitive?.content?.toLongOrNull()
        val timeoutSeconds = config?.get("ping_timeout")?.jsonPrimitive?.content?.toLongOrNull()
        val intervalMs = intervalSeconds?.times(1_000) ?: HEARTBEAT_INTERVAL_FALLBACK_MS
        val timeoutMs = timeoutSeconds?.times(1_000) ?: HEARTBEAT_TIMEOUT_FALLBACK_MS
        return intervalMs.coerceAtLeast(1_000L) to timeoutMs.coerceAtLeast(1_000L)
    }

    /** 指数退避 + 抖动（1s、2s、4s…上限 60s）；jitterMs 由调用方注入便于单测。 */
    fun reconnectDelayMs(attempt: Int, jitterMs: Long = 0L): Long {
        val exponent = attempt.coerceIn(0, 6)
        val base = RECONNECT_BASE_MS shl exponent
        return (base.coerceAtMost(RECONNECT_MAX_MS) + jitterMs).coerceAtMost(RECONNECT_MAX_MS + 5_000L)
    }

    /**
     * ConnectionEstablished.resume_token：缺省 / 显式 null / 空白串均视为无恢复令牌
     * （走 SetSubscriptions 全量恢复）。不能直接取 JsonNull.content——那会得到字符串 "null"。
     */
    fun parseResumeToken(payload: JsonObject): String? {
        val raw = payload["resume_token"] ?: return null
        if (raw is JsonNull) return null
        return runCatching { raw.jsonPrimitive.content }.getOrNull()?.takeIf { it.isNotBlank() }
    }

    fun parse(type: String, payload: JsonObject): RealtimeEvent? = when (type) {
        "PrivateMessageNew" -> {
            val message = runCatching {
                json.decodeFromJsonElement(PrivateMessageItem.serializer(), payload)
            }.getOrNull()
            message?.let {
                RealtimeEvent.PrivateMessageNew(
                    message = it,
                    targetUserId = payload.stringOrNull("target_user_id") ?: it.sender.userId,
                )
            }
        }

        "PrivateMessageEdited" -> RealtimeEvent.PrivateMessageEdited(
            messageId = payload.stringOrNull("message_id") ?: return null,
            targetUserId = payload.stringOrNull("target_user_id") ?: return null,
            content = payload.stringOrNull("content") ?: "",
            editedAt = payload.stringOrNull("edited_at") ?: "",
        )

        "PrivateMessageRecalled" -> RealtimeEvent.PrivateMessageRecalled(
            messageId = payload.stringOrNull("message_id") ?: return null,
            targetUserId = payload.stringOrNull("target_user_id") ?: return null,
            recalledBy = payload["recalled_by"]?.let { decodeUser(it) },
            recalledAt = payload.stringOrNull("recalled_at") ?: "",
        )

        "PrivateMessageRead" -> RealtimeEvent.PrivateMessageRead(
            upToMessageId = payload.stringOrNull("up_to_message_id") ?: return null,
            readAt = payload.stringOrNull("read_at") ?: "",
        )

        "PrivateTyping" -> RealtimeEvent.PrivateTyping(
            targetUserId = payload.stringOrNull("target_user_id") ?: return null,
            actor = payload["actor"]?.let { decodeUser(it) } ?: UserInfo(),
            isTyping = payload.booleanOrNull("is_typing") ?: false,
        )

        "GroupMessageNew" -> {
            val message = runCatching {
                json.decodeFromJsonElement(GroupMessageItem.serializer(), payload)
            }.getOrNull()
            message?.let {
                RealtimeEvent.GroupMessageNew(
                    message = it,
                    groupId = payload.stringOrNull("group_id") ?: return null,
                )
            }
        }

        "GroupMessageEdited" -> RealtimeEvent.GroupMessageEdited(
            messageId = payload.stringOrNull("message_id") ?: return null,
            groupId = payload.stringOrNull("group_id") ?: return null,
            content = payload.stringOrNull("content") ?: "",
            editedAt = payload.stringOrNull("edited_at") ?: "",
        )

        "GroupMessageRecalled" -> RealtimeEvent.GroupMessageRecalled(
            messageId = payload.stringOrNull("message_id") ?: return null,
            groupId = payload.stringOrNull("group_id") ?: return null,
            recalledBy = payload["recalled_by"]?.let { decodeUser(it) },
            recalledAt = payload.stringOrNull("recalled_at") ?: "",
        )

        "GroupMessageReadCountUpdated" -> RealtimeEvent.GroupMessageReadCountUpdated(
            groupId = payload.stringOrNull("group_id") ?: return null,
            messageId = payload.stringOrNull("message_id") ?: return null,
            readCount = payload["read_count"]?.jsonPrimitive?.content?.toIntOrNull() ?: 0,
        )

        "GroupTyping" -> RealtimeEvent.GroupTyping(
            groupId = payload.stringOrNull("group_id") ?: return null,
            actor = payload["actor"]?.let { decodeUser(it) } ?: UserInfo(),
            isTyping = payload.booleanOrNull("is_typing") ?: false,
        )

        "ReactionAdded" -> RealtimeEvent.ReactionAdded(
            targetType = payload.stringOrNull("target_type") ?: return null,
            targetId = payload.stringOrNull("target_id") ?: return null,
            emoji = payload.stringOrNull("emoji") ?: return null,
            actor = payload["actor"]?.let { decodeUser(it) } ?: UserInfo(),
        )

        "ReactionRemoved" -> RealtimeEvent.ReactionRemoved(
            targetType = payload.stringOrNull("target_type") ?: return null,
            targetId = payload.stringOrNull("target_id") ?: return null,
            emoji = payload.stringOrNull("emoji") ?: return null,
            actor = payload["actor"]?.let { decodeUser(it) } ?: UserInfo(),
        )

        else -> null
    }

    /** 消息列表需要被动刷新的聊天类事件。 */
    fun isChatListAffecting(event: RealtimeEvent): Boolean = when (event) {
        is RealtimeEvent.PrivateMessageNew,
        is RealtimeEvent.PrivateMessageEdited,
        is RealtimeEvent.PrivateMessageRecalled,
        is RealtimeEvent.GroupMessageNew,
        is RealtimeEvent.GroupMessageEdited,
        is RealtimeEvent.GroupMessageRecalled,
        -> true

        else -> false
    }

    /** 本地订阅集合：进入聊天页订阅对端/群，离开时取消；重连后按此集合 set 全量恢复。 */
    data class SubscriptionSet(
        val userIds: Set<String> = emptySet(),
        val groupIds: Set<String> = emptySet(),
    ) {
        fun withUsers(ids: Collection<String>): SubscriptionSet = copy(userIds = userIds + ids)

        fun withoutUsers(ids: Collection<String>): SubscriptionSet = copy(userIds = userIds - ids.toSet())

        fun withGroups(ids: Collection<String>): SubscriptionSet = copy(groupIds = groupIds + ids)

        fun withoutGroups(ids: Collection<String>): SubscriptionSet = copy(groupIds = groupIds - ids.toSet())
    }

    private fun decodeUser(element: kotlinx.serialization.json.JsonElement): UserInfo? = runCatching {
        json.decodeFromJsonElement(UserInfo.serializer(), element)
    }.getOrNull()

    private fun JsonObject.stringOrNull(key: String): String? =
        this[key]?.let { runCatching { it.jsonPrimitive.content }.getOrNull() }

    private fun JsonObject.booleanOrNull(key: String): Boolean? =
        this[key]?.let { runCatching { it.jsonPrimitive.content.toBooleanStrictOrNull() }.getOrNull() }
}
