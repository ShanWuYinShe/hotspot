package com.hotspot.app.ui.screens

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.PageHeader

/**
 * 发布（登记 TODO）：首版不做图片上传 / 媒体选择，明确占位而不是假表单。
 * 后续接入 media/presign_upload + media/complete_upload + moments/create。
 */
@Composable
fun PublishScreen() {
    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "发布", subtitle = "分享此刻")
        Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            EmptyState(
                title = "发布功能首版占位",
                description = "图片上传与瞬间发布链路（media/presign_upload → complete_upload → moments/create）为已登记 TODO，后续版本接入。",
                actionLabel = null,
                onAction = null,
            )
        }
    }
}
