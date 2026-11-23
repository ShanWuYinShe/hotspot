package com.hotspot.app.data.media

import android.content.Context
import android.media.MediaRecorder
import android.os.Build
import java.io.File

/**
 * 语音录制：MediaRecorder AAC-ADTS（audio/aac，兼容 chat_audio 白名单）。
 * 只负责本地文件；上传与发送由页面协程编排。
 */
class VoiceRecorder(private val context: Context) {
    private var recorder: MediaRecorder? = null
    private var outputFile: File? = null
    private var startedAt = 0L

    fun start(): Result<Unit> = runCatching {
        cancel()
        val file = File(context.cacheDir, "voice_" + System.currentTimeMillis() + ".aac")
        val mediaRecorder = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            MediaRecorder(context)
        } else {
            @Suppress("DEPRECATION")
            MediaRecorder()
        }
        mediaRecorder.apply {
            setAudioSource(MediaRecorder.AudioSource.MIC)
            setOutputFormat(MediaRecorder.OutputFormat.AAC_ADTS)
            setAudioEncoder(MediaRecorder.AudioEncoder.AAC)
            setAudioSamplingRate(44100)
            setAudioEncodingBitRate(64_000)
            setOutputFile(file.absolutePath)
            prepare()
            start()
        }
        recorder = mediaRecorder
        outputFile = file
        startedAt = System.currentTimeMillis()
    }

    fun stop(): Result<RecordedAudio> = runCatching {
        val mediaRecorder = recorder ?: error("当前没有录音")
        val file = outputFile ?: error("录音文件不存在")
        val durationMs = System.currentTimeMillis() - startedAt
        try {
            mediaRecorder.stop()
        } finally {
            mediaRecorder.release()
            recorder = null
            outputFile = null
        }
        RecordedAudio(file = file, durationSeconds = (durationMs / 1000).toInt())
    }

    fun cancel() {
        runCatching { recorder?.stop() }
        runCatching { recorder?.release() }
        recorder = null
        outputFile?.delete()
        outputFile = null
    }

    data class RecordedAudio(val file: File, val durationSeconds: Int)
}
