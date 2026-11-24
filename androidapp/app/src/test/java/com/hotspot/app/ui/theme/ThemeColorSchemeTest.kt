package com.hotspot.app.ui.theme

import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * ColorScheme 表面族映射不变量。
 *
 * Material3 容器组件（AlertDialog / BottomSheet / DropdownMenu 等）默认取 surfaceContainer*；
 * 这些字段一旦缺失映射，就会回落到 lightColorScheme()/darkColorScheme() 的基线紫色（PR #178）。
 * 本测试对 6 套主题逐字段断言映射结果，防止后续改动把容器色退回默认值。
 */
class ThemeColorSchemeTest {

    @Test
    fun surfaceFamily_mapsToHotSpotSurfaces_forEveryThemeCombo() {
        ThemeId.entries.forEach { theme ->
            listOf(false, true).forEach { isDark ->
                val colors = buildHotSpotColors(theme, isDark)
                val scheme = buildColorScheme(colors)
                val label = theme.key + "-" + if (isDark) "dark" else "light"

                assertEquals(label, colors.surface, scheme.surface)
                assertEquals(label, colors.surface, scheme.surfaceBright)
                assertEquals(label, colors.page, scheme.surfaceDim)
                assertEquals(label, colors.page, scheme.surfaceContainerLowest)
                assertEquals(label, colors.island, scheme.surfaceContainerLow)
                assertEquals(label, colors.surface, scheme.surfaceContainer)
                assertEquals(label, colors.surface, scheme.surfaceContainerHigh)
                assertEquals(label, colors.island, scheme.surfaceContainerHighest)
            }
        }
    }
}
