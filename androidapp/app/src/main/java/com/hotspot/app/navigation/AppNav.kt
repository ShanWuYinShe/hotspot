package com.hotspot.app.navigation

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.hotspot.app.AppContainer
import com.hotspot.app.ui.components.BottomNav
import com.hotspot.app.ui.screens.AdminAnnouncementsScreen
import com.hotspot.app.ui.screens.AdminGroupsScreen
import com.hotspot.app.ui.screens.AdminHomeScreen
import com.hotspot.app.ui.screens.AdminReportsScreen
import com.hotspot.app.ui.screens.AdminUsersScreen
import com.hotspot.app.ui.screens.BlockedUsersScreen
import com.hotspot.app.ui.screens.ChatScreen
import com.hotspot.app.ui.screens.EditProfileScreen
import com.hotspot.app.ui.screens.GroupChatScreen
import com.hotspot.app.ui.screens.GroupDetailScreen
import com.hotspot.app.ui.screens.HomeScreen
import com.hotspot.app.ui.screens.LoginScreen
import com.hotspot.app.ui.screens.MeGroupsScreen
import com.hotspot.app.ui.screens.MeMomentsScreen
import com.hotspot.app.ui.screens.MeScreen
import com.hotspot.app.ui.screens.MessagesScreen
import com.hotspot.app.ui.screens.MomentDetailScreen
import com.hotspot.app.ui.screens.MapScreen
import com.hotspot.app.ui.screens.NearbyGroupsScreen
import com.hotspot.app.ui.screens.NearbyMomentsScreen
import com.hotspot.app.ui.screens.NearbyPeopleScreen
import com.hotspot.app.ui.screens.NearbyScreen
import com.hotspot.app.ui.screens.NotifyScreen
import com.hotspot.app.ui.screens.PublishScreen
import com.hotspot.app.ui.screens.RelationshipsScreen
import com.hotspot.app.ui.screens.ReportsScreen
import com.hotspot.app.ui.screens.SearchScreen
import com.hotspot.app.ui.screens.SettingsScreen
import com.hotspot.app.ui.screens.UserProfileScreen

/**
 * 根组件：登录态门禁 —— 未登录（TokenStore.session == null）只渲染登录页；
 * 登录成功后自动进入主界面，退出登录后自动回到登录页（无需手动导航）。
 */
@Composable
fun HotSpotApp() {
    val session by AppContainer.tokenStore.session.collectAsStateWithLifecycle()
    if (session == null) {
        LoginScreen(onLoggedIn = { })
    } else {
        MainScaffold()
    }
}

