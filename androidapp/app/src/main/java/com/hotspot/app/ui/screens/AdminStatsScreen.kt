package com.hotspot.app.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
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
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.AdminGeoDrilldownItem
import com.hotspot.app.data.model.AdminGeoStatsResponse
import com.hotspot.app.data.model.AdminPlaceRankItem
import com.hotspot.app.data.model.AdminRegionRankItem
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.InlineLoading
import com.hotspot.app.ui.components.ListEnd
import com.hotspot.app.ui.components.ListGroup
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SearchInput
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.TabPills
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.TimeFormat
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * 管理后台 · 统计分析（仅平台管理员）：全站存量、近 7 日新增趋势、热门地点与区域 Top。
 * 数据来自 admin/stats/geo（弱一致快照；totals 不受时间窗影响）。
 */
@Composable
fun AdminStatsScreen(onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    val colors = HotSpotTheme.colors
    var stats by remember { mutableStateOf<AdminGeoStatsResponse?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var reloadKey by remember { mutableIntStateOf(0) }
    var drilldownToken by remember { mutableStateOf<String?>(null) }
    var drilldownLevel by remember { mutableIntStateOf(8) }
    var drilldownRegionLabel by remember { mutableStateOf("") }
    var drilldownTypeIndex by remember { mutableIntStateOf(0) }
    var drilldownItems by remember { mutableStateOf<List<AdminGeoDrilldownItem>>(emptyList()) }
    var drilldownCursor by remember { mutableStateOf<String?>(null) }
    var drilldownLoading by remember { mutableStateOf(false) }
    var drilldownMoreLoading by remember { mutableStateOf(false) }
    var drilldownError by remember { mutableStateOf<String?>(null) }
    var drilldownReloadKey by remember { mutableIntStateOf(0) }
    var drilldownUserId by remember { mutableStateOf<String?>(null) }
    var drilldownGroupId by remember { mutableStateOf<String?>(null) }
    var drilldownQuery by remember { mutableStateOf("") }
    var drilldownSearchLoading by remember { mutableStateOf(false) }
    var drilldownSearchReloadKey by remember { mutableIntStateOf(0) }

    LaunchedEffect(reloadKey) {
        loading = true
        error = null
        when (val result = AppContainer.adminRepository.fetchGeoStats()) {
            is ApiResult.Ok -> stats = result.data
            is ApiResult.Err -> error = result.error.userMessage
        }
        loading = false
    }

    LaunchedEffect(drilldownToken, drilldownTypeIndex, drilldownReloadKey) {
        val token = drilldownToken ?: return@LaunchedEffect
        drilldownLoading = true
        drilldownError = null
        drilldownItems = emptyList()
        drilldownCursor = null
        drilldownQuery = ""
        val types = listOf("user", "group", "moment")
        when (val result = AppContainer.adminRepository.fetchGeoDrilldown(token, drilldownLevel, types[drilldownTypeIndex], null)) {
            is ApiResult.Ok -> {
                drilldownItems = result.data.items
                drilldownCursor = result.data.nextCursor
            }
            is ApiResult.Err -> drilldownError = result.error.userMessage
        }
        drilldownLoading = false
    }

    // 下钻契约没有关键词参数：搜索时先加载完全部分页（最多 10 页），再本地筛选。
    LaunchedEffect(drilldownQuery, drilldownToken, drilldownTypeIndex, drilldownSearchReloadKey) {
        if (drilldownQuery.isBlank() || drilldownLoading) return@LaunchedEffect
        val token = drilldownToken ?: return@LaunchedEffect
        delay(300)
        var next: String? = drilldownCursor ?: return@LaunchedEffect
        drilldownSearchLoading = true
        val types = listOf("user", "group", "moment")
        var pages = 0
        while (next != null && pages < 10) {
            when (val result = AppContainer.adminRepository.fetchGeoDrilldown(token, drilldownLevel, types[drilldownTypeIndex], next)) {
                is ApiResult.Ok -> {
                    drilldownItems = (drilldownItems + result.data.items).distinctBy { it.objectId }
                    drilldownCursor = result.data.nextCursor
                    next = result.data.nextCursor
                }
                is ApiResult.Err -> break
            }
            pages += 1
        }
        drilldownSearchLoading = false
    }

    val drilldownFiltered = remember(drilldownItems, drilldownQuery) {
        val keyword = drilldownQuery.trim()
        if (keyword.isBlank()) drilldownItems else drilldownItems.filter { item ->
            item.title.contains(keyword, ignoreCase = true) ||
                item.objectId.contains(keyword, ignoreCase = true) ||
                item.locationName.orEmpty().contains(keyword, ignoreCase = true)
        }
    }

    fun loadMoreDrilldown() {
        val token = drilldownToken ?: return
        val next = drilldownCursor ?: return
        if (drilldownMoreLoading) return
        drilldownMoreLoading = true
        scope.launch {
            val types = listOf("user", "group", "moment")
            when (val result = AppContainer.adminRepository.fetchGeoDrilldown(token, drilldownLevel, types[drilldownTypeIndex], next)) {
                is ApiResult.Ok -> {
                    drilldownItems = (drilldownItems + result.data.items).distinctBy { it.objectId }
                    drilldownCursor = result.data.nextCursor
                }
                is ApiResult.Err -> Unit
            }
            drilldownMoreLoading = false
        }
    }

    Column(Modifier.fillMaxSize()) {
        PageHeader(
            title = "统计分析",
            subtitle = "平台概览",
            onBack = onBack,
            actions = {
                SystemButton(
                    text = "刷新",
                    onClick = { reloadKey += 1 },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !loading,
                )
            },
        )
        when {
            loading && stats == null -> LoadingState(message = "正在生成统计…")
            error != null && stats == null -> ErrorState(message = error ?: "加载失败", onRetry = { reloadKey += 1 })
            stats == null -> LoadingState()
            else -> {
                val data = stats ?: return@Column
                LazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = androidx.compose.foundation.layout.PaddingValues(
                        horizontal = Spacing.lg,
                        vertical = Spacing.sm,
                    ),
                    verticalArrangement = Arrangement.spacedBy(Spacing.md),
                ) {
                    item {
                        ListGroup {
                            Column(Modifier.fillMaxWidth().padding(Spacing.md)) {
                                Text(
                                    text = "全站存量",
                                    style = MaterialTheme.typography.titleSmall,
                                    color = colors.textPrimary,
                                )
                                Spacer(Modifier.height(Spacing.sm))
                                Row(Modifier.fillMaxWidth()) {
                                    TotalCell("用户", data.totals.user, Modifier.weight(1f))
                                    TotalCell("群组", data.totals.group, Modifier.weight(1f))
                                    TotalCell("瞬间", data.totals.moment, Modifier.weight(1f))
                                }
                                Spacer(Modifier.height(Spacing.sm))
                                Text(
                                    text = "合计 " + data.totals.total + " · 统计于 " +
                                        TimeFormat.relative(data.generatedAt),
                                    style = MaterialTheme.typography.labelSmall,
                                    color = colors.textSecondary,
                                )
                            }
                        }
                    }
                    if (data.trends.isNotEmpty()) {
                        item { SectionTitle("近 " + data.trends.size + " 日新增") }
                        item {
                            ListGroup {
                                TrendChart(
                                    points = data.trends.map { it.user + it.group + it.moment },
                                    labels = data.trends.map { it.date.takeLast(5) },
                                    modifier = Modifier.padding(Spacing.md),
                                )
                            }
                        }
                    }
                    if (data.places.isNotEmpty()) {
                        item { SectionTitle("热门地点 Top " + data.places.size) }
                        item {
                            ListGroup {
                                data.places.forEachIndexed { index, place ->
                                    PlaceRow(index + 1, place)
                                    if (index != data.places.lastIndex) {
                                        HorizontalDivider(color = colors.borderSubtle)
                                    }
                                }
                            }
                        }
                    }
                    if (data.regions.isNotEmpty()) {
                        item { SectionTitle("区域 Top " + data.regions.size + "（S2 网格，点击查看明细）") }
                        item {
                            ListGroup {
                                data.regions.forEachIndexed { index, region ->
                                    RegionRow(
                                        rank = index + 1,
                                        region = region,
                                        onClick = {
                                            drilldownToken = region.cellToken
                                            drilldownLevel = region.level
                                            drilldownRegionLabel = "网格 " + region.cellToken.take(12)
                                            drilldownTypeIndex = 0
                                        },
                                    )
                                    if (index != data.regions.lastIndex) {
                                        HorizontalDivider(color = colors.borderSubtle)
                                    }
                                }
                            }
                        }
                    }
                    item { Spacer(Modifier.height(Spacing.lg)) }
                }
            }
        }
    }

    if (drilldownToken != null) {
        Dialog(
            onDismissRequest = { drilldownToken = null },
            properties = DialogProperties(usePlatformDefaultWidth = false, decorFitsSystemWindows = false),
        ) {
            Surface(Modifier.fillMaxSize(), color = colors.page) {
                Column(Modifier.fillMaxSize().statusBarsPadding().navigationBarsPadding()) {
                    PageHeader(
                        title = "区域明细",
                        subtitle = drilldownRegionLabel,
                        onBack = { drilldownToken = null },
                    )
                    Column(Modifier.padding(horizontal = Spacing.lg)) {
                        TabPills(
                            options = listOf("用户", "群组", "瞬间"),
                            selectedIndex = drilldownTypeIndex,
                            onSelected = { drilldownTypeIndex = it },
                        )
                        Spacer(Modifier.height(Spacing.sm))
                        SearchInput(
                            value = drilldownQuery,
                            onValueChange = { drilldownQuery = it },
                            placeholder = "搜索标题 / 对象 ID / 位置…",
                        )
                        if (drilldownTypeIndex != 2) {
                            Text(
                                text = "点击行查看" + if (drilldownTypeIndex == 0) "用户详情" else "群组详情",
                                style = MaterialTheme.typography.labelSmall,
                                color = colors.textSecondary,
                                modifier = Modifier.padding(top = Spacing.xs, start = Spacing.xs),
                            )
                        }
                        Spacer(Modifier.height(Spacing.sm2))
                    }
                    when {
                        drilldownLoading -> LoadingState(message = "正在加载区域明细…")
                        drilldownError != null -> ErrorState(
                            message = drilldownError ?: "加载失败",
                            onRetry = { drilldownReloadKey += 1 },
                        )
                        drilldownItems.isEmpty() -> EmptyState(
                            title = "该区域暂无明细",
                            description = "切换用户/群组/瞬间，或换个网格看看。",
                        )
                        drilldownFiltered.isEmpty() -> EmptyState(
                            title = "没有匹配的明细",
                            description = "换个标题、对象 ID 或位置关键词试试。",
                        )
                        else -> LazyColumn(
                            modifier = Modifier.fillMaxSize(),
                            contentPadding = androidx.compose.foundation.layout.PaddingValues(
                                horizontal = Spacing.lg,
                                vertical = Spacing.sm,
                            ),
                        ) {
                            item {
                                Text(
                                    text = "共 " + drilldownItems.size + " 条" +
                                        if (drilldownQuery.isNotBlank()) " · 匹配 " + drilldownFiltered.size + " 条" else "",
                                    style = MaterialTheme.typography.labelSmall,
                                    color = colors.textSecondary,
                                    modifier = Modifier.padding(vertical = Spacing.sm, horizontal = Spacing.xs),
                                )
                            }
                            item {
                                ListGroup {
                                    drilldownFiltered.forEachIndexed { index, item ->
                                        Column(
                                            Modifier
                                                .fillMaxWidth()
                                                .clickable(enabled = drilldownTypeIndex != 2) {
                                                    if (drilldownTypeIndex == 0) {
                                                        drilldownUserId = item.objectId
                                                    } else if (drilldownTypeIndex == 1) {
                                                        drilldownGroupId = item.objectId
                                                    }
                                                }
                                                .padding(horizontal = Spacing.md, vertical = Spacing.sm2),
                                        ) {
                                            Text(
                                                text = item.title.ifBlank { item.objectId },
                                                style = MaterialTheme.typography.bodyLarge,
                                                color = colors.textPrimary,
                                                maxLines = 2,
                                                overflow = TextOverflow.Ellipsis,
                                            )
                                            Text(
                                                text = (item.locationName?.takeIf { it.isNotBlank() } ?: "未知位置") +
                                                    " · " + String.format("%.4f", item.latitude) + ", " +
                                                    String.format("%.4f", item.longitude) +
                                                    " · " + TimeFormat.relative(item.createdAt),
                                                style = MaterialTheme.typography.labelSmall,
                                                color = colors.textSecondary,
                                                maxLines = 2,
                                                overflow = TextOverflow.Ellipsis,
                                            )
                                        }
                                        if (index != drilldownFiltered.lastIndex) {
                                            HorizontalDivider(color = colors.borderSubtle)
                                        }
                                    }
                                }
                            }
                            if (drilldownSearchLoading) {
                                item { InlineLoading(text = "正在加载全部明细用于搜索…") }
                            } else if (drilldownQuery.isBlank() && drilldownCursor != null) {
                                item {
                                    Box(Modifier.fillMaxWidth().padding(vertical = Spacing.lg)) {
                                        SystemButton(
                                            text = if (drilldownMoreLoading) "加载中…" else "加载更多",
                                            onClick = { loadMoreDrilldown() },
                                            variant = ButtonVariant.SECONDARY,
                                            size = ButtonSize.SM,
                                            enabled = !drilldownMoreLoading,
                                            loading = drilldownMoreLoading,
                                        )
                                    }
                                }
                            } else {
                                item { Box(Modifier.fillMaxWidth().padding(vertical = Spacing.lg)) { ListEnd() } }
                            }
                        }
                    }
                }
            }
        }
    }

    drilldownUserId?.let { userId ->
        AdminUserDetailDialog(userId = userId, onDismiss = { drilldownUserId = null })
    }
    drilldownGroupId?.let { groupId ->
        AdminGroupDetailDialog(groupId = groupId, onDismiss = { drilldownGroupId = null })
    }
}

