package com.hotspot.app.util

import com.hotspot.app.data.model.GeoPoint
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class MapProjectionTest {
    private val center = GeoPoint(latitude = 34.20, longitude = 117.28)
    private val spanLat = 0.04
    private val spanLng = 0.054
    private val bounds = MapProjection.boundsOf(center, spanLat, spanLng)

    @Test
    fun centerProjectsToCanvasCenter() {
        val (x, y) = MapProjection.project(center, bounds, 400f, 300f)
        assertEquals(200f, x, 0.01f)
        assertEquals(150f, y, 0.01f)
    }

    @Test
    fun northWestCornerProjectsToOrigin() {
        val northWest = GeoPoint(bounds.north, bounds.west)
        val (x, y) = MapProjection.project(northWest, bounds, 400f, 300f)
        assertEquals(0f, x, 0.01f)
        assertEquals(0f, y, 0.01f)
    }

    @Test
    fun panMovesCenterFollowingFinger() {
        // 手指向右下拖动 => 地图内容跟随移动 => 视口中心向西北移动。
        val moved = MapProjection.panCenter(center, dxPx = 100f, dyPx = 60f, width = 400f, height = 300f, spanLat = spanLat, spanLng = spanLng)
        assertTrue(moved.latitude > center.latitude)
        assertTrue(moved.longitude < center.longitude)
    }

    @Test
    fun zoomInShrinksSpanAndZoomOutGrowsIt() {
        val (zoomedLat, _) = MapProjection.zoomedSpan(spanLat, 2f)
        assertTrue(zoomedLat < spanLat)
        val (outLat, _) = MapProjection.zoomedSpan(spanLat, 0.5f)
        assertTrue(outLat > spanLat)
    }

    @Test
    fun spanIsClampedToSafeRange() {
        assertEquals(0.002, MapProjection.zoomedSpan(0.0001, 2f).first, 1e-9)
        assertEquals(0.5, MapProjection.zoomedSpan(10.0, 0.5f).first, 1e-9)
    }

    @Test
    fun longitudeBoundsStayWithinSpecRange() {
        // 贴近日期变更线时经度必须夹在 spec 允许的 [-180,180]，否则 map/canvas 会拒绝越界视口。
        val nearEast = MapProjection.boundsOf(
            GeoPoint(latitude = 0.0, longitude = 179.99),
            spanLat = 0.05,
            spanLng = 0.0675,
        )
        assertEquals(180.0, nearEast.east, 1e-9)
        assertTrue(nearEast.west <= nearEast.east)

        val nearWest = MapProjection.boundsOf(
            GeoPoint(latitude = 0.0, longitude = -179.99),
            spanLat = 0.05,
            spanLng = 0.0675,
        )
        assertEquals(-180.0, nearWest.west, 1e-9)
        assertTrue(nearWest.east >= nearWest.west)
    }

    @Test
    fun latitudeIsClampedWhenPanningFarNorth() {
        val moved = MapProjection.panCenter(center, dxPx = 0f, dyPx = 100000f, width = 400f, height = 300f, spanLat = spanLat, spanLng = spanLng)
        assertTrue(moved.latitude <= 85.0)
    }
}
