package com.hotspot.app.ui.theme

import android.content.Context
import android.provider.Settings
import androidx.compose.animation.core.CubicBezierEasing
import androidx.compose.animation.core.TweenSpec
import androidx.compose.animation.core.tween
import androidx.compose.runtime.Composable
import androidx.compose.runtime.Immutable
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.remember
import androidx.compose.ui.platform.LocalContext

/**
 * 动效常量：时长/曲线来自生成 token（shared.css / utilities.css），
 * 行为时长（非视觉 token）单独登记。
 */
object MotionSpec {
    /** 回顶后 200ms 内不再回闪等行为时长（对齐 Web BackToTop；非视觉 token，不参与生成）。 */
    const val BEHAVIOR_SHORT_MS = 200

    /** 按压缩放：按钮/行内图标统一 0.97（Figma 组件库按压档）。 */
    const val PRESS_SCALE = 0.97f

    /** 卡片按压缩放来自主题 token（--cst-card-tap-scale，当前三主题均为 1.0，稳定不脱靶）。 */
    const val CARD_PRESS_SCALE = 1.0f

    /** 列表/骨架等脉冲周期。 */
    const val SKELETON_MS = Tokens.Motion.skeleton
}

@Immutable
data class HotSpotMotion(val reduced: Boolean) {
    /** reduced-motion 下位移动画时长归零（保留透明度淡入的语义由调用方决定）。 */
    fun duration(ms: Int): Int = if (reduced) 0 else ms

    fun <T> tween(ms: Int): TweenSpec<T> =
        tween(durationMillis = duration(ms), easing = Tokens.Motion.easeStandard)

    val standard: TweenSpec<Float> get() = tween(Tokens.Motion.standard)
    val rapid: TweenSpec<Float> get() = tween(Tokens.Motion.rapid)
    val snappy: TweenSpec<Float> get() = tween(Tokens.Motion.snappy)
}

val LocalHotSpotMotion = compositionLocalOf { HotSpotMotion(reduced = false) }

/**
 * 尊重系统「动画时长缩放」：关闭动画或缩短动画（prefers-reduced-motion 的 Android 等价物）。
 * 读取 Settings.Global.ANIMATOR_DURATION_SCALE，0 表示用户关闭了窗口动画。
 */
@Composable
fun rememberHotSpotMotion(): HotSpotMotion {
    val context = LocalContext.current
    val reduced = remember(context) { isSystemAnimationDisabled(context) }
    return remember(reduced) { HotSpotMotion(reduced = reduced) }
}

private fun isSystemAnimationDisabled(context: Context): Boolean = try {
    Settings.Global.getFloat(
        context.contentResolver,
        Settings.Global.ANIMATOR_DURATION_SCALE,
        1f,
    ) == 0f
} catch (_: Exception) {
    false
}