@Composable
private fun SectionTitle(text: String) {
    Text(
        text = text,
        style = MaterialTheme.typography.titleSmall,
        color = HotSpotTheme.colors.textPrimary,
        modifier = Modifier.padding(start = Spacing.xs, end = Spacing.xs, top = Spacing.sm),
    )
}

@Composable
private fun TotalCell(label: String, value: Int, modifier: Modifier = Modifier) {
    val colors = HotSpotTheme.colors
    Column(modifier, horizontalAlignment = Alignment.CenterHorizontally) {
        Text(
            text = value.toString(),
            style = MaterialTheme.typography.headlineSmall,
            color = colors.textPrimary,
        )
        Text(
            text = label,
            style = MaterialTheme.typography.labelSmall,
            color = colors.textSecondary,
        )
    }
}

@Composable
private fun TrendChart(
    points: List<Int>,
    labels: List<String>,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    val max = points.maxOrNull()?.coerceAtLeast(1) ?: 1
    Column(modifier.fillMaxWidth()) {
        Row(
            modifier = Modifier.fillMaxWidth().height(96.dp),
            horizontalArrangement = Arrangement.spacedBy(Spacing.xs),
            verticalAlignment = Alignment.Bottom,
        ) {
            points.forEachIndexed { index, value ->
                val ratio = value.toFloat() / max.toFloat()
                val barHeight = (64 * ratio).dp.coerceAtLeast(2.dp)
                Column(
                    modifier = Modifier.weight(1f),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.Bottom,
                ) {
                    Text(
                        text = value.toString(),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textSecondary,
                    )
                    Spacer(Modifier.height(2.dp))
                    Box(
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(barHeight)
                            .background(colors.accent, RoundedCornerShape(4.dp)),
                    )
                    Spacer(Modifier.height(Spacing.xs))
                    Text(
                        text = labels.getOrNull(index) ?: "",
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textSecondary,
                        maxLines = 1,
                        overflow = TextOverflow.Visible,
                    )
                }
            }
        }
    }
}


