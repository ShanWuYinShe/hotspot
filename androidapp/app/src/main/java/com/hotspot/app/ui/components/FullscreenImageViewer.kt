package com.hotspot.app.ui.components

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.gestures.rememberTransformableState
import androidx.compose.foundation.gestures.transformable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import coil.compose.AsyncImage
import coil.request.ImageRequest
import com.hotspot.app.R
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.MediaUrls

/**
 * 单图全屏查看器：黑底 + 双指缩放/拖拽 + 双击放大/复位 + 关闭。
 * 仅用于已加载的聊天图片短时签名 URL；不落盘、不提供保存。
 */
@Composable
fun FullscreenImageViewer(
    imageUrl: String?,
    contentDescription: String = "查看图片",
    onDismiss: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    val absoluteUrl = MediaUrls.absolute(imageUrl)
    var scale by remember(absoluteUrl) { mutableFloatStateOf(1f) }
    var offset by remember(absoluteUrl) { mutableStateOf(Offset.Zero) }
    val transformState = rememberTransformableState { zoomChange, panChange, _ ->
        val nextScale = (scale * zoomChange).coerceIn(1f, 5f)
        scale = nextScale
        offset = if (nextScale <= 1.001f) Offset.Zero else offset + panChange
    }
    BackHandler(onBack = onDismiss)
    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false, decorFitsSystemWindows = false),
    ) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(Color.Black),
        ) {
            if (!absoluteUrl.isNullOrBlank()) {
                AsyncImage(
                    model = ImageRequest.Builder(LocalContext.current)
                        .data(absoluteUrl)
                        .crossfade(true)
                        .build(),
                    contentDescription = contentDescription,
                    modifier = Modifier
                        .fillMaxSize()
                        .graphicsLayer(
                            scaleX = scale,
                            scaleY = scale,
                            translationX = offset.x,
                            translationY = offset.y,
                        )
                        .pointerInput(absoluteUrl) {
                            detectTapGestures(
                                onDoubleTap = {
                                    if (scale > 1.001f) {
                                        scale = 1f
                                        offset = Offset.Zero
                                    } else {
                                        scale = 2f
                                    }
                                },
                            )
                        }
                        .transformable(transformState),
                    contentScale = ContentScale.Fit,
                )
            }
            IconButton(
                onClick = onDismiss,
                modifier = Modifier
                    .align(Alignment.TopEnd)
                    .statusBarsPadding()
                    .padding(Spacing.sm)
                    .size(44.dp),
            ) {
                Icon(
                    painter = painterResource(R.drawable.ic_close),
                    contentDescription = "关闭",
                    tint = colors.textOnAccent,
                )
            }
        }
    }
}

/**
 * 多图全屏查看器：左右滑动切换同组图片，底部显示「当前 / 总数」；双击放大/复位。
 *
 * 多图下优先保证横向翻页：放大时先禁用翻页，双击复位后再滑页；双指缩放/拖拽仍由单图查看器承担。
 */
@Composable
fun FullscreenImageViewer(
    images: List<String>,
    initialIndex: Int = 0,
    contentDescription: String = "查看图片",
    onDismiss: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    val urls = images.mapNotNull { MediaUrls.absolute(it) }.distinct()
    if (urls.isEmpty()) return
    val startIndex = initialIndex.coerceIn(0, urls.lastIndex)
    val pagerState = rememberPagerState(initialPage = startIndex, pageCount = { urls.size })
    var scale by remember { mutableFloatStateOf(1f) }
    LaunchedEffect(pagerState.currentPage) {
        scale = 1f
    }
    BackHandler(onBack = onDismiss)
    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false, decorFitsSystemWindows = false),
    ) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(Color.Black),
        ) {
            HorizontalPager(
                state = pagerState,
                modifier = Modifier.fillMaxSize(),
                userScrollEnabled = scale <= 1.001f,
            ) { page ->
                AsyncImage(
                    model = ImageRequest.Builder(LocalContext.current)
                        .data(urls[page])
                        .crossfade(true)
                        .build(),
                    contentDescription = contentDescription,
                    modifier = Modifier
                        .fillMaxSize()
                        .graphicsLayer(scaleX = scale, scaleY = scale)
                        .pointerInput(urls[page]) {
                            detectTapGestures(
                                onDoubleTap = {
                                    scale = if (scale > 1.001f) 1f else 2f
                                    if (scale == 1f) {
                                        // 复位后允许继续翻页
                                    }
                                },
                            )
                        },
                    contentScale = ContentScale.Fit,
                )
            }
            if (urls.size > 1) {
                Box(
                    modifier = Modifier
                        .align(Alignment.BottomCenter)
                        .navigationBarsPadding()
                        .padding(bottom = Spacing.lg)
                        .background(Color.Black.copy(alpha = 0.5f), AppShapes.pill)
                        .padding(horizontal = Spacing.sm2, vertical = Spacing.xs),
                ) {
                    Text(
                        text = (pagerState.currentPage + 1).toString() + " / " + urls.size,
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textOnAccent,
                    )
                }
            }
            IconButton(
                onClick = onDismiss,
                modifier = Modifier
                    .align(Alignment.TopEnd)
                    .statusBarsPadding()
                    .padding(Spacing.sm)
                    .size(44.dp),
            ) {
                Icon(
                    painter = painterResource(R.drawable.ic_close),
                    contentDescription = "关闭",
                    tint = colors.textOnAccent,
                )
            }
        }
    }
}
