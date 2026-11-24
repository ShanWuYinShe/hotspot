package com.hotspot.app.ui.screens

import androidx.compose.foundation.ScrollState
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.WindowInsetsSides
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.only
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.Notifications
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
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.role
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import android.widget.Toast
import coil.compose.AsyncImage
import coil.request.ImageRequest
import com.hotspot.app.AppContainer
import com.hotspot.app.R
import com.hotspot.app.data.api.ApiError
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.AdminBannedMemberItem
import com.hotspot.app.data.model.GroupAnnouncementItem
import com.hotspot.app.data.model.GroupApplicationItem
import com.hotspot.app.data.model.GroupItem
import com.hotspot.app.data.model.GroupMemberItem
import com.hotspot.app.data.model.UserItem
import com.hotspot.app.data.model.GroupSentInviteItem
import com.hotspot.app.data.model.GroupSettings
import com.hotspot.app.data.model.Page
import com.hotspot.app.ui.components.AnnouncementEditorDialog
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.Badge
import com.hotspot.app.ui.components.BannedMembersDialog
import com.hotspot.app.ui.components.ReportDialog
import com.hotspot.app.ui.components.BadgeTone
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.GroupApplicationsDialog
import com.hotspot.app.ui.components.GroupSentInvitesDialog
import com.hotspot.app.ui.components.GroupInfoEditorDialog
import com.hotspot.app.ui.components.InviteMemberDialog
import com.hotspot.app.ui.components.GroupSettingsDialog
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.InlineLoading
import com.hotspot.app.ui.components.ListEnd
import com.hotspot.app.ui.components.ListGroup
import com.hotspot.app.ui.components.ListRow
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.MemberAction
import com.hotspot.app.ui.components.MemberActionSheet
import com.hotspot.app.ui.components.MediaBadge
import com.hotspot.app.ui.components.ReportDialog
import com.hotspot.app.ui.components.SearchInput
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemInput
import com.hotspot.app.ui.components.TabPills
import com.hotspot.app.ui.components.hotSpotFocusRing
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTextStyles
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.OpacityTokens
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.ui.theme.Tokens
import com.hotspot.app.util.MediaUrls
import com.hotspot.app.util.TimeFormat
import com.hotspot.app.util.canManageGroupAnnouncements
import com.hotspot.app.util.formatDistanceShort
import com.hotspot.app.util.groupRoleLabel
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.util.UUID

/*
 * 群详情（推入页 group/{groupId}，设计稿移动端画板 87:1696，路由示例 /groups/group_seed_1）。
 * 数据契约：spec/api_spec_group.md（open_the_door / enter / exit / members/fetch）+
 * spec/api_spec_public.md（announcements/fetch：仅 public 版、请求只有 group_id、响应无分页）。
 *
 * 设计几何常量（390 基线，取值来自 Figma get_metadata 87:1696，单位 dp/sp）：
 * - 封面卡 358x220 = 封面可见高 156 + 白底群名带 64，四角圆角 20（设计稿白卡底部同样收圆）
 * - 顶部导航 42 圆钮（返回 / 更多），左右 16 内距，卡片上方 12 留白
 * - 指标数字 20 semibold / 行高 24，标签 11 次级；「在线」数值用 status-online 色（设计稿为绿色 2）
 * - 活跃成员头像 32 + 重叠 8（步进 24），首屏最多 7 枚；「查看全部 >」11 accent
 * - 底部操作卡 358 宽、16 内距：按钮 44 高 + 说明 11（未加入态带 13 铃铛图标）
 * 其余间距走 Spacing 刻度（28/24/12/8），不引入近邻档。
 */
private val GroupCoverHeight = 156.dp
private val GroupNameBandHeight = 64.dp
private val GroupHeaderCardHeight = GroupCoverHeight + GroupNameBandHeight
private val GroupNavButtonSize = Dimens.backButton
private val GroupAvatarStackSize = 32.dp
private val GroupAvatarStackStep = 24.dp
private val GroupAvatarRingSize = 36.dp
private val GroupHeaderToTabsGap = 28.dp
private val GroupActionButtonHeight = 44.dp
private val GroupStatNumberSize = Tokens.FontSize.xl.sp
private val GroupStatNumberLineHeight = 24.sp
private val GroupStatLabelSize = Tokens.FontSize.xs.sp
private val GroupStatLabelLineHeight = 15.sp
private val GroupNameSize = Tokens.FontSize.xxl.sp
private val GroupNameLineHeight = 32.sp
private val GroupActionTipSize = Tokens.FontSize.xs.sp
private val GroupActionTipLineHeight = 20.sp
private const val ACTIVE_MEMBER_AVATARS = 7
private const val ANNOUNCEMENT_PREVIEW_COUNT = 2
private const val MEMBER_LOAD_MORE_THRESHOLD = 240
private const val ENTER_STATUS_JOINED = "joined"
private const val ENTER_STATUS_PENDING = "pending_approval"

private val GROUP_TABS = listOf("简介", "成员", "公告")

/**
 * 群详情页：封面卡（头像铺满 / 无头像用主题软底 + 群名首字）+ 简介 / 成员 / 公告三 tab +
 * docked 加入操作区。右上「更多」仅在已加入时可展开（退出群组）。
 *
 * 进入页面并行加载三块（群详情 / 成员首屏 / 公告首屏），各块独立 loading 与错误、互不阻塞。
 * 加入/退出真实调 API：password 群先弹口令 Dialog；pending_approval 落「已申请，等待审核」禁用态；
 * 退出走二次确认 Dialog（群主会被服务端 10400 拒绝，弹窗内透传 message）。
 * Android 首版无群聊会话页，因此不放「打开群聊」入口（不伪造按钮）。
 */
