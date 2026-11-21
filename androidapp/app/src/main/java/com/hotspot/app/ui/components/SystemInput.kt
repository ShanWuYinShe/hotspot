package com.hotspot.app.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing

/**
 * 系统输入（表单输入唯一壳）：43 高 / 圆角 16 / 聚焦 accent 2dp 单层边界 / 错误态 danger 单层边界。
 * 表面语言对齐前端 .input-surface（静态 1px 边 → 聚焦 2px 单层边界，无外层柔光环；
 * 双层深浅描边在浅色主题下会被读成「双边框」）。
 */
@Composable
fun SystemInput(
    value: String,
    onValueChange: (String) -> Unit,
    modifier: Modifier = Modifier,
    placeholder: String = "",
    label: String? = null,
    errorText: String? = null,
    enabled: Boolean = true,
    keyboardOptions: KeyboardOptions = KeyboardOptions.Default,
    keyboardActions: KeyboardActions = KeyboardActions.Default,
    visualTransformation: VisualTransformation = VisualTransformation.None,
    singleLine: Boolean = true,
    leading: (@Composable () -> Unit)? = null,
    trailing: (@Composable () -> Unit)? = null,
) {
    val colors = HotSpotTheme.colors
    var focused by remember { mutableStateOf(false) }
    val shape = AppShapes.compact
    val borderColor = when {
        !enabled -> colors.borderSubtle
        errorText != null -> colors.danger
        focused -> colors.fieldBorderFocus
        else -> colors.fieldBorder
    }
    // 聚焦/错误态单层加粗到 2dp：替代「hairline 边 + 内缘柔光环」的双层描边
    val borderWidth = if (focused || errorText != null) 2.dp else 1.dp
    Column(modifier) {
        if (label != null) {
            Text(
                text = label,
                style = MaterialTheme.typography.labelMedium,
                color = colors.textSecondary,
                modifier = Modifier.padding(bottom = Spacing.sm2),
            )
        }
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .height(Dimens.inputHeight)
                .clip(shape)
                .background(if (enabled) colors.fieldBg else colors.fieldBgDisabled, shape)
                .border(borderWidth, borderColor, shape)
                .padding(horizontal = Spacing.md),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (leading != null) {
                leading()
                Spacer(Modifier.width(Spacing.sm))
            }
            BasicTextField(
                value = value,
                onValueChange = onValueChange,
                modifier = Modifier
                    .weight(1f)
                    .onFocusChanged { focused = it.isFocused },
                enabled = enabled,
                singleLine = singleLine,
                textStyle = MaterialTheme.typography.bodyLarge.copy(color = colors.textPrimary),
                cursorBrush = SolidColor(colors.accent),
                keyboardOptions = keyboardOptions,
                keyboardActions = keyboardActions,
                visualTransformation = visualTransformation,
                decorationBox = { innerTextField ->
                    Box(contentAlignment = Alignment.CenterStart) {
                        if (value.isEmpty()) {
                            Text(
                                text = placeholder,
                                style = MaterialTheme.typography.bodyLarge,
                                color = colors.textSecondary,
                                maxLines = 1,
                            )
                        }
                        innerTextField()
                    }
                },
            )
            if (trailing != null) {
                Spacer(Modifier.width(Spacing.sm))
                trailing()
            }
        }
        if (errorText != null) {
            Text(
                text = errorText,
                style = MaterialTheme.typography.bodySmall,
                color = colors.danger,
                modifier = Modifier.padding(top = Spacing.xs),
            )
        }
    }
}
