package com.hotspot.app.ui.screens

import android.widget.Toast
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.SessionItem
import com.hotspot.app.ui.components.Badge
import com.hotspot.app.ui.components.BadgeTone
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemCard
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import kotlinx.coroutines.launch

/**
 * 登录设备管理（spec/api_spec_auth.md）：
 * - fetch_sessions 当前会话置顶，其余按 last_active_at 倒序；
 * - revoke_session 不允许撤销当前会话；
 * - revoke_other_sessions 一键退出其他设备。
 */
@Composable
fun SessionManagementScreen(onBack: () -> Unit) {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val repository = AppContainer.authRepository
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var items by remember { mutableStateOf<List<SessionItem>>(emptyList()) }
    var revoking by remember { mutableStateOf(false) }
    var revokeTarget by remember { mutableStateOf<SessionItem?>(null) }
    var showRevokeOthers by remember { mutableStateOf(false) }

    fun load() {
        scope.launch {
            loading = true
            error = null
            when (val result = repository.fetchSessions()) {
                is ApiResult.Ok -> items = result.data.items
                is ApiResult.Err -> error = result.error.userMessage
            }
            loading = false
        }
    }

    LaunchedEffect(Unit) { load() }

    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "登录设备", onBack = onBack)
        when {
            loading -> LoadingState(Modifier.weight(1f))
            error != null -> ErrorState(error.orEmpty(), onRetry = ::load, modifier = Modifier.weight(1f))
            else -> Column(
                modifier = Modifier
                    .weight(1f)
                    .verticalScroll(rememberScrollState())
                    .padding(horizontal = Spacing.lg),
            ) {
                Spacer(Modifier.height(Spacing.md))
                Text(
                    text = "当前账号共 ${items.size} 台设备在线",
                    style = MaterialTheme.typography.titleSmall,
                    color = colors.textPrimary,
                )
                Spacer(Modifier.height(Spacing.xs))
                Text(
                    text = "撤销后该设备需要重新登录；当前设备请使用「退出登录」。",
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
                if (items.any { !it.isCurrent }) {
                    Spacer(Modifier.height(Spacing.md))
                    SystemButton(
                        text = "退出其他所有设备",
                        onClick = { showRevokeOthers = true },
                        variant = ButtonVariant.DANGER,
                        size = ButtonSize.SM,
                    )
                }
                Spacer(Modifier.height(Spacing.md))
                items.forEach { item ->
                    SessionCard(
                        item = item,
                        onRevoke = { revokeTarget = item },
                    )
                    Spacer(Modifier.height(Spacing.sm))
                }
                Spacer(Modifier.height(Spacing.xxl))
            }
        }
    }

    revokeTarget?.let { target ->
        AlertDialog(
            onDismissRequest = { if (!revoking) revokeTarget = null },
            title = { Text("退出该设备？") },
            text = { Text("${sessionTitle(target)} 将立即失效，需要重新登录。") },
            confirmButton = {
                SystemButton(
                    text = if (revoking) "退出中…" else "退出设备",
                    onClick = {
                        if (revoking) return@SystemButton
                        scope.launch {
                            revoking = true
                            when (val result = repository.revokeSession(target.sessionId)) {
                                is ApiResult.Ok -> {
                                    revokeTarget = null
                                    Toast.makeText(context, "设备已退出", Toast.LENGTH_SHORT).show()
                                    load()
                                }
                                is ApiResult.Err -> error = result.error.userMessage
                            }
                            revoking = false
                        }
                    },
                    variant = ButtonVariant.DANGER,
                    size = ButtonSize.SM,
                    enabled = !revoking,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { if (!revoking) revokeTarget = null },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !revoking,
                )
            },
        )
    }

    if (showRevokeOthers) {
        AlertDialog(
            onDismissRequest = { if (!revoking) showRevokeOthers = false },
            title = { Text("退出其他所有设备？") },
            text = { Text("除当前设备外，所有登录会话都会立即失效，需要重新登录。") },
            confirmButton = {
                SystemButton(
                    text = if (revoking) "退出中…" else "全部退出",
                    onClick = {
                        if (revoking) return@SystemButton
                        scope.launch {
                            revoking = true
                            when (val result = repository.revokeOtherSessions()) {
                                is ApiResult.Ok -> {
                                    showRevokeOthers = false
                                    Toast.makeText(context, "其他设备已退出", Toast.LENGTH_SHORT).show()
                                    load()
                                }
                                is ApiResult.Err -> error = result.error.userMessage
                            }
                            revoking = false
                        }
                    },
                    variant = ButtonVariant.DANGER,
                    size = ButtonSize.SM,
                    enabled = !revoking,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { if (!revoking) showRevokeOthers = false },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !revoking,
                )
            },
        )
    }
}

@Composable
private fun SessionCard(item: SessionItem, onRevoke: () -> Unit) {
    val colors = HotSpotTheme.colors
    SystemCard(
        modifier = Modifier.fillMaxWidth(),
        contentPadding = PaddingValues(Spacing.md),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = sessionTitle(item),
                        style = MaterialTheme.typography.titleSmall,
                        color = colors.textPrimary,
                    )
                    if (item.isCurrent) {
                        Spacer(Modifier.padding(horizontal = Spacing.xs))
                        Badge(text = "当前设备", tone = BadgeTone.ONLINE)
                    }
                }
                val detail = listOfNotNull(
                    item.deviceType?.takeIf { it.isNotBlank() },
                    item.ipAddress?.takeIf { it.isNotBlank() },
                    item.ipLocation?.takeIf { it.isNotBlank() },
                ).joinToString(" · ")
                if (detail.isNotBlank()) {
                    Text(
                        text = detail,
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textSecondary,
                    )
                }
                Text(
                    text = "最近活跃 ${formatSessionTime(item.lastActiveAt)}",
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
            }
            if (!item.isCurrent) {
                SystemButton(
                    text = "退出",
                    onClick = onRevoke,
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                )
            }
        }
    }
}

private fun sessionTitle(item: SessionItem): String =
    item.deviceName?.takeIf { it.isNotBlank() }
        ?: item.deviceType?.takeIf { it.isNotBlank() }
        ?: "未知设备"

private fun formatSessionTime(raw: String): String {
    if (raw.isBlank()) return "—"
    return runCatching {
        DateTimeFormatter.ofPattern("MM-dd HH:mm")
            .withZone(ZoneId.systemDefault())
            .format(Instant.parse(raw))
    }.getOrDefault(raw)
}
