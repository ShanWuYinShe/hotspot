package com.hotspot.app.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableIntStateOf
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
import com.hotspot.app.data.model.Page
import com.hotspot.app.data.model.ReportItem
import com.hotspot.app.ui.components.Badge
import com.hotspot.app.ui.components.BadgeTone
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.InlineLoading
import com.hotspot.app.ui.components.ListEnd
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SystemCard
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import kotlinx.coroutines.launch
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.format.DateTimeFormatter

private val REPORT_FILTERS = listOf(
    "全部" to null,
    "待处理" to "pending",
    "已处理" to "resolved",
    "已驳回" to "dismissed",
)

private val REPORT_DATE = DateTimeFormatter.ofPattern("M月d日 HH:mm").withZone(ZoneId.systemDefault())

/**
 * 我的举报（设计稿 87:3350）：顶部条数 → 全部/待处理/已处理/已驳回筛选 → 举报卡片列表
 * （举报对象·原因 + 描述 + 时间 + 状态徽章 + 处理结果行）。
 * 数据走 reports/fetch（created_at DESC），按状态服务端过滤 + 分页；空/错/加载三态。
 */
@Composable
fun ReportsScreen(onBack: () -> Unit) {
    val colors = HotSpotTheme.colors
    var filterIndex by remember { mutableIntStateOf(0) }

    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "我的举报", onBack = onBack)
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = Spacing.lg),
            horizontalArrangement = Arrangement.spacedBy(Spacing.sm),
        ) {
            REPORT_FILTERS.forEachIndexed { index, (label, _) ->
                val selected = index == filterIndex
                Box(
                    modifier = Modifier
                        .clip(AppShapes.pill)
                        .background(if (selected) colors.subtle else androidx.compose.ui.graphics.Color.Transparent)
                        .clickable { filterIndex = index }
                        .padding(horizontal = Spacing.md, vertical = Spacing.sm2),
                ) {
                    Text(
                        text = label,
                        style = MaterialTheme.typography.labelMedium,
                        fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Normal,
                        color = if (selected) colors.accent else colors.textSecondary,
                    )
                }
            }
        }
        Spacer(Modifier.height(Spacing.sm))
        key(filterIndex) {
            ReportsContent(status = REPORT_FILTERS[filterIndex].second)
        }
    }
}

@Composable
private fun ReportsContent(status: String?) {
    val colors = HotSpotTheme.colors
    val scope = rememberCoroutineScope()
    val state = rememberPagedListState<ReportItem>()
    val listState = rememberLazyListState()
    val loader: suspend (String?) -> ApiResult<Page<ReportItem>> = { cursor ->
        AppContainer.reportsRepository.fetch(status = status, cursor = cursor)
    }

    LaunchedEffect(Unit) { state.initialLoad(loader) }
    InfiniteScrollHandler(state, listState, loader)

    val countLabel = when {
        state.items.isEmpty() -> null
        state.endReached -> state.items.size.toString() + " 条"
        else -> state.items.size.toString() + "+ 条"
    }

    Column(Modifier.fillMaxSize()) {
        if (countLabel != null) {
            Text(
                text = countLabel,
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
                modifier = Modifier
                    .align(Alignment.End)
                    .padding(horizontal = Spacing.lg, vertical = Spacing.xs),
            )
        }
        when {
            state.loading && state.items.isEmpty() -> LoadingState(message = "正在加载举报记录…")
            state.error != null && state.items.isEmpty() -> ErrorState(
                message = state.error ?: "加载失败",
                onRetry = { scope.launch { state.initialLoad(loader) } },
            )
            state.items.isEmpty() -> EmptyState(
                title = if (status == null) "还没有举报记录" else "没有" + REPORT_FILTERS.first { it.second == status }.first + "的举报",
                description = if (status == null) "在瞬间/评论/消息等页面的更多菜单里可以举报违规内容" else "换个筛选看看",
            )
            else -> LazyColumn(
                state = listState,
                modifier = Modifier.fillMaxSize(),
                contentPadding = PaddingValues(
                    start = Spacing.lg,
                    end = Spacing.lg,
                    top = Spacing.xs,
                    bottom = Spacing.xxl,
                ),
            ) {
                item {
                    SystemCard(contentPadding = PaddingValues(Spacing.lg)) {
                        state.items.forEachIndexed { index, item ->
                            ReportRow(item = item)
                            if (index != state.items.lastIndex) {
                                Spacer(Modifier.height(Spacing.md))
                                Box(
                                    Modifier
                                        .fillMaxWidth()
                                        .height(1.dp)
                                        .background(colors.borderSubtle),
                                )
                                Spacer(Modifier.height(Spacing.md))
                            }
                        }
                    }
                }
                item {
                    when {
                        state.loadingMore -> InlineLoading()
                        state.endReached -> ListEnd()
                        else -> Box(Modifier.height(1.dp))
                    }
                }
            }
        }
    }
}

