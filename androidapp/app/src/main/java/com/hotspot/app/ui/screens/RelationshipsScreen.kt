package com.hotspot.app.ui.screens

import android.widget.Toast
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
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
import androidx.compose.material.icons.filled.MailOutline
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.Page
import com.hotspot.app.data.model.RelatedUserItem
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.InlineLoading
import com.hotspot.app.ui.components.ListEnd
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.NearbyFollowButton
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SearchInput
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemCard
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import kotlinx.coroutines.launch

private val RELATION_TABS = listOf("好友", "关注", "粉丝")

/**
 * 关系页（设计稿 87:3300）：好友 / 关注 / 粉丝三个 tab（带计数与下划线选中态）+ 搜索 + 关系列表。
 * - 好友：右侧「发消息」进私聊；
 * - 关注：右侧关注两态，取关后该行从列表移除；
 * - 粉丝：右侧关注两态（回关）。
 * 黑名单 tab 依赖拉黑功能（set_block），Android 首版未实现，暂不放入口（代码内 TODO）。
 */
@Composable
fun RelationshipsScreen(
    initialTab: Int,
    onBack: () -> Unit,
    onOpenChat: (userId: String) -> Unit,
    onOpenUser: (userId: String) -> Unit,
) {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var tabIndex by remember { mutableIntStateOf(initialTab.coerceIn(0, RELATION_TABS.lastIndex)) }
    var query by remember { mutableStateOf("") }
    var counts by remember { mutableStateOf(listOf(0, 0, 0)) }

    LaunchedEffect(Unit) {
        when (val result = AppContainer.profileRepository.account()) {
            is ApiResult.Ok -> {
                val stats = result.data.stats
                counts = listOf(
                    stats?.friendsCount ?: 0,
                    stats?.followingCount ?: 0,
                    stats?.followersCount ?: 0,
                )
            }
            is ApiResult.Err -> Unit
        }
    }

    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "关系", onBack = onBack)
        RelationshipTabs(
            selectedIndex = tabIndex,
            counts = counts,
            onSelected = { tabIndex = it },
        )
        Spacer(Modifier.height(Spacing.md))
        SearchInput(
            value = query,
            onValueChange = { query = it },
            placeholder = "搜索" + RELATION_TABS[tabIndex] + "…",
            modifier = Modifier.padding(horizontal = Spacing.lg),
        )
        Spacer(Modifier.height(Spacing.md))
        key(tabIndex) {
            RelationshipListContent(
                tabIndex = tabIndex,
                query = query,
                onOpenChat = onOpenChat,
                onOpenUser = onOpenUser,
                onError = { message -> Toast.makeText(context, message, Toast.LENGTH_SHORT).show() },
                scope = scope,
            )
        }
    }
}

@Composable
private fun RelationshipTabs(
    selectedIndex: Int,
    counts: List<Int>,
    onSelected: (Int) -> Unit,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = Spacing.lg),
        horizontalArrangement = Arrangement.spacedBy(Spacing.lg2),
    ) {
        RELATION_TABS.forEachIndexed { index, label ->
            val selected = index == selectedIndex
            Column(
                modifier = Modifier
                    .clip(androidx.compose.foundation.shape.RoundedCornerShape(8.dp))
                    .clickable { onSelected(index) }
                    .padding(vertical = Spacing.sm),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = label,
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Normal,
                        color = if (selected) colors.textPrimary else colors.textSecondary,
                    )
                    if (counts.getOrNull(index) != null && counts[index] > 0) {
                        Spacer(Modifier.width(Spacing.xs))
                        Text(
                            text = counts[index].toString(),
                            style = MaterialTheme.typography.labelSmall,
                            color = if (selected) colors.accent else colors.textSecondary,
                        )
                    }
                }
                Spacer(Modifier.height(Spacing.xs))
                Box(
                    modifier = Modifier
                        .size(width = 18.dp, height = 2.dp)
                        .clip(CircleShape)
                        .background(if (selected) colors.accent else androidx.compose.ui.graphics.Color.Transparent),
                )
            }
        }
    }
}

