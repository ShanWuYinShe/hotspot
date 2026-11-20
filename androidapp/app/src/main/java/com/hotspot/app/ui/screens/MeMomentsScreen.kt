package com.hotspot.app.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.grid.rememberLazyGridState
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import coil.request.ImageRequest
import com.hotspot.app.AppContainer
import com.hotspot.app.R
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.MomentItem
import com.hotspot.app.data.model.Page
import com.hotspot.app.ui.components.Badge
import com.hotspot.app.ui.components.BadgeTone
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.InlineLoading
import com.hotspot.app.ui.components.ListEnd
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.TabPills
import com.hotspot.app.ui.components.hotSpotFocusRing
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.HotSpotTextStyles
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.MediaUrls
import kotlinx.coroutines.launch

// 设计登记几何（Figma 我的瞬间 87:3257）：页面 gutter 16、两列网格、卡片封面 4:5、圆角 20。
private val GridGap = Spacing.md
private val GridGutter = Spacing.lg

/**
 * 我的瞬间（推入页 me/moments）：
 * 发布 = moments/fetch_created（含 private 与软删占位）；收藏 = moments/fetch_favorites。
 * 设计稿第三个「喜欢」tab 后端无列表端点，首版不渲染。两个 tab 各自独立 PagedListState，
 * 切换时重置网格滚动位置。
 */
@Composable
fun MeMomentsScreen(
    onBack: () -> Unit,
    onOpenMoment: (String) -> Unit,
) {
    val colors = HotSpotTheme.colors
    val profileRepository = AppContainer.profileRepository
    val scope = rememberCoroutineScope()
    var tabIndex by remember { mutableIntStateOf(0) }
    val states = remember { listOf(PagedListState<MomentItem>(), PagedListState<MomentItem>()) }
    val loadedTabs = remember { mutableStateListOf(false, false) }
    val gridState = rememberLazyGridState()
    val state = states[tabIndex]

    fun loaderFor(index: Int): suspend (String?) -> ApiResult<Page<MomentItem>> = { cursor ->
        if (index == 0) profileRepository.createdMoments(cursor) else profileRepository.favoriteMoments(cursor)
    }

    LaunchedEffect(tabIndex) {
        // 切 tab 重置滚动（两个 tab 各自保留已加载数据）。必须用非挂起的 requestScrollToItem：
        // 首屏网格尚未进入组合（加载态分支），scrollToItem 会挂起等待布局，导致首载永远不发出。
        gridState.requestScrollToItem(0)
        if (!loadedTabs[tabIndex]) {
            states[tabIndex].initialLoad(loaderFor(tabIndex))
            loadedTabs[tabIndex] = true
        }
    }
    InfiniteScrollHandler(state, gridState, loaderFor(tabIndex))

    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "我的瞬间", onBack = onBack)

        // 筛选行：左 pill（发布 N / 收藏 N），右「共 N 则记录」（N = 当前 tab 已加载总数）
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = GridGutter, vertical = Spacing.sm2),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Box(Modifier.weight(1f)) {
                TabPills(
                    options = listOf(
                        "发布 " + states[0].items.size,
                        "收藏 " + states[1].items.size,
                    ),
                    selectedIndex = tabIndex,
                    onSelected = { if (it != tabIndex) tabIndex = it },
                )
            }
            Icon(
                painter = painterResource(R.drawable.ic_calendar),
                contentDescription = null,
                modifier = Modifier.size(14.dp),
                tint = colors.textSecondary,
            )
            Spacer(Modifier.width(Spacing.sm2))
            Text(
                text = "共 " + state.items.size + " 则记录",
                style = HotSpotTextStyles.caption,
                color = colors.textSecondary,
                maxLines = 1,
            )
        }

        when {
            state.loading && state.items.isEmpty() -> LoadingState(message = "正在加载瞬间…")
            state.error != null && state.items.isEmpty() -> ErrorState(
                message = state.error ?: "加载失败",
                onRetry = { scope.launch { state.initialLoad(loaderFor(tabIndex)) } },
            )
            state.items.isEmpty() -> EmptyState(
                title = if (tabIndex == 0) "还没有发布过瞬间" else "还没有收藏过瞬间",
                description = if (tabIndex == 0) {
                    "发布的内容会按时间倒序出现在这里，包括仅自己可见的私密瞬间。"
                } else {
                    "在瞬间详情里收藏喜欢的内容，之后可以在这里快速找到。"
                },
                actionLabel = "刷新",
                onAction = { scope.launch { state.initialLoad(loaderFor(tabIndex)) } },
            )
            else -> LazyVerticalGrid(
                columns = GridCells.Fixed(2),
                state = gridState,
                modifier = Modifier.fillMaxSize(),
                contentPadding = PaddingValues(
                    start = GridGutter,
                    end = GridGutter,
                    top = Spacing.sm,
                    bottom = Spacing.xxl,
                ),
                horizontalArrangement = Arrangement.spacedBy(GridGap),
                verticalArrangement = Arrangement.spacedBy(GridGap),
            ) {
                items(state.items, key = { it.momentId }) { moment ->
                    MomentGridTile(
                        moment = moment,
                        onClick = {
                            if (!moment.isDeleted && moment.momentId.isNotBlank()) {
                                onOpenMoment(moment.momentId)
                            }
                        },
                    )
                }
                item(span = { GridItemSpan(maxLineSpan) }, key = "footer") {
                    when {
                        state.loadingMore -> InlineLoading()
                        state.error != null -> ErrorState(
                            message = state.error ?: "加载失败",
                            onRetry = { scope.launch { state.loadMore(loaderFor(tabIndex)) } },
                        )
                        state.endReached -> ListEnd()
                    }
                }
            }
        }
    }
}

