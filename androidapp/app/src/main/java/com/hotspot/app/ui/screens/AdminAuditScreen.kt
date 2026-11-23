package com.hotspot.app.ui.screens

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Icon
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
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Check
import androidx.compose.ui.text.font.FontWeight
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.AdminAuditLogItem
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.ListEnd
import com.hotspot.app.ui.components.ListGroup
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SearchInput
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.TabPills
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.TimeFormat
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import android.widget.Toast
import androidx.compose.ui.platform.LocalContext

/**
 * 管理后台 · 审计日志（仅平台管理员，admin.audit.read）：
 * 按目标类型筛选 + 搜索操作者/动作/对象/详情；审计契约无 keyword 参数，
 * 搜索时先自动加载完全部分页（最多 10 页）再本地筛选。
 */
@Composable
fun AdminAuditScreen(onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val colors = HotSpotTheme.colors
    var query by remember { mutableStateOf("") }
    var targetIndex by remember { mutableIntStateOf(0) }
    var items by remember { mutableStateOf<List<AdminAuditLogItem>>(emptyList()) }
    var cursor by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var loadingMore by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var reloadKey by remember { mutableIntStateOf(0) }
    var searchItems by remember { mutableStateOf<List<AdminAuditLogItem>>(emptyList()) }
    var searchLoaded by remember { mutableIntStateOf(0) }
    var searchLoading by remember { mutableStateOf(false) }
    var searchError by remember { mutableStateOf<String?>(null) }
    var searchReloadKey by remember { mutableIntStateOf(0) }
    var actionFilter by remember { mutableStateOf<String?>(null) }
    var showActionSheet by remember { mutableStateOf(false) }
    val searching = query.isNotBlank()
    val shownItems = if (searching) searchItems else items
    val targetType = TARGET_TYPES[targetIndex].second

    LaunchedEffect(targetType, actionFilter, reloadKey) {
        loading = true
        error = null
        when (val result = AppContainer.adminRepository.fetchAuditLogs(action = actionFilter, targetType = targetType)) {
            is ApiResult.Ok -> {
                items = result.data.items
                cursor = result.data.nextCursor
            }
            is ApiResult.Err -> error = result.error.userMessage
        }
        loading = false
    }

    // 搜索：契约无 keyword，自动加载完全部分页后本地匹配操作者/动作/对象/详情。
    LaunchedEffect(query, targetType, actionFilter, searchReloadKey) {
        val q = query.trim()
        if (q.isEmpty()) {
            searchItems = emptyList()
            searchLoaded = 0
            searchLoading = false
            searchError = null
            return@LaunchedEffect
        }
        delay(350)
        searchLoading = true
        searchError = null
        val all = mutableListOf<AdminAuditLogItem>()
        var next: String? = null
        for (page in 0 until 10) {
            when (val result = AppContainer.adminRepository.fetchAuditLogs(action = actionFilter, targetType = targetType, cursor = next)) {
                is ApiResult.Ok -> {
                    all += result.data.items
                    next = result.data.nextCursor
                }
                is ApiResult.Err -> {
                    searchError = result.error.userMessage
                    next = null
                    break
                }
            }
            if (next == null) break
        }
        searchLoaded = all.size
        searchItems = all.filter { auditMatches(it, q) }
        searchLoading = false
    }

    fun loadMore() {
        val next = cursor ?: return
        if (loadingMore || loading || searching) return
        loadingMore = true
        scope.launch {
            when (val result = AppContainer.adminRepository.fetchAuditLogs(targetType = targetType, cursor = next)) {
                is ApiResult.Ok -> {
                    items = (items + result.data.items).distinctBy { it.logId }
                    cursor = result.data.nextCursor
                }
                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
            }
            loadingMore = false
        }
    }

    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "审计日志", subtitle = "平台操作留痕", onBack = onBack)
        Column(Modifier.padding(horizontal = Spacing.lg)) {
            SearchInput(value = query, onValueChange = { query = it }, placeholder = "搜索操作者 / 动作 / 对象…")
            Spacer(Modifier.height(Spacing.sm2))
            TabPills(
                options = TARGET_TYPES.map { it.first },
                selectedIndex = targetIndex,
                onSelected = { targetIndex = it },
            )
            Spacer(Modifier.height(Spacing.sm))
            SystemButton(
                text = "动作：" + (actionFilter?.let { AUDIT_ACTION_LABELS[it] } ?: "全部"),
                onClick = { showActionSheet = true },
                variant = ButtonVariant.SECONDARY,
                size = ButtonSize.SM,
            )
            if (searching && !searchLoading && searchError == null) {
                Spacer(Modifier.height(Spacing.sm))
                Text(
                    text = "已加载 " + searchLoaded + " 条 · 匹配 " + searchItems.size + " 条",
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
            }
            Spacer(Modifier.height(Spacing.sm))
        }
        when {
            loading && items.isEmpty() && !searching -> LoadingState(message = "正在加载审计日志…")
            searchLoading && searchItems.isEmpty() -> LoadingState(message = "正在搜索审计日志…")
            searchError != null -> ErrorState(message = searchError ?: "搜索失败", onRetry = { searchReloadKey += 1 })
            error != null && items.isEmpty() && !searching -> ErrorState(message = error ?: "加载失败", onRetry = { reloadKey += 1 })
            shownItems.isEmpty() -> EmptyState(
                title = if (searching) "没有匹配的审计日志" else "暂无审计日志",
                description = if (searching) "换个操作者/动作/对象关键词试试，或清空搜索词。" else "平台管理操作会在这里留痕。",
            )
            else -> LazyColumn(
                modifier = Modifier.fillMaxSize(),
                contentPadding = androidx.compose.foundation.layout.PaddingValues(
                    horizontal = Spacing.lg,
                    vertical = Spacing.sm,
                ),
            ) {
                item {
                    ListGroup {
                        shownItems.forEachIndexed { index, item ->
                            AuditRow(item = item)
                            if (index != shownItems.lastIndex) HorizontalDivider(color = colors.borderSubtle)
                        }
                    }
                }
                if (!searching && cursor != null) {
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

    if (showActionSheet) {
        AuditActionSheet(
            selected = actionFilter,
            onSelect = {
                actionFilter = it
                showActionSheet = false
            },
            onDismiss = { showActionSheet = false },
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun AuditActionSheet(
    selected: String?,
    onSelect: (String?) -> Unit,
    onDismiss: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        containerColor = colors.surface,
        shape = AppShapes.sheet,
    ) {
        Column(
            Modifier
                .fillMaxWidth()
                .navigationBarsPadding()
                .padding(start = Spacing.lg, end = Spacing.lg, bottom = Spacing.lg),
        ) {
            Text(
                text = "按动作筛选",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
                color = colors.textPrimary,
                modifier = Modifier.padding(vertical = Spacing.sm),
            )
            val options = listOf<Pair<String?, String>>(null to "全部动作") +
                AUDIT_ACTION_LABELS.entries.map { it.key to it.value }
            options.forEachIndexed { index, (value, label) ->
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .heightIn(min = Dimens.minTouchTarget)
                        .clickable { onSelect(value) }
                        .padding(horizontal = Spacing.xs, vertical = Spacing.md),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        text = label,
                        style = MaterialTheme.typography.bodyLarge,
                        color = if (value == selected) colors.accent else colors.textPrimary,
                        modifier = Modifier.weight(1f),
                    )
                    if (value == selected) {
                        Icon(
                            imageVector = Icons.Filled.Check,
                            contentDescription = "已选",
                            tint = colors.accent,
                            modifier = Modifier.width(18.dp),
                        )
                    }
                }
                if (index != options.lastIndex) HorizontalDivider(color = colors.borderSubtle)
            }
        }
    }
}

@Composable
private fun AuditRow(item: AdminAuditLogItem) {
    val colors = HotSpotTheme.colors
    val actionLabel = AUDIT_ACTION_LABELS[item.action] ?: item.action.ifBlank { "—" }
    val targetLabel = AUDIT_TARGET_LABELS[item.targetType] ?: item.targetType.ifBlank { "—" }
    Column(Modifier.fillMaxWidth().padding(horizontal = Spacing.md, vertical = Spacing.sm2)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = actionLabel,
                style = MaterialTheme.typography.bodyLarge,
                color = colors.textPrimary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f, fill = false),
            )
            Spacer(Modifier.width(Spacing.sm))
            Text(text = targetLabel, style = MaterialTheme.typography.labelSmall, color = colors.accent)
        }
        Spacer(Modifier.height(2.dp))
        Text(
            text = (item.operator.nickname.ifBlank { "系统" }) + " · " + TimeFormat.relative(item.createdAt),
            style = MaterialTheme.typography.labelSmall,
            color = colors.textSecondary,
        )
        val target = item.targetName?.takeIf { it.isNotBlank() } ?: item.targetId
        if (target.isNotBlank()) {
            Text(
                text = targetLabel + "：" + target,
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
        val detail = formatAuditDetails(item.details)
        if (detail.isNotBlank()) {
            Text(
                text = detail,
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
        }
        val network = listOfNotNull(
            item.ipLocation?.takeIf { it.isNotBlank() }?.let { "归属 " + it },
            item.ipAddress?.takeIf { it.isNotBlank() },
            item.deviceSummary?.takeIf { it.isNotBlank() },
        ).joinToString(" · ")
        if (network.isNotBlank()) {
            Text(
                text = network,
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}

private fun auditMatches(item: AdminAuditLogItem, q: String): Boolean {
    val hay = listOf(
        item.operator.nickname,
        item.operator.userId,
        item.action,
        AUDIT_ACTION_LABELS[item.action] ?: "",
        item.targetType,
        AUDIT_TARGET_LABELS[item.targetType] ?: "",
        item.targetId,
        item.targetName ?: "",
        formatAuditDetails(item.details),
        item.ipLocation ?: "",
        item.ipAddress ?: "",
        item.deviceSummary ?: "",
    ).joinToString(" ").lowercase()
    return hay.contains(q.lowercase())
}

private fun formatAuditDetails(details: JsonElement?): String {
    if (details == null || details is kotlinx.serialization.json.JsonNull) return ""
    return when (details) {
        is JsonObject -> details.entries.joinToString(" · ") { (k, v) -> k + ": " + formatJsonValue(v) }
        else -> formatJsonValue(details)
    }
}

private fun formatJsonValue(value: JsonElement): String = when (value) {
    is JsonPrimitive -> value.content
    is JsonArray -> value.joinToString(", ") { formatJsonValue(it) }
    is JsonObject -> value.entries.joinToString(", ") { (k, v) -> k + ": " + formatJsonValue(v) }
    else -> value.toString()
}

private val TARGET_TYPES = listOf(
    "全部" to null,
    "用户" to "user",
    "群组" to "group",
    "公告" to "announcement",
    "角色" to "role",
    "管理员" to "admin",
    "举报" to "report",
    "标签" to "tag",
    "配置" to "config",
)

private val AUDIT_TARGET_LABELS = mapOf(
    "user" to "用户",
    "group" to "群组",
    "announcement" to "公告",
    "role" to "角色",
    "admin" to "管理员",
    "report" to "举报",
    "tag" to "标签",
    "config" to "配置",
)

private val AUDIT_ACTION_LABELS = mapOf(
    "user.ban" to "封禁用户",
    "user.unban" to "解封用户",
    "group.dismiss" to "解散群组",
    "group.member_mute" to "禁言成员",
    "group.member_ban" to "封禁成员",
    "group.member_remove" to "移除成员",
    "announcement.create" to "发布公告",
    "announcement.update" to "更新公告",
    "announcement.delete" to "删除公告",
    "role.create" to "创建角色",
    "role.update" to "更新角色",
    "role.delete" to "删除角色",
    "role.set_permissions" to "设置角色权限",
    "admin.award" to "授权管理员",
    "admin.revoke" to "撤销管理员",
    "report.judge" to "裁决举报",
    "tag.set_ban" to "标签屏蔽",
    "config.update" to "更新配置",
)