package com.hotspot.app.ui.screens

import android.content.Context
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.GroupItem
import com.hotspot.app.data.model.MomentItem
import com.hotspot.app.data.model.Page
import com.hotspot.app.data.model.UserItem
import com.hotspot.app.data.repo.SearchSuggestion
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
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.TimeFormat
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

private val SEARCH_TABS = listOf("用户", "瞬间", "群组")

/**
 * 全局搜索：用户 / 瞬间 / 群组各自独立分页（后端不提供聚合端点，客户端拼装）。
 * 首版只取第一页；关键字为空时展示最近搜索与输入引导。
 */
@Composable
fun SearchScreen(
    onBack: () -> Unit,
    onOpenGroup: (String) -> Unit,
    onOpenUser: (String) -> Unit,
    onOpenMoment: (String) -> Unit,
) {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    val keyboard = LocalSoftwareKeyboardController.current
    val prefs = remember { context.getSharedPreferences("search_history", Context.MODE_PRIVATE) }
    var recent by remember {
        mutableStateOf(
            prefs.getString("queries", "").orEmpty().split('\n').map { it.trim() }.filter { it.isNotBlank() },
        )
    }
    var query by remember { mutableStateOf("") }
    var submitted by remember { mutableStateOf("") }
    var tagFilter by remember { mutableStateOf<String?>(null) }
    var tabIndex by remember { mutableIntStateOf(0) }
    var suggestions by remember { mutableStateOf<List<SearchSuggestion>>(emptyList()) }

    fun rememberSearch(value: String) {
        val term = value.trim()
        if (term.isBlank()) return
        tagFilter = null
        recent = (listOf(term) + recent.filterNot { it == term }).take(5)
        prefs.edit().putString("queries", recent.joinToString("\n")).apply()
        query = term
        submitted = term
    }

    LaunchedEffect(query) {
        val prefix = query.trim()
        if (prefix.isBlank()) {
            suggestions = emptyList()
            return@LaunchedEffect
        }
        delay(300)
        val point = AppContainer.defaultLocation()?.point
        when (val result = AppContainer.feedRepository.suggest(prefix, point)) {
            is ApiResult.Ok -> suggestions = result.data
            is ApiResult.Err -> suggestions = emptyList()
        }
    }

    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "搜索", onBack = onBack)
        Column(Modifier.padding(horizontal = Spacing.lg)) {
            SystemInput(
                value = query,
                onValueChange = {
                    query = it
                    tagFilter = null
                    if (it.isBlank()) submitted = ""
                },
                placeholder = "搜索用户 / 瞬间 / 群组",
                label = "关键字",
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
                keyboardActions = KeyboardActions(onSearch = { rememberSearch(query) }),
                modifier = Modifier.fillMaxWidth(),
            )
            if (query.isBlank() && recent.isNotEmpty()) {
                Spacer(Modifier.height(Spacing.sm))
                Text(
                    text = "最近搜索",
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
                Spacer(Modifier.height(Spacing.xs))
                Row(
                    modifier = Modifier.horizontalScroll(rememberScrollState()),
                    horizontalArrangement = Arrangement.spacedBy(Spacing.sm),
                ) {
                    recent.forEach { term ->
                        Text(
                            text = term,
                            style = MaterialTheme.typography.labelMedium,
                            color = colors.textSecondary,
                            modifier = Modifier
                                .clip(AppShapes.pill)
                                .background(colors.subtle)
                                .clickable { rememberSearch(term) }
                                .padding(horizontal = Spacing.md, vertical = Spacing.xs),
                        )
                    }
                }
            }
            if (query.isNotBlank() && query.trim() != submitted && suggestions.isNotEmpty()) {
                Spacer(Modifier.height(Spacing.sm))
                ListGroup {
                    suggestions.take(5).forEach { suggestion ->
                        ListRow(
                            title = suggestion.title,
                            subtitle = suggestion.subtitle.ifBlank { suggestion.type },
                            leading = {
                                Avatar(
                                    name = suggestion.title,
                                    imageUrl = suggestion.avatarUrl,
                                    size = Dimens.avatarSm,
                                    shape = AppShapes.iconTile,
                                )
                            },
                            trailing = { Badge(text = suggestion.type, tone = BadgeTone.NEUTRAL) },
                            onClick = {
                                if (suggestion.type == "标签") {
                                    // 标签没有详情页：回填关键词并切到瞬间结果，用 moments/search 的 tag 模式检索。
                                    rememberSearch(suggestion.id)
                                    tagFilter = suggestion.id
                                    tabIndex = 1
                                    keyboard?.hide()
                                } else {
                                    rememberSearch(query)
                                    when (suggestion.type) {
                                        "用户" -> onOpenUser(suggestion.id)
                                        "群组" -> onOpenGroup(suggestion.id)
                                        "瞬间" -> onOpenMoment(suggestion.id)
                                    }
                                }
                            },
                        )
                    }
                }
            }
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
            1 -> MomentResults(query = submitted, tag = tagFilter, onOpenMoment = onOpenMoment)
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
private fun MomentResults(query: String, tag: String?, onOpenMoment: (String) -> Unit) {
    val state = rememberSingleLoadState<MomentItem>()
    var retryKey by remember { mutableIntStateOf(0) }
    LaunchedEffect(query, tag, retryKey) {
        if (query.isBlank() && tag.isNullOrBlank()) {
            state.load { ApiResult.Ok(emptyList()) }
        } else {
            state.load {
                when (val result = AppContainer.searchRepository.searchMoments(
                    keyword = if (tag.isNullOrBlank()) query else "",
                    cursor = null,
                    tag = tag,
                )) {
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
        emptyTitle = if (tag.isNullOrBlank()) "没有找到相关瞬间" else "该标签下暂时没有瞬间",
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
