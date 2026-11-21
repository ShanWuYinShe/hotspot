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
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.Page
import com.hotspot.app.data.model.PrivateChatItem
import com.hotspot.app.data.model.UserItem
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.Badge
import com.hotspot.app.ui.components.BadgeSize
import com.hotspot.app.ui.components.BadgeTone
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.InlineLoading
import com.hotspot.app.ui.components.ListEnd
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SearchInput
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemCard
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.TimeFormat
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * 消息列表（设计稿 87:3617：页头「消息 + 全部已读/N 条未读 + 新会话」→ 搜索会话 → 「最近」→ 会话行）。
 * 通知中心入口在「我的」页，消息页不再放铃铛（与设计稿一致）。
 * 新会话为端内扩展弹层（设计稿未出画板）：搜索用户 → 直接进入私聊。
 */
@Composable
fun MessagesScreen(
    onOpenChat: (userId: String, nickname: String) -> Unit,
) {
    val colors = HotSpotTheme.colors
    val chatRepository = AppContainer.chatRepository
    val scope = rememberCoroutineScope()
    val state = rememberPagedListState<PrivateChatItem>()
    val listState = rememberLazyListState()
    var searchQuery by remember { mutableStateOf("") }
    var showNewChat by remember { mutableStateOf(false) }
    val loader: suspend (String?) -> ApiResult<Page<PrivateChatItem>> = { cursor ->
        chatRepository.conversations(cursor)
    }

    LaunchedEffect(Unit) { state.initialLoad(loader) }
    InfiniteScrollHandler(state, listState, loader)

    val totalUnread = state.items.sumOf { it.unreadCount }
    val query = searchQuery.trim()
    val filtered = if (query.isEmpty()) {
        state.items
    } else {
        state.items.filter { it.otherUser.nickname.contains(query, ignoreCase = true) }
    }

    Column(Modifier.fillMaxSize()) {
        PageHeader(
            title = "消息",
            actions = {
                Text(
                    text = if (totalUnread > 0) "$totalUnread 条未读" else "全部已读",
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
                Spacer(Modifier.width(Spacing.sm))
                SystemButton(
                    text = "新会话",
                    onClick = { showNewChat = true },
                    variant = ButtonVariant.SOFT,
                    size = ButtonSize.SM,
                    leadingIcon = {
                        Icon(
                            imageVector = Icons.Filled.Add,
                            contentDescription = null,
                            modifier = Modifier.size(Dimens.iconXs),
                            tint = colors.textSecondary,
                        )
                    },
                )
            },
        )
        // 搜索会话：本地过滤昵称（对齐 Web MessagesMobile 的 searchQuery）
        SearchInput(
            value = searchQuery,
            onValueChange = { searchQuery = it },
            placeholder = "搜索会话…",
            modifier = Modifier.padding(horizontal = Spacing.lg, vertical = Spacing.sm2),
        )
        when {
            state.loading && state.items.isEmpty() -> LoadingState(message = "正在加载会话…")
            state.error != null && state.items.isEmpty() -> ErrorState(
                message = state.error ?: "加载失败",
                onRetry = { scope.launch { state.initialLoad(loader) } },
            )
            state.items.isEmpty() -> EmptyState(
                title = "还没有会话",
                description = "和附近的人打个招呼，会话就会出现在这里。",
            )
            filtered.isEmpty() -> EmptyState(
                title = "没有找到会话",
                description = "换个昵称再试试。",
            )
            else -> {
                Text(
                    text = "最近",
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                    modifier = Modifier.padding(start = Spacing.lg, top = Spacing.sm, bottom = Spacing.xs),
                )
                LazyColumn(
                    state = listState,
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(
                        start = Spacing.lg,
                        top = Spacing.xs,
                        end = Spacing.lg,
                        bottom = Spacing.xxl,
                    ),
                    verticalArrangement = Arrangement.spacedBy(Spacing.half),
                ) {
                    items(filtered, key = { it.otherUser.userId }) { chat ->
                        ConversationRow(
                            chat = chat,
                            onClick = { onOpenChat(chat.otherUser.userId, chat.otherUser.nickname) },
                        )
                    }
                    // 本地过滤时不展示分页尾注（避免「没有更多了」紧跟单条过滤结果）
                    if (query.isEmpty()) {
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
    }

    if (showNewChat) {
        NewConversationDialog(
            onDismiss = { showNewChat = false },
            onSelect = { user ->
                showNewChat = false
                onOpenChat(user.user.userId, user.user.nickname)
            },
        )
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

/**
 * 新会话弹层（设计稿未出画板，沿用端内弹层语言）：搜索昵称 → 点人进入私聊。
 * 空输入给出用法提示，不做「猜你想搜」的假数据。
 */
@Composable
private fun NewConversationDialog(
    onDismiss: () -> Unit,
    onSelect: (UserItem) -> Unit,
) {
    val colors = HotSpotTheme.colors
    var query by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(false) }
    var errorText by remember { mutableStateOf<String?>(null) }
    var results by remember { mutableStateOf<List<UserItem>>(emptyList()) }

    LaunchedEffect(query) {
        val keyword = query.trim()
        if (keyword.isEmpty()) {
            results = emptyList()
            errorText = null
            loading = false
            return@LaunchedEffect
        }
        delay(300)
        loading = true
        errorText = null
        when (val result = AppContainer.searchRepository.searchUsers(keyword, null)) {
            is ApiResult.Ok -> results = result.data.items
            is ApiResult.Err -> {
                results = emptyList()
                errorText = result.error.userMessage
            }
        }
        loading = false
    }

    Dialog(onDismissRequest = onDismiss) {
        SystemCard(
            modifier = Modifier.fillMaxWidth(),
            shape = AppShapes.sheet,
            contentPadding = PaddingValues(Spacing.lg),
        ) {
            Text(
                text = "新会话",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
                color = colors.textPrimary,
            )
            Spacer(Modifier.height(Spacing.md))
            SearchInput(
                value = query,
                onValueChange = { query = it },
                placeholder = "搜索昵称发起私聊",
            )
            Spacer(Modifier.height(Spacing.sm2))
            when {
                query.isBlank() -> Text(
                    text = "输入昵称，找到人就能开始聊天。",
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.textSecondary,
                )
                loading -> InlineLoading()
                errorText != null -> Text(
                    text = errorText ?: "",
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.danger,
                )
                results.isEmpty() -> Text(
                    text = "没有找到相关用户",
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.textSecondary,
                )
                else -> LazyColumn(Modifier.heightIn(max = 320.dp)) {
                    items(results, key = { it.user.userId }) { user ->
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .clip(AppShapes.compact)
                                .clickable { onSelect(user) }
                                .padding(vertical = Spacing.sm, horizontal = Spacing.xs),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Avatar(
                                name = user.user.nickname,
                                imageUrl = user.user.avatarUrl,
                                size = Dimens.avatarSm,
                            )
                            Spacer(Modifier.width(Spacing.sm))
                            Column(Modifier.weight(1f)) {
                                Text(
                                    text = user.user.nickname.ifBlank { "匿名用户" },
                                    style = MaterialTheme.typography.bodyMedium,
                                    color = colors.textPrimary,
                                    maxLines = 1,
                                    overflow = TextOverflow.Ellipsis,
                                )
                                if (!user.bio.isNullOrBlank()) {
                                    Text(
                                        text = user.bio ?: "",
                                        style = MaterialTheme.typography.bodySmall,
                                        color = colors.textSecondary,
                                        maxLines = 1,
                                        overflow = TextOverflow.Ellipsis,
                                    )
                                }
                            }
                        }
                    }
                }
            }
            Spacer(Modifier.height(Spacing.md))
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.End) {
                SystemButton(
                    text = "取消",
                    onClick = onDismiss,
                    variant = ButtonVariant.GHOST,
                    size = ButtonSize.SM,
                )
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
