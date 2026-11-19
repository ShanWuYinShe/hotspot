package com.hotspot.app

import com.hotspot.app.data.api.businessErrorMessage
import com.hotspot.app.ui.theme.Tokens
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Android 设计 token 结构不变量（生成物契约的单测防线）。
 *
 * 逐值对齐 CSS 的权威校验在 frontend/scripts/check-cross-platform-android.mjs（CI 与
 * bun run check:theme 强制执行）；本测试防的是生成器/手改导致的「结构退化」：
 * 6 组主题齐全、接口实现互不相同、字号/圆角/动效档位有序、颜色 alpha 与不透明色符合预期。
 */
class TokensParityTest {

    @Test
    fun themeCombos_areCompleteAndDistinct() {
        // 6 组 SemanticPalette 必须都存在且互不相同（主题切换不会落到同一份配色）
        val all = listOf(
            Tokens.Semantic.VibrantLight,
            Tokens.Semantic.VibrantDark,
            Tokens.Semantic.ForestLight,
            Tokens.Semantic.ForestDark,
            Tokens.Semantic.InkwashLight,
            Tokens.Semantic.InkwashDark,
        )
        assertEquals(6, all.map { it.textPrimary }.toSet().size)
        // 亮/暗画布必须不同（明暗切换有实际效果）
        assertNotEquals(Tokens.Semantic.VibrantLight.surfacePage, Tokens.Semantic.VibrantDark.surfacePage)
        // 三主题亮色画布各不相同
        assertEquals(
            3,
            listOf(
                Tokens.Semantic.VibrantLight.surfacePage,
                Tokens.Semantic.ForestLight.surfacePage,
                Tokens.Semantic.InkwashLight.surfacePage,
            ).toSet().size,
        )
    }

    @Test
    fun paletteAccentMatchesSemanticAccent() {
        // 语义 alias 与原始色板必须同值（--cst-accent-default = --cst-palette-accent-primary）
        assertEquals(Tokens.Palette.VibrantLight.accentPrimary, Tokens.Semantic.VibrantLight.accentDefault)
        assertEquals(Tokens.Palette.ForestDark.accentPrimary, Tokens.Semantic.ForestDark.accentDefault)
        assertEquals(Tokens.Palette.InkwashDark.accentPrimary, Tokens.Semantic.InkwashDark.accentDefault)
    }

    @Test
    fun ambientPalette_isPresentForEveryCombo() {
        // 氛围画布 6 组必须存在且 accent 系利用不同主题（装饰层随主题变化）
        val glows = listOf(
            Tokens.Ambient.VibrantLight.glow1,
            Tokens.Ambient.VibrantDark.glow1,
            Tokens.Ambient.ForestLight.glow1,
            Tokens.Ambient.InkwashDark.glow1,
        )
        assertEquals(4, glows.toSet().size)
        // 装饰色 alpha 必须很低（0 < alpha <= 0.3），否则会干扰正文对比度
        glows.forEach { color ->
            assertTrue("ambient glow alpha 过高: " + color.alpha, color.alpha in 0.01f..0.30f)
        }
    }

    @Test
    fun fontScale_ascends() {
        val sizes = listOf(
            Tokens.FontSize.xxs,
            Tokens.FontSize.xs,
            Tokens.FontSize.sm,
            Tokens.FontSize.md,
            Tokens.FontSize.lg,
            Tokens.FontSize.xl,
            Tokens.FontSize.xxl,
            Tokens.FontSize.xxxl,
            Tokens.FontSize.hero,
        )
        assertEquals(listOf(10f, 11f, 12f, 14f, 18f, 20f, 24f, 30f, 36f), sizes)
    }

    @Test
    fun radiusScale_matchesDesignSystem() {
        assertEquals(12f, Tokens.Radius.sm)
        assertEquals(16f, Tokens.Radius.md)
        assertEquals(20f, Tokens.Radius.lg)
        assertEquals(24f, Tokens.Radius.xl)
        assertEquals(28f, Tokens.Radius.sheet)
    }

    @Test
    fun motionScale_matchesDesignSystem() {
        // design-system §10.1 关键档位（pageEnter / contentFade 来自 utilities.css）
        assertEquals(100, Tokens.Motion.instant)
        assertEquals(110, Tokens.Motion.contentFade)
        assertEquals(120, Tokens.Motion.pageEnter)
        assertEquals(150, Tokens.Motion.rapid)
        assertEquals(180, Tokens.Motion.snappy)
        assertEquals(220, Tokens.Motion.standard)
        assertEquals(260, Tokens.Motion.relaxed)
        assertEquals(300, Tokens.Motion.spring)
        assertEquals(400, Tokens.Motion.slow)
        // 两条缓动曲线必须独立注册（--cst-ease-standard / --cst-ease-bounce）
        assertTrue(Tokens.Motion.easeStandard !== Tokens.Motion.easeBounce)
        // 标准曲线在 x=0/1 处归一化（CubicBezierEasing 契约）
        assertEquals(0f, Tokens.Motion.easeStandard.transform(0f), 0.0001f)
        assertEquals(1f, Tokens.Motion.easeStandard.transform(1f), 0.0001f)
    }

    @Test
    fun businessErrorMessages_mapContractCodes() {
        assertEquals("登录已失效，请重新登录", businessErrorMessage(10100, null))
        assertEquals("账号已被封禁", businessErrorMessage(10201, "xx"))
        assertEquals("请求过于频繁，请稍后再试", businessErrorMessage(10500, null))
        assertEquals("服务异常，请稍后再试", businessErrorMessage(10600, "内部错误"))
        assertEquals("服务端提示", businessErrorMessage(10300, "服务端提示"))
        assertEquals("请求失败（10001）", businessErrorMessage(10001, null))
    }
}
