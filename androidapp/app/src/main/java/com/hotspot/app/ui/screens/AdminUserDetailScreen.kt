package com.hotspot.app.ui.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.navigationBarsPadding
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
import com.hotspot.app.data.model.AdminUserItem
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
 * 管理后台 · 用户详情（仅平台管理视角，只读）。
 * 数据来自 admin/users/open_the_door；IP/注册地等 PII 仅在管理员持有 admin.users.read_ip 时返回，
 * 权限不足展示「无权限或未记录」，不做本地猜测。封禁/解封仍在用户列表页操作。
 */
@Composable
fun AdminUserDetailScreen(userId: String, onBack: () -> Unit) {
    val colors = HotSpotTheme.colors
    var user by remember { mutableStateOf<AdminUserItem?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var reloadKey by remember { mutableIntStateOf(0) }

    LaunchedEffect(userId, reloadKey) {
        loading = true
        error = null
        when (val result = AppContainer.adminRepository.fetchUserDetail(userId)) {
            is ApiResult.Ok -> user = result.data
            is ApiResult.Err -> error = result.error.userMessage
        }
        loading = false
    }

    Surface(Modifier.fillMaxSize(), color = colors.page) {
        Column(Modifier.fillMaxSize().statusBarsPadding().navigationBarsPadding()) {
            PageHeader(
                title = "用户详情",
                subtitle = "平台管理视角",
                onBack = onBack,
            )
            when {
                loading && user == null -> LoadingState(message = "正在加载用户详情…")
                error != null && user == null -> ErrorState(
                    message = error ?: "加载失败",
                    onRetry = { reloadKey += 1 },
                )
                user == null -> LoadingState()
                else -> {
                    val data = user ?: return@Column
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
                                        name = data.user.nickname,
                                        imageUrl = data.user.avatarUrl,
                                        size = Dimens.avatarLg,
                                    )
                                    Column(Modifier.weight(1f).padding(start = Spacing.sm2)) {
                                        Text(
                                            text = data.user.nickname.ifBlank { "用户" },
                                            style = MaterialTheme.typography.titleMedium,
                                            color = colors.textPrimary,
                                            maxLines = 1,
                                            overflow = TextOverflow.Ellipsis,
                                        )
                                        Text(
                                            text = if (data.status == "banned") "已封禁" else "正常",
                                            style = MaterialTheme.typography.labelSmall,
                                            color = if (data.status == "banned") colors.danger else colors.textSecondary,
                                        )
                                        Text(
                                            text = data.bio?.takeIf { it.isNotBlank() } ?: "暂无简介",
                                            style = MaterialTheme.typography.bodySmall,
                                            color = colors.textSecondary,
                                            maxLines = 3,
                                            overflow = TextOverflow.Ellipsis,
                                        )
                                    }
                                }
                            }
                        }
                        item {
                            SectionLabel("账号")
                            ListGroup {
                                InfoRow("登录账号", data.loginId.ifBlank { "—" })
                                InfoDivider(colors)
                                InfoRow("用户 ID", data.user.userId)
                                InfoDivider(colors)
                                InfoRow("邮箱", data.email?.takeIf { it.isNotBlank() } ?: "未绑定")
                                InfoDivider(colors)
                                InfoRow("手机", data.phone?.takeIf { it.isNotBlank() } ?: "未绑定")
                                InfoDivider(colors)
                                InfoRow("隐私等级", privacyLevelLabel(data.privacyLevel))
                                InfoDivider(colors)
                                InfoRow("注册时间", TimeFormat.relative(data.createdAt))
                                InfoDivider(colors)
                                InfoRow("最近活跃", data.lastActiveAt?.let { TimeFormat.relative(it) } ?: "—")
                                if (data.status == "banned") {
                                    InfoDivider(colors)
                                    InfoRow(
                                        "封禁到期",
                                        data.bannedUntil?.let { TimeFormat.absolute(it) } ?: "永久",
                                        valueColor = colors.danger,
                                    )
                                }
                            }
                        }
                        item {
                            SectionLabel("安全信息（需 admin.users.read_ip）")
                            ListGroup {
                                InfoRow("注册 IP", data.registerIp?.takeIf { it.isNotBlank() } ?: "无权限或未记录")
                                InfoDivider(colors)
                                InfoRow("注册地点", data.registerLocation?.takeIf { it.isNotBlank() } ?: "无权限或未记录")
                                InfoDivider(colors)
                                InfoRow("最近登录 IP", data.lastLoginIp?.takeIf { it.isNotBlank() } ?: "无权限或未记录")
                                InfoDivider(colors)
                                InfoRow("最近登录地点", data.lastLoginLocation?.takeIf { it.isNotBlank() } ?: "无权限或未记录")
                            }
                        }
                        item {
                            Box(Modifier.fillMaxWidth().padding(vertical = Spacing.lg))
                        }
                    }
                }
            }
        }
    }
}

/** 隐私等级展示：契约值为 public/private，管理界面用产品术语，未知值原样兜底。 */
private fun privacyLevelLabel(level: String): String = when (level) {
    "public" -> "公开账号"
    "private" -> "私密账号"
    else -> level.ifBlank { "—" }
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
private fun InfoRow(label: String, value: String, valueColor: androidx.compose.ui.graphics.Color? = null) {
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
            color = valueColor ?: colors.textPrimary,
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

/** 全屏用户详情（列表行点击后使用）。 */
@Composable
fun AdminUserDetailDialog(userId: String, onDismiss: () -> Unit) {
    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false, decorFitsSystemWindows = false),
    ) {
        AdminUserDetailScreen(userId = userId, onBack = onDismiss)
    }
}
