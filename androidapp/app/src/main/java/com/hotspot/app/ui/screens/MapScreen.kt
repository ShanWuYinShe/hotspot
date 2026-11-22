package com.hotspot.app.ui.screens

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectTransformGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableDoubleStateOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.IntSize
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.R
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.GeoPoint
import com.hotspot.app.data.model.MapBounds
import com.hotspot.app.data.model.MapCanvasResponse
import com.hotspot.app.data.model.MapHeatBucket
import com.hotspot.app.data.model.MapHotspotItem
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemCard
import com.hotspot.app.ui.components.TabPills
import com.hotspot.app.ui.components.hotSpotFocusRing
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.MapProjection
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlin.math.roundToInt

/**
 * 附近地图（原生渲染；设计稿无独立地图画板，按 Web 自研地图契约实现：不接第三方 SDK，
 * 经纬度按视口 bounds 线性投影到画布）。
 *
 * 数据源：spec/api_spec_public.md → /public/v1/map/canvas（heatmap / hotspots / empty 三态互斥）；
 * 视口由默认浏览位置（client_config.defaults.default_location）+ 缩放跨度决定，拖动/捏合后
 * 250ms 防抖重新拉取；筛选 data_type = all/user/group/moment。
 */
@Composable
fun MapScreen(
    onBack: () -> Unit,
    onOpenUser: (String) -> Unit,
    onOpenGroup: (String) -> Unit,
    onOpenMoment: (String) -> Unit,
) {
    val colors = HotSpotTheme.colors
    val repo = AppContainer.nearbyRepository
    val scope = rememberCoroutineScope()

    var center by remember { mutableStateOf<GeoPoint?>(null) }
    var locationName by remember { mutableStateOf<String?>(null) }
    var locationError by remember { mutableStateOf<String?>(null) }
    var spanLat by remember { mutableDoubleStateOf(DEFAULT_SPAN_LAT) }
    var dataTypeIndex by remember { mutableIntStateOf(0) }
    var canvasSize by remember { mutableStateOf(IntSize.Zero) }

    var mode by remember { mutableStateOf("empty") }
    var heat by remember { mutableStateOf<List<MapHeatBucket>>(emptyList()) }
    var hotspots by remember { mutableStateOf<List<MapHotspotItem>>(emptyList()) }
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var selected by remember { mutableStateOf<MapHotspotItem?>(null) }

    val filters = listOf("全部", "用户", "群组", "瞬间")
    val dataTypes = listOf(null, "user", "group", "moment")
    val bounds: MapBounds? = center?.let { MapProjection.boundsOf(it, spanLat, spanLat * LONGITUDE_SPAN_RATIO) }

    suspend fun loadLocation() {
        locationError = null
        val location = AppContainer.defaultLocation()
        if (location?.point == null) {
            locationError = "无法获取默认浏览位置，请稍后重试"
            return
        }
        center = location.point
        locationName = location.locationName
    }

    LaunchedEffect(Unit) { loadLocation() }

    LaunchedEffect(bounds, dataTypeIndex) {
        val current = bounds ?: return@LaunchedEffect
        delay(MAP_FETCH_DEBOUNCE_MS)
        loading = true
        error = null
        when (val result = repo.mapCanvas(current, dataTypes[dataTypeIndex])) {
            is ApiResult.Ok -> {
                val response = result.data
                mode = response.mode
                heat = response.heatmap?.items.orEmpty()
                hotspots = response.hotspots?.items.orEmpty()
                selected = null
            }
            is ApiResult.Err -> error = result.error.userMessage
        }
        loading = false
    }

    Column(Modifier.fillMaxSize().background(colors.page)) {
        PageHeader(
            title = "附近地图",
            subtitle = locationName ?: "默认浏览位置",
            onBack = onBack,
        )
        Box(
            Modifier
                .fillMaxSize()
                .onSizeChanged { canvasSize = it }
                .pointerInput(Unit) {
                    detectTransformGestures { _, pan, zoom, _ ->
                        val currentCenter = center ?: return@detectTransformGestures
                        if (canvasSize.width > 0 && canvasSize.height > 0) {
                            center = MapProjection.panCenter(
                                center = currentCenter,
                                dxPx = pan.x,
                                dyPx = pan.y,
                                width = canvasSize.width.toFloat(),
                                height = canvasSize.height.toFloat(),
                                spanLat = spanLat,
                                spanLng = spanLat * LONGITUDE_SPAN_RATIO,
                            )
                        }
                        if (zoom != 1f) {
                            spanLat = MapProjection.zoomedSpan(spanLat, zoom).first
                        }
                    }
                },
        ) {
            MapBackdrop(
                bounds = bounds,
                heat = heat,
                mode = mode,
                modifier = Modifier.fillMaxSize(),
            )

            if (bounds != null && canvasSize.width > 0) {
                val density = androidx.compose.ui.platform.LocalDensity.current.density
                BoxWithConstraints(Modifier.fillMaxSize()) {
                    if (mode == "hotspots") {
                        hotspots.forEach { item ->
                            val point = item.location.point ?: return@forEach
                            val (x, y) = MapProjection.project(
                                point,
                                bounds,
                                canvasSize.width.toFloat(),
                                canvasSize.height.toFloat(),
                            )
                            HotspotPin(
                                item = item,
                                selected = selected?.objectId == item.objectId,
                                modifier = Modifier.offset {
                                    IntOffset(
                                        (x - PIN_SIZE_PX * density / 2).roundToInt(),
                                        (y - PIN_SIZE_PX * density / 2).roundToInt(),
                                    )
                                },
                                onClick = { selected = item },
                            )
                        }
                    }
                }
            }

            // 视口中心 = 默认浏览位置/用户当前浏览位置：蓝色定位点（对齐 Web 我的位置蓝点）。
            if (center != null) {
                Box(Modifier.align(Alignment.Center).size(18.dp).clip(CircleShape).background(colors.accent.copy(alpha = 0.25f)), contentAlignment = Alignment.Center) {
                    Box(Modifier.size(10.dp).clip(CircleShape).background(colors.accent).border(2.dp, Color.White, CircleShape))
                }
            }

            // 顶部筛选 + 右侧控制
            Row(
                modifier = Modifier
                    .align(Alignment.TopCenter)
                    .fillMaxWidth()
                    .padding(horizontal = Spacing.lg, vertical = Spacing.md),
            ) {
                TabPills(
                    options = filters,
                    selectedIndex = dataTypeIndex,
                    onSelected = { dataTypeIndex = it },
                    modifier = Modifier.fillMaxWidth(),
                )
            }
            Column(
                modifier = Modifier
                    .align(Alignment.CenterEnd)
                    .padding(end = Spacing.lg),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                MapControlButton(R.drawable.ic_map_zoom_in, "放大地图") {
                    spanLat = MapProjection.zoomedSpan(spanLat, 1.5f).first
                }
                Spacer(Modifier.height(Spacing.sm))
                MapControlButton(R.drawable.ic_map_zoom_out, "缩小地图") {
                    spanLat = MapProjection.zoomedSpan(spanLat, 0.67f).first
                }
                Spacer(Modifier.height(Spacing.sm))
                MapControlButton(R.drawable.ic_map_recenter, "回到默认位置") {
                    scope.launch { loadLocation() }
                    spanLat = DEFAULT_SPAN_LAT
                }
            }

            when {
                locationError != null && center == null -> ErrorState(
                    message = locationError ?: "加载失败",
                    onRetry = { scope.launch { loadLocation() } },
                    modifier = Modifier.align(Alignment.Center),
                )
                loading && heat.isEmpty() && hotspots.isEmpty() -> LoadingState(
                    message = "正在加载地图…",
                    modifier = Modifier.align(Alignment.Center),
                )
                error != null -> ErrorState(
                    message = error ?: "地图加载失败",
                    onRetry = { dataTypeIndex = dataTypeIndex },
                    modifier = Modifier.align(Alignment.Center),
                )
                !loading && mode == "empty" -> SystemCard(
                    modifier = Modifier
                        .align(Alignment.BottomCenter)
                        .padding(horizontal = Spacing.lg, vertical = Spacing.xl),
                ) {
                    Text(
                        text = "这片区域还没有可展示的内容",
                        style = MaterialTheme.typography.bodyMedium,
                        color = colors.textPrimary,
                    )
                    Text(
                        text = "拖动或缩小地图看看更大的范围",
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.textSecondary,
                        modifier = Modifier.padding(top = Spacing.xs),
                    )
                }
                loading -> Text(
                    text = "正在更新…",
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                    modifier = Modifier
                        .align(Alignment.TopEnd)
                        .padding(top = 64.dp, end = Spacing.lg),
                )
            }

            selected?.let { item ->
                HotspotDetailCard(
                    item = item,
                    modifier = Modifier
                        .align(Alignment.BottomCenter)
                        .padding(horizontal = Spacing.lg, vertical = Spacing.xl),
                    onOpen = {
                        when (item.objectType) {
                            "user" -> onOpenUser(item.objectId)
                            "group" -> onOpenGroup(item.objectId)
                            else -> onOpenMoment(item.objectId)
                        }
                    },
                )
            }
        }
    }
}

