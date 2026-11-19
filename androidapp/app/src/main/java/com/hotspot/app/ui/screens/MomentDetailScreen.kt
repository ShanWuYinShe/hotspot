package com.hotspot.app.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Favorite
import androidx.compose.material.icons.filled.MailOutline
import androidx.compose.material.icons.filled.Share
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import coil.request.ImageRequest
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.MomentItem
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.MediaUrls
import com.hotspot.app.util.TimeFormat
import kotlinx.coroutines.launch

/**
 * 瞬间详情：首版只读（作者/图片/正文/计数）。
 * 评论区与评论输入条为登记 TODO（后端 comments/fetch + comments/create 已就绪，UI 后续补齐）。
 */
@Composable
fun MomentDetailScreen(
    momentId: String,
    onBack: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    val scope = rememberCoroutineScope()
    var moment by remember { mutableStateOf<MomentItem?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }

    suspend fun load() {
        loading = moment == null
        error = null
        when (val result = AppContainer.feedRepository.openMoment(momentId)) {
            is ApiResult.Ok -> moment = result.data
            is ApiResult.Err -> error = result.error.userMessage
        }
        loading = false
    }

    LaunchedEffect(momentId) { load() }

    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "瞬间", onBack = onBack)
        when {
            loading -> LoadingState()
            error != null && moment == null -> ErrorState(
                message = error ?: "加载失败",
                onRetry = { scope.launch { load() } },
            )
            moment == null -> LoadingState()
            else -> {
                val data = moment ?: return@Column
                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .verticalScroll(rememberScrollState())
                        .padding(horizontal = Spacing.lg),
                ) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Avatar(
                            name = data.user.nickname,
                            imageUrl = data.user.avatarUrl,
                            size = Dimens.avatarSm,
                        )
                        Spacer(Modifier.width(Spacing.sm))
                        Column(Modifier.weight(1f)) {
                            Text(
                                text = data.user.nickname.ifBlank { "匿名用户" },
                                style = MaterialTheme.typography.bodyLarge,
                                fontWeight = FontWeight.Medium,
                                color = colors.textPrimary,
                            )
                            Text(
                                text = TimeFormat.relative(data.createdAt) +
                                    (data.location?.locationName?.let { " · " + it } ?: ""),
                                style = MaterialTheme.typography.labelSmall,
                                color = colors.textSecondary,
                            )
                        }
                    }
                    Spacer(Modifier.height(Spacing.md))
                    val imageUrl = MediaUrls.absolute(data.images.firstOrNull()?.url)
                    if (imageUrl != null) {
                        AsyncImage(
                            model = ImageRequest.Builder(LocalContext.current).data(imageUrl).crossfade(true).build(),
                            contentDescription = data.description.ifBlank { "瞬间图片" },
                            modifier = Modifier
                                .fillMaxWidth()
                                .aspectRatio(4f / 5f)
                                .clip(AppShapes.card)
                                .background(colors.subtle),
                            contentScale = ContentScale.Crop,
                        )
                        Spacer(Modifier.height(Spacing.md))
                    }
                    if (data.description.isNotBlank()) {
                        Text(
                            text = data.description,
                            style = MaterialTheme.typography.bodyLarge,
                            color = colors.textPrimary,
                        )
                    }
                    Spacer(Modifier.height(Spacing.md))
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        DetailCount(Icons.Filled.Favorite, data.likeCount)
                        Spacer(Modifier.width(Spacing.xl))
                        DetailCount(Icons.Filled.MailOutline, data.commentsCount)
                        Spacer(Modifier.width(Spacing.xl))
                        DetailCount(Icons.Filled.Share, data.sharesCount)
                    }
                    Spacer(Modifier.height(Spacing.lg))
                    Text(
                        text = "评论区 TODO：首版详情页只读，评论列表与评论输入条后续接入（spec/api_spec_comment.md）。",
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.textSecondary,
                    )
                    Spacer(Modifier.height(Spacing.xxl))
                }
            }
        }
    }
}

@Composable
private fun DetailCount(icon: ImageVector, count: Int) {
    val colors = HotSpotTheme.colors
    Row(verticalAlignment = Alignment.CenterVertically) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            modifier = Modifier.size(18.dp),
            tint = colors.textSecondary,
        )
        Spacer(Modifier.width(Spacing.xs))
        Text(
            text = count.toString(),
            style = MaterialTheme.typography.labelSmall,
            color = colors.textSecondary,
        )
    }
}
