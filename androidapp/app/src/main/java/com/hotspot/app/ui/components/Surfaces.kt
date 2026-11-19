package com.hotspot.app.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.LocalContentColor
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing

/** 内容软岛（design-system §19）：surface-island + 轻投影 + 圆角 20，无 1px 描边。 */
@Composable
fun SystemCard(
    modifier: Modifier = Modifier,
    shape: Shape = AppShapes.card,
    contentPadding: PaddingValues = PaddingValues(Spacing.lg),
    content: @Composable ColumnScope.() -> Unit,
) {
    val colors = HotSpotTheme.colors
    Column(
        modifier = modifier
            .fillMaxWidth()
            .shadow(elevation = 1.dp, shape = shape, ambientColor = colors.shadowDark, spotColor = colors.shadowDark)
            .clip(shape)
            .background(colors.island)
            .padding(contentPadding),
        content = content,
    )
}

/** 列表分组容器（island 语言）：内部行间距 2dp、容器内边 8dp（design-system §19.1）。 */
@Composable
fun ListGroup(
    modifier: Modifier = Modifier,
    content: @Composable ColumnScope.() -> Unit,
) {
    val colors = HotSpotTheme.colors
    Column(
        modifier = modifier
            .fillMaxWidth()
            .shadow(
                elevation = 1.dp,
                shape = AppShapes.card,
                ambientColor = colors.shadowDark,
                spotColor = colors.shadowDark,
            )
            .clip(AppShapes.card)
            .background(colors.island)
            .padding(Spacing.sm),
        verticalArrangement = Arrangement.spacedBy(Spacing.half),
        content = content,
    )
}

/**
 * 列表行（唯一行交互原语）：行透明、按压填充 interaction-hover、圆角 12、最小高 64。
 * 行内自带 leading / title / subtitle / trailing 编排；点击由整行承载。
 */
@Composable
fun ListRow(
    title: String,
    modifier: Modifier = Modifier,
    subtitle: String? = null,
    leading: (@Composable () -> Unit)? = null,
    trailing: (@Composable () -> Unit)? = null,
    onClick: (() -> Unit)? = null,
    minHeight: androidx.compose.ui.unit.Dp = Dimens.listRow,
) {
    val colors = HotSpotTheme.colors
    val interaction = remember { MutableInteractionSource() }
    val pressed by interaction.collectIsPressedAsState()
    val rowShape = RoundedCornerShape(com.hotspot.app.ui.theme.Tokens.Radius.sm.dp)
    val clickableModifier = if (onClick != null) {
        Modifier.clickable(
            interactionSource = interaction,
            indication = null,
            onClick = onClick,
        )
    } else {
        Modifier
    }
    Row(
        modifier = modifier
            .fillMaxWidth()
            .heightIn(min = minHeight)
            .clip(rowShape)
            .background(if (pressed) colors.interactionHover else androidx.compose.ui.graphics.Color.Transparent)
            .then(clickableModifier)
            .then(if (onClick != null) Modifier.hotSpotFocusRing(cornerRadius = 12.dp) else Modifier)
            .padding(horizontal = Spacing.md, vertical = Spacing.sm2),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (leading != null) {
            leading()
            Spacer(Modifier.width(Spacing.md))
        }
        Column(Modifier.weight(1f)) {
            Text(
                text = title,
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Medium,
                color = colors.textPrimary,
                maxLines = 1,
            )
            if (subtitle != null) {
                Spacer(Modifier.heightIn(min = 2.dp))
                Text(
                    text = subtitle,
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.textSecondary,
                    maxLines = 1,
                )
            }
        }
        if (trailing != null) {
            Spacer(Modifier.width(Spacing.sm))
            CompositionLocalProvider(LocalContentColor provides colors.textSecondary) {
                trailing()
            }
        }
    }
}

/** 分区标题：14 semibold + 可选右侧文字动作（design-system §21.6 的轻量实现）。 */
@Composable
fun SectionTitle(
    title: String,
    modifier: Modifier = Modifier,
    action: (@Composable () -> Unit)? = null,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = title,
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.SemiBold,
            color = colors.textPrimary,
            modifier = Modifier.weight(1f),
        )
        action?.invoke()
    }
}

/** 发丝线（内容分隔，不是页面/卡片边界；design-system §19.3 允许保留的功能线）。 */
@Composable
fun SoftDivider(modifier: Modifier = Modifier, color: androidx.compose.ui.graphics.Color? = null) {
    val colors = HotSpotTheme.colors
    Box(
        modifier
            .fillMaxWidth()
            .padding(vertical = Spacing.sm)
            .heightIn(min = 1.dp)
            .background(color ?: colors.borderSubtle),
    )
}
