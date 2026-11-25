package com.hotspot.app

import android.app.Activity
import android.os.Build
import android.os.Bundle
import android.view.View
import android.view.WindowInsetsController
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.WindowInsetsSides
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.only
import androidx.compose.foundation.layout.safeDrawing
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.runtime.Composable
import androidx.compose.runtime.SideEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalView
import androidx.core.view.WindowCompat
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.hotspot.app.navigation.HotSpotApp
import com.hotspot.app.ui.components.AmbientCanvas
import com.hotspot.app.ui.theme.ChromeInsets
import com.hotspot.app.ui.theme.DarkModePref
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.ThemeController

/**
 * 唯一 Activity：edge-to-edge + Compose 根。
 * 背景氛围画布（AmbientCanvas）铺满整窗，内容区按安全区 inset 约束；
 * 主题切换通过 ThemeController StateFlow 驱动重组，系统栏明暗随主题同步。
 */
class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        // API 29 及以下 enableEdgeToEdge 依赖 systemUiVisibility 的 layout flags；
        // 后续设置状态栏图标明暗时若整体覆盖这些 flags，窗口会重新为状态栏让位，
        // 根布局再叠一次 safeDrawing Top，表现为顶部多出一整个状态栏高度。
        WindowCompat.setDecorFitsSystemWindows(window, false)
        val themeController: ThemeController = AppContainer.themeController
        setContent {
            val uiTheme by themeController.state.collectAsStateWithLifecycle()
            val systemDark = isSystemInDarkTheme()
            val dark = when (uiTheme.darkMode) {
                DarkModePref.AUTO -> systemDark
                DarkModePref.LIGHT -> false
                DarkModePref.DARK -> true
            }
            // 沉浸头图页：图片铺到状态栏下并压暗，状态栏图标强制浅色；其余页面跟随主题。
            val lightStatusBarIcons = ChromeInsets.lightStatusBarIcons
            SystemBarsAppearance(dark = lightStatusBarIcons || dark)
            HotSpotTheme(uiTheme = uiTheme) {
                Box(Modifier.fillMaxSize()) {
                    AmbientCanvas(Modifier.fillMaxSize())
                    val immersiveTop = ChromeInsets.edgeToEdgeTop
                    val contentInsets = if (immersiveTop) {
                        WindowInsets.safeDrawing.only(WindowInsetsSides.Horizontal)
                    } else {
                        WindowInsets.safeDrawing.only(WindowInsetsSides.Top + WindowInsetsSides.Horizontal)
                    }
                    Column(
                        Modifier
                            .fillMaxSize()
                            .windowInsetsPadding(contentInsets),
                    ) {
                        HotSpotApp()
                    }
                }
            }
        }
    }

    /**
     * 回前台补连 WebSocket（spec「重连触发条件」：前台切换）；连接已在线时该方法无操作。
     * 退后台不主动断连：消息类推送语义要求尽量保持在线，由系统进程生命周期回收。
     */
    override fun onStart() {
        super.onStart()
        AppContainer.realtimeClient.onForeground()
    }
}

/** 状态栏/导航栏图标明暗（亮色主题用深色图标，暗色主题用浅色图标）。 */
@Composable
private fun SystemBarsAppearance(dark: Boolean) {
    val view = LocalView.current
    if (view.isInEditMode) return
    SideEffect {
        val window = (view.context as? Activity)?.window ?: return@SideEffect
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            val mask = WindowInsetsController.APPEARANCE_LIGHT_STATUS_BARS or
                WindowInsetsController.APPEARANCE_LIGHT_NAVIGATION_BARS
            window.insetsController?.setSystemBarsAppearance(if (dark) 0 else mask, mask)
        } else {
            @Suppress("DEPRECATION")
            val lightFlags = View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR or View.SYSTEM_UI_FLAG_LIGHT_NAVIGATION_BAR
            // 保留 enableEdgeToEdge 设置的 layout flags：整体覆盖会让窗口重新内缩状态栏，
            // 根布局的 safeDrawing Top 就会变成第二份状态栏留白。
            var flags = window.decorView.systemUiVisibility or
                View.SYSTEM_UI_FLAG_LAYOUT_STABLE or
                View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN or
                View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION
            flags = if (dark) flags and lightFlags.inv() else flags or lightFlags
            window.decorView.systemUiVisibility = flags
        }
    }
}
