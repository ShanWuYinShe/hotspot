package com.hotspot.app.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import android.content.ActivityNotFoundException
import android.content.Intent
import android.net.Uri
import android.widget.Toast
import androidx.compose.ui.platform.LocalContext
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.NotifyItem
import com.hotspot.app.data.model.SystemAnnouncementItem
import com.hotspot.app.ui.components.Badge
import com.hotspot.app.ui.components.BadgeTone
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.InlineLoading
import com.hotspot.app.ui.components.ListGroup
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SearchInput
import com.hotspot.app.ui.components.SystemCard
import com.hotspot.app.ui.components.TabPills
import com.hotspot.app.util.filterNotifyItems
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.TimeFormat
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

/**
 * 通知中心：列表 + 未读标记 + 全部已读。
 * payload 是多态 JSON（spec/api_spec_notify.md），首版按类型映射标题并从 payload 抽取预览文本。
 */
@Composable
fun NotifyScreen(
    onBack: () -> Unit,
    onOpenUser: (String) -> Unit,
    onOpenMoment: (String) -> Unit,
    onOpenMomentComment: (String, String) -> Unit,
    onOpenGroup: (String) -> Unit,
    onOpenGroupChat: (String) -> Unit,
    onOpenReports: () -> Unit,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    var items by remember { mutableStateOf<List<NotifyItem>>(emptyList()) }
    var announcements by remember { mutableStateOf<List<SystemAnnouncementItem>>(emptyList()) }
    // (filterId, 文案) 顺序与 Web FILTER_TABS 一致
    val notifyTabs: List<Pair<String, String>> = listOf(
        "all" to "全部",
        "unread" to "未读",
        "reaction" to "点赞",
        "follow" to "关注",
        "comment" to "评论",
        "nearby" to "附近",
    )
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var marking by remember { mutableStateOf(false) }
    var retryKey by remember { mutableIntStateOf(0) }
    var query by remember { mutableStateOf("") }
    var cursor by remember { mutableStateOf<String?>(null) }
    var loadingMore by remember { mutableStateOf(false) }
    var searchLoadingAll by remember { mutableStateOf(false) }
    // 分类档对齐 Web /notify FILTER_TABS（全部/未读/点赞/关注/评论/附近）：
    // 「未读」走服务端 is_read=false；其余分类在已加载页内按 type 过滤（契约单 type 筛选）。
    var filterId by remember { mutableStateOf("all") }
    val unreadOnly = filterId == "unread"

    suspend fun load() {
        loading = items.isEmpty()
        error = null
        when (val result = AppContainer.notifyRepository.fetch(null, if (unreadOnly) false else null)) {
            is ApiResult.Ok -> {
                items = result.data.items
                cursor = result.data.nextCursor
            }
            is ApiResult.Err -> error = result.error.userMessage
        }
        loading = false
    }

    LaunchedEffect(retryKey, unreadOnly) {
        // 筛选条件变化时重置分页，交给服务端按 is_read 返回，避免只在已加载页内过滤。
        items = emptyList()
        cursor = null
        load()
    }

    LaunchedEffect(retryKey) {
        when (val result = AppContainer.systemRepository.announcements()) {
            is ApiResult.Ok -> announcements = result.data
            is ApiResult.Err -> Unit
        }
    }

    fun loadMore() {
        val next = cursor ?: return
        if (loadingMore || loading) return
        loadingMore = true
        scope.launch {
            when (val result = AppContainer.notifyRepository.fetch(next, if (unreadOnly) false else null)) {
                is ApiResult.Ok -> {
                    items = (items + result.data.items).distinctBy { it.notifyId }
                    cursor = result.data.nextCursor
                }
                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
            }
            loadingMore = false
        }
    }

    // 通知契约没有关键词参数：搜索时先加载完分页（最多 10 页），再本地筛选。
    // 把 loading 纳入 key：首屏未加载完时输入关键词不会漏掉「加载完成后再补全分页」。
    LaunchedEffect(query, retryKey, loading) {
        if (query.isBlank()) {
            // 搜索中被清空时 effect 会被取消，这里复位，避免底部一直显示「正在加载」。
            searchLoadingAll = false
            return@LaunchedEffect
        }
        if (loading) return@LaunchedEffect
        var next: String? = cursor ?: return@LaunchedEffect
        searchLoadingAll = true
        var pages = 0
        while (next != null && pages < 10) {
            when (val result = AppContainer.notifyRepository.fetch(next, if (unreadOnly) false else null)) {
                is ApiResult.Ok -> {
                    items = (items + result.data.items).distinctBy { it.notifyId }
                    cursor = result.data.nextCursor
                    next = result.data.nextCursor
                }
                is ApiResult.Err -> break
            }
            pages += 1
        }
        searchLoadingAll = false
    }

    val filteredItems = remember(items, query, filterId) { filterNotifyItems(items, query, filterId) }

    /** 点击通知：先标记已读，再按 payload 跳到对应页面（无可跳转目标时只标已读）。 */
    fun openNotification(item: NotifyItem) {
        if (!item.isRead) {
            scope.launch {
                when (val result = AppContainer.notifyRepository.markRead(item.notifyId)) {
                    // 只更新本地已读态：重新拉首屏会丢掉分页/搜索已加载的条目。
                    is ApiResult.Ok -> items = if (unreadOnly) {
                        items.filterNot { it.notifyId == item.notifyId }
                    } else {
                        items.map { if (it.notifyId == item.notifyId) it.copy(isRead = true) else it }
                    }
                    is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                }
            }
        }
        val payload = item.payload as? JsonObject ?: return
        fun field(key: String): String? = payload[key]?.jsonPrimitive?.contentOrNull
        when (item.type) {
            "new_follower", "mutual_follow" -> {
                val userId = payload["actor"]?.jsonObject?.get("user_id")?.jsonPrimitive?.contentOrNull
                if (!userId.isNullOrBlank()) onOpenUser(userId)
            }
            "moment_commented", "comment_reply", "mentioned" -> {
                val momentId = field("moment_id")
                val commentId = field("comment_id") ?: field("reply_id")
                if (!momentId.isNullOrBlank()) {
                    if (!commentId.isNullOrBlank()) onOpenMomentComment(momentId, commentId) else onOpenMoment(momentId)
                }
            }
            "nearby_new_moment" -> field("moment_id")?.let(onOpenMoment)
            "group_mentioned" -> field("group_id")?.let(onOpenGroupChat)
            "group_invite_received",
            "group_application_received",
            "group_application_accepted",
            "group_application_rejected",
            -> field("group_id")?.let(onOpenGroup)
            "report_resolved", "report_dismissed" -> onOpenReports()
            "reaction_added" -> if (field("target_type") == "moment") field("target_id")?.let(onOpenMoment)
        }
    }

    Column(Modifier.fillMaxSize()) {
        PageHeader(
            title = "通知",
            onBack = onBack,
            actions = {
                if (items.any { !it.isRead }) {
                    SystemButton(
                        text = if (marking) "处理中…" else "全部已读",
                        onClick = {
                            if (marking) return@SystemButton
                            scope.launch {
                                marking = true
                                when (val result = AppContainer.notifyRepository.markAllRead()) {
                                    // 只更新本地已读态，保留已加载的分页与搜索结果。
                                    is ApiResult.Ok -> items = items.map { it.copy(isRead = true) }
                                    is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                                }
                                marking = false
                            }
                        },
                        variant = ButtonVariant.SOFT,
                        size = ButtonSize.SM,
                    )
                }
            },
        )
        Column(Modifier.padding(horizontal = Spacing.lg)) {
            SearchInput(
                value = query,
                onValueChange = { query = it },
                placeholder = "搜索通知内容 / 用户 ID / 类型…",
            )
            Spacer(Modifier.height(Spacing.sm))
            TabPills(
                options = notifyTabs.map { it.second },
                selectedIndex = notifyTabs.indexOfFirst { it.first == filterId }.coerceAtLeast(0),
                onSelected = { index -> filterId = notifyTabs.getOrNull(index)?.first ?: "all" },
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(Spacing.sm))
        }
        if (announcements.isNotEmpty()) {
            Column(Modifier.padding(horizontal = Spacing.lg)) {
                Text(
                    text = "平台公告",
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                    color = HotSpotTheme.colors.textPrimary,
                )
                Spacer(Modifier.height(Spacing.sm))
                announcements.take(3).forEach { announcement ->
                    AnnouncementCard(announcement)
                    Spacer(Modifier.height(Spacing.sm))
                }
            }
        }
        when {
            loading -> LoadingState(message = "正在加载通知…")
            error != null && items.isEmpty() -> ErrorState(
                message = error ?: "加载失败",
                onRetry = { retryKey++ },
            )
            items.isEmpty() -> if (announcements.isEmpty()) {
                EmptyState(
                    title = "暂时没有通知",
                    description = "互动、关注与系统消息都会出现在这里",
                )
            }
            searchLoadingAll && filteredItems.isEmpty() -> LoadingState(message = "正在加载全部通知用于搜索…")
            filteredItems.isEmpty() -> EmptyState(
                title = when {
                    unreadOnly && query.isBlank() -> "没有未读通知"
                    filterId != "all" && query.isBlank() -> "该分类下没有通知"
                    else -> "没有匹配的通知"
                },
                description = when {
                    unreadOnly && query.isBlank() -> "全部通知都已读，切回「全部」查看历史"
                    filterId != "all" && query.isBlank() -> "换个分类看看，或切回「全部」"
                    else -> "换个通知内容、用户 ID 或类型关键词试试"
                },
            )
            else -> LazyColumn(
                modifier = Modifier.fillMaxSize(),
                contentPadding = androidx.compose.foundation.layout.PaddingValues(
                    horizontal = Spacing.lg,
                    vertical = Spacing.sm,
                ),
                verticalArrangement = Arrangement.spacedBy(Spacing.sm),
            ) {
                item {
                    Text(
                        text = "共 " + items.size + " 条" + if (query.isNotBlank()) " · 匹配 " + filteredItems.size + " 条" else "",
                        style = MaterialTheme.typography.labelSmall,
                        color = HotSpotTheme.colors.textSecondary,
                    )
                }
                items(filteredItems, key = { it.notifyId }) { item ->
                    NotificationRow(item, onClick = { openNotification(item) })
                }
                if (searchLoadingAll) {
                    item { InlineLoading(text = "正在加载全部通知用于搜索…") }
                } else if (query.isBlank() && cursor != null) {
                    item {
                        Box(Modifier.fillMaxWidth().padding(vertical = Spacing.lg)) {
                            SystemButton(
                                text = if (loadingMore) "加载中…" else "加载更多",
                                onClick = { loadMore() },
                                variant = ButtonVariant.SECONDARY,
                                size = ButtonSize.SM,
                                enabled = !loadingMore,
                                loading = loadingMore,
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun NotificationRow(item: NotifyItem, onClick: () -> Unit) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(androidx.compose.foundation.shape.RoundedCornerShape(16.dp))
            .background(if (item.isRead) colors.surface else colors.accentSoftBg)
            .clickable(onClick = onClick)
            .padding(horizontal = Spacing.md, vertical = Spacing.md2),
        verticalAlignment = Alignment.Top,
    ) {
        Box(
            modifier = Modifier
                .padding(top = 6.dp)
                .size(8.dp)
                .clip(CircleShape)
                .background(if (item.isRead) androidx.compose.ui.graphics.Color.Transparent else colors.accent),
        )
        Spacer(Modifier.width(Spacing.sm))
        Column(Modifier.weight(1f)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = notifyTitle(item),
                    style = MaterialTheme.typography.bodyLarge,
                    fontWeight = if (item.isRead) FontWeight.Normal else FontWeight.SemiBold,
                    color = colors.textPrimary,
                    modifier = Modifier.weight(1f),
                )
                Text(
                    text = TimeFormat.relative(item.createdAt),
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
            }
            val body = notifyBody(item)
            if (body.isNotBlank()) {
                Spacer(Modifier.height(2.dp))
                Text(
                    text = body,
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.textSecondary,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}

private fun payloadObject(item: NotifyItem): JsonObject? = try {
    item.payload as? JsonObject
} catch (_: Exception) {
    null
}

private fun payloadString(item: NotifyItem, key: String): String? = try {
    payloadObject(item)?.get(key)?.jsonPrimitive?.contentOrNull
} catch (_: Exception) {
    null
}

private fun actorNickname(item: NotifyItem): String? = try {
    payloadObject(item)?.get("actor")?.jsonObject?.get("nickname")?.jsonPrimitive?.contentOrNull
} catch (_: Exception) {
    null
}

private fun notifyTitle(item: NotifyItem): String = when (item.type) {
    "new_follower" -> "有人关注了你"
    "mutual_follow" -> "你们互相关注了"
    "moment_commented" -> "你的瞬间收到新评论"
    "comment_reply" -> "有人回复了你的评论"
    "mentioned" -> "有人在内容里提到了你"
    "group_mentioned" -> "群聊里有人 @ 了你"
    "reaction_added" -> "有人回应了你的内容"
    "group_invite_received" -> "你收到一条群邀请"
    "group_application_received" -> "你收到一条群申请"
    "group_application_accepted" -> "你的群申请已通过"
    "group_application_rejected" -> "你的群申请被拒绝"
    "report_resolved" -> "举报处理结果"
    "system_announcement" -> "系统公告"
    else -> "新通知"
}

private fun notifyBody(item: NotifyItem): String {
    val actor = actorNickname(item)
    val preview = payloadString(item, "comment_preview")
        ?: payloadString(item, "reply_content_preview")
        ?: payloadString(item, "content_preview")
        ?: payloadString(item, "message")
        ?: payloadString(item, "emoji")
    return listOfNotNull(actor, preview).joinToString("：")
}
@Composable
private fun AnnouncementCard(announcement: SystemAnnouncementItem) {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    val actionUrl = announcement.actionUrl?.trim().orEmpty()
    fun openAction() {
        if (actionUrl.isBlank()) return
        try {
            context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(actionUrl)))
        } catch (e: ActivityNotFoundException) {
            Toast.makeText(context, "无法打开链接", Toast.LENGTH_SHORT).show()
        }
    }
    val tone = when (announcement.severity) {
        "warning" -> BadgeTone.WARNING
        "error" -> BadgeTone.DANGER
        else -> BadgeTone.ACCENT
    }
    val label = when (announcement.severity) {
        "warning" -> "警告"
        "error" -> "重要"
        else -> "通知"
    }
    SystemCard(
        contentPadding = androidx.compose.foundation.layout.PaddingValues(Spacing.md),
        modifier = if (actionUrl.isNotBlank()) Modifier.clickable { openAction() } else Modifier,
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = announcement.title,
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.SemiBold,
                color = colors.textPrimary,
                modifier = Modifier.weight(1f),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Spacer(Modifier.width(Spacing.sm))
            Badge(text = label, tone = tone)
        }
        Spacer(Modifier.height(Spacing.xs))
        Text(
            text = announcement.content,
            style = MaterialTheme.typography.bodySmall,
            color = colors.textSecondary,
            maxLines = 3,
            overflow = TextOverflow.Ellipsis,
        )
        if (actionUrl.isNotBlank()) {
            Spacer(Modifier.height(Spacing.xs))
            Text(
                text = "查看详情",
                style = MaterialTheme.typography.labelMedium,
                fontWeight = FontWeight.Medium,
                color = colors.accent,
            )
        }
    }
}
