package com.hotspot.app.ui.components

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.lerp
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.MotionSpec
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.ui.theme.Tokens

/** 按钮变体（Figma 组件库 421:8：primary / secondary / soft / ghost / danger）。 */
enum class ButtonVariant { PRIMARY, SECONDARY, SOFT, GHOST, DANGER }

/** 按钮尺寸档：SM 32 / MD 42 / LG 48。 */
enum class ButtonSize(val height: Dp, val horizontalPadding: Dp) {
    SM(Dimens.buttonSm, Spacing.md),
    MD(Dimens.buttonMd, Spacing.lg),
    LG(Dimens.buttonLg, Spacing.lg2),
}

/**
 * 系统按钮（动作按钮唯一壳）：按压 0.97 缩放 + 状态色；触控热区 ≥44dp；
 * 焦点可见；disabled 0.38 透明度（design-system §19.4 / §20.3）。
 */
@Composable
fun SystemButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    variant: ButtonVariant = ButtonVariant.PRIMARY,
    size: ButtonSize = ButtonSize.MD,
    enabled: Boolean = true,
    loading: Boolean = false,
    leadingIcon: (@Composable () -> Unit)? = null,
    trailingIcon: (@Composable () -> Unit)? = null,
    /**
     * 高度覆盖：仅用于设计稿登记的孤立档位（登录卡 CTA 44 高，Figma 87:1844）。
     * 常规场景一律用 [ButtonSize] 的 SM32 / MD42 / LG48 三档，禁止在页面里散写高度。
     */
    height: Dp? = null,
    /**
     * 全宽拉伸：作用于「视觉壳」本身。仅把 Modifier.fillMaxWidth() 传给外层热区时，
     * 内层视觉仍按内容宽居中（弹窗/行内按钮就此宽），需要设计登记的全宽 CTA 时置 true。
     */
    fillMaxWidth: Boolean = false,
) {
    val colors = HotSpotTheme.colors
    val motion = HotSpotTheme.motion
    val interaction = remember { MutableInteractionSource() }
    val pressed by interaction.collectIsPressedAsState()
    val interactive = enabled && !loading
    val scale by animateFloatAsState(
        targetValue = if (pressed && interactive) MotionSpec.PRESS_SCALE else 1f,
        animationSpec = motion.tween(Tokens.Motion.instant),
        label = "buttonScale",
    )

    val backgroundColor = when (variant) {
        ButtonVariant.PRIMARY -> colors.accent
        ButtonVariant.SECONDARY -> colors.surface
        ButtonVariant.SOFT -> colors.subtle
        ButtonVariant.GHOST -> Color.Transparent
        ButtonVariant.DANGER -> colors.danger
    }
    val foregroundColor = when (variant) {
        ButtonVariant.PRIMARY, ButtonVariant.DANGER -> colors.textOnAccent
        ButtonVariant.SECONDARY -> colors.textPrimary
        ButtonVariant.SOFT -> colors.textPrimary
        ButtonVariant.GHOST -> colors.accent
    }
    val pressedBackground = if (variant == ButtonVariant.GHOST) colors.interactionHover else lerp(backgroundColor, foregroundColor, 0.08f)
    val animatedBackground by animateColorAsState(
        targetValue = if (pressed && interactive) pressedBackground else backgroundColor,
        animationSpec = motion.tween(Tokens.Motion.rapid),
        label = "buttonBackground",
    )

    Box(
        modifier = modifier
            .then(if (fillMaxWidth) Modifier.fillMaxWidth() else Modifier)
            .heightIn(min = Dimens.minTouchTarget)
            .scale(scale)
            .alpha(if (enabled) 1f else com.hotspot.app.ui.theme.OpacityTokens.DISABLED)
            .clickable(
                interactionSource = interaction,
                indication = null,
                enabled = interactive,
                onClick = onClick,
            )
            .hotSpotFocusRing(cornerRadius = 16.dp),
        contentAlignment = Alignment.Center,
    ) {
        Row(
            modifier = Modifier
                .then(if (fillMaxWidth) Modifier.fillMaxWidth() else Modifier)
                .height(height ?: size.height)
                .clip(AppShapes.compact)
                .background(animatedBackground, AppShapes.compact)
                .then(
                    if (variant == ButtonVariant.SECONDARY) {
                        Modifier.border(1.dp, colors.border, AppShapes.compact)
                    } else {
                        Modifier
                    },
                )
                .padding(horizontal = size.horizontalPadding),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.Center,
        ) {
            if (loading) {
                CircularProgressIndicator(
                    modifier = Modifier.size(Dimens.iconXs),
                    strokeWidth = 2.dp,
                    color = foregroundColor,
                )
                Spacer(Modifier.width(Spacing.sm))
            } else if (leadingIcon != null) {
                leadingIcon()
                Spacer(Modifier.width(Spacing.sm2))
            }
            Text(
                text = text,
                style = if (size == ButtonSize.SM) {
                    MaterialTheme.typography.titleSmall
                } else {
                    MaterialTheme.typography.labelLarge
                },
                fontWeight = FontWeight.Medium,
                color = foregroundColor,
                maxLines = 1,
            )
            if (trailingIcon != null) {
                Spacer(Modifier.width(Spacing.sm2))
                trailingIcon()
            }
        }
    }
}