@Composable
private fun ReportRow(item: ReportItem) {
    val colors = HotSpotTheme.colors
    Column(Modifier.fillMaxWidth()) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = "举报" + targetTypeLabel(item.targetType) + " · " + reasonLabel(item.reason),
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = FontWeight.SemiBold,
                color = colors.textPrimary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
            Spacer(Modifier.width(Spacing.sm))
            val (label, tone) = statusStyle(item.status)
            Badge(text = label, tone = tone)
        }
        if (!item.description.isNullOrBlank()) {
            Spacer(Modifier.height(Spacing.xs))
            Text(
                text = item.description,
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
        }
        Spacer(Modifier.height(Spacing.xs))
        Text(
            text = formatReportDate(item.createdAt),
            style = MaterialTheme.typography.labelSmall,
            color = colors.textSecondary,
        )
        val resolution = item.resolution
        if (resolution != null) {
            Spacer(Modifier.height(Spacing.sm2))
            Row(verticalAlignment = Alignment.CenterVertically) {
                val resultText = resolution.note?.takeIf { it.isNotBlank() }
                    ?: actionLabel(resolution.action)
                Text(
                    text = "处理结果：" + resultText,
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f),
                )
                Spacer(Modifier.width(Spacing.sm))
                Text(
                    text = actionLabel(resolution.action),
                    style = MaterialTheme.typography.labelSmall,
                    fontWeight = FontWeight.Medium,
                    color = if (resolution.action == "dismiss") colors.textSecondary else colors.online,
                )
            }
        }
    }
}

private fun statusStyle(status: String): Pair<String, BadgeTone> = when (status) {
    "pending" -> "待处理" to BadgeTone.WARNING
    "resolved" -> "已处理" to BadgeTone.ONLINE
    "dismissed" -> "已驳回" to BadgeTone.NEUTRAL
    else -> status to BadgeTone.NEUTRAL
}

private fun actionLabel(action: String): String = when (action) {
    "resolve" -> "已确认处理"
    "dismiss" -> "已驳回"
    else -> action
}

private fun targetTypeLabel(type: String): String = when (type) {
    "user" -> "用户"
    "moment" -> "瞬间"
    "comment" -> "评论"
    "message" -> "消息"
    "group" -> "群组"
    else -> type
}

private fun reasonLabel(reason: String): String = when (reason) {
    "spam" -> "垃圾信息"
    "harassment" -> "骚扰/欺凌"
    "violence" -> "暴力/威胁"
    "hate_speech" -> "仇恨言论"
    "sexual_content" -> "色情/裸露内容"
    "misinformation" -> "虚假信息"
    "impersonation" -> "冒充他人"
    "self_harm" -> "自伤/自杀相关"
    "other" -> "其他"
    else -> reason
}

private fun formatReportDate(iso: String?): String {
    if (iso.isNullOrBlank()) return ""
    return try {
        REPORT_DATE.format(OffsetDateTime.parse(iso).toInstant())
    } catch (_: Exception) {
        try {
            REPORT_DATE.format(java.time.Instant.parse(iso))
        } catch (_: Exception) {
            ""
        }
    }
}
