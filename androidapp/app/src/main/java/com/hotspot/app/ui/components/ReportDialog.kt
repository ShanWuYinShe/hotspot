package com.hotspot.app.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.ReportCatalog

/**
 * 举报弹窗（用户/评论等共用）：原因 chips + 提交/取消。
 * 提交态由调用方控制（submitting），成功后由调用方关闭并 Toast；本组件不直接发请求。
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun ReportDialog(
    title: String,
    submitting: Boolean,
    onDismiss: () -> Unit,
    onSubmit: (reason: String) -> Unit,
) {
    val colors = HotSpotTheme.colors
    var reason by remember { mutableStateOf<String?>(null) }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(title) },
        text = {
            Column {
                Text(
                    text = "选择举报原因（处理进度可在「设置 → 我的举报」查看）",
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
                Spacer(Modifier.height(Spacing.sm))
                FlowRow(
                    horizontalArrangement = Arrangement.spacedBy(Spacing.sm),
                    verticalArrangement = Arrangement.spacedBy(Spacing.sm),
                ) {
                    ReportCatalog.reasons.forEach { item ->
                        val active = reason == item.key
                        Box(
                            modifier = Modifier
                                .clip(AppShapes.pill)
                                .background(if (active) colors.accentSoftBg else colors.island)
                                .border(1.dp, if (active) colors.accentBorder else colors.borderSubtle, AppShapes.pill)
                                .clickable { reason = item.key }
                                .padding(horizontal = Spacing.md, vertical = Spacing.sm2),
                        ) {
                            Text(
                                text = item.label,
                                style = MaterialTheme.typography.labelSmall,
                                color = if (active) colors.textAccentOnSoft else colors.textSecondary,
                            )
                        }
                    }
                }
            }
        },
        confirmButton = {
            SystemButton(
                text = if (submitting) "提交中…" else "提交",
                onClick = { reason?.let(onSubmit) },
                variant = ButtonVariant.PRIMARY,
                size = ButtonSize.SM,
                enabled = reason != null && !submitting,
            )
        },
        dismissButton = {
            SystemButton(
                text = "取消",
                onClick = onDismiss,
                variant = ButtonVariant.SECONDARY,
                size = ButtonSize.SM,
            )
        },
    )
}
