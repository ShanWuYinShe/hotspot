package com.hotspot.app.ui.components

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing

/** 群主/管理员编辑群名称与简介（groups/admin/update）；其余群设置本轮不动。 */
@Composable
fun GroupInfoEditorDialog(
    initialName: String,
    initialDescription: String,
    submitting: Boolean,
    error: String?,
    onDismiss: () -> Unit,
    onSubmit: (name: String, description: String) -> Unit,
) {
    val colors = HotSpotTheme.colors
    var name by remember(initialName) { mutableStateOf(initialName) }
    var description by remember(initialDescription) { mutableStateOf(initialDescription) }
    val canSubmit = name.isNotBlank() && !submitting
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("编辑群信息") },
        text = {
            Column {
                SystemInput(
                    value = name,
                    onValueChange = { name = it },
                    label = "群名称",
                    placeholder = "请输入群名称",
                )
                Spacer(Modifier.height(Spacing.md))
                SystemTextArea(
                    value = description,
                    onValueChange = { description = it },
                    label = "群简介",
                    placeholder = "介绍一下这个群…",
                    minHeight = com.hotspot.app.ui.theme.Dimens.inputHeight * 2,
                )
                if (error != null) {
                    Spacer(Modifier.height(Spacing.sm))
                    Text(
                        text = error,
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.danger,
                    )
                }
            }
        },
        confirmButton = {
            SystemButton(
                text = if (submitting) "保存中…" else "保存",
                onClick = { if (canSubmit) onSubmit(name.trim(), description.trim()) },
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
