package com.hotspot.app.ui.theme

import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.TextUnit
import androidx.compose.ui.unit.sp

/**
 * 字号档来自生成 token（10/11/12/14/18/20/24/30/36，对齐 design-system §4.1）。
 * 行高按 design-system §4.2 的各字号默认行高映射（10→15 / 11→15 / 12→17 / 14→21 /
 * 18→28 / 20→28 / 24→32 / 30→36 / 36→40）——Compose 不设行高会沿用字体默认行盒，
 * 与 Web 的 Tailwind 行高不一致（系统性偏矮）。
 */
private fun style(
    size: Float,
    lineHeight: Float,
    weight: FontWeight = FontWeight.Normal,
    letterSpacing: TextUnit = 0.sp,
): TextStyle = TextStyle(
    fontFamily = FontFamily.Default,
    fontWeight = weight,
    fontSize = size.sp,
    lineHeight = lineHeight.sp,
    letterSpacing = letterSpacing,
)

private val fs = Tokens.FontSize

val HotSpotTypography = Typography(
    displayLarge = style(fs.hero, 40f, FontWeight.SemiBold, letterSpacing = (-0.12f).sp),
    displayMedium = style(fs.xxxl, 36f, FontWeight.SemiBold, letterSpacing = (-0.12f).sp),
    displaySmall = style(fs.xxl, 32f, FontWeight.SemiBold),
    headlineLarge = style(fs.xxl, 32f, FontWeight.SemiBold),
    headlineMedium = style(fs.xl, 28f, FontWeight.SemiBold),
    headlineSmall = style(fs.lg, 28f, FontWeight.SemiBold),
    titleLarge = style(fs.xl, 28f, FontWeight.SemiBold),
    titleMedium = style(fs.md, 21f, FontWeight.Medium),
    titleSmall = style(fs.sm, 17f, FontWeight.Medium),
    bodyLarge = style(fs.md, 21f),
    bodyMedium = style(fs.sm, 17f),
    bodySmall = style(fs.xs, 15f),
    labelLarge = style(fs.md, 21f, FontWeight.Medium),
    labelMedium = style(fs.sm, 17f, FontWeight.Medium),
    labelSmall = style(fs.xs, 15f, FontWeight.Medium),
)

/** 页面内复用的补充文字样式（Figma 设计规范页 421:5 的 eyebrow / 计数档）。 */
object HotSpotTextStyles {
    /** 极小 uppercase 分组标（--text-xxs 10 / 字距 eyebrow）。 */
    val eyebrow = style(fs.xxs, 15f, FontWeight.Medium, letterSpacing = 1.4.sp)
    /** 聊天气泡正文（对齐 Web --leading-chat：14→20）。 */
    val chat = style(fs.md, 20f)
    /** 计数 / 时间戳 / 辅助文字（--text-xs 11）。 */
    val caption = style(fs.xs, 15f)
    /** 版权/极微说明（--text-xxs 10）。 */
    val micro = style(fs.xxs, 15f)
    /** 数字强调（点赞/评论计数，tabular 近似用等宽字体）。 */
    val numeric = style(fs.sm, 17f, FontWeight.Medium, letterSpacing = 0.sp)
}
