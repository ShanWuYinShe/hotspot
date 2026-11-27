package com.hotspot.app.ui.screens

import android.Manifest
import android.content.pm.PackageManager
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.core.content.ContextCompat
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
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
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import android.content.ClipData
import android.content.ClipboardManager
import android.widget.Toast
import com.hotspot.app.AppContainer
import com.hotspot.app.R
import com.hotspot.app.data.api.ApiError
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.media.ChatFileUploader
import com.hotspot.app.data.media.ChatAudioUploader
import com.hotspot.app.data.media.ChatImageUploader
import com.hotspot.app.data.media.VoiceRecorder
import com.hotspot.app.data.media.ChatVideoUploader
import com.hotspot.app.data.model.GroupMemberItem
import com.hotspot.app.data.model.GroupMessageItem
import com.hotspot.app.data.realtime.RealtimeEvent
import com.hotspot.app.navigation.GroupChatTarget
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.ChatFileBubble
import com.hotspot.app.ui.components.AudioMessageBubble
import com.hotspot.app.ui.components.AudioHoldHintBar
import com.hotspot.app.ui.components.AudioRecorderBar
import com.hotspot.app.ui.components.ChatImageBubble
import com.hotspot.app.ui.components.ChatVideoBubble
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.FullscreenImageViewer
import com.hotspot.app.ui.components.MessageActionSheet
import com.hotspot.app.ui.components.MessageReactionBadge
import com.hotspot.app.ui.components.ReportDialog
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemCard
import com.hotspot.app.ui.components.SystemInput
import com.hotspot.app.ui.components.hotSpotFocusRing
import com.hotspot.app.ui.components.HoldToTalkMicButton
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTextStyles
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.ChatTypingRules
import com.hotspot.app.util.MentionQuery
import com.hotspot.app.util.ChatDrafts
import com.hotspot.app.util.MentionRules
import com.hotspot.app.util.MessageAction
import com.hotspot.app.util.MessageActionRules
import com.hotspot.app.util.toggleThumbReaction
import com.hotspot.app.util.applyReactionDelta
import com.hotspot.app.util.TimeFormat
import java.time.Instant
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * 群聊会话（设计稿 87:3529）：历史 + 文本/图片发送。
 *
 * 契约（spec/api_spec_chat.md）：
 * - 历史 group_messages/fetch：按 sent_at 倒序，reverseLayout = true 让最新消息贴底；
 * - 发送 group_messages/send：MUST 幂等，ApiClient 自动携带 Idempotency-Key；
 * - 非群成员返回 10200 → 展示设计稿空态文案（加入后才能查看群消息）。
 *
 * WebSocket 实时：新消息/编辑/撤回/已读数增量 + 输入中提示（进入群聊订阅 group 频道，离开取消）；
 * @ 提及：输入 @ 弹出群成员候选，插入「@昵称 」并在发送时提交 mentions（spec group_messages/send）。
 * 登记 TODO：富媒体消息。
 */
