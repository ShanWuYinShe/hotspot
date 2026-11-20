package com.hotspot.app.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Favorite
import androidx.compose.material.icons.filled.MailOutline
import androidx.compose.material.icons.filled.Share
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.runtime.withFrameNanos
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import coil.compose.AsyncImage
import coil.request.ImageRequest
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.CommentItem
import com.hotspot.app.data.model.MomentItem
import com.hotspot.app.data.model.UserInfo
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.CommentInputBar
import com.hotspot.app.ui.components.CommentThreadItem
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.InlineLoading
import com.hotspot.app.ui.components.ListEnd
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.PageHeader
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.replyChipLabel
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.HotSpotTextStyles
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.MediaUrls
import com.hotspot.app.util.TimeFormat
import com.hotspot.app.util.groupCommentThreads
import com.hotspot.app.util.mergeCommentPage
import com.hotspot.app.util.toggleCommentLike as applyOptimisticLike
import kotlinx.coroutines.launch
import java.util.UUID

/** 回复目标：parent_id 指向所属顶层评论（回复的回复也挂到同一顶层评论），mentions 带被 @ 的 user_id。 */
private data class CommentReplyTarget(
    val parentCommentId: String,
    val nickname: String,
    val userId: String,
)

/**
 * 瞬间详情：作者/图片/正文/计数 + 评论区（顶层评论 + 楼中楼「方案 B」）+ 常驻贴底评论输入条。
 *
 * 契约：docs/ui-components.md「详情页契约」「评论楼中楼（方案 B）」；
 * API：spec/api_spec_comment.md（fetch/create + 幂等键）/ spec/api_spec_reaction.md（评论点赞）。
 */
