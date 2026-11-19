package com.hotspot.app.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.ColorScheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.Immutable
import androidx.compose.runtime.remember
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color

/** 主题族：与 Web 三主题一一对应（vibrant 默认 / forest 清野 / inkwash 水墨）。 */
enum class ThemeId(val key: String, val label: String) {
    VIBRANT("vibrant", "星际"),
    FOREST("forest", "清野"),
    INKWASH("inkwash", "水墨"),
    ;

    companion object {
        fun fromKey(key: String?): ThemeId = entries.firstOrNull { it.key == key } ?: VIBRANT
    }
}

/** 明暗三档：跟随系统 / 手动浅色 / 手动深色（对齐 design-system §3.3 DarkModePreference）。 */
enum class DarkModePref(val key: String, val label: String) {
    AUTO("auto", "跟随系统"),
    LIGHT("light", "浅色"),
    DARK("dark", "深色"),
    ;

    companion object {
        fun fromKey(key: String?): DarkModePref = entries.firstOrNull { it.key == key } ?: AUTO
    }
}

@Immutable
data class UiTheme(
    val themeId: ThemeId = ThemeId.VIBRANT,
    val darkMode: DarkModePref = DarkModePref.AUTO,
)

/**
 * 语义色扩展：Web --cst-* 语义 token 的 Compose 消费面（Material ColorScheme 覆盖不到的
 * 软岛表面 / 状态四色 / on-soft 文字 / 输入族 / 导航三态等）。所有值来自生成 token，不写死。
 */
@Immutable
data class HotSpotColors(
    val themeId: ThemeId,
    val isDark: Boolean,
    // 画布与表面（--cst-surface-*）
    val page: Color,
    val surface: Color,
    val island: Color,
    val subtle: Color,
    val chat: Color,
    val overlay: Color,
    val overlayStrong: Color,
    val scrim: Color,
    val glassBadge: Color,
    // 文字（--cst-text-* / --cst-content-*）
    val textPrimary: Color,
    val textSecondary: Color,
    val textOnAccent: Color,
    val textOnStatus: Color,
    val textAccentOnSoft: Color,
    val textDangerOnSoft: Color,
    val textWarningOnSoft: Color,
    val textOnlineOnSoft: Color,
    val contentInverse: Color,
    val contentInverseMuted: Color,
    val contentInverseSecondary: Color,
    // 品牌与描边（--cst-accent-* / --cst-border-*）
    val accent: Color,
    val accentSoft: Color,
    val accentSoftBg: Color,
    val accentBorder: Color,
    val accentBorderSoft: Color,
    val border: Color,
    val borderStrong: Color,
    val borderSubtle: Color,
    // 状态四色 + 淡底/淡边（--cst-status-*）
    val online: Color,
    val danger: Color,
    val warning: Color,
    val onlineBg: Color,
    val onlineBorder: Color,
    val dangerBg: Color,
    val dangerBorder: Color,
    val warningBg: Color,
    val warningBorder: Color,
    // 输入族（--cst-field-*）
    val fieldBg: Color,
    val fieldBgDisabled: Color,
    val fieldBorder: Color,
    val fieldBorderHover: Color,
    val fieldBorderFocus: Color,
    val fieldRing: Color,
    val fieldRingDanger: Color,
    // 交互（--cst-interactive-* / 投影原料）
    val interactionHover: Color,
    val interactionActive: Color,
    val shadowDark: Color,
    // 地图标记（--cst-marker-*）
    val markerUser: Color,
    val markerGroup: Color,
    val markerMoment: Color,
    // 导航三态（--cst-surface-navigation-* / --cst-navigation-*）
    val navContent: Color,
    val navContentActive: Color,
    val navIndicator: Color,
    val navHoverSurface: Color,
    val navActiveSurface: Color,
    val navActiveHoverSurface: Color,
    // 氛围画布（--cst-ambient-*，design-system §16）
    val ambient: AmbientPalette,
)

val LocalHotSpotColors = staticCompositionLocalOf<HotSpotColors> {
    error("HotSpotColors 未提供：请用 HotSpotTheme { ... } 包裹 UI")
}

private fun paletteOf(themeId: ThemeId, isDark: Boolean): PaletteColors = when (themeId) {
    ThemeId.VIBRANT -> if (isDark) Tokens.Palette.VibrantDark else Tokens.Palette.VibrantLight
    ThemeId.FOREST -> if (isDark) Tokens.Palette.ForestDark else Tokens.Palette.ForestLight
    ThemeId.INKWASH -> if (isDark) Tokens.Palette.InkwashDark else Tokens.Palette.InkwashLight
}

private fun ambientOf(themeId: ThemeId, isDark: Boolean): AmbientPalette = when (themeId) {
    ThemeId.VIBRANT -> if (isDark) Tokens.Ambient.VibrantDark else Tokens.Ambient.VibrantLight
    ThemeId.FOREST -> if (isDark) Tokens.Ambient.ForestDark else Tokens.Ambient.ForestLight
    ThemeId.INKWASH -> if (isDark) Tokens.Ambient.InkwashDark else Tokens.Ambient.InkwashLight
}

private fun semanticOf(themeId: ThemeId, isDark: Boolean): SemanticPalette = when (themeId) {
    ThemeId.VIBRANT -> if (isDark) Tokens.Semantic.VibrantDark else Tokens.Semantic.VibrantLight
    ThemeId.FOREST -> if (isDark) Tokens.Semantic.ForestDark else Tokens.Semantic.ForestLight
    ThemeId.INKWASH -> if (isDark) Tokens.Semantic.InkwashDark else Tokens.Semantic.InkwashLight
}

