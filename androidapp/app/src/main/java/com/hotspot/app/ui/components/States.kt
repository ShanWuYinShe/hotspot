package com.hotspot.app.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing

/** 加载态（列表/页面首屏统一 spinner；骨架屏留给信息流骨架卡）。 */
@Composable
fun LoadingState(
    modifier: Modifier = Modifier,
    message: String = "加载中…",
) {
    val colors = HotSpotTheme.colors
    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(vertical = Spacing.xxl),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        CircularProgressIndicator(
            modifier = Modifier.size(Dimens.iconMd),
            color = colors.accent,
            strokeWidth = 2.dp,
        )
        Spacer(Modifier.height(Spacing.md))
        Text(
            text = message,
            style = MaterialTheme.typography.bodySmall,
            color = colors.textSecondary,
        )
    }
}

/** 空态：64 软底图标砖 + 14 semibold 标题 + 12 描述 + 可选动作。 */
@Composable
fun EmptyState(
    title: String,
    modifier: Modifier = Modifier,
    description: String? = null,
    actionLabel: String? = null,
    onAction: (() -> Unit)? = null,
) {
    val colors = HotSpotTheme.colors
    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = Spacing.xl, vertical = Spacing.xxl),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Box(
            modifier = Modifier
                .size(64.dp)
                .background(colors.subtle, AppShapes.card),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                imageVector = Icons.Filled.Refresh,
                contentDescription = null,
                modifier = Modifier.size(Dimens.iconLg),
                tint = colors.textSecondary,
            )
        }
        Spacer(Modifier.height(Spacing.md))
        Text(
            text = title,
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.SemiBold,
            color = colors.textPrimary,
            textAlign = TextAlign.Center,
        )
        if (description != null) {
            Spacer(Modifier.height(Spacing.xs))
            Text(
                text = description,
                style = MaterialTheme.typography.bodyMedium,
                color = colors.textSecondary,
                textAlign = TextAlign.Center,
            )
        }
        if (actionLabel != null && onAction != null) {
            Spacer(Modifier.height(Spacing.lg))
            SystemButton(text = actionLabel, onClick = onAction, variant = ButtonVariant.SECONDARY, size = ButtonSize.SM)
        }
    }
}

/** 分页到底收尾（「没有更多了」）：内容耗尽必须渲染，否则用户无法区分已到底与加载卡住。 */
@Composable
fun ListEnd(
    modifier: Modifier = Modifier,
    text: String = "没有更多了",
) {
    val colors = HotSpotTheme.colors
    Box(
        modifier = modifier
            .fillMaxWidth()
            .padding(vertical = Spacing.lg),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.bodySmall,
            color = colors.textSecondary,
        )
    }
}

/** 分页加载中（更多内容的行内提示，不换骨架屏）。 */
@Composable
fun InlineLoading(
    modifier: Modifier = Modifier,
    text: String = "加载更多…",
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = modifier
            .fillMaxWidth()
            .padding(vertical = Spacing.lg),
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        CircularProgressIndicator(
            modifier = Modifier.size(Dimens.iconXs),
            color = colors.accent,
            strokeWidth = 2.dp,
        )
        Spacer(Modifier.width(Spacing.sm))
        Text(
            text = text,
            style = MaterialTheme.typography.bodySmall,
            color = colors.textSecondary,
        )
    }
}

/** 错误态：图标 + 用户可读错误 + 重试（禁止静默转空态）。 */
@Composable
fun ErrorState(
    message: String,
    onRetry: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = Spacing.xl, vertical = Spacing.xxl),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            modifier = Modifier
                .size(64.dp)
                .background(colors.dangerBg, AppShapes.card),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                imageVector = Icons.Filled.Warning,
                contentDescription = null,
                modifier = Modifier.size(Dimens.iconLg),
                tint = colors.danger,
            )
        }
        Spacer(Modifier.height(Spacing.md))
        Text(
            text = message,
            style = MaterialTheme.typography.bodyMedium,
            color = colors.textSecondary,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(Spacing.lg))
        SystemButton(text = "重试", onClick = onRetry, variant = ButtonVariant.SECONDARY, size = ButtonSize.SM)
    }
}
