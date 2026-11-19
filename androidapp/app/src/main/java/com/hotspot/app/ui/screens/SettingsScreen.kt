package com.hotspot.app.ui.screens

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.hotspot.app.AppContainer
import com.hotspot.app.BuildConfig
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.AccountData
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.ListGroup
import com.hotspot.app.ui.components.ListRow
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SectionTitle
import com.hotspot.app.ui.components.SegmentedTabs
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemCard
import com.hotspot.app.ui.theme.DarkModePref
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.ui.theme.ThemeId
import com.hotspot.app.util.originLabel
import kotlinx.coroutines.launch

/**
 * 设置：主题族（vibrant/forest/inkwash）+ 明暗三档（跟随系统/浅色/深色）+ 账号 + 关于。
 * 主题偏好持久化在 ThemeController（SharedPreferences）；切换即时作用于 MaterialTheme。
 */
@Composable
fun SettingsScreen(onBack: () -> Unit) {
    val colors = HotSpotTheme.colors
    val themeController = AppContainer.themeController
    val uiTheme by themeController.state.collectAsStateWithLifecycle()
    val scope = rememberCoroutineScope()
    var account by remember { mutableStateOf<AccountData?>(null) }
    var showLogoutConfirm by remember { mutableStateOf(false) }
    var loggingOut by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
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
            SectionTitle(title = "外观")
            Spacer(Modifier.height(Spacing.sm))
            SystemCard {
                Text(
                    text = "主题",
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.textSecondary,
                )
                Spacer(Modifier.height(Spacing.sm2))
                SegmentedTabs(
                    options = ThemeId.entries.map { it.label },
                    selectedIndex = ThemeId.entries.indexOf(uiTheme.themeId),
                    onSelected = { themeController.setThemeId(ThemeId.entries[it]) },
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(Spacing.lg))
                Text(
                    text = "明暗模式",
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.textSecondary,
                )
                Spacer(Modifier.height(Spacing.sm2))
                SegmentedTabs(
                    options = DarkModePref.entries.map { it.label },
                    selectedIndex = DarkModePref.entries.indexOf(uiTheme.darkMode),
                    onSelected = { themeController.setDarkMode(DarkModePref.entries[it]) },
                    modifier = Modifier.fillMaxWidth(),
                )
            }
            Spacer(Modifier.height(Spacing.xl))
            SectionTitle(title = "账号")
            Spacer(Modifier.height(Spacing.sm))
            ListGroup {
                ListRow(
                    title = "当前账号",
                    subtitle = account?.let { data ->
                        listOfNotNull(data.nickname, data.email, data.phone).joinToString(" · ")
                    } ?: "加载中…",
                    onClick = null,
                )
                ListRow(
                    title = "退出登录",
                    subtitle = "退出后需要重新登录才能继续使用",
                    onClick = { showLogoutConfirm = true },
                )
            }
            Spacer(Modifier.height(Spacing.xl))
            SectionTitle(title = "关于")
            Spacer(Modifier.height(Spacing.sm))
            ListGroup {
                ListRow(title = "版本", subtitle = BuildConfig.VERSION_NAME, onClick = null)
                ListRow(title = "构建类型", subtitle = BuildConfig.BUILD_TYPE, onClick = null)
                ListRow(title = "API 地址", subtitle = originLabel(BuildConfig.API_BASE_URL), onClick = null)
                ListRow(
                    title = "首版未接入",
                    subtitle = "地图原生渲染 / 图片上传 / WebSocket 实时 / 管理后台（TODO）",
                    onClick = null,
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
                    variant = ButtonVariant.SOFT,
                    size = ButtonSize.SM,
                )
            },
        )
    }
}
