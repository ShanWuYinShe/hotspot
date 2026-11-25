package com.hotspot.app.ui.screens

import android.content.Intent
import android.widget.Toast
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.statusBars
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Favorite
import androidx.compose.material.icons.filled.MailOutline
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Surface
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.runtime.withFrameNanos
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import coil.compose.AsyncImage
import coil.request.ImageRequest
import com.hotspot.app.AppContainer
import com.hotspot.app.R
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.CommentItem
import com.hotspot.app.data.model.DefaultLocation
import com.hotspot.app.data.model.GeoPoint
import com.hotspot.app.data.model.MomentItem
import com.hotspot.app.data.model.UserInfo
import com.hotspot.app.ui.components.Avatar
import com.hotspot.app.ui.components.CommentInputBar
import com.hotspot.app.ui.components.FullscreenImageViewer
import com.hotspot.app.ui.components.CommentThreadItem
import com.hotspot.app.ui.components.EmptyState
import com.hotspot.app.ui.components.ErrorState
import com.hotspot.app.ui.components.InlineLoading
import com.hotspot.app.ui.components.ListEnd
import com.hotspot.app.ui.components.LoadingState
import com.hotspot.app.ui.components.MoodChips
import com.hotspot.app.ui.components.ReportDialog
import com.hotspot.app.ui.components.SegmentedTabs
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemTextArea
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.hotSpotFocusRing
import com.hotspot.app.ui.components.replyChipLabel
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.ChromeInsets
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.HotSpotTextStyles
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.MediaUrls
import com.hotspot.app.util.TimeFormat
import com.hotspot.app.util.findCommentInThreads
import com.hotspot.app.util.groupCommentThreads
import com.hotspot.app.util.markCommentDeleted
import com.hotspot.app.util.mergeCommentPage
import com.hotspot.app.util.updateCommentInThreads
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
    focusCommentId: String? = null,
    onBack: () -> Unit,
    onOpenUser: (userId: String) -> Unit,
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
    // 楼中楼「查看全部回复」加载中的顶层评论 id
    var expandingReplyIds by remember { mutableStateOf<Set<String>>(emptySet()) }
    // 删除确认：仅评论作者本人可删（spec/comments/delete 软删除）
    var commentDeleteTarget by remember { mutableStateOf<CommentItem?>(null) }
    var commentDeleting by remember { mutableStateOf(false) }
    // 评论举报（reports/create target_type=comment）：仅非本人评论可举报
    var commentReportTarget by remember { mutableStateOf<CommentItem?>(null) }
    var commentReportSubmitting by remember { mutableStateOf(false) }
    val inputFocusRequester = remember { FocusRequester() }
    // 收藏态：以 open_the_door 的 is_favorited 为初始值，点击即时翻转（失败回滚）
    var favorite by remember { mutableStateOf(false) }
    var favoriteBusy by remember { mutableStateOf(false) }
    // 作者操作：更多菜单 / 编辑文案 / 删除瞬间
    var ownerSheetOpen by remember { mutableStateOf(false) }
    var editDialogOpen by remember { mutableStateOf(false) }
    // 可见范围只能在拥有者视角接口拿到（open_the_door 不含 privacy_level），编辑前先查 fetch_created。
    var editPrivacy by remember { mutableStateOf("public") }
    var editPrivacyKnown by remember { mutableStateOf(false) }
    var editPreparing by remember { mutableStateOf(false) }
    var maxMomentTags by remember { mutableIntStateOf(5) }

    // 头图通栏：本页内容从屏幕顶开始（图片铺到状态栏下），离开时恢复全局安全区。
    DisposableEffect(Unit) {
        ChromeInsets.edgeToEdgeTop = true
        ChromeInsets.lightStatusBarIcons = false
        onDispose {
            ChromeInsets.edgeToEdgeTop = false
            ChromeInsets.lightStatusBarIcons = false
        }
    }
    val hasHeroImage = moment?.images?.isNotEmpty() == true
    // 头图还在状态栏下方时才用浅色图标；图片滚出后及时切回主题图标，避免白底白字。
    val statusBarPx = with(LocalDensity.current) {
        WindowInsets.statusBars.getTop(LocalDensity.current).toFloat()
    }
    val heroCoversStatusBar by remember(hasHeroImage) {
        derivedStateOf {
            if (!hasHeroImage) return@derivedStateOf false
            val hero = listState.layoutInfo.visibleItemsInfo.firstOrNull { it.index == 0 }
                ?: return@derivedStateOf false
            hero.offset + hero.size > statusBarPx
        }
    }
    LaunchedEffect(heroCoversStatusBar) { ChromeInsets.lightStatusBarIcons = heroCoversStatusBar }
    var editSaving by remember { mutableStateOf(false) }
    var editError by remember { mutableStateOf<String?>(null) }
    var locationPickerOpen by remember { mutableStateOf(false) }
    var locationSaving by remember { mutableStateOf(false) }
    var deleteConfirmOpen by remember { mutableStateOf(false) }
    var deleteBusy by remember { mutableStateOf(false) }
    var showMomentReport by remember { mutableStateOf(false) }
    var reportSubmitting by remember { mutableStateOf(false) }
    // 全屏图片查看器（点击多图 hero；携带整组图片支持查看器内左右翻页）
    var viewerImages by remember { mutableStateOf<List<String>>(emptyList()) }
    var viewerIndex by remember { mutableIntStateOf(0) }
    val context = LocalContext.current
    val isOwner = moment?.user?.userId != null && moment?.user?.userId == currentUser?.userId

    suspend fun loadMoment() {
        loading = moment == null
        error = null
        when (val result = feedRepository.openMoment(momentId)) {
            is ApiResult.Ok -> {
                moment = result.data
                commentCount = result.data.commentsCount
                favorite = result.data.isFavorited
            }
            is ApiResult.Err -> error = result.error.userMessage
        }
        loading = false
    }

    /**
     * 收藏 / 取消收藏（spec/api_spec_moment.md → moments/set_favorite）。
     * 乐观翻转 + 失败回滚；未登录时只提示不请求（详情页本身可匿名浏览？本页登录门禁下不可达，纯防御）。
     */
    fun toggleFavorite() {
        if (favoriteBusy) return
        if (!loggedIn) {
            actionError = "登录后才能收藏瞬间"
            return
        }
        val next = !favorite
        favorite = next
        favoriteBusy = true
        actionError = null
        scope.launch {
            when (val result = feedRepository.setFavorite(momentId, next)) {
                is ApiResult.Ok -> Unit
                is ApiResult.Err -> {
                    favorite = !next
                    actionError = result.error.userMessage
                }
            }
            favoriteBusy = false
        }
    }

    /** 分享瞬间：优先 share/create 生成可撤销链接，失败/匿名场景降级为原始详情链接。 */
    fun shareMoment() {
        val authorName = moment?.user?.nickname?.ifBlank { "HotSpot 用户" } ?: "HotSpot 用户"
        scope.launch {
            val fallbackUrl = "https://chaoset.com/moments/" + momentId
            val url = when (val result = AppContainer.profileRepository.createMomentShare(momentId)) {
                is ApiResult.Ok -> result.data.shareCode.takeIf { it.isNotBlank() }
                    ?.let { "https://chaoset.com/share/" + it }
                    ?: fallbackUrl
                is ApiResult.Err -> fallbackUrl
            }
            val send = Intent(Intent.ACTION_SEND).apply {
                type = "text/plain"
                putExtra(Intent.EXTRA_SUBJECT, authorName + "的瞬间")
                putExtra(Intent.EXTRA_TEXT, "「" + authorName + "」的瞬间：" + url)
            }
            context.startActivity(Intent.createChooser(send, "分享瞬间"))
        }
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
    // 通知/外链带入的评论 id：首屏评论加载后滚动到对应线程（只处理一次，避免新评论触发重复滚动）
    var handledFocusCommentId by remember { mutableStateOf<String?>(null) }
    LaunchedEffect(focusCommentId, commentsLoading, comments) {
        val targetId = focusCommentId?.takeIf { it.isNotBlank() } ?: return@LaunchedEffect
        if (handledFocusCommentId == targetId || commentsLoading) return@LaunchedEffect
        val threads = groupCommentThreads(comments)
        val index = threads.indexOfFirst { thread ->
            thread.comment.commentId == targetId || thread.replies.any { it.commentId == targetId }
        }
        if (index >= 0) {
            handledFocusCommentId = targetId
            withFrameNanos { }
            listState.animateScrollToItem(COMMENT_ITEMS_OFFSET + index)
        } else if (!commentsEndReached && !commentsLoadingMore && !commentsLoading) {
            // 目标评论不在首屏：自动续页继续找，直到命中或评论到底（避免用户手动下滑）。
            loadMoreComments()
        }
    }

    fun startReply(parentCommentId: String, nickname: String, userId: String) {
        replyTarget = CommentReplyTarget(parentCommentId, nickname, userId)
        actionError = null
    }

    /** 「查看全部 N 条回复」：拉取 fetch_replies 全量并回填到该顶层评论的 top_replies。 */
    fun expandReplies(rootCommentId: String) {
        if (!loggedIn || rootCommentId in expandingReplyIds) return
        expandingReplyIds = expandingReplyIds + rootCommentId
        scope.launch {
            when (val result = commentRepository.fetchReplies(rootCommentId)) {
                is ApiResult.Ok -> {
                    val replies = result.data.items
                    comments = comments.map { root ->
                        if (root.commentId == rootCommentId) {
                            root.copy(topReplies = replies, repliesCount = replies.size)
                        } else {
                            root
                        }
                    }
                }
                is ApiResult.Err -> actionError = result.error.userMessage
            }
            expandingReplyIds = expandingReplyIds - rootCommentId
        }
    }

    /** 删除自己的评论（软删除：content 清空、保留回复；确认后调用）。 */
    fun deleteComment(commentId: String) {
        if (commentDeleting) return
        commentDeleting = true
        actionError = null
        scope.launch {
            when (val result = commentRepository.deleteComment(commentId)) {
                is ApiResult.Ok -> {
                    comments = markCommentDeleted(comments, commentId)
                    commentCount = (commentCount - 1).coerceAtLeast(0)
                    commentDeleteTarget = null
                }
                is ApiResult.Err -> actionError = result.error.userMessage
            }
            commentDeleting = false
        }
    }

    fun onCommentLikeClick(commentId: String) {
        if (!loggedIn || likingIds.containsKey(commentId)) return
        val target = findCommentInThreads(comments, commentId) ?: return
        if (target.isDeleted) return
        val active = !target.liked
        // 乐观更新：顶层评论与楼中楼回复共用同一套 reactions 聚合，失败回滚到点击前快照
        comments = updateCommentInThreads(comments, commentId) { applyOptimisticLike(it, active) }
        likingIds[commentId] = true
        scope.launch {
            val result = commentRepository.setCommentReaction(commentId, active)
            if (result is ApiResult.Err) {
                comments = updateCommentInThreads(comments, commentId) { target }
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

    LaunchedEffect(momentId) {
        loadMoment()
        runCatching { AppContainer.limits() }.getOrNull()?.let { maxMomentTags = it.maxMomentTags.coerceAtLeast(1) }
    }
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

    Box(Modifier.fillMaxSize()) {
        Column(
            Modifier
                .fillMaxSize()
                .then(if (moment == null) Modifier.windowInsetsPadding(WindowInsets.statusBars) else Modifier),
        ) {
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
                    contentPadding = PaddingValues(bottom = Spacing.xl),
                ) {
                    item(key = "hero") {
                        MomentHero(data = data, onOpenImage = { images, index ->
                            viewerImages = images
                            viewerIndex = index
                        })
                    }
                    item(key = "moment-body") {
                        MomentBody(
                            data = data,
                            commentCount = commentCount,
                            favorite = favorite,
                            favoriteBusy = favoriteBusy,
                            onToggleFavorite = { toggleFavorite() },
                            onAuthorClick = { onOpenUser(data.user.userId) },
                            modifier = Modifier.padding(horizontal = Spacing.lg),
                        )
                    }
                    item(key = "comments-header") {
                        CommentsSectionHeader(
                            count = commentCount,
                            modifier = Modifier.padding(horizontal = Spacing.lg),
                        )
                    }
                    when {
                        commentsLoading && comments.isEmpty() -> item(key = "comments-loading") {
                            Box(Modifier.padding(horizontal = Spacing.lg)) {
                                LoadingState(message = "加载评论中…")
                            }
                        }
                        commentsError != null && comments.isEmpty() -> item(key = "comments-error") {
                            Box(Modifier.padding(horizontal = Spacing.lg)) {
                                ErrorState(
                                    message = commentsError ?: "评论加载失败",
                                    onRetry = { scope.launch { loadComments() } },
                                )
                            }
                        }
                        comments.isEmpty() -> item(key = "comments-empty") {
                            Box(Modifier.padding(horizontal = Spacing.lg)) {
                                EmptyState(
                                    title = "还没有评论",
                                    description = "来说点什么吧，成为第一个评论的人。",
                                )
                            }
                        }
                        else -> {
                            items(threads, key = { "thread-" + it.comment.commentId }) { thread ->
                                Box(Modifier.padding(horizontal = Spacing.lg)) {
                                    CommentThreadItem(
                                        thread = thread,
                                        momentAuthorId = data.user.userId,
                                        currentUserId = currentUser?.userId.orEmpty(),
                                        actionsEnabled = loggedIn,
                                        expandingReplyIds = expandingReplyIds,
                                        onToggleLike = { commentId, _ -> onCommentLikeClick(commentId) },
                                        onReply = { parentCommentId, nickname, userId ->
                                            startReply(parentCommentId, nickname, userId)
                                        },
                                        onExpandReplies = { rootCommentId -> expandReplies(rootCommentId) },
                                        onDelete = { comment -> commentDeleteTarget = comment },
                                        onReport = { comment -> commentReportTarget = comment },
                                    )
                                }
                            }
                            if (commentsError != null) {
                                item(key = "comments-more-error") {
                                    Box(Modifier.padding(horizontal = Spacing.lg)) {
                                        CommentsLoadMoreError(
                                            message = commentsError ?: "加载更多评论失败",
                                            onRetry = { scope.launch { loadMoreComments() } },
                                        )
                                    }
                                }
                            }
                            if (commentsLoadingMore) {
                                item(key = "comments-more-loading") {
                                    Box(Modifier.padding(horizontal = Spacing.lg)) { InlineLoading() }
                                }
                            } else if (commentsEndReached) {
                                item(key = "comments-end") {
                                    Box(Modifier.padding(horizontal = Spacing.lg)) { ListEnd() }
                                }
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
        // 浮层动作：返回 / 分享（设计稿 87:3687 在 hero 上浮动玻璃圆钮，滚动时固定可见）
        MomentTopOverlay(
            onBack = onBack,
            onShare = { shareMoment() },
            showMore = loggedIn,
            onMore = { ownerSheetOpen = true },
        )
    }

    if (ownerSheetOpen) {
        MomentActionSheet(
            isOwner = isOwner,
            onDismiss = { ownerSheetOpen = false },
            onEdit = {
                ownerSheetOpen = false
                editError = null
                editPrivacyKnown = false
                if (!editPreparing) {
                    editPreparing = true
                    scope.launch {
                        when (val result = AppContainer.profileRepository.createdMoments(null)) {
                            is ApiResult.Ok -> result.data.items
                                .firstOrNull { it.momentId == momentId }
                                ?.let { item ->
                                    editPrivacy = item.privacyLevel ?: "public"
                                    editPrivacyKnown = true
                                }
                            is ApiResult.Err -> editError = result.error.userMessage
                        }
                        editPreparing = false
                        editDialogOpen = true
                    }
                }
            },
            onEditLocation = {
                ownerSheetOpen = false
                locationPickerOpen = true
            },
            onDelete = {
                ownerSheetOpen = false
                deleteConfirmOpen = true
            },
            onReport = {
                ownerSheetOpen = false
                showMomentReport = true
            },
        )
    }

    if (showMomentReport) {
        ReportDialog(
            title = "举报瞬间",
            submitting = reportSubmitting,
            onDismiss = { if (!reportSubmitting) showMomentReport = false },
            onSubmit = { reason ->
                if (reportSubmitting) return@ReportDialog
                reportSubmitting = true
                scope.launch {
                    when (val result = AppContainer.reportsRepository.create(
                        targetType = "moment",
                        targetId = momentId,
                        reason = reason,
                    )) {
                        is ApiResult.Ok -> {
                            showMomentReport = false
                            Toast.makeText(
                                context,
                                if (result.data.isDuplicate) "已提交过举报，等待处理" else "举报已提交",
                                Toast.LENGTH_SHORT,
                            ).show()
                        }
                        is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                    }
                    reportSubmitting = false
                }
            },
        )
    }

    if (editDialogOpen) {
        MomentEditDialog(
            initialDescription = moment?.description.orEmpty(),
            initialPrivacy = if (editPrivacyKnown) editPrivacy else null,
            initialTags = moment?.tags.orEmpty(),
            maxTags = maxMomentTags,
            saving = editSaving,
            error = editError,
            onDismiss = { if (!editSaving) editDialogOpen = false },
            onSubmit = { description, privacyLevel, tags ->
                if (editSaving) return@MomentEditDialog
                editSaving = true
                editError = null
                scope.launch {
                    when (val result = feedRepository.updateMoment(momentId, description, privacyLevel, tags)) {
                        is ApiResult.Ok -> {
                            moment = result.data
                            editDialogOpen = false
                        }
                        is ApiResult.Err -> editError = result.error.userMessage
                    }
                    editSaving = false
                }
            },
        )
    }

    if (deleteConfirmOpen) {
        AlertDialog(
            onDismissRequest = { if (!deleteBusy) deleteConfirmOpen = false },
            title = { Text("删除瞬间") },
            text = { Text("删除后瞬间将从所有列表移除，确定删除吗？") },
            confirmButton = {
                SystemButton(
                    text = if (deleteBusy) "删除中…" else "删除",
                    onClick = {
                        if (deleteBusy) return@SystemButton
                        deleteBusy = true
                        scope.launch {
                            when (val result = feedRepository.deleteMoment(momentId)) {
                                is ApiResult.Ok -> {
                                    deleteConfirmOpen = false
                                    onBack()
                                }
                                is ApiResult.Err -> {
                                    actionError = result.error.userMessage
                                    deleteConfirmOpen = false
                                }
                            }
                            deleteBusy = false
                        }
                    },
                    variant = ButtonVariant.DANGER,
                    size = ButtonSize.SM,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { deleteConfirmOpen = false },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                )
            },
        )
    }

    commentReportTarget?.let { target ->
        ReportDialog(
            title = "举报评论",
            submitting = commentReportSubmitting,
            onDismiss = { if (!commentReportSubmitting) commentReportTarget = null },
            onSubmit = { reason ->
                if (commentReportSubmitting) return@ReportDialog
                commentReportSubmitting = true
                scope.launch {
                    when (
                        val result = AppContainer.reportsRepository.create(
                            targetType = "comment",
                            targetId = target.commentId,
                            reason = reason,
                        )
                    ) {
                        is ApiResult.Ok -> {
                            commentReportTarget = null
                            Toast.makeText(
                                context,
                                if (result.data.isDuplicate) "已提交过举报，等待处理" else "举报已提交",
                                Toast.LENGTH_SHORT,
                            ).show()
                        }
                        is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                    }
                    commentReportSubmitting = false
                }
            },
        )
    }

    if (viewerImages.isNotEmpty()) {
        FullscreenImageViewer(
            images = viewerImages,
            initialIndex = viewerIndex,
            contentDescription = "瞬间图片",
            onDismiss = { viewerImages = emptyList() },
        )
    }

    commentDeleteTarget?.let { target ->
        AlertDialog(
            onDismissRequest = { if (!commentDeleting) commentDeleteTarget = null },
            title = { Text("删除评论") },
            text = { Text("删除后不可恢复，确定删除这条评论吗？") },
            confirmButton = {
                SystemButton(
                    text = if (commentDeleting) "删除中…" else "删除",
                    onClick = { deleteComment(target.commentId) },
                    variant = ButtonVariant.DANGER,
                    size = ButtonSize.SM,
                )
            },
            dismissButton = {
                SystemButton(
                    text = "取消",
                    onClick = { commentDeleteTarget = null },
                    variant = ButtonVariant.SECONDARY,
                    size = ButtonSize.SM,
                )
            },
        )
    }

    if (locationPickerOpen) {
        Dialog(
            onDismissRequest = { if (!locationSaving) locationPickerOpen = false },
            properties = DialogProperties(usePlatformDefaultWidth = false, decorFitsSystemWindows = false),
        ) {
            Surface(Modifier.fillMaxSize(), color = HotSpotTheme.colors.page) {
                Box(Modifier.fillMaxSize().statusBarsPadding().navigationBarsPadding()) {
                    MapScreen(
                        onBack = { if (!locationSaving) locationPickerOpen = false },
                        onAnchorSaved = {},
                        pickMode = true,
                        initialPoint = moment?.location?.point,
                        initialLocationName = moment?.location?.locationName,
                        onPicked = { point, label ->
                            if (!locationSaving) {
                                locationSaving = true
                                scope.launch {
                                    val location = DefaultLocation(
                                        point = GeoPoint(point.latitude, point.longitude),
                                        locationName = label.ifBlank { "自定义位置" },
                                    )
                                    when (val result = feedRepository.updateMoment(momentId, location = location)) {
                                        is ApiResult.Ok -> {
                                            moment = result.data
                                            locationPickerOpen = false
                                            Toast.makeText(context, "位置已更新", Toast.LENGTH_SHORT).show()
                                        }
                                        is ApiResult.Err -> Toast.makeText(context, result.error.userMessage, Toast.LENGTH_SHORT).show()
                                    }
                                    locationSaving = false
                                }
                            }
                        },
                    )
                }
            }
        }
    }
}

/** LazyColumn 里评论线程之前固定项数（hero + 瞬间正文 + 评论分区标题），用于滚动定位。 */
private const val COMMENT_ITEMS_OFFSET = 3

/** 评论分区标题（对齐 Web SectionHead：标题 14 semibold + 计数 11 次级）。 */
@Composable
private fun CommentsSectionHeader(count: Int, modifier: Modifier = Modifier) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = modifier
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

/**
 * 瞬间正文（对齐设计稿 87:3687 瞬间正文卡片 / MomentContentBlock）：
 * 作者栏（头像 36 + 昵称 + 位置/发布时间次级行）→ 正文 → 话题标签 → 互动计数。
 * hero 图片由 LazyColumn 首项 [MomentHero] 通栏渲染；返回/分享在 [MomentTopOverlay] 浮层。
 */
@Composable
private fun MomentBody(
    data: MomentItem,
    commentCount: Int,
    favorite: Boolean,
    favoriteBusy: Boolean,
    onToggleFavorite: () -> Unit,
    modifier: Modifier = Modifier,
    onAuthorClick: (() -> Unit)? = null,
) {
    val colors = HotSpotTheme.colors
    Column(modifier.fillMaxWidth()) {
        Row(
            modifier = Modifier.then(
                if (onAuthorClick != null) Modifier.clickable { onAuthorClick() } else Modifier,
            ),
            verticalAlignment = Alignment.CenterVertically,
        ) {
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
                    text = (data.location?.locationName?.let { it + " · " } ?: "") +
                        TimeFormat.relative(data.createdAt),
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
            }
        }
        Spacer(Modifier.height(Spacing.md))
        if (data.description.isNotBlank()) {
            Text(
                text = data.description,
                style = MaterialTheme.typography.bodyLarge,
                color = colors.textPrimary,
            )
        }
        if (data.tags.isNotEmpty()) {
            Spacer(Modifier.height(Spacing.sm2))
            Row(horizontalArrangement = Arrangement.spacedBy(Spacing.sm)) {
                data.tags.take(3).forEach { tag -> TopicTag(tag) }
            }
        }
        Spacer(Modifier.height(Spacing.lg))
        // 动作条（设计稿 87:3687 PostActions）：左 点赞数 + 评论数；右 收藏两态（分享在顶部浮层）。
        Row(verticalAlignment = Alignment.CenterVertically) {
            DetailCount(Icons.Filled.Favorite, data.likeCount)
            Spacer(Modifier.width(Spacing.xl))
            DetailCount(Icons.Filled.MailOutline, commentCount)
            Spacer(Modifier.weight(1f))
            MomentIconButton(
                iconRes = if (favorite) R.drawable.ic_bookmark_filled else R.drawable.ic_bookmark,
                contentDescription = if (favorite) "取消收藏" else "收藏瞬间",
                tint = if (favorite) colors.accent else colors.textSecondary,
                enabled = !favoriteBusy,
                onClick = onToggleFavorite,
            )
        }
        Spacer(Modifier.height(Spacing.xs))
    }
}

/** hero 宽高比（设计稿 390x280：图片通栏、无圆角，作者栏紧随其下）。 */
private const val MomentHeroAspect = 390f / 280f

/** 无图瞬间的顶部留白：让出浮层按钮高度，避免作者栏被返回/分享钮压住。 */
private val MomentHeroNoImageTop = 64.dp

/** 头图顶部压暗渐变高度（状态栏 + 浮动圆钮可读即可，不压暗整张图）。 */
private val MomentHeroScrimHeight = 120.dp

/**
 * hero 头图：通栏贴边、无圆角；多图瞬间用 HorizontalPager + 底部圆点指示器，点击当前图打开全屏查看器。
 * 无图瞬间用固定留白占位（纯文字瞬间不造假图）。
 */
@Composable
private fun MomentHero(data: MomentItem, onOpenImage: (List<String>, Int) -> Unit) {
    val colors = HotSpotTheme.colors
    val images = data.images.mapNotNull { MediaUrls.absolute(it.url) }.distinct()
    if (images.isEmpty()) {
        // 无图瞬间没有头图可衬托，仍让出系统状态栏 + 浮层按钮高度。
        Spacer(
            Modifier
                .height(MomentHeroNoImageTop)
                .windowInsetsPadding(WindowInsets.statusBars),
        )
        return
    }
    val pagerState = rememberPagerState(pageCount = { images.size })
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .aspectRatio(MomentHeroAspect)
            .background(colors.subtle),
    ) {
        HorizontalPager(state = pagerState, modifier = Modifier.fillMaxSize()) { page ->
            AsyncImage(
                model = ImageRequest.Builder(LocalContext.current).data(images[page]).crossfade(true).build(),
                contentDescription = data.description.ifBlank { "瞬间图片" },
                modifier = Modifier
                    .fillMaxSize()
                    .clickable { onOpenImage(images, page) },
                contentScale = ContentScale.Crop,
            )
        }
        // 顶部压暗：状态栏图标与浮动圆钮在亮图上仍可读；随 hero 一起滚动，不污染正文。
        Box(
            Modifier
                .fillMaxWidth()
                .height(MomentHeroScrimHeight)
                .align(Alignment.TopCenter)
                .background(
                    Brush.verticalGradient(
                        listOf(Color.Black.copy(alpha = 0.45f), Color.Transparent),
                    ),
                ),
        )
        if (images.size > 1) {
            Row(
                modifier = Modifier
                    .align(Alignment.BottomCenter)
                    .padding(bottom = Spacing.md)
                    .clip(AppShapes.pill)
                    .background(colors.island.copy(alpha = 0.72f))
                    .padding(horizontal = Spacing.sm2, vertical = Spacing.xs),
                horizontalArrangement = Arrangement.spacedBy(4.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                repeat(images.size) { index ->
                    Box(
                        modifier = Modifier
                            .size(6.dp)
                            .clip(CircleShape)
                            .background(
                                if (index == pagerState.currentPage) colors.accent else colors.textSecondary.copy(alpha = 0.4f),
                            ),
                    )
                }
            }
        }
    }
}

/**
 * 顶部浮层动作（设计稿 87:3687：hero 左上返回 / 右上分享，36 玻璃圆钮 + 发丝边 + 轻投影）；
 * 固定在滚动区上层，与 Web DetailHeader 一致。更多菜单（举报/编辑等）待后端能力接入后补。
 */
@Composable
private fun MomentTopOverlay(
    onBack: () -> Unit,
    onShare: () -> Unit,
    showMore: Boolean = false,
    onMore: () -> Unit = {},
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .windowInsetsPadding(WindowInsets.statusBars)
            .padding(horizontal = Spacing.lg, vertical = Spacing.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        GlassCircleButton(
            icon = Icons.AutoMirrored.Filled.ArrowBack,
            contentDescription = "返回",
            onClick = onBack,
        )
        Spacer(Modifier.weight(1f))
        if (showMore) {
            GlassCircleButton(
                iconRes = R.drawable.ic_action_more,
                contentDescription = "更多操作",
                onClick = onMore,
            )
            Spacer(Modifier.width(Spacing.sm))
        }
        GlassCircleButton(
            iconRes = R.drawable.ic_share,
            contentDescription = "分享瞬间",
            onClick = onShare,
        )
    }
}

/** 36 玻璃圆钮：island 底 92% + 发丝边 + 轻投影（hero 上可读；对齐 Web DetailHeader 圆钮）。 */
@Composable
private fun GlassCircleButton(
    contentDescription: String,
    onClick: () -> Unit,
    icon: ImageVector? = null,
    iconRes: Int? = null,
) {
    val colors = HotSpotTheme.colors
    Box(
        modifier = Modifier
            .size(36.dp)
            .shadow(
                elevation = 2.dp,
                shape = CircleShape,
                ambientColor = colors.shadowDark,
                spotColor = colors.shadowDark,
            )
            .clip(CircleShape)
            .background(colors.island.copy(alpha = 0.92f))
            .border(1.dp, colors.border, CircleShape)
            .clickable(
                interactionSource = remember { MutableInteractionSource() },
                indication = null,
                onClick = onClick,
            )
            .hotSpotFocusRing(cornerRadius = 18.dp),
        contentAlignment = Alignment.Center,
    ) {
        if (icon != null) {
            Icon(
                imageVector = icon,
                contentDescription = contentDescription,
                modifier = Modifier.size(18.dp),
                tint = colors.textPrimary,
            )
        } else if (iconRes != null) {
            Icon(
                painter = painterResource(iconRes),
                contentDescription = contentDescription,
                modifier = Modifier.size(18.dp),
                tint = colors.textPrimary,
            )
        }
    }
}

/** 话题标签（设计稿 87:3687 TopicTag：#云龙湖 24 高胶囊、accent 软底）。 */
@Composable
private fun TopicTag(tag: String) {
    val colors = HotSpotTheme.colors
    Box(
        modifier = Modifier
            .clip(AppShapes.pill)
            .background(colors.accentSoft)
            .padding(horizontal = Spacing.md2, vertical = Spacing.xs),
    ) {
        Text(
            text = "#$tag",
            style = MaterialTheme.typography.labelSmall,
            color = colors.accent,
        )
    }
}

/** 图标动作钮：44dp 触控热区 + 圆形焦点环，图标 18（与动作条数字同视觉重量）。 */
@Composable
private fun MomentIconButton(
    iconRes: Int,
    contentDescription: String,
    tint: Color,
    enabled: Boolean = true,
    onClick: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    Box(
        modifier = Modifier
            .size(Dimens.minTouchTarget)
            .clip(CircleShape)
            .clickable(
                interactionSource = remember { MutableInteractionSource() },
                indication = null,
                enabled = enabled,
                onClick = onClick,
            )
            .hotSpotFocusRing(cornerRadius = 22.dp),
        contentAlignment = Alignment.Center,
    ) {
        Icon(
            painter = painterResource(iconRes),
            contentDescription = contentDescription,
            modifier = Modifier.size(18.dp),
            tint = if (enabled) tint else colors.textSecondary.copy(alpha = 0.5f),
        )
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


/** 瞬间更多操作面板：作者 = 编辑文案/删除瞬间；非作者 = 举报瞬间（M3 ModalBottomSheet）。 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun MomentActionSheet(
    isOwner: Boolean,
    onDismiss: () -> Unit,
    onEdit: () -> Unit,
    onEditLocation: () -> Unit,
    onDelete: () -> Unit,
    onReport: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        containerColor = colors.surface,
        shape = AppShapes.sheet,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .navigationBarsPadding()
                .padding(start = Spacing.lg, end = Spacing.lg, bottom = Spacing.lg),
        ) {
            if (isOwner) {
                OwnerActionRow(text = "编辑文案", color = colors.textPrimary, onClick = onEdit)
                HorizontalDivider(color = colors.borderSubtle)
                OwnerActionRow(text = "修改位置", color = colors.textPrimary, onClick = onEditLocation)
                HorizontalDivider(color = colors.borderSubtle)
                OwnerActionRow(text = "删除瞬间", color = colors.danger, onClick = onDelete)
            } else {
                OwnerActionRow(text = "举报瞬间", color = colors.danger, onClick = onReport)
            }
        }
    }
}

@Composable
private fun OwnerActionRow(text: String, color: Color, onClick: () -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .heightIn(min = Dimens.minTouchTarget)
            .clickable(onClick = onClick)
            .padding(horizontal = Spacing.xs, vertical = Spacing.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.bodyLarge,
            fontWeight = FontWeight.Medium,
            color = color,
        )
    }
}

/** 编辑瞬间文案：只提交 description（spec moments/update Patch 语义，其余字段保持不变）。 */
@Composable
private fun MomentEditDialog(
    initialDescription: String,
    initialPrivacy: String?,
    initialTags: List<String>,
    maxTags: Int,
    saving: Boolean,
    error: String?,
    onDismiss: () -> Unit,
    onSubmit: (description: String, privacyLevel: String?, tags: List<String>) -> Unit,
) {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    var description by remember(initialDescription) { mutableStateOf(initialDescription) }
    var selectedTags by remember(initialTags) { mutableStateOf(initialTags) }
    var privacyIndex by remember(initialPrivacy) {
        mutableIntStateOf(if (initialPrivacy == "private") 1 else 0)
    }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("编辑瞬间") },
        text = {
            Column {
                SystemTextArea(
                    value = description,
                    onValueChange = { description = it },
                    label = "瞬间文案",
                    placeholder = "这一刻的想法…",
                    minHeight = Dimens.inputHeight * 3,
                )
                Spacer(Modifier.height(Spacing.md))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = "此刻氛围",
                        style = MaterialTheme.typography.titleSmall,
                        color = colors.textPrimary,
                    )
                    Spacer(Modifier.width(Spacing.sm))
                    Text(
                        text = selectedTags.size.toString() + "/" + maxTags,
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textSecondary,
                    )
                }
                Spacer(Modifier.height(Spacing.sm))
                MoodChips(
                    selected = selectedTags,
                    extraTags = initialTags,
                    onToggle = { tag ->
                        selectedTags = when {
                            selectedTags.contains(tag) -> selectedTags - tag
                            selectedTags.size >= maxTags -> {
                                Toast.makeText(context, "最多选择 " + maxTags + " 个标签", Toast.LENGTH_SHORT).show()
                                selectedTags
                            }
                            else -> selectedTags + tag
                        }
                    },
                )
                Spacer(Modifier.height(Spacing.md))
                Text(
                    text = "谁可以看到",
                    style = MaterialTheme.typography.titleSmall,
                    color = colors.textPrimary,
                )
                Spacer(Modifier.height(Spacing.sm))
                if (initialPrivacy != null) {
                    SegmentedTabs(
                        options = listOf("附近可见", "仅自己"),
                        selectedIndex = privacyIndex,
                        onSelected = { privacyIndex = it },
                        modifier = Modifier.fillMaxWidth(),
                    )
                } else {
                    Text(
                        text = "暂时无法读取可见范围，本次只修改文案",
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.textSecondary,
                    )
                }
                if (error != null) {
                    Spacer(Modifier.height(Spacing.sm))
                    Text(
                        text = error,
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.danger,
                    )
                }
            }
        },
        confirmButton = {
            SystemButton(
                text = if (saving) "保存中…" else "保存",
                onClick = {
                    if (!saving) {
                        onSubmit(
                            description.trim(),
                            initialPrivacy?.let { if (privacyIndex == 1) "private" else "public" },
                            selectedTags,
                        )
                    }
                },
                variant = ButtonVariant.PRIMARY,
                size = ButtonSize.SM,
                enabled = !saving,
            )
        },
        dismissButton = {
            SystemButton(
                text = "取消",
                onClick = onDismiss,
                variant = ButtonVariant.SECONDARY,
                size = ButtonSize.SM,
            )
        },
    )
}