@Composable
fun GroupDetailScreen(
    groupId: String,
    onBack: () -> Unit,
    onOpenGroupChat: (GroupItem) -> Unit = {},
) {
    val scope = rememberCoroutineScope()
    val repository = AppContainer.groupRepository
    val scrollState = rememberScrollState()

    var detail by remember(groupId) { mutableStateOf<GroupItem?>(null) }
    var detailLoading by remember(groupId) { mutableStateOf(true) }
    var detailError by remember(groupId) { mutableStateOf<String?>(null) }

    val membersState = rememberPagedListState<GroupMemberItem>()
    val memberLoader: suspend (String?) -> ApiResult<Page<GroupMemberItem>> = { cursor ->
        repository.members(groupId, cursor)
    }
    var memberQuery by remember(groupId) { mutableStateOf("") }
    val memberSearchState = rememberPagedListState<GroupMemberItem>()
    val memberSearchLoader: suspend (String?) -> ApiResult<Page<GroupMemberItem>> = { cursor ->
        repository.searchMembers(groupId, memberQuery.trim(), cursor)
    }

    LaunchedEffect(memberQuery) {
        if (memberQuery.isBlank()) {
            memberSearchState.reset()
            return@LaunchedEffect
        }
        delay(350)
        memberSearchState.initialLoad(memberSearchLoader)
    }

    var announcements by remember(groupId) { mutableStateOf<List<GroupAnnouncementItem>?>(null) }
    var announcementsLoading by remember(groupId) { mutableStateOf(true) }
    var announcementsError by remember(groupId) { mutableStateOf<String?>(null) }

    var tabIndex by remember(groupId) { mutableIntStateOf(0) }
    var expandedAnnouncementId by remember(groupId) { mutableStateOf<String?>(null) }

    var joining by remember(groupId) { mutableStateOf(false) }
    var joinPending by remember(groupId) { mutableStateOf(false) }
    var actionError by remember(groupId) { mutableStateOf<String?>(null) }
    var showPasswordDialog by remember(groupId) { mutableStateOf(false) }
    var passwordInput by remember(groupId) { mutableStateOf("") }
    var passwordError by remember(groupId) { mutableStateOf<String?>(null) }
    var showExitDialog by remember(groupId) { mutableStateOf(false) }
    var exiting by remember(groupId) { mutableStateOf(false) }
    var exitError by remember(groupId) { mutableStateOf<String?>(null) }
    var showReportDialog by remember(groupId) { mutableStateOf(false) }
    var reportSubmitting by remember(groupId) { mutableStateOf(false) }
    // 群公告管理（owner/admin）：发布/编辑弹窗 + 删除确认
    var showAnnouncementEditor by remember(groupId) { mutableStateOf(false) }
    var editingAnnouncement by remember(groupId) { mutableStateOf<GroupAnnouncementItem?>(null) }
    var announcementSubmitting by remember(groupId) { mutableStateOf(false) }
    var announcementEditorError by remember(groupId) { mutableStateOf<String?>(null) }
    // 发布公告的幂等键：同一弹窗内的重试/双击复用，成功或取消后作废（spec：create MUST 幂等）。
    var announcementCreateKey by remember(groupId) { mutableStateOf<String?>(null) }
    var deletingAnnouncement by remember(groupId) { mutableStateOf<GroupAnnouncementItem?>(null) }
    var announcementDeleting by remember(groupId) { mutableStateOf(false) }
    // 群信息编辑（owner/admin）：名称 + 简介
    var showGroupInfoEditor by remember(groupId) { mutableStateOf(false) }
    var groupInfoSaving by remember(groupId) { mutableStateOf(false) }
    var groupInfoError by remember(groupId) { mutableStateOf<String?>(null) }
    // 群设置（owner/admin）：搜索可见性、成员可搜索、入群审批、人数上限
    var showGroupSettings by remember(groupId) { mutableStateOf(false) }
    var groupSettings by remember(groupId) { mutableStateOf<GroupSettings?>(null) }
    var groupSettingsLoading by remember(groupId) { mutableStateOf(false) }
    var groupSettingsError by remember(groupId) { mutableStateOf<String?>(null) }
    var groupSettingsSaveError by remember(groupId) { mutableStateOf<String?>(null) }
    var groupSettingsSaving by remember(groupId) { mutableStateOf(false) }
    // 入群申请（群收件箱）
    var showApplications by remember(groupId) { mutableStateOf(false) }
    var applications by remember(groupId) { mutableStateOf<List<GroupApplicationItem>?>(null) }
    var applicationsLoading by remember(groupId) { mutableStateOf(false) }
    var applicationsError by remember(groupId) { mutableStateOf<String?>(null) }
    var applicationProcessingId by remember(groupId) { mutableStateOf<String?>(null) }
    // 邀请成员（owner/admin）：搜索用户 → 发送邀请
    var showInvite by remember(groupId) { mutableStateOf(false) }
    var inviteQuery by remember(groupId) { mutableStateOf("") }
    var inviteResults by remember(groupId) { mutableStateOf<List<UserItem>>(emptyList()) }
    var inviteSearching by remember(groupId) { mutableStateOf(false) }
    var inviteError by remember(groupId) { mutableStateOf<String?>(null) }
    var invitedIds by remember(groupId) { mutableStateOf<Set<String>>(emptySet()) }
    var invitingUserId by remember(groupId) { mutableStateOf<String?>(null) }
    // 邀请记录（owner/admin）：展示全部状态，待接受可撤销
    var showSentInvites by remember(groupId) { mutableStateOf(false) }
    var sentInvites by remember(groupId) { mutableStateOf<List<GroupSentInviteItem>?>(null) }
    var sentInvitesLoading by remember(groupId) { mutableStateOf(false) }
    var sentInvitesError by remember(groupId) { mutableStateOf<String?>(null) }
    var revokeInviteTarget by remember(groupId) { mutableStateOf<GroupSentInviteItem?>(null) }
    var revokingInviteId by remember(groupId) { mutableStateOf<String?>(null) }
    // 封禁/解封（owner/admin）
    var banningMember by remember(groupId) { mutableStateOf<GroupMemberItem?>(null) }
    var banBusy by remember(groupId) { mutableStateOf(false) }
    var showBanned by remember(groupId) { mutableStateOf(false) }
    var bannedMembers by remember(groupId) { mutableStateOf<List<AdminBannedMemberItem>?>(null) }
    var bannedLoading by remember(groupId) { mutableStateOf(false) }
    var bannedError by remember(groupId) { mutableStateOf<String?>(null) }
    var unbanningId by remember(groupId) { mutableStateOf<String?>(null) }
    // 群成员管理（owner/admin）：角色 / 禁言 / 移除
    var managedMember by remember(groupId) { mutableStateOf<GroupMemberItem?>(null) }
    var removingMember by remember(groupId) { mutableStateOf<GroupMemberItem?>(null) }
    var memberActionBusy by remember(groupId) { mutableStateOf(false) }
    var transferTarget by remember(groupId) { mutableStateOf<GroupMemberItem?>(null) }
    var transferBusy by remember(groupId) { mutableStateOf(false) }
    val myUserId = AppContainer.tokenStore.session.value?.userId
    val context = LocalContext.current

    suspend fun loadDetail() {
        detailLoading = detail == null
        detailError = null
        when (val result = repository.open(groupId)) {
            is ApiResult.Ok -> detail = result.data
            is ApiResult.Err -> detailError = result.error.userMessage
        }
        detailLoading = false
    }

    suspend fun loadAnnouncements() {
        announcementsLoading = true
        announcementsError = null
        when (val result = repository.announcements(groupId)) {
            is ApiResult.Ok -> announcements = result.data
            is ApiResult.Err -> announcementsError = result.error.userMessage
        }
        announcementsLoading = false
    }

    /** 公告写操作成功后并行刷新列表与详情（详情里的公告计数同步）。 */
    suspend fun refreshAfterAnnouncementChange() {
        coroutineScope {
            launch { loadAnnouncements() }
            launch { loadDetail() }
        }
    }

    /** 加入/退出成功后刷新「详情 + 成员」：viewer_state 与成员规模以服务端真值决定，不在本地伪造。 */
    suspend fun refreshAfterMembershipChange() {
        loadDetail()
        if (detailError != null) actionError = detailError
        membersState.initialLoad(memberLoader)
        if (memberQuery.isNotBlank()) memberSearchState.initialLoad(memberSearchLoader)
    }

    LaunchedEffect(groupId) {
        // 三块并行加载（任务契约）：各块独立出 loading/错误，互不阻塞
        coroutineScope {
            launch { membersState.initialLoad(memberLoader) }
            launch { loadAnnouncements() }
            launch { loadDetail() }
        }
    }

    // 触底续载成员页（仅成员 tab；内容不足一屏时继续补页，直到到底）
    val shouldLoadMore by remember {
        derivedStateOf {
            scrollState.maxValue == 0 || scrollState.value >= scrollState.maxValue - MEMBER_LOAD_MORE_THRESHOLD
        }
    }
    LaunchedEffect(
        shouldLoadMore,
        membersState.items.size,
        membersState.endReached,
        memberSearchState.items.size,
        memberSearchState.endReached,
        tabIndex,
        memberQuery,
    ) {
        if (tabIndex != 1 || !shouldLoadMore) return@LaunchedEffect
        if (memberQuery.isBlank()) membersState.loadMore(memberLoader) else memberSearchState.loadMore(memberSearchLoader)
    }

    // 切换 tab 时把分页标签行滚到视口顶部（不沿用上一 tab 的滚动位置）
    val tabAnchorPx = with(LocalDensity.current) {
        (GroupHeaderCardHeight + GroupHeaderToTabsGap).toPx().toInt()
    }
    var tabScrollInitialized by remember(groupId) { mutableStateOf(false) }
    LaunchedEffect(tabIndex) {
        if (!tabScrollInitialized) {
            tabScrollInitialized = true
            return@LaunchedEffect
        }
        scrollState.animateScrollTo(tabAnchorPx)
    }

    fun submitJoin(password: String?) {
        if (joining) return
        scope.launch {
            joining = true
            actionError = null
            passwordError = null
            when (val result = repository.enter(groupId, password = password, message = null)) {
                is ApiResult.Ok -> {
                    when (result.data.status) {
                        ENTER_STATUS_JOINED -> {
                            showPasswordDialog = false
                            joinPending = false
                            refreshAfterMembershipChange()
                        }
                        ENTER_STATUS_PENDING -> {
                            showPasswordDialog = false
                            joinPending = true
                        }
                        else -> actionError = "加入结果未知，请稍后刷新重试"
                    }
                }
                is ApiResult.Err -> {
                    // 10200 / 10300 / 10400 由 ApiClient 透传服务端 message（口令错误等）
                    if (showPasswordDialog) passwordError = result.error.userMessage
                    else actionError = result.error.userMessage
                }
            }
            joining = false
        }
    }

    fun requestJoin() {
        actionError = null
        passwordError = null
        if (detail?.joinMethod == "password") {
            passwordInput = ""
            showPasswordDialog = true
        } else {
            submitJoin(password = null)
        }
    }

    fun confirmExit() {
        if (exiting) return
        scope.launch {
            exiting = true
            exitError = null
            when (val result = repository.exit(groupId)) {
                is ApiResult.Ok -> {
                    showExitDialog = false
                    actionError = null
                    joinPending = false
                    refreshAfterMembershipChange()
                }
                is ApiResult.Err -> exitError = result.error.userMessage
            }
            exiting = false
        }
    }

    val currentDetail = detail
    // 退出群组仅成员可用；举报群组对齐 Web：非群主均可举报（未加入也可举报）
    val canLeaveGroup = currentDetail?.viewerState != null && currentDetail.viewerState.role != "owner"
    val canReportGroup = currentDetail != null && currentDetail.viewerState?.role != "owner"
    val canManageGroup = currentDetail?.viewerState?.role == "owner" ||
        currentDetail?.viewerState?.role == "admin"
    val canManageAnnouncements = canManageGroupAnnouncements(currentDetail?.viewerState?.role)
    val isGroupOwner = currentDetail?.viewerState?.role == "owner"

    LaunchedEffect(showInvite, inviteQuery) {
        if (!showInvite) return@LaunchedEffect
        val keyword = inviteQuery.trim()
        if (keyword.isBlank()) {
            inviteResults = emptyList()
            inviteError = null
            // 搜索中清空关键词/关闭弹窗会取消上面的协程，这里不复位就会一直停在「正在搜索…」
            inviteSearching = false
            return@LaunchedEffect
        }
        // 防抖窗口也算搜索中：先置位，避免把「还没发请求」显示成「没有找到匹配的用户」
        inviteSearching = true
        inviteError = null
        delay(300)
        when (val result = AppContainer.searchRepository.searchUsers(keyword, null)) {
            is ApiResult.Ok -> inviteResults = result.data.items
            is ApiResult.Err -> inviteError = result.error.userMessage
        }
        inviteSearching = false
    }
    Column(Modifier.fillMaxSize()) {
        GroupTopNavBar(
            onBack = onBack,
            moreEnabled = canManageGroup || canLeaveGroup || canReportGroup,
            canLeaveGroup = canLeaveGroup,
            canReportGroup = canReportGroup,
            onLeaveGroup = {
                exitError = null
                showExitDialog = true
            },
            onReportGroup = { showReportDialog = true },
            canManageGroup = canManageGroup,
            onEditGroupInfo = {
                groupInfoError = null
                showGroupInfoEditor = true
            },
            onOpenApplications = {
                applications = null
                applicationsError = null
                showApplications = true
                scope.launch {
                    applicationsLoading = true
                    when (val result = repository.applications(groupId)) {
                        is ApiResult.Ok -> applications = result.data
                        is ApiResult.Err -> applicationsError = result.error.userMessage
                    }
                    applicationsLoading = false
                }
            },
            onOpenBanned = {
                bannedMembers = null
                bannedError = null
                showBanned = true
                scope.launch {
                    bannedLoading = true
                    when (val result = repository.bannedMembers(groupId)) {
                        is ApiResult.Ok -> bannedMembers = result.data
                        is ApiResult.Err -> bannedError = result.error.userMessage
                    }
                    bannedLoading = false
                }
            },
            onOpenInvite = {
                showInvite = true
                inviteQuery = ""
                inviteResults = emptyList()
                inviteError = null
                invitedIds = emptySet()
            },
            onOpenSentInvites = {
                showSentInvites = true
                sentInvites = null
                sentInvitesError = null
                scope.launch {
                    sentInvitesLoading = true
                    when (val result = repository.sentInvites(groupId, status = "all")) {
                        is ApiResult.Ok -> sentInvites = result.data.items
                        is ApiResult.Err -> sentInvitesError = result.error.userMessage
                    }
                    sentInvitesLoading = false
                }
            },
            onOpenGroupSettings = {
                groupSettings = null
                groupSettingsError = null
                groupSettingsSaveError = null
                showGroupSettings = true
                scope.launch {
                    groupSettingsLoading = true
                    when (val result = repository.settings(groupId)) {
                        is ApiResult.Ok -> groupSettings = result.data
                        is ApiResult.Err -> groupSettingsError = result.error.userMessage
                    }
                    groupSettingsLoading = false
                }
            },
        )
        Box(Modifier.weight(1f).fillMaxWidth()) {
            when {
                currentDetail == null && detailError != null -> ErrorState(
                    message = detailError ?: "加载失败",
                    onRetry = { scope.launch { loadDetail() } },
                    modifier = Modifier.align(Alignment.TopCenter),
                )
                currentDetail == null -> LoadingState(
                    message = "正在加载群组…",
                    modifier = Modifier.align(Alignment.TopCenter),
                )
                else -> GroupDetailContent(
                    group = currentDetail,
                    tabIndex = tabIndex,
                    onTabSelected = { tabIndex = it },
                    scrollState = scrollState,
                    membersState = membersState,
                    memberSearchState = memberSearchState,
                    memberQuery = memberQuery,
                    onMemberQueryChange = { memberQuery = it },
                    announcements = announcements,
                    announcementsLoading = announcementsLoading,
                    announcementsError = announcementsError,
                    expandedAnnouncementId = expandedAnnouncementId,
                    onToggleAnnouncement = { id ->
                        expandedAnnouncementId = if (expandedAnnouncementId == id) null else id
                    },
                    onRetryAnnouncements = { scope.launch { loadAnnouncements() } },
                    canManageAnnouncements = canManageAnnouncements,
                    onCreateAnnouncement = {
                        editingAnnouncement = null
                        announcementEditorError = null
                        announcementCreateKey = UUID.randomUUID().toString()
                        showAnnouncementEditor = true
                    },
                    onEditAnnouncement = { announcement ->
                        editingAnnouncement = announcement
                        announcementEditorError = null
                        announcementCreateKey = null
                        showAnnouncementEditor = true
                    },
                    onDeleteAnnouncement = { announcement -> deletingAnnouncement = announcement },
                    canManageMembers = canManageGroup,
                    myUserId = myUserId,
                    onManageMember = { member -> managedMember = member },
                    onRetryMembersFirst = { scope.launch { membersState.initialLoad(memberLoader) } },
                    onRetryMembersMore = { scope.launch { membersState.loadMore(memberLoader) } },
                    onRetrySearchFirst = { scope.launch { memberSearchState.initialLoad(memberSearchLoader) } },
                    onRetrySearchMore = { scope.launch { memberSearchState.loadMore(memberSearchLoader) } },
                )
            }
        }
        if (currentDetail != null) {
            GroupActionDock(
                group = currentDetail,
                joining = joining,
                joinPending = joinPending,
                actionError = actionError,
                onJoin = { requestJoin() },
                onOpenGroupChat = { onOpenGroupChat(currentDetail) },
            )
        }
    }

    if (showPasswordDialog && currentDetail != null) {
        GroupPasswordDialog(
            groupName = currentDetail.group.name,
            password = passwordInput,
            errorText = passwordError,
            submitting = joining,
            onPasswordChange = {
                passwordInput = it
                passwordError = null
            },
            onSubmit = { submitJoin(passwordInput.trim()) },
            onDismiss = { if (!joining) showPasswordDialog = false },
        )
    }

    if (showReportDialog) {
        ReportDialog(
            title = "举报群组",
            submitting = reportSubmitting,
            onDismiss = { if (!reportSubmitting) showReportDialog = false },
            onSubmit = { reason ->
                if (reportSubmitting) return@ReportDialog
                reportSubmitting = true
                scope.launch {
                    when (val result = AppContainer.reportsRepository.create("group", groupId, reason)) {
                        is ApiResult.Ok -> {
                            showReportDialog = false
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

    if (showExitDialog && currentDetail != null) {
        GroupExitDialog(
            groupName = currentDetail.group.name,
            errorText = exitError,
            exiting = exiting,
            onConfirm = { confirmExit() },
            onDismiss = { if (!exiting) showExitDialog = false },
        )
    }

    if (showAnnouncementEditor) {
        val target = editingAnnouncement
        AnnouncementEditorDialog(
            title = if (target == null) "发布群公告" else "编辑群公告",
            initialTitle = target?.title.orEmpty(),
            initialContent = target?.content.orEmpty(),
            initialPinned = target?.isPinned == true,
            submitting = announcementSubmitting,
            error = announcementEditorError,
            onDismiss = {
                if (!announcementSubmitting) {
                    showAnnouncementEditor = false
                    announcementCreateKey = null
                }
            },
            onSubmit = { title, content, pinned ->
                if (announcementSubmitting) return@AnnouncementEditorDialog
                announcementSubmitting = true
                announcementEditorError = null
                scope.launch {
                    val result = if (target == null) {
                        val key = announcementCreateKey ?: UUID.randomUUID().toString().also {
                            announcementCreateKey = it
                        }
                        repository.createAnnouncement(groupId, title, content, pinned, key)
                    } else {
                        repository.updateAnnouncement(groupId, target.announcementId, title, content, pinned)
                    }
                    when (result) {
                        is ApiResult.Ok -> {
                            announcementSubmitting = false
                            showAnnouncementEditor = false
                            editingAnnouncement = null
                            announcementCreateKey = null
                            refreshAfterAnnouncementChange()
                            Toast.makeText(context, if (target == null) "公告已发布" else "公告已更新", Toast.LENGTH_SHORT).show()
                        }
                        is ApiResult.Err -> {
                            // 22000/22001 按契约要求换新 key 重试；22002（处理中）保留原 key。
                            val idempotencyCode = (result.error as? ApiError.Business)?.code
                            if (target == null && (idempotencyCode == 22000 || idempotencyCode == 22001)) {
                                announcementCreateKey = null
                            }
                            announcementEditorError = result.error.userMessage
                            announcementSubmitting = false
                        }
                    }
                }
            },
        )
    }

    if (showGroupInfoEditor && currentDetail != null) {
        GroupInfoEditorDialog(
            initialName = currentDetail.group.name,
            initialDescription = currentDetail.description.orEmpty(),
            submitting = groupInfoSaving,
            error = groupInfoError,
            onDismiss = { if (!groupInfoSaving) showGroupInfoEditor = false },
            onSubmit = { name, description ->
                if (groupInfoSaving) return@GroupInfoEditorDialog
                groupInfoSaving = true
                groupInfoError = null
                scope.launch {
                    when (val result = repository.updateGroupInfo(groupId, name, description.ifBlank { null })) {
                        is ApiResult.Ok -> {
                            detail = result.data
                            showGroupInfoEditor = false
                            Toast.makeText(context, "群信息已更新", Toast.LENGTH_SHORT).show()
                        }
                        is ApiResult.Err -> groupInfoError = result.error.userMessage
                    }
                    groupInfoSaving = false
                }
            },
        )
    }

    if (showGroupSettings) {
        GroupSettingsDialog(
            settings = groupSettings,
            loading = groupSettingsLoading,
            loadError = groupSettingsError,
            submitting = groupSettingsSaving,
            saveError = groupSettingsSaveError,
            onRetry = {
                scope.launch {
                    groupSettingsLoading = true
                    groupSettingsError = null
                    when (val result = repository.settings(groupId)) {
                        is ApiResult.Ok -> groupSettings = result.data
                        is ApiResult.Err -> groupSettingsError = result.error.userMessage
                    }
                    groupSettingsLoading = false
                }
            },
            onDismiss = { if (!groupSettingsSaving) showGroupSettings = false },
            onSubmit = { isSearchable, membersSearchable, requiresApproval, maxMembers, joinMethod, password, clearPassword ->
                if (groupSettingsSaving) return@GroupSettingsDialog
                groupSettingsSaving = true
                groupSettingsSaveError = null
                scope.launch {
                    when (
                        val result = repository.updateSettings(
                            groupId = groupId,
                            isSearchable = isSearchable,
                            membersSearchable = membersSearchable,
                            requiresApproval = requiresApproval,
                            maxMembers = maxMembers,
                            joinMethod = joinMethod,
                            password = password,
                            clearPassword = clearPassword,
                        )
                    ) {
                        is ApiResult.Ok -> {
                            showGroupSettings = false
                            loadDetail()
                            Toast.makeText(context, "群设置已保存", Toast.LENGTH_SHORT).show()
                        }
                        is ApiResult.Err -> groupSettingsSaveError = result.error.userMessage
                    }
                    groupSettingsSaving = false
                }
            },
        )
    }

    managedMember?.let { target ->
        val actions = buildList {
            if (target.role == "admin") add(MemberAction.UNSET_ADMIN) else add(MemberAction.SET_ADMIN)
            add(MemberAction.MUTE_1H)
            add(MemberAction.UNMUTE)
            add(MemberAction.REMOVE)
            add(MemberAction.BAN)
            if (isGroupOwner) add(MemberAction.TRANSFER_OWNER)
        }
        MemberActionSheet(
            actions = actions,
            onDismiss = { managedMember = null },
            onAction = { action ->
                managedMember = null
                if (action == MemberAction.REMOVE) {
                    removingMember = target
                } else if (action == MemberAction.TRANSFER_OWNER) {
                    transferTarget = target
                } else if (action == MemberAction.BAN) {
                    banningMember = target
                } else {
                    if (memberActionBusy) return@MemberActionSheet
                    memberActionBusy = true
                    scope.launch {
                        val result = when (action) {
                            MemberAction.SET_ADMIN -> repository.setMemberRole(groupId, target.user.userId, "admin")
                            MemberAction.UNSET_ADMIN -> repository.setMemberRole(groupId, target.user.userId, "member")
                            MemberAction.MUTE_1H -> repository.setMemberMute(groupId, target.user.userId, true, 60)
                            MemberAction.UNMUTE -> repository.setMemberMute(groupId, target.user.userId, false, null)
                            else -> return@launch
                        }
                        when (result) {
                            is ApiResult.Ok -> {
                                membersState.initialLoad(memberLoader)
                                loadDetail()
                                Toast.makeText(context, "成员操作已生效", Toast.LENGTH_SHORT).show()
                            }
                            is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                        }
                        memberActionBusy = false
                    }
                }
            },
        )
    }

    removingMember?.let { target ->
        AlertDialog(
            onDismissRequest = { if (!memberActionBusy) removingMember = null },
            title = { Text("移除成员") },
            text = { Text("确定将「${target.user.nickname.ifBlank { "该成员" }}」移出群组？") },
            confirmButton = {
                SystemButton(
                    text = if (memberActionBusy) "移除中…" else "移除",
                    onClick = {
                        if (memberActionBusy) return@SystemButton
                        memberActionBusy = true
                        scope.launch {
                            when (val result = repository.removeMember(groupId, target.user.userId)) {
                                is ApiResult.Ok -> {
                                    removingMember = null
                                    membersState.initialLoad(memberLoader)
                                    loadDetail()
                                    Toast.makeText(context, "成员已移除", Toast.LENGTH_SHORT).show()
                                }
                                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                            }
                            memberActionBusy = false
                        }
                    },
                    variant = ButtonVariant.DANGER,
                    size = ButtonSize.SM,
                    enabled = !memberActionBusy,
                    loading = memberActionBusy,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { if (!memberActionBusy) removingMember = null },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !memberActionBusy,
                )
            },
        )
    }

    if (showInvite) {
        InviteMemberDialog(
            query = inviteQuery,
            onQueryChange = {
                inviteQuery = it
                inviteError = null
            },
            results = inviteResults,
            searching = inviteSearching,
            error = inviteError,
            invitedIds = invitedIds,
            invitingId = invitingUserId,
            onInvite = { item ->
                val userId = item.user.userId
                if (invitingUserId != null || userId in invitedIds) return@InviteMemberDialog
                invitingUserId = userId
                scope.launch {
                    when (val result = repository.createInvite(groupId, userId)) {
                        is ApiResult.Ok -> {
                            if (result.data.inviteId == null) {
                                Toast.makeText(context, "对方已是群成员", Toast.LENGTH_SHORT).show()
                            } else {
                                invitedIds = invitedIds + userId
                                Toast.makeText(context, "邀请已发送", Toast.LENGTH_SHORT).show()
                            }
                        }
                        is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                    }
                    invitingUserId = null
                }
            },
            onDismiss = { if (invitingUserId == null) showInvite = false },
        )
    }

    if (showBanned) {
        BannedMembersDialog(
            members = bannedMembers,
            loading = bannedLoading,
            error = bannedError,
            processingUserId = unbanningId,
            onRetry = {
                scope.launch {
                    bannedLoading = true
                    bannedError = null
                    when (val result = repository.bannedMembers(groupId)) {
                        is ApiResult.Ok -> bannedMembers = result.data
                        is ApiResult.Err -> bannedError = result.error.userMessage
                    }
                    bannedLoading = false
                }
            },
            onUnban = { member ->
                if (unbanningId != null) return@BannedMembersDialog
                unbanningId = member.user.userId
                scope.launch {
                    when (val result = repository.setBan(groupId, member.user.userId, false)) {
                        is ApiResult.Ok -> {
                            bannedMembers = bannedMembers?.filterNot { it.user.userId == member.user.userId }
                            Toast.makeText(context, "已解除封禁", Toast.LENGTH_SHORT).show()
                        }
                        is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                    }
                    unbanningId = null
                }
            },
            onDismiss = { if (unbanningId == null) showBanned = false },
        )
    }

    if (showApplications) {
        GroupApplicationsDialog(
            applications = applications,
            loading = applicationsLoading,
            error = applicationsError,
            processingId = applicationProcessingId,
            onRetry = {
                scope.launch {
                    applicationsLoading = true
                    applicationsError = null
                    when (val result = repository.applications(groupId)) {
                        is ApiResult.Ok -> applications = result.data
                        is ApiResult.Err -> applicationsError = result.error.userMessage
                    }
                    applicationsLoading = false
                }
            },
            onApprove = { application ->
                if (applicationProcessingId != null) return@GroupApplicationsDialog
                applicationProcessingId = application.applicationId
                scope.launch {
                    when (val result = repository.approveApplication(groupId, application.applicationId)) {
                        is ApiResult.Ok -> {
                            applications = applications?.filterNot { it.applicationId == application.applicationId }
                            membersState.initialLoad(memberLoader)
                            loadDetail()
                            Toast.makeText(context, "已同意入群申请", Toast.LENGTH_SHORT).show()
                        }
                        is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                    }
                    applicationProcessingId = null
                }
            },
            onReject = { application ->
                if (applicationProcessingId != null) return@GroupApplicationsDialog
                applicationProcessingId = application.applicationId
                scope.launch {
                    when (val result = repository.rejectApplication(groupId, application.applicationId)) {
                        is ApiResult.Ok -> {
                            applications = applications?.filterNot { it.applicationId == application.applicationId }
                            Toast.makeText(context, "已拒绝入群申请", Toast.LENGTH_SHORT).show()
                        }
                        is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                    }
                    applicationProcessingId = null
                }
            },
            onDismiss = { if (applicationProcessingId == null) showApplications = false },
        )
    }

    if (showSentInvites) {
        GroupSentInvitesDialog(
            invites = sentInvites,
            loading = sentInvitesLoading,
            error = sentInvitesError,
            revokingId = revokingInviteId,
            onRetry = {
                scope.launch {
                    sentInvitesLoading = true
                    sentInvitesError = null
                    when (val result = repository.sentInvites(groupId, status = "all")) {
                        is ApiResult.Ok -> sentInvites = result.data.items
                        is ApiResult.Err -> sentInvitesError = result.error.userMessage
                    }
                    sentInvitesLoading = false
                }
            },
            onRevoke = { invite -> revokeInviteTarget = invite },
            onDismiss = { if (revokingInviteId == null) showSentInvites = false },
        )
    }

    revokeInviteTarget?.let { target ->
        AlertDialog(
            onDismissRequest = { if (revokingInviteId == null) revokeInviteTarget = null },
            title = { Text("撤销邀请") },
            text = { Text("撤销后「" + target.invitee.nickname.ifBlank { "对方" } + "」将不能再通过这条邀请入群，确定撤销吗？") },
            confirmButton = {
                SystemButton(
                    text = if (revokingInviteId != null) "撤销中…" else "撤销",
                    onClick = {
                        if (revokingInviteId != null) return@SystemButton
                        revokingInviteId = target.inviteId
                        scope.launch {
                            when (val result = repository.revokeInvite(groupId, target.inviteId)) {
                                is ApiResult.Ok -> {
                                    sentInvites = sentInvites?.map {
                                        if (it.inviteId == target.inviteId) it.copy(status = "revoked") else it
                                    }
                                    revokeInviteTarget = null
                                    Toast.makeText(context, "邀请已撤销", Toast.LENGTH_SHORT).show()
                                }
                                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                            }
                            revokingInviteId = null
                        }
                    },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = revokingInviteId == null,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { revokeInviteTarget = null },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = revokingInviteId == null,
                )
            },
        )
    }

    banningMember?.let { target ->
        AlertDialog(
            onDismissRequest = { if (!banBusy) banningMember = null },
            title = { Text("封禁成员") },
            text = {
                Text("确定封禁" + "「" + target.user.nickname.ifBlank { "该成员" } + "」？封禁后将被移出群组且无法再次加入。")
            },
            confirmButton = {
                SystemButton(
                    text = if (banBusy) "封禁中…" else "封禁",
                    onClick = {
                        if (banBusy) return@SystemButton
                        banBusy = true
                        scope.launch {
                            when (val result = repository.setBan(groupId, target.user.userId, true)) {
                                is ApiResult.Ok -> {
                                    banningMember = null
                                    membersState.initialLoad(memberLoader)
                                    loadDetail()
                                    Toast.makeText(context, "成员已封禁", Toast.LENGTH_SHORT).show()
                                }
                                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                            }
                            banBusy = false
                        }
                    },
                    variant = ButtonVariant.DANGER,
                    size = ButtonSize.SM,
                    enabled = !banBusy,
                    loading = banBusy,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { if (!banBusy) banningMember = null },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !banBusy,
                )
            },
        )
    }

    transferTarget?.let { target ->
        AlertDialog(
            onDismissRequest = { if (!transferBusy) transferTarget = null },
            title = { Text("转让群主") },
            text = {
                Text("确定将群主转让给「${target.user.nickname.ifBlank { "该成员" }}」？转让后你将成为普通管理员，此操作不可撤销。")
            },
            confirmButton = {
                SystemButton(
                    text = if (transferBusy) "转让中…" else "确认转让",
                    onClick = {
                        if (transferBusy) return@SystemButton
                        transferBusy = true
                        scope.launch {
                            when (val result = repository.transferOwnership(groupId, target.user.userId)) {
                                is ApiResult.Ok -> {
                                    transferTarget = null
                                    membersState.initialLoad(memberLoader)
                                    loadDetail()
                                    Toast.makeText(context, "群主已转让", Toast.LENGTH_SHORT).show()
                                }
                                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                            }
                            transferBusy = false
                        }
                    },
                    variant = ButtonVariant.DANGER,
                    size = ButtonSize.SM,
                    enabled = !transferBusy,
                    loading = transferBusy,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { if (!transferBusy) transferTarget = null },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !transferBusy,
                )
            },
        )
    }

    deletingAnnouncement?.let { target ->
        AlertDialog(
            onDismissRequest = { if (!announcementDeleting) deletingAnnouncement = null },
            title = { Text("删除公告") },
            text = { Text("确定删除「${target.title.ifBlank { "无标题公告" }}」？删除后不可恢复。") },
            confirmButton = {
                SystemButton(
                    text = if (announcementDeleting) "删除中…" else "删除",
                    onClick = {
                        if (announcementDeleting) return@SystemButton
                        announcementDeleting = true
                        scope.launch {
                            when (val result = repository.deleteAnnouncement(groupId, target.announcementId)) {
                                is ApiResult.Ok -> {
                                    deletingAnnouncement = null
                                    announcementDeleting = false
                                    refreshAfterAnnouncementChange()
                                    Toast.makeText(context, "公告已删除", Toast.LENGTH_SHORT).show()
                                }
                                is ApiResult.Err -> {
                                    announcementDeleting = false
                                    Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                                }
                            }
                        }
                    },
                    variant = ButtonVariant.DANGER,
                    size = ButtonSize.SM,
                    enabled = !announcementDeleting,
                    loading = announcementDeleting,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { if (!announcementDeleting) deletingAnnouncement = null },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !announcementDeleting,
                )
            },
        )
    }
}

/**
 * 顶部导航：左 42 圆钮返回；右 42 圆钮「更多」。
 * 更多菜单：成员且非群主展示「退出群组」；非群主（含未加入）展示「举报群组」；
 * 群主/详情未加载时无可用动作，更多钮禁用。返回钮始终可用。
 */
@Composable
private fun GroupTopNavBar(
    onBack: () -> Unit,
    moreEnabled: Boolean,
    canLeaveGroup: Boolean,
    canReportGroup: Boolean,
    canManageGroup: Boolean,
    onLeaveGroup: () -> Unit,
    onReportGroup: () -> Unit,
    onEditGroupInfo: () -> Unit,
    onOpenGroupSettings: () -> Unit,
    onOpenApplications: () -> Unit,
    onOpenInvite: () -> Unit,
    onOpenSentInvites: () -> Unit,
    onOpenBanned: () -> Unit,
) {
    var menuExpanded by remember { mutableStateOf(false) }
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(start = Spacing.lg, end = Spacing.lg, top = Spacing.md, bottom = Spacing.half),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        CircleNavButton(
            icon = Icons.AutoMirrored.Filled.ArrowBack,
            contentDescription = "返回",
            onClick = onBack,
        )
        Spacer(Modifier.weight(1f))
        Box {
            CircleNavButton(
                icon = Icons.Filled.MoreVert,
                iconRotation = 90f,
                contentDescription = if (moreEnabled) "更多" else "更多（加入群组后可用）",
                enabled = moreEnabled,
                onClick = { menuExpanded = true },
            )
            DropdownMenu(
                expanded = menuExpanded,
                onDismissRequest = { menuExpanded = false },
                // 圆角走项目 token（M3 默认 4dp 不在 token 体系内）；容器色与抬升由 ColorScheme 映射统一提供
                shape = androidx.compose.foundation.shape.RoundedCornerShape(Tokens.Radius.sm.dp),
            ) {
                if (canManageGroup) {
                    DropdownMenuItem(
                        text = {
                            Text(
                                text = "编辑群信息",
                                style = MaterialTheme.typography.bodyLarge,
                                color = HotSpotTheme.colors.textPrimary,
                            )
                        },
                        onClick = {
                            menuExpanded = false
                            onEditGroupInfo()
                        },
                    )
                }
                if (canManageGroup) {
                    DropdownMenuItem(
                        text = {
                            Text(
                                text = "群设置",
                                style = MaterialTheme.typography.bodyLarge,
                                color = HotSpotTheme.colors.textPrimary,
                            )
                        },
                        onClick = {
                            menuExpanded = false
                            onOpenGroupSettings()
                        },
                    )
                }
                if (canManageGroup) {
                    DropdownMenuItem(
                        text = {
                            Text(
                                text = "入群申请",
                                style = MaterialTheme.typography.bodyLarge,
                                color = HotSpotTheme.colors.textPrimary,
                            )
                        },
                        onClick = {
                            menuExpanded = false
                            onOpenApplications()
                        },
                    )
                }
                if (canManageGroup) {
                    DropdownMenuItem(
                        text = {
                            Text(
                                text = "邀请成员",
                                style = MaterialTheme.typography.bodyLarge,
                                color = HotSpotTheme.colors.textPrimary,
                            )
                        },
                        onClick = {
                            menuExpanded = false
                            onOpenInvite()
                        },
                    )
                }
                if (canManageGroup) {
                    DropdownMenuItem(
                        text = {
                            Text(
                                text = "邀请记录",
                                style = MaterialTheme.typography.bodyLarge,
                                color = HotSpotTheme.colors.textPrimary,
                            )
                        },
                        onClick = {
                            menuExpanded = false
                            onOpenSentInvites()
                        },
                    )
                }
                if (canManageGroup) {
                    DropdownMenuItem(
                        text = {
                            Text(
                                text = "封禁列表",
                                style = MaterialTheme.typography.bodyLarge,
                                color = HotSpotTheme.colors.textPrimary,
                            )
                        },
                        onClick = {
                            menuExpanded = false
                            onOpenBanned()
                        },
                    )
                }
                if (canReportGroup) {
                    DropdownMenuItem(
                        text = {
                            Text(
                                text = "举报群组",
                                style = MaterialTheme.typography.bodyLarge,
                                color = HotSpotTheme.colors.danger,
                            )
                        },
                        onClick = {
                            menuExpanded = false
                            onReportGroup()
                        },
                    )
                }
                if (canLeaveGroup) {
                    DropdownMenuItem(
                        text = {
                            Text(
                                text = "退出群组",
                                style = MaterialTheme.typography.bodyLarge,
                                color = HotSpotTheme.colors.danger,
                            )
                        },
                        onClick = {
                            menuExpanded = false
                            onLeaveGroup()
                        },
                    )
                }
            }
        }
    }
}

/** 42 圆钮（44 热区 + 可见焦点环）：返回 / 更多共用；禁用时 0.38 透明度。 */
@Composable
private fun CircleNavButton(
    icon: ImageVector,
    contentDescription: String,
    onClick: () -> Unit,
    enabled: Boolean = true,
    iconRotation: Float = 0f,
) {
    val colors = HotSpotTheme.colors
    Box(
        modifier = Modifier
            .size(Dimens.minTouchTarget)
            .clip(CircleShape)
            .clickable(
                interactionSource = remember { MutableInteractionSource() },
                indication = null,
                enabled = enabled,
                onClick = onClick,
            )
            .hotSpotFocusRing(cornerRadius = 22.dp),
        contentAlignment = Alignment.Center,
    ) {
        Box(
            modifier = Modifier
                .size(GroupNavButtonSize)
                .alpha(if (enabled) 1f else OpacityTokens.DISABLED)
                .shadow(
                    elevation = 1.dp,
                    shape = CircleShape,
                    ambientColor = colors.shadowDark,
                    spotColor = colors.shadowDark,
                )
                .clip(CircleShape)
                .background(colors.surface),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                imageVector = icon,
                contentDescription = contentDescription,
                modifier = Modifier
                    .size(Dimens.iconSm)
                    .rotate(iconRotation),
                tint = colors.textPrimary,
            )
        }
    }
}

/** 页面滚动内容：封面卡 -> 分页标签 -> 当前 tab 内容（整页单一滚动容器，docked 操作区在容器外）。 */
@Composable
private fun GroupDetailContent(
    group: GroupItem,
    tabIndex: Int,
    onTabSelected: (Int) -> Unit,
    scrollState: ScrollState,
    membersState: PagedListState<GroupMemberItem>,
    memberSearchState: PagedListState<GroupMemberItem>,
    memberQuery: String,
    onMemberQueryChange: (String) -> Unit,
    announcements: List<GroupAnnouncementItem>?,
    announcementsLoading: Boolean,
    announcementsError: String?,
    expandedAnnouncementId: String?,
    onToggleAnnouncement: (String) -> Unit,
    onRetryAnnouncements: () -> Unit,
    canManageAnnouncements: Boolean,
    onCreateAnnouncement: () -> Unit,
    onEditAnnouncement: (GroupAnnouncementItem) -> Unit,
    onDeleteAnnouncement: (GroupAnnouncementItem) -> Unit,
    canManageMembers: Boolean,
    myUserId: String?,
    onManageMember: (GroupMemberItem) -> Unit,
    onRetryMembersFirst: () -> Unit,
    onRetryMembersMore: () -> Unit,
    onRetrySearchFirst: () -> Unit,
    onRetrySearchMore: () -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(scrollState)
            .padding(horizontal = Spacing.lg),
    ) {
        GroupHeaderCard(group = group)
        Spacer(Modifier.height(GroupHeaderToTabsGap))
        TabPills(
            options = GROUP_TABS,
            selectedIndex = tabIndex,
            onSelected = onTabSelected,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(Spacing.xl))
        when (tabIndex) {
            0 -> GroupOverviewTab(
                group = group,
                membersState = membersState,
                announcements = announcements,
                announcementsLoading = announcementsLoading,
                announcementsError = announcementsError,
                onRetryAnnouncements = onRetryAnnouncements,
                onViewAllMembers = { onTabSelected(1) },
                onOpenAnnouncements = { onTabSelected(2) },
            )
            1 -> GroupMembersTab(
                state = membersState,
                searchState = memberSearchState,
                memberQuery = memberQuery,
                onMemberQueryChange = onMemberQueryChange,
                totalCount = group.group.membersCount,
                canManage = canManageMembers,
                myUserId = myUserId,
                onManageMember = onManageMember,
                onRetryFirstPage = onRetryMembersFirst,
                onRetryLoadMore = onRetryMembersMore,
                onRetrySearchFirst = onRetrySearchFirst,
                onRetrySearchMore = onRetrySearchMore,
            )
            else -> GroupAnnouncementsTab(
                announcements = announcements,
                loading = announcementsLoading,
                error = announcementsError,
                expandedAnnouncementId = expandedAnnouncementId,
                onToggle = onToggleAnnouncement,
                onRetry = onRetryAnnouncements,
                canManage = canManageAnnouncements,
                onCreate = onCreateAnnouncement,
                onEdit = onEditAnnouncement,
                onDelete = onDeleteAnnouncement,
            )
        }
        Spacer(Modifier.height(Spacing.xxl))
    }
}

/** 封面主卡：封面 358x156（头像铺满 / 无头像主题软底 + 群名首字）+ 白底群名带 64，圆角 20。 */
@Composable
private fun GroupHeaderCard(group: GroupItem) {
    val colors = HotSpotTheme.colors
    val shape = AppShapes.card
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .shadow(
                elevation = 1.dp,
                shape = shape,
                ambientColor = colors.shadowDark,
                spotColor = colors.shadowDark,
            )
            .clip(shape),
    ) {
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(GroupCoverHeight),
        ) {
            val coverUrl = MediaUrls.absolute(group.group.avatarUrl)
            if (coverUrl != null) {
                AsyncImage(
                    model = ImageRequest.Builder(LocalContext.current)
                        .data(coverUrl)
                        .crossfade(true)
                        .build(),
                    contentDescription = (group.group.name.ifBlank { "群组" }) + " 封面",
                    modifier = Modifier.fillMaxSize(),
                    contentScale = ContentScale.Crop,
                )
            } else {
                // 无头像：主题软底 + 群名首字（不造假图片）
                Box(
                    modifier = Modifier
                        .fillMaxSize()
                        .background(colors.accentSoftBg),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = group.group.name.trim().take(1).ifEmpty { "群" },
                        style = MaterialTheme.typography.headlineLarge,
                        color = colors.textAccentOnSoft,
                    )
                }
            }
            val firstTag = group.tags.firstOrNull { it.isNotBlank() }
            if (firstTag != null) {
                MediaBadge(
                    text = firstTag,
                    modifier = Modifier
                        .align(Alignment.BottomStart)
                        .padding(start = Spacing.lg2),
                )
            }
        }
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(GroupNameBandHeight)
                .background(colors.surface)
                .padding(horizontal = Spacing.lg2),
            contentAlignment = Alignment.CenterStart,
        ) {
            Text(
                text = group.group.name.ifBlank { "未命名群组" },
                style = TextStyle(
                    fontSize = GroupNameSize,
                    lineHeight = GroupNameLineHeight,
                    fontWeight = FontWeight.SemiBold,
                ),
                color = colors.textPrimary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}

/** 简介 tab：简介文案 + 指标 4 列 + 活跃成员 + 群公告预告。 */
@Composable
private fun GroupOverviewTab(
    group: GroupItem,
    membersState: PagedListState<GroupMemberItem>,
    announcements: List<GroupAnnouncementItem>?,
    announcementsLoading: Boolean,
    announcementsError: String?,
    onRetryAnnouncements: () -> Unit,
    onViewAllMembers: () -> Unit,
    onOpenAnnouncements: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    val description = group.description?.trim().orEmpty()

    Text(
        text = description.ifEmpty { "这个群组还没有简介" },
        style = MaterialTheme.typography.bodyLarge,
        color = if (description.isEmpty()) colors.textSecondary else colors.textPrimary,
    )
    Spacer(Modifier.height(Spacing.xl))
    GroupStatsRow(
        group = group,
        announcementCount = announcements?.size,
    )
    Spacer(Modifier.height(Spacing.xl))

    val activeMembers = membersState.items.take(ACTIVE_MEMBER_AVATARS)
    if (activeMembers.isNotEmpty()) {
        GroupSectionHeading(
            title = "活跃成员",
            count = group.group.membersCount,
            actionLabel = "查看全部",
            onAction = onViewAllMembers,
        )
        Spacer(Modifier.height(Spacing.sm))
        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            GroupAvatarStack(activeMembers)
            Spacer(Modifier.width(Spacing.md))
            Text(
                text = group.group.membersCount.toString() + " 位成员在群中",
                style = HotSpotTextStyles.caption,
                color = colors.textSecondary,
            )
        }
        Spacer(Modifier.height(Spacing.xl))
    }

    GroupSectionHeading(
        title = "群公告",
        count = announcements?.size,
    )
    Spacer(Modifier.height(Spacing.sm))
    when {
        announcementsLoading && announcements == null -> InlineLoading(text = "正在加载公告…")
        announcementsError != null && announcements == null -> ErrorState(
            message = announcementsError,
            onRetry = onRetryAnnouncements,
        )
        announcements.isNullOrEmpty() -> Box(
            modifier = Modifier
                .fillMaxWidth()
                .clip(AppShapes.card)
                .background(colors.island)
                .padding(vertical = Spacing.xl),
            contentAlignment = Alignment.Center,
        ) {
            Text(
                text = "还没有群公告",
                style = MaterialTheme.typography.bodyMedium,
                color = colors.textSecondary,
            )
        }
        else -> Column(verticalArrangement = Arrangement.spacedBy(Spacing.md)) {
            announcements.take(ANNOUNCEMENT_PREVIEW_COUNT).forEach { announcement ->
                AnnouncementPreviewCard(
                    announcement = announcement,
                    onClick = onOpenAnnouncements,
                )
            }
        }
    }
}

/** 指标 4 列（群成员 / 在线 / 距你 / 公告）：数字 20 semibold + 标签 11 次级。 */
@Composable
private fun GroupStatsRow(
    group: GroupItem,
    announcementCount: Int?,
) {
    val colors = HotSpotTheme.colors
    val online = group.onlineMembersCount
    val distance = formatDistanceShort(group.location?.distance)
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = Spacing.lg),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.Top,
    ) {
        GroupStatItem(value = group.group.membersCount.toString(), label = "群成员")
        GroupStatItem(
            value = online?.toString() ?: "—",
            label = "在线",
            valueColor = colors.online,
        )
        GroupStatItem(value = distance ?: "—", label = "距你")
        GroupStatItem(value = announcementCount?.toString() ?: "—", label = "公告")
    }
}

