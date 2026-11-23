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
import com.hotspot.app.data.model.GroupApplicationItem
import com.hotspot.app.ui.components.InlineLoading
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.TimeFormat

/** 群收件箱：待审批入群申请（owner/admin）。 */
@Composable
fun GroupApplicationsDialog(
    applications: List<GroupApplicationItem>?,
    loading: Boolean,
    error: String?,
    processingId: String?,
    onRetry: () -> Unit,
    onApprove: (GroupApplicationItem) -> Unit,
    onReject: (GroupApplicationItem) -> Unit,
    onDismiss: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("入群申请") },
        text = {
            Column(
                modifier = Modifier
                    .heightIn(max = 420.dp)
                    .verticalScroll(rememberScrollState()),
            ) {
                when {
                    loading && applications == null -> Text("正在加载申请…", color = colors.textSecondary)
                    error != null && applications == null -> {
                        Text(error, color = colors.danger)
                        Spacer(Modifier.height(Spacing.sm))
                        SystemButton(
                            text = "重试",
                            onClick = onRetry,
                            variant = ButtonVariant.SECONDARY,
                            size = ButtonSize.SM,
                        )
                    }
                    applications.isNullOrEmpty() -> Text("暂无待审批的入群申请", color = colors.textSecondary)
                    else -> applications.forEach { application ->
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(vertical = Spacing.sm2),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Avatar(
                                name = application.applicant.nickname,
                                imageUrl = application.applicant.avatarUrl,
                                size = Dimens.avatarSm,
                            )
                            Spacer(Modifier.width(Spacing.sm2))
                            Column(modifier = Modifier.weight(1f)) {
                                Text(
                                    text = application.applicant.nickname.ifBlank { "匿名用户" },
                                    style = MaterialTheme.typography.bodyLarge,
                                    color = colors.textPrimary,
                                    maxLines = 1,
                                    overflow = TextOverflow.Ellipsis,
                                )
                                Text(
                                    text = TimeFormat.relative(application.createdAt) +
                                        if (application.message.isNullOrBlank()) "" else " · " + application.message,
                                    style = MaterialTheme.typography.bodySmall,
                                    color = colors.textSecondary,
                                    maxLines = 2,
                                    overflow = TextOverflow.Ellipsis,
                                )
                            }
                            Spacer(Modifier.width(Spacing.sm))
                            if (processingId == application.applicationId) {
                                InlineLoading()
                            } else {
                                SystemButton(
                                    text = "拒绝",
                                    onClick = { onReject(application) },
                                    variant = ButtonVariant.SECONDARY,
                                    size = ButtonSize.SM,
                                )
                                Spacer(Modifier.width(Spacing.xs))
                                SystemButton(
                                    text = "同意",
                                    onClick = { onApprove(application) },
                                    variant = ButtonVariant.PRIMARY,
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
