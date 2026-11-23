package com.hotspot.app.ui.components

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.widthIn
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import coil.request.ImageRequest
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.util.MediaUrls

/**
 * 聊天图片气泡内容：加载服务端下发的短时签名 URL（15 分钟过期，不做本地持久化）。
 * 签名过期/网络失败时回落「[图片]」文本占位，避免加载失败留下空白气泡。
 */
@Composable
fun ChatImageBubble(
    imageUrl: String?,
    isMine: Boolean,
    contentDescription: String,
    onClick: (() -> Unit)? = null,
) {
    val colors = HotSpotTheme.colors
    val absoluteUrl = MediaUrls.absolute(imageUrl)
    var loadFailed by remember(absoluteUrl) { mutableStateOf(false) }
    if (absoluteUrl.isNullOrBlank() || loadFailed) {
        Text(
            text = "[图片]",
            style = MaterialTheme.typography.bodyLarge,
            fontWeight = FontWeight.Normal,
            color = if (isMine) colors.textOnAccent else colors.textPrimary,
        )
    } else {
        AsyncImage(
            model = ImageRequest.Builder(LocalContext.current)
                .data(absoluteUrl)
                .crossfade(true)
                .build(),
            contentDescription = contentDescription,
            modifier = Modifier
                .widthIn(max = 220.dp)
                .heightIn(max = 240.dp)
                .clip(AppShapes.iconTile)
                .then(if (onClick != null) Modifier.clickable(onClick = onClick) else Modifier),
            contentScale = ContentScale.Crop,
            onError = { loadFailed = true },
        )
    }
}
