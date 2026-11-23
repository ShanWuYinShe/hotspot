package com.hotspot.app.ui.screens

import android.widget.Toast
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.BlockedUserItem
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.ListEnd
import com.hotspot.app.ui.components.ListGroup
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.TimeFormat
import kotlinx.coroutines.launch

/**
 * 屏蔽名单：users/fetch_blocked 列表 + 解除屏蔽（users/set_block active=false）。
 * 拉黑时服务端会自动解除双方关注；解除拉黑不恢复关注关系，需要重新关注。
 */
@Composable
fun BlockedUsersScreen(
    onBack: () -> Unit,
    onOpenUser: (String) -> Unit,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    var items by remember { mutableStateOf<List<BlockedUserItem>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var retryKey by remember { mutableStateOf(0) }
    var nextCursor by remember { mutableStateOf<String?>(null) }
    var loadingMore by remember { mutableStateOf(false) }
    var endReached by remember { mutableStateOf(false) }
    var closingIds by remember { mutableStateOf<Set<String>>(emptySet()) }

    LaunchedEffect(retryKey) {
        loading = true
        error = null
        when (val result = AppContainer.profileRepository.fetchBlocked(cursor = null)) {
            is ApiResult.Ok -> {
                items = result.data.items
                nextCursor = result.data.nextCursor
                endReached = result.data.nextCursor == null
            }
            is ApiResult.Err -> error = result.error.userMessage
        }
        loading = false
    }

    fun loadMore() {
        val cursor = nextCursor ?: return
        if (loading || loadingMore) return
        loadingMore = true
        scope.launch {
            when (val result = AppContainer.profileRepository.fetchBlocked(cursor)) {
                is ApiResult.Ok -> {
                    items = (items + result.data.items).distinctBy { it.user.userId }
                    nextCursor = result.data.nextCursor
                    endReached = result.data.nextCursor == null
                }
                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
            }
            loadingMore = false
        }
    }

    fun unblock(userId: String) {
        if (userId in closingIds) return
        closingIds = closingIds + userId
        scope.launch {
            when (val result = AppContainer.profileRepository.setBlock(userId, blocked = false)) {
                is ApiResult.Ok -> items = items.filterNot { it.user.userId == userId }
                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
            }
            closingIds = closingIds - userId
        }
    }

    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "屏蔽名单", onBack = onBack)
        when {
            loading -> LoadingState(message = "正在加载屏蔽名单…")
            error != null && items.isEmpty() -> ErrorState(
                message = error ?: "加载失败",
                onRetry = { retryKey += 1 },
            )
            items.isEmpty() -> EmptyState(
                title = "还没有屏蔽任何人",
                description = "在他人主页选择「屏蔽」后，会出现在这里；解除后可以重新互动。",
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
                        text = "解除屏蔽不会自动恢复关注关系，需要重新关注。",
                        style = MaterialTheme.typography.labelSmall,
                        color = HotSpotTheme.colors.textSecondary,
                        modifier = Modifier.padding(vertical = Spacing.sm, horizontal = Spacing.xs),
                    )
                }
                item {
                    ListGroup {
                        items.forEach { item -> BlockedUserRow(item, unblocking = item.user.userId in closingIds, onOpen = { onOpenUser(item.user.userId) }, onUnblock = { unblock(item.user.userId) }) }
                    }
                }
                if (!endReached) {
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
}

@Composable
private fun BlockedUserRow(
    item: BlockedUserItem,
    unblocking: Boolean,
    onOpen: () -> Unit,
    onUnblock: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onOpen)
            .padding(horizontal = Spacing.md, vertical = Spacing.sm2),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Avatar(name = item.user.nickname, imageUrl = item.user.avatarUrl, size = Dimens.avatarSm)
        Spacer(Modifier.width(Spacing.sm2))
        Column(Modifier.weight(1f)) {
            Text(
                text = item.user.nickname.ifBlank { "用户" },
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Medium,
                color = colors.textPrimary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                text = buildString {
                    append(if (item.blockedAt.isNotBlank()) "屏蔽于 " + TimeFormat.relative(item.blockedAt) else "已屏蔽")
                    if (!item.reason.isNullOrBlank()) append(" · 原因：" + item.reason)
                },
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
        SystemButton(
            text = if (unblocking) "解除中…" else "解除屏蔽",
            onClick = onUnblock,
            variant = ButtonVariant.SECONDARY,
            size = ButtonSize.SM,
            enabled = !unblocking,
        )
    }
}
