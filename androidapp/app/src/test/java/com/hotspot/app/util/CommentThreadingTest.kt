package com.hotspot.app.util

import com.hotspot.app.data.model.CommentItem
import com.hotspot.app.data.model.Reaction
import com.hotspot.app.data.model.UserInfo
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertSame
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * 评论区纯逻辑单测：楼中楼按 parent_id 归组、分页去重、乐观点赞翻转。
 * 这些是端上评论 UI 的正确性底座（不依赖 Android / Compose，纯 JVM）。
 */
class CommentThreadingTest {

    private fun comment(
        id: String,
        parentId: String? = null,
        nickname: String = "用户" + id,
        topReplies: List<CommentItem> = emptyList(),
        reactions: List<Reaction> = emptyList(),
        isDeleted: Boolean = false,
    ): CommentItem = CommentItem(
        commentId = id,
        user = UserInfo(userId = "u_" + id, nickname = nickname),
        content = "内容 " + id,
        parentId = parentId,
        createdAt = "2026-11-20T00:00:00Z",
        reactions = reactions,
        isDeleted = isDeleted,
        topReplies = topReplies,
    )

    @Test
    fun groupCommentThreads_groupsRepliesByParentId() {
        val rootA = comment("a", topReplies = listOf(comment("a1", parentId = "a")))
        val rootB = comment("b")
        val flatReplyB = comment("b1", parentId = "b")
        val threads = groupCommentThreads(listOf(rootA, rootB, flatReplyB))

        assertEquals(listOf("a", "b"), threads.map { it.comment.commentId })
        assertEquals(listOf("a1"), threads[0].replies.map { it.commentId })
        assertEquals(listOf("b1"), threads[1].replies.map { it.commentId })
    }

    @Test
    fun groupCommentThreads_keepsTopLevelOrderAndDedupes() {
        // top_replies 预览与扁平列表重复同一条回复时只保留一次
        val root = comment("root", topReplies = listOf(comment("r1", parentId = "root")))
        val threads = groupCommentThreads(listOf(root, comment("r1", parentId = "root")))
        assertEquals(1, threads.size)
        assertEquals(listOf("r1"), threads[0].replies.map { it.commentId })
    }

    @Test
    fun groupCommentThreads_orphanReplyIsNotPromotedToRoot() {
        val threads = groupCommentThreads(listOf(comment("orphan", parentId = "missing")))
        assertTrue(threads.isEmpty())
    }

    @Test
    fun findAndUpdateCommentInThreads_coversReplies() {
        val root = comment("a", topReplies = listOf(comment("a1", parentId = "a")))
        val items = listOf(root, comment("b"))

        assertEquals("a1", findCommentInThreads(items, "a1")?.commentId)
        assertNull(findCommentInThreads(items, "missing"))

        val updated = updateCommentInThreads(items, "a1") { it.copy(content = "更新后的回复") }
        assertEquals("更新后的回复", updated.first().topReplies.single().content)
        assertEquals("内容 b", updated[1].content)
    }

    @Test
    fun markCommentDeleted_softDeletesOnlyTargetAndKeepsReplies() {
        val root = comment("a", topReplies = listOf(comment("a1", parentId = "a")))
        val other = comment("b")
        val updated = markCommentDeleted(listOf(root, other), "a")

        val deleted = updated.first { it.commentId == "a" }
        assertTrue(deleted.isDeleted)
        assertEquals("", deleted.content)
        assertEquals(listOf("a1"), deleted.topReplies.map { it.commentId })
        assertEquals("内容 b", updated.first { it.commentId == "b" }.content)
        assertFalse(updated.first { it.commentId == "b" }.isDeleted)
    }

    @Test
    fun mergeCommentPage_appendsAndDedupesById() {
        val first = listOf(comment("a"), comment("b"))
        val merged = mergeCommentPage(first, listOf(comment("b"), comment("c")))
        assertEquals(listOf("a", "b", "c"), merged.map { it.commentId })
    }

    @Test
    fun toggleCommentLike_addsMyReactionOptimistically() {
        val original = comment("c1")
        val liked = toggleCommentLike(original, active = true)
        assertTrue(liked.liked)
        assertEquals(1, liked.likeCount)
        assertEquals(COMMENT_LIKE_EMOJI, liked.reactions.single().emoji)
    }

    @Test
    fun toggleCommentLike_incrementsExistingEmojiAggregate() {
        val original = comment("c1", reactions = listOf(Reaction(emoji = "👍", count = 2, reactedByMe = false)))
        val liked = toggleCommentLike(original, active = true)
        assertTrue(liked.liked)
        assertEquals(3, liked.likeCount)
        // 复用其它用户已存在的同 emoji 聚合，而不是新增第二条 👍
        assertEquals(1, liked.reactions.size)
        assertTrue(liked.reactions.single().reactedByMe)
    }

    @Test
    fun toggleCommentLike_removesMyReactionAndClampsToZero() {
        val original = comment("c1", reactions = listOf(Reaction(emoji = "👍", count = 2, reactedByMe = true)))
        val unliked = toggleCommentLike(original, active = false)
        assertFalse(unliked.liked)
        assertEquals(1, unliked.likeCount)

        val onlyMine = comment("c2", reactions = listOf(Reaction(emoji = "👍", count = 1, reactedByMe = true)))
        val cleared = toggleCommentLike(onlyMine, active = false)
        assertFalse(cleared.liked)
        assertEquals(0, cleared.likeCount)
        assertTrue(cleared.reactions.isEmpty())
    }

    @Test
    fun toggleCommentLike_isNoOpWhenAlreadyInTargetState() {
        val liked = comment("c1", reactions = listOf(Reaction(emoji = "👍", count = 1, reactedByMe = true)))
        assertSame(liked, toggleCommentLike(liked, active = true))
        val unliked = comment("c2")
        assertSame(unliked, toggleCommentLike(unliked, active = false))
    }
}