@Composable
private fun GroupStatItem(
    value: String,
    label: String,
    valueColor: Color? = null,
) {
    val colors = HotSpotTheme.colors
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Text(
            text = value,
            style = TextStyle(
                fontSize = GroupStatNumberSize,
                lineHeight = GroupStatNumberLineHeight,
                fontWeight = FontWeight.SemiBold,
            ),
            color = valueColor ?: colors.textPrimary,
            maxLines = 1,
        )
        Spacer(Modifier.height(Spacing.sm))
        Text(
            text = label,
            style = TextStyle(fontSize = GroupStatLabelSize, lineHeight = GroupStatLabelLineHeight),
            color = colors.textSecondary,
            maxLines = 1,
        )
    }
}

/** 活跃成员头像堆叠：32 头像 + 重叠 8（步进 24），page 色圆环圈出层次。 */
@Composable
private fun GroupAvatarStack(members: List<GroupMemberItem>) {
    val colors = HotSpotTheme.colors
    val stackWidth = GroupAvatarStackStep * (members.size - 1) + GroupAvatarRingSize
    Box(
        Modifier
            .width(stackWidth)
            .height(GroupAvatarRingSize),
    ) {
        members.forEachIndexed { index, member ->
            Box(
                modifier = Modifier
                    .offset(x = GroupAvatarStackStep * index)
                    .size(GroupAvatarRingSize),
                contentAlignment = Alignment.Center,
            ) {
                Box(
                    modifier = Modifier
                        .fillMaxSize()
                        .clip(CircleShape)
                        .background(colors.page),
                )
                Avatar(
                    name = member.user.nickname,
                    imageUrl = member.user.avatarUrl,
                    size = GroupAvatarStackSize,
                )
            }
        }
    }
}

