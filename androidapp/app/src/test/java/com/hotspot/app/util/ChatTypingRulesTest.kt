package com.hotspot.app.util

import org.junit.Assert.assertEquals
import org.junit.Test

/** 聊天输入中提示纯规则单测（对齐 harmonyapp ChatTypingRules 文案口径）。 */
class ChatTypingRulesTest {
    private val names = mapOf(
        "user_aaa" to "阿禾",
        "user_bbb" to "Mika",
        "user_ccc" to "野原",
        "user_ddd" to "青砚",
    )

    private fun nameOf(id: String): String? = names[id]

    @Test
    fun singleTypingUserShowsName() {
        assertEquals("阿禾 正在输入…", ChatTypingRules.buildGroupTypingHint(listOf("user_aaa"), ::nameOf))
    }

    @Test
    fun twoTypingUsersJoinedByPauseMark() {
        assertEquals(
            "阿禾、Mika 正在输入…",
            ChatTypingRules.buildGroupTypingHint(listOf("user_aaa", "user_bbb"), ::nameOf),
        )
    }

    @Test
    fun threeTypingUsersJoinedByPauseMark() {
        assertEquals(
            "阿禾、Mika、野原 正在输入…",
            ChatTypingRules.buildGroupTypingHint(listOf("user_aaa", "user_bbb", "user_ccc"), ::nameOf),
        )
    }

    @Test
    fun moreThanThreeUsersFoldedWithCount() {
        assertEquals(
            "阿禾、Mika、野原 等 4 人正在输入…",
            ChatTypingRules.buildGroupTypingHint(
                listOf("user_aaa", "user_bbb", "user_ccc", "user_ddd"),
                ::nameOf,
            ),
        )
    }

    @Test
    fun unknownUserFallsBackToMemberSuffix() {
        assertEquals("成员1234", ChatTypingRules.resolveTypingDisplayName("user_1234") { null })
        assertEquals("成员xyz", ChatTypingRules.resolveTypingDisplayName("xyz") { null })
    }

    @Test
    fun blankAndDuplicateIdsAreIgnored() {
        assertEquals("", ChatTypingRules.buildGroupTypingHint(emptyList(), ::nameOf))
        assertEquals("阿禾 正在输入…", ChatTypingRules.buildGroupTypingHint(listOf(" ", "user_aaa", "user_aaa"), ::nameOf))
    }
}
