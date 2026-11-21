package com.hotspot.app.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull

/**
 * API DTO：字段名严格对齐 spec/（snake_case），用 @SerialName 映射到 Kotlin camelCase。
 * 所有集合/可空字段给默认值，服务端增删字段由 Json { ignoreUnknownKeys = true } 兜底。
 */

/** 通用分页容器（spec/api_spec_types.md：items + next_cursor，next_cursor=null 表示到底）。 */
@Serializable
data class Page<T>(
    val items: List<T> = emptyList(),
    @SerialName("next_cursor") val nextCursor: String? = null,
)

/** 请求侧位置（spec → Location：原始坐标 + 名称；client_config.defaults.default_location）。 */
@Serializable
data class DefaultLocation(
    val point: GeoPoint = GeoPoint(),
    @SerialName("location_name") val locationName: String = "",
)

@Serializable
data class ClientDefaults(
    @SerialName("default_location") val defaultLocation: DefaultLocation? = null,
)

/** 业务限额（spec/api_spec_types.md → AppLimits；只取发布器用到的最小集合）。 */
@Serializable
data class AppLimits(
    @SerialName("max_moment_tags") val maxMomentTags: Int = 9,
    @SerialName("max_moment_images") val maxMomentImages: Int = 9,
    @SerialName("moment_description_max_length") val momentDescriptionMaxLength: Int = 280,
)

/** 客户端公开配置（spec/api_spec_public.md → POST /public/v1/system/client_config，仅取用途所需字段）。 */
@Serializable
data class ClientConfig(
    val defaults: ClientDefaults = ClientDefaults(),
    val limits: AppLimits = AppLimits(),
)

// ===== 媒体上传（spec/api_spec_media.md）=====

@Serializable
data class PresignFileInput(
    @SerialName("file_name") val fileName: String,
    @SerialName("file_size") val fileSize: Long,
    @SerialName("mime_type") val mimeType: String,
    val category: String = "moment_image",
    val md5: String? = null,
)

@Serializable
data class PresignedUpload(
    @SerialName("upload_id") val uploadId: String = "",
    @SerialName("upload_url") val uploadUrl: String = "",
    val headers: Map<String, String> = emptyMap(),
    @SerialName("expires_at") val expiresAt: String? = null,
)

@Serializable
data class PresignedUploads(val uploads: List<PresignedUpload> = emptyList())

@Serializable
data class UploadedMedia(
    @SerialName("file_id") val fileId: String = "",
    @SerialName("thumbnail_url") val thumbnailUrl: String? = null,
    @SerialName("file_size") val fileSize: Long = 0,
    @SerialName("mime_type") val mimeType: String = "",
)

@Serializable
data class CompletedUploads(val uploads: List<UploadedMedia> = emptyList())

@Serializable
data class UserInfo(
    @SerialName("user_id") val userId: String = "",
    val nickname: String = "",
    @SerialName("avatar_url") val avatarUrl: String? = null,
)

@Serializable
data class UserStats(
    @SerialName("moments_count") val momentsCount: Int = 0,
    @SerialName("friends_count") val friendsCount: Int = 0,
    @SerialName("followers_count") val followersCount: Int = 0,
    @SerialName("following_count") val followingCount: Int = 0,
    @SerialName("groups_count") val groupsCount: Int = 0,
)

@Serializable
data class Relationship(
    @SerialName("is_following") val isFollowing: Boolean = false,
    @SerialName("is_followed_by") val isFollowedBy: Boolean = false,
    @SerialName("is_blocked") val isBlocked: Boolean = false,
)

@Serializable
data class GeoPoint(
    val latitude: Double = 0.0,
    val longitude: Double = 0.0,
)

@Serializable
data class RelativeLocation(
    val point: GeoPoint? = null,
    @SerialName("location_name") val locationName: String? = null,
    val distance: Double? = null,
    @SerialName("bearing_degrees") val bearingDegrees: Double? = null,
)

/**
 * 关系列表项（spec/api_spec_user.md → RelatedUserItem；好友/关注/粉丝三接口共用形状）。
 * followed_back_at 仅 fetch_friends 返回；remark 仅查自己时返回。
 */
/** 举报处理结果摘要（spec → ReportResolution；脱敏，不含处理者身份）。 */
@Serializable
data class ReportResolution(
    val action: String = "",
    @SerialName("resolved_at") val resolvedAt: String? = null,
    val note: String? = null,
)

