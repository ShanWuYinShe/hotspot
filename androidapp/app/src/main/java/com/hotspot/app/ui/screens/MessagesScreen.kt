package com.hotspot.app.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.data.model.PrivateChatItem
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.Badge
import com.hotspot.app.ui.components.BadgeSize
import com.hotspot.app.ui.components.BadgeTone
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.HeaderIconButton
import com.hotspot.app.ui.components.InlineLoading
import com.hotspot.app.ui.components.ListEnd
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.TimeFormat
import kotlinx.coroutines.launch

/** 消息列表：私聊会话（最新在前）；点开进入私聊。未读数用 Badge 语义色。 */
@Composable
fun MessagesScreen(
    onOpenNotify: () -> Unit,
    onOpenChat: (userId: String, nickname: String) -> Unit,
) {
    val colors = HotSpotTheme.colors
    val chatRepository = AppContainer.chatRepository
    val scope = rememberCoroutineScope()
    val state = rememberPagedListState<PrivateChatItem>()
    val listState = rememberLazyListState()
    val loader: suspend (String?) -> com.hotspot.app.data.api.ApiResult<com.hotspot.app.data.model.Page<PrivateChatItem>> = { cursor ->
        chatRepository.conversations(cursor)
    }

    LaunchedEffect(Unit) { state.initialLoad(loader) }
    InfiniteScrollHandler(state, listState, loader)

    Column(Modifier.fillMaxSize()) {
        PageHeader(
            title = "消息",
            subtitle = "私聊与通知都在这里",
            actions = {
                HeaderIconButton(
                    icon = Icons.Filled.Notifications,
                    contentDescription = "通知",
                    onClick = onOpenNotify,
                )
            },
        )
        when {
            state.loading && state.items.isEmpty() -> LoadingState(message = "正在加载会话…")
            state.error != null && state.items.isEmpty() -> ErrorState(
                message = state.error ?: "加载失败",
                onRetry = { scope.launch { state.initialLoad(loader) } },
            )
            state.items.isEmpty() -> EmptyState(
                title = "还没有会话",
                description = "去附近认识新朋友，或在个人主页发起私聊",
            )
            else -> LazyColumn(
                state = listState,
                modifier = Modifier.fillMaxSize(),
                contentPadding = PaddingValues(start = Spacing.lg, top = Spacing.sm, end = Spacing.lg, bottom = Spacing.xxl),
                verticalArrangement = Arrangement.spacedBy(Spacing.half),
            ) {
                items(state.items, key = { it.otherUser.userId }) { chat ->
                    ConversationRow(
                        chat = chat,
                        onClick = { onOpenChat(chat.otherUser.userId, chat.otherUser.nickname) },
                    )
                }
                item(key = "footer") {
                    when {
                        state.loadingMore -> InlineLoading()
                        state.endReached -> ListEnd()
                    }
                }
            }
        }
    }
}

@Composable
private fun ConversationRow(
    chat: PrivateChatItem,
    onClick: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(androidx.compose.foundation.shape.RoundedCornerShape(12.dp))
            .clickable(
                interactionSource = remember { MutableInteractionSource() },
                indication = null,
                onClick = onClick,
            )
            .padding(horizontal = Spacing.md, vertical = Spacing.md2),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box {
            Avatar(
                name = chat.otherUser.nickname,
                imageUrl = chat.otherUser.avatarUrl,
                size = Dimens.avatarMd,
            )
            if (chat.presenceStatus == "online") {
                Box(
                    modifier = Modifier
                        .align(Alignment.BottomEnd)
                        .size(12.dp)
                        .clip(CircleShape)
                        .background(colors.online),
                )
            }
        }
        Spacer(Modifier.width(Spacing.md))
        Column(Modifier.weight(1f)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = chat.otherUser.nickname.ifBlank { "匿名用户" },
                    style = MaterialTheme.typography.bodyLarge,
                    fontWeight = FontWeight.Medium,
                    color = colors.textPrimary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f),
                )
                Text(
                    text = TimeFormat.relative(chat.lastActivityAt),
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
            }
            Spacer(Modifier.height(2.dp))
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = previewText(chat),
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.textSecondary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f),
                )
                if (chat.unreadCount > 0) {
                    Spacer(Modifier.width(Spacing.sm))
                    Badge(
                        text = if (chat.unreadCount > 99) "99+" else chat.unreadCount.toString(),
                        tone = BadgeTone.SOLID,
                        size = BadgeSize.SM,
                    )
                }
            }
        }
    }
}

private fun previewText(chat: PrivateChatItem): String {
    val last = chat.lastMessage ?: return "开始聊天"
    return when (last.type) {
        "text" -> last.content ?: ""
        "image" -> "[图片]"
        "video" -> "[视频]"
        "audio" -> "[语音]"
        "file" -> "[文件]"
        "share" -> "[分享]"
        "system" -> last.content ?: "[系统消息]"
        else -> last.content ?: ""
    }
}
