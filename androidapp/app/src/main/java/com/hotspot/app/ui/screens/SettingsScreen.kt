package com.hotspot.app.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.border
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
import androidx.compose.material3.AlertDialog
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
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.compose.runtime.DisposableEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.hotspot.app.AppContainer
import com.hotspot.app.BuildConfig
import com.hotspot.app.R
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.location.BrowseAnchor
import com.hotspot.app.data.model.AccountData
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.Badge
import com.hotspot.app.ui.components.BadgeTone
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.ListGroup
import com.hotspot.app.ui.components.ListRow
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemCard
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.DarkModePref
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.ui.theme.ThemeId
import com.hotspot.app.ui.theme.Tokens
import kotlinx.coroutines.launch

/**
 * 设置（设计稿 87:4864）：账号信息（编辑资料入口）/ 外观与显示（三主题 + 明暗三档）/
 * 系统与关于（版本、我的举报、退出登录）。
 * 主题偏好持久化在 ThemeController（SharedPreferences），切换即时生效。
 * 「地图与位置」展示当前浏览锚点（/map 选点页写入；未设置时回退默认位置），可跳转地图选点或恢复默认。
 */
@Composable
fun SettingsScreen(
    onBack: () -> Unit,
    onOpenEditProfile: () -> Unit,
    onOpenReports: () -> Unit,
    onOpenBlocked: () -> Unit,
    onOpenAdmin: () -> Unit,
    onOpenMap: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    val themeController = AppContainer.themeController
    val uiTheme by themeController.state.collectAsStateWithLifecycle()
    val scope = rememberCoroutineScope()
    val lifecycleOwner = LocalLifecycleOwner.current
    var anchor by remember { mutableStateOf(AppContainer.browseAnchorStore.current()) }
    var fallbackLocationName by remember { mutableStateOf<String?>(null) }
    var account by remember { mutableStateOf<AccountData?>(null) }
    var showLogoutConfirm by remember { mutableStateOf(false) }
    var loggingOut by remember { mutableStateOf(false) }
    var isAdmin by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        isAdmin = AppContainer.adminRepository.me() is ApiResult.Ok
    }

    // 从地图选点页返回时刷新当前锚点（该页在返回栈中，不一定重新组合）。
    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) {
                anchor = AppContainer.browseAnchorStore.current()
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
    }

    LaunchedEffect(Unit) {
        fallbackLocationName = AppContainer.defaultLocation()?.locationName
        when (val result = AppContainer.profileRepository.account()) {
            is ApiResult.Ok -> account = result.data
            is ApiResult.Err -> Unit
        }
    }

    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "设置", onBack = onBack)
        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = Spacing.lg),
        ) {
            SettingsSubtitleChip("管理你的账号偏好、主题外观与系统选项")
            Spacer(Modifier.height(Spacing.lg))
            SettingsSectionTitle(title = "账号信息", subtitle = "当前登录的账号信息与个人主页入口")
            Spacer(Modifier.height(Spacing.sm))
            SystemCard(contentPadding = PaddingValues(Spacing.md)) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(AppShapes.compact)
                        .clickable(onClick = onOpenEditProfile)
                        .padding(vertical = Spacing.xs),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Avatar(
                        name = account?.nickname.orEmpty(),
                        imageUrl = account?.avatarUrl,
                        size = 48.dp,
                    )
                    Spacer(Modifier.width(Spacing.md))
                    Column(Modifier.weight(1f)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text(
                                text = account?.nickname?.ifBlank { "未设置昵称" } ?: "加载中…",
                                style = MaterialTheme.typography.bodyLarge,
                                fontWeight = FontWeight.SemiBold,
                                color = colors.textPrimary,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                            Spacer(Modifier.width(Spacing.sm))
                            Badge(text = "已登录", tone = BadgeTone.ONLINE)
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
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(
                            text = "编辑资料",
                            style = MaterialTheme.typography.labelMedium,
                            color = colors.accent,
                        )
                        Icon(
                            painter = painterResource(R.drawable.ic_chevron_right),
                            contentDescription = null,
                            modifier = Modifier.size(12.dp),
                            tint = colors.accent,
                        )
                    }
                }
            }
            Spacer(Modifier.height(Spacing.xl))
            SettingsSectionTitle(title = "外观与显示", subtitle = "自定义界面的视觉色彩与深浅模式，全局即时生效")
            Spacer(Modifier.height(Spacing.sm))
            SystemCard {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(
                        painter = painterResource(R.drawable.ic_row_moments),
                        contentDescription = null,
                        modifier = Modifier.size(Dimens.iconSm),
                        tint = colors.textSecondary,
                    )
                    Spacer(Modifier.width(Spacing.sm2))
                    Column {
                        Text(
                            text = "应用主题",
                            style = MaterialTheme.typography.titleSmall,
                            fontWeight = FontWeight.SemiBold,
                            color = colors.textPrimary,
                        )
                        Text(
                            text = "选择全局视觉色彩风格 · 冷色玻璃",
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.textSecondary,
                        )
                    }
                }
                Spacer(Modifier.height(Spacing.sm))
                Row(horizontalArrangement = Arrangement.spacedBy(Spacing.sm)) {
                    ThemeId.entries.forEach { themeId ->
                        ThemeOptionCard(
                            themeId = themeId,
                            selected = uiTheme.themeId == themeId,
                            onClick = { themeController.setThemeId(themeId) },
                            modifier = Modifier.weight(1f),
                        )
                    }
                }
                Spacer(Modifier.height(Spacing.lg))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(
                        painter = painterResource(R.drawable.ic_mode_light),
                        contentDescription = null,
                        modifier = Modifier.size(Dimens.iconSm),
                        tint = colors.textSecondary,
                    )
                    Spacer(Modifier.width(Spacing.sm2))
                    Column {
                        Text(
                            text = "外观模式",
                            style = MaterialTheme.typography.titleSmall,
                            fontWeight = FontWeight.SemiBold,
                            color = colors.textPrimary,
                        )
                        Text(
                            text = darkModeHint(uiTheme.darkMode),
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.textSecondary,
                        )
                    }
                }
                Spacer(Modifier.height(Spacing.sm))
                Row(horizontalArrangement = Arrangement.spacedBy(Spacing.sm)) {
                    DarkModePref.entries.forEach { mode ->
                        DarkModeOptionCard(
                            mode = mode,
                            selected = uiTheme.darkMode == mode,
                            onClick = { themeController.setDarkMode(mode) },
                            modifier = Modifier.weight(1f),
                        )
                    }
                }
            }
            Spacer(Modifier.height(Spacing.xl))
            SettingsSectionTitle(title = "地图与位置", subtitle = "附近与发现使用的浏览锚点，可在选点页重新选择")
            Spacer(Modifier.height(Spacing.sm))
            SystemCard(contentPadding = PaddingValues(Spacing.md)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(
                        painter = painterResource(R.drawable.ic_map_cta_pin),
                        contentDescription = null,
                        modifier = Modifier.size(Dimens.iconSm),
                        tint = colors.textSecondary,
                    )
                    Spacer(Modifier.width(Spacing.sm2))
                    Column(Modifier.weight(1f)) {
                        Text(
                            text = anchor?.label ?: (fallbackLocationName?.ifBlank { null } ?: "默认位置"),
                            style = MaterialTheme.typography.titleSmall,
                            fontWeight = FontWeight.SemiBold,
                            color = colors.textPrimary,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                        Text(
                            text = anchor?.let { "已自定义 · " + formatAnchorCoordinates(it) }
                                ?: "跟随默认位置（未自定义）",
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.textSecondary,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                }
                Spacer(Modifier.height(Spacing.sm))
                Row(horizontalArrangement = Arrangement.spacedBy(Spacing.sm)) {
                    SystemButton(
                        text = "去地图选点",
                        onClick = onOpenMap,
                        variant = ButtonVariant.SECONDARY,
                        size = ButtonSize.SM,
                    )
                    if (anchor != null) {
                        SystemButton(
                            text = "恢复默认",
                            onClick = {
                                AppContainer.clearBrowseAnchor()
                                anchor = null
                            },
                            variant = ButtonVariant.SECONDARY,
                            size = ButtonSize.SM,
                        )
                    }
                }
            }
            Spacer(Modifier.height(Spacing.xl))
            SettingsSectionTitle(title = "系统与关于", subtitle = "客户端版本及当前设备会话状态")
            Spacer(Modifier.height(Spacing.sm))
            SystemCard(contentPadding = PaddingValues(vertical = Spacing.xs)) {
                ListRow(
                    title = "版本",
                    subtitle = BuildConfig.VERSION_NAME + " · " + BuildConfig.BUILD_TYPE,
                    onClick = null,
                )
                ListRow(
                    title = "我的举报",
                    subtitle = "查看举报处理进度与结果",
                    onClick = onOpenReports,
                )
                ListRow(
                    title = "屏蔽名单",
                    subtitle = "管理你屏蔽的用户，解除后可重新互动",
                    onClick = onOpenBlocked,
                )
                if (isAdmin) {
                    ListRow(
                        title = "管理后台",
                        subtitle = "举报、用户/群组治理、公告与统计",
                        onClick = onOpenAdmin,
                    )
                }
                ListRow(
                    title = "退出登录",
                    subtitle = "退出后需要重新登录才能继续使用",
                    onClick = { showLogoutConfirm = true },
                )
            }
            Spacer(Modifier.height(Spacing.xxl))
        }
    }

    if (showLogoutConfirm) {
        AlertDialog(
            onDismissRequest = { showLogoutConfirm = false },
            title = { Text("退出登录？") },
            text = { Text("退出后本机将清除登录凭证，需要重新登录。") },
            confirmButton = {
                SystemButton(
                    text = if (loggingOut) "退出中…" else "退出登录",
                    onClick = {
                        if (loggingOut) return@SystemButton
                        scope.launch {
                            loggingOut = true
                            AppContainer.authRepository.signout()
                            loggingOut = false
                            showLogoutConfirm = false
                        }
                    },
                    variant = ButtonVariant.DANGER,
                    size = ButtonSize.SM,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { showLogoutConfirm = false },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                )
            },
        )
    }
}

