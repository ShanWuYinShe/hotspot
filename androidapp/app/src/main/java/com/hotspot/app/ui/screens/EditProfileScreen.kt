package com.hotspot.app.ui.screens

import android.widget.Toast
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.AccountData
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemCard
import com.hotspot.app.ui.components.SystemInput
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import kotlinx.coroutines.launch

private const val BIO_MAX = 150
private const val TAG_MAX = 5
private val TAG_CATALOG = listOf(
    "摄影", "旅行", "音乐", "美食", "运动", "阅读", "艺术",
    "科技", "游戏", "户外", "时尚", "电影", "官方", "巡检",
)
private val GENDER_OPTIONS = listOf("男" to "male", "女" to "female", "其他" to "other")

/**
 * 编辑资料（设计稿 87:3128）：头像卡（点头像可更换照片）+ 基本信息（昵称 / 个人简介 150 字）
 * + 更多信息（性别单选 / 兴趣标签最多 5 个）+ 吸底「保存资料」。
 * 保存走 users/update_account（patch 语义）；头像上传链路未接入（media/complete_upload TODO），
 * 点击头像只提示占位，不伪造上传入口。
 */
@Composable
fun EditProfileScreen(onBack: () -> Unit) {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var account by remember { mutableStateOf<AccountData?>(null) }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var nickname by remember { mutableStateOf("") }
    var bio by remember { mutableStateOf("") }
    var gender by remember { mutableStateOf<String?>(null) }
    val selectedTags = remember { mutableStateListOf<String>() }

    LaunchedEffect(Unit) {
        when (val result = AppContainer.profileRepository.account()) {
            is ApiResult.Ok -> {
                val data = result.data
                account = data
                nickname = data.nickname
                bio = data.bio.orEmpty()
                gender = data.gender
                selectedTags.clear()
                selectedTags.addAll(data.tags.take(TAG_MAX))
            }
            is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
        }
        loading = false
    }

    fun save() {
        if (saving) return
        val trimmed = nickname.trim()
        if (trimmed.isEmpty()) {
            Toast.makeText(context, "昵称不能为空", Toast.LENGTH_SHORT).show()
            return
        }
        saving = true
        scope.launch {
            when (
                val result = AppContainer.profileRepository.updateAccount(
                    nickname = trimmed,
                    bio = bio.trim(),
                    tags = selectedTags.toList(),
                    gender = gender,
                )
            ) {
                is ApiResult.Ok -> {
                    Toast.makeText(context, "资料已保存", Toast.LENGTH_SHORT).show()
                    onBack()
                }
                is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
            }
            saving = false
        }
    }

    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "编辑资料", onBack = onBack)
        if (loading) {
            LoadingState(message = "正在加载资料…")
            return@Column
        }
        Column(
            modifier = Modifier
                .weight(1f)
                .verticalScroll(rememberScrollState())
                .padding(horizontal = Spacing.lg),
        ) {
            Spacer(Modifier.height(Spacing.xs))
            // 头像卡
            SystemCard(contentPadding = PaddingValues(Spacing.md)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Box {
                        Avatar(
                            name = nickname.ifBlank { account?.nickname.orEmpty() },
                            imageUrl = account?.avatarUrl,
                            size = 56.dp,
                        )
                        Box(
                            modifier = Modifier
                                .align(Alignment.BottomEnd)
                                .size(22.dp)
                                .clip(CircleShape)
                                .background(colors.accent)
                                .clickable {
                                    Toast.makeText(context, "头像上传待接入（首版占位）", Toast.LENGTH_SHORT).show()
                                },
                            contentAlignment = Alignment.Center,
                        ) {
                            Icon(
                                imageVector = Icons.Filled.Edit,
                                contentDescription = "更换头像",
                                modifier = Modifier.size(12.dp),
                                tint = colors.textOnAccent,
                            )
                        }
                    }
                    Spacer(Modifier.width(Spacing.md))
                    Column(Modifier.weight(1f)) {
                        Text(
                            text = nickname.ifBlank { "未设置昵称" },
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.SemiBold,
                            color = colors.textPrimary,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                        Text(
                            text = account?.email?.let { "@" + it.substringBefore("@") } ?: "点头像可更换照片",
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.textSecondary,
                        )
                        Text(
                            text = "点头像可更换照片",
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.textSecondary,
                        )
                    }
                }
            }
            Spacer(Modifier.height(Spacing.lg))
            FormSectionTitle(title = "基本信息", subtitle = "公开显示在你的主页上，附近的人会看到这些信息。")
            Spacer(Modifier.height(Spacing.sm))
            SystemCard {
                FieldLabel("昵称")
                Spacer(Modifier.height(Spacing.xs))
                SystemInput(
                    value = nickname,
                    onValueChange = { if (it.length <= 20) nickname = it },
                    placeholder = "输入昵称",
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(Spacing.md))
                FieldLabel("个人简介")
                Spacer(Modifier.height(Spacing.xs))
                SystemInput(
                    value = bio,
                    onValueChange = { if (it.length <= BIO_MAX) bio = it },
                    placeholder = "介绍一下自己",
                    singleLine = false,
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(96.dp),
                )
                Spacer(Modifier.height(Spacing.xs))
                Text(
                    text = bio.length.toString() + "/" + BIO_MAX,
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                    modifier = Modifier.fillMaxWidth(),
                    textAlign = androidx.compose.ui.text.style.TextAlign.End,
                )
            }
            Spacer(Modifier.height(Spacing.lg))
            FormSectionTitle(title = "更多信息", subtitle = "帮助附近的人更快了解你，随时可以修改或清空。")
            Spacer(Modifier.height(Spacing.sm))
            SystemCard {
                FieldLabel("性别")
                Spacer(Modifier.height(Spacing.sm))
                Row(horizontalArrangement = Arrangement.spacedBy(Spacing.sm)) {
                    GENDER_OPTIONS.forEach { (label, value) ->
                        SelectablePill(
                            text = label,
                            selected = gender == value,
                            onClick = { gender = if (gender == value) null else value },
                        )
                    }
                }
                Spacer(Modifier.height(Spacing.lg))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    FieldLabel("兴趣标签")
                    Spacer(Modifier.width(Spacing.xs))
                    Text(
                        text = "(" + selectedTags.size + "/" + TAG_MAX + ")",
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textSecondary,
                    )
                }
                Spacer(Modifier.height(Spacing.sm))
                TagFlow(
                    selected = selectedTags,
                    onToggle = { tag ->
                        if (selectedTags.contains(tag)) {
                            selectedTags.remove(tag)
                        } else if (selectedTags.size >= TAG_MAX) {
                            Toast.makeText(context, "最多选择 " + TAG_MAX + " 个标签", Toast.LENGTH_SHORT).show()
                        } else {
                            selectedTags.add(tag)
                        }
                    },
                )
            }
            Spacer(Modifier.height(Spacing.xl))
        }
        // 吸底保存（设计稿 87:3128 底部主按钮）
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = Spacing.lg, vertical = Spacing.md),
        ) {
            SystemButton(
                text = "保存资料",
                onClick = { save() },
                variant = ButtonVariant.PRIMARY,
                size = ButtonSize.LG,
                fillMaxWidth = true,
                loading = saving,
                enabled = !saving,
            )
        }
    }
}

