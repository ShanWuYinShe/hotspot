package com.hotspot.app.data.realtime

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * WebSocket 协议层单测（纯 JVM，不依赖 Android/OkHttp）：
 * 校验 spec/api_spec_websocket.md 的事件负载字段、心跳配置解析、重连退避与订阅集合。
 */
class RealtimeProtocolTest {
    private val json = Json { ignoreUnknownKeys = true; explicitNulls = false; coerceInputValues = true }

    @Test
    fun privateMessageNewParsesTargetAndMessage() {
        val payload = json.parseToJsonElement(
            """
            {
              "message_id": "msg_1",
              "target_user_id": "user_bob",
              "sender": {"user_id": "user_alice", "nickname": "Alice", "avatar_url": null},
              "type": "text",
              "content": "hello",
              "share_code": null,
              "reply_to": null,
              "media": null,
              "sent_at": "2026-11-22T06:30:00Z"
            }
            """.trimIndent(),
        ).jsonObject
        val event = RealtimeProtocol.parse("PrivateMessageNew", payload) as RealtimeEvent.PrivateMessageNew
        assertEquals("user_bob", event.targetUserId)
        assertEquals("msg_1", event.message.messageId)
        assertEquals("hello", event.message.content)
        assertEquals("user_alice", event.message.sender.userId)
    }

    @Test
    fun groupMessageNewParsesGroupAndMentions() {
        val payload = json.parseToJsonElement(
            """
            {
              "message_id": "gmsg_1",
              "group_id": "group_1",
              "sender": {"user_id": "user_alice", "nickname": "Alice", "avatar_url": null},
              "type": "text",
              "content": "@Bob 看这里",
              "share_code": null,
              "reply_to": null,
              "mentions": ["user_bob"],
              "media": null,
              "sent_at": "2026-11-22T06:31:00Z"
            }
            """.trimIndent(),
        ).jsonObject
        val event = RealtimeProtocol.parse("GroupMessageNew", payload) as RealtimeEvent.GroupMessageNew
        assertEquals("group_1", event.groupId)
        assertEquals(listOf("user_bob"), event.message.mentions)
        assertEquals("user_alice", event.message.sender.userId)
    }

    @Test
    fun privateMessageReadParsesUpToMessageId() {
        val payload = json.parseToJsonElement(
            """{"up_to_message_id":"msg_9","read_at":"2026-11-22T06:32:00Z"}""",
        ).jsonObject
        val event = RealtimeProtocol.parse("PrivateMessageRead", payload) as RealtimeEvent.PrivateMessageRead
        assertEquals("msg_9", event.upToMessageId)
    }

    @Test
    fun unknownEventIsIgnored() {
        assertNull(RealtimeProtocol.parse("SystemAnnouncement", json.parseToJsonElement("{}").jsonObject))
    }

    @Test
    fun heartbeatConfigUsesServerValues() {
        val payload = json.parseToJsonElement(
            """{"config":{"ping_interval":30,"ping_timeout":10,"max_subscriptions":{"user":500,"group":100}}}""",
        ).jsonObject
        val (interval, timeout) = RealtimeProtocol.parseHeartbeat(payload)
        assertEquals(30_000L, interval)
        assertEquals(10_000L, timeout)
    }

    @Test
    fun heartbeatConfigFallsBackWhenMissing() {
        val (interval, timeout) = RealtimeProtocol.parseHeartbeat(json.parseToJsonElement("{}").jsonObject)
        assertEquals(30_000L, interval)
        assertEquals(10_000L, timeout)
    }

    @Test
    fun reconnectBackoffGrowsAndCaps() {
        assertEquals(1_000L, RealtimeProtocol.reconnectDelayMs(0))
        assertEquals(2_000L, RealtimeProtocol.reconnectDelayMs(1))
        assertEquals(4_000L, RealtimeProtocol.reconnectDelayMs(2))
        assertEquals(60_000L, RealtimeProtocol.reconnectDelayMs(10))
        assertEquals(60_000L, RealtimeProtocol.reconnectDelayMs(99))
        assertTrue(RealtimeProtocol.reconnectDelayMs(2, jitterMs = 300L) in 4_300L..4_301L)
    }

    @Test
    fun subscriptionSetAddsAndRemovesIndependently() {
        val empty = RealtimeProtocol.SubscriptionSet()
        val users = empty.withUsers(listOf("u1", "u2")).withGroups(listOf("g1"))
        assertEquals(setOf("u1", "u2"), users.userIds)
        assertEquals(setOf("g1"), users.groupIds)
        val afterRemove = users.withoutUsers(listOf("u1")).withoutGroups(listOf("g1"))
        assertEquals(setOf("u2"), afterRemove.userIds)
        assertTrue(afterRemove.groupIds.isEmpty())
        assertNotNull(users)
    }
}
