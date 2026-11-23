package com.hotspot.app.ui.theme

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

/**
 * 页面级系统栏/安全区覆盖状态（仅用于「头图通栏」这类沉浸页面）。
 *
 * 应用根默认给所有页面加 safeDrawing Top 内边距，普通纯色页面状态栏与背景同色、视觉连续；
 * 但带通栏头图的详情页若仍内缩，状态栏会露出一条页面底色，与图片割裂。
 * 沉浸页面进入时置 [edgeToEdgeTop]（根不再加 Top 内边距，内容从屏幕顶开始），
 * 并可按头部媒体设置 [lightStatusBarIcons]（图片上强制浅色状态栏图标，配合顶部压暗渐变）。
 * 页面销毁时复位，其他页面恢复默认内边距。
 */
object ChromeInsets {
    var edgeToEdgeTop by mutableStateOf(false)
    var lightStatusBarIcons by mutableStateOf(false)
}
