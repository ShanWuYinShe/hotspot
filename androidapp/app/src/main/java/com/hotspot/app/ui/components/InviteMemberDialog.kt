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
import com.hotspot.app.data.model.UserItem
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing

/** 邀请成员（owner/admin）：按昵称/账号搜索用户并发送邀请。 */
@Composable
fun InviteMemberDialog(
    query: String,
    onQueryChange: (String) -> Unit,
    results: List<UserItem>,
    searching: Boolean,
    error: String?,
    invitedIds: Set<String>,
    invitingId: String?,
    onInvite: (UserItem) -> Unit,
    onDismiss: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("邀请成员") },
        text = {
            Column(
                modifier = Modifier
                    .heightIn(max = 440.dp)
                    .verticalScroll(rememberScrollState()),
            ) {
                SearchInput(
                    value = query,
                    onValueChange = onQueryChange,
                    placeholder = "搜索昵称或账号",
                )
                Spacer(Modifier.height(Spacing.sm))
                when {
                    searching -> Text("正在搜索…", color = colors.textSecondary)
                    error != null -> Text(error, color = colors.danger)
                    query.isBlank() -> Text("输入昵称或账号查找用户", color = colors.textSecondary)
                    results.isEmpty() -> Text("没有找到匹配的用户", color = colors.textSecondary)
                    else -> results.forEach { item ->
                        val userId = item.user.userId
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(vertical = Spacing.sm2),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Avatar(
                                name = item.user.nickname,
                                imageUrl = item.user.avatarUrl,
                                size = Dimens.avatarSm,
                            )
                            Spacer(Modifier.width(Spacing.sm2))
                            Column(modifier = Modifier.weight(1f)) {
                                Text(
                                    text = item.user.nickname.ifBlank { "匿名用户" },
                                    style = MaterialTheme.typography.bodyLarge,
                                    color = colors.textPrimary,
                                    maxLines = 1,
                                    overflow = TextOverflow.Ellipsis,
                                )
                                if (!item.bio.isNullOrBlank()) {
                                    Text(
                                        text = item.bio,
                                        style = MaterialTheme.typography.bodySmall,
                                        color = colors.textSecondary,
                                        maxLines = 1,
                                        overflow = TextOverflow.Ellipsis,
                                    )
                                }
                            }
                            Spacer(Modifier.width(Spacing.sm))
                            when {
                                invitingId == userId -> InlineLoading()
                                userId in invitedIds -> Text(
                                    text = "已邀请",
                                    style = MaterialTheme.typography.labelMedium,
                                    color = colors.textSecondary,
                                )
                                else -> SystemButton(
                                    text = "邀请",
                                    onClick = { onInvite(item) },
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
