package com.hotspot.app.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.hotspot.app.AppContainer
import com.hotspot.app.R
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.MediaUrls

/**
 * 语音消息气泡：点击交给应用级 [com.hotspot.app.data.audio.AudioPlaybackController] 播放/停止。
 * 全局同一时间只播放一条语音；播放中展示进度条与「已播放 / 总时长」，播完/停止后显示实际总时长。
 * 服务端 media.duration 可能为 null，此时以播放器实际时长为准。
 */
@Composable
fun AudioMessageBubble(
    audioUrl: String?,
    isMine: Boolean,
    durationSeconds: Int?,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    val absoluteUrl = MediaUrls.absolute(audioUrl)
    val playback by AppContainer.audioPlaybackController.state.collectAsStateWithLifecycle()
    val isCurrent = absoluteUrl != null && playback.url == absoluteUrl
    val playing = isCurrent && playback.playing
    val totalMs = when {
        isCurrent && playback.totalMs > 0L -> playback.totalMs
        durationSeconds != null && durationSeconds > 0 -> durationSeconds.toLong() * 1000L
        else -> 0L
    }
    val elapsedMs = if (playing) playback.elapsedMs else 0L
    val progress = if (totalMs > 0L) (elapsedMs.toFloat() / totalMs.toFloat()).coerceIn(0f, 1f) else 0f
    val foreground = if (isMine) colors.textOnAccent else colors.accent
    val track = if (isMine) colors.textOnAccent.copy(alpha = 0.28f) else colors.accent.copy(alpha = 0.22f)
    Row(
        modifier = modifier
            .clip(CircleShape)
            .clickable(enabled = !absoluteUrl.isNullOrBlank()) {
                val url = absoluteUrl ?: return@clickable
                AppContainer.audioPlaybackController.toggle(
                    url = url,
                    durationHintMs = (durationSeconds ?: 0).toLong() * 1000L,
                )
            }
            .padding(vertical = Spacing.xs, horizontal = Spacing.sm),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(if (playing) R.drawable.ic_pause else R.drawable.ic_action_play),
            contentDescription = if (playing) "停止播放" else "播放语音",
            modifier = Modifier.size(Dimens.iconSm),
            tint = foreground,
        )
        Spacer(Modifier.width(Spacing.xs))
        Column {
            Text(
                text = when {
                    playing -> formatAudioTime(elapsedMs) + " / " + formatAudioTime(totalMs)
                    totalMs > 0L -> formatAudioTime(totalMs)
                    else -> "语音"
                },
                style = MaterialTheme.typography.bodyMedium,
                color = if (isMine) colors.textOnAccent else colors.textPrimary,
            )
            if (playing) {
                Spacer(Modifier.height(3.dp))
                Box(
                    modifier = Modifier
                        .width(72.dp)
                        .height(3.dp)
                        .clip(CircleShape)
                        .background(track),
                ) {
                    Box(
                        modifier = Modifier
                            .fillMaxWidth(progress)
                            .height(3.dp)
                            .background(foreground, CircleShape),
                    )
                }
            }
        }
        Spacer(Modifier.width(Spacing.xs))
    }
}

/** 毫秒 → m:ss（仅用于语音时长/进度展示）。 */
private fun formatAudioTime(millis: Long): String {
    val totalSeconds = (millis / 1000L).coerceAtLeast(0L)
    val minutes = totalSeconds / 60L
    val seconds = totalSeconds % 60L
    return minutes.toString() + ":" + seconds.toString().padStart(2, '0')
}
