package com.hotspot.app.ui.theme

import androidx.compose.ui.unit.dp

/**
 * 间距刻度：对齐 design-system §5（4px 网格 + 微步，不强制收敛）。
 * 页面只消费这些档位，禁止在组件里写裸 dp 间距字面量。
 */
object Spacing {
    val half = 2.dp
    val three = 3.dp
    val xs = 4.dp
    val sm2 = 6.dp
    val sm = 8.dp
    val md2 = 10.dp
    val md = 12.dp
    val md3 = 14.dp
    val lg = 16.dp
    val lgPlus = 18.dp
    val lg2 = 20.dp
    val xl = 24.dp
    val xxl = 32.dp
}

/**
 * 透明度基线（design-system §11 鸿蒙 OPACITY 的平台等价档；前端无 0.38 档，
 * 该值只用于 disabled 的可辨性，不用于装饰）。
 */
object OpacityTokens {
    const val DISABLED = 0.38f
}

/**
 * 组件与布局尺寸：对齐 docs/design-system.md §12.2 与 Figma 设计规范页（421:8）组件库规格。
 * 这些是布局几何（非主题风格），不随主题变化。
 */
object Dimens {
    // 页面
    val pageGutter = Spacing.lg // 移动端 16
    val pageTopGap = Spacing.lg
    val sectionGap = Spacing.xl // 24
    val sectionInnerGap = Spacing.md // 12

    // 触控
    val minTouchTarget = 44.dp // mobile-ui-conventions：有效热区 ≥44dp

    // 按钮档位（Figma 组件库：SM32 / MD42 / LG48）
    val buttonSm = 32.dp
    val buttonMd = 42.dp
    val buttonLg = 48.dp

    // 输入
    val inputHeight = 43.dp // 表单输入（16 圆角）
    val searchInputHeight = 39.dp // 搜索胶囊（39 高）

    // 徽章 / Tab / 列表 / 导航
    val badgeSm = 20.dp
    val badgeMd = 24.dp
    val tabBar = 34.dp
    val listRow = 64.dp
    val bottomNavHeight = 56.dp
    val backButton = 42.dp
    val iconButton = 36.dp

    // 图标（独立刻度，与字号解耦）
    val icon2xs = 14.dp
    val iconXs = 16.dp
    val iconSm = 20.dp
    val iconMd = 24.dp
    val iconLg = 32.dp
    val iconEmpty = 48.dp

    // 头像
    val avatarXxs = 22.dp
    val avatarXs = 32.dp
    val avatarSm = 40.dp
    val avatarMd = 48.dp
    val avatarLg = 80.dp
}
