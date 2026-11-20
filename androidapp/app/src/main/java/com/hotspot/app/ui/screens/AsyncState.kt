package com.hotspot.app.ui.screens

import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.lazy.grid.LazyGridState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.Page

/** 把分页结果压成单页列表（附近枢纽 / 搜索首屏等只取第一页的场景）。 */
internal suspend fun <T> loadFirstPage(
    loader: suspend (String?) -> ApiResult<Page<T>>,
): ApiResult<List<T>> = when (val result = loader(null)) {
    is ApiResult.Ok -> ApiResult.Ok(result.data.items)
    is ApiResult.Err -> result
}

/**
 * 分页列表状态机（loading / refreshing / loadingMore / error / endReached），
 * 与前端 InfiniteSentinel + ListEnd 的语义对齐：
 * - 首屏加载失败 → ErrorState + 重试
 * - 加载更多失败 → 展示行内错误，不吞掉已有内容
 * - next_cursor=null → endReached，渲染「没有更多了」
 */
class PagedListState<T> {
    var items by mutableStateOf<List<T>>(emptyList())
        private set
    var cursor by mutableStateOf<String?>(null)
        private set
    var loading by mutableStateOf(false)
        private set
    var refreshing by mutableStateOf(false)
        private set
    var loadingMore by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set
    var endReached by mutableStateOf(false)
        private set

    suspend fun initialLoad(loader: suspend (String?) -> ApiResult<Page<T>>) {
        if (loading) return
        loading = true
        error = null
        when (val result = loader(null)) {
            is ApiResult.Ok -> applyFirstPage(result.data)
            is ApiResult.Err -> error = result.error.userMessage
        }
        loading = false
    }

    suspend fun refresh(loader: suspend (String?) -> ApiResult<Page<T>>) {
        refreshing = true
        error = null
        when (val result = loader(null)) {
            is ApiResult.Ok -> applyFirstPage(result.data)
            is ApiResult.Err -> error = result.error.userMessage
        }
        refreshing = false
    }

    suspend fun loadMore(loader: suspend (String?) -> ApiResult<Page<T>>) {
        val next = cursor
        if (loadingMore || refreshing || endReached || next == null) return
        loadingMore = true
        when (val result = loader(next)) {
            is ApiResult.Ok -> {
                items = items + result.data.items
                cursor = result.data.nextCursor
                endReached = result.data.nextCursor == null
            }
            is ApiResult.Err -> error = result.error.userMessage
        }
        loadingMore = false
    }

    private fun applyFirstPage(page: Page<T>) {
        items = page.items
        cursor = page.nextCursor
        endReached = page.nextCursor == null
        error = null
    }
}

/** 单页加载状态（附近枢纽 / 搜索首屏等不需要无限滚动的列表）。 */
class SingleLoadState<T> {
    var items by mutableStateOf<List<T>>(emptyList())
        private set
    var loading by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set
    var loaded by mutableStateOf(false)
        private set

    suspend fun load(loader: suspend () -> ApiResult<List<T>>) {
        if (loading) return
        loading = true
        error = null
        when (val result = loader()) {
            is ApiResult.Ok -> {
                items = result.data
                loaded = true
            }
            is ApiResult.Err -> error = result.error.userMessage
        }
        loading = false
    }
}

@Composable
fun <T> rememberSingleLoadState(): SingleLoadState<T> = remember { SingleLoadState<T>() }

@Composable
fun <T> rememberPagedListState(): PagedListState<T> = remember { PagedListState<T>() }

/** 无限滚动：列表滚动到距底部 3 项内时自动续载（不足一屏时会连续补页直至到底或填满）。 */
@Composable
fun <T> InfiniteScrollHandler(
    state: PagedListState<T>,
    listState: LazyListState,
    loader: suspend (String?) -> ApiResult<Page<T>>,
) {
    val shouldLoadMore by remember {
        derivedStateOf {
            val total = listState.layoutInfo.totalItemsCount
            val lastVisible = listState.layoutInfo.visibleItemsInfo.lastOrNull()?.index ?: -1
            total > 0 && lastVisible >= total - 3
        }
    }
    LaunchedEffect(shouldLoadMore, state.items.size, state.endReached) {
        if (shouldLoadMore) state.loadMore(loader)
    }
}

/** 无限滚动（网格版）：2 列网格同样在距底 3 项内续载（我的瞬间页复用同一分页语义）。 */
@Composable
fun <T> InfiniteScrollHandler(
    state: PagedListState<T>,
    gridState: LazyGridState,
    loader: suspend (String?) -> ApiResult<Page<T>>,
) {
    val shouldLoadMore by remember {
        derivedStateOf {
            val total = gridState.layoutInfo.totalItemsCount
            val lastVisible = gridState.layoutInfo.visibleItemsInfo.lastOrNull()?.index ?: -1
            total > 0 && lastVisible >= total - 3
        }
    }
    LaunchedEffect(shouldLoadMore, state.items.size, state.endReached) {
        if (shouldLoadMore) state.loadMore(loader)
    }
}
