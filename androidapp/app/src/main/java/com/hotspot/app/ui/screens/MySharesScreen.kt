package com.hotspot.app.ui.screens

import android.content.ClipData
import android.content.ClipboardManager
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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
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
import com.hotspot.app.data.model.ShareListItem
import com.hotspot.app.ui.components.Badge
import com.hotspot.app.ui.components.BadgeTone
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.InlineLoading
import com.hotspot.app.ui.components.ListEnd
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemCard
import com.hotspot.app.ui.components.TabPills
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import kotlinx.coroutines.launch

private const val SITE_BASE_URL = "https://chaoset.com"

private val SHARE_TABS = listOf("全部", "瞬间", "用户", "群组", "位置")
private val SHARE_TYPES = listOf(null, "moment", "user", "group", "location")

/** 我的分享：spec/share fetch_created + revoke，按类型筛选与 keyset 分页。 */
@Composable
fun MySharesScreen(onBack: () -> Unit) {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val repository = AppContainer.profileRepository
    var tabIndex by remember { mutableStateOf(0) }
    var items by remember { mutableStateOf<List<ShareListItem>>(emptyList()) }
    var cursor by remember { mutableStateOf<String?>(null) }
    var endReached by remember { mutableStateOf(false) }
    var loading by remember { mutableStateOf(true) }
    var loadingMore by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var revoking by remember { mutableStateOf(false) }
    var revokeTarget by remember { mutableStateOf<ShareListItem?>(null) }

    fun load(reset: Boolean) {
        if (loadingMore) return
        if (reset) {
            loading = true
            error = null
            items = emptyList()
            cursor = null
            endReached = false
        } else {
            loadingMore = true
        }
        scope.launch {
            when (val result = repository.createdShares(SHARE_TYPES[tabIndex], if (reset) null else cursor)) {
                is ApiResult.Ok -> {
                    items = if (reset) result.data.items else items + result.data.items
                    cursor = result.data.nextCursor
                    endReached = result.data.nextCursor == null
                }
                is ApiResult.Err -> if (reset) {
                    error = result.error.userMessage
                } else {
                    Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                }
            }
            loading = false
            loadingMore = false
        }
    }

    LaunchedEffect(tabIndex) { load(true) }

    fun copyLink(item: ShareListItem) {
        val url = SITE_BASE_URL + "/share/" + item.shareCode
        val clipboard = context.getSystemService(ClipboardManager::class.java)
        clipboard?.setPrimaryClip(ClipData.newPlainText("分享链接", url))
        Toast.makeText(context, "分享链接已复制", Toast.LENGTH_SHORT).show()
    }

    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "我的分享", onBack = onBack)
        TabPills(
            options = SHARE_TABS,
            selectedIndex = tabIndex,
            onSelected = { tabIndex = it },
            modifier = Modifier.padding(horizontal = Spacing.lg),
        )
        Spacer(Modifier.height(Spacing.md))
        when {
            loading -> LoadingState(Modifier.weight(1f))
            error != null -> ErrorState(error.orEmpty(), onRetry = { load(true) }, modifier = Modifier.weight(1f))
            items.isEmpty() -> EmptyState(
                title = if (tabIndex == 0) "还没有分享记录" else "该类型下没有分享",
                description = if (tabIndex == 0) "在瞬间、群组或个人主页分享后，链接会集中显示在这里。" else "换个类型看看。",
                modifier = Modifier.weight(1f),
            )
            else -> LazyColumn(
                modifier = Modifier.weight(1f),
                contentPadding = PaddingValues(horizontal = Spacing.lg, vertical = Spacing.xs),
                verticalArrangement = Arrangement.spacedBy(Spacing.sm),
            ) {
                items(items, key = { it.shareId }) { item ->
                    ShareCard(
                        item = item,
                        onCopy = { copyLink(item) },
                        onRevoke = { revokeTarget = item },
                    )
                }
                item {
                    if (loadingMore) {
                        InlineLoading()
                    } else if (!endReached) {
                        SystemButton(
                            text = "加载更多",
                            onClick = { load(false) },
                            variant = ButtonVariant.SECONDARY,
                            size = ButtonSize.SM,
                            modifier = Modifier.fillMaxWidth(),
                        )
                    } else {
                        ListEnd()
                    }
                }
            }
        }
    }

    revokeTarget?.let { target ->
        AlertDialog(
            onDismissRequest = { if (!revoking) revokeTarget = null },
            title = { Text("撤销这个分享？") },
            text = { Text("撤销后链接立即失效，已复制出去的链接也无法再打开。") },
            confirmButton = {
                SystemButton(
                    text = if (revoking) "撤销中…" else "撤销",
                    onClick = {
                        if (revoking) return@SystemButton
                        scope.launch {
                            revoking = true
                            when (val result = repository.revokeShare(target.shareId)) {
                                is ApiResult.Ok -> {
                                    items = items.map { if (it.shareId == target.shareId) it.copy(isRevoked = true, preview = null) else it }
                                    revokeTarget = null
                                    Toast.makeText(context, "分享已撤销", Toast.LENGTH_SHORT).show()
                                }
                                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
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
}

@Composable
private fun ShareCard(item: ShareListItem, onCopy: () -> Unit, onRevoke: () -> Unit) {
    val colors = HotSpotTheme.colors
    SystemCard(contentPadding = PaddingValues(Spacing.md)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Badge(text = shareTypeLabel(item.shareType), tone = BadgeTone.NEUTRAL)
            Spacer(Modifier.padding(horizontal = Spacing.xs))
            if (item.isRevoked) {
                Badge(text = "已撤销", tone = BadgeTone.WARNING)
            } else if (!item.expiresAt.isNullOrBlank()) {
                Badge(text = "有效期至 " + formatShareTime(item.expiresAt), tone = BadgeTone.OUTLINE)
            } else {
                Badge(text = "长期有效", tone = BadgeTone.OUTLINE)
            }
        }
        Spacer(Modifier.height(Spacing.xs))
        Text(
            text = item.preview?.title?.ifBlank { null } ?: if (item.isRevoked) "分享已撤销" else "分享内容暂不可见",
            style = MaterialTheme.typography.titleSmall,
            color = colors.textPrimary,
        )
        item.preview?.subtitle?.takeIf { it.isNotBlank() }?.let { subtitle ->
            Text(text = subtitle, style = MaterialTheme.typography.labelSmall, color = colors.textSecondary)
        }
        Text(
            text = "创建于 " + formatShareTime(item.createdAt),
            style = MaterialTheme.typography.labelSmall,
            color = colors.textSecondary,
        )
        Spacer(Modifier.height(Spacing.sm))
        Row(horizontalArrangement = Arrangement.spacedBy(Spacing.sm)) {
            if (!item.isRevoked) {
                SystemButton(
                    text = "复制链接",
                    onClick = onCopy,
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                )
                SystemButton(
                    text = "撤销",
                    onClick = onRevoke,
                    variant = ButtonVariant.DANGER,
                    size = ButtonSize.SM,
                )
            } else {
                Text(text = "链接已失效", style = MaterialTheme.typography.labelSmall, color = colors.textSecondary)
            }
        }
    }
}

private fun shareTypeLabel(type: String): String = when (type) {
    "moment" -> "瞬间"
    "user" -> "用户"
    "group" -> "群组"
    "location" -> "位置"
    else -> type
}

private fun formatShareTime(raw: String?): String {
    if (raw.isNullOrBlank()) return "—"
    return runCatching {
        DateTimeFormatter.ofPattern("MM-dd HH:mm")
            .withZone(ZoneId.systemDefault())
            .format(Instant.parse(raw))
    }.getOrDefault(raw)
}
