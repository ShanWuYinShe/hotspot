package com.hotspot.app.ui.screens

import android.widget.Toast
import androidx.compose.foundation.clickable
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
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.itemsIndexed
import androidx.compose.foundation.lazy.grid.rememberLazyGridState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Place
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.getValue
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.Page
import com.hotspot.app.data.model.UserItem
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.HeaderIconButton
import com.hotspot.app.ui.components.InlineLoading
import com.hotspot.app.ui.components.ListEnd
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.NearbyFilterChips
import com.hotspot.app.ui.components.NearbyLocationChip
import com.hotspot.app.ui.components.NearbyUserCard
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import kotlinx.coroutines.launch

/** 附近的人筛选（设计稿 87:4192 顶部胶囊行）。 */
private val PEOPLE_FILTERS = listOf("全部", "摄影", "户外", "旅行", "音乐", "美食", "阅读")
private val PEOPLE_SORTS = listOf("距离最近" to "distance", "最新加入" to "newest")

/**
 * 附近的人（设计稿 87:4192）：标题/定位 → 兴趣筛选胶囊 → 排序 → 两列用户卡（关注两态）。
 * 关注走 users/set_follow，本地乐观更新，失败回滚并 Toast 服务端 message。
 */
@Composable
fun NearbyPeopleScreen(onBack: () -> Unit, onOpenSearch: () -> Unit) {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    var filterIndex by remember { mutableIntStateOf(0) }
    var sortIndex by remember { mutableIntStateOf(0) }

    Column(Modifier.fillMaxSize()) {
        PageHeader(
            title = "附近的人",
            onBack = onBack,
            actions = {
                HeaderIconButton(icon = Icons.Filled.Search, contentDescription = "搜索", onClick = onOpenSearch)
                HeaderIconButton(
                    icon = Icons.Filled.Place,
                    contentDescription = "附近地图",
                    onClick = { Toast.makeText(context, "地图原生渲染 TODO（首版占位）", Toast.LENGTH_SHORT).show() },
                )
            },
        )
        val tag = PEOPLE_FILTERS.getOrNull(filterIndex)?.takeIf { it != "全部" }
        val sortBy = PEOPLE_SORTS[sortIndex].second
        key(filterIndex, sortIndex) {
            NearbyPeopleContent(
                tag = tag,
                sortBy = sortBy,
                filterIndex = filterIndex,
                sortIndex = sortIndex,
                onFilterSelected = { filterIndex = it },
                onSortSelected = { sortIndex = it },
            )
        }
    }
}

