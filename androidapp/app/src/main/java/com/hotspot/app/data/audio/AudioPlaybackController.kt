package com.hotspot.app.data.audio

import android.media.MediaPlayer
import android.os.Handler
import android.os.Looper
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/** 语音播放状态：url 为当前/最近一次播放的短时签名 URL；播放结束保留 totalMs 供气泡显示时长。 */
data class AudioPlaybackState(
    val url: String? = null,
    val playing: Boolean = false,
    val elapsedMs: Long = 0L,
    val totalMs: Long = 0L,
)

/**
 * 应用级语音播放控制器（全局唯一 MediaPlayer）：
 * - 同一时间只播放一条语音，切换气泡会先停止上一条；
 * - 每 200ms 在主线程轮询进度，供气泡展示进度条；
 * - 聊天页退出时由页面调用 [stop] 释放。
 */
class AudioPlaybackController {
    private val handler = Handler(Looper.getMainLooper())
    private val _state = MutableStateFlow(AudioPlaybackState())
    val state: StateFlow<AudioPlaybackState> = _state.asStateFlow()

    private var player: MediaPlayer? = null
    private var currentUrl: String? = null

    private val tick = object : Runnable {
        override fun run() {
            val mediaPlayer = player ?: return
            runCatching {
                _state.value = AudioPlaybackState(
                    url = currentUrl,
                    playing = true,
                    elapsedMs = mediaPlayer.currentPosition.toLong(),
                    totalMs = mediaPlayer.duration.toLong().coerceAtLeast(0L),
                )
                handler.postDelayed(this, PROGRESS_INTERVAL_MS)
            }
        }
    }

    /** 点击同一条正在播放的语音 = 停止；点击其它语音 = 切换播放。 */
    fun toggle(url: String, durationHintMs: Long = 0L) {
        if (currentUrl == url && player != null) {
            stop()
            return
        }
        releasePlayer()
        val mediaPlayer = MediaPlayer()
        player = mediaPlayer
        currentUrl = url
        _state.value = AudioPlaybackState(url = url, playing = false, totalMs = durationHintMs.coerceAtLeast(0L))
        mediaPlayer.setOnPreparedListener { prepared ->
            if (player !== prepared) {
                prepared.release()
                return@setOnPreparedListener
            }
            prepared.start()
            _state.value = AudioPlaybackState(
                url = url,
                playing = true,
                elapsedMs = 0L,
                totalMs = prepared.duration.toLong().coerceAtLeast(0L),
            )
            handler.post(tick)
        }
        mediaPlayer.setOnCompletionListener {
            if (player === mediaPlayer) {
                val total = _state.value.totalMs
                releasePlayer(keepUrl = true)
                _state.value = AudioPlaybackState(url = url, playing = false, elapsedMs = 0L, totalMs = total)
            }
        }
        mediaPlayer.setOnErrorListener { _, _, _ ->
            val total = _state.value.totalMs
            releasePlayer(keepUrl = true)
            _state.value = AudioPlaybackState(url = url, playing = false, elapsedMs = 0L, totalMs = total)
            true
        }
        runCatching {
            mediaPlayer.setDataSource(url)
            mediaPlayer.prepareAsync()
        }.onFailure {
            releasePlayer()
            _state.value = AudioPlaybackState()
        }
    }

    /** 停止并释放当前播放器；状态回到空闲。 */
    fun stop() {
        releasePlayer()
        _state.value = AudioPlaybackState()
    }

    private fun releasePlayer(keepUrl: Boolean = false) {
        handler.removeCallbacks(tick)
        runCatching { player?.stop() }
        runCatching { player?.release() }
        player = null
        if (!keepUrl) currentUrl = null
    }

    private companion object {
        const val PROGRESS_INTERVAL_MS = 200L
    }
}
