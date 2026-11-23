package com.hotspot.app.ui.screens

import android.widget.Toast
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Checkbox
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextOverflow
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.AdminOperatorItem
import com.hotspot.app.data.model.AdminPermissionItem
import com.hotspot.app.data.model.AdminRoleItem
import com.hotspot.app.data.model.AdminUserItem
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.ListEnd
import com.hotspot.app.ui.components.ListGroup
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemInput
import com.hotspot.app.ui.components.TabPills
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.TimeFormat
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * 管理后台 · 权限与角色（仅平台管理员 RBAC manage）：平台管理员列表、角色定义与权限字典。
 * 当前为只读视图；授权/撤销管理员（rbac/admins/award|revoke）待后续接入。
 */
@Composable
fun AdminRbacScreen(onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val colors = HotSpotTheme.colors
    var tabIndex by remember { mutableIntStateOf(0) }
    var admins by remember { mutableStateOf<List<AdminOperatorItem>>(emptyList()) }
    var adminCursor by remember { mutableStateOf<String?>(null) }
    var roles by remember { mutableStateOf<List<AdminRoleItem>>(emptyList()) }
    var permissions by remember { mutableStateOf<List<AdminPermissionItem>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var loadingMore by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var reloadKey by remember { mutableIntStateOf(0) }
    var awardOpen by remember { mutableStateOf(false) }
    var awardQuery by remember { mutableStateOf("") }
    var awardSearching by remember { mutableStateOf(false) }
    var awardResults by remember { mutableStateOf<List<AdminUserItem>>(emptyList()) }
    var awardSelected by remember { mutableStateOf<AdminUserItem?>(null) }
    var awardAll by remember { mutableStateOf(false) }
    var awardRoleIds by remember { mutableStateOf(setOf<String>()) }
    var awardSaving by remember { mutableStateOf(false) }
    var revokeTarget by remember { mutableStateOf<AdminOperatorItem?>(null) }
    var revoking by remember { mutableStateOf(false) }

    LaunchedEffect(reloadKey) {
        loading = true
        error = null
        val rolesResult = AppContainer.adminRepository.fetchRoles()
        val permissionsResult = AppContainer.adminRepository.fetchPermissions()
        val adminsResult = AppContainer.adminRepository.fetchAdmins(null)
        val failure = listOfNotNull(
            (rolesResult as? ApiResult.Err)?.error?.userMessage,
            (permissionsResult as? ApiResult.Err)?.error?.userMessage,
            (adminsResult as? ApiResult.Err)?.error?.userMessage,
        ).firstOrNull()
        if (failure != null) {
            error = failure
        } else {
            roles = (rolesResult as ApiResult.Ok).data.items
            permissions = (permissionsResult as ApiResult.Ok).data.items
            val page = (adminsResult as ApiResult.Ok).data
            admins = page.items
            adminCursor = page.nextCursor
        }
        loading = false
    }

    LaunchedEffect(awardQuery, awardOpen) {
        if (!awardOpen) return@LaunchedEffect
        if (awardQuery.isBlank()) {
            awardResults = emptyList()
            return@LaunchedEffect
        }
        delay(350)
        awardSearching = true
        when (val result = AppContainer.adminRepository.searchUsers(awardQuery, null, null)) {
            is ApiResult.Ok -> awardResults = result.data.items.take(6)
            is ApiResult.Err -> awardResults = emptyList()
        }
        awardSearching = false
    }

    fun loadMoreAdmins() {
        val next = adminCursor ?: return
        if (loadingMore) return
        loadingMore = true
        scope.launch {
            when (val result = AppContainer.adminRepository.fetchAdmins(next)) {
                is ApiResult.Ok -> {
                    admins = (admins + result.data.items).distinctBy { it.user.userId }
                    adminCursor = result.data.nextCursor
                }
                is ApiResult.Err -> Unit
            }
            loadingMore = false
        }
    }

    Column(Modifier.fillMaxSize()) {
        PageHeader(
            title = "权限与角色",
            subtitle = "平台管理员",
            onBack = onBack,
            actions = {
                if (tabIndex == 0) {
                    SystemButton(
                        text = "授权",
                        onClick = {
                            awardQuery = ""
                            awardResults = emptyList()
                            awardSelected = null
                            awardAll = false
                            awardRoleIds = emptySet()
                            awardOpen = true
                        },
                        variant = ButtonVariant.PRIMARY,
                        size = ButtonSize.SM,
                    )
                    Spacer(Modifier.width(Spacing.sm))
                }
                SystemButton(
                    text = "刷新",
                    onClick = { reloadKey += 1 },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !loading,
                )
            },
        )
        Column(Modifier.padding(horizontal = Spacing.lg)) {
            TabPills(
                options = listOf("管理员", "角色", "权限"),
                selectedIndex = tabIndex,
                onSelected = { tabIndex = it },
            )
            Spacer(Modifier.height(Spacing.sm))
        }
        when {
            loading && admins.isEmpty() && roles.isEmpty() -> LoadingState(message = "正在加载权限配置…")
            error != null && admins.isEmpty() -> ErrorState(message = error ?: "加载失败", onRetry = { reloadKey += 1 })
            tabIndex == 0 -> {
                if (admins.isEmpty()) {
                    EmptyState(title = "暂无平台管理员", description = "授权后管理员会出现在这里。")
                } else {
                    LazyColumn(
                        modifier = Modifier.fillMaxSize(),
                        contentPadding = androidx.compose.foundation.layout.PaddingValues(
                            horizontal = Spacing.lg,
                            vertical = Spacing.sm,
                        ),
                    ) {
                        item {
                            ListGroup {
                                admins.forEachIndexed { index, item ->
                                    AdminRow(
                                        item = item,
                                        onRevoke = if (item.permissions.contains("*")) null else {
                                            { revokeTarget = item }
                                        },
                                    )
                                    if (index != admins.lastIndex) {
                                        HorizontalDivider(color = colors.borderSubtle)
                                    }
                                }
                            }
                        }
                        if (adminCursor != null) {
                            item {
                                Box(Modifier.padding(vertical = Spacing.lg)) {
                                    SystemButton(
                                        text = if (loadingMore) "加载中…" else "加载更多",
                                        onClick = { loadMoreAdmins() },
                                        variant = ButtonVariant.SECONDARY,
                                        size = ButtonSize.SM,
                                        enabled = !loadingMore,
                                        loading = loadingMore,
                                    )
                                }
                            }
                        } else {
                            item { Box(Modifier.padding(vertical = Spacing.lg)) { ListEnd() } }
                        }
                    }
                }
            }
            tabIndex == 1 -> {
                if (roles.isEmpty()) {
                    EmptyState(title = "暂无角色", description = "服务端尚未配置 RBAC 角色。")
                } else {
                    LazyColumn(
                        modifier = Modifier.fillMaxSize(),
                        contentPadding = androidx.compose.foundation.layout.PaddingValues(
                            horizontal = Spacing.lg,
                            vertical = Spacing.sm,
                        ),
                    ) {
                        item {
                            ListGroup {
                                roles.forEachIndexed { index, role ->
                                    RoleRow(role)
                                    if (index != roles.lastIndex) {
                                        HorizontalDivider(color = colors.borderSubtle)
                                    }
                                }
                            }
                        }
                        item { Box(Modifier.padding(vertical = Spacing.lg)) { ListEnd() } }
                    }
                }
            }
            else -> {
                if (permissions.isEmpty()) {
                    EmptyState(title = "暂无权限项", description = "服务端尚未返回权限字典。")
                } else {
                    LazyColumn(
                        modifier = Modifier.fillMaxSize(),
                        contentPadding = androidx.compose.foundation.layout.PaddingValues(
                            horizontal = Spacing.lg,
                            vertical = Spacing.sm,
                        ),
                    ) {
                        item {
                            ListGroup {
                                permissions.forEachIndexed { index, permission ->
                                    PermissionRow(permission)
                                    if (index != permissions.lastIndex) {
                                        HorizontalDivider(color = colors.borderSubtle)
                                    }
                                }
                            }
                        }
                        item { Box(Modifier.padding(vertical = Spacing.lg)) { ListEnd() } }
                    }
                }
            }
        }
    }
    if (awardOpen) {
        AlertDialog(
            onDismissRequest = { if (!awardSaving) awardOpen = false },
            title = { Text("授权平台管理员") },
            text = {
                Column {
                    SystemInput(
                        value = awardQuery,
                        onValueChange = { awardQuery = it },
                        label = "搜索用户昵称",
                        placeholder = "输入昵称，例如 Mika…",
                        enabled = !awardSaving,
                    )
                    if (awardSelected != null) {
                        Spacer(Modifier.height(Spacing.xs))
                        Text(
                            text = "已选：" + (awardSelected?.user?.nickname ?: "") +
                                "（" + (awardSelected?.user?.userId ?: "") + "）",
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.accent,
                        )
                    }
                    if (awardSearching) {
                        Spacer(Modifier.height(Spacing.xs))
                        Text("搜索中…", style = MaterialTheme.typography.labelSmall, color = colors.textSecondary)
                    } else if (awardResults.isNotEmpty()) {
                        Spacer(Modifier.height(Spacing.xs))
                        awardResults.forEach { candidate ->
                            Row(
                                Modifier
                                    .fillMaxWidth()
                                    .clickable(enabled = !awardSaving) {
                                        awardSelected = candidate
                                        awardResults = emptyList()
                                        awardQuery = candidate.user.nickname
                                    }
                                    .padding(vertical = Spacing.xs),
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Text(
                                    text = candidate.user.nickname.ifBlank { candidate.user.userId },
                                    style = MaterialTheme.typography.bodySmall,
                                    color = colors.textPrimary,
                                    modifier = Modifier.weight(1f),
                                )
                                Text(
                                    text = candidate.user.userId,
                                    style = MaterialTheme.typography.labelSmall,
                                    color = colors.textSecondary,
                                )
                            }
                        }
                    }
                    Spacer(Modifier.height(Spacing.sm))
                    Text("角色", style = MaterialTheme.typography.titleSmall, color = colors.textPrimary)
                    roles.forEach { role ->
                        Row(
                            Modifier
                                .fillMaxWidth()
                                .clickable(enabled = !awardAll && !awardSaving) {
                                    awardRoleIds = if (role.roleId in awardRoleIds) {
                                        awardRoleIds - role.roleId
                                    } else {
                                        awardRoleIds + role.roleId
                                    }
                                }
                                .padding(vertical = Spacing.xs),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Checkbox(
                                checked = role.roleId in awardRoleIds,
                                onCheckedChange = null,
                                enabled = !awardAll && !awardSaving,
                            )
                            Text(
                                text = role.name + "（" + role.permissions.size + " 项权限）",
                                style = MaterialTheme.typography.bodySmall,
                                color = if (awardAll) colors.textSecondary else colors.textPrimary,
                            )
                        }
                    }
                    Row(
                        Modifier.fillMaxWidth(),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(
                            text = "授予全部权限（仅 superadmin 可执行）",
                            style = MaterialTheme.typography.bodySmall,
                            color = colors.textSecondary,
                            modifier = Modifier.weight(1f),
                        )
                        Switch(checked = awardAll, onCheckedChange = { awardAll = it }, enabled = !awardSaving)
                    }
                }
            },
            confirmButton = {
                SystemButton(
                    text = if (awardSaving) "授权中…" else "授权",
                    onClick = {
                        if (awardSaving) return@SystemButton
                        val target = awardSelected
                        if (target == null) {
                            Toast.makeText(context, "请先选择用户", Toast.LENGTH_SHORT).show()
                            return@SystemButton
                        }
                        if (!awardAll && awardRoleIds.isEmpty()) {
                            Toast.makeText(context, "请选择角色或授予全部权限", Toast.LENGTH_SHORT).show()
                            return@SystemButton
                        }
                        awardSaving = true
                        scope.launch {
                            when (val result = AppContainer.adminRepository.awardAdmin(
                                userId = target.user.userId,
                                awardAllPermissions = awardAll,
                                roleIds = awardRoleIds.toList(),
                            )) {
                                is ApiResult.Ok -> {
                                    awardOpen = false
                                    reloadKey += 1
                                    Toast.makeText(context, "管理员已授权", Toast.LENGTH_SHORT).show()
                                }
                                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                            }
                            awardSaving = false
                        }
                    },
                    variant = ButtonVariant.PRIMARY,
                    size = ButtonSize.SM,
                    enabled = !awardSaving,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { awardOpen = false },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !awardSaving,
                )
            },
        )
    }

    revokeTarget?.let { target ->
        AlertDialog(
            onDismissRequest = { if (!revoking) revokeTarget = null },
            title = { Text("撤销管理员") },
            text = {
                Text(
                    text = "撤销「" + (target.user.nickname.ifBlank { target.user.userId }) +
                        "」的平台管理员权限？撤销后该用户立即失去管理后台访问。",
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.textSecondary,
                )
            },
            confirmButton = {
                SystemButton(
                    text = if (revoking) "撤销中…" else "撤销",
                    onClick = {
                        if (revoking) return@SystemButton
                        revoking = true
                        scope.launch {
                            when (val result = AppContainer.adminRepository.revokeAdmin(target.user.userId)) {
                                is ApiResult.Ok -> {
                                    revokeTarget = null
                                    reloadKey += 1
                                    Toast.makeText(context, "已撤销管理员", Toast.LENGTH_SHORT).show()
                                }
                                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                            }
                            revoking = false
                        }
                    },
                    variant = ButtonVariant.DANGER,
                    size = ButtonSize.SM,
                    enabled = !revoking,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { revokeTarget = null },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                    enabled = !revoking,
                )
            },
        )
    }
}

@Composable
private fun AdminRow(
    item: AdminOperatorItem,
    onRevoke: (() -> Unit)? = null,
) {
    val colors = HotSpotTheme.colors
    val hasAll = item.permissions.contains("*")
    Row(
        modifier = Modifier.fillMaxWidth().padding(horizontal = Spacing.md, vertical = Spacing.sm2),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Avatar(name = item.user.nickname, imageUrl = item.user.avatarUrl, size = Dimens.avatarSm)
        Spacer(Modifier.width(Spacing.sm2))
        Column(Modifier.weight(1f)) {
            Text(
                text = item.user.nickname.ifBlank { item.user.userId },
                style = MaterialTheme.typography.bodyLarge,
                color = colors.textPrimary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                text = if (hasAll) "全部权限" + roleSuffix(item) else (item.permissions.size.toString() + " 项权限" + roleSuffix(item)),
                style = MaterialTheme.typography.bodySmall,
                color = colors.textSecondary,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                text = "最近授权 " + TimeFormat.relative(item.grantedAt),
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
            )
        }
        if (onRevoke != null) {
            SystemButton(
                text = "撤销",
                onClick = onRevoke,
                variant = ButtonVariant.SECONDARY,
                size = ButtonSize.SM,
            )
        }
    }
}

private fun roleSuffix(item: AdminOperatorItem): String =
    if (item.roles.isEmpty()) "" else " · 角色：" + item.roles.joinToString("、") { it.name }

@Composable
private fun RoleRow(role: AdminRoleItem) {
    val colors = HotSpotTheme.colors
    Column(Modifier.fillMaxWidth().padding(horizontal = Spacing.md, vertical = Spacing.sm2)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = role.name,
                style = MaterialTheme.typography.bodyLarge,
                color = colors.textPrimary,
                modifier = Modifier.weight(1f),
            )
            Text(
                text = role.permissions.size.toString() + " 项权限",
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
            )
        }
        role.description?.takeIf { it.isNotBlank() }?.let {
            Text(
                text = it,
                style = MaterialTheme.typography.bodySmall,
                color = colors.textSecondary,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
        }
        Spacer(Modifier.height(Spacing.xs))
        Text(
            text = if (role.permissions.isEmpty()) "无权限" else role.permissions.joinToString("、"),
            style = MaterialTheme.typography.labelSmall,
            color = colors.textSecondary,
            maxLines = 3,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

@Composable
private fun PermissionRow(permission: AdminPermissionItem) {
    val colors = HotSpotTheme.colors
    Column(Modifier.fillMaxWidth().padding(horizontal = Spacing.md, vertical = Spacing.sm2)) {
        Text(
            text = permission.key,
            style = MaterialTheme.typography.bodyLarge,
            color = colors.textPrimary,
        )
        Text(
            text = permission.description,
            style = MaterialTheme.typography.bodySmall,
            color = colors.textSecondary,
        )
    }
}
