package com.hotspot.app.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing

/** 非压图徽章 tone（design-system §17 / ui-components：neutral/accent/solid/danger/warning/online/outline）。 */
enum class BadgeTone { NEUTRAL, ACCENT, SOLID, DANGER, WARNING, ONLINE, OUTLINE }

/** 徽章尺寸严格两档：sm=20 / md=24（Figma 组件库 Badge 24 高，sm 为紧凑档）。 */
enum class BadgeSize(val height: Dp, val horizontalPadding: Dp, val fontSp: Float) {
    SM(20.dp, Spacing.sm, com.hotspot.app.ui.theme.Tokens.FontSize.xxs),
    MD(24.dp, Spacing.md2, com.hotspot.app.ui.theme.Tokens.FontSize.xs),
}

/**
 * 徽章/标签唯一原语：inline-flex 居中 + 胶囊圆角 + 语义 tone。
 * 数字未读另行实现（UnreadBadge），压图徽章走 MediaBadge。
 */
@Composable
fun Badge(
    text: String,
    modifier: Modifier = Modifier,
    tone: BadgeTone = BadgeTone.NEUTRAL,
    size: BadgeSize = BadgeSize.SM,
) {
    val colors = HotSpotTheme.colors
    val (background, foreground) = when (tone) {
        BadgeTone.NEUTRAL -> colors.subtle to colors.textSecondary
        BadgeTone.ACCENT -> colors.accentSoftBg to colors.textAccentOnSoft
        BadgeTone.SOLID -> colors.accent to colors.textOnAccent
        BadgeTone.DANGER -> colors.dangerBg to colors.textDangerOnSoft
        BadgeTone.WARNING -> colors.warningBg to colors.textWarningOnSoft
        BadgeTone.ONLINE -> colors.onlineBg to colors.textOnlineOnSoft
        BadgeTone.OUTLINE -> Color.Transparent to colors.textSecondary
    }
    val shape = RoundedCornerShape(percent = 50)
    Box(
        modifier = modifier
            .height(size.height)
            .background(background, shape)
            .then(if (tone == BadgeTone.OUTLINE) Modifier.border(1.dp, colors.border, shape) else Modifier)
            .padding(horizontal = size.horizontalPadding),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = text,
            style = TextStyle(
                fontSize = size.fontSp.sp,
                lineHeight = size.fontSp.sp,
                fontWeight = FontWeight.Medium,
            ),
            color = foreground,
            maxLines = 1,
        )
    }
}

/** 压图徽章（media-badge）：深色强遮罩 + 反色字，用于图片卡角标（对比度硬约束）。 */
@Composable
fun MediaBadge(
    text: String,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    Box(
        modifier = modifier
            .height(20.dp)
            .background(colors.overlayStrong, AppShapes.pill)
            .padding(horizontal = Spacing.sm),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = text,
            style = TextStyle(
                fontSize = com.hotspot.app.ui.theme.Tokens.FontSize.xxs.sp,
                lineHeight = com.hotspot.app.ui.theme.Tokens.FontSize.xxs.sp,
                fontWeight = FontWeight.Medium,
            ),
            color = colors.contentInverse,
            maxLines = 1,
        )
    }
}
