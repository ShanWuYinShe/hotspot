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