@Composable
private fun NearbyPeopleContent(
    tag: String?,
    sortBy: String,
    filterIndex: Int,
    sortIndex: Int,
    onFilterSelected: (Int) -> Unit,
    onSortSelected: (Int) -> Unit,
) {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val state = rememberPagedListState<UserItem>()
    val gridState = rememberLazyGridState()
    val followStates = remember { mutableStateMapOf<String, Boolean>() }
    val followBusy = remember { mutableStateMapOf<String, Boolean>() }
    var locationLabel by remember { mutableStateOf("附近 · 5 km 内") }
    val selfId = AppContainer.tokenStore.session.value?.userId
    val loader: suspend (String?) -> ApiResult<Page<UserItem>> = { cursor ->
        AppContainer.nearbyRepository.nearbyPeople(
            cursor,
            tag,
            sortBy,
            AppContainer.defaultLocation()?.point,
        )
    }

    LaunchedEffect(Unit) {
        AppContainer.defaultLocation()?.let { location ->
            if (location.locationName.isNotBlank()) locationLabel = location.locationName + " · 5 km 内"
        }
        state.initialLoad(loader)
    }
    InfiniteScrollHandler(state, gridState, loader)

    fun toggleFollow(user: UserItem) {
        val userId = user.user.userId
        if (userId.isBlank() || followBusy[userId] == true) return
        val current = followStates[userId] ?: (user.relation?.isFollowing ?: false)
        val next = !current
        followStates[userId] = next
        followBusy[userId] = true
        scope.launch {
            when (val result = AppContainer.profileRepository.setFollow(userId, next)) {
                is ApiResult.Ok -> Unit
                is ApiResult.Err -> {
                    followStates[userId] = current
                    Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                }
            }
            followBusy[userId] = false
        }
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(top = Spacing.xs),
    ) {
        Row(Modifier.padding(horizontal = Spacing.lg), verticalAlignment = Alignment.CenterVertically) {
            val visibleCount = state.items.count { it.user.userId != selfId }
            val countText = if (visibleCount > 0) " · " + visibleCount + " 人" else ""
            NearbyLocationChip(locationLabel + countText)
        }
        Spacer(Modifier.height(Spacing.sm))
        NearbyFilterChips(
            options = PEOPLE_FILTERS,
            selectedIndex = filterIndex,
            onSelected = onFilterSelected,
            modifier = Modifier.padding(horizontal = Spacing.lg),
        )
        Spacer(Modifier.height(Spacing.sm2))
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = Spacing.lg),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Spacer(Modifier.weight(1f))
            SortMenu(
                label = PEOPLE_SORTS[sortIndex].first,
                options = PEOPLE_SORTS.map { it.first },
                selectedIndex = sortIndex,
                onSelected = onSortSelected,
            )
        }
        Spacer(Modifier.height(Spacing.xs))
        when {
            state.loading && state.items.isEmpty() -> LoadingState(message = "正在寻找附近的人…")
            state.error != null && state.items.isEmpty() -> ErrorState(
                message = state.error ?: "加载失败",
                onRetry = { scope.launch { state.initialLoad(loader) } },
            )
            state.items.count { it.user.userId != selfId } == 0 ->
                EmptyState(title = "附近暂时没有人", description = "换个兴趣或稍后再来看看")
            else -> LazyVerticalGrid(
                columns = GridCells.Fixed(2),
                state = gridState,
                modifier = Modifier.fillMaxSize(),
                contentPadding = PaddingValues(
                    start = Spacing.lg,
                    end = Spacing.lg,
                    bottom = Spacing.xxl,
                ),
                horizontalArrangement = Arrangement.spacedBy(Spacing.md),
                verticalArrangement = Arrangement.spacedBy(Spacing.md),
            ) {
                itemsIndexed(
                    state.items.filter { it.user.userId != selfId },
                    key = { _, user -> user.user.userId },
                ) { _, user ->
                    NearbyUserCard(
                        user = user,
                        following = followStates[user.user.userId] ?: (user.relation?.isFollowing ?: false),
                        followBusy = followBusy[user.user.userId] == true,
                        onToggleFollow = { toggleFollow(user) },
                    )
                }
                item(span = { GridItemSpan(maxLineSpan) }) {
                    when {
                        state.loadingMore -> InlineLoading()
                        state.endReached -> ListEnd()
                        else -> Box(Modifier.height(1.dp))
                    }
                }
            }
        }
    }
}

/** 排序下拉（设计稿右上「排序 ⌄」）。 */
@Composable
fun SortMenu(
    label: String,
    options: List<String>,
    selectedIndex: Int,
    onSelected: (Int) -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    var expanded by remember { mutableStateOf(false) }
    Box(modifier) {
        Row(
            modifier = Modifier
                .padding(vertical = Spacing.xs)
                .clickable { expanded = true },
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = "排序 · " + label,
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
            )
            Text(
                text = " ⌄",
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
            )
        }
        DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            options.forEachIndexed { index, option ->
                DropdownMenuItem(
                    text = {
                        Text(
                            text = option,
                            style = MaterialTheme.typography.bodyMedium,
                            color = if (index == selectedIndex) colors.accent else colors.textPrimary,
                        )
                    },
                    onClick = {
                        expanded = false
                        onSelected(index)
                    },
                )
            }
        }
    }
}
