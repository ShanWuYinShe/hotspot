package com.hotspot.app.data.location

import android.content.Context
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/** 用户在地图选点页设置的浏览锚点（附近页按这里计算距离与推荐）。 */
@Serializable
data class BrowseAnchor(
    val latitude: Double,
    val longitude: Double,
    val label: String,
)

/**
 * 浏览锚点本地存储（SharedPreferences）：
 * - current：当前生效锚点；null = 未设置，回退 server client_config.defaults.default_location。
 * - recent：最近选择（最多 5 条，去重）供地图页「最近使用」展示。
 */
class BrowseAnchorStore(context: Context) {
    private val prefs = context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
    private val json = Json { ignoreUnknownKeys = true }

    fun current(): BrowseAnchor? = runCatching {
        prefs.getString(KEY_CURRENT, null)?.let { json.decodeFromString<BrowseAnchor>(it) }
    }.getOrNull()

    fun recent(): List<BrowseAnchor> = runCatching {
        prefs.getString(KEY_RECENT, null)?.let {
            json.decodeFromString<List<BrowseAnchor>>(it)
        }
    }.getOrNull().orEmpty()

    /** 保存当前锚点并把它插入最近使用（按坐标+名称去重，最多保留 5 条）。 */
    fun save(anchor: BrowseAnchor) {
        val updated = (listOf(anchor) + recent().filterNot { samePlace(it, anchor) }).take(MAX_RECENT)
        prefs.edit()
            .putString(KEY_CURRENT, json.encodeToString(BrowseAnchor.serializer(), anchor))
            .putString(KEY_RECENT, json.encodeToString(kotlinx.serialization.builtins.ListSerializer(BrowseAnchor.serializer()), updated))
            .apply()
    }

    fun addRecent(anchor: BrowseAnchor) {
        val updated = (listOf(anchor) + recent().filterNot { samePlace(it, anchor) }).take(MAX_RECENT)
        prefs.edit()
            .putString(KEY_RECENT, json.encodeToString(kotlinx.serialization.builtins.ListSerializer(BrowseAnchor.serializer()), updated))
            .apply()
    }

    fun clear() {
        prefs.edit().remove(KEY_CURRENT).remove(KEY_RECENT).apply()
    }

    private fun samePlace(a: BrowseAnchor, b: BrowseAnchor): Boolean =
        a.label == b.label && kotlin.math.abs(a.latitude - b.latitude) < 1e-5 && kotlin.math.abs(a.longitude - b.longitude) < 1e-5

    private companion object {
        const val PREFS_NAME = "hotspot_browse_anchor"
        const val KEY_CURRENT = "current"
        const val KEY_RECENT = "recent"
        const val MAX_RECENT = 5
    }
}