@Composable
private fun MainScaffold() {
    val navController = rememberNavController()
    val selfUserId = AppContainer.tokenStore.session.value?.userId
    // 他人主页入口：点到自己时回个人中心，避免出现「关注自己」这类无意义动作
    fun openUser(userId: String) {
        if (selfUserId != null && userId == selfUserId) {
            navController.navigate(Routes.ME)
        } else {
            navController.navigate(Routes.user(userId))
        }
    }
    val backStackEntry by navController.currentBackStackEntryAsState()
    val currentRoute = backStackEntry?.destination?.route
    val showBottomBar = currentRoute != null && currentRoute in Routes.bottomBarRoutes

    Column(Modifier.fillMaxSize()) {
        Box(Modifier.weight(1f)) {
            NavHost(
                navController = navController,
                startDestination = Routes.HOME,
                modifier = Modifier.fillMaxSize(),
            ) {
                composable(Routes.HOME) {
                    HomeScreen(
                        onOpenSearch = { navController.navigate(Routes.SEARCH) },
                        onOpenNotify = { navController.navigate(Routes.NOTIFY) },
                        onOpenMoment = { momentId -> navController.navigate(Routes.moment(momentId)) },
                        onOpenUser = { userId -> openUser(userId) },
                    )
                }
                composable(Routes.NEARBY) {
                    NearbyScreen(
                        onOpenSearch = { navController.navigate(Routes.SEARCH) },
                        onOpenPeople = { navController.navigate(Routes.NEARBY_PEOPLE) },
                        onOpenGroups = { navController.navigate(Routes.NEARBY_GROUPS) },
                        onOpenMoments = { navController.navigate(Routes.NEARBY_MOMENTS) },
                        onOpenMoment = { momentId -> navController.navigate(Routes.moment(momentId)) },
                        onOpenGroup = { groupId -> navController.navigate(Routes.group(groupId)) },
                        onOpenUser = { userId -> openUser(userId) },
                        onOpenMap = { navController.navigate(Routes.NEARBY_MAP) },
                    )
                }
                composable(Routes.NEARBY_PEOPLE) {
                    NearbyPeopleScreen(
                        onBack = { navController.popBackStack() },
                        onOpenSearch = { navController.navigate(Routes.SEARCH) },
                        onOpenUser = { userId -> openUser(userId) },
                        onOpenMap = { navController.navigate(Routes.NEARBY_MAP) },
                    )
                }
                composable(Routes.NEARBY_GROUPS) {
                    NearbyGroupsScreen(
                        onBack = { navController.popBackStack() },
                        onOpenSearch = { navController.navigate(Routes.SEARCH) },
                        onOpenGroup = { groupId -> navController.navigate(Routes.group(groupId)) },
                        onOpenMap = { navController.navigate(Routes.NEARBY_MAP) },
                    )
                }
                composable(Routes.NEARBY_MOMENTS) {
                    NearbyMomentsScreen(
                        onBack = { navController.popBackStack() },
                        onOpenSearch = { navController.navigate(Routes.SEARCH) },
                        onOpenMoment = { momentId -> navController.navigate(Routes.moment(momentId)) },
                        onOpenUser = { userId -> openUser(userId) },
                        onOpenMap = { navController.navigate(Routes.NEARBY_MAP) },
                    )
                }
                composable(Routes.NEARBY_MAP) {
                    MapScreen(
                        onBack = { navController.popBackStack() },
                        onAnchorSaved = {
                            if (navController.previousBackStackEntry?.destination?.route == Routes.SETTINGS) {
                                // 从设置进入选点：保存后回到设置，让「地图与位置」区块即时刷新。
                                navController.popBackStack()
                            } else {
                                // 锚点变更后重建附近枢纽，让距离/推荐按新锚点重新拉取。
                                navController.popBackStack(Routes.NEARBY, inclusive = true)
                                navController.navigate(Routes.NEARBY) { launchSingleTop = true }
                            }
                        },
                    )
                }
                composable(Routes.PUBLISH) { PublishScreen() }
                composable(Routes.MESSAGES) {
                    MessagesScreen(
                        onOpenChat = { userId, nickname ->
                            ChatTarget.select(userId, nickname)
                            navController.navigate(Routes.chat(userId))
                        },
                        onOpenGroupChat = { item ->
                            GroupChatTarget.select(
                                groupId = item.group.groupId,
                                name = item.group.name,
                                avatarUrl = item.group.avatarUrl,
                                membersCount = item.group.membersCount,
                                onlineCount = item.onlineMembersCount ?: 0,
                            )
                            navController.navigate(Routes.groupChat(item.group.groupId))
                        },
                    )
                }
                composable(Routes.ME) {
                    MeScreen(
                        onOpenSettings = { navController.navigate(Routes.SETTINGS) },
                        onOpenNotify = { navController.navigate(Routes.NOTIFY) },
                        onOpenMoments = { navController.navigate(Routes.ME_MOMENTS) },
                        onOpenGroups = { navController.navigate(Routes.ME_GROUPS) },
                        onOpenEditProfile = { navController.navigate(Routes.ME_EDIT) },
                        onOpenRelationships = { tab -> navController.navigate(Routes.relationships(tab)) },
                    )
                }
                composable(Routes.ME_EDIT) {
                    EditProfileScreen(onBack = { navController.popBackStack() })
                }
                composable(
                    route = Routes.RELATIONSHIPS,
                    arguments = listOf(navArgument("tab") { type = NavType.IntType; defaultValue = 0 }),
                ) { entry ->
                    RelationshipsScreen(
                        initialTab = entry.arguments?.getInt("tab") ?: 0,
                        onBack = { navController.popBackStack() },
                        onOpenChat = { userId -> navController.navigate(Routes.chat(userId)) },
                        onOpenUser = { userId -> openUser(userId) },
                    )
                }
                composable(Routes.ME_MOMENTS) {
                    MeMomentsScreen(
                        onBack = { navController.popBackStack() },
                        onOpenMoment = { momentId -> navController.navigate(Routes.moment(momentId)) },
                    )
                }
                composable(Routes.ME_GROUPS) {
                    MeGroupsScreen(
                        onBack = { navController.popBackStack() },
                        onOpenGroup = { groupId -> navController.navigate(Routes.group(groupId)) },
                    )
                }
                composable(Routes.SEARCH) {
                    SearchScreen(
                        onBack = { navController.popBackStack() },
                        onOpenGroup = { groupId -> navController.navigate(Routes.group(groupId)) },
                        onOpenUser = { userId -> openUser(userId) },
                        onOpenMoment = { momentId -> navController.navigate(Routes.moment(momentId)) },
                    )
                }
                composable(Routes.NOTIFY) {
                    NotifyScreen(
                        onBack = { navController.popBackStack() },
                        onOpenUser = { userId -> openUser(userId) },
                        onOpenMoment = { momentId -> navController.navigate(Routes.moment(momentId)) },
                        onOpenMomentComment = { momentId, commentId ->
                            navController.navigate(Routes.moment(momentId, commentId))
                        },
                        onOpenGroup = { groupId -> navController.navigate(Routes.group(groupId)) },
                        onOpenGroupChat = { groupId -> navController.navigate(Routes.groupChat(groupId)) },
                        onOpenReports = { navController.navigate(Routes.ME_REPORTS) },
                    )
                }
                composable(Routes.SETTINGS) {
                    SettingsScreen(
                        onBack = { navController.popBackStack() },
                        onOpenEditProfile = { navController.navigate(Routes.ME_EDIT) },
                        onOpenReports = { navController.navigate(Routes.ME_REPORTS) },
                        onOpenBlocked = { navController.navigate(Routes.ME_BLOCKED) },
                        onOpenAdmin = { navController.navigate(Routes.ADMIN_HOME) },
                        onOpenMap = { navController.navigate(Routes.NEARBY_MAP) },
                    )
                }
                composable(Routes.ME_REPORTS) {
                    ReportsScreen(onBack = { navController.popBackStack() })
                }
                composable(Routes.ME_BLOCKED) {
                    BlockedUsersScreen(
                        onBack = { navController.popBackStack() },
                        onOpenUser = { userId -> openUser(userId) },
                    )
                }
                composable(Routes.ADMIN_HOME) {
                    AdminHomeScreen(
                        onBack = { navController.popBackStack() },
                        onOpenReports = { navController.navigate(Routes.ADMIN_REPORTS) },
                        onOpenUsers = { navController.navigate(Routes.ADMIN_USERS) },
                        onOpenGroups = { navController.navigate(Routes.ADMIN_GROUPS) },
                        onOpenAnnouncements = { navController.navigate(Routes.ADMIN_ANNOUNCEMENTS) },
                    )
                }
                composable(Routes.ADMIN_REPORTS) {
                    AdminReportsScreen(onBack = { navController.popBackStack() })
                }
                composable(Routes.ADMIN_USERS) {
                    AdminUsersScreen(onBack = { navController.popBackStack() })
                }
                composable(Routes.ADMIN_GROUPS) {
                    AdminGroupsScreen(onBack = { navController.popBackStack() })
                }
                composable(Routes.ADMIN_ANNOUNCEMENTS) {
                    AdminAnnouncementsScreen(onBack = { navController.popBackStack() })
                }
                composable(
                    route = Routes.MOMENT_DETAIL,
                    arguments = listOf(
                        navArgument("momentId") { type = NavType.StringType },
                        navArgument("commentId") {
                            type = NavType.StringType
                            nullable = true
                            defaultValue = null
                        },
                    ),
                ) { entry ->
                    MomentDetailScreen(
                        momentId = entry.arguments?.getString("momentId").orEmpty(),
                        focusCommentId = entry.arguments?.getString("commentId"),
                        onBack = { navController.popBackStack() },
                        onOpenUser = { userId -> openUser(userId) },
                    )
                }
                composable(
                    route = Routes.USER,
                    arguments = listOf(navArgument("userId") { type = NavType.StringType }),
                ) { entry ->
                    UserProfileScreen(
                        userId = entry.arguments?.getString("userId").orEmpty(),
                        onBack = { navController.popBackStack() },
                        onOpenMoment = { momentId -> navController.navigate(Routes.moment(momentId)) },
                        onOpenMessage = { userId ->
                            ChatTarget.select(userId, "")
                            navController.navigate(Routes.chat(userId))
                        },
                    )
                }
                composable(
                    route = Routes.CHAT,
                    arguments = listOf(navArgument("userId") { type = NavType.StringType }),
                ) {
                    ChatScreen(onBack = { navController.popBackStack() })
                }
                composable(
                    route = Routes.GROUP_CHAT,
                    arguments = listOf(navArgument("groupId") { type = NavType.StringType }),
                ) {
                    GroupChatScreen(
                        onBack = {
                            GroupChatTarget.clear()
                            navController.popBackStack()
                        },
                    )
                }
                composable(
                    route = Routes.GROUP,
                    arguments = listOf(navArgument("groupId") { type = NavType.StringType }),
                ) { entry ->
                    GroupDetailScreen(
                        groupId = entry.arguments?.getString("groupId").orEmpty(),
                        onBack = { navController.popBackStack() },
                        onOpenGroupChat = { item ->
                            GroupChatTarget.select(
                                groupId = item.group.groupId,
                                name = item.group.name,
                                avatarUrl = item.group.avatarUrl,
                                membersCount = item.group.membersCount,
                                onlineCount = item.onlineMembersCount ?: 0,
                            )
                            navController.navigate(Routes.groupChat(item.group.groupId))
                        },
                    )
                }
            }
        }
        if (showBottomBar) {
            BottomNav(
                currentRoute = currentRoute,
                onNavigate = { route ->
                    navController.navigate(route) {
                        popUpTo(navController.graph.findStartDestination().id) { saveState = true }
                        launchSingleTop = true
                        restoreState = true
                    }
                },
            )
        }
    }
}
