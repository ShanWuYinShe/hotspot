package com.hotspot.app.ui.screens

import android.widget.Toast
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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.R
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.GroupApplicationItem
import com.hotspot.app.data.model.GroupInviteItem
import com.hotspot.app.data.model.GroupItem
import com.hotspot.app.data.model.MessageBrief
import com.hotspot.app.data.model.Page
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.Badge
import com.hotspot.app.ui.components.BadgeTone
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.EmptyStateIconTone
import com.hotspot.app.ui.components.HeaderIconButton
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.InlineLoading
import com.hotspot.app.ui.components.ListEnd
import com.hotspot.app.ui.components.ListGroup
import com.hotspot.app.ui.components.ListRow
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SearchInput
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemCard
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTextStyles
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import kotlinx.coroutines.launch

/** 列表距底 240px 内自动续载（滚动容器版分页语义，与 InfiniteScrollHandler 同阈值语义）。 */
private val LoadMoreThreshold = 240

/**
 * 我的群组（推入页 me/groups，设计稿 87:3219）：
 * 数据走 groups/fetch_joined（成员视角，含 viewer_state 置顶与 last_message 预览）；
 * 搜索框只对「已加载列表」按群名/标签本地过滤（首版无服务端群组搜索注入）。
 * 行点击进入群详情（group/{groupId}，设计稿 87:1696）。
 */
