package com.hotspot.app.util

import com.hotspot.app.data.model.ShareListItem
import com.hotspot.app.data.model.SharePreview
import org.junit.Assert.assertEquals
import org.junit.Test

class ShareFilterTest {
    private val items = listOf(
        ShareListItem(
            shareId = "s1",
            shareCode = "abc123",
            shareType = "moment",
            createdAt = "2026-11-28T00:00:00Z",
            preview = SharePreview(title = "云龙湖落日", subtitle = "阿禾的瞬间"),
        ),
        ShareListItem(
            shareId = "s2",
            shareCode = "def456",
            shareType = "group",
            createdAt = "2026-11-27T00:00:00Z",
            isRevoked = true,
            preview = null,
        ),
    )

    @Test
    fun `空词返回原列表`() {
        assertEquals(items, filterShares(items, "   "))
    }

    @Test
    fun `匹配标题与副标题`() {
        assertEquals(listOf("s1"), filterShares(items, "云龙湖").map { it.shareId })
        assertEquals(listOf("s1"), filterShares(items, "阿禾").map { it.shareId })
    }

    @Test
    fun `匹配类型文案与分享码`() {
        assertEquals(listOf("s2"), filterShares(items, "群组").map { it.shareId })
        assertEquals(listOf("s1"), filterShares(items, "abc123").map { it.shareId })
    }

    @Test
    fun `匹配可用状态`() {
        assertEquals(listOf("s2"), filterShares(items, "已失效").map { it.shareId })
    }

    @Test
    fun `无匹配返回空列表`() {
        assertEquals(emptyList<ShareListItem>(), filterShares(items, "不存在的关键词"))
    }
}
