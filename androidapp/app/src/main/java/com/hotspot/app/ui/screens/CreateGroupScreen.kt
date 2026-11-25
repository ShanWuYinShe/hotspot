package com.hotspot.app.ui.screens

import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Place
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.media.AvatarUploader
import com.hotspot.app.data.model.DefaultLocation
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SegmentedTabs
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemInput
import com.hotspot.app.ui.components.SystemTextArea
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import kotlinx.coroutines.launch
import java.util.UUID

private val GROUP_CATEGORIES = listOf("摄影", "音乐", "户外", "阅读", "美食", "健身", "游戏", "其他")

/**
 * 创建群组（groups/create，owner = 当前用户）：
 * 封面（可选）/名称/简介/分类/可见性 + ClientConfig 默认位置；成功回调 group_id 进入群详情。
 * 设计稿无独立建群画板，按 design-system 原语组合（对齐 Web CreateGroupSheet 语义）。
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun CreateGroupScreen(
    onBack: () -> Unit,
    onCreated: (String) -> Unit,
) {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val repository = AppContainer.groupRepository
    var name by remember { mutableStateOf("") }
    var description by remember { mutableStateOf("") }
    var category by remember { mutableStateOf(GROUP_CATEGORIES.first()) }
    var visibilityIndex by remember { mutableIntStateOf(0) }
    var requiresApproval by remember { mutableStateOf(false) }
    var location by remember { mutableStateOf<DefaultLocation?>(null) }
    var coverUri by remember { mutableStateOf<Uri?>(null) }
    var avatarFileId by remember { mutableStateOf<String?>(null) }
    var uploadingCover by remember { mutableStateOf(false) }
    var creating by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val idempotencyKey = remember { UUID.randomUUID().toString() }
    val coverPicker = rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()) { uri ->
        if (uri != null && !uploadingCover) {
            uploadingCover = true
            error = null
            scope.launch {
                AvatarUploader.upload(context, uri, category = "group_avatar")
                    .onSuccess {
                        coverUri = uri
                        avatarFileId = it.fileId
                    }
                    .onFailure { error = it.message ?: "群头像读取或上传失败，请重新选择图片" }
                uploadingCover = false
            }
        }
    }

    LaunchedEffect(Unit) { location = AppContainer.defaultLocation() }

    val canCreate = name.trim().length >= 2 && location != null && !creating && !uploadingCover

    fun submit() {
        if (!canCreate) return
        creating = true
        error = null
        scope.launch {
            val loc = location ?: return@launch
            when (
                val result = repository.createGroup(
                    name = name.trim(),
                    description = description.trim().ifBlank { null },
                    tags = listOf(category),
                    joinMethod = if (visibilityIndex == 0) "direct" else "invite_only",
                    // 仅邀请群组不需要新人审核；公开群组才允许开启。
                    requiresApproval = requiresApproval && visibilityIndex == 0,
                    location = loc,
                    avatarFileId = avatarFileId,
                    idempotencyKey = idempotencyKey,
                )
            ) {
                is ApiResult.Ok -> onCreated(result.data.group.groupId)
                is ApiResult.Err -> error = result.error.userMessage
            }
            creating = false
        }
    }

    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "创建群组", onBack = onBack)
        Column(
            modifier = Modifier
                .weight(1f)
                .verticalScroll(rememberScrollState())
                .padding(horizontal = Spacing.lg),
        ) {
            Spacer(Modifier.height(Spacing.md))
            Row(verticalAlignment = Alignment.CenterVertically) {
                Box(contentAlignment = Alignment.Center) {
                    Avatar(
                        name = name.ifBlank { "群" },
                        imageUrl = coverUri?.toString(),
                        size = 64.dp,
                    )
                    if (uploadingCover) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(24.dp),
                            strokeWidth = 2.dp,
                            color = colors.accent,
                        )
                    }
                }
                Spacer(Modifier.width(Spacing.md))
                Column {
                    SystemButton(
                        text = if (uploadingCover) "上传中…" else "选择群头像",
                        onClick = {
                            coverPicker.launch(
                                PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly),
                            )
                        },
                        variant = ButtonVariant.SECONDARY,
                        size = ButtonSize.SM,
                        enabled = !uploadingCover,
                    )
                    Spacer(Modifier.height(Spacing.xs))
                    Text(
                        text = "可选 · JPG/PNG/WebP ≤5MB",
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textSecondary,
                    )
                }
                if (avatarFileId != null) {
                    Spacer(Modifier.width(Spacing.sm))
                    Text(
                        text = "移除封面",
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.accent,
                        modifier = Modifier
                            .clip(AppShapes.pill)
                            .clickable {
                                if (!uploadingCover) {
                                    coverUri = null
                                    avatarFileId = null
                                }
                            }
                            .padding(horizontal = Spacing.sm, vertical = Spacing.xs),
                    )
                }
            }
            Spacer(Modifier.height(Spacing.md))
            SystemInput(
                value = name,
                onValueChange = { name = it },
                label = "群名称",
                placeholder = "给群组起个名字（至少 2 个字）",
            )
            Spacer(Modifier.height(Spacing.md))
            SystemTextArea(
                value = description,
                onValueChange = { description = it },
                label = "群简介",
                placeholder = "介绍一下这个群…",
                minHeight = Dimens.inputHeight * 2,
            )
            Spacer(Modifier.height(Spacing.lg))
            Text(
                text = "分类",
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.SemiBold,
                color = colors.textPrimary,
            )
            Spacer(Modifier.height(Spacing.sm))
            FlowRow(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(Spacing.sm),
                verticalArrangement = Arrangement.spacedBy(Spacing.sm),
            ) {
                GROUP_CATEGORIES.forEach { tag ->
                    val selected = category == tag
                    Box(
                        modifier = Modifier
                            .clip(AppShapes.pill)
                            .background(if (selected) colors.accentSoftBg else colors.island)
                            .border(
                                1.dp,
                                if (selected) colors.accentBorder else colors.borderSubtle,
                                AppShapes.pill,
                            )
                            .clickable { category = tag }
                            .padding(horizontal = Spacing.md, vertical = Spacing.sm2),
                    ) {
                        Text(
                            text = tag,
                            style = MaterialTheme.typography.labelSmall,
                            fontWeight = if (selected) FontWeight.Medium else FontWeight.Normal,
                            color = if (selected) colors.textAccentOnSoft else colors.textSecondary,
                        )
                    }
                }
            }
            Spacer(Modifier.height(Spacing.lg))
            Text(
                text = "谁可以加入",
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.SemiBold,
                color = colors.textPrimary,
            )
            Spacer(Modifier.height(Spacing.sm))
            SegmentedTabs(
                options = listOf("公开群组", "仅邀请"),
                selectedIndex = visibilityIndex,
                onSelected = { visibilityIndex = it },
                modifier = Modifier.fillMaxWidth(),
            )
            if (visibilityIndex == 0) {
                Spacer(Modifier.height(Spacing.md))
                Text(
                    text = "新人加入",
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                    color = colors.textPrimary,
                )
                Spacer(Modifier.height(Spacing.sm))
                SegmentedTabs(
                    options = listOf("直接加入", "申请需审核"),
                    selectedIndex = if (requiresApproval) 1 else 0,
                    onSelected = { requiresApproval = it == 1 },
                    modifier = Modifier.fillMaxWidth(),
                )
            }
            Spacer(Modifier.height(Spacing.lg))
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(AppShapes.card)
                    .background(colors.island)
                    .padding(Spacing.md),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    imageVector = Icons.Filled.Place,
                    contentDescription = null,
                    modifier = Modifier.size(Dimens.iconSm),
                    tint = colors.accent,
                )
                Spacer(Modifier.width(Spacing.sm))
                Column {
                    Text(
                        text = location?.locationName?.takeIf { it.isNotBlank() } ?: "正在获取当前位置…",
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Medium,
                        color = colors.textPrimary,
                    )
                    Text(
                        text = "群组位置使用当前浏览位置；定位不可用时无法创建",
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textSecondary,
                    )
                }
            }
            if (error != null) {
                Spacer(Modifier.height(Spacing.sm))
                Text(
                    text = error ?: "",
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.danger,
                )
            }
            Spacer(Modifier.height(Spacing.xxl))
        }
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = Spacing.lg, vertical = Spacing.md),
        ) {
            SystemButton(
                text = "创建群组",
                onClick = { submit() },
                variant = ButtonVariant.PRIMARY,
                size = ButtonSize.LG,
                fillMaxWidth = true,
                enabled = canCreate,
                loading = creating,
            )
        }
    }
}
