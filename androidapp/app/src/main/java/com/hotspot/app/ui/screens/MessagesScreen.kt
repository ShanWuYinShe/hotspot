package com.hotspot.app.ui.screens

import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.combinedClickable
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
import androidx.compose.material3.AlertDialog
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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import android.widget.Toast
import androidx.compose.ui.window.Dialog
import com.hotspot.app.AppContainer
import com.hotspot.app.R
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.GroupItem
import com.hotspot.app.data.model.Page
import com.hotspot.app.data.model.PrivateChatItem
import com.hotspot.app.data.model.UserItem
import com.hotspot.app.data.realtime.RealtimeProtocol
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
import com.hotspot.app.util.ChatDrafts
import com.hotspot.app.util.TimeFormat
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

private const val REALTIME_REFRESH_DEBOUNCE_MS = 250L

/**
 * 消息列表（设计稿 87:3617：页头「消息 + 全部已读/N 条未读 + 新会话」→ 搜索会话 → 「最近」→ 会话行）。
 * 通知中心入口在「我的」页，消息页不再放铃铛（与设计稿一致）。
 * 新会话为端内扩展弹层（设计稿未出画板）：搜索用户 → 直接进入私聊。
 * WebSocket 实时：收到聊天类增量后 250ms 防抖补拉私聊会话与群会话列表（WS 只推增量）。
 */
