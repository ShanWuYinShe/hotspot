package com.hotspot.app.util

import org.junit.Assert.assertEquals
import org.junit.Test

class MessageActionRulesTest {
    @Test
    fun ownTextMessageCanCopyEditAndRecall() {
        assertEquals(
            listOf(MessageAction.LIKE, MessageAction.COPY, MessageAction.EDIT, MessageAction.RECALL),
            MessageActionRules.availableActions(isMine = true, isRecalled = false, type = "text", content = "hi"),
        )
    }

    @Test
    fun otherTextMessageCanCopyAndReportButNotEdit() {
        assertEquals(
            listOf(MessageAction.LIKE, MessageAction.COPY, MessageAction.REPORT),
            MessageActionRules.availableActions(isMine = false, isRecalled = false, type = "text", content = "hi"),
        )
    }

    @Test
    fun imageMessageHasNoCopyOrEditOnlyRecallOrReport() {
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
    fun blankTextMessageCannotCopyOrEdit() {
        assertEquals(
            listOf(MessageAction.LIKE, MessageAction.RECALL),
            MessageActionRules.availableActions(isMine = true, isRecalled = false, type = "text", content = "  "),
        )
    }

    @Test
    fun recalledOrSystemMessageHasNoActions() {
        assertEquals(emptyList<MessageAction>(), MessageActionRules.availableActions(true, true, "text", "hi"))
        assertEquals(emptyList<MessageAction>(), MessageActionRules.availableActions(false, false, "system", null))
    }
}
