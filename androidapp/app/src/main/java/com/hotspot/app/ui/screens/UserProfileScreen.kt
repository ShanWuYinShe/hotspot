package com.hotspot.app.ui.screens

import android.content.Context
import android.content.Intent
import android.widget.Toast
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material.icons.filled.Place
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import com.hotspot.app.AppContainer
import com.hotspot.app.R
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.MomentItem
import com.hotspot.app.data.model.Relationship
import com.hotspot.app.data.model.UserItem
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.Badge
import com.hotspot.app.ui.components.BadgeTone
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.InlineLoading
import com.hotspot.app.ui.components.ListEnd
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.ReportDialog
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemCard
import com.hotspot.app.ui.components.SystemInput
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.MediaUrls
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch

private const val SITE_BASE_URL = "https://chaoset.com"

/**
 * 他人主页（设计稿 87:5163）：返回 + 右侧玻璃胶囊（分享资料 / 更多）浮层；
 * 资料卡（头像 + 昵称 + 在线徽章 + @句柄 + 简介 + 位置/标签 + 瞬间/粉丝/关注 + 关注/私信）
 * + 「最近瞬间」带图画廊（文字瞬间不计入，翻页接 users/fetch_moments）。
 * 更多菜单：设置备注（仅已关注）/ 屏蔽与解除 / 举报用户，行为对齐 Web users/[userId]。
 */
