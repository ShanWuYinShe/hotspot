package com.hotspot.app.ui.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Settings
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.AccountData
import com.hotspot.app.data.model.UserStats
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.Badge
import com.hotspot.app.ui.components.BadgeTone
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.HeaderIconButton
import com.hotspot.app.ui.components.ListGroup
import com.hotspot.app.ui.components.ListRow
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SectionTitle
import com.hotspot.app.ui.components.SystemCard
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import kotlinx.coroutines.launch

/**
 * 我的：资料卡（头像/昵称/简介/统计）+ 常用入口。
 * 资料数据走 /api/v1/users/account；未登录态由根导航统一拦截到登录页。
 */
@Composable
fun MeScreen(
    onOpenSettings: () -> Unit,
    onOpenNotify: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    val scope = rememberCoroutineScope()
    var account by remember { mutableStateOf<AccountData?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }

    suspend fun load() {
        loading = account == null
        error = null
        when (val result = AppContainer.profileRepository.account()) {
            is ApiResult.Ok -> account = result.data
            is ApiResult.Err -> error = result.error.userMessage
        }
        loading = false
    }

    LaunchedEffect(Unit) { load() }

    Column(Modifier.fillMaxSize()) {
        PageHeader(
            title = "我的",
            actions = {
                HeaderIconButton(
                    icon = Icons.Filled.Notifications,
                    contentDescription = "通知",
                    onClick = onOpenNotify,
                )
                HeaderIconButton(
                    icon = Icons.Filled.Settings,
                    contentDescription = "设置",
                    onClick = onOpenSettings,
                )
            },
        )
        when {
            loading -> LoadingState(message = "正在加载资料…")
            error != null && account == null -> ErrorState(
                message = error ?: "加载失败",
                onRetry = { scope.launch { load() } },
            )
            account == null -> EmptyState(title = "资料暂不可用", description = "请稍后重试")
            else -> {
                val data = account ?: return@Column
                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .verticalScroll(rememberScrollState())
                        .padding(horizontal = Spacing.lg),
                ) {
                    SystemCard {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Avatar(
                                name = data.nickname,
                                imageUrl = data.avatarUrl,
                                size = Dimens.avatarLg,
                            )
                            Spacer(Modifier.width(Spacing.md))
                            Column(Modifier.weight(1f)) {
                                Row(verticalAlignment = Alignment.CenterVertically) {
                                    Text(
                                        text = data.nickname.ifBlank { "未设置昵称" },
                                        style = MaterialTheme.typography.headlineSmall,
                                        fontWeight = FontWeight.SemiBold,
                                        color = colors.textPrimary,
                                        maxLines = 1,
                                        overflow = TextOverflow.Ellipsis,
                                    )
                                    Spacer(Modifier.width(Spacing.sm))
                                    if (data.presenceStatus == "online") {
                                        Badge(text = "在线", tone = BadgeTone.ONLINE)
                                    }
                                    if (data.userType == "temp") {
                                        Badge(text = "访客", tone = BadgeTone.WARNING)
                                    }
                                }
                                if (!data.bio.isNullOrBlank()) {
                                    Spacer(Modifier.height(Spacing.xs))
                                    Text(
                                        text = data.bio,
                                        style = MaterialTheme.typography.bodySmall,
                                        color = colors.textSecondary,
                                        maxLines = 2,
                                        overflow = TextOverflow.Ellipsis,
                                    )
                                }
                            }
                        }
                        Spacer(Modifier.height(Spacing.lg))
                        UserMetrics(data.stats)
                    }
                    Spacer(Modifier.height(Spacing.xl))
                    SectionTitle(title = "我的内容")
                    Spacer(Modifier.height(Spacing.sm))
                    ListGroup {
                        ListRow(
                            title = "我的瞬间",
                            subtitle = "TODO：首版占位，后续接入 moments/fetch_created",
                            onClick = null,
                        )
                        ListRow(
                            title = "我的群组",
                            subtitle = "TODO：首版占位，后续接入 groups/fetch_joined",
                            onClick = null,
                        )
                    }
                    Spacer(Modifier.height(Spacing.xl))
                    SectionTitle(title = "账号")
                    Spacer(Modifier.height(Spacing.sm))
                    ListGroup {
                        ListRow(title = "通知中心", subtitle = "互动与系统通知", onClick = onOpenNotify)
                        ListRow(title = "设置", subtitle = "主题外观 / 账号 / 关于", onClick = onOpenSettings)
                        ListRow(
                            title = "管理后台",
                            subtitle = "TODO：平台管理员功能不在首版移动端范围",
                            onClick = null,
                        )
                    }
                    Spacer(Modifier.height(Spacing.xxl))
                }
            }
        }
    }
}

@Composable
private fun UserMetrics(stats: UserStats?) {
    val colors = HotSpotTheme.colors
    val metrics = listOf(
        "瞬间" to (stats?.momentsCount ?: 0),
        "好友" to (stats?.friendsCount ?: 0),
        "关注" to (stats?.followingCount ?: 0),
        "粉丝" to (stats?.followersCount ?: 0),
    )
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceEvenly,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        metrics.forEach { (label, value) ->
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text(
                    text = value.toString(),
                    style = MaterialTheme.typography.titleLarge,
                    fontWeight = FontWeight.SemiBold,
                    color = colors.textPrimary,
                )
                Text(
                    text = label,
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
            }
        }
    }
}