/** 我的举报记录（spec → ReportItem；只取客户端展示所需字段，snapshot 不展示）。 */
@Serializable
data class ReportItem(
    @SerialName("report_id") val reportId: String = "",
    @SerialName("target_type") val targetType: String = "",
    @SerialName("target_id") val targetId: String = "",
    val reason: String = "",
    val description: String? = null,
    val status: String = "",
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("handled_at") val handledAt: String? = null,
    val resolution: ReportResolution? = null,
)

/** reports/create 响应（spec → ReportCreated）。 */
@Serializable
data class ReportCreated(
    @SerialName("report_id") val reportId: String = "",
    @SerialName("is_duplicate") val isDuplicate: Boolean = false,
)

@Serializable
data class RelatedUserItem(
    val user: UserItem = UserItem(),
    @SerialName("followed_at") val followedAt: String = "",
    @SerialName("followed_back_at") val followedBackAt: String? = null,
    val remark: String? = null,
)

@Serializable
data class UserItem(
    val user: UserInfo = UserInfo(),
    val bio: String? = null,
    val tags: List<String> = emptyList(),
    @SerialName("presence_status") val presenceStatus: String? = null,
    @SerialName("last_online_at") val lastOnlineAt: String? = null,
    val location: RelativeLocation? = null,
    val stats: UserStats? = null,
    val relation: Relationship? = null,
)

/** users/open_the_door 响应（spec → UserProfileResult）：资料 + 可选主页首屏最近瞬间。 */
@Serializable
data class UserDoorResult(
    val user: UserItem = UserItem(),
    @SerialName("recent_moments") val recentMoments: Page<MomentItem>? = null,
)

@Serializable
data class MediaAsset(
    val url: String = "",
    @SerialName("thumbnail_url") val thumbnailUrl: String? = null,
    val width: Int? = null,
    val height: Int? = null,
    @SerialName("mime_type") val mimeType: String? = null,
)

@Serializable
data class Reaction(
    val emoji: String = "",
    val count: Int = 0,
    @SerialName("reacted_by_me") val reactedByMe: Boolean = false,
)

@Serializable
data class MomentSummary(
    @SerialName("moment_id") val momentId: String = "",
    @SerialName("description_preview") val descriptionPreview: String = "",
    @SerialName("cover_url") val coverUrl: String? = null,
)

@Serializable
data class MomentItem(
    @SerialName("moment_id") val momentId: String = "",
    val description: String = "",
    val images: List<MediaAsset> = emptyList(),
    val tags: List<String> = emptyList(),
    val mentions: List<String> = emptyList(),
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("comments_count") val commentsCount: Int = 0,
    @SerialName("shares_count") val sharesCount: Int = 0,
    val reactions: List<Reaction> = emptyList(),
    val user: UserInfo = UserInfo(),
    @SerialName("is_deleted") val isDeleted: Boolean = false,
    val location: RelativeLocation? = null,
    @SerialName("is_favorited") val isFavorited: Boolean = false,
    // 拥有者视角扩展（spec api_spec_moment.md → OwnerMomentItem）：仅 moments/fetch_created、
    // moments/create、moments/update 下发，其它端点恒缺省。取值 'public' / 'private'。
    @SerialName("privacy_level") val privacyLevel: String? = null,
    // 收藏视角扩展（spec → FavoritedMomentItem）：仅 moments/fetch_favorites 下发（UTC ISO 8601）。
    @SerialName("favorited_at") val favoritedAt: String? = null,
) {
    /** 点赞数：Reaction 聚合里 emoji 点赞（👍 / ❤ 等）的总和；无反应时为 0。 */
    val likeCount: Int get() = reactions.sumOf { it.count }
}

/**
 * 评论（spec/api_spec_types.md CommentItem）：两级模型（顶层评论 + 一级回复）。
 * fetch 顶层列表时 top_replies 携带前 N 条回复预览（默认 3），其它端点为空数组。
 * reactions / top_replies 服务端可能下发 null，依赖 Json { coerceInputValues = true } 回落到默认空表。
 */
@Serializable
data class CommentItem(
    @SerialName("comment_id") val commentId: String = "",
    val user: UserInfo = UserInfo(),
    val content: String = "",
    @SerialName("parent_id") val parentId: String? = null,
    val mentions: List<String> = emptyList(),
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("replies_count") val repliesCount: Int = 0,
    val reactions: List<Reaction> = emptyList(),
    @SerialName("is_deleted") val isDeleted: Boolean = false,
    @SerialName("top_replies") val topReplies: List<CommentItem> = emptyList(),
) {
    /** 点赞数（reactions[].count 求和，与 Web / 鸿蒙同口径）。 */
    val likeCount: Int get() = reactions.sumOf { it.count }

    /** 我是否已点赞（reactions[].reacted_by_me 任一为 true）。 */
    val liked: Boolean get() = reactions.any { it.reactedByMe }
}

