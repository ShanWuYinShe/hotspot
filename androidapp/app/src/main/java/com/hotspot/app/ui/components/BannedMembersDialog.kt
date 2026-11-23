package com.hotspot.app.ui.components

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.hotspot.app.data.model.AdminBannedMemberItem
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.TimeFormat

/** 群封禁列表（owner/admin）：展示被封禁成员并支持解除封禁。 */
@Composable
fun BannedMembersDialog(
    members: List<AdminBannedMemberItem>?,
    loading: Boolean,
    error: String?,
    processingUserId: String?,
    onRetry: () -> Unit,
    onUnban: (AdminBannedMemberItem) -> Unit,
    onDismiss: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("封禁列表") },
        text = {
            Column(
                modifier = Modifier
                    .heightIn(max = 420.dp)
                    .verticalScroll(rememberScrollState()),
            ) {
                when {
                    loading && members == null -> Text("正在加载封禁列表…", color = colors.textSecondary)
                    error != null && members == null -> {
                        Text(error, color = colors.danger)
                        Spacer(Modifier.height(Spacing.sm))
                        SystemButton(
                            text = "重试",
                            onClick = onRetry,
                            variant = ButtonVariant.SECONDARY,
                            size = ButtonSize.SM,
                        )
                    }
                    members.isNullOrEmpty() -> Text("暂无被封禁的成员", color = colors.textSecondary)
                    else -> members.forEach { member ->
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(vertical = Spacing.sm2),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Avatar(
                                name = member.user.nickname,
                                imageUrl = member.user.avatarUrl,
                                size = Dimens.avatarSm,
                            )
                            Spacer(Modifier.width(Spacing.sm2))
                            Column(modifier = Modifier.weight(1f)) {
                                Text(
                                    text = member.user.nickname.ifBlank { "匿名用户" },
                                    style = MaterialTheme.typography.bodyLarge,
                                    color = colors.textPrimary,
                                    maxLines = 1,
                                    overflow = TextOverflow.Ellipsis,
                                )
                                Text(
                                    text = TimeFormat.relative(member.bannedAt) +
                                        if (member.reason.isNullOrBlank()) "" else " · " + member.reason,
                                    style = MaterialTheme.typography.bodySmall,
                                    color = colors.textSecondary,
                                    maxLines = 2,
                                    overflow = TextOverflow.Ellipsis,
                                )
                            }
                            Spacer(Modifier.width(Spacing.sm))
                            if (processingUserId == member.user.userId) {
                                InlineLoading()
                            } else {
                                SystemButton(
                                    text = "解除封禁",
                                    onClick = { onUnban(member) },
                                    variant = ButtonVariant.SECONDARY,
                                    size = ButtonSize.SM,
                                )
                            }
                        }
                        HorizontalDivider(color = colors.borderSubtle)
                    }
                }
            }
        },
        confirmButton = {},
        dismissButton = {
            SystemButton(
                text = "关闭",
                onClick = onDismiss,
                variant = ButtonVariant.SECONDARY,
                size = ButtonSize.SM,
            )
        },
    )
}
