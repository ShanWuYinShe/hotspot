package com.hotspot.app.util

import org.junit.Assert.assertEquals
import org.junit.Test

class MessageActionRulesTest {
    @Test
    fun ownTextMessageCanCopyAndRecall() {
        assertEquals(
            listOf(MessageAction.LIKE, MessageAction.COPY, MessageAction.RECALL),
            MessageActionRules.availableActions(isMine = true, isRecalled = false, type = "text", content = "hi"),
        )
    }

    @Test
    fun otherTextMessageCanCopyAndReport() {
        assertEquals(
            listOf(MessageAction.LIKE, MessageAction.COPY, MessageAction.REPORT),
            MessageActionRules.availableActions(isMine = false, isRecalled = false, type = "text", content = "hi"),
        )
    }

    @Test
    fun imageMessageHasNoCopyOnlyRecallOrReport() {
        assertEquals(
            listOf(MessageAction.LIKE, MessageAction.RECALL),
            MessageActionRules.availableActions(isMine = true, isRecalled = false, type = "image", content = null),
        )
        assertEquals(
            listOf(MessageAction.LIKE, MessageAction.REPORT),
            MessageActionRules.availableActions(isMine = false, isRecalled = false, type = "image", content = null),
        )
    }

    @Test
    fun recalledOrSystemMessageHasNoActions() {
        assertEquals(emptyList<MessageAction>(), MessageActionRules.availableActions(true, true, "text", "hi"))
        assertEquals(emptyList<MessageAction>(), MessageActionRules.availableActions(false, false, "system", null))
    }
}