@Composable
private fun FormSectionTitle(title: String, subtitle: String) {
    val colors = HotSpotTheme.colors
    Column {
        Text(
            text = title,
            style = MaterialTheme.typography.titleSmall,
            fontWeight = FontWeight.SemiBold,
            color = colors.textPrimary,
        )
        Spacer(Modifier.height(2.dp))
        Text(
            text = subtitle,
            style = MaterialTheme.typography.labelSmall,
            color = colors.textSecondary,
        )
    }
}

@Composable
private fun FieldLabel(text: String) {
    val colors = HotSpotTheme.colors
    Text(
        text = text,
        style = MaterialTheme.typography.labelMedium,
        color = colors.textSecondary,
    )
}

@Composable
private fun SelectablePill(text: String, selected: Boolean, onClick: () -> Unit) {
    val colors = HotSpotTheme.colors
    Box(
        modifier = Modifier
            .clip(AppShapes.pill)
            .background(if (selected) colors.accentSoftBg else colors.island)
            .border(1.dp, if (selected) colors.accentBorder else colors.borderSubtle, AppShapes.pill)
            .clickable(onClick = onClick)
            .padding(horizontal = Spacing.lg, vertical = Spacing.sm2),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.labelMedium,
            fontWeight = if (selected) FontWeight.Medium else FontWeight.Normal,
            color = if (selected) colors.textAccentOnSoft else colors.textSecondary,
        )
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun TagFlow(selected: List<String>, onToggle: (String) -> Unit) {
    val colors = HotSpotTheme.colors
    // 目录外的既有标签也要可见可取消（否则计数与实际选中不一致）
    val tags = TAG_CATALOG + selected.filterNot { TAG_CATALOG.contains(it) }
    FlowRow(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(Spacing.sm),
        verticalArrangement = Arrangement.spacedBy(Spacing.sm),
    ) {
        tags.forEach { tag ->
            val isSelected = selected.contains(tag)
            Box(
                modifier = Modifier
                    .clip(AppShapes.pill)
                    .background(if (isSelected) colors.accentSoftBg else colors.island)
                    .border(1.dp, if (isSelected) colors.accentBorder else colors.borderSubtle, AppShapes.pill)
                    .clickable { onToggle(tag) }
                    .padding(horizontal = Spacing.md, vertical = Spacing.sm2),
            ) {
                Text(
                    text = "#" + tag,
                    style = MaterialTheme.typography.labelSmall,
                    fontWeight = if (isSelected) FontWeight.Medium else FontWeight.Normal,
                    color = if (isSelected) colors.textAccentOnSoft else colors.textSecondary,
                )
            }
        }
    }
}
