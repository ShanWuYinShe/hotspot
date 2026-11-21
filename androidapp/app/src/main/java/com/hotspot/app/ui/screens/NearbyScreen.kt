package com.hotspot.app.ui.screens

import android.widget.Toast
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
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
import androidx.compose.material.icons.filled.Place
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.GroupItem
import com.hotspot.app.data.model.MomentItem
import com.hotspot.app.data.model.UserItem
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.HeaderIconButton
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.NearbyGroupCard
import com.hotspot.app.ui.components.NearbyLocationChip
import com.hotspot.app.ui.components.NearbyMomentCard
import com.hotspot.app.ui.components.NearbySectionHead
import com.hotspot.app.ui.components.NearbyStatsCard
import com.hotspot.app.ui.components.NearbyUserRow
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import kotlinx.coroutines.launch

/**
 * 附近枢纽（设计稿 87:4409）：顶部「附近 + 定位胶囊 + 搜索/地图」→ 实时附近统计卡
 * → 附近的人预览（含关注）→ 附近瞬间两列预览 → 附近群组两列预览；各段「查看全部」进叶子页。
 * 距离由后端下发，客户端只展示；地图原生渲染仍是已登记 TODO（入口保留并明确提示）。
 */
@Composable
fun NearbyScreen(
    onOpenSearch: () -> Unit,
    onOpenPeople: () -> Unit,
    onOpenGroups: () -> Unit,
    onOpenMoments: () -> Unit,
    onOpenMoment: (momentId: String) -> Unit,
    onOpenGroup: (groupId: String) -> Unit,
) {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val peopleState = rememberSingleLoadState<UserItem>()
    val groupsState = rememberSingleLoadState<GroupItem>()
    val momentsState = rememberSingleLoadState<MomentItem>()
    val refreshKey = remember { mutableStateMapOf<String, Int>() }
    val refreshTick = refreshKey["tick"] ?: 0
    var locationLabel by remember { mutableStateOf("附近 · 5 km 内") }
    // 当前登录用户：附近的人里排除自己（对齐 Web / 鸿蒙）
    val selfId = AppContainer.tokenStore.session.value?.userId

    LaunchedEffect(refreshTick) {
        // 浏览锚点：client_config.defaults.default_location（后端回退链在 groups/moments 上不一致，显式传参）
        val anchor = AppContainer.defaultLocation()
        if (!anchor?.locationName.isNullOrBlank()) {
            locationLabel = anchor!!.locationName + " · 5 km 内"
        }
        val point = anchor?.point
        peopleState.load { loadFirstPage { cursor -> AppContainer.nearbyRepository.nearbyPeople(cursor, referencePoint = point) } }
        groupsState.load { loadFirstPage { cursor -> AppContainer.nearbyRepository.nearbyGroups(cursor, referencePoint = point) } }
        momentsState.load { loadFirstPage { cursor -> AppContainer.nearbyRepository.nearbyMoments(cursor, referencePoint = point) } }
    }

    // 关注态（本地乐观更新）+ 加入/申请态
    val followStates = remember { mutableStateMapOf<String, Boolean>() }
    val followBusy = remember { mutableStateMapOf<String, Boolean>() }
    val joinedStates = remember { mutableStateMapOf<String, Boolean>() }
    val pendingStates = remember { mutableStateMapOf<String, Boolean>() }
    val joinBusy = remember { mutableStateMapOf<String, Boolean>() }

    fun toggleFollow(user: UserItem) {
        val userId = user.user.userId
        if (userId.isBlank() || followBusy[userId] == true) return
        val current = followStates[userId] ?: (user.relation?.isFollowing ?: false)
        val next = !current
        followStates[userId] = next
        followBusy[userId] = true
        scope.launch {
            when (val result = AppContainer.profileRepository.setFollow(userId, next)) {
                is ApiResult.Ok -> Unit
                is ApiResult.Err -> {
                    followStates[userId] = current
                    Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                }
            }
            followBusy[userId] = false
        }
    }

    fun joinGroup(group: GroupItem) {
        val groupId = group.group.groupId
        if (groupId.isBlank() || joinBusy[groupId] == true) return
        joinBusy[groupId] = true
        scope.launch {
            when (val result = AppContainer.groupRepository.enter(groupId)) {
                is ApiResult.Ok -> {
                    if (result.data.status == "pending") {
                        pendingStates[groupId] = true
                    } else {
                        joinedStates[groupId] = true
                    }
                }
                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
            }
            joinBusy[groupId] = false
        }
    }

    val nearbyPeople = peopleState.items.filter { it.user.userId != selfId }
    val loadingAll = peopleState.loading && peopleState.items.isEmpty() && momentsState.items.isEmpty()
    val loadFailed = peopleState.error != null && peopleState.items.isEmpty()

    Column(Modifier.fillMaxSize()) {
        // 顶部：标题 + 定位胶囊 + 搜索/地图
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = Spacing.lg, vertical = Spacing.md),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = "附近",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.SemiBold,
                color = colors.textPrimary,
            )
            Spacer(Modifier.width(Spacing.sm))
            NearbyLocationChip(locationLabel)
            Spacer(Modifier.weight(1f))
            HeaderIconButton(
                icon = Icons.Filled.Search,
                contentDescription = "搜索",
                onClick = onOpenSearch,
            )
            HeaderIconButton(
                icon = Icons.Filled.Place,
                contentDescription = "附近地图",
                onClick = {
                    Toast.makeText(context, "地图原生渲染 TODO（首版占位）", Toast.LENGTH_SHORT).show()
                },
            )
        }
        when {
            loadingAll -> LoadingState(message = "正在看看附近…")
            loadFailed -> ErrorState(
                message = peopleState.error ?: "加载失败",
                onRetry = { refreshKey["tick"] = refreshTick + 1 },
            )
            else -> Column(
                modifier = Modifier
                    .fillMaxSize()
                    .verticalScroll(rememberScrollState())
                    .padding(horizontal = Spacing.lg),
            ) {
                NearbyStatsCard(
                    subtitle = locationLabel,
                    peopleCount = if (peopleState.loaded) nearbyPeople.size.toString() else "–",
                    momentsCount = if (momentsState.loaded) momentsState.items.size.toString() else "–",
                    onlineCount = nearbyPeople.count { it.presenceStatus == "online" }.toString(),
                )
                Spacer(Modifier.height(Spacing.lg))
                NearbySectionHead(
                    title = "附近的人",
                    count = nearbyPeople.size.takeIf { peopleState.loaded },
                    onSeeAll = onOpenPeople,
                )
                if (nearbyPeople.isEmpty()) {
                    if (peopleState.loaded) {
                        EmptyState(title = "附近暂时没有人", description = "稍后再来看看")
                    }
                } else {
                    nearbyPeople.take(4).forEach { user ->
                        NearbyUserRow(
                            user = user,
                            following = followStates[user.user.userId] ?: (user.relation?.isFollowing ?: false),
                            followBusy = followBusy[user.user.userId] == true,
                            onToggleFollow = { toggleFollow(user) },
                        )
                    }
                }
                Spacer(Modifier.height(Spacing.lg))
                NearbySectionHead(
                    title = "附近瞬间",
                    count = momentsState.items.size.takeIf { momentsState.loaded },
                    onSeeAll = onOpenMoments,
                )
                NearbyMomentPreviewGrid(
                    moments = momentsState.items.take(4),
                    onOpenMoment = onOpenMoment,
                )
                Spacer(Modifier.height(Spacing.lg))
                NearbySectionHead(
                    title = "附近群组",
                    count = groupsState.items.size.takeIf { groupsState.loaded },
                    onSeeAll = onOpenGroups,
                )
                NearbyGroupPreviewGrid(
                    groups = groupsState.items.take(2),
                    joinedStates = joinedStates,
                    pendingStates = pendingStates,
                    joinBusy = joinBusy,
                    onOpenGroup = onOpenGroup,
                    onJoin = { joinGroup(it) },
                )
                Spacer(Modifier.height(Spacing.xxl))
            }
        }
    }
}

