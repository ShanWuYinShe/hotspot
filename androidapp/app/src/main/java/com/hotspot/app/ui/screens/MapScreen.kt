package com.hotspot.app.ui.screens

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.R
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.location.BrowseAnchor
import com.hotspot.app.data.model.GeoPoint
import com.hotspot.app.data.model.MapHotspotItem
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.HotSpotTextStyles
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.ui.theme.Tokens
import com.hotspot.app.util.MapPlaceSearch
import com.hotspot.app.util.MapProjection
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * 地图选点（Figma 87:1866「地图选点 · /map」，390×844）：
 * 全屏底图 + 顶部返回/筛选浮动条 + 右侧缩放/回中/图层控件 + 中心定位针 +
 * 底部选点面板（搜索地点/最近使用/设为浏览锚点）。
 *
 * 交互：搜索当前视口附近的地点候选（map/canvas hotspots，支持「纬度,经度」坐标），
 * 选择后中心点/名称更新并写入最近使用；「将地图中心设为浏览锚点」持久化本地锚点，
 * 附近页按它计算距离与推荐。
 */
@Composable
fun MapScreen(
    onBack: () -> Unit,
    onAnchorSaved: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    val repo = AppContainer.nearbyRepository
    val scope = rememberCoroutineScope()
    val focusManager = LocalFocusManager.current

    var center by remember { mutableStateOf<GeoPoint?>(null) }
    var locationLabel by remember { mutableStateOf("") }
    var initialLoading by remember { mutableStateOf(true) }
    var initialError by remember { mutableStateOf<String?>(null) }

    var query by remember { mutableStateOf("") }
    var results by remember { mutableStateOf<List<PlaceCandidate>>(emptyList()) }
    var searching by remember { mutableStateOf(false) }
    var filterIndex by remember { mutableIntStateOf(0) }
    var filterMenuOpen by remember { mutableStateOf(false) }
    var recent by remember { mutableStateOf(AppContainer.browseAnchorStore.recent()) }
    var zoom by remember { mutableFloatStateOf(1f) }
    var warmLayer by remember { mutableStateOf(false) }
    var layerMenuOpen by remember { mutableStateOf(false) }

    val filters = listOf("全部", "用户", "群组", "瞬间")
    val filterTypes = listOf(null, "user", "group", "moment")

    suspend fun loadInitial() {
        initialError = null
        initialLoading = true
        val location = AppContainer.defaultLocation()
        if (location?.point == null) {
            initialError = "无法获取默认浏览位置，请稍后重试"
        } else {
            center = location.point
            locationLabel = location.locationName.ifBlank { "默认浏览位置" }
        }
        initialLoading = false
    }

    LaunchedEffect(Unit) { loadInitial() }

    // 搜索候选：视口附近 hotspots（map/canvas 仅返回聚合或明细之一；0.3° 视口通常为 hotspots）。
    LaunchedEffect(query, filterIndex, center) {
        val keyword = query.trim()
        val base = center
        if (keyword.isBlank() || base == null) {
            results = emptyList()
            searching = false
            return@LaunchedEffect
        }
        delay(SEARCH_DEBOUNCE_MS)
        searching = true
        val candidates = mutableListOf<PlaceCandidate>()
        MapPlaceSearch.parseCoordinate(keyword)?.let { point ->
            candidates += PlaceCandidate(
                label = "使用坐标 " + MapPlaceSearch.formatCoordinate(point),
                subtitle = "自定义坐标",
                point = point,
                objectType = "coordinate",
            )
        }
        val bounds = MapProjection.boundsOf(
            center = base,
            spanLat = SEARCH_VIEWPORT_SPAN / zoom,
            spanLng = SEARCH_VIEWPORT_SPAN * 1.35 / zoom,
        )
        when (val response = repo.mapCanvas(bounds, filterTypes[filterIndex])) {
            is ApiResult.Ok -> {
                if (response.data.mode == "hotspots") {
                    response.data.hotspots?.items.orEmpty()
                        .filter { item -> item.matches(keyword) }
                        .take(MAX_RESULTS)
                        .forEach { item ->
                            val point = item.location.point
                            if (point != null) {
                                candidates += item.toCandidate(point)
                            }
                        }
                }
            }
            is ApiResult.Err -> Unit
        }
        results = candidates
        searching = false
    }

    fun selectCandidate(candidate: PlaceCandidate) {
        center = candidate.point
        locationLabel = if (candidate.objectType == "coordinate") "自定义位置" else candidate.label
        // 选中后清空搜索词，回到「最近使用」展示刚选中的地点（与 Figma 空态/列表同一区域）。
        query = ""
        focusManager.clearFocus()
        AppContainer.browseAnchorStore.addRecent(
            BrowseAnchor(candidate.point.latitude, candidate.point.longitude, candidate.label),
        )
        recent = AppContainer.browseAnchorStore.recent()
    }

    fun saveAnchor() {
        val point = center ?: return
        val label = locationLabel.ifBlank { "自定义位置" }
        AppContainer.setBrowseAnchor(BrowseAnchor(point.latitude, point.longitude, label))
        onAnchorSaved()
    }

    Box(
        Modifier
            .fillMaxSize()
            .background(colors.page)
            .imePadding(),
    ) {
        // 底图：surface-page + 左侧淡蓝渐变（对齐 Figma MapTilesLayer + gradient overlay）。
        Canvas(Modifier.fillMaxSize()) {
            drawRect(colors.page)
            if (warmLayer) {
                drawRect(colors.ambient.glow1.copy(alpha = 0.5f))
            }
            drawRect(
                brush = Brush.horizontalGradient(
                    colors = listOf(colors.accent.copy(alpha = 0.03f), Color.Transparent),
                    startX = 0f,
                    endX = size.width * 0.65f,
                ),
            )
        }

        // 顶部浮动操作条：返回（42 圆钮）+ 右侧筛选胶囊（80×36）。
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(start = 14.dp, end = 14.dp, top = 15.dp)
                .height(42.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            MapGlassCircle(size = 42.dp, onClick = onBack) {
                Icon(
                    painter = painterResource(R.drawable.ic_map_back),
                    contentDescription = "返回",
                    modifier = Modifier.size(15.dp),
                    tint = colors.textSecondary,
                )
            }
            Spacer(Modifier.weight(1f))
            Box {
                Row(
                    modifier = Modifier
                        .width(80.dp)
                        .height(36.dp)
                        .shadow(3.dp, AppShapes.pill)
                        .clip(AppShapes.pill)
                        .background(colors.surface)
                        .border(1.dp, colors.borderSubtle, AppShapes.pill)
                        .clickable { filterMenuOpen = true },
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.Center,
                ) {
                    Icon(
                        painter = painterResource(R.drawable.ic_map_filter),
                        contentDescription = null,
                        modifier = Modifier.size(14.dp),
                        tint = colors.textSecondary,
                    )
                    Spacer(Modifier.width(6.dp))
                    Text(
                        text = "筛选",
                        style = MaterialTheme.typography.labelMedium,
                        color = colors.textPrimary,
                    )
                }
                DropdownMenu(expanded = filterMenuOpen, onDismissRequest = { filterMenuOpen = false }) {
                    filters.forEachIndexed { index, label ->
                        DropdownMenuItem(
                            text = { Text(label) },
                            onClick = {
                                filterIndex = index
                                filterMenuOpen = false
                            },
                        )
                    }
                }
            }
        }

        // 右侧浮动控件：缩放（+/-）、回到默认位置、底图切换（图层）。
        Column(
            modifier = Modifier
                .align(Alignment.TopEnd)
                .padding(top = 292.dp, end = 12.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Column(
                modifier = Modifier
                    .width(42.dp)
                    .height(82.dp)
                    .shadow(4.dp, AppShapes.card)
                    .clip(AppShapes.card)
                    .background(colors.surface)
                    .border(1.dp, colors.borderSubtle, AppShapes.card),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Box(
                    modifier = Modifier
                        .weight(1f)
                        .fillMaxWidth()
                        .clickable { zoom = (zoom + ZOOM_STEP).coerceAtMost(MAX_ZOOM) },
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        painter = painterResource(R.drawable.ic_map_zoom_in),
                        contentDescription = "放大地图",
                        modifier = Modifier.size(16.dp),
                        tint = colors.textSecondary,
                    )
                }
                Box(
                    modifier = Modifier
                        .weight(1f)
                        .fillMaxWidth()
                        .clickable { zoom = (zoom - ZOOM_STEP).coerceAtLeast(MIN_ZOOM) },
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        painter = painterResource(R.drawable.ic_map_zoom_out),
                        contentDescription = "缩小地图",
                        modifier = Modifier.size(16.dp),
                        tint = colors.textSecondary,
                    )
                }
            }
            Spacer(Modifier.height(8.dp))
            MapGlassCircle(size = 42.dp, onClick = { scope.launch { loadInitial() } }) {
                Icon(
                    painter = painterResource(R.drawable.ic_map_locate),
                    contentDescription = "回到默认位置",
                    modifier = Modifier.size(18.dp),
                    tint = colors.textSecondary,
                )
            }
            Spacer(Modifier.height(8.dp))
            Box {
                MapGlassCircle(size = 40.dp, onClick = { layerMenuOpen = true }) {
                    Icon(
                        painter = painterResource(R.drawable.ic_map_layers),
                        contentDescription = "切换底图",
                        modifier = Modifier.size(16.dp),
                        tint = colors.textSecondary,
                    )
                }
                DropdownMenu(expanded = layerMenuOpen, onDismissRequest = { layerMenuOpen = false }) {
                    DropdownMenuItem(
                        text = { Text(if (warmLayer) "默认底图" else "暖色底图") },
                        onClick = {
                            warmLayer = !warmLayer
                            layerMenuOpen = false
                        },
                    )
                }
            }
        }

        // 中心定位针（Figma ModernMapPin 36×50，针尖对准画布中心）。
        Box(
            modifier = Modifier
                .align(Alignment.Center)
                .offset(y = (-41).dp)
                .size(width = 36.dp, height = 50.dp),
        ) {
            androidx.compose.foundation.Image(
                painter = painterResource(R.drawable.map_pin),
                contentDescription = "地图中心",
                modifier = Modifier.fillMaxSize(),
            )
        }

        // 底部选点面板（318dp，Figma 87:3061/3062）：拖拽把手 + 搜索 + 最近使用 + CTA + 版权。
        // 各元素按设计稿绝对坐标排布：把手 y8、搜索 y20、最近使用区 y67、CTA y201、说明 y251、版权 y266。
        Box(
            modifier = Modifier
                .align(Alignment.BottomCenter)
                .fillMaxWidth()
                .height(318.dp)
                .shadow(24.dp, RoundedCornerShape(topStart = Tokens.Radius.sheet.dp, topEnd = Tokens.Radius.sheet.dp))
                .clip(RoundedCornerShape(topStart = Tokens.Radius.sheet.dp, topEnd = Tokens.Radius.sheet.dp))
                .background(colors.surface),
        ) {
            Box(
                modifier = Modifier
                    .align(Alignment.TopCenter)
                    .offset(y = 8.dp)
                    .width(44.dp)
                    .height(6.dp)
                    .clip(AppShapes.pill)
                    .background(colors.fieldBorderHover.copy(alpha = 0.7f)),
            )
            Row(
                modifier = Modifier
                    .align(Alignment.TopCenter)
                    .offset(y = 20.dp)
                    .padding(horizontal = 16.dp)
                    .fillMaxWidth()
                    .height(39.dp)
                    .clip(AppShapes.pill)
                    .background(colors.surface)
                    .border(1.dp, colors.borderSubtle, AppShapes.pill)
                    .padding(start = 15.dp, end = 15.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    painter = painterResource(R.drawable.ic_map_search),
                    contentDescription = null,
                    modifier = Modifier.size(16.dp),
                    tint = colors.textSecondary,
                )
                Spacer(Modifier.width(10.dp))
                Box(Modifier.weight(1f)) {
                    if (query.isEmpty()) {
                        Text(
                            text = "搜索地点、地标或坐标",
                            style = MaterialTheme.typography.bodyLarge,
                            color = colors.textPrimary.copy(alpha = 0.55f),
                        )
                    }
                    BasicTextField(
                        value = query,
                        onValueChange = { query = it },
                        singleLine = true,
                        textStyle = MaterialTheme.typography.bodyLarge.copy(color = colors.textPrimary),
                        cursorBrush = SolidColor(colors.accent),
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Text),
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            }

            Box(
                Modifier
                    .align(Alignment.TopCenter)
                    .offset(y = 67.dp)
                    .padding(horizontal = 16.dp)
                    .fillMaxWidth()
                    .height(126.dp),
            ) {
                if (query.isNotBlank()) {
                    SearchResultList(
                        results = results,
                        searching = searching,
                        query = query,
                        onSelect = { selectCandidate(it) },
                    )
                } else {
                    Text(
                        text = "最近使用",
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.textPrimary,
                    )
                    if (recent.isEmpty()) {
                        Column(
                            Modifier
                                .fillMaxSize()
                                .padding(top = 12.dp),
                            horizontalAlignment = Alignment.CenterHorizontally,
                        ) {
                            Box(
                                Modifier
                                    .size(38.dp)
                                    .clip(CircleShape)
                                    .background(colors.accentSoftBg),
                                contentAlignment = Alignment.Center,
                            ) {
                                Icon(
                                    painter = painterResource(R.drawable.ic_map_clock),
                                    contentDescription = null,
                                    modifier = Modifier.size(20.dp),
                                    tint = colors.textSecondary,
                                )
                            }
                            Spacer(Modifier.height(6.dp))
                            Text(
                                "暂无最近记录",
                                style = MaterialTheme.typography.bodyLarge,
                                color = colors.textPrimary,
                            )
                            Spacer(Modifier.height(4.dp))
                            Text(
                                "搜索并选择地点后，会记录在这里",
                                style = MaterialTheme.typography.bodyMedium,
                                color = colors.textSecondary,
                                textAlign = TextAlign.Center,
                            )
                        }
                    } else {
                        LazyColumn(
                            Modifier
                                .fillMaxSize()
                                .padding(top = 8.dp),
                        ) {
                            items(recent) { anchor ->
                                PlaceRow(
                                    title = anchor.label,
                                    subtitle = MapPlaceSearch.formatCoordinate(GeoPoint(anchor.latitude, anchor.longitude))
                                        .takeIf { !anchor.label.contains(it) }
                                        .orEmpty(),
                                    onClick = {
                                        center = GeoPoint(anchor.latitude, anchor.longitude)
                                        locationLabel = anchor.label
                                    },
                                )
                            }
                        }
                    }
                }
            }

            Box(
                modifier = Modifier
                    .align(Alignment.TopCenter)
                    .offset(y = 201.dp)
                    .padding(horizontal = 16.dp)
                    .fillMaxWidth()
                    .height(44.dp)
                    .clip(AppShapes.compact)
                    .background(colors.accent)
                    .clickable(enabled = center != null) { saveAnchor() },
                contentAlignment = Alignment.Center,
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(
                        painter = painterResource(R.drawable.ic_map_cta_pin),
                        contentDescription = null,
                        modifier = Modifier.size(16.dp),
                        tint = colors.textOnAccent,
                    )
                    Spacer(Modifier.width(8.dp))
                    Text(
                        text = "将地图中心设为浏览锚点",
                        style = MaterialTheme.typography.labelMedium,
                        color = colors.textOnAccent,
                    )
                }
            }
            Text(
                text = "设置后，附近页按这里计算距离与推荐",
                style = HotSpotTextStyles.micro,
                color = colors.textSecondary,
                textAlign = TextAlign.Center,
                modifier = Modifier
                    .align(Alignment.TopCenter)
                    .offset(y = 251.dp)
                    .fillMaxWidth(),
            )
            Text(
                text = "示意底图",
                style = HotSpotTextStyles.micro,
                color = colors.textOnStatus,
                modifier = Modifier
                    .align(Alignment.TopEnd)
                    .offset(y = 266.dp)
                    .padding(end = 14.dp)
                    .clip(AppShapes.badge)
                    .background(colors.scrim)
                    .padding(horizontal = 7.dp, vertical = 2.dp),
            )
        }

        if (initialLoading) {
            Text(
                text = "正在获取浏览锚点…",
                style = MaterialTheme.typography.bodyMedium,
                color = colors.textSecondary,
                modifier = Modifier.align(Alignment.Center),
            )
        }
        if (initialError != null) {
            Text(
                text = initialError ?: "",
                style = MaterialTheme.typography.bodyMedium,
                color = colors.danger,
                modifier = Modifier
                    .align(Alignment.Center)
                    .padding(horizontal = 32.dp),
                textAlign = TextAlign.Center,
            )
        }
    }
}

