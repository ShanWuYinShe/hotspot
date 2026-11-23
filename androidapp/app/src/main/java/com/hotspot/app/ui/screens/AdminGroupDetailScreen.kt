package com.hotspot.app.ui.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.AdminGroupItem
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.ListGroup
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotColors
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.TimeFormat

/**
 * 管理后台 · 群组详情（仅平台管理视角，只读）。
 * 数据来自 admin/groups/open_the_door；解散群组仍在群组列表操作。
 */
@Composable
fun AdminGroupDetailScreen(groupId: String, onBack: () -> Unit) {
    val colors = HotSpotTheme.colors
    var group by remember { mutableStateOf<AdminGroupItem?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var reloadKey by remember { mutableIntStateOf(0) }

    LaunchedEffect(groupId, reloadKey) {
        loading = true
        error = null
        when (val result = AppContainer.adminRepository.fetchGroupDetail(groupId)) {
            is ApiResult.Ok -> group = result.data
            is ApiResult.Err -> error = result.error.userMessage
        }
        loading = false
    }

    Surface(Modifier.fillMaxSize(), color = colors.page) {
        Column(Modifier.fillMaxSize().statusBarsPadding().navigationBarsPadding()) {
            PageHeader(
                title = "群组详情",
                subtitle = "平台管理视角",
                onBack = onBack,
            )
            when {
                loading && group == null -> LoadingState(message = "正在加载群组详情…")
                error != null && group == null -> ErrorState(
                    message = error ?: "加载失败",
                    onRetry = { reloadKey += 1 },
                )
                group == null -> LoadingState()
                else -> {
                    val data = group ?: return@Column
                    LazyColumn(
                        modifier = Modifier.fillMaxSize(),
                        contentPadding = androidx.compose.foundation.layout.PaddingValues(
                            horizontal = Spacing.lg,
                            vertical = Spacing.sm,
                        ),
                        verticalArrangement = Arrangement.spacedBy(Spacing.md),
                    ) {
                        item {
                            ListGroup {
                                Row(
                                    modifier = Modifier.fillMaxWidth().padding(Spacing.md),
                                    verticalAlignment = Alignment.CenterVertically,
                                ) {
                                    Avatar(
                                        name = data.group.name,
                                        imageUrl = data.group.avatarUrl,
                                        size = Dimens.avatarLg,
                                    )
                                    Column(Modifier.weight(1f).padding(start = Spacing.sm2)) {
                                        Text(
                                            text = data.group.name.ifBlank { "未命名群组" },
                                            style = MaterialTheme.typography.titleMedium,
                                            color = colors.textPrimary,
                                            maxLines = 1,
                                            overflow = TextOverflow.Ellipsis,
                                        )
                                        Text(
                                            text = data.group.membersCount.toString() + " 位成员",
                                            style = MaterialTheme.typography.bodySmall,
                                            color = colors.textSecondary,
                                        )
                                        Text(
                                            text = data.group.groupId,
                                            style = MaterialTheme.typography.labelSmall,
                                            color = colors.textSecondary,
                                            maxLines = 1,
                                            overflow = TextOverflow.Ellipsis,
                                        )
                                    }
                                }
                            }
                        }
                        item {
                            SectionLabel("群主")
                            ListGroup {
                                Row(
                                    modifier = Modifier.fillMaxWidth().padding(Spacing.md),
                                    verticalAlignment = Alignment.CenterVertically,
                                ) {
                                    Avatar(
                                        name = data.owner.nickname,
                                        imageUrl = data.owner.avatarUrl,
                                        size = Dimens.avatarSm,
                                    )
                                    Spacer(Modifier.width(Spacing.sm2))
                                    Column(Modifier.weight(1f)) {
                                        Text(
                                            text = data.owner.nickname.ifBlank { "—" },
                                            style = MaterialTheme.typography.bodyLarge,
                                            color = colors.textPrimary,
                                        )
                                        Text(
                                            text = data.owner.userId.ifBlank { data.ownerId.orEmpty() },
                                            style = MaterialTheme.typography.labelSmall,
                                            color = colors.textSecondary,
                                        )
                                    }
                                }
                            }
                        }
                        item {
                            SectionLabel("基本信息")
                            ListGroup {
                                InfoRow("群组 ID", data.group.groupId)
                                InfoDivider(colors)
                                InfoRow("成员数", data.group.membersCount.toString())
                                InfoDivider(colors)
                                InfoRow("创建时间", TimeFormat.relative(data.createdAt))
                                InfoDivider(colors)
                                InfoRow("最近活跃", data.lastActivityAt?.let { TimeFormat.relative(it) } ?: "—")
                                InfoDivider(colors)
                                InfoRow(
                                    "标签",
                                    if (data.tags.isEmpty()) "无" else data.tags.joinToString("、"),
                                )
                            }
                        }
                        item {
                            SectionLabel("简介")
                            ListGroup {
                                Text(
                                    text = data.description?.takeIf { it.isNotBlank() } ?: "暂无简介",
                                    style = MaterialTheme.typography.bodyMedium,
                                    color = colors.textPrimary,
                                    modifier = Modifier.fillMaxWidth().padding(Spacing.md),
                                )
                            }
                        }
                        item { Box(Modifier.fillMaxWidth().padding(vertical = Spacing.lg)) }
                    }
                }
            }
        }
    }
}

@Composable
private fun SectionLabel(text: String) {
    val colors = HotSpotTheme.colors
    Text(
        text = text,
        style = MaterialTheme.typography.titleSmall,
        color = colors.textPrimary,
        modifier = Modifier.padding(start = Spacing.xs, end = Spacing.xs, top = Spacing.sm),
    )
}

@Composable
private fun InfoRow(label: String, value: String) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier.fillMaxWidth().padding(horizontal = Spacing.md, vertical = Spacing.sm2),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodySmall,
            color = colors.textSecondary,
            modifier = Modifier.width(96.dp),
        )
        Text(
            text = value,
            style = MaterialTheme.typography.bodyMedium,
            color = colors.textPrimary,
            maxLines = 3,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(1f),
        )
    }
}

@Composable
private fun InfoDivider(colors: HotSpotColors) {
    HorizontalDivider(color = colors.borderSubtle)
}

/** 全屏群组详情（列表行点击后使用）。 */
@Composable
fun AdminGroupDetailDialog(groupId: String, onDismiss: () -> Unit) {
    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false, decorFitsSystemWindows = false),
    ) {
        AdminGroupDetailScreen(groupId = groupId, onBack = onDismiss)
    }
}
