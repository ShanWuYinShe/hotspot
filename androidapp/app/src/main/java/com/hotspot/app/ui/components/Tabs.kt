package com.hotspot.app.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Check
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing

/**
 * 分段控制器（segmented）：accent 实底胶囊 + ✓ 图标 + semibold（design-system §18：
 * 选中态不得仅依赖颜色，灰度下仍可辨）。
 */
@Composable
fun SegmentedTabs(
    options: List<String>,
    selectedIndex: Int,
    onSelected: (Int) -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    val trackShape = RoundedCornerShape(percent = 50)
    Row(
        modifier = modifier
            .fillMaxWidth()
            .height(Dimens.tabBar)
            .clip(trackShape)
            .background(colors.subtle, trackShape)
            .padding(2.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        options.forEachIndexed { index, label ->
            val selected = index == selectedIndex
            Box(
                modifier = Modifier
                    .weight(1f)
                    .fillMaxHeight()
                    .clip(trackShape)
                    .background(if (selected) colors.accent else androidx.compose.ui.graphics.Color.Transparent, trackShape)
                    .clickable(
                        interactionSource = remember { MutableInteractionSource() },
                        indication = null,
                    ) { onSelected(index) }
                    .hotSpotFocusRing(cornerRadius = 17.dp),
                contentAlignment = Alignment.Center,
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    if (selected) {
                        Icon(
                            imageVector = Icons.Filled.Check,
                            contentDescription = null,
                            modifier = Modifier.size(12.dp),
                            tint = colors.textOnAccent,
                        )
                        Spacer(Modifier.width(Spacing.xs))
                    }
                    Text(
                        text = label,
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Normal,
                        color = if (selected) colors.textOnAccent else colors.textSecondary,
                        maxLines = 1,
                    )
                }
            }
        }
    }
}

/** 筛选/分类胶囊（pills）：选中 accent-soft + accent 字 + semibold；可横滚。 */
@Composable
fun TabPills(
    options: List<String>,
    selectedIndex: Int,
    onSelected: (Int) -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    val shape = RoundedCornerShape(percent = 50)
    LazyRow(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(Spacing.sm),
        contentPadding = androidx.compose.foundation.layout.PaddingValues(horizontal = 2.dp),
    ) {
        itemsIndexed(options) { index, label ->
            val selected = index == selectedIndex
            Box(
                modifier = Modifier
                    .height(Dimens.tabBar)
                    .widthIn(min = 56.dp)
                    .clip(shape)
                    .background(if (selected) colors.accentSoftBg else colors.subtle, shape)
                    .clickable(
                        interactionSource = remember { MutableInteractionSource() },
                        indication = null,
                    ) { onSelected(index) }
                    .hotSpotFocusRing(cornerRadius = 17.dp)
                    .padding(horizontal = Spacing.md3),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    text = label,
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Normal,
                    color = if (selected) colors.textAccentOnSoft else colors.textSecondary,
                    maxLines = 1,
                )
            }
        }
    }
}
