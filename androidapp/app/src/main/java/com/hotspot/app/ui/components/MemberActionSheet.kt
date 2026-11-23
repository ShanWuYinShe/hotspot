package com.hotspot.app.ui.components

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing

/** 群成员管理操作（owner/admin）：角色、禁言、移除。 */
enum class MemberAction(val label: String, val destructive: Boolean = false) {
    SET_ADMIN("设为管理员"),
    UNSET_ADMIN("取消管理员"),
    MUTE_1H("禁言 1 小时"),
    UNMUTE("解除禁言"),
    REMOVE("移除成员", destructive = true),
    TRANSFER_OWNER("转让群主", destructive = true),
    BAN("封禁成员", destructive = true),
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MemberActionSheet(
    actions: List<MemberAction>,
    onAction: (MemberAction) -> Unit,
    onDismiss: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        containerColor = colors.surface,
        shape = AppShapes.sheet,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .navigationBarsPadding()
                .padding(start = Spacing.lg, end = Spacing.lg, bottom = Spacing.lg),
        ) {
            actions.forEachIndexed { index, action ->
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .heightIn(min = Dimens.minTouchTarget)
                        .clickable { onAction(action) }
                        .padding(horizontal = Spacing.xs, vertical = Spacing.md),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        text = action.label,
                        style = MaterialTheme.typography.bodyLarge,
                        fontWeight = FontWeight.Medium,
                        color = if (action.destructive) colors.danger else colors.textPrimary,
                    )
                }
                if (index != actions.lastIndex) {
                    HorizontalDivider(color = colors.borderSubtle)
                }
            }
        }
    }
}