@Composable
fun UserProfileScreen(
    userId: String,
    onBack: () -> Unit,
    onOpenMoment: (String) -> Unit,
    onOpenMessage: (String) -> Unit,
) {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val listState = rememberLazyListState()

    var user by remember { mutableStateOf<UserItem?>(null) }
    var relation by remember { mutableStateOf<Relationship?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var following by remember { mutableStateOf(false) }
    var blocked by remember { mutableStateOf(false) }
    var followBusy by remember { mutableStateOf(false) }
    var remark by remember { mutableStateOf<String?>(null) }
    var remarkDraft by remember { mutableStateOf("") }
    var menuOpen by remember { mutableStateOf(false) }
    var showRemark by remember { mutableStateOf(false) }
    var remarkSaving by remember { mutableStateOf(false) }
    var showBlockConfirm by remember { mutableStateOf(false) }
    var showReport by remember { mutableStateOf(false) }
    var reportSubmitting by remember { mutableStateOf(false) }

    val gallery = remember { mutableStateListOf<MomentItem>() }
    var nextCursor by remember { mutableStateOf<String?>(null) }
    var loadingMore by remember { mutableStateOf(false) }
    var endReached by remember { mutableStateOf(false) }

    suspend fun load(initial: Boolean) {
        if (initial) {
            loading = true
            error = null
        }
        when (val result = AppContainer.profileRepository.openUserDoor(userId, includeRecentMoments = true)) {
            is ApiResult.Ok -> {
                user = result.data.user
                relation = result.data.user.relation
                following = result.data.user.relation?.isFollowing == true
                blocked = result.data.user.relation?.isBlocked == true
                val page = result.data.recentMoments
                gallery.clear()
                gallery.addAll(page?.items.orEmpty().filter { it.images.isNotEmpty() })
                nextCursor = page?.nextCursor
                endReached = page?.nextCursor == null
                loading = false
            }
            is ApiResult.Err -> {
                loading = false
                error = if (result.error is com.hotspot.app.data.api.ApiError.Business && (result.error as com.hotspot.app.data.api.ApiError.Business).code == 10300) {
                    "该用户不存在或已隐藏资料"
                } else {
                    result.error.userMessage
                }
            }
        }
    }

    suspend fun loadMore() {
        val cursor = nextCursor ?: return
        if (loadingMore || endReached) return
        loadingMore = true
        when (val result = AppContainer.profileRepository.userMoments(userId, cursor)) {
            is ApiResult.Ok -> {
                gallery.addAll(result.data.items.filter { it.images.isNotEmpty() })
                nextCursor = result.data.nextCursor
                endReached = result.data.nextCursor == null
            }
            is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
        }
        loadingMore = false
    }

    LaunchedEffect(userId) { load(initial = true) }

    // 画廊到底自动翻页（image-only 过滤后条目变少，提前两行触发）
    val shouldLoadMore by remember(gallery.size, endReached, loadingMore) {
        derivedStateOf {
            val info = listState.layoutInfo
            val last = info.visibleItemsInfo.lastOrNull()?.index ?: 0
            last >= info.totalItemsCount - 3 && nextCursor != null && !endReached && !loadingMore
        }
    }
    LaunchedEffect(shouldLoadMore, gallery.size) {
        if (shouldLoadMore) loadMore()
    }

    Box(Modifier.fillMaxSize()) {
        when {
            loading -> LoadingState(message = "正在加载用户主页…")
            error != null -> ErrorState(message = error ?: "加载失败", onRetry = { scope.launch { load(initial = true) } })
            user != null -> {
                val data = user!!
                LazyColumn(
                    state = listState,
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(
                        start = Spacing.lg,
                        end = Spacing.lg,
                        top = 56.dp,
                        bottom = Spacing.xxl,
                    ),
                ) {
                    item {
                        ProfileCard(
                            user = data,
                            following = following,
                            blocked = blocked,
                            followBusy = followBusy,
                            remark = remark,
                            onToggleFollow = {
                                if (followBusy || blocked) return@ProfileCard
                                val target = !following
                                following = target
                                followBusy = true
                                scope.launch {
                                    when (val result = AppContainer.profileRepository.setFollow(userId, target)) {
                                        is ApiResult.Ok -> Toast.makeText(
                                            context,
                                            if (target) "已关注" else "已取消关注",
                                            Toast.LENGTH_SHORT,
                                        ).show()
                                        is ApiResult.Err -> {
                                            following = !target
                                            Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                                        }
                                    }
                                    followBusy = false
                                }
                            },
                            onMessage = {
                                if (!blocked) onOpenMessage(userId) else Toast.makeText(context, "已屏蔽该用户，无法私信", Toast.LENGTH_SHORT).show()
                            },
                        )
                    }
                    item {
                        Spacer(Modifier.height(Spacing.xl))
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Icon(
                                painter = painterResource(R.drawable.ic_action_gallery),
                                contentDescription = null,
                                modifier = Modifier.size(Dimens.iconSm),
                                tint = colors.textPrimary,
                            )
                            Spacer(Modifier.width(Spacing.sm))
                            Text(
                                text = "最近瞬间",
                                style = MaterialTheme.typography.titleSmall,
                                fontWeight = FontWeight.SemiBold,
                                color = colors.textPrimary,
                            )
                        }
                        Spacer(Modifier.height(Spacing.sm))
                    }
                    if (gallery.isEmpty()) {
                        item { GalleryEmpty() }
                    } else {
                        items((gallery.size + 1) / 2) { rowIndex ->
                            Row(horizontalArrangement = Arrangement.spacedBy(Spacing.sm)) {
                                for (column in 0..1) {
                                    val index = rowIndex * 2 + column
                                    if (index < gallery.size) {
                                        GalleryTile(
                                            moment = gallery[index],
                                            modifier = Modifier.weight(1f),
                                            onClick = { onOpenMoment(gallery[index].momentId) },
                                        )
                                    } else {
                                        Spacer(Modifier.weight(1f))
                                    }
                                }
                            }
                            Spacer(Modifier.height(Spacing.sm))
                        }
                    }
                    item {
                        when {
                            loadingMore -> InlineLoading()
                            endReached && gallery.isNotEmpty() -> ListEnd()
                            else -> Box(Modifier.height(1.dp))
                        }
                    }
                }
            }
        }

        // 顶部浮层工具（对齐设计稿：返回玻璃圆钮 + 右侧分享/更多玻璃胶囊）
        Row(
            modifier = Modifier
                .align(Alignment.TopStart)
                .fillMaxWidth()
                .padding(horizontal = Spacing.lg, vertical = Spacing.sm),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            GlassCircleButton(iconRes = null, contentDescription = "返回", onClick = onBack, back = true)
            Spacer(Modifier.weight(1f))
            Row(
                modifier = Modifier
                    .clip(AppShapes.pill)
                    .background(colors.island)
                    .border(1.dp, colors.borderSubtle, AppShapes.pill)
                    .padding(horizontal = Spacing.xs, vertical = 2.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                GlassIconButton(iconRes = R.drawable.ic_action_share, contentDescription = "分享资料") {
                    shareProfile(context, scope, user?.user?.nickname.orEmpty(), userId)
                }
                Box {
                    GlassIconButton(iconRes = R.drawable.ic_action_more, contentDescription = "更多操作") {
                        menuOpen = true
                    }
                    DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
                        DropdownMenuItem(
                            text = { Text(if (following) "设置备注" else "设置备注（关注后可设）") },
                            enabled = following && !blocked,
                            onClick = {
                                menuOpen = false
                                remarkDraft = remark.orEmpty()
                                showRemark = true
                            },
                        )
                        DropdownMenuItem(
                            text = { Text(if (blocked) "解除屏蔽" else "屏蔽用户") },
                            onClick = {
                                menuOpen = false
                                showBlockConfirm = true
                            },
                        )
                        DropdownMenuItem(
                            text = { Text("举报用户") },
                            onClick = {
                                menuOpen = false
                                showReport = true
                            },
                        )
                    }
                }
            }
        }
    }

    if (showRemark) {
        AlertDialog(
            onDismissRequest = { showRemark = false },
            title = { Text("设置备注") },
            text = {
                Column {
                    SystemInput(
                        value = remarkDraft,
                        onValueChange = { remarkDraft = it.take(20) },
                        placeholder = "给 TA 一个备注名（≤20 字）",
                        singleLine = true,
                    )
                    if (!following) {
                        Spacer(Modifier.height(Spacing.xs))
                        Text(
                            text = "关注对方后可设备注",
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.textSecondary,
                        )
                    }
                }
            },
            confirmButton = {
                SystemButton(
                    text = if (remarkSaving) "保存中…" else "保存",
                    onClick = {
                        if (remarkSaving) return@SystemButton
                        remarkSaving = true
                        scope.launch {
                            when (val result = AppContainer.profileRepository.setRemark(userId, remarkDraft.trim())) {
                                is ApiResult.Ok -> {
                                    remark = remarkDraft.trim().ifBlank { null }
                                    showRemark = false
                                    Toast.makeText(context, "备注已保存", Toast.LENGTH_SHORT).show()
                                }
                                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                            }
                            remarkSaving = false
                        }
                    },
                    variant = ButtonVariant.PRIMARY,
                    size = ButtonSize.SM,
                    enabled = following && !remarkSaving,
                )
            },
            dismissButton = {
                SystemButton(text = "取消", onClick = { showRemark = false }, variant = ButtonVariant.SECONDARY, size = ButtonSize.SM)
            },
        )
    }

    if (showBlockConfirm) {
        AlertDialog(
            onDismissRequest = { showBlockConfirm = false },
            title = { Text(if (blocked) "解除屏蔽？" else "屏蔽该用户？") },
            text = {
                Text(
                    if (blocked) "解除后可以重新关注对方并接收其动态。" else "屏蔽后将无法收到对方消息与动态，且自动取消关注。",
                )
            },
            confirmButton = {
                SystemButton(
                    text = if (blocked) "解除屏蔽" else "屏蔽",
                    onClick = {
                        showBlockConfirm = false
                        val target = !blocked
                        scope.launch {
                            when (val result = AppContainer.profileRepository.setBlock(userId, target)) {
                                is ApiResult.Ok -> {
                                    blocked = target
                                    if (target) following = false
                                    Toast.makeText(
                                        context,
                                        if (target) "已屏蔽" else "已解除屏蔽",
                                        Toast.LENGTH_SHORT,
                                    ).show()
                                }
                                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                            }
                        }
                    },
                    variant = if (blocked) ButtonVariant.PRIMARY else ButtonVariant.DANGER,
                    size = ButtonSize.SM,
                )
            },
            dismissButton = {
                SystemButton(text = "取消", onClick = { showBlockConfirm = false }, variant = ButtonVariant.SECONDARY, size = ButtonSize.SM)
            },
        )
    }

    if (showReport) {
        ReportDialog(
            title = "举报用户",
            submitting = reportSubmitting,
            onDismiss = { if (!reportSubmitting) showReport = false },
            onSubmit = { reason ->
                if (reportSubmitting) return@ReportDialog
                reportSubmitting = true
                scope.launch {
                    when (
                        val result = AppContainer.reportsRepository.create(
                            targetType = "user",
                            targetId = userId,
                            reason = reason,
                        )
                    ) {
                        is ApiResult.Ok -> {
                            showReport = false
                            Toast.makeText(
                                context,
                                if (result.data.isDuplicate) "已提交过举报，等待处理" else "举报已提交",
                                Toast.LENGTH_SHORT,
                            ).show()
                        }
                        is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                    }
                    reportSubmitting = false
                }
            },
        )
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun ProfileCard(
    user: UserItem,
    following: Boolean,
    blocked: Boolean,
    followBusy: Boolean,
    remark: String?,
    onToggleFollow: () -> Unit,
    onMessage: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    val stats = user.stats
    SystemCard(contentPadding = PaddingValues(Spacing.lg)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Avatar(name = user.user.nickname, imageUrl = user.user.avatarUrl, size = 64.dp)
            Spacer(Modifier.width(Spacing.md))
            Column(Modifier.weight(1f)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = remark?.takeIf { it.isNotBlank() } ?: user.user.nickname.ifBlank { "匿名用户" },
                        style = MaterialTheme.typography.titleLarge,
                        fontWeight = FontWeight.SemiBold,
                        color = colors.textPrimary,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                    val presence = user.presenceStatus
                    if (presence != null) {
                        Spacer(Modifier.width(Spacing.sm))
                        Badge(
                            text = if (presence == "online") "在线" else "离线",
                            tone = if (presence == "online") BadgeTone.ONLINE else BadgeTone.NEUTRAL,
                        )
                    }
                }
                Text(
                    text = "@" + handleOf(user.user.userId),
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
            }
        }
        val bio = user.bio
        if (!bio.isNullOrBlank()) {
            Spacer(Modifier.height(Spacing.sm2))
            Text(
                text = bio,
                style = MaterialTheme.typography.bodyMedium,
                color = colors.textSecondary,
            )
        }
        if (user.location?.locationName != null || user.tags.isNotEmpty()) {
            Spacer(Modifier.height(Spacing.sm2))
            FlowRow(
                horizontalArrangement = Arrangement.spacedBy(Spacing.sm),
                verticalArrangement = Arrangement.spacedBy(Spacing.xs),
            ) {
                val locationName = user.location?.locationName
                if (!locationName.isNullOrBlank()) {
                    Row(
                        modifier = Modifier
                            .clip(AppShapes.pill)
                            .background(colors.subtle)
                            .padding(horizontal = Spacing.sm, vertical = 3.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Icon(
                            imageVector = Icons.Filled.Place,
                            contentDescription = null,
                            modifier = Modifier.size(12.dp),
                            tint = colors.textSecondary,
                        )
                        Spacer(Modifier.width(2.dp))
                        Text(
                            text = locationName,
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.textSecondary,
                        )
                    }
                }
                user.tags.take(4).forEach { tag ->
                    Box(
                        modifier = Modifier
                            .clip(AppShapes.pill)
                            .background(colors.accentSoftBg)
                            .padding(horizontal = Spacing.sm, vertical = 3.dp),
                    ) {
                        Text(
                            text = "#" + tag,
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.textAccentOnSoft,
                        )
                    }
                }
            }
        }
        Spacer(Modifier.height(Spacing.lg))
        Row(Modifier.fillMaxWidth()) {
            StatColumn(value = stats?.momentsCount ?: 0, label = "瞬间", modifier = Modifier.weight(1f))
            StatColumn(value = stats?.followersCount ?: 0, label = "粉丝", modifier = Modifier.weight(1f))
            StatColumn(value = stats?.followingCount ?: 0, label = "关注", modifier = Modifier.weight(1f))
        }
        Spacer(Modifier.height(Spacing.lg))
        Row(horizontalArrangement = Arrangement.spacedBy(Spacing.sm)) {
            SystemButton(
                text = when {
                    blocked -> "已屏蔽"
                    following -> "已关注"
                    else -> "关注"
                },
                onClick = onToggleFollow,
                variant = if (following) ButtonVariant.SECONDARY else ButtonVariant.PRIMARY,
                size = ButtonSize.MD,
                enabled = !blocked && !followBusy,
                loading = followBusy,
                modifier = Modifier.weight(1f),
                fillMaxWidth = true,
            )
            SystemButton(
                text = "私信",
                onClick = onMessage,
                variant = ButtonVariant.SECONDARY,
                size = ButtonSize.MD,
                leadingIcon = {
                    Icon(
                        painter = painterResource(R.drawable.ic_nav_messages),
                        contentDescription = null,
                        modifier = Modifier.size(14.dp),
                        tint = colors.textPrimary,
                    )
                },
                enabled = !blocked,
                modifier = Modifier.weight(1f),
                fillMaxWidth = true,
            )
        }
        if (blocked) {
            Spacer(Modifier.height(Spacing.xs))
            Text(
                text = "你已屏蔽此用户；关注和私信已停用。",
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
            )
        }
    }
}

@Composable
private fun StatColumn(value: Int, label: String, modifier: Modifier = Modifier) {
    val colors = HotSpotTheme.colors
    Column(modifier = modifier, horizontalAlignment = Alignment.CenterHorizontally) {
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

@Composable
private fun GalleryTile(moment: MomentItem, modifier: Modifier = Modifier, onClick: () -> Unit) {
    val colors = HotSpotTheme.colors
    val imageUrl = MediaUrls.absolute(moment.images.firstOrNull()?.url)
    Box(
        modifier = modifier
            .aspectRatio(1f)
            .clip(RoundedCornerShape(14.dp))
            .background(colors.subtle)
            .clickable(onClick = onClick),
    ) {
        AsyncImage(
            model = imageUrl,
            contentDescription = moment.description.ifBlank { "瞬间图片" },
            modifier = Modifier.fillMaxSize(),
            contentScale = ContentScale.Crop,
        )
    }
}

@Composable
private fun GalleryEmpty() {
    val colors = HotSpotTheme.colors
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = Spacing.xxl),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            modifier = Modifier
                .size(48.dp)
                .clip(CircleShape)
                .background(colors.subtle),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painter = painterResource(R.drawable.ic_action_gallery),
                contentDescription = null,
                modifier = Modifier.size(Dimens.iconSm),
                tint = colors.textSecondary,
            )
        }
        Spacer(Modifier.height(Spacing.md))
        Text(
            text = "还没有带图瞬间",
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = FontWeight.Medium,
            color = colors.textPrimary,
        )
        Spacer(Modifier.height(Spacing.xs))
        Text(
            text = "带图瞬间会展示在这里；文字瞬间暂不计入画廊。",
            style = MaterialTheme.typography.labelSmall,
            color = colors.textSecondary,
            textAlign = TextAlign.Center,
        )
    }
}

