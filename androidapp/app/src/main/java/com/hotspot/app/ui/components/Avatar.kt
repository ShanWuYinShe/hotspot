package com.hotspot.app.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import coil.request.ImageRequest
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.util.MediaUrls

/**
 * 头像：首字母占位层始终渲染（有 name 取首字母；无 name 用占位符），图片 absolute 覆盖渐显；
 * 失败保留占位不闪烁（对齐前端 shared/Avatar 契约）。
 */
@Composable
fun Avatar(
    name: String,
    imageUrl: String?,
    modifier: Modifier = Modifier,
    size: Dp = Dimens.avatarSm,
    shape: Shape = CircleShape,
    contentDescription: String? = null,
) {
    val colors = HotSpotTheme.colors
    val initial = name.trim().take(1).ifEmpty { "?" }
    val absoluteUrl = MediaUrls.absolute(imageUrl)
    Box(
        modifier = modifier
            .size(size)
            .clip(shape)
            .background(colors.accentSoftBg),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = initial,
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.SemiBold,
            color = colors.textAccentOnSoft,
            maxLines = 1,
        )
        if (absoluteUrl != null) {
            AsyncImage(
                model = ImageRequest.Builder(LocalContext.current)
                    .data(absoluteUrl)
                    .crossfade(true)
                    .build(),
                contentDescription = contentDescription ?: name,
                modifier = Modifier
                    .size(size)
                    .clip(shape),
                contentScale = ContentScale.Crop,
            )
        }
    }
}