@Composable
private fun SearchResultList(
    results: List<PlaceCandidate>,
    searching: Boolean,
    query: String,
    onSelect: (PlaceCandidate) -> Unit,
) {
    val colors = HotSpotTheme.colors
    when {
        results.isNotEmpty() -> LazyColumn(Modifier.fillMaxSize()) {
            items(results) { candidate ->
                PlaceRow(
                    title = candidate.label,
                    subtitle = candidate.subtitle,
                    onClick = { onSelect(candidate) },
                )
            }
        }
        searching -> Text("正在搜索…", style = MaterialTheme.typography.bodyMedium, color = colors.textSecondary)
        else -> Text(
            "未找到与「" + query.take(12) + "」匹配的地点，可输入「纬度,经度」",
            style = MaterialTheme.typography.bodyMedium,
            color = colors.textSecondary,
            modifier = Modifier.padding(top = 6.dp),
        )
    }
}

@Composable
private fun PlaceRow(title: String, subtitle: String, onClick: () -> Unit) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .heightIn(min = 42.dp)
            .clickable(onClick = onClick)
            .padding(vertical = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(R.drawable.ic_map_cta_pin),
            contentDescription = null,
            modifier = Modifier.size(16.dp),
            tint = colors.accent,
        )
        Spacer(Modifier.width(10.dp))
        Column(Modifier.weight(1f)) {
            Text(
                text = title,
                style = MaterialTheme.typography.bodyLarge,
                color = colors.textPrimary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            if (subtitle.isNotBlank()) {
                Text(
                    text = subtitle,
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.textSecondary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}

@Composable
private fun MapGlassCircle(
    size: androidx.compose.ui.unit.Dp,
    onClick: () -> Unit,
    content: @Composable () -> Unit,
) {
    val colors = HotSpotTheme.colors
    Box(
        modifier = Modifier
            .size(size)
            .shadow(3.dp, CircleShape)
            .clip(CircleShape)
            .background(colors.surface.copy(alpha = 0.86f))
            .border(1.dp, colors.interactionActive, CircleShape)
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        content()
    }
}

private data class PlaceCandidate(
    val label: String,
    val subtitle: String,
    val point: GeoPoint,
    val objectType: String,
)

private fun MapHotspotItem.matches(keyword: String): Boolean {
    val lower = keyword.lowercase()
    return title.lowercase().contains(lower) || subtitle.orEmpty().lowercase().contains(lower)
}

private fun MapHotspotItem.toCandidate(point: GeoPoint): PlaceCandidate = PlaceCandidate(
    label = title.ifBlank { "未命名地点" },
    subtitle = subtitle.orEmpty(),
    point = point,
    objectType = objectType,
)

private const val SEARCH_DEBOUNCE_MS = 250L
private const val SEARCH_VIEWPORT_SPAN = 0.3
private const val MAX_RESULTS = 20
private const val ZOOM_STEP = 0.2f
private const val MIN_ZOOM = 0.6f
private const val MAX_ZOOM = 2.0f
