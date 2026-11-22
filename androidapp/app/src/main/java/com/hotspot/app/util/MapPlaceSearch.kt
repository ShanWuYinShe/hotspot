package com.hotspot.app.util

import com.hotspot.app.data.model.GeoPoint
import java.util.Locale

/** 地图选点页的纯函数：坐标解析/格式化（可 JVM 单测，不依赖 Android）。 */
object MapPlaceSearch {
    /** 支持「纬度,经度」与中文逗号；越界或非数字返回 null。 */
    fun parseCoordinate(input: String): GeoPoint? {
        val parts = input.split(',', '，')
        if (parts.size != 2) return null
        val lat = parts[0].trim().toDoubleOrNull() ?: return null
        val lng = parts[1].trim().toDoubleOrNull() ?: return null
        if (lat !in -90.0..90.0 || lng !in -180.0..180.0) return null
        return GeoPoint(latitude = lat, longitude = lng)
    }

    fun formatCoordinate(point: GeoPoint): String =
        String.format(Locale.US, "%.5f, %.5f", point.latitude, point.longitude)
}
