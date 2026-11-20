package com.hotspot.app.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing

/**
 * 页面头部（移动端）：返回 + 标题 + 右侧动作。
 * - 有返回钮的子页标题让出返回位（design-system §21.10 已登记例外）
 * - 页头不带横向 padding（横向留白由页面统一 gutter 提供）
 */
@Composable
fun PageHeader(
    title: String,
    modifier: Modifier = Modifier,
    onBack: (() -> Unit)? = null,
    subtitle: String? = null,
    actions: @Composable RowScope.() -> Unit = {},
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = modifier
            .fillMaxWidth()
            .height(Dimens.backButton + Spacing.md2)
            .padding(horizontal = Spacing.lg),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (onBack != null) {
            // 返回钮：44 热区 + 42 圆钮（surface 底 + 轻投影，design-system §27.2「返回 42 圆钮」；
            // 设计稿 87:3257 / 87:3219 均为白色圆形返回钮）
            Box(
                modifier = Modifier
                    .size(Dimens.minTouchTarget)
                    .clip(CircleShape)
                    .clickable(
                        interactionSource = remember { MutableInteractionSource() },
                        indication = null,
                        onClick = onBack,
                    )
                    .hotSpotFocusRing(cornerRadius = 22.dp),
                contentAlignment = Alignment.Center,
            ) {
                Box(
                    modifier = Modifier
                        .size(Dimens.backButton)
                        .shadow(
                            elevation = 1.dp,
                            shape = CircleShape,
                            ambientColor = colors.shadowDark,
                            spotColor = colors.shadowDark,
                        )
                        .clip(CircleShape)
                        .background(colors.surface),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        imageVector = Icons.Filled.ArrowBack,
                        contentDescription = "返回",
                        modifier = Modifier.size(Dimens.iconSm),
                        tint = colors.textPrimary,
                    )
                }
            }
            Spacer(Modifier.width(Spacing.xs))
        }
        Column(Modifier.weight(1f)) {
            Text(
                text = title,
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.SemiBold,
                color = colors.textPrimary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            if (subtitle != null) {
                Text(
                    text = subtitle,
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.textSecondary,
                    maxLines = 1,
                )
            }
        }
        Row(verticalAlignment = Alignment.CenterVertically, content = actions)
    }
}

/** 页头图标动作钮（36 视觉 + 44 热区，ghost 风格；页头右侧纯图标动作统一入口）。 */
@Composable
fun HeaderIconButton(
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    contentDescription: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    badgeCount: Int = 0,
) {
    val colors = HotSpotTheme.colors
    Box(
        modifier = modifier
            .size(Dimens.minTouchTarget)
            .clip(CircleShape)
            .clickable(
                interactionSource = remember { MutableInteractionSource() },
                indication = null,
                onClick = onClick,
            )
            .hotSpotFocusRing(cornerRadius = 22.dp),
        contentAlignment = Alignment.Center,
    ) {
        Box(
            modifier = Modifier
                .size(Dimens.iconButton)
                .clip(AppShapes.pill)
                .background(colors.subtle),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                imageVector = icon,
                contentDescription = contentDescription,
                modifier = Modifier.size(Dimens.iconSm),
                tint = colors.textPrimary,
            )
        }
        if (badgeCount > 0) {
            Box(
                modifier = Modifier
                    .align(Alignment.TopEnd)
                    .padding(top = Spacing.xs, end = Spacing.xs)
                    .size(8.dp)
                    .background(colors.danger, CircleShape),
            )
        }
    }
}
