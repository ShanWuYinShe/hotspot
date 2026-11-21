package com.hotspot.app.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Icon
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
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import com.hotspot.app.R
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing

/**
 * 搜索输入（搜索语义唯一壳，输入族「搜索派」）：39 高 / 全圆角胶囊 / 左搜索图标 16 /
 * 字号 14 / 图标与文字间距 10 / surface-primary 底 + hairline 边（geometry 来源：Figma
 * 我的群组 87:3219 → 搜索栏 755:73：39 高、radius 9999、icon x=15、text x=41）。
 * 表单输入（43 / 圆角 16）继续走 SystemInput，两派不在页面内互换。
 */
@Composable
fun SearchInput(
    value: String,
    onValueChange: (String) -> Unit,
    placeholder: String,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    keyboardActions: KeyboardActions = KeyboardActions.Default,
) {
    val colors = HotSpotTheme.colors
    var focused by remember { mutableStateOf(false) }
    val shape = AppShapes.pill
    val borderColor = when {
        !enabled -> colors.borderSubtle
        focused -> colors.fieldBorderFocus
        else -> colors.fieldBorder
    }
    Row(
        modifier = modifier
            .fillMaxWidth()
            .height(Dimens.searchInputHeight)
            .clip(shape)
            .background(if (enabled) colors.surface else colors.fieldBgDisabled, shape)
            // 聚焦单层加粗到 2dp（不再叠内缘柔光环，避免双层描边）
            .border(if (focused) 2.dp else 1.dp, borderColor, shape)
            .padding(horizontal = Spacing.md3),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(R.drawable.ic_search),
            contentDescription = null,
            modifier = Modifier.size(Dimens.iconXs),
            tint = colors.textSecondary,
        )
        Spacer(Modifier.width(Spacing.md2))
        BasicTextField(
            value = value,
            onValueChange = onValueChange,
            modifier = Modifier
                .weight(1f)
                .onFocusChanged { focused = it.isFocused },
            enabled = enabled,
            singleLine = true,
            textStyle = MaterialTheme.typography.bodyLarge.copy(color = colors.textPrimary),
            cursorBrush = SolidColor(colors.accent),
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
            keyboardActions = keyboardActions,
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
    }
}