@Composable
private fun MapBackdrop(
    bounds: MapBounds?,
    heat: List<MapHeatBucket>,
    mode: String,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    Canvas(modifier) {
        drawRect(
            brush = Brush.linearGradient(
                colors = listOf(colors.ambient.sky, colors.ambient.horizon),
                start = Offset.Zero,
                end = Offset(size.width, size.height),
            ),
        )
        // 经纬网格：仅作空间参照，不做行政区概念（契约要求）。
        val step = size.width / 6f
        var x = 0f
        while (x <= size.width) {
            drawLine(colors.ambient.line.copy(alpha = 0.35f), Offset(x, 0f), Offset(x, size.height), 1f)
            x += step
        }
        var y = 0f
        while (y <= size.height) {
            drawLine(colors.ambient.line.copy(alpha = 0.35f), Offset(0f, y), Offset(size.width, y), 1f)
            y += step
        }
        if (mode != "heatmap" || bounds == null) return@Canvas
        val maxCount = heat.maxOfOrNull { it.totalCount }?.coerceAtLeast(1) ?: 1
        heat.forEach { bucket ->
            val (px, py) = MapProjection.project(bucket.location, bounds, size.width, size.height)
            val ratio = bucket.totalCount.toFloat() / maxCount
            val radius = size.width * (0.05f + 0.12f * ratio)
            drawCircle(
                brush = Brush.radialGradient(
                    colors = listOf(colors.markerMoment.copy(alpha = 0.10f + 0.22f * ratio), Color.Transparent),
                    center = Offset(px, py),
                    radius = radius,
                ),
                radius = radius,
                center = Offset(px, py),
            )
            drawCircle(
                color = colors.markerMoment.copy(alpha = 0.45f),
                radius = 3.dp.toPx() + 4.dp.toPx() * ratio,
                center = Offset(px, py),
            )
        }
    }
}

