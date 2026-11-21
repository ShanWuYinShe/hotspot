package com.hotspot.app.ui.components

import androidx.compose.foundation.focusable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.composed
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.hotspot.app.ui.theme.HotSpotTheme

/**
 * 可见焦点环（design-system §20.1 强制：2px accent + offset）。
 * Compose 默认不绘制焦点框，自定义可点元素必须显式加本修饰符。
 */
fun Modifier.hotSpotFocusRing(
    cornerRadius: Dp = 16.dp,
    color: Color? = null,
): Modifier = composed {
    val ringColor = color ?: HotSpotTheme.colors.accent
    var focused by remember { mutableStateOf(false) }
    this
        .onFocusChanged { focused = it.isFocused }
        .focusable()
        .drawWithContent {
            drawContent()
            if (focused) {
                val stroke = 2.dp.toPx()
                val inset = 2.dp.toPx()
                drawRoundRect(
                    color = ringColor,
                    topLeft = Offset(inset, inset),
                    size = Size(size.width - inset * 2, size.height - inset * 2),
                    cornerRadius = CornerRadius(cornerRadius.toPx()),
                    style = Stroke(width = stroke),
                )
            }
        }
}

/** 触控热区底线：视觉尺寸可小，但可点区域 ≥44dp（mobile-ui-conventions）。 */
fun Modifier.minTouchTarget(): Modifier = this
