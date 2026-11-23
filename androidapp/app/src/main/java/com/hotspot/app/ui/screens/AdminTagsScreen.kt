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
import com.hotspot.app.data.model.AdminTagItem
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
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.TimeFormat
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * 管理后台 · 标签管理（仅平台管理员）：搜索标签、按使用次数排序、软屏蔽/解屏蔽。
 * 已屏蔽标签仍在列表中（管理端唯一入口），历史引用保留；物理删除 hard_delete 不在 UI 暴露（高危不可逆）。
 */
@Composable
fun AdminTagsScreen(onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val colors = HotSpotTheme.colors
    var query by remember { mutableStateOf("") }
    var items by remember { mutableStateOf<List<AdminTagItem>>(emptyList()) }
    var cursor by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var loadingMore by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var reloadKey by remember { mutableIntStateOf(0) }
    var actingTag by remember { mutableStateOf<AdminTagItem?>(null) }
    var acting by remember { mutableStateOf(false) }

    LaunchedEffect(query, reloadKey) {
        if (query.isNotBlank()) delay(350)
        loading = true
        error = null
        when (val result = AppContainer.adminRepository.searchTags(query, null)) {
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
            when (val result = AppContainer.adminRepository.searchTags(query, next)) {
                is ApiResult.Ok -> {
                    items = (items + result.data.items).distinctBy { it.tagId.ifBlank { it.name } }
                    cursor = result.data.nextCursor
                }
                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
            }
            loadingMore = false
        }
    }

    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "标签管理", subtitle = "平台标签", onBack = onBack)
        Column(Modifier.padding(horizontal = Spacing.lg)) {
            SearchInput(value = query, onValueChange = { query = it }, placeholder = "搜索标签名…")
            Spacer(Modifier.height(Spacing.sm2))
        }
        when {
            loading && items.isEmpty() -> LoadingState(message = "正在搜索标签…")
            error != null && items.isEmpty() -> ErrorState(message = error ?: "加载失败", onRetry = { reloadKey += 1 })
            items.isEmpty() -> EmptyState(
                title = "没有找到标签",
                description = "换个关键词试试；已屏蔽标签也会出现在结果中。",
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
                        items.forEachIndexed { index, item ->
                            TagRow(item = item, onSetBan = { actingTag = item })
                            if (index != items.lastIndex) {
                                HorizontalDivider(color = colors.borderSubtle)
                            }
                        }
                    }
                }
                if (cursor != null) {
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

    actingTag?.let { target ->
        val willBlock = !target.isBlocked
        AlertDialog(
            onDismissRequest = { if (!acting) actingTag = null },
            title = { Text(if (willBlock) "屏蔽标签" else "解除屏蔽") },
            text = {
                Text(
                    if (willBlock) {
                        "屏蔽「" + target.name + "」后，该标签不能再被新内容使用或搜索命中；已有历史引用保留不变。"
                    } else {
                        "解除屏蔽后，「" + target.name + "」恢复可创建与可搜索。"
                    },
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.textSecondary,
                )
            },
            confirmButton = {
                SystemButton(
                    text = if (acting) "处理中…" else if (willBlock) "屏蔽" else "解除屏蔽",
                    onClick = {
                        if (acting) return@SystemButton
                        acting = true
                        scope.launch {
                            when (val result = AppContainer.adminRepository.setTagBan(target.name, active = willBlock)) {
                                is ApiResult.Ok -> {
                                    items = items.map { if (it.name == target.name) it.copy(isBlocked = willBlock) else it }
                                    actingTag = null
                                    Toast.makeText(context, if (willBlock) "标签已屏蔽" else "已解除屏蔽", Toast.LENGTH_SHORT).show()
                                }
                                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                            }
                            acting = false
                        }
                    },
                    variant = if (willBlock) ButtonVariant.DANGER else ButtonVariant.PRIMARY,
                    size = ButtonSize.SM,
                    enabled = !acting,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { actingTag = null },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !acting,
                )
            },
        )
    }
}

@Composable
private fun TagRow(item: AdminTagItem, onSetBan: () -> Unit) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier.fillMaxWidth().padding(horizontal = Spacing.md, vertical = Spacing.sm2),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = item.name,
                    style = MaterialTheme.typography.bodyLarge,
                    color = if (item.isBlocked) colors.textSecondary else colors.textPrimary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f, fill = false),
                )
                if (item.isBlocked) {
                    Spacer(Modifier.width(Spacing.sm))
                    Text(
                        text = "已屏蔽",
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.danger,
                    )
                }
            }
            Text(
                text = "使用 " + item.usageCount + " 次 · 创建于 " + TimeFormat.relative(item.createdAt),
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
            )
        }
        Spacer(Modifier.width(Spacing.sm))
        SystemButton(
            text = if (item.isBlocked) "解除屏蔽" else "屏蔽",
            onClick = onSetBan,
            variant = ButtonVariant.SECONDARY,
            size = ButtonSize.SM,
        )
    }
}