/**
 * 网格卡：封面 4:5 / 圆角 20。软删占位为 surface-subtle 块 + 居中 12 次级说明（不可点）；
 * private 瞬间右上角 Badge sm neutral「私密」。
 */
@Composable
private fun MomentGridTile(
    moment: MomentItem,
    onClick: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    val imageUrl = if (moment.isDeleted) null else MediaUrls.absolute(moment.images.firstOrNull()?.url)
    val interaction = remember { MutableInteractionSource() }
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .aspectRatio(4f / 5f)
            .clip(AppShapes.card)
            .background(colors.subtle)
            .then(
                if (moment.isDeleted) {
                    Modifier
                } else {
                    Modifier
                        .clickable(
                            interactionSource = interaction,
                            indication = null,
                            onClick = onClick,
                        )
                        .hotSpotFocusRing(cornerRadius = 20.dp)
                },
            ),
    ) {
        when {
            moment.isDeleted -> Text(
                text = "该瞬间已删除",
                style = MaterialTheme.typography.bodyMedium,
                color = colors.textSecondary,
                textAlign = TextAlign.Center,
                modifier = Modifier.align(Alignment.Center).padding(Spacing.md),
            )
            imageUrl != null -> AsyncImage(
                model = ImageRequest.Builder(LocalContext.current).data(imageUrl).crossfade(true).build(),
                contentDescription = moment.description.ifBlank { "瞬间封面" },
                modifier = Modifier.fillMaxSize(),
                contentScale = ContentScale.Crop,
            )
            else -> Text(
                text = moment.description.ifBlank { "无配图瞬间" },
                style = MaterialTheme.typography.bodyMedium,
                color = colors.textSecondary,
                textAlign = TextAlign.Start,
                maxLines = 4,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier
                    .align(Alignment.Center)
                    .padding(Spacing.md),
            )
        }
        if (!moment.isDeleted && moment.privacyLevel == "private") {
            Badge(
                text = "私密",
                tone = BadgeTone.NEUTRAL,
                modifier = Modifier
                    .align(Alignment.TopEnd)
                    .padding(Spacing.sm),
            )
        }
    }
}