@Composable
private fun NearbyMomentPreviewGrid(
    moments: List<MomentItem>,
    onOpenMoment: (String) -> Unit,
) {
    if (moments.isEmpty()) return
    Column(verticalArrangement = Arrangement.spacedBy(Spacing.md)) {
        moments.chunked(2).forEach { row ->
            Row(horizontalArrangement = Arrangement.spacedBy(Spacing.md)) {
                row.forEach { moment ->
                    NearbyMomentCard(
                        moment = moment,
                        onClick = { onOpenMoment(moment.momentId) },
                        modifier = Modifier.weight(1f),
                    )
                }
                if (row.size == 1) Spacer(Modifier.weight(1f))
            }
        }
    }
}

@Composable
private fun NearbyGroupPreviewGrid(
    groups: List<GroupItem>,
    joinedStates: Map<String, Boolean>,
    pendingStates: Map<String, Boolean>,
    joinBusy: Map<String, Boolean>,
    onOpenGroup: (String) -> Unit,
    onJoin: (GroupItem) -> Unit,
) {
    if (groups.isEmpty()) return
    Row(horizontalArrangement = Arrangement.spacedBy(Spacing.md)) {
        groups.forEach { group ->
            NearbyGroupCard(
                group = group,
                joined = joinedStates[group.group.groupId] ?: (group.viewerState != null),
                pending = pendingStates[group.group.groupId] == true,
                busy = joinBusy[group.group.groupId] == true,
                onClick = { onOpenGroup(group.group.groupId) },
                onJoin = { onJoin(group) },
                modifier = Modifier.weight(1f),
            )
        }
        if (groups.size == 1) Spacer(Modifier.weight(1f))
    }
}