/** 区块标题：14 semibold + 计数 11 次级 + 可选右侧动作（「查看全部 >」，44 热区）。 */
@Composable
private fun GroupSectionHeading(
    title: String,
    count: Int?,
    actionLabel: String? = null,
    onAction: (() -> Unit)? = null,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .heightIn(min = Dimens.minTouchTarget),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = title,
            style = MaterialTheme.typography.bodyLarge.copy(fontWeight = FontWeight.SemiBold),
            color = colors.textPrimary,
        )
        if (count != null) {
            Spacer(Modifier.width(Spacing.sm))
            Text(
                text = count.toString(),
                style = HotSpotTextStyles.caption,
                color = colors.textSecondary,
            )
        }
        Spacer(Modifier.weight(1f))
        if (actionLabel != null && onAction != null) {
            Row(
                modifier = Modifier
                    .heightIn(min = Dimens.minTouchTarget)
                    .clip(AppShapes.pill)
                    .clickable(
                        interactionSource = remember { MutableInteractionSource() },
                        indication = null,
                        onClick = onAction,
                    )
                    .hotSpotFocusRing(cornerRadius = Tokens.Radius.full.dp)
                    .padding(start = Spacing.sm),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = actionLabel,
                    style = HotSpotTextStyles.caption.copy(fontWeight = FontWeight.Medium),
                    color = colors.accent,
                )
                Icon(
                    imageVector = Icons.AutoMirrored.Filled.KeyboardArrowRight,
                    contentDescription = null,
                    modifier = Modifier.size(Dimens.icon2xs),
                    tint = colors.accent,
                )
            }
        }
    }
}

