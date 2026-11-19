package com.hotspot.app.ui.theme

import androidx.compose.ui.graphics.Color

/**
 * 氛围画布接口（design-system §16）：三主题 × 明暗的天光/微光/地平线/暗角等装饰色。
 * 装饰层专用，不参与正文配色；alpha 很低，保证文字对比度不被干扰。
 */
interface AmbientPalette {
    val dot: Color
    val emblem: Color
    val glow1: Color
    val glow2: Color
    val horizon: Color
    val ink1: Color
    val ink2: Color
    val ink3: Color
    val line: Color
    val sky: Color
    val sun: Color
    val vignette: Color
}
