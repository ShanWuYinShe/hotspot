package com.hotspot.app.util

import com.hotspot.app.data.model.NotifyItem

/** 通知分类档是否命中（与 Web /notify FILTER_TABS 同口径：all/unread 不过滤 type）。 */
fun notifyCategoryMatches(type: String, filterId: String): Boolean =
    filterId == "all" || filterId == "unread" || type == filterId

/**
 * 通知列表过滤：分类优先，其次关键词（内容 / 用户 ID / 类型）。
 * 「未读」的服务端 is_read=false 由调用方在请求参数里处理，这里不再按 isRead 过滤，
 * 保证与 Web「已加载页内本地过滤」的行为一致。
 */
fun filterNotifyItems(items: List<NotifyItem>, query: String, filterId: String): List<NotifyItem> {
    val keyword = query.trim()
    return items.filter { item ->
        notifyCategoryMatches(item.type, filterId) &&
            (
                keyword.isBlank() ||
                    item.type.contains(keyword, ignoreCase = true) ||
                    item.notifyId.contains(keyword, ignoreCase = true) ||
                    item.payload.toString().contains(keyword, ignoreCase = true)
                )
    }
}
