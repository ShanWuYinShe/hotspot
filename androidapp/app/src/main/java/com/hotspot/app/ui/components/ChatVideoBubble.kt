package com.hotspot.app.ui.components

import android.content.Intent
import android.net.Uri
import android.widget.Toast
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import coil.request.ImageRequest
import com.hotspot.app.R
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.MediaUrls

/**
 * 聊天视频气泡：优先展示服务端视频缩略图 + 播放角标；无缩略图时用深色占位块。
 * 点击通过系统 ACTION_VIEW 打开签名 URL，不做应用内播放器（首版范围）。
 */
@Composable
fun ChatVideoBubble(
    fileName: String?,
    durationSeconds: Int?,
    thumbnailUrl: String?,
    videoUrl: String?,
) {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    val absoluteVideoUrl = MediaUrls.absolute(videoUrl)
    val absoluteThumb = MediaUrls.absolute(thumbnailUrl)
    val shape = remember { RoundedCornerShape(12.dp) }
    val open = {
        if (!absoluteVideoUrl.isNullOrBlank()) {
            runCatching {
                context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(absoluteVideoUrl)))
            }.onFailure {
                Toast.makeText(context, "没有可播放该视频的应用", Toast.LENGTH_SHORT).show()
            }
        }
    }
    Box(
        modifier = Modifier
            .widthIn(max = 220.dp)
            .width(220.dp)
            .aspectRatio(16f / 9f)
            .clip(shape)
            .background(colors.scrim)
            .clickable(onClick = open),
        contentAlignment = Alignment.Center,
    ) {
        if (!absoluteThumb.isNullOrBlank()) {
            AsyncImage(
                model = ImageRequest.Builder(context).data(absoluteThumb).crossfade(true).build(),
                contentDescription = "视频缩略图",
                modifier = Modifier.fillMaxSize(),
                contentScale = ContentScale.Crop,
            )
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .background(colors.scrim.copy(alpha = 0.25f)),
            )
        }
        Icon(
            painter = painterResource(R.drawable.ic_action_play),
            contentDescription = "播放视频",
            modifier = Modifier.size(40.dp),
            tint = Color.White,
        )
        if (!fileName.isNullOrBlank()) {
            Text(
                text = fileName,
                style = MaterialTheme.typography.labelSmall,
                fontWeight = FontWeight.Medium,
                color = Color.White,
                maxLines = 1,
                modifier = Modifier
                    .align(Alignment.BottomStart)
                    .padding(Spacing.sm),
            )
        }
        if (durationSeconds != null && durationSeconds > 0) {
            Text(
                text = formatDuration(durationSeconds),
                style = MaterialTheme.typography.labelSmall,
                color = Color.White,
                modifier = Modifier
                    .align(Alignment.BottomEnd)
                    .padding(Spacing.sm),
            )
        }
    }
}

private fun formatDuration(seconds: Int): String {
    val minutes = seconds / 60
    val remain = seconds % 60
    return if (minutes > 0) "%d:%02d".format(minutes, remain) else "0:%02d".format(remain)
}
