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
import com.hotspot.app.data.model.GroupSentInviteItem
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.TimeFormat

/** 邀请状态文案（spec ApplicationStatus：pending/accepted/rejected/revoked）。 */
private fun inviteStatusLabel(status: String): String = when (status) {
    "pending" -> "待接受"
    "accepted" -> "已加入"
    "rejected" -> "已拒绝"
    "revoked" -> "已撤销"
    else -> status
}

/**
 * 我发出的群邀请（owner/admin）：展示全部历史状态，pending 可撤销。
 * 排序由服务端按 created_at DESC；撤销成功由调用方刷新列表。
 */
@Composable
fun GroupSentInvitesDialog(
    invites: List<GroupSentInviteItem>?,
    loading: Boolean,
    error: String?,
    revokingId: String?,
    onRetry: () -> Unit,
    onRevoke: (GroupSentInviteItem) -> Unit,
    onDismiss: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("邀请记录") },
        text = {
            Column(
                modifier = Modifier
                    .heightIn(max = 420.dp)
                    .verticalScroll(rememberScrollState()),
            ) {
                when {
                    loading && invites == null -> Text("正在加载邀请记录…", color = colors.textSecondary)
                    error != null && invites == null -> {
                        Text(error, color = colors.danger)
                        Spacer(Modifier.height(Spacing.sm))
                        SystemButton(
                            text = "重试",
                            onClick = onRetry,
                            variant = ButtonVariant.SECONDARY,
                            size = ButtonSize.SM,
                        )
                    }
                    invites.isNullOrEmpty() -> Text("还没有发出过邀请", color = colors.textSecondary)
                    else -> invites.forEach { invite ->
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(vertical = Spacing.sm2),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Avatar(
                                name = invite.invitee.nickname,
                                imageUrl = invite.invitee.avatarUrl,
                                size = Dimens.avatarSm,
                            )
                            Spacer(Modifier.width(Spacing.sm2))
                            Column(modifier = Modifier.weight(1f)) {
                                Row(verticalAlignment = Alignment.CenterVertically) {
                                    Text(
                                        text = invite.invitee.nickname.ifBlank { "匿名用户" },
                                        style = MaterialTheme.typography.bodyLarge,
                                        color = colors.textPrimary,
                                        maxLines = 1,
                                        overflow = TextOverflow.Ellipsis,
                                        modifier = Modifier.weight(1f, fill = false),
                                    )
                                    Spacer(Modifier.width(Spacing.xs))
                                    Text(
                                        text = inviteStatusLabel(invite.status),
                                        style = MaterialTheme.typography.labelSmall,
                                        color = if (invite.status == "pending") colors.accent else colors.textSecondary,
                                    )
                                }
                                Text(
                                    text = TimeFormat.relative(invite.createdAt) +
                                        " · 邀请人 " + invite.inviter.nickname.ifBlank { "管理员" } +
                                        if (invite.message.isNullOrBlank()) "" else " · " + invite.message,
                                    style = MaterialTheme.typography.bodySmall,
                                    color = colors.textSecondary,
                                    maxLines = 2,
                                    overflow = TextOverflow.Ellipsis,
                                )
                            }
                            Spacer(Modifier.width(Spacing.sm))
                            if (invite.status == "pending") {
                                if (revokingId == invite.inviteId) {
                                    InlineLoading()
                                } else {
                                    SystemButton(
                                        text = "撤销",
                                        onClick = { onRevoke(invite) },
                                        variant = ButtonVariant.SECONDARY,
                                        size = ButtonSize.SM,
                                        enabled = revokingId == null,
                                    )
                                }
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
