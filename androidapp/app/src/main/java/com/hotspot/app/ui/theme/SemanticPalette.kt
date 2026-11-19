package com.hotspot.app.ui.theme

import androidx.compose.ui.graphics.Color

/**
 * 语义色统一接口：6 组主题 × 明暗（Tokens.Semantic.*）均实现本接口。
 * Theme.kt 据此构建与 Web --cst-* 语义 token 一一对应的 HotSpotColors / Material3 ColorScheme。
 * 本文件手写（接口签名与 Tokens.kt 生成物同源：任何 token 增删都会在编译期暴露）。
 */
interface SemanticPalette {
    val accentBorder: Color
    val accentBorderSoft: Color
    val accentDefault: Color
    val accentSoft: Color
    val accentSoftBg: Color
    val borderDefault: Color
    val borderStrong: Color
    val borderSubtle: Color
    val contentInverse: Color
    val contentInverseMuted: Color
    val contentInverseSecondary: Color
    val contentOnMarker: Color
    val fieldBg: Color
    val fieldBgDisabled: Color
    val fieldBorder: Color
    val fieldBorderFocus: Color
    val fieldBorderHover: Color
    val fieldRing: Color
    val fieldRingDanger: Color
    val glowAccent: Color
    val glowDanger: Color
    val interactiveActive: Color
    val interactiveActiveFill: Color
    val interactiveHover: Color
    val interactiveHoverFill: Color
    val markerGroup: Color
    val markerGroupBg: Color
    val markerGroupBorder: Color
    val markerGroupGlow: Color
    val markerMoment: Color
    val markerMomentBg: Color
    val markerMomentBorder: Color
    val markerMomentGlow: Color
    val markerUser: Color
    val markerUserBg: Color
    val markerUserBorder: Color
    val markerUserGlow: Color
    val navigationActiveIndicator: Color
    val navigationContent: Color
    val navigationContentActive: Color
    val statusDanger: Color
    val statusDangerBg: Color
    val statusDangerBorder: Color
    val statusOnline: Color
    val statusOnlineBg: Color
    val statusOnlineBorder: Color
    val statusWarning: Color
    val statusWarningBg: Color
    val statusWarningBorder: Color
    val surfaceCardMenu: Color
    val surfaceChat: Color
    val surfaceGlassBadge: Color
    val surfaceIsland: Color
    val surfaceMediaScrimStrong: Color
    val surfaceNavGlass: Color
    val surfaceNavigationActive: Color
    val surfaceNavigationActiveHover: Color
    val surfaceNavigationHover: Color
    val surfaceOverlay: Color
    val surfaceOverlayStrong: Color
    val surfacePage: Color
    val surfacePanel: Color
    val surfacePrimary: Color
    val surfaceRaised: Color
    val surfaceScrim: Color
    val surfaceSubtle: Color
    val surfaceToggleOff: Color
    val surfaceToggleThumb: Color
    val textAccentOnSoft: Color
    val textOnAccent: Color
    val textOnStatus: Color
    val textPrimary: Color
    val textSecondary: Color
    val textStatusDangerOnSoft: Color
    val textStatusOnlineOnSoft: Color
    val textStatusWarningOnSoft: Color
}
