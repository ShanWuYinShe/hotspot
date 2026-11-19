package com.hotspot.app.ui.screens

import androidx.compose.foundation.background
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
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
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
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.PrivateMessageItem
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
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.TimeFormat
import kotlinx.coroutines.launch

/**
 * 私聊：首版只读历史消息 + 简单文本发送（WebSocket 实时推送 / 已读回执 / 富媒体为登记 TODO）。
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
    val listState = rememberLazyListState()

    var messages by remember { mutableStateOf<List<PrivateMessageItem>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var sending by remember { mutableStateOf(false) }
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

    fun send() {
        val text = input.trim()
        if (text.isEmpty() || sending) return
        scope.launch {
            sending = true
            when (val result = chatRepository.sendTextMessage(targetUserId, text)) {
                is ApiResult.Ok -> {
                    input = ""
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
            subtitle = "私聊",
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
            SystemInput(
                value = input,
                onValueChange = { input = it },
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
                val text = when (message.type) {
                    "text" -> message.content ?: ""
                    "image" -> "[图片]"
                    "video" -> "[视频]"
                    "audio" -> "[语音]"
                    "file" -> "[文件]"
                    "share" -> "[分享]"
                    else -> message.content ?: "[消息]"
                }
                Text(
                    text = text,
                    style = MaterialTheme.typography.bodyLarge,
                    fontWeight = FontWeight.Normal,
                    color = if (isMine) colors.textOnAccent else colors.textPrimary,
                )
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
