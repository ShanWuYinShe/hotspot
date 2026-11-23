package com.hotspot.app.ui.screens

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material3.Text
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.hotspot.app.ui.components.ListGroup
import com.hotspot.app.ui.components.ListRow
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing

/** 管理后台首页（仅平台管理员）：举报审核 / 用户管理。 */
@Composable
fun AdminHomeScreen(
    onBack: () -> Unit,
    onOpenReports: () -> Unit,
    onOpenUsers: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "管理后台", subtitle = "平台管理", onBack = onBack)
        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = androidx.compose.foundation.layout.PaddingValues(
                horizontal = Spacing.lg,
                vertical = Spacing.sm,
            ),
        ) {
            item {
                Text(
                    text = "仅平台管理员可见；操作会记录审计日志并通知相关用户。",
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                    modifier = Modifier.padding(vertical = Spacing.sm, horizontal = Spacing.xs),
                )
            }
            item {
                ListGroup {
                    ListRow(
                        title = "举报审核",
                        subtitle = "处理待审核举报：通过或驳回，并通知举报人",
                        onClick = onOpenReports,
                    )
                    ListRow(
                        title = "用户管理",
                        subtitle = "搜索用户、查看账号状态、封禁与解封",
                        onClick = onOpenUsers,
                    )
                }
            }
            item { Spacer(Modifier.height(Spacing.lg)) }
        }
    }
}
