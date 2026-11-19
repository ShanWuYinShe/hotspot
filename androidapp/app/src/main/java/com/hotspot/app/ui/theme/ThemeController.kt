package com.hotspot.app.ui.theme

import android.content.Context
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * 主题偏好持久化（主题族 + 明暗三档）：与 Web 主题选择语义一致。
 * 用 SharedPreferences（非敏感偏好，无需加密）。
 */
class ThemeController(context: Context) {
    private val prefs = context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    private val _state = MutableStateFlow(load())
    val state: StateFlow<UiTheme> = _state.asStateFlow()

    private fun load(): UiTheme = UiTheme(
        themeId = ThemeId.fromKey(prefs.getString(KEY_THEME_ID, null)),
        darkMode = DarkModePref.fromKey(prefs.getString(KEY_DARK_MODE, null)),
    )

    fun setThemeId(themeId: ThemeId) {
        prefs.edit().putString(KEY_THEME_ID, themeId.key).apply()
        _state.value = _state.value.copy(themeId = themeId)
    }

    fun setDarkMode(darkMode: DarkModePref) {
        prefs.edit().putString(KEY_DARK_MODE, darkMode.key).apply()
        _state.value = _state.value.copy(darkMode = darkMode)
    }

    private companion object {
        const val PREFS_NAME = "hotspot_theme"
        const val KEY_THEME_ID = "theme_id"
        const val KEY_DARK_MODE = "dark_mode"
    }
}