@Composable
private fun PlaceRow(rank: Int, place: AdminPlaceRankItem) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier.fillMaxWidth().padding(horizontal = Spacing.md, vertical = Spacing.sm2),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = rank.toString(),
            style = MaterialTheme.typography.labelSmall,
            color = colors.textSecondary,
            modifier = Modifier.width(Spacing.lg),
        )
        Column(Modifier.weight(1f)) {
            Text(
                text = place.locationName,
                style = MaterialTheme.typography.bodyLarge,
                color = colors.textPrimary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                text = "用户 " + place.breakdown.user + " · 群组 " + place.breakdown.group +
                    " · 瞬间 " + place.breakdown.moment,
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
            )
        }
        Text(
            text = place.totalCount.toString(),
            style = MaterialTheme.typography.bodyLarge,
            color = colors.textPrimary,
        )
    }
}

@Composable
private fun RegionRow(
    rank: Int,
    region: AdminRegionRankItem,
    onClick: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(horizontal = Spacing.md, vertical = Spacing.sm2),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = rank.toString(),
            style = MaterialTheme.typography.labelSmall,
            color = colors.textSecondary,
            modifier = Modifier.width(Spacing.lg),
        )
        Column(Modifier.weight(1f)) {
            Text(
                text = "网格 " + region.cellToken.take(12) + "…（L" + region.level + "）",
                style = MaterialTheme.typography.bodyLarge,
                color = colors.textPrimary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                text = "中心 " + String.format("%.4f", region.center.latitude) + ", " +
                    String.format("%.4f", region.center.longitude) +
                    " · 用户 " + region.breakdown.user + " · 群组 " + region.breakdown.group +
                    " · 瞬间 " + region.breakdown.moment,
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
        Text(
            text = region.totalCount.toString(),
            style = MaterialTheme.typography.bodyLarge,
            color = colors.textPrimary,
        )
    }
}