package com.hotspot.app.ui.components

import android.widget.Toast
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import com.hotspot.app.data.media.AvatarUploader
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import kotlinx.coroutines.launch

/** 群主/管理员编辑群名称与简介（groups/admin/update）；其余群设置本轮不动。 */
@Composable
fun GroupInfoEditorDialog(
    initialName: String,
    initialDescription: String,
    initialAvatarUrl: String?,
    submitting: Boolean,
    error: String?,
    onDismiss: () -> Unit,
    onSubmit: (name: String, description: String, avatarFileId: String?) -> Unit,
) {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var name by remember(initialName) { mutableStateOf(initialName) }
    var description by remember(initialDescription) { mutableStateOf(initialDescription) }
    var avatarFileId by remember { mutableStateOf<String?>(null) }
    var avatarUploading by remember { mutableStateOf(false) }
    var avatarError by remember { mutableStateOf<String?>(null) }
    val avatarPicker = rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()) { uri ->
        if (uri != null && !avatarUploading) {
            avatarUploading = true
            avatarError = null
            scope.launch {
                try {
                    AvatarUploader.upload(context, uri, category = "group_avatar")
                        .onSuccess {
                            avatarFileId = it.fileId
                            Toast.makeText(context, "群头像已上传，保存后生效", Toast.LENGTH_SHORT).show()
                        }
                        .onFailure { avatarError = it.message ?: "群头像上传失败，请重试" }
                } finally {
                    avatarUploading = false
                }
            }
        }
    }
    val canSubmit = name.isNotBlank() && !submitting
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("编辑群信息") },
        text = {
            Column {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Box(contentAlignment = Alignment.Center) {
                        Avatar(
                            name = name,
                            imageUrl = initialAvatarUrl,
                            size = 56.dp,
                        )
                        if (avatarUploading) {
                            CircularProgressIndicator(
                                modifier = Modifier.size(24.dp),
                                strokeWidth = 2.dp,
                                color = colors.accent,
                            )
                        }
                    }
                    Spacer(Modifier.width(Spacing.md))
                    Column {
                        SystemButton(
                            text = if (avatarUploading) "上传中…" else "更换群头像",
                            onClick = {
                                avatarPicker.launch(
                                    PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly),
                                )
                            },
                            variant = ButtonVariant.SECONDARY,
                            size = ButtonSize.SM,
                            enabled = !avatarUploading,
                        )
                        Spacer(Modifier.height(Spacing.xs))
                        Text(
                            text = if (avatarFileId != null) "已选择新头像，保存后生效" else "JPG/PNG/WebP，最大 5MB",
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.textSecondary,
                        )
                    }
                }
                if (avatarError != null) {
                    Spacer(Modifier.height(Spacing.xs))
                    Text(
                        text = avatarError ?: "",
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.danger,
                    )
                }
                Spacer(Modifier.height(Spacing.md))
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
                onClick = { if (canSubmit) onSubmit(name.trim(), description.trim(), avatarFileId) },
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