@Composable
fun GroupChatScreen(
    onBack: () -> Unit,
    onOpenGroup: (String) -> Unit,
) {
    val colors = HotSpotTheme.colors
    val chatRepository = AppContainer.chatRepository
    val myUserId = AppContainer.tokenStore.session.value?.userId
    val target = GroupChatTarget
    val groupId = target.groupId
    val scope = rememberCoroutineScope()
    val realtime = AppContainer.realtimeClient
    val listState = rememberLazyListState()

    var messages by remember { mutableStateOf<List<GroupMessageItem>>(emptyList()) }
    var typingUserIds by remember { mutableStateOf<List<String>>(emptyList()) }
    var typingVersion by remember { mutableStateOf(0) }
    var lastTypingSentAt by remember { mutableStateOf(0L) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var notMember by remember { mutableStateOf(false) }
    var sending by remember { mutableStateOf(false) }
    var uploading by remember { mutableStateOf(false) }
    var fileUploading by remember { mutableStateOf(false) }
    val draftContext = LocalContext.current
    val draftKey = "group_$groupId"
    var input by remember(draftKey) { mutableStateOf(ChatDrafts.load(draftContext, draftKey)) }
    var readMarked by remember { mutableStateOf(false) }
    var members by remember { mutableStateOf<List<GroupMemberItem>>(emptyList()) }
    var selectedMentionIds by remember { mutableStateOf<Set<String>>(emptySet()) }
    var cursorResetToken by remember { mutableStateOf(0) }
    // 消息举报：长按他人消息（spec/reports/create target_type=message，服务端冻结快照）
    var reportTarget by remember { mutableStateOf<GroupMessageItem?>(null) }
    var reportSubmitting by remember { mutableStateOf(false) }
    // 长按消息操作面板（复制/撤回/举报）
    var actionTarget by remember { mutableStateOf<GroupMessageItem?>(null) }
    // 全屏图片查看器（点击群聊图片）
    var viewerImageUrl by remember { mutableStateOf<String?>(null) }

    suspend fun reload() {
        loading = messages.isEmpty()
        error = null
        notMember = false
        when (val result = chatRepository.groupMessages(groupId, null)) {
            is ApiResult.Ok -> messages = result.data.items
            is ApiResult.Err -> {
                val business = result.error as? ApiError.Business
                if (business?.code == 10200) {
                    notMember = true
                } else {
                    error = result.error.userMessage
                }
            }
        }
        loading = false
    }

    LaunchedEffect(groupId) { reload() }

    // 发送图片：系统 Photo Picker → chat_image 三步上传 → group_messages/send(type=image, file_id)。
    val context = LocalContext.current

    // 发送文件：系统 DocumentsUI → chat_file 流式三步上传 → group_messages/send(type=file, file_id)。
    val filePicker = rememberLauncherForActivityResult(ActivityResultContracts.OpenDocument()) { uri ->
        if (uri == null) return@rememberLauncherForActivityResult
        scope.launch {
            fileUploading = true
            error = null
            try {
                ChatFileUploader.upload(context, uri)
                    .onSuccess { uploaded ->
                        when (val result = chatRepository.sendGroupFileMessage(groupId, uploaded.fileId)) {
                            is ApiResult.Ok -> {
                                reload()
                                listState.animateScrollToItem(0)
                            }
                            is ApiResult.Err -> error = result.error.userMessage
                        }
                    }
                    .onFailure { error = it.message ?: "文件发送失败，请重试" }
            } finally {
                fileUploading = false
            }
        }
    }
    // 发送图片/视频：系统 Photo Picker（ImageAndVideo）→ 按 MIME 分流 chat_image / chat_video 三步上传。
    val imagePicker = rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()) { uri ->
        if (uri == null) return@rememberLauncherForActivityResult
        scope.launch {
            uploading = true
            error = null
            try {
                val mime = context.contentResolver.getType(uri)?.lowercase().orEmpty()
                if (mime.startsWith("video/")) {
                    ChatVideoUploader.upload(context, uri)
                        .onSuccess { uploaded ->
                            when (val result = chatRepository.sendGroupVideoMessage(groupId, uploaded.fileId)) {
                                is ApiResult.Ok -> {
                                    reload()
                                    listState.animateScrollToItem(0)
                                }
                                is ApiResult.Err -> error = result.error.userMessage
                            }
                        }
                        .onFailure { error = it.message ?: "视频发送失败，请重试" }
                } else {
                    ChatImageUploader.upload(context, uri)
                        .onSuccess { uploaded ->
                            when (val result = chatRepository.sendGroupImageMessage(groupId, uploaded.fileId)) {
                                is ApiResult.Ok -> {
                                    reload()
                                    listState.animateScrollToItem(0)
                                }
                                is ApiResult.Err -> error = result.error.userMessage
                            }
                        }
                        .onFailure { error = it.message ?: "图片发送失败，请重试" }
                }
            } finally {
                uploading = false
            }
        }
    }

    // 语音消息：点麦克风开始录音（AAC/ADTS，chat_audio），录音栏可取消或发送。
    var recording by remember { mutableStateOf(false) }
    var recordingSeconds by remember { mutableIntStateOf(0) }
    var audioSending by remember { mutableStateOf(false) }
    val recorder = remember { VoiceRecorder(context) }
    DisposableEffect(Unit) {
        onDispose {
            recorder.cancel()
            // 离开聊天页停止语音，避免全局播放器在别的页面继续响。
            AppContainer.audioPlaybackController.stop()
        }
    }
    fun beginVoiceRecording() {
        recorder.start()
            .onSuccess {
                recordingSeconds = 0
                recording = true
            }
            .onFailure { Toast.makeText(context, it.message ?: "无法开始录音", Toast.LENGTH_SHORT).show() }
    }
    var micGranted by remember {
        mutableStateOf(
            ContextCompat.checkSelfPermission(context, Manifest.permission.RECORD_AUDIO) ==
                PackageManager.PERMISSION_GRANTED,
        )
    }
    // 按住说话：录音期间输入行保持组合，另在顶部显示「松开发送 · 上滑取消」提示条。
    var gestureHolding by remember { mutableStateOf(false) }
    var slideCancel by remember { mutableStateOf(false) }
    val micPermissionLauncher = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
        if (granted) {
            micGranted = true
            beginVoiceRecording()
        } else {
            Toast.makeText(context, "需要麦克风权限才能发送语音", Toast.LENGTH_SHORT).show()
        }
    }
    val requestMicPermission = { micPermissionLauncher.launch(Manifest.permission.RECORD_AUDIO) }
    fun sendVoiceRecording() {
        if (audioSending) return
        val audio = recorder.stop().getOrNull()
        recording = false
        if (audio == null || audio.durationSeconds < 1) {
            audio?.file?.delete()
            Toast.makeText(context, "录音太短，请至少录制 1 秒", Toast.LENGTH_SHORT).show()
            return
        }
        audioSending = true
        scope.launch {
            try {
                ChatAudioUploader.upload(audio.file)
                    .onSuccess { uploaded ->
                        when (val result = chatRepository.sendGroupAudioMessage(groupId, uploaded.fileId)) {
                            is ApiResult.Ok -> {
                                reload()
                                listState.animateScrollToItem(0)
                            }
                            is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                        }
                    }
                    .onFailure { Toast.makeText(context, it.message ?: "语音发送失败，请重试", Toast.LENGTH_SHORT).show() }
            } finally {
                audio.file.delete()
                audioSending = false
            }
        }
    }
    LaunchedEffect(recording) {
        if (recording) {
            recordingSeconds = 0
            while (true) {
                delay(1000)
                recordingSeconds += 1
            }
        }
    }

    // @ 提及候选：群成员列表（公开信息；失败不阻塞聊天，只是不弹候选）。
    LaunchedEffect(groupId) {
        when (val result = AppContainer.groupRepository.members(groupId, null)) {
            is ApiResult.Ok -> members = result.data.items
            is ApiResult.Err -> Unit
        }
    }

    // 输入中成员 6 秒内无新事件自动清空（任一成员继续输入会重置计时）。
    LaunchedEffect(typingVersion) {
        if (typingUserIds.isNotEmpty()) {
            delay(ChatTypingRules.TYPING_AUTO_CLEAR_MS)
            typingUserIds = emptyList()
        }
    }
    DisposableEffect(groupId) {
        onDispose { realtime.sendGroupTyping(groupId, false) }
    }

    // 打开群聊即视为已读（对齐 Web：会话停在最新消息近底部时标记已读）；
    // group_messages/mark_read 幂等，失败静默、下次进入重试，不阻塞消息展示。
    LaunchedEffect(messages.isNotEmpty(), readMarked) {
        if (messages.isNotEmpty() && !readMarked) {
            readMarked = true
            chatRepository.markGroupRead(groupId)
        }
    }

    // WebSocket 实时增量：进入群聊订阅 group 频道，离开取消；新消息去重合并并补 mark_read。
    LaunchedEffect(groupId) {
        realtime.subscribeGroups(listOf(groupId))
        try {
            realtime.events.collect { event ->
                when (event) {
                    is RealtimeEvent.GroupMessageNew -> {
                        if (event.groupId == groupId) {
                            val senderId = event.message.sender.userId
                            if (typingUserIds.contains(senderId)) {
                                typingUserIds = typingUserIds - senderId
                                typingVersion++
                            }
                            messages = mergeGroupMessage(messages, event.message)
                            if (listState.firstVisibleItemIndex <= 1) {
                                listState.animateScrollToItem(0)
                            }
                            if (event.message.sender.userId != myUserId) {
                                scope.launch { chatRepository.markGroupRead(groupId) }
                            }
                        }
                    }

                    is RealtimeEvent.GroupMessageEdited -> {
                        if (event.groupId == groupId) {
                            messages = messages.map { message ->
                                if (message.messageId == event.messageId) {
                                    message.copy(content = event.content, editedAt = event.editedAt)
                                } else {
                                    message
                                }
                            }
                        }
                    }

                    is RealtimeEvent.GroupMessageRecalled -> {
                        if (event.groupId == groupId) {
                            messages = messages.map { message ->
                                if (message.messageId == event.messageId) {
                                    message.copy(recalledAt = event.recalledAt)
                                } else {
                                    message
                                }
                            }
                        }
                    }

                    // 表情反应实时同步（对齐 Web / 鸿蒙）：事件不带计数，按 ±1 合并
                    is RealtimeEvent.ReactionAdded -> {
                        if (event.targetType == "message") {
                            val mine = event.actor.userId == myUserId
                            messages = messages.map { message ->
                                if (message.messageId == event.targetId) {
                                    message.copy(reactions = applyReactionDelta(message.reactions, event.emoji, true, mine))
                                } else {
                                    message
                                }
                            }
                        }
                    }

                    is RealtimeEvent.ReactionRemoved -> {
                        if (event.targetType == "message") {
                            val mine = event.actor.userId == myUserId
                            messages = messages.map { message ->
                                if (message.messageId == event.targetId) {
                                    message.copy(reactions = applyReactionDelta(message.reactions, event.emoji, false, mine))
                                } else {
                                    message
                                }
                            }
                        }
                    }

                    is RealtimeEvent.GroupTyping -> {
                        val actorId = event.actor.userId
                        if (event.groupId == groupId && actorId.isNotBlank() && actorId != myUserId) {
                            typingUserIds = if (event.isTyping) {
                                (typingUserIds + actorId).distinct()
                            } else {
                                typingUserIds - actorId
                            }
                            typingVersion++
                        }
                    }

                    is RealtimeEvent.GroupMessageReadCountUpdated -> {
                        if (event.groupId == groupId) {
                            messages = messages.map { message ->
                                if (message.messageId == event.messageId) {
                                    message.copy(readCount = event.readCount)
                                } else {
                                    message
                                }
                            }
                        }
                    }

                    else -> Unit
                }
            }
        } finally {
            realtime.unsubscribeGroups(listOf(groupId))
        }
    }

    fun onInputChange(value: String) {
        input = value
        ChatDrafts.save(draftContext, draftKey, value)
        val now = System.currentTimeMillis()
        if (value.isBlank()) {
            realtime.sendGroupTyping(groupId, false)
            lastTypingSentAt = 0L
        } else if (now - lastTypingSentAt >= ChatTypingRules.TYPING_THROTTLE_MS) {
            realtime.sendGroupTyping(groupId, true)
            lastTypingSentAt = now
        }
    }

    // 停止输入 3 秒后发 is_typing=false（输入变化会取消并重启该计时）。
    LaunchedEffect(input) {
        if (input.isBlank()) return@LaunchedEffect
        delay(ChatTypingRules.TYPING_IDLE_STOP_MS)
        realtime.sendGroupTyping(groupId, false)
        lastTypingSentAt = 0L
    }

    fun send() {
        val text = input.trim()
        if (text.isEmpty() || sending) return
        val mentionIds = MentionRules.activeMentionIds(
            text = text,
            candidates = members.map { it.user.userId to it.user.nickname },
            selectedIds = selectedMentionIds,
        )
        scope.launch {
            sending = true
            when (val result = chatRepository.sendGroupTextMessage(groupId, text, mentionIds)) {
                is ApiResult.Ok -> {
                    input = ""
                    ChatDrafts.clear(draftContext, draftKey)
                    selectedMentionIds = emptySet()
                    realtime.sendGroupTyping(groupId, false)
                    lastTypingSentAt = 0L
                    reload()
                    listState.animateScrollToItem(0)
                }
                is ApiResult.Err -> error = result.error.userMessage
            }
            sending = false
        }
    }

    val typingHint = ChatTypingRules.buildGroupTypingHint(typingUserIds) { userId ->
        messages.lastOrNull { it.sender.userId == userId }?.sender?.nickname
    }
    val mentionQuery = MentionRules.findActiveQuery(input)
    val mentionCandidates = if (mentionQuery == null) {
        emptyList()
    } else {
        members
            .filter { it.user.userId != myUserId }
            .filter { it.user.nickname.contains(mentionQuery.keyword, ignoreCase = true) }
            .take(MENTION_CANDIDATE_LIMIT)
    }

    Column(Modifier.fillMaxSize()) {
        GroupChatHeader(
            name = target.name.ifBlank { "群聊" },
            avatarUrl = target.avatarUrl,
            membersCount = target.membersCount,
            onlineCount = target.onlineCount,
            typingHint = typingHint,
            onBack = {
                GroupChatTarget.clear()
                onBack()
            },
            onTitleClick = if (groupId.isNotBlank()) {
                { onOpenGroup(groupId) }
            } else {
                null
            },
        )
        Box(Modifier.weight(1f)) {
            when {
                loading && messages.isEmpty() -> LoadingState(message = "正在加载群消息…")
                notMember -> ErrorState(
                    message = "你还不是这个群组的成员，加入后才能查看群消息",
                    onRetry = { scope.launch { reload() } },
                )
                error != null && messages.isEmpty() -> ErrorState(
                    message = error ?: "加载失败",
                    onRetry = { scope.launch { reload() } },
                )
                messages.isEmpty() -> EmptyState(
                    title = "还没有消息",
                    description = "发一条消息，和群友打个招呼",
                )
                else -> LazyColumn(
                    state = listState,
                    reverseLayout = true,
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(horizontal = Spacing.lg, vertical = Spacing.md),
                    verticalArrangement = Arrangement.spacedBy(Spacing.sm),
                ) {
                    items(messages, key = { it.messageId }) { message ->
                        GroupMessageRow(
                            message = message,
                            isMine = message.sender.userId == myUserId,
                            onLongPress = if (!message.isRecalled && message.type != "system") {
                                { actionTarget = message }
                            } else {
                                null
                            },
                            onImageClick = { viewerImageUrl = it },
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
        if (mentionCandidates.isNotEmpty() && mentionQuery != null) {
            MentionCandidatePanel(
                query = mentionQuery,
                candidates = mentionCandidates,
                onPick = { member ->
                    val newValue = MentionRules.insertMention(input, input.length, mentionQuery, member.user.nickname)
                    selectedMentionIds = selectedMentionIds + member.user.userId
                    onInputChange(newValue)
                    cursorResetToken++
                },
            )
        }
        Column(Modifier.fillMaxWidth()) {
            if (gestureHolding) {
                AudioHoldHintBar(seconds = recordingSeconds, slideCancel = slideCancel)
            }
            if ((recording && !gestureHolding) || audioSending) {
                Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(colors.surface)
                    .navigationBarsPadding()
                    .imePadding(),
            ) {
                AudioRecorderBar(
                    seconds = recordingSeconds,
                    sending = audioSending,
                    onCancel = {
                        recorder.cancel()
                        recording = false
                        recordingSeconds = 0
                    },
                    onSend = { sendVoiceRecording() },
                )
            }
        } else {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(colors.surface)
                    .navigationBarsPadding()
                    .imePadding()
                    .padding(horizontal = Spacing.md, vertical = Spacing.sm),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                if (micGranted) {
                    HoldToTalkMicButton(
                        enabled = !uploading,
                        tint = colors.accent,
                        modifier = Modifier.size(44.dp),
                        onPressStart = {
                            gestureHolding = true
                            slideCancel = false
                            beginVoiceRecording()
                        },
                        onSlideCancelChange = { slideCancel = it },
                        onRelease = { cancelled ->
                            gestureHolding = false
                            slideCancel = false
                            if (cancelled) {
                                recorder.cancel()
                                recording = false
                                recordingSeconds = 0
                                Toast.makeText(context, "已取消录音", Toast.LENGTH_SHORT).show()
                            } else {
                                sendVoiceRecording()
                            }
                        },
                    )
                } else {
                    Box(
                        modifier = Modifier
                            .size(44.dp)
                            .clip(CircleShape)
                            .clickable { requestMicPermission() },
                        contentAlignment = Alignment.Center,
                    ) {
                        Icon(
                            painter = painterResource(R.drawable.ic_mic),
                            contentDescription = "按住说话",
                            modifier = Modifier.size(Dimens.iconMd),
                            tint = colors.accent,
                        )
                    }
                }
                Box(
                    modifier = Modifier
                        .size(44.dp)
                        .clip(CircleShape)
                        .clickable(enabled = !uploading) {
                            imagePicker.launch(
                                PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageAndVideo),
                            )
                        },
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        painter = painterResource(R.drawable.ic_action_gallery),
                        contentDescription = if (uploading) "媒体发送中" else "发送图片或视频",
                        modifier = Modifier.size(Dimens.iconMd),
                        tint = if (uploading) colors.textSecondary else colors.accent,
                    )
                }
                Box(
                    modifier = Modifier
                        .size(44.dp)
                        .clip(CircleShape)
                        .clickable(enabled = !fileUploading) { filePicker.launch(arrayOf("*/*")) },
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        painter = painterResource(R.drawable.ic_action_file),
                        contentDescription = if (fileUploading) "文件发送中" else "发送文件",
                        modifier = Modifier.size(Dimens.iconMd),
                        tint = if (fileUploading) colors.textSecondary else colors.accent,
                    )
                }
                SystemInput(
                    value = input,
                    onValueChange = { onInputChange(it) },
                    placeholder = "发消息…",
                    modifier = Modifier.weight(1f),
                    cursorResetToken = cursorResetToken,
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
    }

    actionTarget?.let { target ->
        val actions = MessageActionRules.availableActions(
            isMine = target.sender.userId == myUserId,
            isRecalled = target.isRecalled,
            type = target.type,
            content = target.content,
        )
        if (actions.isNotEmpty()) {
            MessageActionSheet(
                actions = actions,
                liked = target.reactions.any { it.reactedByMe },
                editContent = target.content.orEmpty(),
                onEditSubmit = { text ->
                    actionTarget = null
                    scope.launch {
                        when (val result = chatRepository.updateGroupMessage(target.messageId, text)) {
                            is ApiResult.Ok -> messages = messages.map { message ->
                                if (message.messageId == target.messageId) {
                                    // 接口只返回成功码，内容以本次提交值与本地时间标记（WS 回环到达后幂等覆盖）
                                    message.copy(content = text, editedAt = Instant.now().toString())
                                } else {
                                    message
                                }
                            }
                            is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                        }
                    }
                },
                onDismiss = { actionTarget = null },
                onAction = { action ->
                    actionTarget = null
                    when (action) {
                        MessageAction.LIKE -> {
                            val next = !target.reactions.any { it.reactedByMe }
                            val previous = target
                            messages = messages.map { message ->
                                if (message.messageId == target.messageId) {
                                    message.copy(reactions = toggleThumbReaction(message.reactions, next))
                                } else {
                                    message
                                }
                            }
                            scope.launch {
                                when (val result = chatRepository.setMessageReaction(target.messageId, next)) {
                                    is ApiResult.Ok -> Unit
                                    is ApiResult.Err -> {
                                        messages = messages.map { message ->
                                            if (message.messageId == target.messageId) message.copy(reactions = previous.reactions) else message
                                        }
                                        Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                                    }
                                }
                            }
                        }
                        MessageAction.COPY -> {
                            val text = target.content.orEmpty()
                            if (text.isNotBlank()) {
                                val clipboard = context.getSystemService(ClipboardManager::class.java)
                                clipboard?.setPrimaryClip(ClipData.newPlainText("消息", text))
                                Toast.makeText(context, "已复制", Toast.LENGTH_SHORT).show()
                            }
                        }
                        MessageAction.RECALL -> scope.launch {
                            when (val result = chatRepository.recallGroupMessage(target.messageId)) {
                                is ApiResult.Ok -> {
                                    messages = messages.map { message ->
                                        if (message.messageId == target.messageId) {
                                            message.copy(recalledAt = Instant.now().toString())
                                        } else {
                                            message
                                        }
                                    }
                                }
                                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                            }
                        }
                        MessageAction.REPORT -> reportTarget = target
                        // EDIT 由 MessageActionSheet 内部进入编辑态，经 onEditSubmit 上抛，不会走到这里
                        MessageAction.EDIT -> Unit
                    }
                },
            )
        } else {
            actionTarget = null
        }
    }

    reportTarget?.let { target ->
        ReportDialog(
            title = "举报消息",
            submitting = reportSubmitting,
            onDismiss = { if (!reportSubmitting) reportTarget = null },
            onSubmit = { reason ->
                if (reportSubmitting) return@ReportDialog
                reportSubmitting = true
                scope.launch {
                    when (val result = AppContainer.reportsRepository.create("message", target.messageId, reason)) {
                        is ApiResult.Ok -> {
                            reportTarget = null
                            Toast.makeText(
                                context,
                                if (result.data.isDuplicate) "已提交过举报，等待处理" else "举报已提交",
                                Toast.LENGTH_SHORT,
                            ).show()
                        }
                        is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                    }
                    reportSubmitting = false
                }
            },
        )
    }

    viewerImageUrl?.let { url ->
        FullscreenImageViewer(
            imageUrl = url,
            contentDescription = "群聊图片",
            onDismiss = { viewerImageUrl = null },
        )
    }
}

/** 群聊头部：返回钮 + 群头像（在线点）+ 群名 + 成员数（设计稿 87:3529）。 */
@Composable
private fun GroupChatHeader(
    name: String,
    avatarUrl: String?,
    membersCount: Int,
    onlineCount: Int,
    typingHint: String = "",
    onBack: () -> Unit,
    onTitleClick: (() -> Unit)? = null,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .height(Dimens.backButton + Spacing.md2)
            .padding(horizontal = Spacing.lg),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            modifier = Modifier
                .size(Dimens.minTouchTarget)
                .clip(CircleShape)
                .clickable(
                    interactionSource = remember { MutableInteractionSource() },
                    indication = null,
                    onClick = onBack,
                )
                .hotSpotFocusRing(cornerRadius = 22.dp),
            contentAlignment = Alignment.Center,
        ) {
            Box(
                modifier = Modifier
                    .size(Dimens.backButton)
                    .shadow(
                        elevation = 1.dp,
                        shape = CircleShape,
                        ambientColor = colors.shadowDark,
                        spotColor = colors.shadowDark,
                    )
                    .clip(CircleShape)
                    .background(colors.surface),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                    contentDescription = "返回",
                    modifier = Modifier.size(Dimens.iconSm),
                    tint = colors.textPrimary,
                )
            }
        }
        Spacer(Modifier.width(Spacing.sm))
        Box {
            Avatar(name = name, imageUrl = avatarUrl, size = Dimens.avatarXs)
            if (onlineCount > 0) {
                Box(
                    modifier = Modifier
                        .align(Alignment.BottomEnd)
                        .size(Spacing.md2)
                        .clip(CircleShape)
                        .background(colors.surface)
                        .padding(2.dp)
                        .clip(CircleShape)
                        .background(colors.accent),
                )
            }
        }
        Spacer(Modifier.width(Spacing.sm))
        Column(
            Modifier
                .weight(1f)
                .then(if (onTitleClick != null) Modifier.clickable(onClick = onTitleClick) else Modifier),
        ) {
            Text(
                text = name,
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
                color = colors.textPrimary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                text = if (typingHint.isNotBlank()) typingHint else "$membersCount 位成员",
                style = MaterialTheme.typography.bodySmall,
                color = if (typingHint.isNotBlank()) colors.accent else colors.textSecondary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}

/** 群聊消息行：系统/撤回消息居中；他人消息带 22dp 头像 + 昵称，自己消息右对齐。 */
@Composable
private fun GroupMessageRow(
    message: GroupMessageItem,
    isMine: Boolean,
    onLongPress: (() -> Unit)? = null,
    onImageClick: ((String) -> Unit)? = null,
) {
    val colors = HotSpotTheme.colors
    if (message.type == "system" || message.isRecalled) {
        Text(
            text = if (message.isRecalled) "该消息已撤回" else (message.content ?: ""),
            style = MaterialTheme.typography.labelSmall,
            color = colors.textSecondary,
            textAlign = TextAlign.Center,
            modifier = Modifier
                .fillMaxWidth()
                .padding(vertical = Spacing.xs),
        )
        return
    }
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = if (isMine) Arrangement.End else Arrangement.Start,
    ) {
        if (!isMine) {
            Avatar(
                name = message.sender.nickname,
                imageUrl = message.sender.avatarUrl,
                size = Dimens.avatarXxs,
            )
            Spacer(Modifier.width(Spacing.sm))
        }
        Column(horizontalAlignment = if (isMine) Alignment.End else Alignment.Start) {
            if (!isMine) {
                Text(
                    text = message.sender.nickname.ifBlank { "群友" },
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                    modifier = Modifier.padding(bottom = 2.dp, start = Spacing.xs),
                )
            }
            val imageUrl = message.media?.url
            if (!message.isRecalled && message.type == "image" && !imageUrl.isNullOrBlank()) {
                // 图片不套气泡外壳（对齐 Web / 鸿蒙）：圆角图直接贴聊天背景，时间在图下
                ChatImageBubble(
                    imageUrl = imageUrl,
                    contentDescription = "群聊图片",
                    onClick = onImageClick?.let { click -> { click(imageUrl) } },
                    onLongClick = onLongPress,
                )
            } else {
            Box(
                modifier = Modifier
                    .widthIn(max = 264.dp)
                    .clip(
                        if (isMine) {
                            RoundedCornerShape(topStart = 16.dp, topEnd = 16.dp, bottomStart = 16.dp, bottomEnd = 4.dp)
                        } else {
                            RoundedCornerShape(topStart = 16.dp, topEnd = 16.dp, bottomStart = 4.dp, bottomEnd = 16.dp)
                        },
                    )
                    .background(if (isMine) colors.accent else colors.island)
                    .then(
                        if (onLongPress != null) {
                            Modifier.combinedClickable(onClick = {}, onLongClick = onLongPress)
                        } else {
                            Modifier
                        },
                    )
                    .padding(horizontal = Spacing.md, vertical = Spacing.sm2),
            ) {
                val imageUrl = message.media?.url
                if (!message.isRecalled && message.type == "audio") {
                    AudioMessageBubble(
                        audioUrl = imageUrl,
                        isMine = isMine,
                        durationSeconds = message.media?.duration,
                    )
                } else if (!message.isRecalled && message.type == "video") {
                    ChatVideoBubble(
                        fileName = message.media?.fileName,
                        durationSeconds = message.media?.duration,
                        thumbnailUrl = message.media?.thumbnailUrl,
                        videoUrl = imageUrl,
                    )
                } else if (!message.isRecalled && message.type == "file") {
                    ChatFileBubble(
                        fileName = message.media?.fileName,
                        fileSize = message.media?.fileSize,
                        fileUrl = imageUrl,
                        isMine = isMine,
                    )
                } else {
                    Text(
                        text = highlightedMentionText(
                            text = messageText(message),
                            mentionColor = if (isMine) colors.textOnAccent else colors.accent,
                        ),
                        style = HotSpotTextStyles.chat,
                        fontWeight = FontWeight.Normal,
                        color = if (isMine) colors.textOnAccent else colors.textPrimary,
                    )
                }
            }
            }
            MessageReactionBadge(message.reactions)
            Row(
                modifier = Modifier.padding(top = 2.dp, start = Spacing.xs, end = Spacing.xs),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = TimeFormat.relative(message.sentAt),
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
                if (!message.isRecalled && !message.editedAt.isNullOrBlank()) {
                    Text(
                        text = "· 已编辑",
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textSecondary,
                        modifier = Modifier.padding(start = 2.dp),
                    )
                }
            }
        }
    }
}

/** @提及候选面板：输入 @ 时展示匹配的群成员，点击插入「@昵称 」。 */
@Composable
private fun MentionCandidatePanel(
    query: MentionQuery,
    candidates: List<GroupMemberItem>,
    onPick: (GroupMemberItem) -> Unit,
) {
    val colors = HotSpotTheme.colors
    SystemCard(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = Spacing.lg, vertical = Spacing.xs),
        contentPadding = PaddingValues(vertical = Spacing.xs),
    ) {
        Text(
            text = "选择要 @ 的成员" + if (query.keyword.isBlank()) "" else "（" + query.keyword + "）",
            style = MaterialTheme.typography.labelSmall,
            color = colors.textSecondary,
            modifier = Modifier.padding(horizontal = Spacing.md2, vertical = Spacing.xs),
        )
        candidates.forEach { member ->
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable { onPick(member) }
                    .padding(horizontal = Spacing.md2, vertical = Spacing.sm),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Avatar(
                    name = member.user.nickname,
                    imageUrl = member.user.avatarUrl,
                    size = Dimens.avatarXxs,
                )
                Spacer(Modifier.width(Spacing.sm))
                Text(
                    text = member.user.nickname.ifBlank { "群成员" },
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.textPrimary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}

/** 消息气泡内 @昵称 高亮（他人消息用强调色加粗，自己气泡用反白加粗保持可读）。 */
private fun highlightedMentionText(text: String, mentionColor: Color): AnnotatedString {
    val mentionStyle = SpanStyle(color = mentionColor, fontWeight = FontWeight.SemiBold)
    return buildAnnotatedString {
        var last = 0
        Regex("@[^\\s@]+").findAll(text).forEach { match ->
            append(text.substring(last, match.range.first))
            withStyle(mentionStyle) { append(match.value) }
            last = match.range.last + 1
        }
        append(text.substring(last))
    }
}

private fun mergeGroupMessage(
    current: List<GroupMessageItem>,
    incoming: GroupMessageItem,
): List<GroupMessageItem> = (current.filterNot { it.messageId == incoming.messageId } + incoming)
    .sortedByDescending { it.sentAt }

private const val MENTION_CANDIDATE_LIMIT = 5

/** 首版只渲染文本；其余类型给占位文案（媒体/分享渲染为后续迭代）。 */
private fun messageText(message: GroupMessageItem): String = when (message.type) {
    "text" -> message.content ?: ""
    "image" -> "[图片]"
    "video" -> "[视频]"
    "audio" -> "[语音]"
    "file" -> "[文件]"
    "share" -> "[分享]"
    else -> message.content ?: "[消息]"
}
