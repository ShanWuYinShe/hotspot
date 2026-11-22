package com.hotspot.app.ui.screens

import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import coil.request.ImageRequest
import com.hotspot.app.AppContainer
import com.hotspot.app.R
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.media.ChatImageUploader
import com.hotspot.app.data.model.PrivateMessageItem
import com.hotspot.app.data.realtime.RealtimeEvent
import com.hotspot.app.navigation.ChatTarget
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemInput
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.ChatTypingRules
import com.hotspot.app.util.MediaUrls
import com.hotspot.app.util.TimeFormat
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * 私聊：历史消息 + 文本/图片发送 + WebSocket 实时增量（新消息/编辑/撤回，进入会话订阅对端频道）。
 * 历史消息按 sent_at 倒序返回，reverseLayout = true 让最新消息贴底（对齐前端「短会话底对齐」）。
 */
@Composable
fun ChatScreen(onBack: () -> Unit) {
    val colors = HotSpotTheme.colors
    val chatRepository = AppContainer.chatRepository
    val myUserId = AppContainer.tokenStore.session.value?.userId
    val targetUserId = ChatTarget.userId
    val nickname = ChatTarget.nickname.ifBlank { "私聊" }
    val scope = rememberCoroutineScope()
    val realtime = AppContainer.realtimeClient
    val listState = rememberLazyListState()

    var messages by remember { mutableStateOf<List<PrivateMessageItem>>(emptyList()) }
    var peerTyping by remember { mutableStateOf(false) }
    var peerTypingAt by remember { mutableStateOf(0L) }
    var lastTypingSentAt by remember { mutableStateOf(0L) }
    var readMarked by remember { mutableStateOf(false) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var sending by remember { mutableStateOf(false) }
    var uploading by remember { mutableStateOf(false) }
    var input by remember { mutableStateOf("") }

    suspend fun reload() {
        loading = messages.isEmpty()
        error = null
        when (val result = chatRepository.messages(targetUserId, null)) {
            is ApiResult.Ok -> messages = result.data.items
            is ApiResult.Err -> error = result.error.userMessage
        }
        loading = false
    }

    LaunchedEffect(targetUserId) { reload() }

    // 发送图片：系统 Photo Picker → chat_image 三步上传 → private_messages/send(type=image, file_id)。
    val context = LocalContext.current
    val imagePicker = rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()) { uri ->
        if (uri == null) return@rememberLauncherForActivityResult
        scope.launch {
            uploading = true
            error = null
            ChatImageUploader.upload(context, uri)
                .onSuccess { uploaded ->
                    when (val result = chatRepository.sendImageMessage(targetUserId, uploaded.fileId)) {
                        is ApiResult.Ok -> {
                            reload()
                            listState.animateScrollToItem(0)
                        }
                        is ApiResult.Err -> error = result.error.userMessage
                    }
                }
                .onFailure { error = it.message ?: "图片发送失败，请重试" }
            uploading = false
        }
    }

    // 对端输入中提示：收到 is_typing=true 后 6 秒内无新事件自动清除（防丢包常驻）。
    LaunchedEffect(peerTypingAt) {
        if (peerTypingAt > 0L) {
            delay(ChatTypingRules.TYPING_AUTO_CLEAR_MS)
            peerTyping = false
        }
    }
    DisposableEffect(targetUserId) {
        onDispose { realtime.sendPrivateTyping(targetUserId, false) }
    }

    // 打开私聊即视为已读（对齐 Web：会话停在最新消息近底部时 markRead）；
    // private_messages/mark_read(scope='session') 幂等，失败静默、下次进入重试。
    LaunchedEffect(messages.isNotEmpty(), readMarked) {
        if (messages.isNotEmpty() && !readMarked) {
            readMarked = true
            chatRepository.markPrivateRead(targetUserId)
        }
    }

    // WebSocket 实时增量（spec/api_spec_websocket.md）：进入会话订阅对端 user 频道，离开时取消。
    // 新消息按 message_id 去重合并；收到对端新消息时补一次 mark_read。
    LaunchedEffect(targetUserId) {
        realtime.subscribeUsers(listOf(targetUserId))
        try {
            realtime.events.collect { event ->
                when (event) {
                    is RealtimeEvent.PrivateMessageNew -> {
                        val message = event.message
                        val fromPeer = message.sender.userId == targetUserId
                        val selfEcho = message.sender.userId == myUserId && event.targetUserId == targetUserId
                        if (fromPeer || selfEcho) {
                            messages = mergePrivateMessage(messages, message)
                            if (listState.firstVisibleItemIndex <= 1) {
                                listState.animateScrollToItem(0)
                            }
                            if (fromPeer) {
                                scope.launch { chatRepository.markPrivateRead(targetUserId) }
                            }
                        }
                    }

                    is RealtimeEvent.PrivateMessageEdited -> {
                        messages = messages.map { message ->
                            if (message.messageId == event.messageId) {
                                message.copy(content = event.content, editedAt = event.editedAt)
                            } else {
                                message
                            }
                        }
                    }

                    is RealtimeEvent.PrivateTyping -> {
                        if (event.actor.userId == targetUserId) {
                            peerTyping = event.isTyping
                            if (event.isTyping) peerTypingAt = System.currentTimeMillis()
                        }
                    }

                    is RealtimeEvent.PrivateMessageRecalled -> {
                        messages = messages.map { message ->
                            if (message.messageId == event.messageId) {
                                message.copy(recalledAt = event.recalledAt)
                            } else {
                                message
                            }
                        }
                    }

                    else -> Unit
                }
            }
        } finally {
            realtime.unsubscribeUsers(listOf(targetUserId))
        }
    }

    fun onInputChange(value: String) {
        input = value
        val now = System.currentTimeMillis()
        if (value.isBlank()) {
            realtime.sendPrivateTyping(targetUserId, false)
            lastTypingSentAt = 0L
        } else if (now - lastTypingSentAt >= ChatTypingRules.TYPING_THROTTLE_MS) {
            realtime.sendPrivateTyping(targetUserId, true)
            lastTypingSentAt = now
        }
    }

    // 停止输入 3 秒后发 is_typing=false（输入变化会取消并重启该计时）。
    LaunchedEffect(input) {
        if (input.isBlank()) return@LaunchedEffect
        delay(ChatTypingRules.TYPING_IDLE_STOP_MS)
        realtime.sendPrivateTyping(targetUserId, false)
        lastTypingSentAt = 0L
    }

    fun send() {
        val text = input.trim()
        if (text.isEmpty() || sending) return
        scope.launch {
            sending = true
            when (val result = chatRepository.sendTextMessage(targetUserId, text)) {
                is ApiResult.Ok -> {
                    input = ""
                    realtime.sendPrivateTyping(targetUserId, false)
                    lastTypingSentAt = 0L
                    reload()
                    listState.animateScrollToItem(0)
                }
                is ApiResult.Err -> error = result.error.userMessage
            }
            sending = false
        }
    }

    Column(Modifier.fillMaxSize()) {
        PageHeader(
            title = nickname,
            subtitle = if (peerTyping) "正在输入…" else "私聊",
            onBack = {
                ChatTarget.clear()
                onBack()
            },
        )
        Box(Modifier.weight(1f)) {
            when {
                loading && messages.isEmpty() -> LoadingState(message = "正在加载消息…")
                error != null && messages.isEmpty() -> ErrorState(
                    message = error ?: "加载失败",
                    onRetry = { scope.launch { reload() } },
                )
                messages.isEmpty() -> EmptyState(
                    title = "还没有消息",
                    description = "发一条消息开始聊天",
                )
                else -> LazyColumn(
                    state = listState,
                    reverseLayout = true,
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(horizontal = Spacing.lg, vertical = Spacing.md),
                    verticalArrangement = Arrangement.spacedBy(Spacing.sm),
                ) {
                    items(messages, key = { it.messageId }) { message ->
                        MessageBubble(
                            message = message,
                            isMine = message.sender.userId == myUserId,
                        )
                    }
                }
            }
        }
        if (error != null && messages.isNotEmpty()) {
            Text(
                text = error ?: "",
                style = MaterialTheme.typography.bodySmall,
                color = colors.danger,
                modifier = Modifier.padding(horizontal = Spacing.lg),
            )
        }
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .background(colors.surface)
                .navigationBarsPadding()
                .imePadding()
                .padding(horizontal = Spacing.md, vertical = Spacing.sm),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Box(
                modifier = Modifier
                    .size(44.dp)
                    .clip(CircleShape)
                    .clickable(enabled = !uploading) {
                        imagePicker.launch(
                            PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly),
                        )
                    },
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painter = painterResource(R.drawable.ic_action_gallery),
                    contentDescription = if (uploading) "图片发送中" else "发送图片",
                    modifier = Modifier.size(Dimens.iconMd),
                    tint = if (uploading) colors.textSecondary else colors.accent,
                )
            }
            SystemInput(
                value = input,
                onValueChange = { onInputChange(it) },
                placeholder = "发消息…",
                modifier = Modifier.weight(1f),
            )
            Spacer(Modifier.width(Spacing.sm))
            SystemButton(
                text = "发送",
                onClick = { send() },
                variant = ButtonVariant.PRIMARY,
                size = ButtonSize.SM,
                enabled = input.isNotBlank(),
                loading = sending,
            )
        }
    }
}

