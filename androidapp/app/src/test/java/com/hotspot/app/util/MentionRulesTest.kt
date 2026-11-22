package com.hotspot.app.util

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class MentionRulesTest {
    private val members = listOf(
        "user_a" to "阿禾",
        "user_b" to "Mika",
    )

    @Test
    fun queryAtStartOfInput() {
        val query = MentionRules.findActiveQuery("@阿")
        assertEquals(MentionQuery(0, "阿"), query)
    }

    @Test
    fun queryAfterWhitespace() {
        val query = MentionRules.findActiveQuery("你好 @Mi")
        assertEquals(MentionQuery(3, "Mi"), query)
    }

    @Test
    fun emailLikeAtIsNotMentionQuery() {
        assertNull(MentionRules.findActiveQuery("mail@example"))
    }

    @Test
    fun finishedMentionWithSpaceClosesPanel() {
        assertNull(MentionRules.findActiveQuery("@阿禾 说"))
    }

    @Test
    fun insertMentionReplacesQueryAndAddsSpace() {
        val text = "你好 @阿"
        val query = MentionRules.findActiveQuery(text)!!
        val inserted = MentionRules.insertMention(text, text.length, query, "阿禾")
        assertEquals("你好 @阿禾 ", inserted)
        assertTrue(MentionRules.containsMention(inserted, "阿禾"))
    }

    @Test
    fun onlyStillPresentMentionsAreSent() {
        val text = "@阿禾 你看 @Mika"
        val ids = MentionRules.activeMentionIds(text, members, setOf("user_a", "user_b"))
        assertEquals(listOf("user_a", "user_b"), ids)
        val afterEdit = "你看 @Mika"
        assertEquals(
            listOf("user_b"),
            MentionRules.activeMentionIds(afterEdit, members, setOf("user_a", "user_b")),
        )
        assertFalse(MentionRules.containsMention("你看 @禾", "阿禾"))
    }

    @Test
    fun mentionMustBeFollowedByBoundary() {
        assertFalse(MentionRules.containsMention("@阿禾禾", "阿禾"))
        assertTrue(MentionRules.containsMention("@阿禾 你好", "阿禾"))
        assertTrue(MentionRules.containsMention("@阿禾", "阿禾"))
    }
}
