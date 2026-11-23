package com.hotspot.app.ui.screens

import android.widget.Toast
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
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
import androidx.compose.ui.platform.LocalContext
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.AdminReportItem
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.ListGroup
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemInput
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.ReportCatalog
import com.hotspot.app.util.TimeFormat
import kotlinx.coroutines.launch

/**
 * 管理后台 · 举报审核（仅平台管理员）：拉取 pending 举报，逐条 resolve（确认有效）或 dismiss（驳回）。
 * 权限由后端 RBAC 校验；入口只在设置页对管理员显示。
 */
@Composable
fun AdminReportsScreen(onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val colors = HotSpotTheme.colors
    var items by remember { mutableStateOf<List<AdminReportItem>>(emptyList()) }
    var total by remember { mutableStateOf<Int?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var retryKey by remember { mutableStateOf(0) }
    var judgeTarget by remember { mutableStateOf<AdminReportItem?>(null) }
    var judgeAction by remember { mutableStateOf("dismiss") }
    var note by remember { mutableStateOf("") }
    var submitting by remember { mutableStateOf(false) }

    LaunchedEffect(retryKey) {
        loading = true
        error = null
        when (val result = AppContainer.adminRepository.fetchReports(status = "pending", cursor = null)) {
            is ApiResult.Ok -> {
                items = result.data.items
                total = result.data.total
            }
            is ApiResult.Err -> error = result.error.userMessage
        }
        loading = false
    }

    fun openJudge(report: AdminReportItem, action: String) {
        judgeTarget = report
        judgeAction = action
        note = ""
    }

    fun submitJudge() {
        val target = judgeTarget ?: return
        if (submitting) return
        submitting = true
        scope.launch {
            when (
                val result = AppContainer.adminRepository.judge(
                    reportId = target.reportId,
                    action = judgeAction,
                    note = note,
                )
            ) {
                is ApiResult.Ok -> {
                    items = items.filterNot { it.reportId == target.reportId }
                    total = total?.let { (it - 1).coerceAtLeast(0) }
                    judgeTarget = null
                    Toast.makeText(
                        context,
                        if (judgeAction == "resolve") "已确认举报并处理" else "已驳回举报",
                        Toast.LENGTH_SHORT,
                    ).show()
                }
                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
            }
            submitting = false
        }
    }

    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "管理后台", subtitle = "举报审核", onBack = onBack)
        when {
            loading -> LoadingState(message = "正在加载待处理举报…")
            error != null && items.isEmpty() -> ErrorState(message = error ?: "加载失败", onRetry = { retryKey += 1 })
            items.isEmpty() -> EmptyState(
                title = "暂无待处理举报",
                description = "用户提交的举报会出现在这里，审核后自动通知举报人。",
            )
            else -> LazyColumn(
                modifier = Modifier.fillMaxSize(),
                contentPadding = androidx.compose.foundation.layout.PaddingValues(
                    horizontal = Spacing.lg,
                    vertical = Spacing.sm,
                ),
            ) {
                item {
                    Text(
                        text = "待处理 " + (total ?: items.size) + " 条",
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textSecondary,
                        modifier = Modifier.padding(vertical = Spacing.sm, horizontal = Spacing.xs),
                    )
                }
                item {
                    ListGroup {
                        items.forEach { report ->
                            AdminReportRow(
                                report = report,
                                onResolve = { openJudge(report, "resolve") },
                                onDismiss = { openJudge(report, "dismiss") },
                            )
                        }
                    }
                }
                item { Spacer(Modifier.height(Spacing.lg)) }
            }
        }
    }

    judgeTarget?.let { target ->
        AlertDialog(
            onDismissRequest = { if (!submitting) judgeTarget = null },
            title = { Text(if (judgeAction == "resolve") "确认举报有效" else "驳回举报") },
            text = {
                Column {
                    Text(
                        text = ReportCatalog.targetTypeLabel(target.targetType) + " · " +
                            ReportCatalog.reasonLabel(target.reason) + " · 举报人 " + target.reporter.nickname,
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.textSecondary,
                    )
                    Spacer(Modifier.height(Spacing.sm))
                    SystemInput(
                        value = note,
                        onValueChange = { note = it },
                        label = "处理备注（可选，0-500 字）",
                        placeholder = if (judgeAction == "resolve") "说明处理结果…" else "说明驳回原因…",
                        singleLine = false,
                        enabled = !submitting,
                    )
                    if (judgeAction == "resolve") {
                        Spacer(Modifier.height(Spacing.sm))
                        Text(
                            text = "确认后会执行举报处理并通知举报人；本次不附带封禁。",
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.textSecondary,
                        )
                    }
                }
            },
            confirmButton = {
                SystemButton(
                    text = if (submitting) "提交中…" else if (judgeAction == "resolve") "确认" else "驳回",
                    onClick = { submitJudge() },
                    variant = if (judgeAction == "resolve") ButtonVariant.PRIMARY else ButtonVariant.DANGER,
                    size = ButtonSize.SM,
                    enabled = !submitting,
                    loading = submitting,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { judgeTarget = null },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !submitting,
                )
            },
        )
    }
}

@Composable
private fun AdminReportRow(
    report: AdminReportItem,
    onResolve: () -> Unit,
    onDismiss: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = Spacing.md, vertical = Spacing.sm2),
    ) {
        Row(verticalAlignment = androidx.compose.ui.Alignment.CenterVertically) {
            Text(
                text = ReportCatalog.targetTypeLabel(report.targetType) + " · " + ReportCatalog.reasonLabel(report.reason),
                style = MaterialTheme.typography.bodyLarge,
                color = colors.textPrimary,
                modifier = Modifier.weight(1f),
            )
            Text(
                text = TimeFormat.relative(report.createdAt),
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
            )
        }
        Spacer(Modifier.height(Spacing.xs))
        Text(
            text = "举报人：" + report.reporter.nickname.ifBlank { "用户" } +
                if (!report.description.isNullOrBlank()) " · " + report.description else "",
            style = MaterialTheme.typography.labelSmall,
            color = colors.textSecondary,
        )
        Spacer(Modifier.height(Spacing.sm))
        Row(horizontalArrangement = Arrangement.spacedBy(Spacing.sm)) {
            SystemButton(
                text = "通过",
                onClick = onResolve,
                variant = ButtonVariant.PRIMARY,
                size = ButtonSize.SM,
            )
            SystemButton(
                text = "驳回",
                onClick = onDismiss,
                variant = ButtonVariant.SECONDARY,
                size = ButtonSize.SM,
            )
        }
    }
}