@Composable
private fun RelationshipListContent(
    tabIndex: Int,
    query: String,
    onOpenChat: (String) -> Unit,
    onOpenUser: (String) -> Unit,
    onError: (String) -> Unit,
    scope: kotlinx.coroutines.CoroutineScope,
) {
    val colors = HotSpotTheme.colors
    val state = rememberPagedListState<RelatedUserItem>()
    val listState = rememberLazyListState()
    val selfId = AppContainer.tokenStore.session.value?.userId.orEmpty()
    val followStates = remember { mutableStateMapOf<String, Boolean>() }
    val followBusy = remember { mutableStateMapOf<String, Boolean>() }
    val removedIds = remember { mutableStateMapOf<String, Boolean>() }

    val loader: suspend (String?) -> ApiResult<Page<RelatedUserItem>> = { cursor ->
        when (tabIndex) {
            0 -> AppContainer.profileRepository.friends(selfId, cursor)
            1 -> AppContainer.profileRepository.following(selfId, cursor)
            else -> AppContainer.profileRepository.followers(selfId, cursor)
        }
    }

    LaunchedEffect(Unit) { state.initialLoad(loader) }
    InfiniteScrollHandler(state, listState, loader)

    // 后端 fetch_following/fetch_followers 未回填 user.relation（实测 relation=null），
    // 端上按列表语义兜底：关注列表默认「已关注」，粉丝列表按 relation 判断是否已回关。
    fun effectiveFollowing(item: RelatedUserItem): Boolean =
        followStates[item.user.user.userId]
            ?: if (tabIndex == 1) true else (item.user.relation?.isFollowing ?: false)

    fun toggleFollow(item: RelatedUserItem) {
        val userId = item.user.user.userId
        if (userId.isBlank() || followBusy[userId] == true) return
        val current = effectiveFollowing(item)
        val next = !current
        followStates[userId] = next
        followBusy[userId] = true
        scope.launch {
            when (val result = AppContainer.profileRepository.setFollow(userId, next)) {
                is ApiResult.Ok -> {
                    // 关注 tab 取关后不再属于该列表：从当前视图移除
                    if (!next && tabIndex == 1) removedIds[userId] = true
                }
                is ApiResult.Err -> {
                    followStates[userId] = current
                    onError(result.error.userMessage)
                }
            }
            followBusy[userId] = false
        }
    }

    val keyword = query.trim()
    val visible = state.items.filter { item ->
        val userId = item.user.user.userId
        removedIds[userId] != true && (keyword.isBlank() || item.user.user.nickname.contains(keyword, ignoreCase = true))
    }

    when {
        state.loading && state.items.isEmpty() -> LoadingState(message = "正在加载" + RELATION_TABS[tabIndex] + "…")
        state.error != null && state.items.isEmpty() -> ErrorState(
            message = state.error ?: "加载失败",
            onRetry = { scope.launch { state.initialLoad(loader) } },
        )
        state.items.isEmpty() -> EmptyState(
            title = when (tabIndex) {
                0 -> "还没有好友"
                1 -> "还没有关注的人"
                else -> "还没有粉丝"
            },
            description = when (tabIndex) {
                0 -> "互相关注后成为好友"
                1 -> "去附近或搜索里发现感兴趣的人"
                else -> "在附近或瞬间里被关注后会出现在这里"
            },
        )
        visible.isEmpty() -> EmptyState(title = "没有匹配的结果", description = "换个关键词试试")
        else -> LazyColumn(
            state = listState,
            modifier = Modifier.fillMaxSize(),
            contentPadding = androidx.compose.foundation.layout.PaddingValues(
                start = Spacing.lg,
                end = Spacing.lg,
                bottom = Spacing.xxl,
            ),
        ) {
            item {
                SystemCard(contentPadding = androidx.compose.foundation.layout.PaddingValues(vertical = Spacing.xs)) {
                    visible.forEachIndexed { index, item ->
                        RelationshipRow(
                            item = item,
                            tabIndex = tabIndex,
                            following = effectiveFollowing(item),
                            followBusy = followBusy[item.user.user.userId] == true,
                            onToggleFollow = { toggleFollow(item) },
                            onOpenChat = { onOpenChat(item.user.user.userId) },
                            onOpenUser = { onOpenUser(item.user.user.userId) },
                        )
                        if (index != visible.lastIndex) {
                            Box(
                                Modifier
                                    .padding(horizontal = Spacing.md)
                                    .height(1.dp)
                                    .fillMaxWidth()
                                    .background(colors.borderSubtle),
                            )
                        }
                    }
                }
            }
            item {
                when {
                    state.loadingMore -> InlineLoading()
                    state.endReached -> ListEnd()
                    else -> Box(Modifier.height(1.dp))
                }
            }
        }
    }
}

@Composable
private fun RelationshipRow(
    item: RelatedUserItem,
    tabIndex: Int,
    following: Boolean,
    followBusy: Boolean,
    onToggleFollow: () -> Unit,
    onOpenChat: () -> Unit,
    onOpenUser: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    val user = item.user
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onOpenUser)
            .padding(horizontal = Spacing.md, vertical = Spacing.sm),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Avatar(name = user.user.nickname, imageUrl = user.user.avatarUrl, size = Dimens.avatarSm)
        Spacer(Modifier.width(Spacing.md))
        Column(Modifier.weight(1f)) {
            Text(
                text = user.user.nickname.ifBlank { "匿名用户" },
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Medium,
                color = colors.textPrimary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                text = item.remark?.takeIf { it.isNotBlank() } ?: user.bio?.takeIf { it.isNotBlank() } ?: "热点邻居",
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
        Text(
            text = if (user.presenceStatus == "online") "在线" else "离线",
            style = MaterialTheme.typography.labelSmall,
            color = if (user.presenceStatus == "online") colors.online else colors.textSecondary,
        )
        Spacer(Modifier.width(Spacing.sm))
        if (tabIndex == 0) {
            SystemButton(
                text = "发消息",
                onClick = onOpenChat,
                variant = ButtonVariant.SOFT,
                size = ButtonSize.SM,
                leadingIcon = {
                    Icon(
                        imageVector = Icons.Filled.MailOutline,
                        contentDescription = null,
                        modifier = Modifier.size(14.dp),
                        tint = colors.textAccentOnSoft,
                    )
                },
            )
        } else {
            NearbyFollowButton(following = following, busy = followBusy, onClick = onToggleFollow)
        }
    }
}
