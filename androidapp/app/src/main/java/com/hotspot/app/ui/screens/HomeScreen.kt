package com.hotspot.app.ui.screens

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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
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
import com.hotspot.app.ui.components.MomentCard
import com.hotspot.app.ui.components.SegmentedTabs
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import kotlinx.coroutines.launch

private val FEED_TABS = listOf("为你推荐", "正在关注")

/**
 * 首页信息流：品牌头 + 分段（为你推荐 = moments/search 热门模式；正在关注 = fetch_following）
 * + 整卡图片卡列表；下拉刷新 + 无限滚动；loading / empty / error 三态齐全。
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HomeScreen(
    onOpenSearch: () -> Unit,
    onOpenNotify: () -> Unit,
    onOpenMoment: (momentId: String) -> Unit,
) {
    val colors = HotSpotTheme.colors
    val feedRepository = AppContainer.feedRepository
    val scope = rememberCoroutineScope()
    var tabIndex by remember { mutableIntStateOf(0) }
    val state = remember(tabIndex) { PagedListState<MomentItem>() }
    val listState = rememberLazyListState()

    val loader: suspend (String?) -> ApiResult<Page<MomentItem>> = { cursor ->
        if (tabIndex == 0) feedRepository.recommended(cursor) else feedRepository.following(cursor)
    }

    LaunchedEffect(tabIndex) { state.initialLoad(loader) }
    InfiniteScrollHandler(state, listState, loader)

    Column(Modifier.fillMaxSize()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = Spacing.lg, vertical = Spacing.sm2),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = "HotSpot",
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.SemiBold,
                color = colors.textPrimary,
            )
            Spacer(Modifier.weight(1f))
            HeaderIconButton(
                icon = Icons.Filled.Search,
                contentDescription = "搜索",
                onClick = onOpenSearch,
            )
            HeaderIconButton(
                icon = Icons.Filled.Notifications,
                contentDescription = "通知",
                onClick = onOpenNotify,
            )
        }
        SegmentedTabs(
            options = FEED_TABS,
            selectedIndex = tabIndex,
            onSelected = { tabIndex = it },
            modifier = Modifier.padding(horizontal = Spacing.lg),
        )
        Spacer(Modifier.height(Spacing.sm))

        when {
            state.loading && state.items.isEmpty() -> LoadingState(message = "正在加载动态…")
            state.error != null && state.items.isEmpty() -> ErrorState(
                message = state.error ?: "加载失败",
                onRetry = { scope.launch { state.initialLoad(loader) } },
            )
            state.items.isEmpty() -> EmptyState(
                title = if (tabIndex == 0) "暂时没有推荐内容" else "关注的人还没有发布动态",
                description = if (tabIndex == 0) "稍后再来看看，或者先关注一些感兴趣的人" else "去附近看看，发现新的朋友",
                actionLabel = "刷新",
                onAction = { scope.launch { state.initialLoad(loader) } },
            )
            else -> PullToRefreshBox(
                isRefreshing = state.refreshing,
                onRefresh = { scope.launch { state.refresh(loader) } },
                modifier = Modifier.fillMaxSize(),
            ) {
                LazyColumn(
                    state = listState,
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(
                        start = Spacing.lg,
                        end = Spacing.lg,
                        top = Spacing.sm,
                        bottom = Spacing.xxl,
                    ),
                    verticalArrangement = Arrangement.spacedBy(Spacing.md),
                ) {
                    items(state.items, key = { it.momentId }) { moment ->
                        MomentCard(moment = moment, onClick = { onOpenMoment(moment.momentId) })
                    }
                    item(key = "footer") {
                        when {
                            state.loadingMore -> InlineLoading()
                            state.error != null -> ErrorState(
                                message = state.error ?: "加载失败",
                                onRetry = { scope.launch { state.loadMore(loader) } },
                            )
                            state.endReached -> ListEnd()
                        }
                    }
                }
            }
        }
    }
}
