package com.hotspot.app.ui.components

import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.background
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing

/** 录音中的输入栏：红点 + 计时 + 取消/发送；发送中显示 loading 并禁用操作。 */
@Composable
fun AudioRecorderBar(
    seconds: Int,
    sending: Boolean,
    onCancel: () -> Unit,
    onSend: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(colors.dangerBg)
            .padding(horizontal = Spacing.md, vertical = Spacing.sm),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Spacer(
            Modifier
                .size(10.dp)
                .background(colors.danger, CircleShape),
        )
        Spacer(Modifier.width(Spacing.sm))
        Text(
            text = String.format("%02d:%02d", seconds / 60, seconds % 60),
            style = MaterialTheme.typography.bodyLarge,
            color = colors.danger,
        )
        Spacer(Modifier.width(Spacing.sm))
        Text(
            text = if (sending) "发送中…" else "录音中",
            style = MaterialTheme.typography.bodySmall,
            color = colors.textSecondary,
        )
        Spacer(Modifier.weight(1f))
        SystemButton(
            text = "取消",
            onClick = onCancel,
            variant = ButtonVariant.SECONDARY,
            size = ButtonSize.SM,
            enabled = !sending,
        )
        Spacer(Modifier.width(Spacing.xs))
        SystemButton(
            text = "发送",
            onClick = onSend,
            variant = ButtonVariant.PRIMARY,
            size = ButtonSize.SM,
            enabled = !sending,
            loading = sending,
        )
    }
}


/** 按住说话时的状态条：计时 + 「松开发送 · 上滑取消」；上滑后切换为「松开取消」。 */
@Composable
fun AudioHoldHintBar(
    seconds: Int,
    slideCancel: Boolean,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(if (slideCancel) colors.dangerBg else colors.island)
            .padding(horizontal = Spacing.md, vertical = Spacing.sm),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Spacer(
            Modifier
                .size(10.dp)
                .background(if (slideCancel) colors.danger else colors.accent, CircleShape),
        )
        Spacer(Modifier.width(Spacing.sm))
        Text(
            text = String.format("%02d:%02d", seconds / 60, seconds % 60),
            style = MaterialTheme.typography.bodyLarge,
            color = if (slideCancel) colors.danger else colors.textPrimary,
        )
        Spacer(Modifier.width(Spacing.sm))
        Text(
            text = if (slideCancel) "松开取消" else "松开发送 · 上滑取消",
            style = MaterialTheme.typography.bodySmall,
            color = if (slideCancel) colors.danger else colors.textSecondary,
        )
    }
}

