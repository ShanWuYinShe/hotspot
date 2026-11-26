package com.hotspot.app.ui.components

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
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
import androidx.compose.ui.unit.dp
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.MessageAction

/**
 * 消息长按操作面板（私聊/群聊共用）：复制 / 撤回（自己的消息）/ 举报（他人消息）。
 * 使用 M3 ModalBottomSheet + 项目 token 容器色；点击遮罩或返回键关闭。
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MessageActionSheet(
    actions: List<MessageAction>,
    onAction: (MessageAction) -> Unit,
    onDismiss: () -> Unit,
    liked: Boolean = false,
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
                        text = actionLabel(action, liked),
                        style = MaterialTheme.typography.bodyLarge,
                        fontWeight = FontWeight.Medium,
                        color = if (action == MessageAction.RECALL) colors.danger else colors.textPrimary,
                    )
                }
                if (index != actions.lastIndex) {
                    HorizontalDivider(color = colors.borderSubtle)
                }
            }
        }
    }
}

private fun actionLabel(action: MessageAction, liked: Boolean): String = when (action) {
    MessageAction.LIKE -> if (liked) "取消点赞" else "点赞"
    MessageAction.COPY -> "复制"
    MessageAction.RECALL -> "撤回"
    MessageAction.REPORT -> "举报"
}
