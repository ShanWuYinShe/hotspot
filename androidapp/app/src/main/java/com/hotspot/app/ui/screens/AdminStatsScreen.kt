package com.hotspot.app.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.AdminGeoStatsResponse
import com.hotspot.app.data.model.AdminPlaceRankItem
import com.hotspot.app.data.model.AdminRegionRankItem
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.ListGroup
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.TimeFormat

/**
 * 管理后台 · 统计分析（仅平台管理员）：全站存量、近 7 日新增趋势、热门地点与区域 Top。
 * 数据来自 admin/stats/geo（弱一致快照；totals 不受时间窗影响）。
 */
@Composable
fun AdminStatsScreen(onBack: () -> Unit) {
    val colors = HotSpotTheme.colors
    var stats by remember { mutableStateOf<AdminGeoStatsResponse?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var reloadKey by remember { mutableIntStateOf(0) }

    LaunchedEffect(reloadKey) {
        loading = true
        error = null
        when (val result = AppContainer.adminRepository.fetchGeoStats()) {
            is ApiResult.Ok -> stats = result.data
            is ApiResult.Err -> error = result.error.userMessage
        }
        loading = false
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
                        item { SectionTitle("区域 Top " + data.regions.size + "（S2 网格）") }
                        item {
                            ListGroup {
                                data.regions.forEachIndexed { index, region ->
                                    RegionRow(index + 1, region)
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
private fun RegionRow(rank: Int, region: AdminRegionRankItem) {
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
