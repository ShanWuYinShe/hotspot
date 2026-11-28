package com.hotspot.app.util

import com.hotspot.app.data.model.NotifyItem
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import org.junit.Assert.assertEquals
import org.junit.Test

class NotifyFilterTest {
    private val items = listOf(
        NotifyItem(notifyId = "n1", type = "reaction", isRead = false),
        NotifyItem(notifyId = "n2", type = "follow", isRead = true),
        NotifyItem(notifyId = "n3", type = "comment", isRead = false, payload = buildJsonObject { put("text", "云龙湖") }),
        NotifyItem(notifyId = "n4", type = "nearby", isRead = true, payload = JsonNull),
    )

    @Test
    fun `全部与未读档不按 type 过滤`() {
        assertEquals(4, filterNotifyItems(items, "", "all").size)
        assertEquals(4, filterNotifyItems(items, "", "unread").size)
    }

    @Test
    fun `分类档只保留对应 type`() {
        assertEquals(listOf("n1"), filterNotifyItems(items, "", "reaction").map { it.notifyId })
        assertEquals(listOf("n3"), filterNotifyItems(items, "", "comment").map { it.notifyId })
    }

    @Test
    fun `关键词匹配 payload 内容与类别名`() {
        assertEquals(listOf("n3"), filterNotifyItems(items, "云龙湖", "all").map { it.notifyId })
        assertEquals(listOf("n1", "n2", "n3", "n4"), filterNotifyItems(items, "n", "all").map { it.notifyId })
    }

    @Test
    fun `分类与关键词同时生效`() {
        assertEquals(emptyList<String>(), filterNotifyItems(items, "云龙湖", "reaction").map { it.notifyId })
        assertEquals(listOf("n1"), filterNotifyItems(items, "reaction", "reaction").map { it.notifyId })
    }

    @Test
    fun `空列表与未知分类返回空`() {
        assertEquals(emptyList<NotifyItem>(), filterNotifyItems(emptyList(), "x", "all"))
        assertEquals(emptyList<NotifyItem>(), filterNotifyItems(items, "", "unknown"))
    }
}