@Composable
private fun GlassCircleButton(
    iconRes: Int?,
    contentDescription: String,
    onClick: () -> Unit,
    back: Boolean = false,
) {
    val colors = HotSpotTheme.colors
    Box(
        modifier = Modifier
            .size(Dimens.backButton)
            .shadow(1.dp, CircleShape, ambientColor = colors.shadowDark, spotColor = colors.shadowDark)
            .clip(CircleShape)
            .background(colors.island)
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        if (back) {
            Icon(
                imageVector = Icons.Filled.ArrowBack,
                contentDescription = contentDescription,
                modifier = Modifier.size(Dimens.iconSm),
                tint = colors.textPrimary,
            )
        } else if (iconRes != null) {
            Icon(
                painter = painterResource(iconRes),
                contentDescription = contentDescription,
                modifier = Modifier.size(Dimens.iconSm),
                tint = colors.textPrimary,
            )
        }
    }
}

@Composable
private fun GlassIconButton(iconRes: Int, contentDescription: String, onClick: () -> Unit) {
    val colors = HotSpotTheme.colors
    Box(
        modifier = Modifier
            .size(Dimens.minTouchTarget)
            .clip(CircleShape)
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Icon(
            painter = painterResource(iconRes),
            contentDescription = contentDescription,
            modifier = Modifier.size(Dimens.iconSm),
            tint = colors.textPrimary,
        )
    }
}

