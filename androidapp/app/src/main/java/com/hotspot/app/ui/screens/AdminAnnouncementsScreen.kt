package com.hotspot.app.ui.screens

import android.widget.Toast
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Switch
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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextOverflow
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.AdminAnnouncementItem
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.InlineLoading
import com.hotspot.app.ui.components.ListEnd
import com.hotspot.app.ui.components.ListGroup
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SearchInput
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemInput
import com.hotspot.app.ui.components.SystemTextArea
import com.hotspot.app.ui.components.TabPills
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.TimeFormat
import kotlinx.coroutines.launch

private val AnnouncementSeverities = listOf("info" to "通知", "warning" to "警告", "error" to "错误")

private fun severityLabel(value: String): String =
    AnnouncementSeverities.firstOrNull { it.first == value }?.second ?: value

/**
 * 管理后台 · 系统公告（仅平台管理员）：按严重级别筛选、发布/编辑/删除系统公告。
 * 发布走 admin/announcements/create（MUST 幂等），编辑/删除走 update/delete；
 * 发布时可选 send_push（推送由服务端执行，客户端只透传）。
 */
@Composable
fun AdminAnnouncementsScreen(onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val colors = HotSpotTheme.colors
    var severityIndex by remember { mutableIntStateOf(0) }
    var items by remember { mutableStateOf<List<AdminAnnouncementItem>>(emptyList()) }
    var cursor by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var loadingMore by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var reloadKey by remember { mutableIntStateOf(0) }
    var editorOpen by remember { mutableStateOf(false) }
    var editing by remember { mutableStateOf<AdminAnnouncementItem?>(null) }
    var title by remember { mutableStateOf("") }
    var content by remember { mutableStateOf("") }
    var actionUrl by remember { mutableStateOf("") }
    var editorSeverityIndex by remember { mutableIntStateOf(0) }
    var sendPush by remember { mutableStateOf(false) }
    var saving by remember { mutableStateOf(false) }
    var deleteTarget by remember { mutableStateOf<AdminAnnouncementItem?>(null) }
    var deleting by remember { mutableStateOf(false) }
    var query by remember { mutableStateOf("") }
    var searchLoadingAll by remember { mutableStateOf(false) }

    val severityFilter = if (severityIndex == 0) null else AnnouncementSeverities[severityIndex - 1].first

    LaunchedEffect(severityIndex, reloadKey) {
        loading = true
        error = null
        when (val result = AppContainer.adminRepository.fetchAnnouncements(severityFilter, null)) {
            is ApiResult.Ok -> {
                items = result.data.items
                cursor = result.data.nextCursor
            }
            is ApiResult.Err -> error = result.error.userMessage
        }
        loading = false
    }

    fun loadMore() {
        val next = cursor ?: return
        if (loadingMore || loading) return
        loadingMore = true
        scope.launch {
            when (val result = AppContainer.adminRepository.fetchAnnouncements(severityFilter, next)) {
                is ApiResult.Ok -> {
                    items = (items + result.data.items).distinctBy { it.announcementId }
                    cursor = result.data.nextCursor
                }
                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
            }
            loadingMore = false
        }
    }

    // 契约的 announcements/fetch 没有关键词参数：搜索时先加载完分页（最多 10 页），再做本地筛选。
    LaunchedEffect(query, severityIndex, reloadKey) {
        if (query.isBlank() || loading) return@LaunchedEffect
        var next: String? = cursor ?: return@LaunchedEffect
        searchLoadingAll = true
        var pages = 0
        while (next != null && pages < 10) {
            when (val result = AppContainer.adminRepository.fetchAnnouncements(severityFilter, next)) {
                is ApiResult.Ok -> {
                    items = (items + result.data.items).distinctBy { it.announcementId }
                    cursor = result.data.nextCursor
                    next = result.data.nextCursor
                }
                is ApiResult.Err -> break
            }
            pages += 1
        }
        searchLoadingAll = false
    }

    val filteredItems = remember(items, query) {
        val keyword = query.trim()
        if (keyword.isBlank()) items else items.filter { item ->
            item.title.contains(keyword, ignoreCase = true) ||
                item.content.contains(keyword, ignoreCase = true) ||
                item.createdBy.nickname.contains(keyword, ignoreCase = true)
        }
    }

    fun openEditor(item: AdminAnnouncementItem?) {
        editing = item
        title = item?.title.orEmpty()
        content = item?.content.orEmpty()
        actionUrl = item?.actionUrl.orEmpty()
        editorSeverityIndex = item?.let { AnnouncementSeverities.indexOfFirst { s -> s.first == it.severity } }
            ?.coerceAtLeast(0) ?: 0
        sendPush = false
        editorOpen = true
    }

    fun save() {
        if (saving) return
        if (title.isBlank() || content.isBlank()) {
            Toast.makeText(context, "标题与内容不能为空", Toast.LENGTH_SHORT).show()
            return
        }
        saving = true
        scope.launch {
            val severity = AnnouncementSeverities[editorSeverityIndex].first
            val result = if (editing == null) {
                AppContainer.adminRepository.createAnnouncement(
                    title = title.trim(),
                    content = content.trim(),
                    severity = severity,
                    actionUrl = actionUrl.trim().ifBlank { null },
                    sendPush = sendPush,
                )
            } else {
                AppContainer.adminRepository.updateAnnouncement(
                    announcementId = editing!!.announcementId,
                    title = title.trim(),
                    content = content.trim(),
                    severity = severity,
                    actionUrl = actionUrl.trim().ifBlank { null },
                    clearActionUrl = actionUrl.isBlank() && !editing!!.actionUrl.isNullOrBlank(),
                )
            }
            when (result) {
                is ApiResult.Ok -> {
                    editorOpen = false
                    reloadKey += 1
                    Toast.makeText(context, if (editing == null) "公告已发布" else "公告已更新", Toast.LENGTH_SHORT).show()
                }
                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
            }
            saving = false
        }
    }

    Column(Modifier.fillMaxSize()) {
        PageHeader(
            title = "系统公告",
            subtitle = "平台公告",
            onBack = onBack,
            actions = {
                SystemButton(
                    text = "发布",
                    onClick = { openEditor(null) },
                    variant = ButtonVariant.PRIMARY,
                    size = ButtonSize.SM,
                )
            },
        )
        Column(Modifier.padding(horizontal = Spacing.lg)) {
            TabPills(
                options = listOf("全部") + AnnouncementSeverities.map { it.second },
                selectedIndex = severityIndex,
                onSelected = { severityIndex = it },
            )
            Spacer(Modifier.height(Spacing.sm2))
            SearchInput(
                value = query,
                onValueChange = { query = it },
                placeholder = "搜索标题 / 内容 / 发布人…",
            )
            Spacer(Modifier.height(Spacing.sm))
        }
        when {
            loading && items.isEmpty() -> LoadingState(message = "正在加载公告…")
            error != null && items.isEmpty() -> ErrorState(message = error ?: "加载失败", onRetry = { reloadKey += 1 })
            items.isEmpty() -> EmptyState(
                title = "还没有系统公告",
                description = "发布全站公告，用户会在通知中心与公告入口看到。",
            )
            filteredItems.isEmpty() -> EmptyState(
                title = "没有匹配的公告",
                description = "换个标题、正文或发布人关键词试试。",
            )
            else -> LazyColumn(
                modifier = Modifier.fillMaxSize(),
                contentPadding = androidx.compose.foundation.layout.PaddingValues(
                    horizontal = Spacing.lg,
                    vertical = Spacing.sm,
                ),
            ) {
                item {
                    Text(
                        text = "共 " + items.size + " 条" + if (query.isNotBlank()) " · 匹配 " + filteredItems.size + " 条" else "",
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textSecondary,
                        modifier = Modifier.padding(vertical = Spacing.sm, horizontal = Spacing.xs),
                    )
                }
                item {
                    ListGroup {
                        filteredItems.forEach { item ->
                            AnnouncementRow(
                                item = item,
                                onEdit = { openEditor(item) },
                                onDelete = { deleteTarget = item },
                            )
                        }
                    }
                }
                if (searchLoadingAll) {
                    item { InlineLoading(text = "正在加载全部公告用于搜索…") }
                } else if (query.isBlank() && cursor != null) {
                    item {
                        Box(Modifier.padding(vertical = Spacing.lg)) {
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
                } else {
                    item { Box(Modifier.padding(vertical = Spacing.lg)) { ListEnd() } }
                }
            }
        }
    }

    if (editorOpen) {
        AlertDialog(
            onDismissRequest = { if (!saving) editorOpen = false },
            title = { Text(if (editing == null) "发布系统公告" else "编辑系统公告") },
            text = {
                Column {
                    SystemInput(
                        value = title,
                        onValueChange = { title = it },
                        label = "标题",
                        placeholder = "公告标题…",
                        enabled = !saving,
                    )
                    Spacer(Modifier.height(Spacing.sm))
                    SystemTextArea(
                        value = content,
                        onValueChange = { content = it },
                        label = "内容",
                        placeholder = "公告正文…",
                        minHeight = Dimens.inputHeight * 3,
                        enabled = !saving,
                    )
                    Spacer(Modifier.height(Spacing.sm))
                    Text("级别", style = MaterialTheme.typography.titleSmall, color = colors.textPrimary)
                    Spacer(Modifier.height(Spacing.xs))
                    TabPills(
                        options = AnnouncementSeverities.map { it.second },
                        selectedIndex = editorSeverityIndex,
                        onSelected = { editorSeverityIndex = it },
                    )
                    Spacer(Modifier.height(Spacing.sm))
                    SystemInput(
                        value = actionUrl,
                        onValueChange = { actionUrl = it },
                        label = "跳转链接（可选）",
                        placeholder = "https://…",
                        enabled = !saving,
                    )
                    if (editing == null) {
                        Spacer(Modifier.height(Spacing.sm))
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text(
                                text = "同时推送给全站用户",
                                style = MaterialTheme.typography.bodySmall,
                                color = colors.textSecondary,
                                modifier = Modifier.weight(1f),
                            )
                            Switch(checked = sendPush, onCheckedChange = { sendPush = it }, enabled = !saving)
                        }
                    }
                }
            },
            confirmButton = {
                SystemButton(
                    text = if (saving) "保存中…" else "保存",
                    onClick = { save() },
                    variant = ButtonVariant.PRIMARY,
                    size = ButtonSize.SM,
                    enabled = !saving,
                    loading = saving,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { editorOpen = false },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !saving,
                )
            },
        )
    }

    deleteTarget?.let { target ->
        AlertDialog(
            onDismissRequest = { if (!deleting) deleteTarget = null },
            title = { Text("删除公告") },
            text = { Text("确定删除「" + target.title + "」吗？删除后用户将不再看到这条公告。") },
            confirmButton = {
                SystemButton(
                    text = if (deleting) "删除中…" else "删除",
                    onClick = {
                        if (deleting) return@SystemButton
                        deleting = true
                        scope.launch {
                            when (val result = AppContainer.adminRepository.deleteAnnouncement(target.announcementId)) {
                                is ApiResult.Ok -> {
                                    items = items.filterNot { it.announcementId == target.announcementId }
                                    deleteTarget = null
                                    Toast.makeText(context, "公告已删除", Toast.LENGTH_SHORT).show()
                                }
                                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                            }
                            deleting = false
                        }
                    },
                    variant = ButtonVariant.DANGER,
                    size = ButtonSize.SM,
                    enabled = !deleting,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { deleteTarget = null },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !deleting,
                )
            },
        )
    }
}

@Composable
private fun AnnouncementRow(
    item: AdminAnnouncementItem,
    onEdit: () -> Unit,
    onDelete: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    Column(Modifier.fillMaxWidth().padding(horizontal = Spacing.md, vertical = Spacing.sm2)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = "[" + severityLabel(item.severity) + "] " + item.title,
                style = MaterialTheme.typography.bodyLarge,
                color = colors.textPrimary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
            Text(
                text = TimeFormat.relative(item.createdAt),
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
            )
        }
        Spacer(Modifier.height(Spacing.xs))
        Text(
            text = item.content,
            style = MaterialTheme.typography.bodySmall,
            color = colors.textSecondary,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
        )
        Spacer(Modifier.height(Spacing.xs))
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = "发布人 " + item.createdBy.nickname.ifBlank { "管理员" } +
                    " · 阅读 " + item.viewCount +
                    if (item.pushSentAt != null) " · 已推送" else "",
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
                modifier = Modifier.weight(1f),
            )
            SystemButton(
                text = "编辑",
                onClick = onEdit,
                variant = ButtonVariant.SECONDARY,
                size = ButtonSize.SM,
            )
            Spacer(Modifier.width(Spacing.xs))
            SystemButton(
                text = "删除",
                onClick = onDelete,
                variant = ButtonVariant.SECONDARY,
                size = ButtonSize.SM,
            )
        }
        Spacer(Modifier.height(Spacing.xs))
        HorizontalDivider(color = colors.borderSubtle)
    }
}