/** 简介页公告预告卡：置顶 Badge + 标题 + 单行摘要 + 箭头，点击切到公告 tab（真实 tab 交互）。 */
@Composable
private fun AnnouncementPreviewCard(
    announcement: GroupAnnouncementItem,
    onClick: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    val shape = AppShapes.card
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .shadow(
                elevation = 1.dp,
                shape = shape,
                ambientColor = colors.shadowDark,
                spotColor = colors.shadowDark,
            )
            .clip(shape)
            .background(colors.island)
            .clickable(
                interactionSource = remember { MutableInteractionSource() },
                indication = null,
                onClick = onClick,
            )
            .hotSpotFocusRing(cornerRadius = Tokens.Radius.lg.dp)
            .padding(Spacing.md3),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            modifier = Modifier
                .size(36.dp)
                .clip(CircleShape)
                .background(colors.subtle),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painter = painterResource(R.drawable.ic_megaphone),
                contentDescription = null,
                modifier = Modifier.size(Dimens.iconXs),
                tint = colors.textSecondary,
            )
        }
        Spacer(Modifier.width(Spacing.md2))
        Column(Modifier.weight(1f)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                if (announcement.isPinned) {
                    Badge(text = "置顶", tone = BadgeTone.ACCENT)
                    Spacer(Modifier.width(Spacing.sm))
                }
                Text(
                    text = announcement.title.ifBlank { "无标题公告" },
                    style = MaterialTheme.typography.bodyLarge.copy(fontWeight = FontWeight.SemiBold),
                    color = colors.textPrimary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            Spacer(Modifier.height(Spacing.xs))
            Text(
                text = announcement.content,
                style = MaterialTheme.typography.bodyMedium,
                color = colors.textSecondary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
        Spacer(Modifier.width(Spacing.sm))
        Icon(
            imageVector = Icons.AutoMirrored.Filled.KeyboardArrowRight,
            contentDescription = null,
            modifier = Modifier.size(Dimens.iconXs),
            tint = colors.textSecondary,
        )
    }
}

/** 成员 tab：头像 40 + 昵称 + 角色徽章（owner 群主 / admin 管理员 / member 不显示）；行不可点。 */
@Composable
private fun GroupMembersTab(
    state: PagedListState<GroupMemberItem>,
    searchState: PagedListState<GroupMemberItem>,
    memberQuery: String,
    onMemberQueryChange: (String) -> Unit,
    totalCount: Int,
    canManage: Boolean,
    myUserId: String?,
    onManageMember: (GroupMemberItem) -> Unit,
    onRetryFirstPage: () -> Unit,
    onRetryLoadMore: () -> Unit,
    onRetrySearchFirst: () -> Unit,
    onRetrySearchMore: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    SearchInput(
        value = memberQuery,
        onValueChange = onMemberQueryChange,
        placeholder = "搜索成员昵称…",
    )
    Spacer(Modifier.height(Spacing.sm))
    val searching = memberQuery.isNotBlank()
    val shownState = if (searching) searchState else state
    val onRetryFirst = if (searching) onRetrySearchFirst else onRetryFirstPage
    val onRetryMore = if (searching) onRetrySearchMore else onRetryLoadMore
    when {
        shownState.loading && shownState.items.isEmpty() -> LoadingState(
            message = if (searching) "正在搜索成员…" else "正在加载成员…",
        )
        shownState.error != null && shownState.items.isEmpty() -> ErrorState(
            message = shownState.error ?: "加载失败",
            onRetry = onRetryFirst,
        )
        shownState.items.isEmpty() -> EmptyState(
            title = if (searching) "没有找到匹配的成员" else "还没有成员",
            description = if (searching) "换个昵称关键词试试" else "这个群组暂时没有可展示的成员",
        )
        else -> {
            Text(
                text = if (searching) "搜索「" + memberQuery.trim() + "」· " + shownState.items.size + " 位成员"
                else "共 " + totalCount + " 位成员",
                style = HotSpotTextStyles.caption,
                color = colors.textSecondary,
            )
            Spacer(Modifier.height(Spacing.sm))
            ListGroup {
                shownState.items.forEach { member ->
                    GroupMemberRow(
                        member = member,
                        manageable = canManage && member.role != "owner" && member.user.userId != myUserId,
                        onManage = { onManageMember(member) },
                    )
                }
            }
            when {
                shownState.loadingMore -> InlineLoading()
                shownState.error != null -> ErrorState(
                    message = shownState.error ?: "加载失败",
                    onRetry = onRetryMore,
                )
                shownState.endReached -> ListEnd()
            }
        }
    }
}

@Composable
private fun GroupMemberRow(
    member: GroupMemberItem,
    manageable: Boolean = false,
    onManage: () -> Unit = {},
) {
    val colors = HotSpotTheme.colors
    val roleLabel = groupRoleLabel(member.role)
    val trailingSlot: (@Composable () -> Unit)? = if (roleLabel != null || manageable) {
        {
            Row(verticalAlignment = Alignment.CenterVertically) {
                if (roleLabel != null) {
                    Badge(
                        text = roleLabel,
                        tone = if (member.role == "owner") BadgeTone.ACCENT else BadgeTone.NEUTRAL,
                    )
                }
                if (manageable) {
                    Spacer(Modifier.width(Spacing.sm))
                    Text(
                        text = "管理",
                        style = MaterialTheme.typography.labelMedium,
                        color = colors.accent,
                        modifier = Modifier
                            .heightIn(min = Dimens.minTouchTarget)
                            .clip(AppShapes.compact)
                            .clickable(onClick = onManage)
                            .padding(horizontal = Spacing.md),
                    )
                }
            }
        }
    } else {
        null
    }
    ListRow(
        title = member.user.nickname.ifBlank { "匿名成员" },
        leading = {
            Avatar(
                name = member.user.nickname,
                imageUrl = member.user.avatarUrl,
                size = Dimens.avatarSm,
            )
        },
        trailing = trailingSlot,
        onClick = null,
    )
}

/** 公告 tab：置顶 Badge + 标题 + 时间 + 正文；点击行内展开/收起全文（真实展开交互）。 */
@Composable
private fun GroupAnnouncementsTab(
    announcements: List<GroupAnnouncementItem>?,
    loading: Boolean,
    error: String?,
    expandedAnnouncementId: String?,
    onToggle: (String) -> Unit,
    onRetry: () -> Unit,
    canManage: Boolean,
    onCreate: () -> Unit,
    onEdit: (GroupAnnouncementItem) -> Unit,
    onDelete: (GroupAnnouncementItem) -> Unit,
) {
    Column(verticalArrangement = Arrangement.spacedBy(Spacing.md)) {
        if (canManage) {
            SystemButton(
                text = "发布公告",
                onClick = onCreate,
                variant = ButtonVariant.PRIMARY,
                size = ButtonSize.SM,
                fillMaxWidth = true,
            )
        }
        when {
            loading && announcements == null -> LoadingState(message = "正在加载公告…")
            error != null && announcements == null -> ErrorState(message = error, onRetry = onRetry)
            announcements.isNullOrEmpty() -> EmptyState(
                title = "还没有群公告",
                description = if (canManage) "点击上方按钮发布第一条公告" else "群主发布公告后会显示在这里",
            )
            else -> {
                announcements.forEach { announcement ->
                    AnnouncementCard(
                        announcement = announcement,
                        expanded = announcement.announcementId == expandedAnnouncementId,
                        onToggle = { onToggle(announcement.announcementId) },
                        canManage = canManage,
                        onEdit = { onEdit(announcement) },
                        onDelete = { onDelete(announcement) },
                    )
                }
                // 公告为单次全量返回（契约无分页），列表渲染完即到底
                ListEnd()
            }
        }
    }
}

@Composable
private fun AnnouncementCard(
    announcement: GroupAnnouncementItem,
    expanded: Boolean,
    onToggle: () -> Unit,
    canManage: Boolean = false,
    onEdit: () -> Unit = {},
    onDelete: () -> Unit = {},
) {
    val colors = HotSpotTheme.colors
    val shape = AppShapes.card
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .shadow(
                elevation = 1.dp,
                shape = shape,
                ambientColor = colors.shadowDark,
                spotColor = colors.shadowDark,
            )
            .clip(shape)
            .background(colors.island)
            .clickable(
                interactionSource = remember { MutableInteractionSource() },
                indication = null,
                onClick = onToggle,
            )
            .hotSpotFocusRing(cornerRadius = Tokens.Radius.lg.dp)
            .padding(Spacing.md3),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            if (announcement.isPinned) {
                Badge(text = "置顶", tone = BadgeTone.ACCENT)
                Spacer(Modifier.width(Spacing.sm))
            }
            Text(
                text = announcement.title.ifBlank { "无标题公告" },
                style = MaterialTheme.typography.bodyLarge.copy(fontWeight = FontWeight.SemiBold),
                color = colors.textPrimary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
            Spacer(Modifier.width(Spacing.sm))
            Text(
                text = TimeFormat.relative(announcement.createdAt),
                style = HotSpotTextStyles.caption,
                color = colors.textSecondary,
            )
        }
        Spacer(Modifier.height(Spacing.xs))
        Text(
            text = announcement.content,
            style = MaterialTheme.typography.bodyMedium,
            color = colors.textSecondary,
            maxLines = if (expanded) Int.MAX_VALUE else 1,
            overflow = TextOverflow.Ellipsis,
        )
        if (expanded && canManage) {
            Spacer(Modifier.height(Spacing.sm))
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.End,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                AnnouncementAction(label = "编辑", color = colors.accent, onClick = onEdit)
                Spacer(Modifier.width(Spacing.sm))
                AnnouncementAction(label = "删除", color = colors.danger, onClick = onDelete)
            }
        }
    }
}