@Composable
fun MessagesScreen(
    onOpenChat: (userId: String, nickname: String) -> Unit,
    onOpenGroupChat: (GroupItem) -> Unit,
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
    val groupRepository = AppContainer.profileRepository
    var groupChats by remember { mutableStateOf<List<GroupItem>>(emptyList()) }
    var groupLoading by remember { mutableStateOf(true) }
    var groupError by remember { mutableStateOf<String?>(null) }
    val context = LocalContext.current
    var actionEntry by remember { mutableStateOf<ConversationEntry?>(null) }
    var deleteEntry by remember { mutableStateOf<ConversationEntry?>(null) }
    var actionBusy by remember { mutableStateOf(false) }

    suspend fun reloadGroups() {
        groupLoading = true
        groupError = null
        when (val result = groupRepository.joinedGroups(null)) {
            is ApiResult.Ok -> groupChats = result.data.items
            is ApiResult.Err -> groupError = result.error.userMessage
        }
        groupLoading = false
    }

    LaunchedEffect(Unit) { state.initialLoad(loader) }
    LaunchedEffect(Unit) { reloadGroups() }
    InfiniteScrollHandler(state, listState, loader)

    // WebSocket 实时增量：收到聊天类事件后轻量补拉会话列表（spec：WS 只推增量，列表以 HTTP 补拉为准）。
    LaunchedEffect(Unit) {
        AppContainer.realtimeClient.events.collect { event ->
            if (RealtimeProtocol.isChatListAffecting(event)) {
                delay(REALTIME_REFRESH_DEBOUNCE_MS)
                state.initialLoad(loader)
                reloadGroups()
            }
        }
    }

    // 私聊 + 群聊聚合（对齐 Web useChatListData：置顶 → 有未读 → 最新时间倒序）
    val conversations = remember(state.items, groupChats) {
        buildConversations(state.items, groupChats)
    }
    val totalUnread = conversations.sumOf { it.unread }
    val query = searchQuery.trim()
    val filtered = if (query.isEmpty()) {
        conversations
    } else {
        conversations.filter { it.name.contains(query, ignoreCase = true) }
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
            (state.loading || groupLoading) && conversations.isEmpty() -> LoadingState(message = "正在加载会话…")
            state.error != null && conversations.isEmpty() -> ErrorState(
                message = state.error ?: "加载失败",
                onRetry = { scope.launch { state.initialLoad(loader) } },
            )
            groupError != null && conversations.isEmpty() -> ErrorState(
                message = groupError ?: "群聊加载失败",
                onRetry = { scope.launch { reloadGroups() } },
            )
            conversations.isEmpty() -> EmptyState(
                title = "还没有会话",
                description = "和附近的人打个招呼，会话就会出现在这里。",
            )
            filtered.isEmpty() -> EmptyState(
                title = "没有找到会话",
                description = "换个昵称或群名再试试。",
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
                    if (groupError != null && query.isEmpty()) {
                        item(key = "group-error") {
                            GroupLoadErrorRow(
                                message = groupError ?: "",
                                onRetry = { scope.launch { reloadGroups() } },
                            )
                        }
                    }
                    items(filtered, key = { it.key }) { entry ->
                        ConversationRow(
                            entry = entry,
                            onClick = {
                                if (entry.kind == ConversationKind.PRIVATE) {
                                    entry.privateChat?.let { onOpenChat(it.otherUser.userId, it.otherUser.nickname) }
                                } else {
                                    entry.group?.let(onOpenGroupChat)
                                }
                            },
                            onLongClick = { actionEntry = entry },
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

    actionEntry?.let { entry ->
        AlertDialog(
            onDismissRequest = { if (!actionBusy) actionEntry = null },
            title = { Text(entry.name.ifBlank { "会话操作" }) },
            text = {
                Column {
                    SystemButton(
                        text = if (entry.pinned) "取消置顶" else "置顶会话",
                        onClick = {
                            if (actionBusy) return@SystemButton
                            scope.launch {
                                actionBusy = true
                                val result = if (entry.kind == ConversationKind.PRIVATE) {
                                    chatRepository.updateConversationPreferences(
                                        targetUserId = entry.privateChat?.otherUser?.userId.orEmpty(),
                                        isPinned = !entry.pinned,
                                    )
                                } else {
                                    AppContainer.groupRepository.updateGroupPreferences(
                                        groupId = entry.group?.group?.groupId.orEmpty(),
                                        isPinned = !entry.pinned,
                                    )
                                }
                                actionBusy = false
                                when (result) {
                                    is ApiResult.Ok -> {
                                        actionEntry = null
                                        state.initialLoad(loader)
                                        reloadGroups()
                                    }
                                    is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                                }
                            }
                        },
                        variant = ButtonVariant.SECONDARY,
                        size = ButtonSize.SM,
                        fillMaxWidth = true,
                        enabled = !actionBusy,
                    )
                    Spacer(Modifier.height(Spacing.sm))
                    SystemButton(
                        text = if (entry.muted) "取消免打扰" else "消息免打扰",
                        onClick = {
                            if (actionBusy) return@SystemButton
                            scope.launch {
                                actionBusy = true
                                val result = if (entry.kind == ConversationKind.PRIVATE) {
                                    chatRepository.updateConversationPreferences(
                                        targetUserId = entry.privateChat?.otherUser?.userId.orEmpty(),
                                        isMuted = !entry.muted,
                                    )
                                } else {
                                    AppContainer.groupRepository.updateGroupPreferences(
                                        groupId = entry.group?.group?.groupId.orEmpty(),
                                        isMuted = !entry.muted,
                                    )
                                }
                                actionBusy = false
                                when (result) {
                                    is ApiResult.Ok -> {
                                        actionEntry = null
                                        state.initialLoad(loader)
                                        reloadGroups()
                                    }
                                    is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                                }
                            }
                        },
                        variant = ButtonVariant.SECONDARY,
                        size = ButtonSize.SM,
                        fillMaxWidth = true,
                        enabled = !actionBusy,
                    )
                    if (entry.kind == ConversationKind.PRIVATE) {
                        Spacer(Modifier.height(Spacing.sm))
                        SystemButton(
                            text = "删除会话",
                            onClick = {
                                if (!actionBusy) {
                                    actionEntry = null
                                    deleteEntry = entry
                                }
                            },
                            variant = ButtonVariant.DANGER,
                            size = ButtonSize.SM,
                            fillMaxWidth = true,
                            enabled = !actionBusy,
                        )
                    }
                }
            },
            confirmButton = {},
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { if (!actionBusy) actionEntry = null },
                    variant = ButtonVariant.GHOST,
                    size = ButtonSize.SM,
                    enabled = !actionBusy,
                )
            },
        )
    }

    deleteEntry?.let { entry ->
        AlertDialog(
            onDismissRequest = { if (!actionBusy) deleteEntry = null },
            title = { Text("删除会话？") },
            text = { Text("将移除「" + entry.name + "」的会话入口与本地历史，对方视角不受影响。") },
            confirmButton = {
                SystemButton(
                    text = if (actionBusy) "删除中…" else "删除",
                    onClick = {
                        if (actionBusy) return@SystemButton
                        scope.launch {
                            actionBusy = true
                            val result = chatRepository.deleteConversation(entry.privateChat?.otherUser?.userId.orEmpty())
                            actionBusy = false
                            when (result) {
                                is ApiResult.Ok -> {
                                    ChatDrafts.clear(
                                        context,
                                        "private_" + entry.privateChat?.otherUser?.userId.orEmpty(),
                                    )
                                    deleteEntry = null
                                    state.initialLoad(loader)
                                }
                                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                            }
                        }
                    },
                    variant = ButtonVariant.DANGER,
                    size = ButtonSize.SM,
                    enabled = !actionBusy,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { if (!actionBusy) deleteEntry = null },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !actionBusy,
                )
            },
        )
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

@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun ConversationRow(
    entry: ConversationEntry,
    onClick: () -> Unit,
    onLongClick: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(androidx.compose.foundation.shape.RoundedCornerShape(12.dp))
            .combinedClickable(
                interactionSource = remember { MutableInteractionSource() },
                indication = null,
                onClick = onClick,
                onLongClick = onLongClick,
            )
            .padding(start = Spacing.lg, end = Spacing.md, top = Spacing.md2, bottom = Spacing.md2),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box {
            Avatar(
                name = entry.name,
                imageUrl = entry.avatarUrl,
                size = Dimens.avatarMd,
                // 群聊头像用圆角方（对齐 Web ChatListItem shape=rounded），与私聊圆形 + 在线点区分
                shape = if (entry.kind == ConversationKind.GROUP) AppShapes.iconTile else CircleShape,
            )
            when {
                entry.kind == ConversationKind.GROUP -> Box(
                    modifier = Modifier
                        .align(Alignment.BottomEnd)
                        .size(16.dp)
                        .clip(CircleShape)
                        .background(colors.surface)
                        .padding(2.dp)
                        .clip(CircleShape)
                        .background(colors.accent),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        painter = painterResource(R.drawable.ic_badge_group),
                        contentDescription = "群聊",
                        modifier = Modifier.size(9.dp),
                        tint = colors.textOnAccent,
                    )
                }
                entry.online -> Box(
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
                    text = entry.name.ifBlank { "匿名用户" },
                    style = MaterialTheme.typography.bodyLarge,
                    fontWeight = FontWeight.Medium,
                    color = colors.textPrimary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f),
                )
                Text(
                    text = entry.timeLabel,
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
            }
            Spacer(Modifier.height(2.dp))
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = entry.preview,
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.textSecondary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f),
                )
                if (entry.pinned) {
                    Spacer(Modifier.width(Spacing.sm))
                    Badge(text = "置顶", tone = BadgeTone.NEUTRAL, size = BadgeSize.SM)
                }
                if (entry.unread > 0) {
                    Spacer(Modifier.width(Spacing.sm))
                    Badge(
                        text = if (entry.unread > 99) "99+" else entry.unread.toString(),
                        tone = BadgeTone.SOLID,
                        size = BadgeSize.SM,
                    )
                }
            }
        }
    }
}

