package com.hotspot.app.data.media

import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.PresignFileInput
import com.hotspot.app.data.model.UploadedMedia
import java.io.File

/**
 * 聊天语音上传（spec/api_spec_media.md → chat_audio）：
 * 20MB / audio/aac 白名单，presign → 流式 PUT → complete；录音文件只存在于 cacheDir，上传后删除。
 */
object ChatAudioUploader {
    private const val MAX_AUDIO_BYTES = 20L * 1024 * 1024
    private const val MIME = "audio/aac"

    suspend fun upload(file: File): Result<UploadedMedia> = runCatching {
        require(file.exists() && file.length() > 0) { "录音文件为空，请重试" }
        require(file.length() <= MAX_AUDIO_BYTES) { "语音不能超过 20MB" }
        val presigned = when (
            val result = AppContainer.mediaRepository.presign(
                listOf(
                    PresignFileInput(
                        fileName = file.name,
                        fileSize = file.length(),
                        mimeType = MIME,
                        category = "chat_audio",
                    ),
                ),
            )
        ) {
            is ApiResult.Ok -> result.data.uploads.firstOrNull() ?: error("预签名结果为空")
            is ApiResult.Err -> error(result.error.userMessage)
        }
        AppContainer.mediaRepository.putObjectStream(
            uploadUrl = presigned.uploadUrl,
            headers = presigned.headers,
            contentLength = file.length(),
            mimeType = MIME,
            openStream = { file.inputStream() },
        ).getOrThrow()
        when (val completed = AppContainer.mediaRepository.complete(listOf(presigned.uploadId))) {
            is ApiResult.Ok -> completed.data.uploads.firstOrNull() ?: error("上传结果为空")
            is ApiResult.Err -> error(completed.error.userMessage)
        }
    }
}
