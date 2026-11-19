package com.hotspot.app.ui.components

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import com.hotspot.app.ui.theme.HotSpotTheme

/**
 * 氛围画布（design-system §16）：天光 / 双层径向微光 / 暗角。装饰层唯一实现，
 * 只消费 --cst-ambient-* 生成 token；静态绘制（无逐帧动画），不参与交互与布局。
 */
@Composable
fun AmbientCanvas(modifier: Modifier = Modifier) {
    val colors = HotSpotTheme.colors
    val ambient = colors.ambient
    Canvas(modifier.fillMaxSize()) {
        drawRect(colors.page)
        val maxDim = size.maxDimension
        // 天光（--cst-ambient-sky）：顶部中央向下扩散
        drawCircle(
            brush = Brush.radialGradient(
                colors = listOf(ambient.sky, Color.Transparent),
                center = Offset(size.width * 0.5f, 0f),
                radius = maxDim * 0.55f,
            ),
            radius = maxDim * 0.55f,
            center = Offset(size.width * 0.5f, 0f),
        )
        // 顶部广角微光（--cst-ambient-glow-1）
        drawCircle(
            brush = Brush.radialGradient(
                colors = listOf(ambient.glow1, Color.Transparent),
                center = Offset(size.width * 0.12f, size.height * 0.06f),
                radius = maxDim * 0.45f,
            ),
            radius = maxDim * 0.45f,
            center = Offset(size.width * 0.12f, size.height * 0.06f),
        )
        // 底部地平线微光（--cst-ambient-glow-2）
        drawCircle(
            brush = Brush.radialGradient(
                colors = listOf(ambient.glow2, Color.Transparent),
                center = Offset(size.width * 0.92f, size.height * 0.88f),
                radius = maxDim * 0.42f,
            ),
            radius = maxDim * 0.42f,
            center = Offset(size.width * 0.92f, size.height * 0.88f),
        )
        // 暗角（--cst-ambient-vignette）：边缘收束
        drawRect(
            brush = Brush.radialGradient(
                colors = listOf(Color.Transparent, ambient.vignette),
                center = Offset(size.width * 0.5f, size.height * 0.45f),
                radius = maxDim * 0.75f,
            ),
        )
    }
}
