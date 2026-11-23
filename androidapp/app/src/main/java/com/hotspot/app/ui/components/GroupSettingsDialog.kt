package com.hotspot.app.ui.components

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
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
import androidx.compose.ui.text.input.KeyboardType
import com.hotspot.app.data.model.GroupSettings
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing

/** 群设置（owner/admin）：搜索可见性、成员可搜索、入群审批、人数上限；加入方式/密码只读展示。 */
@Composable
fun GroupSettingsDialog(
    settings: GroupSettings?,
    loading: Boolean,
    loadError: String?,
    submitting: Boolean,
    saveError: String?,
    onRetry: () -> Unit,
    onDismiss: () -> Unit,
    onSubmit: (isSearchable: Boolean, membersSearchable: Boolean, requiresApproval: Boolean, maxMembers: Int) -> Unit,
) {
    val colors = HotSpotTheme.colors
    var isSearchable by remember(settings) { mutableStateOf(settings?.isSearchable ?: true) }
    var membersSearchable by remember(settings) { mutableStateOf(settings?.membersSearchable ?: true) }
    var requiresApproval by remember(settings) { mutableStateOf(settings?.requiresApproval ?: false) }
    var maxMembersText by remember(settings) { mutableStateOf((settings?.maxMembers ?: 200).toString()) }
    val maxMembers = maxMembersText.toIntOrNull()
    val canSubmit = settings != null && !submitting && maxMembers != null && maxMembers in 2..2000
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("群设置") },
        text = {
            if (loading) {
                Text("正在加载群设置…", color = colors.textSecondary)
            } else if (loadError != null) {
                Column {
                    Text(loadError, color = colors.danger)
                    Spacer(Modifier.height(Spacing.sm))
                    SystemButton(
                        text = "重试",
                        onClick = onRetry,
                        variant = ButtonVariant.SECONDARY,
                        size = ButtonSize.SM,
                    )
                }
            } else if (settings != null) {
                Column(modifier = Modifier.verticalScroll(rememberScrollState())) {
                    SettingSwitchRow(
                        label = "允许搜索到本群",
                        checked = isSearchable,
                        enabled = !submitting,
                        onChange = { isSearchable = it },
                    )
                    SettingSwitchRow(
                        label = "允许查看成员列表",
                        checked = membersSearchable,
                        enabled = !submitting,
                        onChange = { membersSearchable = it },
                    )
                    SettingSwitchRow(
                        label = "加入需要审批",
                        checked = requiresApproval,
                        enabled = !submitting,
                        onChange = { requiresApproval = it },
                    )
                    Spacer(Modifier.height(Spacing.sm))
                    SystemInput(
                        value = maxMembersText,
                        onValueChange = { maxMembersText = it.filter(Char::isDigit) },
                        label = "群成员上限（2-2000）",
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                        errorText = if (maxMembersText.isNotBlank() && (maxMembers == null || maxMembers !in 2..2000)) {
                            "请输入 2-2000 的整数"
                        } else {
                            null
                        },
                    )
                    Spacer(Modifier.height(Spacing.sm))
                    Text(
                        text = "加入方式：" + joinMethodLabel(settings.joinMethod) + if (settings.hasPassword) "（已设置密码）" else "",
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.textSecondary,
                    )
                    if (saveError != null) {
                        Spacer(Modifier.height(Spacing.sm))
                        Text(saveError, style = MaterialTheme.typography.bodySmall, color = colors.danger)
                    }
                }
            }
        },
        confirmButton = {
            if (settings != null && !loading && loadError == null) {
                SystemButton(
                    text = if (submitting) "保存中…" else "保存",
                    onClick = {
                        if (canSubmit) {
                            onSubmit(isSearchable, membersSearchable, requiresApproval, maxMembers ?: 200)
                        }
                    },
                    variant = ButtonVariant.PRIMARY,
                    size = ButtonSize.SM,
                    enabled = canSubmit,
                    loading = submitting,
                )
            }
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

@Composable
private fun SettingSwitchRow(
    label: String,
    checked: Boolean,
    enabled: Boolean,
    onChange: (Boolean) -> Unit,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = Spacing.xs),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodyLarge,
            color = colors.textPrimary,
            modifier = Modifier.weight(1f),
        )
        Switch(
            checked = checked,
            onCheckedChange = onChange,
            enabled = enabled,
            colors = SwitchDefaults.colors(
                checkedThumbColor = colors.textOnAccent,
                checkedTrackColor = colors.accent,
            ),
        )
    }
}

private fun joinMethodLabel(joinMethod: String): String = when (joinMethod) {
    "password" -> "需要密码"
    "invite_only" -> "仅限邀请"
    else -> "公开加入"
}
