package com.hotspot.app.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Place
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.Icon
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.R
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.AccountData
import com.hotspot.app.data.model.UserStats
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.Badge
import com.hotspot.app.ui.components.BadgeTone
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.HeaderIconButton
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemCard
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import kotlinx.coroutines.launch

/**
 * 个人中心（设计稿 87:3393）：资料卡（头像/昵称/@/简介/所在地 + 瞬间·粉丝·群组三指标 + 编辑资料/设置按钮）
 * → 「我的内容」5 行（我的瞬间/我的群组/好友/关注/粉丝，右侧计数）→ 悬浮岛底栏。
 * 数据走 users/account；关系页、编辑资料页各自拉取，回到本页时重新加载以反映修改。
 */
@Composable
fun MeScreen(
    onOpenSettings: () -> Unit,
    onOpenNotify: () -> Unit,
    onOpenMoments: () -> Unit,
    onOpenGroups: () -> Unit,
    onOpenEditProfile: () -> Unit,
    onOpenRelationships: (tab: Int) -> Unit,
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
        // 标题行（设计稿左对齐「个人中心」；通知入口保留——通知页设计入口在其他位置，功能不缺失）
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = Spacing.lg, vertical = Spacing.md),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = "个人中心",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.SemiBold,
                color = colors.textPrimary,
            )
            Spacer(Modifier.weight(1f))
            HeaderIconButton(
                icon = Icons.Filled.Notifications,
                contentDescription = "通知",
                onClick = onOpenNotify,
            )
        }
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
                    MeProfileCard(
                        account = data,
                        onOpenEditProfile = onOpenEditProfile,
                        onOpenSettings = onOpenSettings,
                        onOpenMoments = onOpenMoments,
                        onOpenGroups = onOpenGroups,
                        onOpenFollowers = { onOpenRelationships(2) },
                    )
                    Spacer(Modifier.height(Spacing.xl))
                    MeContentSection(
                        stats = data.stats,
                        onOpenMoments = onOpenMoments,
                        onOpenGroups = onOpenGroups,
                        onOpenRelationships = onOpenRelationships,
                    )
                    Spacer(Modifier.height(Spacing.xxl))
                }
            }
        }
    }
}

@Composable
private fun MeProfileCard(
    account: AccountData,
    onOpenEditProfile: () -> Unit,
    onOpenSettings: () -> Unit,
    onOpenMoments: () -> Unit,
    onOpenGroups: () -> Unit,
    onOpenFollowers: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    SystemCard(contentPadding = PaddingValues(Spacing.lg)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Avatar(
                name = account.nickname,
                imageUrl = account.avatarUrl,
                size = 64.dp,
            )
            Spacer(Modifier.width(Spacing.md))
            Column(Modifier.weight(1f)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = account.nickname.ifBlank { "未设置昵称" },
                        style = MaterialTheme.typography.titleLarge,
                        fontWeight = FontWeight.SemiBold,
                        color = colors.textPrimary,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                    if (account.presenceStatus == "online") {
                        Spacer(Modifier.width(Spacing.sm))
                        Badge(text = "在线", tone = BadgeTone.ONLINE)
                    }
                    if (account.userType == "temp") {
                        Spacer(Modifier.width(Spacing.xs))
                        Badge(text = "访客", tone = BadgeTone.WARNING)
                    }
                }
                val handle = accountHandle(account)
                if (handle != null) {
                    Text(
                        text = handle,
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textSecondary,
                    )
                }
            }
        }
        if (!account.bio.isNullOrBlank()) {
            Spacer(Modifier.height(Spacing.md))
            Text(
                text = account.bio,
                style = MaterialTheme.typography.bodySmall,
                color = colors.textSecondary,
                maxLines = 3,
                overflow = TextOverflow.Ellipsis,
            )
        }
        val locationName = account.location?.locationName
        if (!locationName.isNullOrBlank()) {
            Spacer(Modifier.height(Spacing.sm))
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(
                    imageVector = Icons.Filled.Place,
                    contentDescription = null,
                    modifier = Modifier.size(12.dp),
                    tint = colors.textSecondary,
                )
                Spacer(Modifier.width(Spacing.xs))
                Text(
                    text = locationName,
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
            }
        }
        Spacer(Modifier.height(Spacing.lg))
        MeMetrics(stats = account.stats, onOpenMoments = onOpenMoments, onOpenGroups = onOpenGroups, onOpenFollowers = onOpenFollowers)
        Spacer(Modifier.height(Spacing.lg))
        Row(horizontalArrangement = Arrangement.spacedBy(Spacing.sm)) {
            SystemButton(
                text = "编辑资料",
                onClick = onOpenEditProfile,
                variant = ButtonVariant.PRIMARY,
                size = ButtonSize.MD,
                fillMaxWidth = false,
                leadingIcon = {
                    Icon(
                        imageVector = Icons.Filled.Edit,
                        contentDescription = null,
                        modifier = Modifier.size(Dimens.iconXs),
                        tint = colors.textOnAccent,
                    )
                },
                modifier = Modifier.weight(1f),
            )
            SystemButton(
                text = "设置",
                onClick = onOpenSettings,
                variant = ButtonVariant.SOFT,
                size = ButtonSize.MD,
                fillMaxWidth = false,
                leadingIcon = {
                    Icon(
                        imageVector = Icons.Filled.Settings,
                        contentDescription = null,
                        modifier = Modifier.size(Dimens.iconXs),
                        tint = colors.textAccentOnSoft,
                    )
                },
                modifier = Modifier.weight(1f),
            )
        }
    }
}

