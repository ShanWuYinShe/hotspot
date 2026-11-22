package com.hotspot.app.util

import com.hotspot.app.data.model.GeoPoint
import com.hotspot.app.data.model.MapBounds

/**
 * 地图经纬度 ↔ 屏幕坐标投影（与 Web 自研地图同口径：按 bounds 线性投影，
 * 经纬度跨度等比映射到画布宽高；不接第三方地图 SDK）。
 */
object MapProjection {
    fun boundsOf(center: GeoPoint, spanLat: Double, spanLng: Double): MapBounds {
        val halfLat = spanLat / 2
        val halfLng = spanLng / 2
        return MapBounds(
            north = (center.latitude + halfLat).coerceAtMost(90.0),
            south = (center.latitude - halfLat).coerceAtLeast(-90.0),
            east = center.longitude + halfLng,
            west = center.longitude - halfLng,
        )
    }

    /** 返回画布像素坐标；bounds 退化（跨度为 0）时返回画布中心。 */
    fun project(location: GeoPoint, bounds: MapBounds, width: Float, height: Float): Pair<Float, Float> {
        val spanLng = bounds.east - bounds.west
        val spanLat = bounds.north - bounds.south
        val x = if (spanLng == 0.0) width / 2 else (((location.longitude - bounds.west) / spanLng) * width).toFloat()
        val y = if (spanLat == 0.0) height / 2 else (((bounds.north - location.latitude) / spanLat) * height).toFloat()
        return x to y
    }

    /** 手势平移后的新中心：拖拽 (dx,dy) 像素 = 地图内容跟随手指移动。 */
    fun panCenter(
        center: GeoPoint,
        dxPx: Float,
        dyPx: Float,
        width: Float,
        height: Float,
        spanLat: Double,
        spanLng: Double,
    ): GeoPoint {
        if (width <= 0f || height <= 0f) return center
        val deltaLat = (dyPx / height) * spanLat
        val deltaLng = -(dxPx / width) * spanLng
        return GeoPoint(
            latitude = (center.latitude + deltaLat).coerceIn(-85.0, 85.0),
            longitude = ((center.longitude + deltaLng + 180.0).mod(360.0)) - 180.0,
        )
    }

    /** 捏合缩放后的跨度（scale > 1 放大）；限制在 0.002°~0.5° 纬度跨度（0.5° ≈ 55km，可看到 heatmap 聚合）。 */
    fun zoomedSpan(spanLat: Double, scale: Float): Pair<Double, Double> {
        val safeScale = scale.coerceIn(0.5f, 2f)
        val next = (spanLat / safeScale).coerceIn(0.002, 0.5)
        // 经度跨度按纬度余弦修正，保持视觉比例（低纬近似，避免极区拉伸）。
        return next to (next * 1.35)
    }
}