private fun formatAnchorCoordinates(anchor: BrowseAnchor): String =
    String.format(java.util.Locale.US, "%.4f, %.4f", anchor.latitude, anchor.longitude)

@Composable
private fun SettingsSubtitleChip(text: String) {
    val colors = HotSpotTheme.colors
    Box(
        modifier = Modifier
            .clip(AppShapes.pill)
            .background(colors.island)
            .border(1.dp, colors.borderSubtle, AppShapes.pill)
            .padding(horizontal = Spacing.md2, vertical = Spacing.xs),
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.labelSmall,
            color = colors.textSecondary,
        )
    }
}

@Composable
private fun SettingsSectionTitle(title: String, subtitle: String) {
    val colors = HotSpotTheme.colors
    Column {
        Text(
            text = title,
            style = MaterialTheme.typography.titleSmall,
            fontWeight = FontWeight.SemiBold,
            color = colors.textPrimary,
        )
        Spacer(Modifier.height(2.dp))
        Text(
            text = subtitle,
            style = MaterialTheme.typography.labelSmall,
            color = colors.textSecondary,
        )
    }
}

/** 主题预览卡：三色点（主色/画布/在线绿）+ 名称，选中 accent 描边（设计稿 87:4864 应用主题）。 */
@Composable
private fun ThemeOptionCard(
    themeId: ThemeId,
    selected: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    val palette = when (themeId) {
        ThemeId.VIBRANT -> Tokens.Palette.VibrantLight
        ThemeId.FOREST -> Tokens.Palette.ForestLight
        ThemeId.INKWASH -> Tokens.Palette.InkwashLight
    }
    Column(
        modifier = modifier
            .height(72.dp)
            .clip(RoundedCornerShape(16.dp))
            .background(colors.island)
            .border(
                width = if (selected) 2.dp else 1.dp,
                color = if (selected) colors.accent else colors.borderSubtle,
                shape = RoundedCornerShape(16.dp),
            )
            .clickable(onClick = onClick)
            .padding(Spacing.sm2),
    ) {
        Row(horizontalArrangement = Arrangement.spacedBy(3.dp)) {
            Box(Modifier.size(8.dp).clip(CircleShape).background(palette.accentPrimary))
            Box(
                Modifier
                    .size(8.dp)
                    .clip(CircleShape)
                    .background(palette.pageBg)
                    .border(0.5.dp, palette.border, CircleShape),
            )
            Box(Modifier.size(8.dp).clip(CircleShape).background(palette.online))
        }
        Spacer(Modifier.weight(1f))
        Text(
            text = themeId.label,
            style = MaterialTheme.typography.labelSmall,
            fontWeight = FontWeight.Medium,
            color = if (selected) colors.textPrimary else colors.textSecondary,
        )
    }
}

