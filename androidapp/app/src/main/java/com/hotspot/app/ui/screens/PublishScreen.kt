package com.hotspot.app.ui.screens

import android.content.Context
import android.net.Uri
import android.provider.OpenableColumns
import android.widget.Toast
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
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Place
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import com.hotspot.app.AppContainer
import com.hotspot.app.R
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.AppLimits
import com.hotspot.app.data.model.DefaultLocation
import com.hotspot.app.data.model.GeoPoint
import com.hotspot.app.data.model.PresignFileInput
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SegmentedTabs
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemCard
import com.hotspot.app.ui.components.SystemInput
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

private val MOOD_TAGS = listOf("松弛", "漫游", "咖啡", "展览", "自然", "夜色")

/**
 * 发布瞬间（Android 首版落地）：文案 + 图片（相册选择 → media/presign_upload → PUT → complete_upload）
 * + 氛围标签 + 可见范围 + 位置（ClientConfig 默认浏览位置，无设备定位/地图）→ moments/create（幂等）。
 * 结构与 Web CreateSheet 同语义（三端一致）；Android 设计稿未提供发布画板，UI 由 design-system 原语组合。
 */
@Composable
fun PublishScreen() {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    val scope = rememberCoroutineScope()

    var limits by remember { mutableStateOf(AppLimits()) }
    var location by remember { mutableStateOf<DefaultLocation?>(null) }
    var description by remember { mutableStateOf("") }
    var privacyIndex by remember { mutableIntStateOf(0) }
    var publishing by remember { mutableStateOf(false) }
    var status by remember { mutableStateOf("") }
    val moods = remember { mutableStateListOf<String>() }
    val images = remember { mutableStateListOf<Uri>() }

    LaunchedEffect(Unit) {
        limits = AppContainer.limits()
        location = AppContainer.defaultLocation()
    }

    val maxImages = limits.maxMomentImages.coerceIn(2, 9)
    val pickLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.PickMultipleVisualMedia(maxItems = maxImages),
    ) { uris ->
        uris.forEach { uri ->
            if (images.size < limits.maxMomentImages && !images.contains(uri)) images.add(uri)
        }
    }

    val canPublish = description.isNotBlank() && !publishing

    fun publish() {
        if (!canPublish) return
        publishing = true
        status = ""
        scope.launch {
            try {
                val fileIds = uploadImagesForPublish(context, images.toList()) { status = it }
                if (fileIds == null) {
                    status = ""
                    return@launch
                }
                status = "正在发布…"
                val effectiveLocation = location ?: DefaultLocation(GeoPoint(0.0, 0.0), "")
                when (
                    val result = AppContainer.momentsRepository.create(
                        description = description.trim(),
                        location = effectiveLocation,
                        tags = moods.toList(),
                        privacyLevel = if (privacyIndex == 0) "public" else "private",
                        imageFileIds = fileIds,
                    )
                ) {
                    is ApiResult.Ok -> {
                        Toast.makeText(context, "发布成功", Toast.LENGTH_SHORT).show()
                        description = ""
                        moods.clear()
                        images.clear()
                        privacyIndex = 0
                    }
                    is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                }
            } finally {
                status = ""
                publishing = false
            }
        }
    }

    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "发布", subtitle = "分享此刻")
        Column(
            modifier = Modifier
                .weight(1f)
                .verticalScroll(rememberScrollState())
                .padding(horizontal = Spacing.lg),
        ) {
            SystemInput(
                value = description,
                onValueChange = { value ->
                    if (value.length <= limits.momentDescriptionMaxLength) description = value
                },
                placeholder = "分享此刻的想法…",
                singleLine = false,
                keyboardOptions = androidx.compose.foundation.text.KeyboardOptions(keyboardType = KeyboardType.Text),
                modifier = Modifier
                    .fillMaxWidth()
                    .height(150.dp),
            )
            Spacer(Modifier.height(Spacing.xs))
            Text(
                text = description.length.toString() + "/" + limits.momentDescriptionMaxLength,
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
                modifier = Modifier.fillMaxWidth(),
                textAlign = androidx.compose.ui.text.style.TextAlign.End,
            )
            Spacer(Modifier.height(Spacing.lg))
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = "图片",
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                    color = colors.textPrimary,
                )
                Spacer(Modifier.width(Spacing.sm))
                Text(
                    text = images.size.toString() + "/" + limits.maxMomentImages,
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
            }
            Spacer(Modifier.height(Spacing.sm))
            LazyRow(horizontalArrangement = Arrangement.spacedBy(Spacing.sm)) {
                items(images.size) { index ->
                    Box {
                        AsyncImage(
                            model = images[index],
                            contentDescription = "待发布图片 " + (index + 1),
                            modifier = Modifier
                                .size(84.dp)
                                .clip(RoundedCornerShape(12.dp)),
                            contentScale = ContentScale.Crop,
                        )
                        Box(
                            modifier = Modifier
                                .align(Alignment.TopEnd)
                                .padding(4.dp)
                                .size(20.dp)
                                .clip(CircleShape)
                                .background(colors.overlayStrong)
                                .clickable { images.removeAt(index) },
                            contentAlignment = Alignment.Center,
                        ) {
                            Icon(
                                imageVector = Icons.Filled.Close,
                                contentDescription = "移除图片",
                                modifier = Modifier.size(12.dp),
                                tint = colors.contentInverse,
                            )
                        }
                    }
                }
                if (images.size < limits.maxMomentImages) {
                    item {
                        Box(
                            modifier = Modifier
                                .size(84.dp)
                                .clip(RoundedCornerShape(12.dp))
                                .background(colors.island)
                                .border(1.dp, colors.borderSubtle, RoundedCornerShape(12.dp))
                                .clickable {
                                    pickLauncher.launch(
                                        PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly),
                                    )
                                },
                            contentAlignment = Alignment.Center,
                        ) {
                            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                                Icon(
                                    painter = painterResource(R.drawable.ic_nav_publish),
                                    contentDescription = "添加图片",
                                    modifier = Modifier.size(Dimens.iconSm),
                                    tint = colors.textSecondary,
                                )
                                Spacer(Modifier.height(Spacing.xs))
                                Text(
                                    text = "添加",
                                    style = MaterialTheme.typography.labelSmall,
                                    color = colors.textSecondary,
                                )
                            }
                        }
                    }
                }
            }
            Spacer(Modifier.height(Spacing.lg))
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = "此刻氛围",
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                    color = colors.textPrimary,
                )
                Spacer(Modifier.width(Spacing.sm))
                Text(
                    text = moods.size.toString() + "/" + limits.maxMomentTags,
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
            }
            Spacer(Modifier.height(Spacing.sm))
            MoodChips(
                selected = moods,
                onToggle = { tag ->
                    if (moods.contains(tag)) {
                        moods.remove(tag)
                    } else if (moods.size >= limits.maxMomentTags) {
                        Toast.makeText(context, "最多选择 " + limits.maxMomentTags + " 个标签", Toast.LENGTH_SHORT).show()
                    } else {
                        moods.add(tag)
                    }
                },
            )
            Spacer(Modifier.height(Spacing.lg))
            Text(
                text = "谁可以看到",
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.SemiBold,
                color = colors.textPrimary,
            )
            Spacer(Modifier.height(Spacing.sm))
            SegmentedTabs(
                options = listOf("附近可见", "仅自己"),
                selectedIndex = privacyIndex,
                onSelected = { privacyIndex = it },
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(Spacing.lg))
            SystemCard(contentPadding = PaddingValues(Spacing.md)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(
                        imageVector = Icons.Filled.Place,
                        contentDescription = null,
                        modifier = Modifier.size(Dimens.iconSm),
                        tint = colors.accent,
                    )
                    Spacer(Modifier.width(Spacing.sm))
                    Column {
                        Text(
                            text = location?.locationName?.takeIf { it.isNotBlank() } ?: "未知位置",
                            style = MaterialTheme.typography.bodyMedium,
                            fontWeight = FontWeight.Medium,
                            color = colors.textPrimary,
                        )
                        Text(
                            text = "发布位置（使用默认浏览位置；地图与设备定位待接入）",
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.textSecondary,
                        )
                    }
                }
            }
            Spacer(Modifier.height(Spacing.xxl))
        }
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = Spacing.lg, vertical = Spacing.md),
        ) {
            Column {
                if (status.isNotBlank()) {
                    Text(
                        text = status,
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textSecondary,
                        modifier = Modifier.padding(bottom = Spacing.xs),
                    )
                }
                SystemButton(
                    text = "发布瞬间",
                    onClick = { publish() },
                    variant = ButtonVariant.PRIMARY,
                    size = ButtonSize.LG,
                    fillMaxWidth = true,
                    enabled = canPublish,
                    loading = publishing,
                )
            }
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun MoodChips(selected: List<String>, onToggle: (String) -> Unit) {
    val colors = HotSpotTheme.colors
    FlowRow(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(Spacing.sm),
        verticalArrangement = Arrangement.spacedBy(Spacing.sm),
    ) {
        MOOD_TAGS.forEach { tag ->
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



private suspend fun uploadImagesForPublish(
    context: Context,
    selected: List<Uri>,
    onStatus: (String) -> Unit,
): List<String>? {
    if (selected.isEmpty()) return emptyList()
    onStatus("正在准备上传…")
    val payloads = withContext(Dispatchers.IO) {
        selected.mapNotNull { uri -> readLocalImage(context, uri) }
    }
    if (payloads.size != selected.size) {
        Toast.makeText(context, "部分图片读取失败，请重试", Toast.LENGTH_SHORT).show()
        return null
    }
    val presign = AppContainer.mediaRepository.presign(
        payloads.map { image ->
            PresignFileInput(
                fileName = image.name,
                fileSize = image.bytes.size.toLong(),
                mimeType = image.mime,
            )
        },
    )
    val uploads = when (presign) {
        is ApiResult.Err -> {
            Toast.makeText(context, presign.error.userMessage, Toast.LENGTH_SHORT).show()
            return null
        }
        is ApiResult.Ok -> presign.data.uploads
    }
    var failed: String? = null
    uploads.forEachIndexed { index, upload ->
        val image = payloads.getOrNull(index) ?: return@forEachIndexed
        val put = AppContainer.mediaRepository.putObject(
            uploadUrl = upload.uploadUrl,
            headers = upload.headers,
            bytes = image.bytes,
            mimeType = image.mime,
        ) { fraction ->
            onStatus("正在上传第 " + (index + 1) + "/" + uploads.size + " 张 · " + (fraction * 100).toInt() + "%")
        }
        if (put.isFailure) {
            failed = put.exceptionOrNull()?.message ?: "图片上传失败，请重试"
        }
    }
    if (failed != null) {
        Toast.makeText(context, failed, Toast.LENGTH_SHORT).show()
        return null
    }
    return when (val completed = AppContainer.mediaRepository.complete(uploads.map { it.uploadId })) {
        is ApiResult.Ok -> completed.data.uploads.map { it.fileId }
        is ApiResult.Err -> {
            Toast.makeText(context, completed.error.userMessage, Toast.LENGTH_SHORT).show()
            null
        }
    }
}

private data class LocalImage(val bytes: ByteArray, val name: String, val mime: String)

private suspend fun readLocalImage(context: Context, uri: Uri): LocalImage? = withContext(Dispatchers.IO) {
    runCatching {
        val mime = context.contentResolver.getType(uri) ?: "image/jpeg"
        val bytes = context.contentResolver.openInputStream(uri)?.use { it.readBytes() }
            ?: error("无法读取图片")
        val displayName = queryDisplayName(context, uri)
        val fallbackExt = when {
            mime.contains("png") -> "png"
            mime.contains("webp") -> "webp"
            else -> "jpg"
        }
        LocalImage(
            bytes = bytes,
            name = displayName ?: ("moment_" + System.currentTimeMillis() + "." + fallbackExt),
            mime = mime,
        )
    }.getOrNull()
}

private fun queryDisplayName(context: Context, uri: Uri): String? = runCatching {
    context.contentResolver.query(uri, arrayOf(OpenableColumns.DISPLAY_NAME), null, null, null)?.use { cursor ->
        if (cursor.moveToFirst()) {
            val index = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME)
            if (index >= 0) cursor.getString(index) else null
        } else {
            null
        }
    }
}.getOrNull()
