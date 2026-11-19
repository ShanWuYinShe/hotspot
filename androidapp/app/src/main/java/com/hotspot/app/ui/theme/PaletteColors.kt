package com.hotspot.app.ui.theme

import androidx.compose.ui.graphics.Color

/**
 * 原始色板接口：6 组主题 × 明暗（Tokens.Palette.*）均实现本接口。
 * 组件优先消费 SemanticPalette；需要色板原料（如阴影 dark/light 色）时用它。
 */
interface PaletteColors {
    val accentPrimary: Color
    val accentSoft: Color
    val accentSoftBorder: Color
    val activeHover: Color
    val bg: Color
    val border: Color
    val borderPressed: Color
    val chatBg: Color
    val glow: Color
    val hot: Color
    val hover: Color
    val mapGroup: Color
    val mapMoment: Color
    val mapUser: Color
    val onAccent: Color
    val online: Color
    val overlayStrong: Color
    val pageBg: Color
    val shadowDark: Color
    val shadowLight: Color
    val text: Color
    val textMuted: Color
    val warning: Color
}
