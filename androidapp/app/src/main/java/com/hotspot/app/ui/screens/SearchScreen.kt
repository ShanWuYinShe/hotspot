package com.hotspot.app.ui.screens

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.GroupItem
import com.hotspot.app.data.model.MomentItem
import com.hotspot.app.data.model.Page
import com.hotspot.app.data.model.UserItem
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.Badge
import com.hotspot.app.ui.components.BadgeTone
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.ListGroup
import com.hotspot.app.ui.components.ListRow
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SystemInput
import com.hotspot.app.ui.components.TabPills
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.TimeFormat
import kotlinx.coroutines.launch

private val SEARCH_TABS = listOf("用户", "瞬间", "群组")

/**
 * 全局搜索：用户 / 瞬间 / 群组各自独立分页（后端不提供聚合端点，客户端拼装）。
 * 首版只取第一页；关键字为空时展示输入引导。
 */
@Composable
fun SearchScreen(
    onBack: () -> Unit,
    onOpenGroup: (String) -> Unit,
    onOpenUser: (String) -> Unit,
    onOpenMoment: (String) -> Unit,
) {
    val colors = HotSpotTheme.colors
    var query by remember { mutableStateOf("") }
    var submitted by remember { mutableStateOf("") }
    var tabIndex by remember { mutableIntStateOf(0) }

    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "搜索", onBack = onBack)
        Column(Modifier.padding(horizontal = Spacing.lg)) {
            SystemInput(
                value = query,
                onValueChange = { query = it },
                placeholder = "搜索用户 / 瞬间 / 群组",
                label = "关键字",
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
                keyboardActions = KeyboardActions(onSearch = { submitted = query.trim() }),
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(Spacing.md))
            TabPills(
                options = SEARCH_TABS,
                selectedIndex = tabIndex,
                onSelected = { tabIndex = it },
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(Spacing.md))
        }
        when (tabIndex) {
            0 -> UserResults(query = submitted, onOpenUser = onOpenUser)
            1 -> MomentResults(query = submitted, onOpenMoment = onOpenMoment)
            else -> GroupResults(query = submitted, onOpenGroup = onOpenGroup)
        }
    }
}

@Composable
private fun UserResults(query: String, onOpenUser: (String) -> Unit) {
    val scope = rememberCoroutineScope()
    val state = rememberSingleLoadState<UserItem>()
    var retryKey by remember { mutableIntStateOf(0) }
    LaunchedEffect(query, retryKey) {
        if (query.isBlank()) {
            state.load { ApiResult.Ok(emptyList()) }
        } else {
            state.load {
                when (val result = AppContainer.searchRepository.searchUsers(query, null)) {
                    is ApiResult.Ok -> ApiResult.Ok(result.data.items)
                    is ApiResult.Err -> result
                }
            }
        }
    }
    SearchResultContainer(
        query = query,
        loading = state.loading,
        error = state.error,
        empty = state.items.isEmpty(),
        emptyTitle = "没有找到相关用户",
        onRetry = { retryKey++ },
    ) {
        ListGroup {
            state.items.forEach { user ->
                ListRow(
                    title = user.user.nickname.ifBlank { "匿名用户" },
                    subtitle = user.bio ?: "",
                    leading = {
                        Avatar(
                            name = user.user.nickname,
                            imageUrl = user.user.avatarUrl,
                            size = com.hotspot.app.ui.theme.Dimens.avatarSm,
                        )
                    },
                    trailing = {
                        if (user.relation?.isFollowing == true) {
                            Badge(text = "已关注", tone = BadgeTone.ACCENT)
                        }
                    },
                    onClick = { onOpenUser(user.user.userId) },
                )
            }
        }
    }
}

@Composable
private fun MomentResults(query: String, onOpenMoment: (String) -> Unit) {
    val state = rememberSingleLoadState<MomentItem>()
    var retryKey by remember { mutableIntStateOf(0) }
    LaunchedEffect(query, retryKey) {
        if (query.isBlank()) {
            state.load { ApiResult.Ok(emptyList()) }
        } else {
            state.load {
                when (val result = AppContainer.searchRepository.searchMoments(query, null)) {
                    is ApiResult.Ok -> ApiResult.Ok(result.data.items)
                    is ApiResult.Err -> result
                }
            }
        }
    }
    SearchResultContainer(
        query = query,
        loading = state.loading,
        error = state.error,
        empty = state.items.isEmpty(),
        emptyTitle = "没有找到相关瞬间",
        onRetry = { retryKey++ },
    ) {
        ListGroup {
            state.items.forEach { moment ->
                ListRow(
                    title = moment.description.ifBlank { "无描述" },
                    subtitle = TimeFormat.relative(moment.createdAt),
                    leading = {
                        Avatar(
                            name = moment.user.nickname,
                            imageUrl = moment.images.firstOrNull()?.url ?: moment.user.avatarUrl,
                            size = com.hotspot.app.ui.theme.Dimens.avatarSm,
                            shape = AppShapes.compact,
                        )
                    },
                    onClick = { onOpenMoment(moment.momentId) },
                )
            }
        }
    }
}

@Composable
private fun GroupResults(
    query: String,
    onOpenGroup: (String) -> Unit,
) {
    val state = rememberSingleLoadState<GroupItem>()
    var retryKey by remember { mutableIntStateOf(0) }
    LaunchedEffect(query, retryKey) {
        if (query.isBlank()) {
            state.load { ApiResult.Ok(emptyList()) }
        } else {
            state.load {
                when (val result = AppContainer.searchRepository.searchGroups(query, null)) {
                    is ApiResult.Ok -> ApiResult.Ok(result.data.items)
                    is ApiResult.Err -> result
                }
            }
        }
    }
    SearchResultContainer(
        query = query,
        loading = state.loading,
        error = state.error,
        empty = state.items.isEmpty(),
        emptyTitle = "没有找到相关群组",
        onRetry = { retryKey++ },
    ) {
        ListGroup {
            state.items.forEach { group ->
                ListRow(
                    title = group.group.name.ifBlank { "未命名群组" },
                    subtitle = group.group.membersCount.toString() + " 位成员",
                    leading = {
                        Avatar(
                            name = group.group.name,
                            imageUrl = group.group.avatarUrl,
                            size = com.hotspot.app.ui.theme.Dimens.avatarSm,
                            shape = AppShapes.compact,
                        )
                    },
                    onClick = { onOpenGroup(group.group.groupId) },
                )
            }
        }
    }
}

@Composable
private fun SearchResultContainer(
    query: String,
    loading: Boolean,
    error: String?,
    empty: Boolean,
    emptyTitle: String,
    onRetry: () -> Unit,
    content: @Composable () -> Unit,
) {
    val colors = HotSpotTheme.colors
    when {
        query.isBlank() -> EmptyState(
            title = "输入关键字开始搜索",
            description = "支持昵称、简介、瞬间描述与群组名称",
        )
        loading -> LoadingState()
        error != null -> ErrorState(message = error, onRetry = onRetry)
        empty -> EmptyState(title = emptyTitle, description = "换个关键字试试")
        else -> Column(Modifier.padding(horizontal = Spacing.lg)) {
            LazyColumn(Modifier.fillMaxSize()) {
                item { content() }
            }
        }
    }
}
