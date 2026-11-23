package com.hotspot.app.ui.components

import android.media.MediaPlayer
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.dp
import com.hotspot.app.R
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.MediaUrls

/**
 * 语音消息气泡：点击播放/停止（MediaPlayer，短时签名 URL）。
 * 不做进度条，按钮 + 「语音/播放中」+ 可选时长（服务端 media.duration 可能为 null）。
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
    var playing by remember(absoluteUrl) { mutableStateOf(false) }
    var player by remember(absoluteUrl) { mutableStateOf<MediaPlayer?>(null) }
    fun stopAndRelease() {
        runCatching { player?.release() }
        player = null
    }
    fun stop() {
        runCatching { player?.stop() }
        stopAndRelease()
        playing = false
    }
    DisposableEffect(Unit) {
        onDispose { stopAndRelease() }
    }
    Row(
        modifier = modifier
            .clip(CircleShape)
            .clickable {
                if (playing) {
                    stop()
                } else if (!absoluteUrl.isNullOrBlank()) {
                    stopAndRelease()
                    val mediaPlayer = MediaPlayer()
                    player = mediaPlayer
                    mediaPlayer.setOnPreparedListener {
                        it.start()
                        playing = true
                    }
                    mediaPlayer.setOnCompletionListener {
                        playing = false
                        stopAndRelease()
                    }
                    mediaPlayer.setOnErrorListener { _, _, _ ->
                        playing = false
                        stopAndRelease()
                        true
                    }
                    runCatching {
                        mediaPlayer.setDataSource(absoluteUrl)
                        mediaPlayer.prepareAsync()
                    }.onFailure {
                        playing = false
                        stopAndRelease()
                    }
                }
            }
            .padding(vertical = Spacing.xs, horizontal = Spacing.sm),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(if (playing) R.drawable.ic_pause else R.drawable.ic_action_play),
            contentDescription = if (playing) "停止播放" else "播放语音",
            modifier = Modifier.size(Dimens.iconSm),
            tint = if (isMine) colors.textOnAccent else colors.accent,
        )
        Spacer(Modifier.width(Spacing.xs))
        Text(
            text = if (playing) {
                "播放中…"
            } else if (durationSeconds != null && durationSeconds > 0) {
                durationSeconds.toString() + " 秒"
            } else {
                "语音"
            },
            style = MaterialTheme.typography.bodyMedium,
            color = if (isMine) colors.textOnAccent else colors.textPrimary,
        )
        Spacer(Modifier.width(Spacing.xs))
    }
}
