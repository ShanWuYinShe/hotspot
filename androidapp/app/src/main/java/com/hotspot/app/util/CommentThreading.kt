package com.hotspot.app.util

import com.hotspot.app.data.model.CommentItem
import com.hotspot.app.data.model.Reaction

/**
 * 评论楼中楼归组与乐观点赞的纯逻辑（无 Compose / Android 依赖，可纯 JVM 单测）。
 *
 * 数据来源：comments/fetch 返回顶层评论（parent_id = null），其 top_replies 携带前 N 条回复预览；
 * 若后续在扁平列表里再收到 parent_id 非空的回复（分页/本地追加），同样按 parent_id 归组。
 */

/** 楼中楼线程：一条顶层评论 + 其下回复（回复按 created_at 正序，服务端已排好）。 */
data class CommentThread(
    val comment: CommentItem,
    val replies: List<CommentItem>,
)

/** 评论点赞 emoji（须在服务端 reaction_emojis 白名单内，与 Web / 鸿蒙同口径）。 */
const val COMMENT_LIKE_EMOJI = "👍"

/**
 * 把扁平评论列表按 parent_id 归组为「顶层评论 + 回复」线程列表。
 * - 顶层顺序保持不变（服务端 created_at 倒序）；
 * - top_replies 预览 + 扁平列表中的同父回复合并去重（分页/本地追加可能重复）；
 * - 父评论不在当前页的孤立回复不作为主评论渲染（避免把回复误当顶层，首版不处理该分页边界）。
 */
fun groupCommentThreads(items: List<CommentItem>): List<CommentThread> {
    val roots = items.filter { it.parentId.isNullOrBlank() }
    val repliesByParent = items
        .filter { !it.parentId.isNullOrBlank() }
        .groupBy { it.parentId.orEmpty() }
    return roots.map { root ->
        val preview = root.topReplies.filter {
            it.parentId.isNullOrBlank() || it.parentId == root.commentId
        }
        val merged = (preview + repliesByParent[root.commentId].orEmpty())
            .distinctBy { it.commentId }
        CommentThread(comment = root, replies = merged)
    }
}

/** 加载更多：追加下一页评论并按 comment_id 去重（cursor 在页面原地重试时不会产生重复行）。 */
fun mergeCommentPage(current: List<CommentItem>, incoming: List<CommentItem>): List<CommentItem> {
    val known = current.mapTo(mutableSetOf()) { it.commentId }
    return current + incoming.filter { known.add(it.commentId) }
}

/**
 * 评论点赞的乐观转换：返回翻转后的评论副本（reactions 聚合同步 ±1）。
 * - active = true：已有我的反应则复用该 emoji（计数 +1、reacted_by_me = true），否则新增 👍 反应；
 * - active = false：移除我的反应（计数 -1，归零后删除该 emoji 条目）；
 * - 已经是目标态时原样返回（防连点重复翻转）。
 */
fun toggleCommentLike(
    comment: CommentItem,
    active: Boolean,
    emoji: String = COMMENT_LIKE_EMOJI,
): CommentItem {
    val mine: Reaction? = comment.reactions.firstOrNull { it.reactedByMe }
    if (active == (mine != null)) return comment
    val reactions = comment.reactions.toMutableList()
    if (active) {
        val index = reactions.indexOfFirst { it.emoji == emoji }
        if (index >= 0) {
            reactions[index] = reactions[index].copy(
                count = reactions[index].count + 1,
                reactedByMe = true,
            )
        } else {
            reactions.add(Reaction(emoji = emoji, count = 1, reactedByMe = true))
        }
    } else if (mine != null) {
        val index = reactions.indexOfFirst { it.emoji == mine.emoji }
        if (index >= 0) {
            val count = (reactions[index].count - 1).coerceAtLeast(0)
            if (count == 0) reactions.removeAt(index) else reactions[index] = reactions[index].copy(count = count, reactedByMe = false)
        }
    }
    return comment.copy(reactions = reactions)
}