@Composable
fun MomentDetailScreen(
    momentId: String,
    onBack: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    val scope = rememberCoroutineScope()
    val listState = rememberLazyListState()
    val session by AppContainer.tokenStore.session.collectAsStateWithLifecycle()
    val loggedIn = session != null
    val commentRepository = AppContainer.commentRepository
    val feedRepository = AppContainer.feedRepository

    var moment by remember { mutableStateOf<MomentItem?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var currentUser by remember { mutableStateOf<UserInfo?>(null) }

    // 评论列表：顶层评论（含 top_replies 预览）+ cursor 分页；回复归组在渲染时按 parent_id 完成
    var comments by remember { mutableStateOf<List<CommentItem>>(emptyList()) }
    var commentsLoading by remember { mutableStateOf(true) }
    var commentsLoadingMore by remember { mutableStateOf(false) }
    var commentsError by remember { mutableStateOf<String?>(null) }
    var commentsCursor by remember { mutableStateOf<String?>(null) }
    var commentsEndReached by remember { mutableStateOf(false) }
    var commentCount by remember { mutableStateOf(0) }

    // 发送态：输入 / 回复目标 / 幂等键（成功后清理，失败保留供同一次提交重试）/ 动作错误
    var input by remember { mutableStateOf("") }
    var replyTarget by remember { mutableStateOf<CommentReplyTarget?>(null) }
    var sending by remember { mutableStateOf(false) }
    var actionError by remember { mutableStateOf<String?>(null) }
    var pendingIdempotencyKey by remember { mutableStateOf<String?>(null) }
    val likingIds = remember { mutableStateMapOf<String, Boolean>() }
    val inputFocusRequester = remember { FocusRequester() }

    suspend fun loadMoment() {
        loading = moment == null
        error = null
        when (val result = feedRepository.openMoment(momentId)) {
            is ApiResult.Ok -> {
                moment = result.data
                commentCount = result.data.commentsCount
            }
            is ApiResult.Err -> error = result.error.userMessage
        }
        loading = false
    }

    suspend fun loadComments() {
        commentsLoading = true
        commentsError = null
        when (val result = commentRepository.fetchComments(momentId, null)) {
            is ApiResult.Ok -> {
                comments = result.data.items
                commentsCursor = result.data.nextCursor
                commentsEndReached = result.data.nextCursor == null
            }
            is ApiResult.Err -> commentsError = result.error.userMessage
        }
        commentsLoading = false
    }

    suspend fun loadMoreComments() {
        val cursor = commentsCursor ?: return
        if (commentsLoading || commentsLoadingMore) return
        commentsLoadingMore = true
        commentsError = null
        when (val result = commentRepository.fetchComments(momentId, cursor)) {
            is ApiResult.Ok -> {
                comments = mergeCommentPage(comments, result.data.items)
                commentsCursor = result.data.nextCursor
                commentsEndReached = result.data.nextCursor == null
            }
            is ApiResult.Err -> commentsError = result.error.userMessage
        }
        commentsLoadingMore = false
    }

    // 发送/回复成功后的滚动请求（LazyColumn 索引依赖评论数，放入 effect 内在一帧后再滚）
    var scrollToEndRequest by remember { mutableStateOf(0) }
    var scrollToThreadRequest by remember { mutableStateOf<String?>(null) }

    fun startReply(parentCommentId: String, nickname: String, userId: String) {
        replyTarget = CommentReplyTarget(parentCommentId, nickname, userId)
        actionError = null
    }

    fun onCommentLikeClick(commentId: String) {
        if (!loggedIn || likingIds.containsKey(commentId)) return
        val target = comments.firstOrNull { it.commentId == commentId } ?: return
        if (target.isDeleted) return
        val active = !target.liked
        // 乐观更新：先翻转本地 reactions 聚合，失败回滚到点击前快照
        comments = comments.map { if (it.commentId == commentId) applyOptimisticLike(it, active) else it }
        likingIds[commentId] = true
        scope.launch {
            val result = commentRepository.setCommentReaction(commentId, active)
            if (result is ApiResult.Err) {
                comments = comments.map { if (it.commentId == commentId) target else it }
                actionError = result.error.userMessage
            }
            likingIds.remove(commentId)
        }
    }

    fun sendComment() {
        val text = input.trim()
        if (text.isEmpty() || sending || !loggedIn) return
        val target = replyTarget
        val idempotencyKey = pendingIdempotencyKey ?: UUID.randomUUID().toString()
        pendingIdempotencyKey = idempotencyKey
        sending = true
        actionError = null
        scope.launch {
            val result = commentRepository.createComment(
                momentId = momentId,
                content = text,
                parentId = target?.parentCommentId,
                mentions = target?.let { listOf(it.userId) } ?: emptyList(),
                idempotencyKey = idempotencyKey,
            )
            when (result) {
                is ApiResult.Ok -> {
                    val created = result.data
                    input = ""
                    replyTarget = null
                    pendingIdempotencyKey = null
                    commentCount += 1
                    if (target == null) {
                        comments = comments + created
                        scrollToEndRequest += 1
                    } else {
                        if (comments.any { it.commentId == target.parentCommentId }) {
                            comments = comments.map { root ->
                                if (root.commentId == target.parentCommentId) {
                                    root.copy(
                                        topReplies = root.topReplies + created,
                                        repliesCount = root.repliesCount + 1,
                                    )
                                } else {
                                    root
                                }
                            }
                        } else {
                            // 父评论不在当前页（分页边界）：重新拉首屏，回复不丢失
                            loadComments()
                        }
                        scrollToThreadRequest = target.parentCommentId
                    }
                }
                is ApiResult.Err -> actionError = result.error.userMessage
            }
            sending = false
        }
    }

    LaunchedEffect(momentId) { loadMoment() }
    LaunchedEffect(momentId) { loadComments() }
    LaunchedEffect(loggedIn) {
        if (!loggedIn || currentUser != null) return@LaunchedEffect
        when (val result = AppContainer.profileRepository.account()) {
            is ApiResult.Ok -> currentUser = UserInfo(
                userId = result.data.userId,
                nickname = result.data.nickname,
                avatarUrl = result.data.avatarUrl,
            )
            // 头像/昵称降级为首字母占位，不阻断评论
            is ApiResult.Err -> Unit
        }
    }
    // 回复点击后聚焦输入条（输入条常驻贴底，聚焦即唤起 IME 并显示回复目标）
    LaunchedEffect(replyTarget) {
        if (replyTarget != null) {
            withFrameNanos { }
            runCatching { inputFocusRequester.requestFocus() }
        }
    }
    LaunchedEffect(scrollToEndRequest) {
        if (scrollToEndRequest == 0) return@LaunchedEffect
        withFrameNanos { }
        val last = listState.layoutInfo.totalItemsCount - 1
        if (last >= 0) listState.animateScrollToItem(last)
    }
    LaunchedEffect(scrollToThreadRequest) {
        val targetId = scrollToThreadRequest ?: return@LaunchedEffect
        withFrameNanos { }
        val index = groupCommentThreads(comments).indexOfFirst { it.comment.commentId == targetId }
        if (index >= 0) listState.animateScrollToItem(COMMENT_ITEMS_OFFSET + index)
    }

    // 触底自动续载（简单可靠：距底 2 项内触发；错误态停止自动重试，交由行内「重试」）
    val shouldLoadMore by remember {
        derivedStateOf {
            val info = listState.layoutInfo
            val lastVisible = info.visibleItemsInfo.lastOrNull()?.index ?: -1
            info.totalItemsCount > 0 && lastVisible >= info.totalItemsCount - 2
        }
    }
    LaunchedEffect(shouldLoadMore, comments.size, commentsEndReached, commentsLoading, commentsLoadingMore, commentsError) {
        if (shouldLoadMore && comments.isNotEmpty() && !commentsLoading && !commentsLoadingMore &&
            !commentsEndReached && commentsError == null
        ) {
            loadMoreComments()
        }
    }

    Column(Modifier.fillMaxSize()) {
        PageHeader(title = "瞬间", onBack = onBack)
        when {
            loading && moment == null -> LoadingState()
            error != null && moment == null -> ErrorState(
                message = error ?: "加载失败",
                onRetry = { scope.launch { loadMoment() } },
            )
            moment == null -> LoadingState()
            else -> {
                val data = moment ?: return@Column
                val threads = groupCommentThreads(comments)
                LazyColumn(
                    state = listState,
                    modifier = Modifier.weight(1f),
                    contentPadding = PaddingValues(
                        start = Spacing.lg,
                        end = Spacing.lg,
                        top = Spacing.md,
                        bottom = Spacing.xl,
                    ),
                ) {
                    item(key = "moment-body") {
                        MomentBody(data = data, commentCount = commentCount)
                    }
                    item(key = "comments-header") {
                        CommentsSectionHeader(count = commentCount)
                    }
                    when {
                        commentsLoading && comments.isEmpty() -> item(key = "comments-loading") {
                            LoadingState(message = "加载评论中…")
                        }
                        commentsError != null && comments.isEmpty() -> item(key = "comments-error") {
                            ErrorState(
                                message = commentsError ?: "评论加载失败",
                                onRetry = { scope.launch { loadComments() } },
                            )
                        }
                        comments.isEmpty() -> item(key = "comments-empty") {
                            EmptyState(
                                title = "还没有评论",
                                description = "来说点什么吧，成为第一个评论的人。",
                            )
                        }
                        else -> {
                            items(threads, key = { "thread-" + it.comment.commentId }) { thread ->
                                CommentThreadItem(
                                    thread = thread,
                                    momentAuthorId = data.user.userId,
                                    actionsEnabled = loggedIn,
                                    onToggleLike = { commentId, _ -> onCommentLikeClick(commentId) },
                                    onReply = { parentCommentId, nickname, userId ->
                                        startReply(parentCommentId, nickname, userId)
                                    },
                                )
                            }
                            if (commentsError != null) {
                                item(key = "comments-more-error") {
                                    CommentsLoadMoreError(
                                        message = commentsError ?: "加载更多评论失败",
                                        onRetry = { scope.launch { loadMoreComments() } },
                                    )
                                }
                            }
                            if (commentsLoadingMore) {
                                item(key = "comments-more-loading") { InlineLoading() }
                            } else if (commentsEndReached) {
                                item(key = "comments-end") { ListEnd() }
                            }
                        }
                    }
                }
                if (actionError != null) {
                    Text(
                        text = actionError ?: "",
                        style = HotSpotTextStyles.caption,
                        color = colors.danger,
                        modifier = Modifier
                            .fillMaxWidth()
                            .background(colors.surface)
                            .padding(horizontal = Spacing.lg, vertical = Spacing.xs),
                    )
                }
                // 未登录（TokenStore 无 token）不展示输入条：只读浏览，不崩
                if (loggedIn) {
                    CommentInputBar(
                        value = input,
                        onValueChange = { value ->
                            input = value
                            // 文本被修改后是新的提交意图：丢弃上一次失败保留的幂等键
                            pendingIdempotencyKey = null
                        },
                        onSend = { sendComment() },
                        currentUser = currentUser,
                        replyLabel = replyTarget?.let { replyChipLabel(it.nickname) },
                        onCancelReply = { replyTarget = null },
                        sending = sending,
                        focusRequester = inputFocusRequester,
                    )
                }
            }
        }
    }
}

/** LazyColumn 里评论线程之前的固定项数（瞬间正文 + 评论分区标题），用于滚动定位。 */
private const val COMMENT_ITEMS_OFFSET = 2

/** 评论分区标题（对齐 Web SectionHead：标题 14 semibold + 计数 11 次级）。 */
@Composable
private fun CommentsSectionHeader(count: Int) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = Spacing.lg, bottom = Spacing.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = "评论讨论",
            style = MaterialTheme.typography.bodyLarge.copy(fontWeight = FontWeight.SemiBold),
            color = colors.textPrimary,
        )
        Spacer(Modifier.width(Spacing.sm2))
        Text(
            text = count.toString(),
            style = HotSpotTextStyles.caption,
            color = colors.textSecondary,
        )
    }
}

/** 加载更多评论失败：展示行内错误 + 重试（不吞掉已有评论）。 */
@Composable
private fun CommentsLoadMoreError(message: String, onRetry: () -> Unit) {
    val colors = HotSpotTheme.colors
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = Spacing.md),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            text = message,
            style = MaterialTheme.typography.bodySmall,
            color = colors.textSecondary,
        )
        Spacer(Modifier.height(Spacing.sm))
        SystemButton(
            text = "重试",
            onClick = onRetry,
            variant = ButtonVariant.SECONDARY,
            size = ButtonSize.SM,
        )
    }
}

/** 瞬间正文：作者行 / 图片 / 正文 / 互动计数（动作条与评论区在列表后续项）。 */
@Composable
private fun MomentBody(data: MomentItem, commentCount: Int) {
    val colors = HotSpotTheme.colors
    Column(Modifier.fillMaxWidth()) {
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
            DetailCount(Icons.Filled.MailOutline, commentCount)
            Spacer(Modifier.width(Spacing.xl))
            DetailCount(Icons.Filled.Share, data.sharesCount)
        }
        Spacer(Modifier.height(Spacing.xs))
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
