package com.hotspot.app.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** 地图视口边界（spec/api_spec_public.md → Bounds；north >= south、east >= west）。 */
@Serializable
data class MapBounds(
    val north: Double = 0.0,
    val south: Double = 0.0,
    val east: Double = 0.0,
    val west: Double = 0.0,
)

@Serializable
data class MapViewport(val bounds: MapBounds = MapBounds())

@Serializable
data class MapHeatBreakdown(
    val user: Int? = null,
    val group: Int? = null,
    val moment: Int? = null,
)

@Serializable
data class MapHeatBucket(
    @SerialName("cell_token") val cellToken: String = "",
    val level: Int = 0,
    val location: GeoPoint = GeoPoint(),
    @SerialName("total_count") val totalCount: Int = 0,
    val breakdown: MapHeatBreakdown? = null,
)

@Serializable
data class MapHeatCanvas(val items: List<MapHeatBucket> = emptyList()) {
    val maxCount: Int get() = items.maxOfOrNull { it.totalCount } ?: 0
}

@Serializable
data class MapHotspotItem(
    @SerialName("object_type") val objectType: String = "moment",
    @SerialName("object_id") val objectId: String = "",
    val title: String = "",
    val subtitle: String? = null,
    @SerialName("cover_url") val coverUrl: String? = null,
    val location: RelativeLocation = RelativeLocation(),
    @SerialName("created_at") val createdAt: String? = null,
)

/**
 * 地图视口渲染响应（spec/api_spec_public.md → MapCanvasResponse tagged union）：
 * mode = 'heatmap' | 'hotspots' | 'empty'，heatmap 与 hotspots 互斥。
 */
@Serializable
data class MapCanvasResponse(
    val mode: String = "empty",
    val heatmap: MapHeatCanvas? = null,
    val hotspots: Page<MapHotspotItem>? = null,
)