@Serializable
data class GroupInfo(
    @SerialName("group_id") val groupId: String = "",
    val name: String = "",
    @SerialName("avatar_url") val avatarUrl: String? = null,
    @SerialName("members_count") val membersCount: Int = 0,
)

/**
 * 群组查看者状态（spec api_spec_types.md → GroupViewerState；GroupItem.viewer_state）。
 * 非成员为 null；groups/fetch_joined 恒非 null。
 */
@Serializable
data class GroupViewerState(
    val role: String = "member",
    @SerialName("unread_count") val unreadCount: Int = 0,
    @SerialName("is_muted") val isMuted: Boolean = false,
    @SerialName("muted_until") val mutedUntil: String? = null,
    @SerialName("is_pinned") val isPinned: Boolean = false,
    @SerialName("joined_at") val joinedAt: String = "",
)

/**
 * 群组统一公开类型（spec api_spec_types.md → GroupItem）：形状恒定，含成员视角扩展
 * viewer_state 与列表补充字段；groups/fetch_joined 另返回 last_message（GroupJoinedItem）。
 */
@Serializable
data class GroupItem(
    val group: GroupInfo = GroupInfo(),
    val description: String? = null,
    val tags: List<String> = emptyList(),
    @SerialName("join_method") val joinMethod: String? = null,
    @SerialName("requires_approval") val requiresApproval: Boolean = false,
    val location: RelativeLocation? = null,
    @SerialName("last_activity_at") val lastActivityAt: String? = null,
    @SerialName("online_members_count") val onlineMembersCount: Int? = null,
    @SerialName("next_activity_at") val nextActivityAt: String? = null,
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("owner_id") val ownerId: String? = null,
    @SerialName("viewer_state") val viewerState: GroupViewerState? = null,
    @SerialName("friends_in_group") val friendsInGroup: Int? = null,
    // 仅 groups/fetch_joined 下发（spec → GroupJoinedItem）：最近一条消息预览，null = 无消息。
    @SerialName("last_message") val lastMessage: MessageBrief? = null,
)

/**
 * 群成员列表项（spec/api_spec_group.md → GroupMemberItem，members/fetch 与 members/search 共用）。
 * presence_status / last_online_at 为隐私受控恒定 nullable（null = 成员关闭在线可见）。
 */
@Serializable
data class GroupMemberItem(
    val user: UserInfo = UserInfo(),
    val role: String = "member",
    @SerialName("joined_at") val joinedAt: String = "",
    @SerialName("presence_status") val presenceStatus: String? = null,
    @SerialName("last_online_at") val lastOnlineAt: String? = null,
)

/** 群公告列表项（spec/api_spec_types.md → GroupAnnouncementItem）。 */
@Serializable
data class GroupAnnouncementItem(
    @SerialName("announcement_id") val announcementId: String = "",
    @SerialName("group_id") val groupId: String = "",
    val title: String = "",
    val content: String = "",
    @SerialName("created_by") val createdBy: UserInfo = UserInfo(),
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("is_pinned") val isPinned: Boolean = false,
)

/**
 * 群公告列表响应（spec/api_spec_public.md → GroupAnnouncementList）：只有 items，无分页容器
 * （契约请求仅 group_id，返回条数由服务端 group_announcements_count 配置控制）。
 */
@Serializable
data class GroupAnnouncementList(
    val items: List<GroupAnnouncementItem> = emptyList(),
)

/**
 * 加入群组响应（spec/api_spec_group.md → EnterGroupResult）。
 * status: 'joined'（已加入）/ 'pending_approval'（已提交申请，等待审核）。
 */
@Serializable
data class EnterGroupResult(
    val status: String = "",
    @SerialName("application_id") val applicationId: String? = null,
)

@Serializable
data class MessageBrief(
    @SerialName("message_id") val messageId: String = "",
    val type: String = "text",
    val sender: UserInfo = UserInfo(),
    val content: String? = null,
    @SerialName("share_code") val shareCode: String? = null,
    @SerialName("sent_at") val sentAt: String = "",
)

