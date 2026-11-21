package com.hotspot.app.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.WindowInsetsSides
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.ime
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.only
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.union
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Send
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.hotspot.app.data.model.UserInfo
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.ui.theme.Tokens

// 输入条几何（docs/ui-components.md「详情页契约」/ Figma 522:7）：
// 头像 36、发送钮 40——头像/按钮刻度（32/40/42）是近邻档，按「缺档不新增近邻档位」登记为组件几何。
private val CommentAvatarSize = 36.dp
private val SendButtonSize = 40.dp

/**
 * 评论输入条（页面私有原语，全端唯一形态 docked，docs/ui-components.md「详情页契约」）：
 * 头像 36 + 输入槽（43 高 / 圆角 16 / surface-subtle 底，focus 转 surface-primary + accent 环）
 * + 发送钮 40×40（有内容 accent 实底、空 surface-subtle），底部按 nav bar / IME inset 避让。
 * 回复态在输入条上方显示可取消的「回复 @昵称」条。
 *
 * 输入族登记例外：评论输入条与聊天输入条同类（输入条语义，不是表单字段），
 * 因此不走 SystemInput，直接使用 .input-surface 等价的原语内表面状态。
 */
@Composable
fun CommentInputBar(
    value: String,
    onValueChange: (String) -> Unit,
    onSend: () -> Unit,
    currentUser: UserInfo?,
    replyLabel: String?,
    onCancelReply: () -> Unit,
    modifier: Modifier = Modifier,
    sending: Boolean = false,
    focusRequester: FocusRequester = remember { FocusRequester() },
) {
    val colors = HotSpotTheme.colors
    var focused by remember { mutableStateOf(false) }
    val canSend = value.isNotBlank() && !sending

    Column(
        modifier = modifier
            .fillMaxWidth()
            .background(colors.surface)
            .drawTopHairline(colors.borderSubtle)
            .windowInsetsPadding(
                WindowInsets.navigationBars.union(WindowInsets.ime).only(WindowInsetsSides.Bottom),
            ),
    ) {
        if (replyLabel != null) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(Dimens.minTouchTarget)
                    .background(colors.subtle)
                    .padding(start = Spacing.md3, end = Spacing.sm),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = replyLabel,
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.textSecondary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f),
                )
                Box(
                    modifier = Modifier
                        .fillMaxHeight()
                        .widthIn(min = 56.dp)
                        .clickable(
                            interactionSource = remember { MutableInteractionSource() },
                            indication = null,
                            onClick = onCancelReply,
                        )
                        .hotSpotFocusRing(cornerRadius = Tokens.Radius.sm.dp),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = "取消",
                        style = MaterialTheme.typography.bodySmall.copy(fontWeight = FontWeight.Medium),
                        color = colors.accent,
                    )
                }
            }
        }
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = Spacing.md, vertical = Spacing.sm),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Avatar(
                name = currentUser?.nickname?.ifBlank { "我" } ?: "我",
                imageUrl = currentUser?.avatarUrl,
                size = CommentAvatarSize,
            )
            Spacer(Modifier.width(Spacing.md2))
            Box(
                modifier = Modifier
                    .weight(1f)
                    .height(Dimens.inputHeight)
                    .clip(AppShapes.compact)
                    .background(if (focused) colors.surface else colors.subtle, AppShapes.compact)
                    // 聚焦单层加粗到 2dp（不再叠内缘柔光环，避免双层描边）
                    .border(if (focused) 2.dp else 1.dp, if (focused) colors.fieldBorderFocus else colors.border, AppShapes.compact)
                    .padding(horizontal = Spacing.md3),
                contentAlignment = Alignment.CenterStart,
            ) {
                if (value.isEmpty()) {
                    Text(
                        text = "写下你的评论…",
                        style = MaterialTheme.typography.bodyLarge,
                        color = colors.textSecondary,
                        maxLines = 1,
                    )
                }
                BasicTextField(
                    value = value,
                    onValueChange = onValueChange,
                    modifier = Modifier
                        .fillMaxWidth()
                        .focusRequester(focusRequester)
                        .onFocusChanged { focused = it.isFocused },
                    // 发送中不禁用输入框：禁用会丢焦点、IME 自动收起（发送后想继续评论要重新点）
                    singleLine = true,
                    textStyle = MaterialTheme.typography.bodyLarge.copy(color = colors.textPrimary),
                    cursorBrush = SolidColor(colors.accent),
                    keyboardOptions = KeyboardOptions(imeAction = ImeAction.Send),
                    keyboardActions = KeyboardActions(onSend = { if (canSend) onSend() }),
                )
            }
            Spacer(Modifier.width(Spacing.sm))
            Box(
                modifier = Modifier
                    .size(Dimens.minTouchTarget)
                    .clip(AppShapes.pill)
                    .clickable(
                        interactionSource = remember { MutableInteractionSource() },
                        indication = null,
                        enabled = canSend,
                        onClick = onSend,
                    )
                    .hotSpotFocusRing(cornerRadius = Tokens.Radius.full.dp),
                contentAlignment = Alignment.Center,
            ) {
                Box(
                    modifier = Modifier
                        .size(SendButtonSize)
                        .clip(AppShapes.pill)
                        .background(if (canSend) colors.accent else colors.subtle),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        imageVector = Icons.AutoMirrored.Filled.Send,
                        contentDescription = "发送评论",
                        modifier = Modifier.size(Dimens.iconXs),
                        tint = if (canSend) colors.textOnAccent else colors.textSecondary,
                    )
                }
            }
        }
    }
}

/** 输入条顶部发丝线（内容分隔，非页面/卡片边界；design-system §19.3 允许保留的功能线）。 */
private fun Modifier.drawTopHairline(color: Color): Modifier = drawBehind {
    drawLine(
        color = color,
        start = Offset.Zero,
        end = Offset(size.width, 0f),
        strokeWidth = 1.dp.toPx(),
    )
}

/** 回复态指示文案（输入条上方可取消 chip，与 Web「回复 @昵称」口径一致）。 */
fun replyChipLabel(nickname: String): String =
    if (nickname.isBlank()) "回复" else "回复 @" + nickname
