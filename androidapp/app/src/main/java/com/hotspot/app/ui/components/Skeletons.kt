package com.hotspot.app.ui.components

import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.MotionSpec
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.ui.theme.rememberHotSpotMotion

/**
 * 名片卡墙首屏骨架（2 列白底卡：顶部标签行 + 居中头像 + 两行文本 + 底部 pill）。
 *
 * 与 NearbyUserCard / NearbyGroupCard 同构（padding、头像 56、行距一致），加载前后网格几何不跳动；
 * 行式列表等待仍用 LoadingState，只有「卡墙」场景用本组件。
 * reduced-motion（系统关闭动画）下静态不脉冲。
 */
@Composable
fun CardGridSkeleton(
    columns: Int = 2,
    rows: Int = 3,
    modifier: Modifier = Modifier,
) {
    val motion = rememberHotSpotMotion()
    val pulse = if (motion.reduced) {
        1f
    } else {
        val transition = rememberInfiniteTransition(label = "card-grid-skeleton")
        transition.animateFloat(
            initialValue = 0.45f,
            targetValue = 1f,
            animationSpec = infiniteRepeatable(
                animation = tween(durationMillis = MotionSpec.SKELETON_MS, easing = LinearEasing),
                repeatMode = RepeatMode.Reverse,
            ),
            label = "card-grid-skeleton-alpha",
        ).value
    }
    val count = (columns * rows).coerceAtLeast(1)
    LazyVerticalGrid(
        columns = GridCells.Fixed(columns),
        modifier = modifier.fillMaxSize(),
        userScrollEnabled = false,
        contentPadding = PaddingValues(start = Spacing.lg, end = Spacing.lg, bottom = Spacing.xxl),
        horizontalArrangement = Arrangement.spacedBy(Spacing.md),
        verticalArrangement = Arrangement.spacedBy(Spacing.md),
    ) {
        items(count) { index ->
            CardSkeleton(pulse = pulse, key = index)
        }
    }
}

@Composable
private fun CardSkeleton(pulse: Float, key: Int) {
    val colors = HotSpotTheme.colors
    val blockColor = colors.subtle.copy(alpha = colors.subtle.alpha * pulse)
    Box(
        modifier = Modifier
            .shadow(1.dp, AppShapes.card, ambientColor = colors.shadowDark, spotColor = colors.shadowDark)
            .clip(AppShapes.card)
            .background(colors.island)
            .border(1.dp, colors.border, AppShapes.card)
            .padding(Spacing.md),
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                SkeletonBlock(width = 44.dp, height = 14.dp, shape = AppShapes.pill, color = blockColor)
                Spacer(Modifier.weight(1f))
                SkeletonBlock(width = 32.dp, height = 10.dp, shape = AppShapes.pill, color = blockColor)
            }
            Spacer(Modifier.height(Spacing.md2))
            SkeletonBlock(width = 56.dp, height = 56.dp, shape = CircleShape, color = blockColor)
            Spacer(Modifier.height(Spacing.sm))
            SkeletonBlock(width = 96.dp, height = 16.dp, shape = AppShapes.compact, color = blockColor)
            Spacer(Modifier.height(6.dp))
            SkeletonBlock(width = 64.dp, height = 10.dp, shape = AppShapes.compact, color = blockColor)
            Spacer(Modifier.height(Spacing.md))
            SkeletonBlock(width = 76.dp, height = 28.dp, shape = AppShapes.pill, color = blockColor)
        }
    }
}

@Composable
private fun SkeletonBlock(
    width: Dp,
    height: Dp,
    shape: RoundedCornerShape,
    color: androidx.compose.ui.graphics.Color,
) {
    Box(
        modifier = Modifier
            .size(width = width, height = height)
            .clip(shape)
            .background(color),
    )
}
