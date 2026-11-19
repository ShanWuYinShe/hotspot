package com.hotspot.app.ui.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiError
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.GroupItem
import com.hotspot.app.data.model.MomentItem
import com.hotspot.app.data.model.Page
import com.hotspot.app.data.model.UserItem
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.Badge
import com.hotspot.app.ui.components.BadgeSize
import com.hotspot.app.ui.components.BadgeTone
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.HeaderIconButton
import com.hotspot.app.ui.components.ListGroup
import com.hotspot.app.ui.components.ListRow
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SystemCard
import com.hotspot.app.ui.components.TabPills
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.MediaUrls
import com.hotspot.app.util.TimeFormat

/**
 * 附近枢纽：附近的人 / 附近群组 / 附近瞬间三分区入口 + 地图占位。
 * 距离由后端计算，客户端只展示；地图原生渲染为已登记 TODO。
 */
@Composable
fun NearbyScreen(
    onOpenSearch: () -> Unit,
    onOpenMessages: () -> Unit,
) {
    var tabIndex by remember { mutableIntStateOf(0) }
    var refreshKey by remember { mutableIntStateOf(0) }
    val tabs = listOf("附近的人", "附近群组", "附近瞬间")

    Column(Modifier.fillMaxSize()) {
        PageHeader(
            title = "附近",
            subtitle = "看看附近正在发生什么",
            actions = {
                HeaderIconButton(
                    icon = Icons.Filled.Search,
                    contentDescription = "搜索",
                    onClick = onOpenSearch,
                )
                HeaderIconButton(
                    icon = Icons.Filled.Refresh,
                    contentDescription = "刷新附近",
                    onClick = { refreshKey++ },
                )
            },
        )
        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = Spacing.lg),
        ) {
            MapPlaceholderCard()
            Spacer(Modifier.height(Spacing.lg))
            TabPills(
                options = tabs,
                selectedIndex = tabIndex,
                onSelected = { tabIndex = it },
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(Spacing.md))
            when (tabIndex) {
                0 -> NearbyPeopleList(refreshKey)
                1 -> NearbyGroupsList(refreshKey)
                else -> NearbyMomentsList(refreshKey)
            }
            Spacer(Modifier.height(Spacing.xxl))
        }
    }
}

/** 地图占位（登记 TODO）：首版不做地图原生渲染，明确标注为占位而非假数据。 */
@Composable
private fun MapPlaceholderCard() {
    val colors = HotSpotTheme.colors
    SystemCard {
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(148.dp),
            contentAlignment = Alignment.Center,
        ) {
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text(
                    text = "附近地图",
                    style = MaterialTheme.typography.titleMedium,
                    color = colors.textPrimary,
                )
                Spacer(Modifier.height(Spacing.xs))
                Text(
                    text = "地图原生渲染 TODO（首版占位）\n距离与位置由后端计算，客户端只展示",
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.textSecondary,
                    textAlign = TextAlign.Center,
                )
            }
        }
    }
}

@Composable
private fun NearbyPeopleList(refreshKey: Int) {
    val state = rememberSingleLoadState<UserItem>()
    var localKey by remember { mutableIntStateOf(0) }
    LaunchedEffect(refreshKey, localKey) {
        state.load { loadFirstPage { cursor -> AppContainer.nearbyRepository.nearbyPeople(cursor) } }
    }
    when {
        state.loading && state.items.isEmpty() -> LoadingState(message = "正在寻找附近的人…")
        state.error != null && state.items.isEmpty() -> ErrorState(
            message = state.error ?: "加载失败",
            onRetry = { localKey++ },
        )
        state.items.isEmpty() -> EmptyState(title = "附近暂时没有人", description = "稍后再来看看")
        else -> ListGroup {
            state.items.forEach { user ->
                ListRow(
                    title = user.user.nickname.ifBlank { "匿名用户" },
                    subtitle = user.bio ?: presenceLabel(user.presenceStatus),
                    leading = {
                        Avatar(
                            name = user.user.nickname,
                            imageUrl = user.user.avatarUrl,
                            size = com.hotspot.app.ui.theme.Dimens.avatarMd,
                        )
                    },
                    trailing = {
                        val distance = user.location?.distance
                        if (distance != null) {
                            Badge(text = formatDistance(distance), tone = BadgeTone.NEUTRAL, size = BadgeSize.SM)
                        }
                    },
                )
            }
        }
    }
}

@Composable
private fun NearbyGroupsList(refreshKey: Int) {
    val state = rememberSingleLoadState<GroupItem>()
    var localKey by remember { mutableIntStateOf(0) }
    LaunchedEffect(refreshKey, localKey) {
        state.load { loadFirstPage { cursor -> AppContainer.nearbyRepository.nearbyGroups(cursor) } }
    }
    when {
        state.loading && state.items.isEmpty() -> LoadingState(message = "正在寻找附近群组…")
        state.error != null && state.items.isEmpty() -> ErrorState(message = state.error ?: "加载失败", onRetry = { localKey++ })
        state.items.isEmpty() -> EmptyState(title = "附近暂时没有群组", description = "稍后再来看看")
        else -> ListGroup {
            state.items.forEach { group ->
                ListRow(
                    title = group.group.name.ifBlank { "未命名群组" },
                    subtitle = group.description ?: (group.group.membersCount.toString() + " 位成员"),
                    leading = {
                        Avatar(
                            name = group.group.name,
                            imageUrl = group.group.avatarUrl,
                            size = com.hotspot.app.ui.theme.Dimens.avatarMd,
                            shape = AppShapes.compact,
                        )
                    },
                )
            }
        }
    }
}

@Composable
private fun NearbyMomentsList(refreshKey: Int) {
    val state = rememberSingleLoadState<MomentItem>()
    var localKey by remember { mutableIntStateOf(0) }
    LaunchedEffect(refreshKey, localKey) {
        state.load { loadFirstPage { cursor -> AppContainer.nearbyRepository.nearbyMoments(cursor) } }
    }
    when {
        state.loading && state.items.isEmpty() -> LoadingState(message = "正在寻找附近瞬间…")
        state.error != null && state.items.isEmpty() -> ErrorState(message = state.error ?: "加载失败", onRetry = { localKey++ })
        state.items.isEmpty() -> EmptyState(title = "附近暂时没有瞬间", description = "稍后再来看看")
        else -> ListGroup {
            state.items.forEach { moment ->
                ListRow(
                    title = moment.description.ifBlank { "无描述" },
                    subtitle = TimeFormat.relative(moment.createdAt) +
                        (moment.location?.locationName?.let { " · " + it } ?: ""),
                    leading = {
                        Avatar(
                            name = moment.user.nickname,
                            imageUrl = moment.images.firstOrNull()?.url ?: moment.user.avatarUrl,
                            size = com.hotspot.app.ui.theme.Dimens.avatarMd,
                            shape = AppShapes.compact,
                        )
                    },
                )
            }
        }
    }
}

private fun presenceLabel(status: String?): String = if (status == "online") "在线" else ""

private fun formatDistance(meters: Double): String = when {
    meters < 1000 -> meters.toInt().toString() + " m"
    else -> String.format("%.1f km", meters / 1000)
}
