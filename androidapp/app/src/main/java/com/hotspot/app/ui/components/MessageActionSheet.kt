package com.hotspot.app.ui.components

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.MessageAction

/**
 * 消息长按操作面板（私聊/群聊共用）：复制 / 编辑（自己的文本）/ 撤回（自己的消息）/ 举报（他人消息）。
 * 使用 M3 ModalBottomSheet + 项目 token 容器色；点「编辑」在面板内切换为输入态（对齐鸿蒙
 * MessageActionsSheet 的 编辑 → TextArea + 完成/取消），提交经 onEditSubmit 上抛，组件保持纯展示。
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MessageActionSheet(
    actions: List<MessageAction>,
    onAction: (MessageAction) -> Unit,
    onDismiss: () -> Unit,
    liked: Boolean = false,
    editContent: String = "",
    onEditSubmit: ((String) -> Unit)? = null,
) {
    val colors = HotSpotTheme.colors
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    var editing by remember { mutableStateOf(false) }
    var draft by remember { mutableStateOf("") }

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
                .imePadding()
                .padding(start = Spacing.lg, end = Spacing.lg, bottom = Spacing.lg),
        ) {
            if (editing) {
                Text(
                    text = "编辑消息",
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold,
                    color = colors.textPrimary,
                )
                Spacer(Modifier.height(Spacing.md2))
                SystemTextArea(
                    value = draft,
                    onValueChange = { draft = it },
                    placeholder = "输入修改后的内容",
                    minHeight = Dimens.inputHeight * 2,
                )
                Spacer(Modifier.height(Spacing.md2))
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(Spacing.sm),
                ) {
                    SystemButton(
                        text = "取消",
                        onClick = { editing = false },
                        variant = ButtonVariant.SECONDARY,
                        size = ButtonSize.MD,
                        modifier = Modifier.weight(1f),
                    )
                    SystemButton(
                        text = "完成",
                        onClick = {
                            val trimmed = draft.trim()
                            if (trimmed.isNotEmpty()) {
                                editing = false
                                onEditSubmit?.invoke(trimmed)
                            }
                        },
                        variant = ButtonVariant.PRIMARY,
                        size = ButtonSize.MD,
                        enabled = draft.trim().isNotEmpty(),
                        modifier = Modifier.weight(1f),
                    )
                }
            } else {
                actions.forEachIndexed { index, action ->
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .heightIn(min = Dimens.minTouchTarget)
                            .clickable {
                                if (action == MessageAction.EDIT) {
                                    draft = editContent
                                    editing = true
                                } else {
                                    onAction(action)
                                }
                            }
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
}

private fun actionLabel(action: MessageAction, liked: Boolean): String = when (action) {
    MessageAction.LIKE -> if (liked) "取消点赞" else "点赞"
    MessageAction.COPY -> "复制"
    MessageAction.EDIT -> "编辑"
    MessageAction.RECALL -> "撤回"
    MessageAction.REPORT -> "举报"
}