/** 群聊加载失败的非阻断提示行：私聊列表仍可用，只提示群会话未加载并提供重试。 */
@Composable
private fun GroupLoadErrorRow(message: String, onRetry: () -> Unit) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = Spacing.md, vertical = Spacing.sm),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = "群聊加载失败：" + message,
            style = MaterialTheme.typography.bodySmall,
            color = colors.danger,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(1f),
        )
        Spacer(Modifier.width(Spacing.sm))
        SystemButton(
            text = "重试",
            onClick = onRetry,
            variant = ButtonVariant.GHOST,
            size = ButtonSize.SM,
        )
    }
}

private enum class ConversationKind { PRIVATE, GROUP }

/** 私聊/群聊统一会话行模型（渲染与排序都只用这一层）。 */
private data class ConversationEntry(
    val key: String,
    val kind: ConversationKind,
    val name: String,
    val avatarUrl: String?,
    val preview: String,
    val timeLabel: String,
    val timestamp: Long,
    val unread: Int,
    val pinned: Boolean,
    val muted: Boolean,
    val online: Boolean,
    val privateChat: PrivateChatItem? = null,
    val group: GroupItem? = null,
)

/**
 * 私聊 + 群聊聚合，排序对齐 Web lib/chat-list compareChatItems：
 * 置顶 → 有未读 → 最新时间倒序 → id 稳定 tie-breaker。
 * 群聊数据来自 groups/fetch_joined（viewer_state.unread_count / last_message）。
 */
