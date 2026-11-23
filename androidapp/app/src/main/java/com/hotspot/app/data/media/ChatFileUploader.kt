package com.hotspot.app.data.media

import android.content.ContentResolver
import android.content.Context
import android.net.Uri
import android.provider.OpenableColumns
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.PresignFileInput
import com.hotspot.app.data.model.UploadedMedia
import com.hotspot.app.util.ChatFileRules
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * 聊天文件上传（spec/api_spec_media.md）：presign_upload(category=chat_file) → PUT 流式直传 → complete_upload。
 * 100MB 上限与黑名单由 ChatFileRules 先拦；PUT 走 MediaRepository.putObjectStream，不把整份文件读进内存。
 */
object ChatFileUploader {
    suspend fun upload(context: Context, uri: Uri): Result<UploadedMedia> = runCatching {
        val meta = withContext(Dispatchers.IO) { readMeta(context, uri) }
        ChatFileRules.blockedReason(meta.name, meta.size, meta.mime)?.let { error(it) }

        val presigned = when (
            val result = AppContainer.mediaRepository.presign(
                listOf(
                    PresignFileInput(
                        fileName = meta.name,
                        fileSize = meta.size,
                        mimeType = meta.mime,
                        category = "chat_file",
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
            openStream = { context.contentResolver.openInputStream(uri) ?: error("无法读取文件") },
        ).getOrThrow()
        when (val completed = AppContainer.mediaRepository.complete(listOf(presigned.uploadId))) {
            is ApiResult.Ok -> completed.data.uploads.firstOrNull() ?: error("上传结果为空")
            is ApiResult.Err -> error(completed.error.userMessage)
        }
    }

    data class LocalFileMeta(val name: String, val size: Long, val mime: String)

    private fun readMeta(context: Context, uri: Uri): LocalFileMeta {
        val resolver = context.contentResolver
        val openable = queryOpenable(resolver, uri)
        val displayName = openable?.first?.takeIf { it.isNotBlank() }
            ?: ("chat_file_" + System.currentTimeMillis())
        val size = openable?.second
            ?: openAssetLength(resolver, uri)
            ?: error("无法读取文件大小，请换一个文件")
        val mime = resolver.getType(uri)?.substringBefore(';')?.trim()?.lowercase().orEmpty()
            .ifBlank { "application/octet-stream" }
        return LocalFileMeta(name = displayName, size = size, mime = mime)
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