/** 公告卡内文字动作：视觉是 label，触控热区撑到 44dp 并保留焦点环。 */
@Composable
private fun AnnouncementAction(
    label: String,
    color: Color,
    onClick: () -> Unit,
) {
    Box(
        modifier = Modifier
            .heightIn(min = Dimens.minTouchTarget)
            .clip(AppShapes.compact)
            .clickable(
                interactionSource = remember { MutableInteractionSource() },
                indication = null,
                onClick = onClick,
            )
            .hotSpotFocusRing(cornerRadius = Tokens.Radius.sm.dp)
            .semantics { role = Role.Button }
            .padding(horizontal = Spacing.md),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.labelMedium,
            color = color,
        )
    }
}

/**
 * docked 加入操作区（设计稿 87:1696 底部卡：358 宽、16 内距、按钮 44 + 说明 11）：
 * - 已加入：次级禁用样式「已加入」+ 说明「你已在这个群组中」（退出入口只在右上更多菜单）
 * - 审核中：次级禁用样式「已申请，等待审核」+ 说明
 * - 未加入：主按钮「加入群组」+ 铃铛说明（设计稿原文）
 * 底部按系统导航栏 inset 避让（MainActivity 只注入 top/horizontal 安全区）。
 */
@Composable
private fun GroupActionDock(
    group: GroupItem,
    joining: Boolean,
    joinPending: Boolean,
    actionError: String?,
    onJoin: () -> Unit,
    onOpenGroupChat: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    val joined = group.viewerState != null
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .windowInsetsPadding(
                WindowInsets.navigationBars.only(WindowInsetsSides.Bottom),
            )
            .padding(horizontal = Spacing.lg, vertical = Spacing.md),
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .shadow(
                    elevation = 1.dp,
                    shape = AppShapes.card,
                    ambientColor = colors.shadowDark,
                    spotColor = colors.shadowDark,
                )
                .clip(AppShapes.card)
                .background(colors.surface)
                .padding(Spacing.lg),
        ) {
            if (actionError != null) {
                Text(
                    text = actionError,
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.danger,
                )
                Spacer(Modifier.height(Spacing.sm))
            }
            when {
                joined -> {
                    // 已加入：主入口打开群聊（对齐 Web GroupActionCard「打开群聊」）；退出入口仍在右上更多菜单
                    SystemButton(
                        text = "打开群聊",
                        onClick = onOpenGroupChat,
                        variant = ButtonVariant.PRIMARY,
                        fillMaxWidth = true,
                        height = GroupActionButtonHeight,
                    )
                    Spacer(Modifier.height(Spacing.md3))
                    GroupActionTip(text = "你已在这个群组中，打开即可查看群内消息")
                }
                joinPending -> {
                    SystemButton(
                        text = "已申请，等待审核",
                        onClick = {},
                        variant = ButtonVariant.SECONDARY,
                        enabled = false,
                        fillMaxWidth = true,
                        height = GroupActionButtonHeight,
                    )
                    Spacer(Modifier.height(Spacing.md3))
                    GroupActionTip(text = "群主通过申请后即可加入群聊")
                }
                else -> {
                    SystemButton(
                        text = "加入群组",
                        onClick = onJoin,
                        variant = ButtonVariant.PRIMARY,
                        loading = joining,
                        fillMaxWidth = true,
                        height = GroupActionButtonHeight,
                    )
                    Spacer(Modifier.height(Spacing.md3))
                    GroupActionTip(
                        text = "加入后可管理群组通知并进入群聊",
                        showBellIcon = true,
                    )
                }
            }
        }
    }
}

