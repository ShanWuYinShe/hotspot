package com.hotspot.app.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp
import com.hotspot.app.data.model.Reaction
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing

/** 消息反应胶囊：展示非零表情计数（最多 4 个）。 */
@Composable
fun MessageReactionBadge(reactions: List<Reaction>, modifier: Modifier = Modifier) {
    val visible = reactions.filter { it.count > 0 }.take(4)
    if (visible.isEmpty()) return
    val colors = HotSpotTheme.colors
    Row(
        modifier = modifier.padding(top = 2.dp, start = Spacing.xs, end = Spacing.xs),
        horizontalArrangement = Arrangement.spacedBy(Spacing.xs),
    ) {
        visible.forEach { reaction ->
            Text(
                text = reaction.emoji + " " + reaction.count,
                style = MaterialTheme.typography.labelSmall,
                color = if (reaction.reactedByMe) colors.accent else colors.textSecondary,
                modifier = Modifier
                    .clip(RoundedCornerShape(10.dp))
                    .background(colors.subtle)
                    .padding(horizontal = Spacing.sm, vertical = 2.dp),
            )
        }
    }
}
