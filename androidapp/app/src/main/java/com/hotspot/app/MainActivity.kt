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
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.hotspot.app.navigation.HotSpotApp
import com.hotspot.app.ui.components.AmbientCanvas
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
        val themeController: ThemeController = AppContainer.themeController
        setContent {
            val uiTheme by themeController.state.collectAsStateWithLifecycle()
            val systemDark = isSystemInDarkTheme()
            val dark = when (uiTheme.darkMode) {
                DarkModePref.AUTO -> systemDark
                DarkModePref.LIGHT -> false
                DarkModePref.DARK -> true
            }
            SystemBarsAppearance(dark = dark)
            HotSpotTheme(uiTheme = uiTheme) {
                Box(Modifier.fillMaxSize()) {
                    AmbientCanvas(Modifier.fillMaxSize())
                    Column(
                        Modifier
                            .fillMaxSize()
                            .windowInsetsPadding(
                                WindowInsets.safeDrawing.only(
                                    WindowInsetsSides.Top + WindowInsetsSides.Horizontal,
                                ),
                            ),
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
            window.decorView.systemUiVisibility = if (dark) {
                0
            } else {
                View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR or View.SYSTEM_UI_FLAG_LIGHT_NAVIGATION_BAR
            }
        }
    }
}
