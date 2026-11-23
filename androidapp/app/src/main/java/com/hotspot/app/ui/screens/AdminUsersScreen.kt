package com.hotspot.app.ui.screens

import android.widget.Toast
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
import com.hotspot.app.data.model.AdminUserItem
import com.hotspot.app.ui.components.Avatar
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
import com.hotspot.app.ui.components.TabPills
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.util.TimeFormat
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * 管理后台 · 用户管理（仅平台管理员）：昵称搜索 + 账号状态筛选 + 封禁/解封。
 * 封禁走 admin/users/set_ban（active=true，可选 1/7/30 天或永久）；解封 active=false。
 * 权限与「禁止封禁自身/管理员保护」由后端校验，客户端只做二次确认。
 */
@Composable
fun AdminUsersScreen(onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val colors = HotSpotTheme.colors
    var query by remember { mutableStateOf("") }
    var statusIndex by remember { mutableIntStateOf(0) }
    var items by remember { mutableStateOf<List<AdminUserItem>>(emptyList()) }
    var cursor by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var loadingMore by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var reloadKey by remember { mutableIntStateOf(0) }
    var banTarget by remember { mutableStateOf<AdminUserItem?>(null) }
    var banReason by remember { mutableStateOf("") }
    var banDurationIndex by remember { mutableIntStateOf(0) }
    var banBusy by remember { mutableStateOf(false) }
    var unbanTarget by remember { mutableStateOf<AdminUserItem?>(null) }
    var unbanBusy by remember { mutableStateOf(false) }
    var selectedUserId by remember { mutableStateOf<String?>(null) }

    val statusFilter = when (statusIndex) {
        1 -> "normal"
        2 -> "banned"
        else -> null
    }

    LaunchedEffect(query, statusIndex, reloadKey) {
        if (query.isNotBlank()) delay(350)
        loading = true
        error = null
        when (val result = AppContainer.adminRepository.searchUsers(query, statusFilter, null)) {
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
            when (val result = AppContainer.adminRepository.searchUsers(query, statusFilter, next)) {
                is ApiResult.Ok -> {
                    items = (items + result.data.items).distinctBy { it.user.userId }
                    cursor = result.data.nextCursor
                }
                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
            }
            loadingMore = false
        }
    }

    fun reload() {
        reloadKey += 1
    }

    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "用户管理", subtitle = "平台用户 · 点击行查看详情", onBack = onBack)
        Column(Modifier.padding(horizontal = Spacing.lg)) {
            SearchInput(value = query, onValueChange = { query = it }, placeholder = "搜索昵称…")
            Spacer(Modifier.height(Spacing.sm2))
            TabPills(
                options = listOf("全部", "正常", "已封禁"),
                selectedIndex = statusIndex,
                onSelected = { statusIndex = it },
            )
            Spacer(Modifier.height(Spacing.sm))
        }
        when {
            loading && items.isEmpty() -> LoadingState(message = "正在搜索用户…")
            error != null && items.isEmpty() -> ErrorState(message = error ?: "加载失败", onRetry = { reload() })
            items.isEmpty() -> EmptyState(
                title = "没有找到用户",
                description = "换个昵称关键词试试，或切换账号状态筛选。",
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
                            AdminUserRow(
                                item = item,
                                onOpen = { selectedUserId = item.user.userId },
                                onBan = { banTarget = item; banReason = ""; banDurationIndex = 0 },
                                onUnban = { unbanTarget = item },
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

    banTarget?.let { target ->
        AlertDialog(
            onDismissRequest = { if (!banBusy) banTarget = null },
            title = { Text("封禁用户") },
            text = {
                Column {
                    Text(
                        text = "封禁「" + target.user.nickname.ifBlank { target.user.userId } + "」会立即断开其会话，确定继续吗？",
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.textSecondary,
                    )
                    Spacer(Modifier.height(Spacing.md))
                    Text("封禁时长", style = MaterialTheme.typography.titleSmall, color = colors.textPrimary)
                    Spacer(Modifier.height(Spacing.xs))
                    TabPills(
                        options = listOf("永久", "1 天", "7 天", "30 天"),
                        selectedIndex = banDurationIndex,
                        onSelected = { banDurationIndex = it },
                    )
                    Spacer(Modifier.height(Spacing.md))
                    SystemInput(
                        value = banReason,
                        onValueChange = { banReason = it },
                        label = "封禁原因（可选，0-500 字）",
                        placeholder = "填写原因便于审计追溯…",
                        singleLine = false,
                        enabled = !banBusy,
                    )
                }
            },
            confirmButton = {
                SystemButton(
                    text = if (banBusy) "封禁中…" else "封禁",
                    onClick = {
                        if (banBusy) return@SystemButton
                        banBusy = true
                        scope.launch {
                            val days = when (banDurationIndex) {
                                1 -> 1
                                2 -> 7
                                3 -> 30
                                else -> null
                            }
                            when (val result = AppContainer.adminRepository.setUserBan(
                                userId = target.user.userId,
                                active = true,
                                reason = banReason.ifBlank { null },
                                durationDays = days,
                            )) {
                                is ApiResult.Ok -> {
                                    banTarget = null
                                    reload()
                                    Toast.makeText(context, "已封禁该用户", Toast.LENGTH_SHORT).show()
                                }
                                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                            }
                            banBusy = false
                        }
                    },
                    variant = ButtonVariant.DANGER,
                    size = ButtonSize.SM,
                    enabled = !banBusy,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { banTarget = null },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !banBusy,
                )
            },
        )
    }

    unbanTarget?.let { target ->
        AlertDialog(
            onDismissRequest = { if (!unbanBusy) unbanTarget = null },
            title = { Text("解除封禁") },
            text = { Text("确定解除「" + target.user.nickname.ifBlank { "该用户" } + "」的封禁吗？解除后可正常登录。") },
            confirmButton = {
                SystemButton(
                    text = if (unbanBusy) "解封中…" else "解封",
                    onClick = {
                        if (unbanBusy) return@SystemButton
                        unbanBusy = true
                        scope.launch {
                            when (val result = AppContainer.adminRepository.setUserBan(
                                userId = target.user.userId,
                                active = false,
                                reason = null,
                                durationDays = null,
                            )) {
                                is ApiResult.Ok -> {
                                    unbanTarget = null
                                    reload()
                                    Toast.makeText(context, "已解除封禁", Toast.LENGTH_SHORT).show()
                                }
                                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                            }
                            unbanBusy = false
                        }
                    },
                    variant = ButtonVariant.PRIMARY,
                    size = ButtonSize.SM,
                    enabled = !unbanBusy,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { unbanTarget = null },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !unbanBusy,
                )
            },
        )
    }

    selectedUserId?.let { userId ->
        AdminUserDetailDialog(userId = userId, onDismiss = { selectedUserId = null })
    }
}

@Composable
private fun AdminUserRow(
    item: AdminUserItem,
    onOpen: () -> Unit,
    onBan: () -> Unit,
    onUnban: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    val banned = item.status == "banned"
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onOpen)
            .padding(horizontal = Spacing.md, vertical = Spacing.sm2),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Avatar(name = item.user.nickname, imageUrl = item.user.avatarUrl, size = Dimens.avatarSm)
        Spacer(Modifier.width(Spacing.sm2))
        Column(modifier = Modifier.weight(1f)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = item.user.nickname.ifBlank { "用户" },
                    style = MaterialTheme.typography.bodyLarge,
                    color = colors.textPrimary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f, fill = false),
                )
                Spacer(Modifier.width(Spacing.xs))
                Text(
                    text = if (banned) "已封禁" else "正常",
                    style = MaterialTheme.typography.labelSmall,
                    color = if (banned) colors.danger else colors.textSecondary,
                )
            }
            Text(
                text = "账号 " + item.loginId.ifBlank { "—" } +
                    " · 注册于 " + TimeFormat.relative(item.createdAt) +
                    if (banned && item.bannedUntil != null) " · 到期 " + TimeFormat.relative(item.bannedUntil) else "",
                style = MaterialTheme.typography.bodySmall,
                color = colors.textSecondary,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
        }
        Spacer(Modifier.width(Spacing.sm))
        if (banned) {
            SystemButton(
                text = "解封",
                onClick = onUnban,
                variant = ButtonVariant.SECONDARY,
                size = ButtonSize.SM,
            )
        } else {
            SystemButton(
                text = "封禁",
                onClick = onBan,
                variant = ButtonVariant.SECONDARY,
                size = ButtonSize.SM,
            )
        }
    }
}