@Serializable
data class MessageMedia(
    val url: String = "",
    @SerialName("thumbnail_url") val thumbnailUrl: String? = null,
    @SerialName("file_name") val fileName: String? = null,
    @SerialName("file_size") val fileSize: Long? = null,
    val duration: Int? = null,
)

@Serializable
data class PrivateMessageItem(
    @SerialName("message_id") val messageId: String = "",
    val type: String = "text",
    val content: String? = null,
    val media: MessageMedia? = null,
    @SerialName("share_code") val shareCode: String? = null,
    val sender: UserInfo = UserInfo(),
    @SerialName("reply_to") val replyTo: MessageBrief? = null,
    @SerialName("sent_at") val sentAt: String = "",
    @SerialName("edited_at") val editedAt: String? = null,
    @SerialName("recalled_at") val recalledAt: String? = null,
    @SerialName("is_read") val isRead: Boolean = false,
    val reactions: List<Reaction> = emptyList(),
)

/**
 * 群聊消息项（spec/api_spec_chat.md → GroupMessageItem）。
 * 发送者统一为 sender: UserInfo；read_count 为已读人数（最终一致，null = 不适用）；
 * recalled_at 非 null 即已撤回；type=system 为群系统消息（居中展示）。
 */
@Serializable
data class GroupMessageItem(
    @SerialName("message_id") val messageId: String = "",
    val type: String = "text",
    val content: String? = null,
    val media: MessageMedia? = null,
    @SerialName("share_code") val shareCode: String? = null,
    val sender: UserInfo = UserInfo(),
    @SerialName("reply_to") val replyTo: MessageBrief? = null,
    val mentions: List<String> = emptyList(),
    @SerialName("recalled_by") val recalledBy: UserInfo? = null,
    @SerialName("sent_at") val sentAt: String = "",
    @SerialName("edited_at") val editedAt: String? = null,
    @SerialName("recalled_at") val recalledAt: String? = null,
    @SerialName("read_count") val readCount: Int? = null,
    val reactions: List<Reaction> = emptyList(),
)

@Serializable
data class PrivateChatItem(
    @SerialName("other_user") val otherUser: UserInfo = UserInfo(),
    @SerialName("unread_count") val unreadCount: Int = 0,
    @SerialName("is_muted") val isMuted: Boolean = false,
    @SerialName("is_pinned") val isPinned: Boolean = false,
    @SerialName("last_activity_at") val lastActivityAt: String = "",
    @SerialName("last_message") val lastMessage: MessageBrief? = null,
    @SerialName("presence_status") val presenceStatus: String? = null,
    @SerialName("can_message") val canMessage: Boolean = true,
)

@Serializable
data class MessageSentResult(
    @SerialName("message_id") val messageId: String = "",
    @SerialName("sent_at") val sentAt: String = "",
)

@Serializable
data class NotifyItem(
    @SerialName("notify_id") val notifyId: String = "",
    val type: String = "",
    @SerialName("is_read") val isRead: Boolean = false,
    @SerialName("created_at") val createdAt: String = "",
    val payload: JsonElement = JsonNull,
)

@Serializable
data class UnreadSummary(
    val total: Int = 0,
)

@Serializable
data class AccountData(
    @SerialName("user_id") val userId: String = "",
    val nickname: String = "",
    @SerialName("avatar_url") val avatarUrl: String? = null,
    @SerialName("user_type") val userType: String = "normal",
    val bio: String? = null,
    val tags: List<String> = emptyList(),
    val gender: String? = null,
    val email: String? = null,
    val phone: String? = null,
    @SerialName("presence_status") val presenceStatus: String = "offline",
    @SerialName("last_online_at") val lastOnlineAt: String? = null,
    // 账户存储的原始位置（spec → Location：point + location_name，个人中心显示所在地）
    val location: DefaultLocation? = null,
    @SerialName("created_at") val createdAt: String = "",
    val stats: UserStats? = null,
)

/** App 端 AuthTokenPair（spec/api_public.md：client_type='app' 时响应体携带 token）。 */
@Serializable
data class AuthTokenPair(
    @SerialName("client_type") val clientType: String? = null,
    @SerialName("user_id") val userId: String? = null,
    @SerialName("access_token") val accessToken: String? = null,
    @SerialName("refresh_token") val refreshToken: String? = null,
    @SerialName("access_token_expires_at") val accessTokenExpiresAt: String? = null,
    @SerialName("refresh_token_expires_at") val refreshTokenExpiresAt: String? = null,
)
