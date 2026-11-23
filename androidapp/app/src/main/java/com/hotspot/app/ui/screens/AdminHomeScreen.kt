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
    onOpenGroups: () -> Unit,
    onOpenAnnouncements: () -> Unit,
    onOpenStats: () -> Unit,
    onOpenRbac: () -> Unit,
    onOpenTags: () -> Unit,
    onOpenAudit: () -> Unit,
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
                    ListRow(
                        title = "群组管理",
                        subtitle = "搜索群组、查看群主与成员数、解散违规群组",
                        onClick = onOpenGroups,
                    )
                    ListRow(
                        title = "系统公告",
                        subtitle = "发布、编辑与删除全站公告",
                        onClick = onOpenAnnouncements,
                    )
                    ListRow(
                        title = "统计分析",
                        subtitle = "全站存量、新增趋势与热门地点/区域排名",
                        onClick = onOpenStats,
                    )
                    ListRow(
                        title = "权限与角色",
                        subtitle = "查看平台管理员、RBAC 角色与权限字典",
                        onClick = onOpenRbac,
                    )
                    ListRow(
                        title = "标签管理",
                        subtitle = "搜索标签、查看使用次数、软屏蔽与解屏蔽",
                        onClick = onOpenTags,
                    )
                    ListRow(
                        title = "审计日志",
                        subtitle = "查看平台操作留痕：操作者、动作、对象与详情",
                        onClick = onOpenAudit,
                    )
                }
            }
            item { Spacer(Modifier.height(Spacing.lg)) }
        }
    }
}