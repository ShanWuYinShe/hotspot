package com.hotspot.app.ui.components

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.RadioButton
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
import androidx.compose.ui.text.input.PasswordVisualTransformation
import com.hotspot.app.data.model.GroupSettings
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing

/** 群设置（owner/admin）：搜索可见性、成员可搜索、入群审批、人数上限、加入方式与群密码。 */
@Composable
fun GroupSettingsDialog(
    settings: GroupSettings?,
    loading: Boolean,
    loadError: String?,
    submitting: Boolean,
    saveError: String?,
    onRetry: () -> Unit,
    onDismiss: () -> Unit,
    onSubmit: (
        isSearchable: Boolean,
        membersSearchable: Boolean,
        requiresApproval: Boolean,
        maxMembers: Int,
        joinMethod: String,
        password: String?,
        clearPassword: Boolean,
    ) -> Unit,
) {
    val colors = HotSpotTheme.colors
    var isSearchable by remember(settings) { mutableStateOf(settings?.isSearchable ?: true) }
    var membersSearchable by remember(settings) { mutableStateOf(settings?.membersSearchable ?: true) }
    var requiresApproval by remember(settings) { mutableStateOf(settings?.requiresApproval ?: false) }
    var maxMembersText by remember(settings) { mutableStateOf((settings?.maxMembers ?: 200).toString()) }
    var joinMethod by remember(settings) { mutableStateOf(settings?.joinMethod ?: "direct") }
    var passwordDraft by remember(settings) { mutableStateOf("") }
    var clearPassword by remember(settings) { mutableStateOf(false) }
    val maxMembers = maxMembersText.toIntOrNull()
    val passwordMissing = joinMethod == "password" && settings?.hasPassword != true && !clearPassword && passwordDraft.isBlank()
    val passwordTooShort = passwordDraft.isNotBlank() && passwordDraft.length < 8
    val canSubmit = settings != null && !submitting && maxMembers != null && maxMembers in 2..2000 &&
        !passwordMissing && !passwordTooShort
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
                        text = "加入方式",
                        style = MaterialTheme.typography.bodyLarge,
                        color = colors.textPrimary,
                    )
                    JoinMethodRow(
                        label = "公开加入",
                        description = "任何人都可以直接加入",
                        selected = joinMethod == "direct",
                        enabled = !submitting,
                        onClick = {
                            joinMethod = "direct"
                            clearPassword = false
                        },
                    )
                    JoinMethodRow(
                        label = "需要密码",
                        description = if (settings.hasPassword) "已设置群密码，可重设" else "加入时需要输入群密码",
                        selected = joinMethod == "password",
                        enabled = !submitting,
                        onClick = { joinMethod = "password" },
                    )
                    JoinMethodRow(
                        label = "仅限邀请",
                        description = "只有收到邀请的用户可以加入",
                        selected = joinMethod == "invite_only",
                        enabled = !submitting,
                        onClick = {
                            joinMethod = "invite_only"
                            clearPassword = false
                        },
                    )
                    if (joinMethod == "password") {
                        Spacer(Modifier.height(Spacing.sm))
                        SystemInput(
                            value = passwordDraft,
                            onValueChange = { passwordDraft = it },
                            label = if (settings.hasPassword) "新群密码（8-32 位，留空保持原密码）" else "群密码（8-32 位）",
                            visualTransformation = PasswordVisualTransformation(),
                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
                            enabled = !submitting && !clearPassword,
                            errorText = when {
                                passwordMissing -> "需要设置群密码"
                                passwordTooShort -> "密码至少 8 位"
                                else -> null
                            },
                        )
                        if (settings.hasPassword) {
                            SettingSwitchRow(
                                label = "清除群密码",
                                checked = clearPassword,
                                enabled = !submitting,
                                onChange = {
                                    clearPassword = it
                                    if (it) {
                                        passwordDraft = ""
                                        joinMethod = "direct"
                                    }
                                },
                            )
                        }
                    }
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
                            onSubmit(
                                isSearchable,
                                membersSearchable,
                                requiresApproval,
                                maxMembers ?: 200,
                                joinMethod,
                                passwordDraft.ifBlank { null },
                                clearPassword,
                            )
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

@Composable
private fun JoinMethodRow(
    label: String,
    description: String,
    selected: Boolean,
    enabled: Boolean,
    onClick: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(enabled = enabled, onClick = onClick)
            .padding(vertical = Spacing.xs),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        RadioButton(selected = selected, onClick = if (enabled) onClick else null, enabled = enabled)
        Spacer(Modifier.width(Spacing.xs))
        Column(Modifier.weight(1f)) {
            Text(label, style = MaterialTheme.typography.bodyLarge, color = colors.textPrimary)
            Text(description, style = MaterialTheme.typography.labelSmall, color = colors.textSecondary)
        }
    }
}
