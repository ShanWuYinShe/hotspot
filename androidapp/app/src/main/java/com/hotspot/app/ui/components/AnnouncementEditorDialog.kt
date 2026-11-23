package com.hotspot.app.ui.components

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing

/**
 * 群公告发布/编辑弹窗（owner/admin）：标题 + 正文 + 置顶开关。
 * 提交态/错误由调用方控制；成功后由调用方关闭并刷新列表。
 */
@Composable
fun AnnouncementEditorDialog(
    title: String,
    initialTitle: String,
    initialContent: String,
    initialPinned: Boolean,
    submitting: Boolean,
    error: String?,
    onDismiss: () -> Unit,
    onSubmit: (title: String, content: String, isPinned: Boolean) -> Unit,
) {
    val colors = HotSpotTheme.colors
    var titleInput by remember(initialTitle) { mutableStateOf(initialTitle) }
    var contentInput by remember(initialContent) { mutableStateOf(initialContent) }
    var pinned by remember(initialPinned) { mutableStateOf(initialPinned) }
    val canSubmit = titleInput.isNotBlank() && contentInput.isNotBlank() && !submitting
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(title) },
        text = {
            Column {
                SystemInput(
                    value = titleInput,
                    onValueChange = { titleInput = it },
                    label = "标题",
                    placeholder = "一句话说明公告主题",
                )
                Spacer(Modifier.height(Spacing.md))
                SystemTextArea(
                    value = contentInput,
                    onValueChange = { contentInput = it },
                    label = "正文",
                    placeholder = "写清时间、地点、注意事项…",
                )
                Spacer(Modifier.height(Spacing.md))
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        text = "置顶公告",
                        style = MaterialTheme.typography.bodyLarge,
                        color = colors.textPrimary,
                        modifier = Modifier.weight(1f),
                    )
                    Switch(
                        checked = pinned,
                        onCheckedChange = { pinned = it },
                        colors = SwitchDefaults.colors(
                            checkedThumbColor = colors.textOnAccent,
                            checkedTrackColor = colors.accent,
                        ),
                    )
                }
                if (error != null) {
                    Spacer(Modifier.height(Spacing.sm))
                    Text(
                        text = error,
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.danger,
                        modifier = Modifier.padding(top = Spacing.xs),
                    )
                }
            }
        },
        confirmButton = {
            SystemButton(
                text = if (submitting) "提交中…" else "提交",
                onClick = { if (canSubmit) onSubmit(titleInput.trim(), contentInput.trim(), pinned) },
                variant = ButtonVariant.PRIMARY,
                size = ButtonSize.SM,
                enabled = canSubmit,
                loading = submitting,
            )
        },
        dismissButton = {
            SystemButton(
                text = "取消",
                onClick = onDismiss,
                variant = ButtonVariant.SECONDARY,
                size = ButtonSize.SM,
                enabled = !submitting,
            )
        },
    )
}