private fun buildHotSpotColors(themeId: ThemeId, isDark: Boolean): HotSpotColors {
    val s = semanticOf(themeId, isDark)
    val p = paletteOf(themeId, isDark)
    return HotSpotColors(
        themeId = themeId,
        isDark = isDark,
        page = s.surfacePage,
        surface = s.surfacePrimary,
        island = s.surfaceIsland,
        subtle = s.surfaceSubtle,
        chat = s.surfaceChat,
        overlay = s.surfaceOverlay,
        overlayStrong = s.surfaceOverlayStrong,
        scrim = s.surfaceScrim,
        glassBadge = s.surfaceGlassBadge,
        textPrimary = s.textPrimary,
        textSecondary = s.textSecondary,
        textOnAccent = s.textOnAccent,
        textOnStatus = s.textOnStatus,
        textAccentOnSoft = s.textAccentOnSoft,
        textDangerOnSoft = s.textStatusDangerOnSoft,
        textWarningOnSoft = s.textStatusWarningOnSoft,
        textOnlineOnSoft = s.textStatusOnlineOnSoft,
        contentInverse = s.contentInverse,
        contentInverseMuted = s.contentInverseMuted,
        contentInverseSecondary = s.contentInverseSecondary,
        accent = s.accentDefault,
        accentSoft = s.accentSoft,
        accentSoftBg = s.accentSoftBg,
        accentBorder = s.accentBorder,
        accentBorderSoft = s.accentBorderSoft,
        border = s.borderDefault,
        borderStrong = s.borderStrong,
        borderSubtle = s.borderSubtle,
        online = s.statusOnline,
        danger = s.statusDanger,
        warning = s.statusWarning,
        onlineBg = s.statusOnlineBg,
        onlineBorder = s.statusOnlineBorder,
        dangerBg = s.statusDangerBg,
        dangerBorder = s.statusDangerBorder,
        warningBg = s.statusWarningBg,
        warningBorder = s.statusWarningBorder,
        fieldBg = s.fieldBg,
        fieldBgDisabled = s.fieldBgDisabled,
        fieldBorder = s.fieldBorder,
        fieldBorderHover = s.fieldBorderHover,
        fieldBorderFocus = s.fieldBorderFocus,
        fieldRing = s.fieldRing,
        fieldRingDanger = s.fieldRingDanger,
        interactionHover = s.interactiveHover,
        interactionActive = s.interactiveActive,
        shadowDark = p.shadowDark,
        markerUser = s.markerUser,
        markerGroup = s.markerGroup,
        markerMoment = s.markerMoment,
        navContent = s.navigationContent,
        navContentActive = s.navigationContentActive,
        navIndicator = s.navigationActiveIndicator,
        navHoverSurface = s.surfaceNavigationHover,
        navActiveSurface = s.surfaceNavigationActive,
        navActiveHoverSurface = s.surfaceNavigationActiveHover,
        ambient = ambientOf(themeId, isDark),
    )
}

private fun buildColorScheme(colors: HotSpotColors): ColorScheme {
    val base = if (colors.isDark) darkColorScheme() else lightColorScheme()
    return base.copy(
        primary = colors.accent,
        onPrimary = colors.textOnAccent,
        primaryContainer = colors.accentSoftBg,
        onPrimaryContainer = colors.textAccentOnSoft,
        secondary = colors.online,
        onSecondary = colors.textOnStatus,
        secondaryContainer = colors.onlineBg,
        onSecondaryContainer = colors.textOnlineOnSoft,
        tertiary = colors.warning,
        onTertiary = colors.textOnStatus,
        tertiaryContainer = colors.warningBg,
        onTertiaryContainer = colors.textWarningOnSoft,
        error = colors.danger,
        onError = colors.contentInverse,
        errorContainer = colors.dangerBg,
        onErrorContainer = colors.textDangerOnSoft,
        background = colors.page,
        onBackground = colors.textPrimary,
        surface = colors.surface,
        onSurface = colors.textPrimary,
        surfaceVariant = colors.subtle,
        onSurfaceVariant = colors.textSecondary,
        surfaceTint = colors.accent,
        outline = colors.border,
        outlineVariant = colors.borderSubtle,
        scrim = colors.scrim,
        inverseSurface = colors.contentInverse,
        inverseOnSurface = colors.textPrimary,
        inversePrimary = colors.accent,
    )
}

/**
 * 应用主题：3 主题 × 明暗，通过 local composition 暴露语义色与动效。
 * 消费方用 HotSpotTheme.colors / MaterialTheme.colorScheme 取色，禁止硬编码。
 */
object HotSpotTheme {
    val colors: HotSpotColors
        @Composable get() = LocalHotSpotColors.current

    val motion: HotSpotMotion
        @Composable get() = LocalHotSpotMotion.current
}

@Composable
fun HotSpotTheme(
    uiTheme: UiTheme = UiTheme(),
    content: @Composable () -> Unit,
) {
    val systemDark = isSystemInDarkTheme()
    val isDark = when (uiTheme.darkMode) {
        DarkModePref.AUTO -> systemDark
        DarkModePref.LIGHT -> false
        DarkModePref.DARK -> true
    }
    val colors = remember(uiTheme.themeId, isDark) { buildHotSpotColors(uiTheme.themeId, isDark) }
    val scheme = remember(colors) { buildColorScheme(colors) }
    val motion = rememberHotSpotMotion()
    CompositionLocalProvider(
        LocalHotSpotColors provides colors,
        LocalHotSpotMotion provides motion,
    ) {
        MaterialTheme(
            colorScheme = scheme,
            typography = HotSpotTypography,
            shapes = HotSpotShapes,
            content = content,
        )
    }
}
