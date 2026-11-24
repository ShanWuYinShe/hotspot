package com.hotspot.app.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
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
 * 聊天图片内容（私聊/群聊共用）：
 * - 不套气泡外壳（对齐 Web / 鸿蒙：圆角图直接贴聊天背景，时间在图下，外层由 MessageBubble 负责）；
 * - 按图片真实宽高比展示（[aspectRatio]），不裁切也不留白；加载中露出 surfaceSubtle 占位；
 * - Coil crossfade 淡入；失败回落圆角占位 +「图片加载失败」文案（不再留空白气泡）；
 * - 点击打开全屏查看器，长按打开消息操作面板（点击/长按由调用方注入）。
 */
@Composable
fun ChatImageBubble(
    imageUrl: String?,
    contentDescription: String,
    onClick: (() -> Unit)? = null,
    onLongClick: (() -> Unit)? = null,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    val absoluteUrl = MediaUrls.absolute(imageUrl)
    var loadFailed by remember(absoluteUrl) { mutableStateOf(false) }
    var aspectRatio by remember(absoluteUrl) { mutableStateOf(0f) }
    // 16dp：对齐 Web rounded-md / 鸿蒙 RADIUS.MD
    val shape = AppShapes.compact
    val interactModifier = when {
        onClick != null && onLongClick != null -> Modifier.combinedClickable(onClick = onClick, onLongClick = onLongClick)
        onClick != null -> Modifier.clickable(onClick = onClick)
        onLongClick != null -> Modifier.combinedClickable(onClick = {}, onLongClick = onLongClick)
        else -> Modifier
    }

    if (absoluteUrl.isNullOrBlank() || loadFailed) {
        Box(
            modifier = modifier
                .width(180.dp)
                .height(135.dp)
                .clip(shape)
                .background(colors.subtle)
                .then(interactModifier),
            contentAlignment = Alignment.Center,
        ) {
            Text(
                text = "图片加载失败",
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = FontWeight.Normal,
                color = colors.textSecondary,
            )
        }
        return
    }

    val ratioModifier = if (aspectRatio > 0f) {
        Modifier.aspectRatio(aspectRatio)
    } else {
        Modifier.aspectRatio(4f / 3f)
    }
    AsyncImage(
        model = ImageRequest.Builder(LocalContext.current)
            .data(absoluteUrl)
            .crossfade(true)
            .build(),
        contentDescription = contentDescription,
        modifier = modifier
            .widthIn(max = 220.dp)
            .heightIn(max = 300.dp)
            .then(ratioModifier)
            .clip(shape)
            .background(colors.subtle)
            .then(interactModifier),
        contentScale = ContentScale.Fit,
        onSuccess = { state ->
            val drawable = state.result.drawable
            if (drawable.intrinsicWidth > 0 && drawable.intrinsicHeight > 0) {
                aspectRatio = drawable.intrinsicWidth.toFloat() / drawable.intrinsicHeight.toFloat()
            }
        },
        onError = { loadFailed = true },
    )
}
