package com.hotspot.app.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing

/** 氛围标签预设（发布与编辑瞬间共用，对齐 Web CreateSheet 的标签目录）。 */
val MoodTags = listOf("松弛", "漫游", "咖啡", "展览", "自然", "夜色")

/**
 * 氛围标签多选 chips。extraTags 用于编辑时把瞬间已有的自定义标签一并展示，
 * 避免只显示预设标签导致用户无法保留/取消历史标签。
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun MoodChips(
    selected: List<String>,
    onToggle: (String) -> Unit,
    extraTags: List<String> = emptyList(),
) {
    val colors = HotSpotTheme.colors
    FlowRow(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(Spacing.sm),
        verticalArrangement = Arrangement.spacedBy(Spacing.sm),
    ) {
        (MoodTags + extraTags).distinct().forEach { tag ->
            val isSelected = selected.contains(tag)
            Box(
                modifier = Modifier
                    .clip(AppShapes.pill)
                    .background(if (isSelected) colors.accentSoftBg else colors.island)
                    .border(1.dp, if (isSelected) colors.accentBorder else colors.borderSubtle, AppShapes.pill)
                    .clickable { onToggle(tag) }
                    .padding(horizontal = Spacing.md, vertical = Spacing.sm2),
            ) {
                Text(
                    text = "#" + tag,
                    style = MaterialTheme.typography.labelSmall,
                    fontWeight = if (isSelected) FontWeight.Medium else FontWeight.Normal,
                    color = if (isSelected) colors.textAccentOnSoft else colors.textSecondary,
                )
            }
        }
    }
}
