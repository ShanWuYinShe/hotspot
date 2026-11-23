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
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.NotifyItem
import com.hotspot.app.ui.components.Badge
import com.hotspot.app.ui.components.BadgeTone
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.ListGroup
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
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
    onOpenGroup: (String) -> Unit,
    onOpenGroupChat: (String) -> Unit,
    onOpenReports: () -> Unit,
) {
    val scope = rememberCoroutineScope()
    var items by remember { mutableStateOf<List<NotifyItem>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var marking by remember { mutableStateOf(false) }
    var retryKey by remember { mutableIntStateOf(0) }

    suspend fun load() {
        loading = items.isEmpty()
        error = null
        when (val result = AppContainer.notifyRepository.fetch(null)) {
            is ApiResult.Ok -> items = result.data.items
            is ApiResult.Err -> error = result.error.userMessage
        }
        loading = false
    }

    LaunchedEffect(retryKey) { load() }

    /** 点击通知：先标记已读，再按 payload 跳到对应页面（无可跳转目标时只标已读）。 */
    fun openNotification(item: NotifyItem) {
        if (!item.isRead) {
            scope.launch {
                AppContainer.notifyRepository.markRead(item.notifyId)
                load()
            }
        }
        val payload = item.payload as? JsonObject ?: return
        fun field(key: String): String? = payload[key]?.jsonPrimitive?.contentOrNull
        when (item.type) {
            "new_follower", "mutual_follow" -> {
                val userId = payload["actor"]?.jsonObject?.get("user_id")?.jsonPrimitive?.contentOrNull
                if (!userId.isNullOrBlank()) onOpenUser(userId)
            }
            "moment_commented", "comment_reply", "mentioned", "nearby_new_moment" -> field("moment_id")?.let(onOpenMoment)
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
                                AppContainer.notifyRepository.markAllRead()
                                marking = false
                                load()
                            }
                        },
                        variant = ButtonVariant.SOFT,
                        size = ButtonSize.SM,
                    )
                }
            },
        )
        when {
            loading -> LoadingState(message = "正在加载通知…")
            error != null && items.isEmpty() -> ErrorState(
                message = error ?: "加载失败",
                onRetry = { retryKey++ },
            )
            items.isEmpty() -> EmptyState(
                title = "暂时没有通知",
                description = "互动、关注与系统消息都会出现在这里",
            )
            else -> LazyColumn(
                modifier = Modifier.fillMaxSize(),
                contentPadding = androidx.compose.foundation.layout.PaddingValues(
                    horizontal = Spacing.lg,
                    vertical = Spacing.sm,
                ),
                verticalArrangement = Arrangement.spacedBy(Spacing.sm),
            ) {
                items(items, key = { it.notifyId }) { item ->
                    NotificationRow(item, onClick = { openNotification(item) })
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
