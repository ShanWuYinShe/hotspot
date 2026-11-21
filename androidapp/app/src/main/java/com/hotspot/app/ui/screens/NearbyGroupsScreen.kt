package com.hotspot.app.ui.screens

import android.widget.Toast
import androidx.compose.foundation.background
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
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.itemsIndexed
import androidx.compose.foundation.lazy.grid.rememberLazyGridState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Place
import androidx.compose.material.icons.filled.Search
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.GroupItem
import com.hotspot.app.data.model.Page
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.HeaderIconButton
import com.hotspot.app.ui.components.InlineLoading
import com.hotspot.app.ui.components.ListEnd
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.NearbyFilterChips
import com.hotspot.app.ui.components.NearbyGroupCard
import com.hotspot.app.ui.components.NearbyLocationChip
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import kotlinx.coroutines.launch

/** 附近群组筛选（设计稿 87:3798 顶部胶囊行）。 */
private val GROUP_FILTERS = listOf("全部", "摄影", "户外", "旅行", "音乐", "美食", "阅读")
private val GROUP_SORTS = listOf("距离最近" to "distance", "最新创建" to "newest")

/**
 * 附近群组（设计稿 87:3798）：标题/在线数 → 定位与群数 → 分类筛选 → 两列群卡（加入两态）。
 * 加入走 groups/enter（服务端区分直入/审批/口令；本页只处理 direct/approval，口令群提示去详情页）。
 */
@Composable
fun NearbyGroupsScreen(
    onBack: () -> Unit,
    onOpenSearch: () -> Unit,
    onOpenGroup: (groupId: String) -> Unit,
) {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    var filterIndex by remember { mutableIntStateOf(0) }
    var sortIndex by remember { mutableIntStateOf(0) }

    Column(Modifier.fillMaxSize()) {
        PageHeader(
            title = "附近群组",
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
        val tag = GROUP_FILTERS.getOrNull(filterIndex)?.takeIf { it != "全部" }
        key(filterIndex, sortIndex) {
            NearbyGroupsContent(
                tag = tag,
                sortBy = GROUP_SORTS[sortIndex].second,
                filterIndex = filterIndex,
                sortIndex = sortIndex,
                onFilterSelected = { filterIndex = it },
                onSortSelected = { sortIndex = it },
                onOpenGroup = onOpenGroup,
            )
        }
    }
}

@Composable
private fun NearbyGroupsContent(
    tag: String?,
    sortBy: String,
    filterIndex: Int,
    sortIndex: Int,
    onFilterSelected: (Int) -> Unit,
    onSortSelected: (Int) -> Unit,
    onOpenGroup: (String) -> Unit,
) {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val state = rememberPagedListState<GroupItem>()
    val gridState = rememberLazyGridState()
    val joinedStates = remember { mutableStateMapOf<String, Boolean>() }
    val pendingStates = remember { mutableStateMapOf<String, Boolean>() }
    val joinBusy = remember { mutableStateMapOf<String, Boolean>() }
    var locationLabel by remember { mutableStateOf("附近 · 5 km 内") }
    val loader: suspend (String?) -> ApiResult<Page<GroupItem>> = { cursor ->
        AppContainer.nearbyRepository.nearbyGroups(
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

    fun join(group: GroupItem) {
        val groupId = group.group.groupId
        if (groupId.isBlank() || joinBusy[groupId] == true) return
        joinBusy[groupId] = true
        scope.launch {
            when (val result = AppContainer.groupRepository.enter(groupId)) {
                is ApiResult.Ok -> {
                    if (result.data.status == "pending") pendingStates[groupId] = true else joinedStates[groupId] = true
                }
                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
            }
            joinBusy[groupId] = false
        }
    }

    val onlineCount = state.items.count { (it.onlineMembersCount ?: 0) > 0 }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(top = Spacing.xs),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = Spacing.lg),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            val countText = if (state.items.isNotEmpty()) " · " + state.items.size + " 个群" else ""
            NearbyLocationChip(locationLabel + countText)
            Spacer(Modifier.weight(1f))
            if (onlineCount > 0) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Box(
                        Modifier
                            .size(6.dp)
                            .clip(androidx.compose.foundation.shape.CircleShape)
                            .background(colors.online),
                    )
                    Spacer(Modifier.width(Spacing.xs))
                    Text(
                        text = onlineCount.toString() + " 个群在线",
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.online,
                    )
                }
            }
        }
        Spacer(Modifier.height(Spacing.sm))
        NearbyFilterChips(
            options = GROUP_FILTERS,
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
                label = GROUP_SORTS[sortIndex].first,
                options = GROUP_SORTS.map { it.first },
                selectedIndex = sortIndex,
                onSelected = onSortSelected,
            )
        }
        Spacer(Modifier.height(Spacing.xs))
        when {
            state.loading && state.items.isEmpty() -> LoadingState(message = "正在寻找附近群组…")
            state.error != null && state.items.isEmpty() -> ErrorState(
                message = state.error ?: "加载失败",
                onRetry = { scope.launch { state.initialLoad(loader) } },
            )
            state.items.isEmpty() -> EmptyState(title = "附近暂时没有群组", description = "换个分类或稍后再来看看")
            else -> LazyVerticalGrid(
                columns = GridCells.Fixed(2),
                state = gridState,
                modifier = Modifier.fillMaxSize(),
                contentPadding = PaddingValues(start = Spacing.lg, end = Spacing.lg, bottom = Spacing.xxl),
                horizontalArrangement = Arrangement.spacedBy(Spacing.md),
                verticalArrangement = Arrangement.spacedBy(Spacing.md),
            ) {
                itemsIndexed(state.items, key = { _, group -> group.group.groupId }) { _, group ->
                    NearbyGroupCard(
                        group = group,
                        joined = joinedStates[group.group.groupId] ?: (group.viewerState != null),
                        pending = pendingStates[group.group.groupId] == true,
                        busy = joinBusy[group.group.groupId] == true,
                        onClick = { onOpenGroup(group.group.groupId) },
                        onJoin = { join(group) },
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