@Composable
fun MeGroupsScreen(
    onBack: () -> Unit,
    onCreateGroup: () -> Unit,
    onOpenGroup: (String) -> Unit,
) {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    val profileRepository = AppContainer.profileRepository
    val scope = rememberCoroutineScope()
    val state = remember { PagedListState<GroupItem>() }
    val scrollState = rememberScrollState()
    var query by remember { mutableStateOf("") }
    var invites by remember { mutableStateOf<List<GroupInviteItem>>(emptyList()) }
    var applications by remember { mutableStateOf<List<GroupApplicationItem>>(emptyList()) }
    var inviteBusyId by remember { mutableStateOf<String?>(null) }
    var cancelBusyId by remember { mutableStateOf<String?>(null) }

    val loader: suspend (String?) -> ApiResult<Page<GroupItem>> = { cursor ->
        profileRepository.joinedGroups(cursor)
    }

    suspend fun loadInvites() {
        when (val result = AppContainer.groupRepository.receivedInvites()) {
            is ApiResult.Ok -> invites = result.data.items
            is ApiResult.Err -> Unit
        }
    }

    suspend fun loadApplications() {
        when (val result = AppContainer.groupRepository.submittedApplications()) {
            is ApiResult.Ok -> applications = result.data.items
            is ApiResult.Err -> Unit
        }
    }

    LaunchedEffect(Unit) {
        state.initialLoad(loader)
        loadInvites()
        loadApplications()
    }

    // 触底续载：内容不足一屏（maxValue=0）时也继续补页，直到到底或撑满视口。
    val shouldLoadMore by remember {
        derivedStateOf {
            scrollState.maxValue == 0 || scrollState.value >= scrollState.maxValue - LoadMoreThreshold
        }
    }
    LaunchedEffect(shouldLoadMore, state.items.size, state.endReached) {
        if (shouldLoadMore) state.loadMore(loader)
    }

    val loaded = state.items
    val keyword = query.trim()
    val visible = if (keyword.isEmpty()) {
        loaded
    } else {
        loaded.filter { item ->
            item.group.name.contains(keyword, ignoreCase = true) ||
                item.tags.any { it.contains(keyword, ignoreCase = true) }
        }
    }
    val onlineAvailable = loaded.any { it.onlineMembersCount != null }
    val onlineTotal = loaded.sumOf { it.onlineMembersCount ?: 0 }

    Column(Modifier.fillMaxSize()) {
        PageHeader(
            title = "我的群组",
            onBack = onBack,
            actions = {
                HeaderIconButton(
                    icon = Icons.Filled.Add,
                    contentDescription = "创建群组",
                    onClick = onCreateGroup,
                )
            },
        )

        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = Spacing.lg),
        ) {
            Text(
                text = buildString {
                    append("共 ")
                    append(loaded.size)
                    append(" 个群组")
                    if (onlineAvailable) {
                        append(" · ")
                        append(onlineTotal)
                        append(" 人在线")
                    }
                },
                style = HotSpotTextStyles.caption,
                color = colors.textSecondary,
                modifier = Modifier.padding(top = Spacing.xs),
            )
            Spacer(Modifier.height(Spacing.sm))
            SearchInput(
                value = query,
                onValueChange = { query = it },
                placeholder = "搜索我的群组…",
            )
            if (invites.isNotEmpty()) {
                Spacer(Modifier.height(Spacing.md))
                SystemCard(contentPadding = PaddingValues(Spacing.md)) {
                    Text(
                        text = "入群邀请（" + invites.size + "）",
                        style = MaterialTheme.typography.titleSmall,
                        color = colors.textPrimary,
                    )
                    invites.forEach { invite ->
                        Spacer(Modifier.height(Spacing.sm))
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Avatar(
                                name = invite.group.name,
                                imageUrl = invite.group.avatarUrl,
                                size = Dimens.avatarSm,
                            )
                            Column(Modifier.weight(1f).padding(start = Spacing.sm)) {
                                Text(
                                    text = invite.group.name.ifBlank { "未命名群组" },
                                    style = MaterialTheme.typography.bodyMedium,
                                    color = colors.textPrimary,
                                )
                                Text(
                                    text = invite.inviter.nickname.ifBlank { "群主" } + " 邀请你加入",
                                    style = HotSpotTextStyles.caption,
                                    color = colors.textSecondary,
                                )
                            }
                        }
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.End,
                        ) {
                            SystemButton(
                                text = "拒绝",
                                onClick = {
                                    if (inviteBusyId != null) return@SystemButton
                                    scope.launch {
                                        inviteBusyId = invite.inviteId
                                        when (val result = AppContainer.groupRepository.rejectInvite(invite.inviteId)) {
                                            is ApiResult.Ok -> {
                                                invites = invites.filterNot { it.inviteId == invite.inviteId }
                                                Toast.makeText(context, "已拒绝邀请", Toast.LENGTH_SHORT).show()
                                            }
                                            is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                                        }
                                        inviteBusyId = null
                                    }
                                },
                                variant = ButtonVariant.SECONDARY,
                                size = ButtonSize.SM,
                                enabled = inviteBusyId == null,
                            )
                            Spacer(Modifier.size(Spacing.sm))
                            SystemButton(
                                text = "接受",
                                onClick = {
                                    if (inviteBusyId != null) return@SystemButton
                                    scope.launch {
                                        inviteBusyId = invite.inviteId
                                        when (val result = AppContainer.groupRepository.acceptInvite(invite.inviteId)) {
                                            is ApiResult.Ok -> {
                                                invites = invites.filterNot { it.inviteId == invite.inviteId }
                                                Toast.makeText(context, "已加入群组", Toast.LENGTH_SHORT).show()
                                                state.initialLoad(loader)
                                            }
                                            is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                                        }
                                        inviteBusyId = null
                                    }
                                },
                                variant = ButtonVariant.PRIMARY,
                                size = ButtonSize.SM,
                                enabled = inviteBusyId == null,
                            )
                        }
                    }
                }
            }
            if (applications.isNotEmpty()) {
                Spacer(Modifier.height(Spacing.md))
                SystemCard(contentPadding = PaddingValues(Spacing.md)) {
                    Text(
                        text = "我的入群申请（" + applications.size + "）",
                        style = MaterialTheme.typography.titleSmall,
                        color = colors.textPrimary,
                    )
                    applications.forEach { application ->
                        Spacer(Modifier.height(Spacing.sm))
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Avatar(
                                name = application.group.name,
                                imageUrl = application.group.avatarUrl,
                                size = Dimens.avatarSm,
                            )
                            Column(Modifier.weight(1f).padding(start = Spacing.sm)) {
                                Text(
                                    text = application.group.name.ifBlank { "未命名群组" },
                                    style = MaterialTheme.typography.bodyMedium,
                                    color = colors.textPrimary,
                                )
                                Text(
                                    text = "审核中，等待群管理员处理",
                                    style = HotSpotTextStyles.caption,
                                    color = colors.textSecondary,
                                )
                            }
                            SystemButton(
                                text = "撤回",
                                onClick = {
                                    if (cancelBusyId != null) return@SystemButton
                                    scope.launch {
                                        cancelBusyId = application.applicationId
                                        when (val result = AppContainer.groupRepository.cancelApplication(
                                            groupId = application.group.groupId,
                                            applicationId = application.applicationId,
                                        )) {
                                            is ApiResult.Ok -> {
                                                applications = applications.filterNot { it.applicationId == application.applicationId }
                                                Toast.makeText(context, "申请已撤回", Toast.LENGTH_SHORT).show()
                                            }
                                            is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                                        }
                                        cancelBusyId = null
                                    }
                                },
                                variant = ButtonVariant.SECONDARY,
                                size = ButtonSize.SM,
                                enabled = cancelBusyId == null,
                            )
                        }
                    }
                }
            }
            Spacer(Modifier.height(Spacing.md))
        }

        when {
            state.loading && loaded.isEmpty() -> LoadingState(message = "正在加载群组…")
            state.error != null && loaded.isEmpty() -> ErrorState(
                message = state.error ?: "加载失败",
                onRetry = { scope.launch { state.initialLoad(loader) } },
            )
            loaded.isEmpty() -> EmptyState(
                title = "还没有加入群组",
                description = "去附近群组发现感兴趣的小组，加入后就能在这里快速进入。",
                iconTone = EmptyStateIconTone.ACCENT_SOFT,
                iconContent = {
                    Icon(
                        painter = painterResource(R.drawable.ic_people),
                        contentDescription = null,
                        modifier = Modifier.size(Dimens.iconLg),
                        tint = colors.accent,
                    )
                },
            )
            visible.isEmpty() -> EmptyState(
                title = "没有匹配的群组",
                description = "换个关键词试试，或清空搜索查看全部群组。",
                icon = Icons.Filled.Search,
            )
            else -> Box(Modifier.weight(1f)) {
                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .verticalScroll(scrollState)
                        .padding(horizontal = Spacing.lg),
                ) {
                    ListGroup {
                        visible.forEach { item ->
                            GroupJoinedRow(item, onOpenGroup = onOpenGroup)
                        }
                    }
                    when {
                        state.loadingMore -> InlineLoading()
                        state.error != null -> ErrorState(
                            message = state.error ?: "加载失败",
                            onRetry = { scope.launch { state.loadMore(loader) } },
                        )
                        state.endReached -> ListEnd()
                    }
                    Spacer(Modifier.height(Spacing.xl))
                }
            }
        }
    }
}

