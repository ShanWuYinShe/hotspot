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
import androidx.compose.runtime.mutableIntStateOf
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
import com.hotspot.app.data.model.MomentItem
import com.hotspot.app.data.model.Page
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.HeaderIconButton
import com.hotspot.app.ui.components.InlineLoading
import com.hotspot.app.ui.components.ListEnd
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.NearbyFilterChips
import com.hotspot.app.ui.components.NearbyLocationChip
import com.hotspot.app.ui.components.NearbyMomentCard
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import kotlinx.coroutines.launch

/** 附近瞬间筛选（设计稿 87:3995 顶部胶囊：全部 / 夜色）。 */
private val MOMENT_FILTERS = listOf("全部", "夜色")

/**
 * 附近瞬间（设计稿 87:3995）：标题/定位 → 筛选胶囊 → 两列瞬间卡（日期胶囊 + 图/文 + 作者 + 互动数）。
 * 「夜色」为端内标签筛选（moments/fetch_nearby 无 tag 入参，客户端按 moments.tags 过滤）。
 */
@Composable
fun NearbyMomentsScreen(onBack: () -> Unit, onOpenSearch: () -> Unit, onOpenMoment: (String) -> Unit) {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    var filterIndex by remember { mutableIntStateOf(0) }

    Column(Modifier.fillMaxSize()) {
        PageHeader(
            title = "附近瞬间",
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
        NearbyMomentsContent(
            filterIndex = filterIndex,
            onFilterSelected = { filterIndex = it },
            onOpenMoment = onOpenMoment,
        )
    }
}

@Composable
private fun NearbyMomentsContent(
    filterIndex: Int,
    onFilterSelected: (Int) -> Unit,
    onOpenMoment: (String) -> Unit,
) {
    val colors = HotSpotTheme.colors
    val scope = rememberCoroutineScope()
    val state = rememberPagedListState<MomentItem>()
    val gridState = rememberLazyGridState()
    var locationLabel by remember { mutableStateOf("附近 · 5 km 内") }
    val loader: suspend (String?) -> ApiResult<Page<MomentItem>> = { cursor ->
        AppContainer.nearbyRepository.nearbyMoments(cursor, AppContainer.defaultLocation()?.point)
    }

    LaunchedEffect(Unit) {
        AppContainer.defaultLocation()?.let { location ->
            if (location.locationName.isNotBlank()) locationLabel = location.locationName + " · 5 km 内"
        }
        state.initialLoad(loader)
    }
    InfiniteScrollHandler(state, gridState, loader)

    val visible = if (filterIndex == 1) {
        state.items.filter { it.tags.contains("夜色") }
    } else {
        state.items
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(top = Spacing.xs),
    ) {
        Row(Modifier.padding(horizontal = Spacing.lg), verticalAlignment = Alignment.CenterVertically) {
            val countText = if (state.items.isNotEmpty()) " · " + state.items.size + " 则" else ""
            NearbyLocationChip(locationLabel + countText)
        }
        Spacer(Modifier.height(Spacing.sm))
        NearbyFilterChips(
            options = MOMENT_FILTERS,
            selectedIndex = filterIndex,
            onSelected = onFilterSelected,
            modifier = Modifier.padding(horizontal = Spacing.lg),
        )
        Spacer(Modifier.height(Spacing.sm))
        when {
            state.loading && state.items.isEmpty() -> LoadingState(message = "正在寻找附近瞬间…")
            state.error != null && state.items.isEmpty() -> ErrorState(
                message = state.error ?: "加载失败",
                onRetry = { scope.launch { state.initialLoad(loader) } },
            )
            visible.isEmpty() -> EmptyState(title = "附近暂时没有瞬间", description = "换个筛选或稍后再来看看")
            else -> LazyVerticalGrid(
                columns = GridCells.Fixed(2),
                state = gridState,
                modifier = Modifier.fillMaxSize(),
                contentPadding = PaddingValues(start = Spacing.lg, end = Spacing.lg, bottom = Spacing.xxl),
                horizontalArrangement = Arrangement.spacedBy(Spacing.md),
                verticalArrangement = Arrangement.spacedBy(Spacing.md),
            ) {
                itemsIndexed(visible, key = { _, moment -> moment.momentId }) { _, moment ->
                    NearbyMomentCard(moment = moment, onClick = { onOpenMoment(moment.momentId) })
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