/** 明暗模式卡：图标 + 名称 + 右上单选点（设计稿 87:4864 外观模式）。 */
@Composable
private fun DarkModeOptionCard(
    mode: DarkModePref,
    selected: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    val iconRes = when (mode) {
        DarkModePref.AUTO -> R.drawable.ic_mode_auto
        DarkModePref.LIGHT -> R.drawable.ic_mode_light
        DarkModePref.DARK -> R.drawable.ic_mode_dark
    }
    Box(
        modifier = modifier
            .height(72.dp)
            .clip(RoundedCornerShape(16.dp))
            .background(colors.island)
            .border(
                width = if (selected) 2.dp else 1.dp,
                color = if (selected) colors.accent else colors.borderSubtle,
                shape = RoundedCornerShape(16.dp),
            )
            .clickable(onClick = onClick)
            .padding(Spacing.sm2),
    ) {
        Icon(
            painter = painterResource(iconRes),
            contentDescription = null,
            modifier = Modifier
                .align(Alignment.TopStart)
                .size(18.dp),
            tint = if (selected) colors.accent else colors.textSecondary,
        )
        Box(
            modifier = Modifier
                .align(Alignment.TopEnd)
                .size(14.dp)
                .clip(CircleShape)
                .border(
                    1.5.dp,
                    if (selected) colors.accent else colors.borderSubtle,
                    CircleShape,
                ),
            contentAlignment = Alignment.Center,
        ) {
            if (selected) {
                Box(Modifier.size(6.dp).clip(CircleShape).background(colors.accent))
            }
        }
        Text(
            text = mode.label,
            style = MaterialTheme.typography.labelSmall,
            fontWeight = FontWeight.Medium,
            color = if (selected) colors.textPrimary else colors.textSecondary,
            modifier = Modifier.align(Alignment.BottomStart),
        )
    }
}

private fun darkModeHint(mode: DarkModePref): String = when (mode) {
    DarkModePref.AUTO -> "跟随系统外观"
    DarkModePref.LIGHT -> "当前使用浅色外观"
    DarkModePref.DARK -> "当前使用深色外观"
}

/** @ 句柄：契约未返回 login_id，用 email 本地部分 / 手机号回填（与个人中心同口径）。 */
private fun accountHandle(account: AccountData?): String? {
    if (account == null) return null
    val email = account.email
    if (!email.isNullOrBlank()) return "@" + email.substringBefore("@")
    val phone = account.phone
    if (!phone.isNullOrBlank()) return "@" + phone
    return null
}