private fun mergePrivateMessage(
    current: List<PrivateMessageItem>,
    incoming: PrivateMessageItem,
): List<PrivateMessageItem> = (current.filterNot { it.messageId == incoming.messageId } + incoming)
    .sortedByDescending { it.sentAt }

@Composable
private fun MessageBubble(
    message: PrivateMessageItem,
    isMine: Boolean,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = if (isMine) Arrangement.End else Arrangement.Start,
    ) {
        Column(horizontalAlignment = if (isMine) Alignment.End else Alignment.Start) {
            if (!isMine) {
                Text(
                    text = message.sender.nickname.ifBlank { "对方" },
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                    modifier = Modifier.padding(bottom = 2.dp, start = Spacing.xs),
                )
            }
            Box(
                modifier = Modifier
                    .widthIn(max = 280.dp)
                    .clip(
                        if (isMine) {
                            RoundedCornerShape(topStart = 16.dp, topEnd = 16.dp, bottomStart = 16.dp, bottomEnd = 4.dp)
                        } else {
                            RoundedCornerShape(topStart = 16.dp, topEnd = 16.dp, bottomStart = 4.dp, bottomEnd = 16.dp)
                        },
                    )
                    .background(if (isMine) colors.accent else colors.island)
                    .padding(horizontal = Spacing.md3, vertical = Spacing.sm2),
            ) {
                val imageUrl = message.media?.url
                if (message.recalledAt == null && message.type == "image" && !imageUrl.isNullOrBlank()) {
                    AsyncImage(
                        model = ImageRequest.Builder(LocalContext.current)
                            .data(MediaUrls.absolute(imageUrl))
                            .crossfade(true)
                            .build(),
                        contentDescription = "聊天图片",
                        modifier = Modifier
                            .widthIn(max = 220.dp)
                            .heightIn(max = 240.dp)
                            .clip(RoundedCornerShape(12.dp)),
                        contentScale = ContentScale.Crop,
                    )
                } else {
                    val text = when {
                        message.recalledAt != null -> "消息已撤回"
                        else -> when (message.type) {
                        "text" -> message.content ?: ""
                        "image" -> "[图片]"
                        "video" -> "[视频]"
                        "audio" -> "[语音]"
                        "file" -> "[文件]"
                        "share" -> "[分享]"
                        else -> message.content ?: "[消息]"
                        }
                    }
                    Text(
                        text = text,
                        style = MaterialTheme.typography.bodyLarge,
                        fontWeight = FontWeight.Normal,
                        color = if (isMine) colors.textOnAccent else colors.textPrimary,
                    )
                }
            }
            Text(
                text = TimeFormat.relative(message.sentAt),
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
                modifier = Modifier.padding(top = 2.dp, start = Spacing.xs, end = Spacing.xs),
            )
        }
    }
}