@Composable
private fun HotspotPin(
    item: MapHotspotItem,
    selected: Boolean,
    modifier: Modifier = Modifier,
    onClick: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    val (tint, icon) = when (item.objectType) {
        "user" -> colors.markerUser to R.drawable.ic_nav_me
        "group" -> colors.markerGroup to R.drawable.ic_row_groups
        else -> colors.markerMoment to R.drawable.ic_nav_home
    }
    Box(
        modifier = modifier
            .size(if (selected) 44.dp else 36.dp)
            .shadow(2.dp, CircleShape)
            .clip(CircleShape)
            .background(tint)
            .border(2.dp, Color.White, CircleShape)
            .clickable(onClick = onClick)
            .hotSpotFocusRing(cornerRadius = 22.dp),
        contentAlignment = Alignment.Center,
    ) {
        Icon(
            painter = painterResource(icon),
            contentDescription = item.title.ifBlank { "地图标记" },
            modifier = Modifier.size(Dimens.iconSm),
            tint = Color.White,
        )
    }
}

@Composable
private fun HotspotDetailCard(
    item: MapHotspotItem,
    modifier: Modifier = Modifier,
    onOpen: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    SystemCard(modifier = modifier.fillMaxWidth()) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Avatar(name = item.title, imageUrl = item.coverUrl, size = Dimens.avatarSm)
            Spacer(Modifier.width(Spacing.sm))
            Column(Modifier.weight(1f)) {
                Text(
                    text = item.title.ifBlank { "地图条目" },
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                    color = colors.textPrimary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                Text(
                    text = item.subtitle ?: when (item.objectType) {
                        "user" -> "用户"
                        "group" -> "群组"
                        else -> "瞬间"
                    },
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.textSecondary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            Spacer(Modifier.width(Spacing.sm))
            SystemButton(
                text = "查看",
                onClick = onOpen,
                variant = ButtonVariant.PRIMARY,
                size = ButtonSize.SM,
            )
        }
    }
}

@Composable
private fun MapControlButton(iconRes: Int, description: String, onClick: () -> Unit) {
    val colors = HotSpotTheme.colors
    Box(
        modifier = Modifier
            .size(Dimens.minTouchTarget)
            .clip(CircleShape)
            .clickable(onClick = onClick)
            .hotSpotFocusRing(cornerRadius = 22.dp),
        contentAlignment = Alignment.Center,
    ) {
        Box(
            modifier = Modifier
                .size(38.dp)
                .shadow(2.dp, CircleShape)
                .clip(CircleShape)
                .background(colors.surface),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painter = painterResource(iconRes),
                contentDescription = description,
                modifier = Modifier.size(Dimens.iconSm),
                tint = colors.textPrimary,
            )
        }
    }
}

private const val DEFAULT_SPAN_LAT = 0.05
private const val LONGITUDE_SPAN_RATIO = 1.35
private const val MAP_FETCH_DEBOUNCE_MS = 250L
private const val PIN_SIZE_PX = 36f
