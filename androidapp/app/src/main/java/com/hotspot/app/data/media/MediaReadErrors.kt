package com.hotspot.app.data.media

import android.content.ContentResolver
import android.net.Uri
import java.io.FileNotFoundException
import java.io.InputStream

/**
 * Photo Picker 选到云端占位图时，ContentResolver 会在打开阶段抛 FileNotFoundException
 * （真实设备实测：相册有缩略图但原图未下载）。统一转成可读文案，其它异常原样抛出。
 */
object MediaReadErrors {
    fun describe(error: Throwable, label: String): String? = when (error) {
        is FileNotFoundException -> "${label}尚未下载到本机，请换一个或先在相册中下载"
        is SecurityException -> "没有访问该${label}的权限，请重新选择"
        else -> null
    }

    fun open(resolver: ContentResolver, uri: Uri, label: String): InputStream? =
        try {
            resolver.openInputStream(uri)
        } catch (e: FileNotFoundException) {
            throw IllegalStateException(describe(e, label), e)
        } catch (e: SecurityException) {
            throw IllegalStateException(describe(e, label), e)
        }
}
