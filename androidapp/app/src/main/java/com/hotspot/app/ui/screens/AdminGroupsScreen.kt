package com.hotspot.app.ui.screens

import android.widget.Toast
import androidx.compose.foundation.clickable
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
import com.hotspot.app.data.model.AdminGroupItem
import com.hotspot.app.ui.components.Avatar
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
import com.hotspot.app.ui.components.SystemInput
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.TimeFormat
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * 管理后台 · 群组管理（仅平台管理员）：群名搜索 + 解散群组。
 * 解散走 admin/groups/dismiss：reason 仅进审计日志，不推送给群成员；成员会收到 GroupDismissed 事件。
 */
@Composable
fun AdminGroupsScreen(onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val colors = HotSpotTheme.colors
    var query by remember { mutableStateOf("") }
    var items by remember { mutableStateOf<List<AdminGroupItem>>(emptyList()) }
    var cursor by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var loadingMore by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var reloadKey by remember { mutableIntStateOf(0) }
    var dismissTarget by remember { mutableStateOf<AdminGroupItem?>(null) }
    var dismissReason by remember { mutableStateOf("") }
    var selectedGroupId by remember { mutableStateOf<String?>(null) }
    var dismissBusy by remember { mutableStateOf(false) }

    LaunchedEffect(query, reloadKey) {
        if (query.isNotBlank()) delay(350)
        loading = true
        error = null
        when (val result = AppContainer.adminRepository.searchGroups(query, null)) {
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
            when (val result = AppContainer.adminRepository.searchGroups(query, next)) {
                is ApiResult.Ok -> {
                    items = (items + result.data.items).distinctBy { it.group.groupId }
                    cursor = result.data.nextCursor
                }
                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
            }
            loadingMore = false
        }
    }

    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "群组管理", subtitle = "平台群组 · 点击行查看详情", onBack = onBack)
        Column(Modifier.padding(horizontal = Spacing.lg)) {
            SearchInput(value = query, onValueChange = { query = it }, placeholder = "搜索群名…")
            Spacer(Modifier.height(Spacing.sm2))
        }
        when {
            loading && items.isEmpty() -> LoadingState(message = "正在搜索群组…")
            error != null && items.isEmpty() -> ErrorState(message = error ?: "加载失败", onRetry = { reloadKey += 1 })
            items.isEmpty() -> EmptyState(
                title = "没有找到群组",
                description = "换个群名关键词试试。",
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
                        items.forEach { item ->
                            AdminGroupRow(
                                item = item,
                                onOpen = { selectedGroupId = item.group.groupId },
                                onDismiss = { dismissTarget = item; dismissReason = "" },
                            )
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

    dismissTarget?.let { target ->
        AlertDialog(
            onDismissRequest = { if (!dismissBusy) dismissTarget = null },
            title = { Text("解散群组") },
            text = {
                Column {
                    Text(
                        text = "解散「" + target.group.name.ifBlank { target.group.groupId } + "」后，所有成员会收到解散事件且无法继续访问，确定继续吗？",
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.textSecondary,
                    )
                    Spacer(Modifier.height(Spacing.md))
                    SystemInput(
                        value = dismissReason,
                        onValueChange = { dismissReason = it },
                        label = "解散原因（仅审计日志可见，0-500 字）",
                        placeholder = "填写处置原因，便于审计追溯…",
                        singleLine = false,
                        enabled = !dismissBusy,
                    )
                }
            },
            confirmButton = {
                SystemButton(
                    text = if (dismissBusy) "解散中…" else "解散",
                    onClick = {
                        if (dismissBusy) return@SystemButton
                        dismissBusy = true
                        scope.launch {
                            when (val result = AppContainer.adminRepository.dismissGroup(
                                groupId = target.group.groupId,
                                reason = dismissReason.ifBlank { null },
                            )) {
                                is ApiResult.Ok -> {
                                    items = items.filterNot { it.group.groupId == target.group.groupId }
                                    dismissTarget = null
                                    Toast.makeText(context, "群组已解散", Toast.LENGTH_SHORT).show()
                                }
                                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                            }
                            dismissBusy = false
                        }
                    },
                    variant = ButtonVariant.DANGER,
                    size = ButtonSize.SM,
                    enabled = !dismissBusy,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { dismissTarget = null },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !dismissBusy,
                )
            },
        )
    }

    selectedGroupId?.let { groupId ->
        AdminGroupDetailDialog(groupId = groupId, onDismiss = { selectedGroupId = null })
    }
}

@Composable
private fun AdminGroupRow(
    item: AdminGroupItem,
    onOpen: () -> Unit,
    onDismiss: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onOpen)
            .padding(horizontal = Spacing.md, vertical = Spacing.sm2),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Avatar(
            name = item.group.name,
            imageUrl = item.group.avatarUrl,
            size = Dimens.avatarSm,
        )
        Spacer(Modifier.width(Spacing.sm2))
        Column(Modifier.weight(1f)) {
            Text(
                text = item.group.name.ifBlank { "未命名群组" },
                style = MaterialTheme.typography.bodyLarge,
                color = colors.textPrimary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                text = item.group.membersCount.toString() + " 成员 · 群主 " +
                    item.owner.nickname.ifBlank { "—" } +
                    " · 创建于 " + TimeFormat.relative(item.createdAt),
                style = MaterialTheme.typography.bodySmall,
                color = colors.textSecondary,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
        }
        Spacer(Modifier.width(Spacing.sm))
        SystemButton(
            text = "解散",
            onClick = onDismiss,
            variant = ButtonVariant.SECONDARY,
            size = ButtonSize.SM,
        )
    }
}