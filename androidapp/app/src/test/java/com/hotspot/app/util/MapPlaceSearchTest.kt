package com.hotspot.app.util

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class MapPlaceSearchTest {
    @Test
    fun parsesAsciiCommaCoordinate() {
        val point = MapPlaceSearch.parseCoordinate("34.2,117.28")
        assertEquals(34.2, point!!.latitude, 1e-9)
        assertEquals(117.28, point.longitude, 1e-9)
    }

    @Test
    fun parsesChineseCommaAndWhitespace() {
        val point = MapPlaceSearch.parseCoordinate(" 34.2 ， 117.28 ")
        assertEquals(34.2, point!!.latitude, 1e-9)
        assertEquals(117.28, point.longitude, 1e-9)
    }

    @Test
    fun rejectsOutOfRangeOrNonNumericInput() {
        assertNull(MapPlaceSearch.parseCoordinate("91,117.28"))
        assertNull(MapPlaceSearch.parseCoordinate("34.2,181"))
        assertNull(MapPlaceSearch.parseCoordinate("云龙湖"))
        assertNull(MapPlaceSearch.parseCoordinate("34.2"))
    }

    @Test
    fun formatsFiveDecimalPlaces() {
        assertEquals(
            "34.20000, 117.28000",
            MapPlaceSearch.formatCoordinate(com.hotspot.app.data.model.GeoPoint(34.2, 117.28)),
        )
    }
}