/**
 * 群组行（与 ListRow 行高一致）：头像 + 群名 + 「成员数 · 最近消息预览」；
 * viewer_state.is_pinned 时右侧 20 档「置顶」标记；整行点击进入群详情。
 */
@Composable
private fun GroupJoinedRow(
    item: GroupItem,
    onOpenGroup: (String) -> Unit,
) {
    val pinned = item.viewerState?.isPinned == true
    ListRow(
        leading = {
            Avatar(
                name = item.group.name,
                imageUrl = item.group.avatarUrl,
                size = Dimens.avatarSm,
            )
        },
        title = item.group.name.ifBlank { "未命名群组" },
        subtitle = item.group.membersCount.toString() + " 成员 · " + messagePreview(item.lastMessage),
        trailing = if (pinned) {
            { Badge(text = "置顶", tone = BadgeTone.NEUTRAL) }
        } else {
            null
        },
        onClick = { onOpenGroup(item.group.groupId) },
    )
}

/** 最近消息预览：与鸿蒙/Web 同口径（文本取 content，媒体类型用方括号占位）。 */
private fun messagePreview(brief: MessageBrief?): String {
    if (brief == null) return "还没有消息"
    val content = brief.content?.trim().orEmpty()
    return when (brief.type) {
        "text" -> content.ifEmpty { "新消息" }
        "image" -> "[图片]"
        "video" -> "[视频]"
        "audio" -> "[语音]"
        "file" -> "[文件]"
        "share" -> "[分享]"
        "system" -> "系统消息"
        else -> "[消息]"
    }
}
