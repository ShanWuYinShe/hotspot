package com.hotspot.app.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.unit.dp
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing

/** 多行输入（公告正文等长文本）：与 SystemInput 同一表面语言，默认至少 3 行高。 */
@Composable
fun SystemTextArea(
    value: String,
    onValueChange: (String) -> Unit,
    modifier: Modifier = Modifier,
    placeholder: String = "",
    label: String? = null,
    errorText: String? = null,
    enabled: Boolean = true,
    minHeight: androidx.compose.ui.unit.Dp = Dimens.inputHeight * 3,
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
        BasicTextField(
            value = value,
            onValueChange = onValueChange,
            modifier = Modifier
                .fillMaxWidth()
                .heightIn(min = minHeight)
                .clip(shape)
                .background(if (enabled) colors.fieldBg else colors.fieldBgDisabled, shape)
                .border(borderWidth, borderColor, shape)
                .padding(horizontal = Spacing.md, vertical = Spacing.sm2)
                .onFocusChanged { focused = it.isFocused },
            enabled = enabled,
            textStyle = MaterialTheme.typography.bodyLarge.copy(color = colors.textPrimary),
            cursorBrush = SolidColor(colors.accent),
            decorationBox = { innerTextField ->
                Box {
                    if (value.isEmpty()) {
                        Text(
                            text = placeholder,
                            style = MaterialTheme.typography.bodyLarge,
                            color = colors.textSecondary,
                        )
                    }
                    innerTextField()
                }
            },
        )
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
