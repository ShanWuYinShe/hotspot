package com.hotspot.app.ui.screens

import android.content.ActivityNotFoundException
import android.content.Intent
import android.net.Uri
import android.widget.Toast
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
import androidx.compose.ui.platform.LocalContext
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
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.hotspot.app.AppContainer
import com.hotspot.app.BuildConfig
import com.hotspot.app.util.ChatDrafts
import com.hotspot.app.data.update.AppUpdateInfo
import com.hotspot.app.data.update.UpdateChecker
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
import com.hotspot.app.ui.components.SegmentedTabs
import com.hotspot.app.ui.components.SystemCard
import com.hotspot.app.ui.components.SystemInput
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
    onOpenShares: () -> Unit,
    onOpenBlocked: () -> Unit,
    onOpenSessions: () -> Unit,
    onOpenAdmin: () -> Unit,
    onOpenMap: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    val themeController = AppContainer.themeController
    val uiTheme by themeController.state.collectAsStateWithLifecycle()
    val scope = rememberCoroutineScope()
    val lifecycleOwner = LocalLifecycleOwner.current
    var anchor by remember { mutableStateOf(AppContainer.browseAnchorStore.current()) }
    var fallbackLocationName by remember { mutableStateOf<String?>(null) }
    var account by remember { mutableStateOf<AccountData?>(null) }
    var showLogoutConfirm by remember { mutableStateOf(false) }
    var loggingOut by remember { mutableStateOf(false) }
    val updateChecker = remember { UpdateChecker() }
    var updateChecking by remember { mutableStateOf(false) }
    var updateInfo by remember { mutableStateOf<AppUpdateInfo?>(null) }
    var showUpgrade by remember { mutableStateOf(false) }
    var upgradeLoginId by remember { mutableStateOf("") }
    var upgradePassword by remember { mutableStateOf("") }
    var upgradeConfirm by remember { mutableStateOf("") }
    var upgradeError by remember { mutableStateOf<String?>(null) }
    var upgradeBusy by remember { mutableStateOf(false) }
    var locationSyncing by remember { mutableStateOf(false) }
    var privacyLevel by remember { mutableStateOf("public") }
    var locationVisible by remember { mutableStateOf(true) }
    var nearbyMomentNotify by remember { mutableStateOf(false) }
    var searchable by remember { mutableStateOf(true) }
    var allowPrivateChat by remember { mutableStateOf(true) }
    var allowFollow by remember { mutableStateOf(true) }
    var presenceVisible by remember { mutableStateOf(true) }
    var socialListsVisible by remember { mutableStateOf(true) }
    var groupsVisible by remember { mutableStateOf(true) }
    var reactedMomentsVisible by remember { mutableStateOf(true) }
    var statsVisible by remember { mutableStateOf(true) }
    var privacyLoaded by remember { mutableStateOf(false) }
    var privacyUpdating by remember { mutableStateOf(false) }
    var isAdmin by remember { mutableStateOf(false) }
    var showChangePassword by remember { mutableStateOf(false) }
    var oldPassword by remember { mutableStateOf("") }
    var newPassword by remember { mutableStateOf("") }
    var confirmPassword by remember { mutableStateOf("") }
    var changingPassword by remember { mutableStateOf(false) }
    var passwordError by remember { mutableStateOf<String?>(null) }
    var showDeleteAccount by remember { mutableStateOf(false) }
    var deleteConfirmation by remember { mutableStateOf("") }
    var deletingAccount by remember { mutableStateOf(false) }
    var deleteError by remember { mutableStateOf<String?>(null) }

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
        when (val settings = AppContainer.profileRepository.settings()) {
            is ApiResult.Ok -> {
                privacyLevel = settings.data.privacyLevel
                locationVisible = settings.data.locationVisible
                nearbyMomentNotify = settings.data.nearbyMomentNotify
                searchable = settings.data.searchable
                allowPrivateChat = settings.data.allowPrivateChat
                allowFollow = settings.data.allowFollow
                presenceVisible = settings.data.presenceVisible
                socialListsVisible = settings.data.socialListsVisible
                groupsVisible = settings.data.groupsVisible
                reactedMomentsVisible = settings.data.reactedMomentsVisible
                statsVisible = settings.data.statsVisible
            }
            is ApiResult.Err -> Toast.makeText(context, settings.error.userMessage, Toast.LENGTH_SHORT).show()
        }
        privacyLoaded = true
    }

    fun updatePrivacy(
        privacy: String? = null,
        location: Boolean? = null,
        nearbyNotify: Boolean? = null,
        searchableValue: Boolean? = null,
        privateChatValue: Boolean? = null,
        followValue: Boolean? = null,
        presenceValue: Boolean? = null,
        socialListsValue: Boolean? = null,
        groupsValue: Boolean? = null,
        reactedMomentsValue: Boolean? = null,
        statsValue: Boolean? = null,
    ) {
        if (!privacyLoaded || privacyUpdating) return
        privacyUpdating = true
        scope.launch {
            when (val result = AppContainer.profileRepository.updateSettings(
                privacyLevel = privacy,
                locationVisible = location,
                nearbyMomentNotify = nearbyNotify,
                searchable = searchableValue,
                allowPrivateChat = privateChatValue,
                allowFollow = followValue,
                presenceVisible = presenceValue,
                socialListsVisible = socialListsValue,
                groupsVisible = groupsValue,
                reactedMomentsVisible = reactedMomentsValue,
                statsVisible = statsValue,
            )) {
                is ApiResult.Ok -> {
                    if (privacy != null) privacyLevel = privacy
                    if (location != null) locationVisible = location
                    if (nearbyNotify != null) nearbyMomentNotify = nearbyNotify
                    if (searchableValue != null) searchable = searchableValue
                    if (privateChatValue != null) allowPrivateChat = privateChatValue
                    if (followValue != null) allowFollow = followValue
                    if (presenceValue != null) presenceVisible = presenceValue
                    if (socialListsValue != null) socialListsVisible = socialListsValue
                    if (groupsValue != null) groupsVisible = groupsValue
                    if (reactedMomentsValue != null) reactedMomentsVisible = reactedMomentsValue
                    if (statsValue != null) statsVisible = statsValue
                    Toast.makeText(context, "设置已更新", Toast.LENGTH_SHORT).show()
                }
                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
            }
            privacyUpdating = false
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
                            if (account?.userType == "temp") {
                                Badge(text = "访客", tone = BadgeTone.WARNING)
                            } else {
                                Badge(text = "已登录", tone = BadgeTone.ONLINE)
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
            SettingsSectionTitle(title = "账号安全", subtitle = "修改密码或注销账号，操作立即生效")
            Spacer(Modifier.height(Spacing.sm))
            SystemCard(contentPadding = PaddingValues(vertical = Spacing.xs)) {
                if (account?.userType == "temp") {
                    ListRow(
                        title = "升级为正式账号",
                        subtitle = "设置用户名与密码，保留访客期间的内容",
                        onClick = {
                            upgradeLoginId = ""
                            upgradePassword = ""
                            upgradeConfirm = ""
                            upgradeError = null
                            showUpgrade = true
                        },
                    )
                }
                ListRow(
                    title = "修改密码",
                    subtitle = "验证当前密码后设置新密码，其他设备会自动退出",
                    onClick = {
                        oldPassword = ""
                        newPassword = ""
                        confirmPassword = ""
                        passwordError = null
                        showChangePassword = true
                    },
                )
                ListRow(
                    title = "登录设备",
                    subtitle = "查看活跃设备，退出其他设备",
                    onClick = onOpenSessions,
                )
                ListRow(
                    title = "删除账号",
                    subtitle = "永久注销账号，公开内容将匿名化保留",
                    onClick = {
                        deleteConfirmation = ""
                        deleteError = null
                        showDeleteAccount = true
                    },
                )
            }
            Spacer(Modifier.height(Spacing.xl))
            SettingsSectionTitle(title = "隐私与安全", subtitle = "控制个人主页可见性、位置共享与账号保护")
            Spacer(Modifier.height(Spacing.sm))
            SystemCard(contentPadding = PaddingValues(Spacing.md)) {
                Text(
                    text = "账号可见性",
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                    color = colors.textPrimary,
                )
                Spacer(Modifier.height(Spacing.xs))
                Text(
                    text = if (privacyLevel == "private") "仅关注者可查看你的主页与瞬间" else "所有用户可查看你的公开主页与瞬间",
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
                Spacer(Modifier.height(Spacing.sm))
                SegmentedTabs(
                    options = listOf("公开", "私密"),
                    selectedIndex = if (privacyLevel == "private") 1 else 0,
                    onSelected = { index -> updatePrivacy(privacy = if (index == 1) "private" else "public") },
                    modifier = Modifier.fillMaxWidth(),
                )
                if (!privacyLoaded || privacyUpdating) {
                    Spacer(Modifier.height(Spacing.xs))
                    Text(
                        text = if (!privacyLoaded) "加载隐私设置…" else "保存中…",
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textSecondary,
                    )
                }
                Spacer(Modifier.height(Spacing.lg))
                Text(
                    text = "位置共享",
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                    color = colors.textPrimary,
                )
                Spacer(Modifier.height(Spacing.xs))
                Text(
                    text = if (locationVisible) "附近用户可在发现与地图中看到你" else "你的位置不会出现在附近与地图中",
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
                Spacer(Modifier.height(Spacing.sm))
                SegmentedTabs(
                    options = listOf("共享", "隐藏"),
                    selectedIndex = if (locationVisible) 0 else 1,
                    onSelected = { index -> updatePrivacy(location = index == 0) },
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(Spacing.lg))
                Text(
                    text = "附近新瞬间通知",
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                    color = colors.textPrimary,
                )
                Spacer(Modifier.height(Spacing.xs))
                Text(
                    text = if (nearbyMomentNotify) "附近出现新瞬间时会通知你" else "不会收到附近新瞬间通知",
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
                Spacer(Modifier.height(Spacing.sm))
                SegmentedTabs(
                    options = listOf("开启", "关闭"),
                    selectedIndex = if (nearbyMomentNotify) 0 else 1,
                    onSelected = { index -> updatePrivacy(nearbyNotify = index == 0) },
                    modifier = Modifier.fillMaxWidth(),
                )
            }
            Spacer(Modifier.height(Spacing.xl))
            SettingsSectionTitle(title = "互动与发现", subtitle = "控制他人能否搜索到你、关注你或给你发私信")
            Spacer(Modifier.height(Spacing.sm))
            SystemCard(contentPadding = PaddingValues(Spacing.md)) {
                Text(
                    text = "允许被搜索",
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                    color = colors.textPrimary,
                )
                Spacer(Modifier.height(Spacing.xs))
                Text(
                    text = if (searchable) "其他用户可通过昵称搜索到你" else "你不会出现在搜索结果中",
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
                Spacer(Modifier.height(Spacing.sm))
                SegmentedTabs(
                    options = listOf("允许", "禁止"),
                    selectedIndex = if (searchable) 0 else 1,
                    onSelected = { index -> updatePrivacy(searchableValue = index == 0) },
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(Spacing.lg))
                Text(
                    text = "允许关注",
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                    color = colors.textPrimary,
                )
                Spacer(Modifier.height(Spacing.xs))
                Text(
                    text = if (allowFollow) "他人可以关注你并接收你的公开动态" else "他人无法关注你",
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
                Spacer(Modifier.height(Spacing.sm))
                SegmentedTabs(
                    options = listOf("允许", "禁止"),
                    selectedIndex = if (allowFollow) 0 else 1,
                    onSelected = { index -> updatePrivacy(followValue = index == 0) },
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(Spacing.lg))
                Text(
                    text = "允许私信",
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                    color = colors.textPrimary,
                )
                Spacer(Modifier.height(Spacing.xs))
                Text(
                    text = if (allowPrivateChat) "其他用户可给你发送私信" else "他人无法给你发送私信",
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
                Spacer(Modifier.height(Spacing.sm))
                SegmentedTabs(
                    options = listOf("允许", "禁止"),
                    selectedIndex = if (allowPrivateChat) 0 else 1,
                    onSelected = { index -> updatePrivacy(privateChatValue = index == 0) },
                    modifier = Modifier.fillMaxWidth(),
                )
            }
            Spacer(Modifier.height(Spacing.xl))
            SettingsSectionTitle(title = "资料展示", subtitle = "控制个人主页上哪些信息对他人可见")
            Spacer(Modifier.height(Spacing.sm))
            SystemCard(contentPadding = PaddingValues(Spacing.md)) {
                VisibilityToggle(
                    title = "在线状态",
                    description = if (presenceVisible) "他人可看到你的在线状态与最近活跃时间" else "你的在线状态与最近活跃时间对他人隐藏",
                    visible = presenceVisible,
                    onSelected = { index -> updatePrivacy(presenceValue = index == 0) },
                )
                Spacer(Modifier.height(Spacing.lg))
                VisibilityToggle(
                    title = "关系列表",
                    description = if (socialListsVisible) "他人可查看你的好友、关注与粉丝列表" else "你的好友、关注与粉丝列表仅自己可见",
                    visible = socialListsVisible,
                    onSelected = { index -> updatePrivacy(socialListsValue = index == 0) },
                )
                Spacer(Modifier.height(Spacing.lg))
                VisibilityToggle(
                    title = "群组列表",
                    description = if (groupsVisible) "他人可查看你加入的群组" else "你加入的群组仅自己可见",
                    visible = groupsVisible,
                    onSelected = { index -> updatePrivacy(groupsValue = index == 0) },
                )
                Spacer(Modifier.height(Spacing.lg))
                VisibilityToggle(
                    title = "反应过的瞬间",
                    description = if (reactedMomentsVisible) "他人可查看你点赞过的瞬间" else "你点赞过的瞬间仅自己可见",
                    visible = reactedMomentsVisible,
                    onSelected = { index -> updatePrivacy(reactedMomentsValue = index == 0) },
                )
                Spacer(Modifier.height(Spacing.lg))
                VisibilityToggle(
                    title = "统计信息",
                    description = if (statsVisible) "他人可查看你的瞬间/粉丝/关注统计" else "你的统计数据对他人隐藏",
                    visible = statsVisible,
                    onSelected = { index -> updatePrivacy(statsValue = index == 0) },
                )
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
                Spacer(Modifier.height(Spacing.sm))
                SystemButton(
                    text = if (locationSyncing) "正在更新位置…" else "同步当前位置",
                    onClick = {
                        if (locationSyncing) return@SystemButton
                        scope.launch {
                            val current = anchor
                            val fallback = AppContainer.defaultLocation()
                            val latitude = current?.latitude ?: fallback?.point?.latitude
                            val longitude = current?.longitude ?: fallback?.point?.longitude
                            if (latitude == null || longitude == null) {
                                Toast.makeText(context, "暂无可同步的位置", Toast.LENGTH_SHORT).show()
                                return@launch
                            }
                            val label = current?.label
                                ?: fallback?.locationName?.ifBlank { null }
                                ?: "当前位置"
                            locationSyncing = true
                            when (val result = AppContainer.profileRepository.updateLocation(latitude, longitude, label)) {
                                is ApiResult.Ok -> Toast.makeText(context, "位置已更新", Toast.LENGTH_SHORT).show()
                                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                            }
                            locationSyncing = false
                        }
                    },
                    variant = ButtonVariant.PRIMARY,
                    size = ButtonSize.SM,
                    loading = locationSyncing,
                    enabled = !locationSyncing,
                )
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
                    title = "检查更新",
                    subtitle = if (updateChecking) "正在检查新版本…" else "当前版本 " + BuildConfig.VERSION_NAME + " · 点击检查",
                    onClick = {
                        if (!updateChecking) {
                            scope.launch {
                                updateChecking = true
                                updateChecker.check(BuildConfig.VERSION_NAME)
                                    .onSuccess { info ->
                                        if (info.isNewer) {
                                            updateInfo = info
                                        } else {
                                            Toast.makeText(context, "已是最新版本", Toast.LENGTH_SHORT).show()
                                        }
                                    }
                                    .onFailure { error ->
                                        Toast.makeText(context, error.message ?: "更新检查失败", Toast.LENGTH_SHORT).show()
                                    }
                                updateChecking = false
                            }
                        }
                    },
                )
                ListRow(
                    title = "我的分享",
                    subtitle = "管理已创建的分享链接与撤销记录",
                    onClick = onOpenShares,
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

    if (showChangePassword) {
        val ready = oldPassword.isNotBlank() && newPassword.length >= 8 &&
            newPassword == confirmPassword && !changingPassword
        AlertDialog(
            onDismissRequest = { if (!changingPassword) showChangePassword = false },
            title = { Text("修改密码") },
            text = {
                Column {
                    Text(
                        text = "修改成功后，当前设备保持登录，其他设备会自动退出。",
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textSecondary,
                    )
                    Spacer(Modifier.height(Spacing.md))
                    SystemInput(
                        value = oldPassword,
                        onValueChange = { oldPassword = it; passwordError = null },
                        label = "当前密码",
                        visualTransformation = PasswordVisualTransformation(),
                        enabled = !changingPassword,
                    )
                    Spacer(Modifier.height(Spacing.md))
                    SystemInput(
                        value = newPassword,
                        onValueChange = { newPassword = it; passwordError = null },
                        label = "新密码（至少 8 位）",
                        visualTransformation = PasswordVisualTransformation(),
                        enabled = !changingPassword,
                    )
                    Spacer(Modifier.height(Spacing.md))
                    SystemInput(
                        value = confirmPassword,
                        onValueChange = { confirmPassword = it; passwordError = null },
                        label = "确认新密码",
                        visualTransformation = PasswordVisualTransformation(),
                        enabled = !changingPassword,
                        errorText = if (confirmPassword.isNotEmpty() && confirmPassword != newPassword) "两次输入不一致" else null,
                    )
                    if (passwordError != null) {
                        Spacer(Modifier.height(Spacing.sm))
                        Text(
                            text = passwordError.orEmpty(),
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.danger,
                        )
                    }
                }
            },
            confirmButton = {
                SystemButton(
                    text = if (changingPassword) "保存中…" else "保存",
                    onClick = {
                        if (!ready) return@SystemButton
                        scope.launch {
                            changingPassword = true
                            when (val result = AppContainer.authRepository.changePassword(oldPassword, newPassword)) {
                                is ApiResult.Ok -> {
                                    showChangePassword = false
                                    Toast.makeText(context, "密码已修改，其他设备已退出登录", Toast.LENGTH_SHORT).show()
                                }
                                is ApiResult.Err -> passwordError = result.error.userMessage
                            }
                            changingPassword = false
                        }
                    },
                    variant = ButtonVariant.PRIMARY,
                    size = ButtonSize.SM,
                    enabled = ready,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { if (!changingPassword) showChangePassword = false },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !changingPassword,
                )
            },
        )
    }

    if (showDeleteAccount) {
        val ready = deleteConfirmation == "DELETE_MY_ACCOUNT" && !deletingAccount
        AlertDialog(
            onDismissRequest = { if (!deletingAccount) showDeleteAccount = false },
            title = { Text("删除账号？") },
            text = {
                Column {
                    Text(
                        text = "删除后不可恢复：登录凭证、私聊记录、收藏与关注关系会被清除；你发布的公开内容会匿名化保留。",
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textSecondary,
                    )
                    Spacer(Modifier.height(Spacing.md))
                    SystemInput(
                        value = deleteConfirmation,
                        onValueChange = { deleteConfirmation = it; deleteError = null },
                        label = "输入 DELETE_MY_ACCOUNT 确认",
                        placeholder = "DELETE_MY_ACCOUNT",
                        enabled = !deletingAccount,
                    )
                    if (deleteError != null) {
                        Spacer(Modifier.height(Spacing.sm))
                        Text(
                            text = deleteError.orEmpty(),
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.danger,
                        )
                    }
                }
            },
            confirmButton = {
                SystemButton(
                    text = if (deletingAccount) "注销中…" else "永久删除",
                    onClick = {
                        if (!ready) return@SystemButton
                        scope.launch {
                            deletingAccount = true
                            when (val result = AppContainer.authRepository.deleteAccount(deleteConfirmation)) {
                                is ApiResult.Ok -> {
                                    showDeleteAccount = false
                                    AppContainer.tokenStore.clear()
                                    Toast.makeText(context, "账号已注销", Toast.LENGTH_SHORT).show()
                                }
                                is ApiResult.Err -> deleteError = result.error.userMessage
                            }
                            deletingAccount = false
                        }
                    },
                    variant = ButtonVariant.DANGER,
                    size = ButtonSize.SM,
                    enabled = ready,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { if (!deletingAccount) showDeleteAccount = false },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !deletingAccount,
                )
            },
        )
    }

    if (showUpgrade) {
        AlertDialog(
            onDismissRequest = { if (!upgradeBusy) showUpgrade = false },
            title = { Text("升级为正式账号") },
            text = {
                Column {
                    Text(
                        text = "设置用户名与密码后，当前访客账号及内容会保留，并可用该凭证重新登录。",
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.textSecondary,
                    )
                    Spacer(Modifier.height(Spacing.md))
                    SystemInput(
                        value = upgradeLoginId,
                        onValueChange = { upgradeLoginId = it; upgradeError = null },
                        label = "用户名",
                        placeholder = "暂不支持手机号/邮箱",
                        enabled = !upgradeBusy,
                    )
                    Spacer(Modifier.height(Spacing.sm))
                    SystemInput(
                        value = upgradePassword,
                        onValueChange = { upgradePassword = it; upgradeError = null },
                        label = "密码",
                        placeholder = "至少 8 位，包含字母和数字",
                        enabled = !upgradeBusy,
                        visualTransformation = PasswordVisualTransformation(),
                    )
                    Spacer(Modifier.height(Spacing.sm))
                    SystemInput(
                        value = upgradeConfirm,
                        onValueChange = { upgradeConfirm = it; upgradeError = null },
                        label = "确认密码",
                        placeholder = "再次输入密码",
                        enabled = !upgradeBusy,
                        visualTransformation = PasswordVisualTransformation(),
                    )
                    upgradeError?.let { message ->
                        Spacer(Modifier.height(Spacing.sm))
                        Text(
                            text = message,
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.danger,
                        )
                    }
                }
            },
            confirmButton = {
                SystemButton(
                    text = if (upgradeBusy) "升级中…" else "确认升级",
                    onClick = {
                        if (upgradeBusy) return@SystemButton
                        if (upgradeLoginId.isBlank()) {
                            upgradeError = "请输入用户名"
                            return@SystemButton
                        }
                        if (upgradePassword.length < 8) {
                            upgradeError = "密码至少 8 位，且包含字母和数字"
                            return@SystemButton
                        }
                        if (upgradePassword != upgradeConfirm) {
                            upgradeError = "两次输入的密码不一致"
                            return@SystemButton
                        }
                        scope.launch {
                            upgradeBusy = true
                            upgradeError = null
                            when (val result = AppContainer.profileRepository.upgradeAccount(
                                loginId = upgradeLoginId.trim(),
                                password = upgradePassword,
                            )) {
                                is ApiResult.Ok -> {
                                    account = result.data
                                    showUpgrade = false
                                    Toast.makeText(context, "已升级为正式账号", Toast.LENGTH_SHORT).show()
                                }
                                is ApiResult.Err -> upgradeError = result.error.userMessage
                            }
                            upgradeBusy = false
                        }
                    },
                    variant = ButtonVariant.PRIMARY,
                    size = ButtonSize.SM,
                    enabled = !upgradeBusy,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { if (!upgradeBusy) showUpgrade = false },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !upgradeBusy,
                )
            },
        )
    }

    updateInfo?.takeIf { it.isNewer }?.let { info ->
        AlertDialog(
            onDismissRequest = { updateInfo = null },
            title = { Text("发现新版本 v" + info.latestVersion) },
            text = {
                Text(
                    text = "当前版本 " + info.currentVersion + " → 最新版本 " + info.latestVersion +
                        "\n\n" + releaseHighlights(info.notes),
                )
            },
            confirmButton = {
                SystemButton(
                    text = "去下载",
                    onClick = {
                        val url = info.apkUrl?.takeIf { it.isNotBlank() } ?: info.pageUrl
                        if (url.isNotBlank()) {
                            try {
                                context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                            } catch (_: ActivityNotFoundException) {
                                Toast.makeText(context, "没有可用的浏览器", Toast.LENGTH_SHORT).show()
                            }
                        }
                        updateInfo = null
                    },
                    variant = ButtonVariant.PRIMARY,
                    size = ButtonSize.SM,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "稍后",
                    onClick = { updateInfo = null },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                )
            },
        )
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
                            ChatDrafts.clearAll(context)
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

/** 从 GitHub Release 正文提炼变更条目：优先「What's Changed」下的 bullet，过滤链接与署名。 */
private fun releaseHighlights(notes: String): String {
    val lines = notes.lineSequence().map { it.trim() }.toList()
    val start = lines.indexOfFirst { it.startsWith("## What's Changed") }
    val relevant = if (start >= 0) lines.drop(start + 1) else lines
    val bullets = relevant
        .filter { it.startsWith("* ") || it.startsWith("- ") }
        .map { it.removePrefix("* ").removePrefix("- ").substringBefore(" by @").trim() }
        .filter { it.isNotBlank() && !it.startsWith("http") }
        .take(6)
    return if (bullets.isEmpty()) {
        "建议更新到最新版本，获得最新功能与修复。"
    } else {
        bullets.joinToString("\n") { "• " + it }
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
private fun VisibilityToggle(
    title: String,
    description: String,
    visible: Boolean,
    onSelected: (Int) -> Unit,
) {
    val colors = HotSpotTheme.colors
    Text(
        text = title,
        style = MaterialTheme.typography.titleSmall,
        fontWeight = FontWeight.SemiBold,
        color = colors.textPrimary,
    )
    Spacer(Modifier.height(Spacing.xs))
    Text(
        text = description,
        style = MaterialTheme.typography.labelSmall,
        color = colors.textSecondary,
    )
    Spacer(Modifier.height(Spacing.sm))
    SegmentedTabs(
        options = listOf("公开", "隐藏"),
        selectedIndex = if (visible) 0 else 1,
        onSelected = onSelected,
        modifier = Modifier.fillMaxWidth(),
    )
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
