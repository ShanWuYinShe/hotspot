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
import com.hotspot.app.ui.screens.ChatScreen
import com.hotspot.app.ui.screens.HomeScreen
import com.hotspot.app.ui.screens.LoginScreen
import com.hotspot.app.ui.screens.MeGroupsScreen
import com.hotspot.app.ui.screens.MeMomentsScreen
import com.hotspot.app.ui.screens.MeScreen
import com.hotspot.app.ui.screens.MessagesScreen
import com.hotspot.app.ui.screens.MomentDetailScreen
import com.hotspot.app.ui.screens.NearbyScreen
import com.hotspot.app.ui.screens.NotifyScreen
import com.hotspot.app.ui.screens.PublishScreen
import com.hotspot.app.ui.screens.SearchScreen
import com.hotspot.app.ui.screens.SettingsScreen

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
    val backStackEntry by navController.currentBackStackEntryAsState()
    val currentRoute = backStackEntry?.destination?.route
    val showBottomBar = currentRoute != null && currentRoute in Routes.tabRoutes

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
                    )
                }
                composable(Routes.NEARBY) {
                    NearbyScreen(
                        onOpenSearch = { navController.navigate(Routes.SEARCH) },
                        onOpenMessages = { navController.navigate(Routes.MESSAGES) },
                    )
                }
                composable(Routes.PUBLISH) { PublishScreen() }
                composable(Routes.MESSAGES) {
                    MessagesScreen(
                        onOpenNotify = { navController.navigate(Routes.NOTIFY) },
                        onOpenChat = { userId, nickname ->
                            ChatTarget.select(userId, nickname)
                            navController.navigate(Routes.chat(userId))
                        },
                    )
                }
                composable(Routes.ME) {
                    MeScreen(
                        onOpenSettings = { navController.navigate(Routes.SETTINGS) },
                        onOpenNotify = { navController.navigate(Routes.NOTIFY) },
                        onOpenMoments = { navController.navigate(Routes.ME_MOMENTS) },
                        onOpenGroups = { navController.navigate(Routes.ME_GROUPS) },
                    )
                }
                composable(Routes.ME_MOMENTS) {
                    MeMomentsScreen(
                        onBack = { navController.popBackStack() },
                        onOpenMoment = { momentId -> navController.navigate(Routes.moment(momentId)) },
                    )
                }
                composable(Routes.ME_GROUPS) {
                    MeGroupsScreen(onBack = { navController.popBackStack() })
                }
                composable(Routes.SEARCH) {
                    SearchScreen(onBack = { navController.popBackStack() })
                }
                composable(Routes.NOTIFY) {
                    NotifyScreen(onBack = { navController.popBackStack() })
                }
                composable(Routes.SETTINGS) {
                    SettingsScreen(onBack = { navController.popBackStack() })
                }
                composable(
                    route = Routes.MOMENT_DETAIL,
                    arguments = listOf(navArgument("momentId") { type = NavType.StringType }),
                ) { entry ->
                    MomentDetailScreen(
                        momentId = entry.arguments?.getString("momentId").orEmpty(),
                        onBack = { navController.popBackStack() },
                    )
                }
                composable(
                    route = Routes.CHAT,
                    arguments = listOf(navArgument("userId") { type = NavType.StringType }),
                ) {
                    ChatScreen(onBack = { navController.popBackStack() })
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