private fun buildConversations(
    privateChats: List<PrivateChatItem>,
    groups: List<GroupItem>,
): List<ConversationEntry> {
    val priv = privateChats.map { chat ->
        ConversationEntry(
            key = "chat-" + chat.otherUser.userId,
            kind = ConversationKind.PRIVATE,
            name = chat.otherUser.nickname,
            avatarUrl = chat.otherUser.avatarUrl,
            preview = previewText(chat),
            timeLabel = TimeFormat.relative(chat.lastMessage?.sentAt ?: chat.lastActivityAt),
            timestamp = parseIso(chat.lastMessage?.sentAt ?: chat.lastActivityAt),
            unread = chat.unreadCount,
            pinned = chat.isPinned,
            muted = chat.isMuted,
            online = chat.presenceStatus == "online",
            privateChat = chat,
        )
    }
    val groupRows = groups.map { item ->
        ConversationEntry(
            key = "group-" + item.group.groupId,
            kind = ConversationKind.GROUP,
            name = item.group.name.ifBlank { "未命名群组" },
            avatarUrl = item.group.avatarUrl,
            preview = groupPreview(item),
            timeLabel = TimeFormat.relative(item.lastMessage?.sentAt ?: item.lastActivityAt),
            timestamp = parseIso(item.lastMessage?.sentAt ?: item.lastActivityAt),
            unread = item.viewerState?.unreadCount ?: 0,
            pinned = item.viewerState?.isPinned ?: false,
            muted = item.viewerState?.isMuted ?: false,
            online = false,
            group = item,
        )
    }
    return (priv + groupRows).sortedWith(
        compareByDescending<ConversationEntry> { it.pinned }
            .thenByDescending { it.unread > 0 }
            .thenByDescending { it.timestamp }
            .thenBy { it.key },
    )
}

private fun parseIso(iso: String?): Long =
    if (iso.isNullOrBlank()) 0L else runCatching { java.time.Instant.parse(iso).toEpochMilli() }.getOrDefault(0L)

/** 群聊预览带发送者前缀（"野原：..."），与 Web 群会话行语义一致。 */
private fun groupPreview(item: GroupItem): String {
    val last = item.lastMessage ?: return "群聊已创建"
    val body = when (last.type) {
        "text" -> last.content ?: ""
        "image" -> "[图片]"
        "video" -> "[视频]"
        "audio" -> "[语音]"
        "file" -> "[文件]"
        "share" -> "[分享]"
        "system" -> last.content ?: "[系统消息]"
        else -> last.content ?: ""
    }
    val sender = last.sender.nickname
    return if (sender.isBlank()) body else sender + "：" + body
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