/** 操作区说明行（11 次级；未加入态带 13 铃铛图标，设计稿 87:1696 ActionTip）。 */
@Composable
private fun GroupActionTip(
    text: String,
    showBellIcon: Boolean = false,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (showBellIcon) {
            Icon(
                imageVector = Icons.Filled.Notifications,
                contentDescription = null,
                modifier = Modifier.size(Dimens.icon2xs),
                tint = colors.accent,
            )
            Spacer(Modifier.width(Spacing.sm2))
        }
        Text(
            text = text,
            style = TextStyle(fontSize = GroupActionTipSize, lineHeight = GroupActionTipLineHeight),
            color = colors.textSecondary,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

/** 口令群加入 Dialog：口令错误透传服务端 message（10401/10400 等）。 */
@Composable
private fun GroupPasswordDialog(
    groupName: String,
    password: String,
    errorText: String?,
    submitting: Boolean,
    onPasswordChange: (String) -> Unit,
    onSubmit: () -> Unit,
    onDismiss: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        titleContentColor = HotSpotTheme.colors.textPrimary,
        textContentColor = HotSpotTheme.colors.textSecondary,
        title = { Text("输入群口令") },
        text = {
            Column {
                Text(
                    text = "「" + groupName.ifBlank { "该群组" } + "」需要口令才能加入。",
                    style = MaterialTheme.typography.bodyMedium,
                    color = HotSpotTheme.colors.textSecondary,
                )
                Spacer(Modifier.height(Spacing.md))
                SystemInput(
                    value = password,
                    onValueChange = onPasswordChange,
                    label = "群口令",
                    placeholder = "请输入群口令",
                    errorText = errorText,
                    keyboardOptions = KeyboardOptions(
                        keyboardType = KeyboardType.Password,
                        imeAction = ImeAction.Done,
                    ),
                    keyboardActions = KeyboardActions(onDone = { if (password.isNotBlank()) onSubmit() }),
                    visualTransformation = PasswordVisualTransformation(),
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        },
        confirmButton = {
            SystemButton(
                text = if (submitting) "加入中…" else "加入",
                onClick = onSubmit,
                variant = ButtonVariant.PRIMARY,
                enabled = password.isNotBlank() && !submitting,
            )
        },
        dismissButton = {
            SystemButton(
                text = "取消",
                onClick = onDismiss,
                variant = ButtonVariant.SOFT,
                enabled = !submitting,
            )
        },
    )
}

/** 退出群组二次确认 Dialog：失败（如群主需先转让）在弹窗内透传服务端 message。 */
@Composable
private fun GroupExitDialog(
    groupName: String,
    errorText: String?,
    exiting: Boolean,
    onConfirm: () -> Unit,
    onDismiss: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        titleContentColor = HotSpotTheme.colors.textPrimary,
        textContentColor = HotSpotTheme.colors.textSecondary,
        title = { Text("退出群组「" + groupName.ifBlank { "该群组" } + "」？") },
        text = {
            Column {
                Text(
                    text = "退出后将不再接收该群的聊天与公告。",
                    style = MaterialTheme.typography.bodyMedium,
                    color = HotSpotTheme.colors.textSecondary,
                )
                if (errorText != null) {
                    Spacer(Modifier.height(Spacing.sm))
                    Text(
                        text = errorText,
                        style = MaterialTheme.typography.bodySmall,
                        color = HotSpotTheme.colors.danger,
                    )
                }
            }
        },
        confirmButton = {
            SystemButton(
                text = if (exiting) "退出中…" else "确认退出",
                onClick = onConfirm,
                variant = ButtonVariant.DANGER,
                enabled = !exiting,
            )
        },
        dismissButton = {
            SystemButton(
                text = "取消",
                onClick = onDismiss,
                variant = ButtonVariant.SOFT,
                enabled = !exiting,
            )
        },
    )
}
