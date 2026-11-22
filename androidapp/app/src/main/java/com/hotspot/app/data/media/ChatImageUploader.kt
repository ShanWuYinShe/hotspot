package com.hotspot.app.data.media

import android.content.Context
import android.net.Uri
import android.provider.OpenableColumns
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.PresignFileInput
import com.hotspot.app.data.model.UploadedMedia
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * 聊天图片上传（spec/api_spec_media.md）：presign_upload(category=chat_image) → PUT 直传 → complete_upload。
 * 复用 MediaRepository 的三步链路；10MB / 图片 MIME 白名单在客户端先拦一次，服务端仍会二次校验。
 */
object ChatImageUploader {
    private const val MAX_CHAT_IMAGE_BYTES = 10L * 1024 * 1024
    private val ALLOWED_MIME = setOf("image/jpeg", "image/png", "image/webp", "image/gif")

    suspend fun upload(context: Context, uri: Uri): Result<UploadedMedia> = runCatching {
        val payload = withContext(Dispatchers.IO) { readLocalImage(context, uri) }
        require(payload.mime in ALLOWED_MIME) { "仅支持 JPG/PNG/WebP/GIF 图片" }
        require(payload.bytes.size.toLong() <= MAX_CHAT_IMAGE_BYTES) { "图片不能超过 10MB" }

        val presigned = when (
            val result = AppContainer.mediaRepository.presign(
                listOf(
                    PresignFileInput(
                        fileName = payload.name,
                        fileSize = payload.bytes.size.toLong(),
                        mimeType = payload.mime,
                        category = "chat_image",
                    ),
                ),
            )
        ) {
            is ApiResult.Ok -> result.data.uploads.firstOrNull() ?: error("预签名结果为空")
            is ApiResult.Err -> error(result.error.userMessage)
        }
        AppContainer.mediaRepository.putObject(
            uploadUrl = presigned.uploadUrl,
            headers = presigned.headers,
            bytes = payload.bytes,
            mimeType = payload.mime,
        ).getOrThrow()
        when (val completed = AppContainer.mediaRepository.complete(listOf(presigned.uploadId))) {
            is ApiResult.Ok -> completed.data.uploads.firstOrNull() ?: error("上传结果为空")
            is ApiResult.Err -> error(completed.error.userMessage)
        }
    }

    private data class LocalImage(val bytes: ByteArray, val name: String, val mime: String)

    private fun readLocalImage(context: Context, uri: Uri): LocalImage {
        val mime = context.contentResolver.getType(uri) ?: "image/jpeg"
        val bytes = context.contentResolver.openInputStream(uri)?.use { it.readBytes() }
            ?: error("无法读取图片")
        val displayName = runCatching {
            context.contentResolver.query(uri, arrayOf(OpenableColumns.DISPLAY_NAME), null, null, null)
                ?.use { cursor ->
                    if (cursor.moveToFirst()) cursor.getString(0) else null
                }
        }.getOrNull()
        val extension = when {
            mime.contains("png") -> "png"
            mime.contains("webp") -> "webp"
            mime.contains("gif") -> "gif"
            else -> "jpg"
        }
        return LocalImage(
            bytes = bytes,
            name = displayName ?: ("chat_" + System.currentTimeMillis() + "." + extension),
            mime = mime,
        )
    }
}
