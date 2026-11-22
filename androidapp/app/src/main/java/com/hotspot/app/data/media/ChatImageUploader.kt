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
import java.io.ByteArrayOutputStream
import java.io.InputStream

/**
 * 聊天图片上传（spec/api_spec_media.md）：presign_upload(category=chat_image) → PUT 直传 → complete_upload。
 * 复用 MediaRepository 的三步链路；10MB / 图片 MIME 白名单在客户端先拦一次，服务端仍会二次校验。
 *
 * 10MB 校验必须在读取前完成：Photo Picker 可能返回几十 MB 的原图，先 readBytes 再判大小会让
 * 超限图片先整份进内存（低内存设备直接 OOM）。provider 能给出尺寸时先拒绝；给不出时边读边限流。
 */
object ChatImageUploader {
    private const val MAX_CHAT_IMAGE_BYTES = 10L * 1024 * 1024
    private const val READ_BUFFER_BYTES = 16 * 1024
    private val ALLOWED_MIME = setOf("image/jpeg", "image/png", "image/webp", "image/gif")

    suspend fun upload(context: Context, uri: Uri): Result<UploadedMedia> = runCatching {
        val payload = withContext(Dispatchers.IO) { readLocalImage(context, uri) }
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

    private data class OpenableMeta(val displayName: String?, val size: Long?)

    private fun readLocalImage(context: Context, uri: Uri): LocalImage {
        val resolver = context.contentResolver
        val meta = queryOpenable(resolver, uri)
        val mime = resolveMime(resolver.getType(uri), meta.displayName)
        require(mime in ALLOWED_MIME) { "仅支持 JPG/PNG/WebP/GIF 图片" }

        val declaredSize = meta.size ?: openAssetLength(resolver, uri)
        require(declaredSize == null || declaredSize <= MAX_CHAT_IMAGE_BYTES) { "图片不能超过 10MB" }

        val bytes = resolver.openInputStream(uri)?.use { readAtMost(it, MAX_CHAT_IMAGE_BYTES) }
            ?: error("无法读取图片")
        val extension = when (mime) {
            "image/png" -> "png"
            "image/webp" -> "webp"
            "image/gif" -> "gif"
            else -> "jpg"
        }
        return LocalImage(
            bytes = bytes,
            name = meta.displayName?.takeIf { it.isNotBlank() }
                ?: ("chat_" + System.currentTimeMillis() + "." + extension),
            mime = mime,
        )
    }

    /** DISPLAY_NAME / SIZE 一次查询；provider 不支持或抛异常时按未知处理，不阻塞上传。 */
    private fun queryOpenable(resolver: ContentResolver, uri: Uri): OpenableMeta = runCatching {
        resolver.query(
            uri,
            arrayOf(OpenableColumns.DISPLAY_NAME, OpenableColumns.SIZE),
            null,
            null,
            null,
        )?.use { cursor ->
            if (!cursor.moveToFirst()) return@use OpenableMeta(null, null)
            val nameIndex = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME)
            val sizeIndex = cursor.getColumnIndex(OpenableColumns.SIZE)
            val name = nameIndex.takeIf { it >= 0 && !cursor.isNull(it) }?.let(cursor::getString)
            val size = sizeIndex.takeIf { it >= 0 && !cursor.isNull(it) }?.let(cursor::getLong)
            OpenableMeta(name, size?.takeIf { it > 0 })
        } ?: OpenableMeta(null, null)
    }.getOrDefault(OpenableMeta(null, null))

    /** 部分 provider 的 cursor 不带 SIZE，退回文件描述符长度（未知为 -1）。 */
    private fun openAssetLength(resolver: ContentResolver, uri: Uri): Long? = runCatching {
        resolver.openAssetFileDescriptor(uri, "r")?.use { descriptor ->
            descriptor.length.takeIf { it >= 0 }
        }
    }.getOrNull()

    /** 部分 provider 返回 "image/jpg" 这类别名；无 MIME 时按扩展名兜底。 */
    private fun resolveMime(raw: String?, displayName: String?): String {
        val declared = raw?.substringBefore(';')?.trim()?.lowercase().orEmpty()
        if (declared.isNotEmpty()) {
            return if (declared == "image/jpg" || declared == "image/pjpeg") "image/jpeg" else declared
        }
        return when (displayName?.substringAfterLast('.', "")?.lowercase()) {
            "png" -> "image/png"
            "webp" -> "image/webp"
            "gif" -> "image/gif"
            else -> "image/jpeg"
        }
    }

    /** 读取时按上限截断，provider 未给出 SIZE（或实际内容与声明不符）时仍不会整份读入超限文件。 */
    private fun readAtMost(input: InputStream, limit: Long): ByteArray {
        val output = ByteArrayOutputStream()
        val buffer = ByteArray(READ_BUFFER_BYTES)
        var total = 0L
        while (true) {
            val read = input.read(buffer)
            if (read < 0) break
            total += read
            require(total <= limit) { "图片不能超过 10MB" }
            output.write(buffer, 0, read)
        }
        return output.toByteArray()
    }
}
