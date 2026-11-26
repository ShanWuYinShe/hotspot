package com.hotspot.app.data.update

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient
import okhttp3.Request
import java.util.concurrent.TimeUnit

private const val RELEASE_API = "https://api.github.com/repos/ShanWuYinShe/hotspot/releases/latest"

@Serializable
private data class GitHubRelease(
    @SerialName("tag_name") val tagName: String = "",
    @SerialName("html_url") val htmlUrl: String = "",
    val body: String? = null,
    val assets: List<GitHubAsset> = emptyList(),
)

@Serializable
private data class GitHubAsset(
    val name: String = "",
    @SerialName("browser_download_url") val browserDownloadUrl: String = "",
)

/** 检查结果：latestVersion 已去掉 v 前缀；apkUrl 优先匹配 Android APK 资产。 */
data class AppUpdateInfo(
    val latestVersion: String,
    val currentVersion: String,
    val isNewer: Boolean,
    val notes: String,
    val pageUrl: String,
    val apkUrl: String?,
)

/** 通过 GitHub Releases API 检查 Android 新版本（匿名只读，无需后端）。 */
class UpdateChecker {
    private val json = Json { ignoreUnknownKeys = true }
    private val client = OkHttpClient.Builder()
        .connectTimeout(10, TimeUnit.SECONDS)
        .readTimeout(15, TimeUnit.SECONDS)
        .callTimeout(20, TimeUnit.SECONDS)
        .build()

    suspend fun check(currentVersion: String): Result<AppUpdateInfo> = withContext(Dispatchers.IO) {
        runCatching {
            val request = Request.Builder()
                .url(RELEASE_API)
                .header("Accept", "application/vnd.github+json")
                .header("User-Agent", "HotSpot-Android")
                .build()
            client.newCall(request).execute().use { response ->
                if (!response.isSuccessful) error("更新检查失败（HTTP " + response.code + "）")
                val release = json.decodeFromString<GitHubRelease>(response.body.string())
                val latest = release.tagName.removePrefix("v").trim()
                if (latest.isBlank()) error("更新信息缺少版本号")
                AppUpdateInfo(
                    latestVersion = latest,
                    currentVersion = currentVersion.removePrefix("v"),
                    isNewer = isNewerVersion(latest, currentVersion),
                    notes = release.body.orEmpty(),
                    pageUrl = release.htmlUrl,
                    apkUrl = release.assets.firstOrNull { it.name.startsWith("hotspot-android-") }?.browserDownloadUrl,
                )
            }
        }
    }
}

/** 语义版本比较：按数字段比较，长度不足补 0；仅用于 x.y.z tag。 */
internal fun isNewerVersion(latest: String, current: String): Boolean {
    fun parts(value: String): List<Int> = value.removePrefix("v").split('.').map { it.toIntOrNull() ?: 0 }
    val a = parts(latest)
    val b = parts(current)
    val size = maxOf(a.size, b.size)
    for (i in 0 until size) {
        val x = a.getOrElse(i) { 0 }
        val y = b.getOrElse(i) { 0 }
        if (x != y) return x > y
    }
    return false
}
