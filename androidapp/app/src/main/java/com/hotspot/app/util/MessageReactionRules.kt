package com.hotspot.app.util

import com.hotspot.app.data.model.Reaction

/** 消息表情规则：与后端 reactions/set 白名单同口径，当前仅上点赞。 */
const val THUMB_REACTION = "👍"

/** 乐观切换 👍 反应：active=true 加一；false 减一并在归零时移除。 */
fun toggleThumbReaction(reactions: List<Reaction>, active: Boolean): List<Reaction> =
    if (active) {
        if (reactions.any { it.emoji == THUMB_REACTION }) {
            reactions.map {
                if (it.emoji == THUMB_REACTION) it.copy(count = it.count + 1, reactedByMe = true) else it
            }
        } else {
            reactions + Reaction(THUMB_REACTION, 1, true)
        }
    } else {
        reactions.mapNotNull {
            when {
                it.emoji != THUMB_REACTION -> it
                it.count <= 1 -> null
                else -> it.copy(count = it.count - 1, reactedByMe = false)
            }
        }
    }


/**
 * 通用反应增量合并（WS ReactionAdded / ReactionRemoved 用：事件只带 target/emoji/actor，
 * 不带最新计数，客户端按 ±1 合并）。active=true 计数 +1（无则新增），false 计数 -1（归零移除）；
 * mine 用于标记「这条反应是我发的」，与 reactedByMe 语义一致。
 */
fun applyReactionDelta(
    reactions: List<Reaction>,
    emoji: String,
    active: Boolean,
    mine: Boolean,
): List<Reaction> {
    if (emoji.isBlank()) return reactions
    if (active) {
        return if (reactions.any { it.emoji == emoji }) {
            reactions.map {
                if (it.emoji == emoji) it.copy(count = it.count + 1, reactedByMe = it.reactedByMe || mine) else it
            }
        } else {
            reactions + Reaction(emoji = emoji, count = 1, reactedByMe = mine)
        }
    }
    return reactions.mapNotNull {
        when {
            it.emoji != emoji -> it
            it.count <= 1 -> null
            else -> it.copy(count = it.count - 1, reactedByMe = if (mine) false else it.reactedByMe)
        }
    }
}