private fun handleOf(userId: String): String {
    var handle = userId
    if (handle.startsWith("user_seed_")) handle = handle.removePrefix("user_seed_")
    else if (handle.startsWith("user_")) handle = handle.removePrefix("user_")
    return handle.ifBlank { "user" }
}

/** 分享资料：优先 share/create 生成可撤销分享码，失败/匿名场景降级为原始主页链接。 */
private fun shareProfile(context: Context, scope: CoroutineScope, nickname: String, userId: String) {
    scope.launch {
        val fallbackUrl = SITE_BASE_URL + "/users/" + userId
        val url = when (val result = AppContainer.profileRepository.createUserShare(userId)) {
            is ApiResult.Ok -> result.data.shareCode.takeIf { it.isNotBlank() }
                ?.let { SITE_BASE_URL + "/share/" + it }
                ?: fallbackUrl
            is ApiResult.Err -> fallbackUrl
        }
        val intent = Intent(Intent.ACTION_SEND).apply {
            type = "text/plain"
            putExtra(Intent.EXTRA_SUBJECT, nickname.ifBlank { "HotSpot 用户" } + "的主页")
            putExtra(Intent.EXTRA_TEXT, "「" + nickname.ifBlank { "HotSpot 用户" } + "」的主页：" + url)
        }
        context.startActivity(Intent.createChooser(intent, "分享资料"))
    }
}
