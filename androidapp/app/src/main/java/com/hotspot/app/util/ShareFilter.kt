package com.hotspot.app.util

import com.hotspot.app.data.model.ShareListItem

/**
 * 「我的分享」本地搜索（W19 对齐 Web lib/shares.filterShares）。
 * 契约 share/fetch_created 无 keyword 参数，因此只过滤已加载的记录；
 * 匹配分享类型文案、标题、副标题、分享码与可用状态（有效/已失效/已撤销）。
 */
fun filterShares(items: List<ShareListItem>, query: String): List<ShareListItem> {
    val q = query.trim().lowercase()
    if (q.isEmpty()) return items
    return items.filter { item ->
        val preview = item.preview
        listOf(
            shareTypeLabel(item.shareType),
            preview?.title ?: "",
            preview?.subtitle ?: "",
            item.shareCode,
            if (item.isRevoked) "已失效 已撤销" else "有效",
        ).joinToString(" ").lowercase().contains(q)
    }
}

/** 分享类型文案（与 Web lib/shares SHARE_TYPE_LABELS 同口径）。 */
fun shareTypeLabel(type: String): String = when (type) {
    "moment" -> "瞬间"
    "user" -> "用户"
    "group" -> "群组"
    "location" -> "位置"
    else -> "分享"
}
