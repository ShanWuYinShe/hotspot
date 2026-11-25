package com.hotspot.app.data.media

import android.content.ContentResolver
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
 * 聊天视频上传（spec/api_spec_media.md → chat_video）：
 * 100MB / video/mp4·video/quicktime 白名单，presign → 流式 PUT → complete，不整份读入内存。
 */
object ChatVideoUploader {
    private const val MAX_VIDEO_BYTES = 100L * 1024 * 1024
    private val ALLOWED_MIME = setOf("video/mp4", "video/quicktime")

    suspend fun upload(context: Context, uri: Uri): Result<UploadedMedia> = runCatching {
        val meta = withContext(Dispatchers.IO) { readMeta(context, uri) }
        require(meta.mime in ALLOWED_MIME) { "仅支持 MP4 / MOV 视频" }
        require(meta.size <= MAX_VIDEO_BYTES) { "视频不能超过 100MB" }

        val presigned = when (
            val result = AppContainer.mediaRepository.presign(
                listOf(
                    PresignFileInput(
                        fileName = meta.name,
                        fileSize = meta.size,
                        mimeType = meta.mime,
                        category = "chat_video",
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
            contentLength = meta.size,
            mimeType = meta.mime,
            openStream = { MediaReadErrors.open(context.contentResolver, uri, "视频") ?: error("无法读取视频") },
        ).getOrThrow()
        when (val completed = AppContainer.mediaRepository.complete(listOf(presigned.uploadId))) {
            is ApiResult.Ok -> completed.data.uploads.firstOrNull() ?: error("上传结果为空")
            is ApiResult.Err -> error(completed.error.userMessage)
        }
    }

    data class LocalVideoMeta(val name: String, val size: Long, val mime: String)

    private fun readMeta(context: Context, uri: Uri): LocalVideoMeta {
        val resolver = context.contentResolver
        val openable = queryOpenable(resolver, uri)
        val displayName = openable?.first?.takeIf { it.isNotBlank() }
            ?: ("chat_video_" + System.currentTimeMillis())
        val size = openable?.second
            ?: openAssetLength(resolver, uri)
            ?: error("无法读取视频大小，请换一个视频")
        val mime = resolveMime(resolver.getType(uri), displayName)
        return LocalVideoMeta(name = displayName, size = size, mime = mime)
    }

    private fun resolveMime(raw: String?, fileName: String): String {
        val normalized = raw?.substringBefore(';')?.trim()?.lowercase().orEmpty()
        if (normalized in ALLOWED_MIME) return normalized
        return when (fileName.substringAfterLast('.', "").lowercase()) {
            "mov" -> "video/quicktime"
            else -> "video/mp4"
        }
    }

    private fun queryOpenable(resolver: ContentResolver, uri: Uri): Pair<String?, Long?>? = runCatching {
        resolver.query(
            uri,
            arrayOf(OpenableColumns.DISPLAY_NAME, OpenableColumns.SIZE),
            null,
            null,
            null,
        )?.use { cursor ->
            if (!cursor.moveToFirst()) return@use null
            val nameIndex = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME)
            val sizeIndex = cursor.getColumnIndex(OpenableColumns.SIZE)
            val name = nameIndex.takeIf { it >= 0 && !cursor.isNull(it) }?.let(cursor::getString)
            val size = sizeIndex.takeIf { it >= 0 && !cursor.isNull(it) }?.let(cursor::getLong)?.takeIf { it > 0 }
            name to size
        }
    }.getOrNull()

    private fun openAssetLength(resolver: ContentResolver, uri: Uri): Long? = runCatching {
        resolver.openAssetFileDescriptor(uri, "r")?.use { descriptor -> descriptor.length.takeIf { it > 0 } }
    }.getOrNull()
}
