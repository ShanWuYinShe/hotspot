package com.hotspot.app.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Tokens

/**
 * 评论作者微徽章（docs/ui-components.md 详情页契约 / Badge size="author"）：
 * 28×16dp、圆角 4dp、10sp Medium、accent-soft 底 + text-accent-on-soft 字，与昵称间距 6dp。
 * 4dp 圆角是微徽章物理形状的登记例外（圆角刻度最小档 12 不适用），禁止扩散到其它徽章。
 */
@Composable
fun CommentAuthorBadge(modifier: Modifier = Modifier) {
    val colors = HotSpotTheme.colors
    Box(
        modifier = modifier
            .width(28.dp)
            .height(16.dp)
            .background(colors.accentSoftBg, RoundedCornerShape(4.dp)),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = "作者",
            style = TextStyle(
                fontSize = Tokens.FontSize.xxs.sp,
                lineHeight = Tokens.FontSize.xxs.sp,
                fontWeight = FontWeight.Medium,
            ),
            color = colors.textAccentOnSoft,
            maxLines = 1,
        )
    }
}
