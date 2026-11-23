package com.hotspot.app.util

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * 群详情纯逻辑单测（不依赖 Android / Compose）：
 * - 指标「距你」格式化（<1km 用 m，否则 km 一位小数；null 显示占位由 UI 层负责）
 * - 成员角色徽章文案（owner/admin 显示，member 不显示）
 * - 公告管理入口权限（仅 owner/admin）
 */
class GroupFormatTest {

    @Test
    fun formatDistanceShort_nullOrInvalid_returnsNull() {
        assertNull(formatDistanceShort(null))
        assertNull(formatDistanceShort(Double.NaN))
        assertNull(formatDistanceShort(-1.0))
    }

    @Test
    fun formatDistanceShort_metersBelowOneKilometer_usesMeterUnit() {
        assertEquals("0m", formatDistanceShort(0.0))
        assertEquals("426m", formatDistanceShort(426.0))
        assertEquals("999m", formatDistanceShort(999.4))
    }

    @Test
    fun formatDistanceShort_fromOneKilometer_usesSingleDecimalKilometer() {
        assertEquals("1.0km", formatDistanceShort(1000.0))
        assertEquals("1.5km", formatDistanceShort(1499.0))
        assertEquals("327.8km", formatDistanceShort(327847.0))
    }

    @Test
    fun groupRoleLabel_mapsSpecRoleToBadgeText() {
        assertEquals("群主", groupRoleLabel("owner"))
        assertEquals("管理员", groupRoleLabel("admin"))
        assertNull(groupRoleLabel("member"))
        assertNull(groupRoleLabel(null))
        assertNull(groupRoleLabel("unknown"))
    }

    @Test
    fun canManageGroupAnnouncements_onlyOwnerOrAdmin() {
        assertTrue(canManageGroupAnnouncements("owner"))
        assertTrue(canManageGroupAnnouncements("admin"))
        assertFalse(canManageGroupAnnouncements("member"))
        assertFalse(canManageGroupAnnouncements(null))
        assertFalse(canManageGroupAnnouncements("unknown"))
    }
}
