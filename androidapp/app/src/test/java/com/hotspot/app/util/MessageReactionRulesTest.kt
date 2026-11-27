package com.hotspot.app.util

import com.hotspot.app.data.model.Reaction
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class MessageReactionRulesTest {
    private val base = listOf(Reaction(emoji = "👍", count = 2, reactedByMe = false), Reaction(emoji = "❤️", count = 1, reactedByMe = true))

    @Test
    fun `他人新增已有 emoji 计数加一且不改 mine`() {
        val next = applyReactionDelta(base, "👍", true, false)
        assertEquals(Reaction("👍", 3, false), next.first { it.emoji == "👍" })
    }

    @Test
    fun `新增 emoji 追加计数一`() {
        val next = applyReactionDelta(base, "🔥", true, false)
        assertEquals(Reaction("🔥", 1, false), next.first { it.emoji == "🔥" })
    }

    @Test
    fun `自己的新增标记 reactedByMe`() {
        val next = applyReactionDelta(base, "🔥", true, true)
        assertEquals(Reaction("🔥", 1, true), next.first { it.emoji == "🔥" })
    }

    @Test
    fun `移除后计数归零的 emoji 被删除`() {
        val next = applyReactionDelta(base, "❤️", false, true)
        assertNull(next.firstOrNull { it.emoji == "❤️" })
    }

    @Test
    fun `他人移除保留 reactedByMe`() {
        val next = applyReactionDelta(base, "❤️", false, false)
        assertNull(next.firstOrNull { it.emoji == "❤️" })
        val thumbs = applyReactionDelta(listOf(Reaction("👍", 2, true)), "👍", false, false)
        assertEquals(Reaction("👍", 1, true), thumbs.first())
    }

    @Test
    fun `空 emoji 原样返回`() {
        assertEquals(base, applyReactionDelta(base, "", true, false))
    }
}