@Composable
private fun MeMetrics(
    stats: UserStats?,
    onOpenMoments: () -> Unit,
    onOpenGroups: () -> Unit,
    onOpenFollowers: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    val metrics = listOf(
        Triple("瞬间", stats?.momentsCount ?: 0, onOpenMoments),
        Triple("粉丝", stats?.followersCount ?: 0, onOpenFollowers),
        Triple("群组", stats?.groupsCount ?: 0, onOpenGroups),
    )
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceEvenly) {
        metrics.forEach { (label, value, onClick) ->
            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                modifier = Modifier
                    .clip(AppShapes.compact)
                    .clickable(onClick = onClick)
                    .padding(horizontal = Spacing.lg, vertical = Spacing.xs),
            ) {
                Text(
                    text = value.toString(),
                    style = MaterialTheme.typography.headlineSmall,
                    fontWeight = FontWeight.SemiBold,
                    color = colors.textPrimary,
                )
                Spacer(Modifier.height(2.dp))
                Text(
                    text = label,
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
            }
        }
    }
}

/** 「我的内容」：标题 + 5 行入口（右计数 + chevron），对齐设计稿 87:3393。 */
@Composable
private fun MeContentSection(
    stats: UserStats?,
    onOpenMoments: () -> Unit,
    onOpenGroups: () -> Unit,
    onOpenRelationships: (Int) -> Unit,
) {
    val colors = HotSpotTheme.colors
    Row(verticalAlignment = Alignment.CenterVertically) {
        Icon(
            painter = painterResource(R.drawable.ic_row_groups),
            contentDescription = null,
            modifier = Modifier.size(Dimens.iconSm),
            tint = colors.textSecondary,
        )
        Spacer(Modifier.width(Spacing.sm2))
        Column {
            Text(
                text = "我的内容",
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.SemiBold,
                color = colors.textPrimary,
            )
            Text(
                text = "内容与关系资产",
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
            )
        }
    }
    Spacer(Modifier.height(Spacing.md))
    SystemCard(contentPadding = PaddingValues(vertical = Spacing.xs)) {
        MeContentRow(
            iconRes = R.drawable.ic_row_moments,
            title = "我的瞬间",
            subtitle = "记录城市里的光影",
            countText = (stats?.momentsCount ?: 0).toString() + " 条",
            onClick = onOpenMoments,
        )
        MeContentRow(
            iconRes = R.drawable.ic_row_groups,
            title = "我的群组",
            subtitle = "和同好一起探索",
            countText = (stats?.groupsCount ?: 0).toString() + " 个",
            onClick = onOpenGroups,
        )
        MeContentRow(
            iconRes = R.drawable.ic_row_friends,
            title = "好友",
            subtitle = "可以直接聊天的人",
            countText = (stats?.friendsCount ?: 0).toString() + " 位",
            onClick = { onOpenRelationships(0) },
        )
        MeContentRow(
            iconRes = R.drawable.ic_row_following,
            title = "关注",
            subtitle = "你订阅的创作者",
            countText = (stats?.followingCount ?: 0).toString() + " 人",
            onClick = { onOpenRelationships(1) },
        )
        MeContentRow(
            iconRes = R.drawable.ic_row_fans,
            title = "粉丝",
            subtitle = "关注你的人",
            countText = (stats?.followersCount ?: 0).toString() + " 人",
            onClick = { onOpenRelationships(2) },
        )
    }
}

@Composable
private fun MeContentRow(
    iconRes: Int,
    title: String,
    subtitle: String,
    countText: String,
    onClick: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = Spacing.sm, vertical = Spacing.sm),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            modifier = Modifier
                .size(36.dp)
                .clip(CircleShape)
                .background(colors.accentSoftBg),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painter = painterResource(iconRes),
                contentDescription = null,
                modifier = Modifier.size(Dimens.iconSm),
                tint = colors.accent,
            )
        }
        Spacer(Modifier.width(Spacing.md))
        Column(Modifier.weight(1f)) {
            Text(
                text = title,
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Medium,
                color = colors.textPrimary,
            )
            Text(
                text = subtitle,
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
        Text(
            text = countText,
            style = MaterialTheme.typography.labelSmall,
            color = colors.textSecondary,
        )
        Spacer(Modifier.width(Spacing.xs))
        Icon(
            imageVector = Icons.Filled.KeyboardArrowRight,
            contentDescription = null,
            modifier = Modifier.size(Dimens.iconSm),
            tint = colors.textSecondary,
        )
    }
}

/** @ 句柄：契约未返回 login_id，用 email 本地部分 / 手机号回填展示，无则省略。 */
private fun accountHandle(account: AccountData): String? {
    val email = account.email
    if (!email.isNullOrBlank()) return "@" + email.substringBefore("@")
    val phone = account.phone
    if (!phone.isNullOrBlank()) return "@" + phone
    return null
}
