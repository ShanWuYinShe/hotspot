package com.hotspot.app.util

import com.hotspot.app.BuildConfig
import java.net.URI
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.temporal.ChronoUnit

/**
 * 媒体 URL 归一：spec 允许 *_url 返回绝对 URL，或站内相对路径（如 /public/v1/media/avatars/<id>）。
 * 相对路径按「当前 API origin」拼接（BuildConfig.API_BASE_URL 只含 origin，不含 /api/v1 段）。
 */
object MediaUrls {
    fun absolute(url: String?): String? {
        if (url.isNullOrBlank()) return null
        if (url.startsWith("http://") || url.startsWith("https://")) return url
        return if (url.startsWith("/")) BuildConfig.API_BASE_URL + url else BuildConfig.API_BASE_URL + "/" + url
    }
}

/** 时间展示：API 统一返回 UTC ISO 8601；列表展示相对时间，详情展示本地时间。 */
object TimeFormat {
    private val formatter = DateTimeFormatter.ofPattern("MM-dd HH:mm").withZone(ZoneId.systemDefault())

    fun relative(iso: String?): String {
        val instant = parse(iso) ?: return ""
        val now = Instant.now()
        val minutes = ChronoUnit.MINUTES.between(instant, now)
        return when {
            minutes < 1 -> "刚刚"
            minutes < 60 -> minutes.toString() + " 分钟前"
            minutes < 60 * 24 -> (minutes / 60).toString() + " 小时前"
            minutes < 60 * 24 * 7 -> (minutes / (60 * 24)).toString() + " 天前"
            else -> formatter.format(instant)
        }
    }

    fun absolute(iso: String?): String {
        val instant = parse(iso) ?: return ""
        return formatter.format(instant)
    }

    private fun parse(iso: String?): Instant? {
        if (iso.isNullOrBlank()) return null
        return try {
            Instant.parse(iso)
        } catch (_: Exception) {
            try {
                java.time.OffsetDateTime.parse(iso).toInstant()
            } catch (_: Exception) {
                null
            }
        }
    }
}

/** 主机名展示（设置页 About 用；URL 非法时原样返回）。 */
fun originLabel(baseUrl: String): String = try {
    URI(baseUrl).host ?: baseUrl
} catch (_: Exception) {
    baseUrl
}
