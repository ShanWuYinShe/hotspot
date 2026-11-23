package com.hotspot.app.data.repo

import com.hotspot.app.data.api.ApiClient
import com.hotspot.app.data.api.ApiEndpoint
import com.hotspot.app.data.api.ApiError
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.auth.AuthSession
import com.hotspot.app.data.auth.TokenStore
import com.hotspot.app.data.model.AccountData
import com.hotspot.app.data.model.AuthTokenPair
import com.hotspot.app.data.model.BlockedUserItem
import com.hotspot.app.data.model.ClientConfig
import com.hotspot.app.data.model.CommentItem
import com.hotspot.app.data.model.DefaultLocation
import com.hotspot.app.data.model.EnterGroupResult
import com.hotspot.app.data.model.GeoPoint
import com.hotspot.app.data.model.GroupAnnouncementItem
import com.hotspot.app.data.model.GroupApplicationItem
import com.hotspot.app.data.model.GroupAnnouncementList
import com.hotspot.app.data.model.GroupItem
import com.hotspot.app.data.model.AdminBannedMemberItem
import com.hotspot.app.data.model.GroupInvitation
import com.hotspot.app.data.model.GroupMemberItem
import com.hotspot.app.data.model.GroupSettings
import com.hotspot.app.data.model.GroupMessageItem
import com.hotspot.app.data.model.MapBounds
import com.hotspot.app.data.model.MapCanvasResponse
import com.hotspot.app.data.model.MessageSentResult
import com.hotspot.app.data.model.CompletedUploads
import com.hotspot.app.data.model.MomentItem
import com.hotspot.app.data.model.NotifyItem
import com.hotspot.app.data.model.Page
import com.hotspot.app.data.model.PresignFileInput
import com.hotspot.app.data.model.PresignedUploads
import com.hotspot.app.data.model.PrivateChatItem
import com.hotspot.app.data.model.PrivateMessageItem
import com.hotspot.app.data.model.RelatedUserItem
import com.hotspot.app.data.model.ReportCreated
import com.hotspot.app.data.model.ReportItem
import com.hotspot.app.data.model.UnreadSummary
import com.hotspot.app.data.model.UserDoorResult
import com.hotspot.app.data.model.UserItem
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody
import okio.BufferedSink
import java.io.InputStream
import java.time.Duration
import java.util.UUID

/** 分页参数（spec/api_spec_types.md PaginationParams：limit 默认 20 / cursor 不透明透传）。 */
internal fun pagination(cursor: String?, limit: Int = 20): JsonObject = buildJsonObject {
    put("limit", limit)
    if (cursor != null) put("cursor", cursor)
}

internal fun emptyObject(): JsonObject = buildJsonObject { }

/** 认证：登录 / 注册（用户名型，免验证码）/ 登出。字段严格按 spec/api_spec_public.md。 */
class AuthRepository(
    private val api: ApiClient,
    private val tokens: TokenStore,
) {
    suspend fun signin(loginId: String, password: String): ApiResult<AuthSession> {
        val body = buildJsonObject {
            put("login_id", loginId)
            put("password", password)
            put("client_type", "app")
            put("device_id", tokens.deviceId())
        }
        return toSession(api.post(ApiEndpoint.SIGNIN, body, AuthTokenPair.serializer()))
    }

    suspend fun register(loginId: String, nickname: String, password: String): ApiResult<AuthSession> {
        val body = buildJsonObject {
            put("login_id", loginId)
            // 用户名型 login_id 必须为 null（同传验证码 = 参数冲突）；手机号/邮箱型在验证码服务接入前不可用。
            put("verification_code", JsonNull)
            put("nickname", nickname)
            put("password", password)
            put("client_type", "app")
            put("device_id", tokens.deviceId())
        }
        return toSession(
            api.post(
                ApiEndpoint.REGISTER,
                body,
                AuthTokenPair.serializer(),
                idempotencyKey = UUID.randomUUID().toString(),
            ),
        )
    }

    /**
     * 访客模式（spec api_spec_public.md → POST /public/v1/auth/guest，MUST 幂等）：
     * 每次调用创建独立访客账号（user_type=temp）；平台开关关闭时返回 10400
     * （message「当前未开放访客模式」），由 ApiClient.businessErrorMessage 透传服务端 message。
     */
    suspend fun guest(nickname: String? = null): ApiResult<AuthSession> {
        val body = buildJsonObject {
            put("nickname", nickname)
            put("client_type", "app")
            put("device_id", tokens.deviceId())
        }
        return toSession(
            api.post(
                ApiEndpoint.GUEST,
                body,
                AuthTokenPair.serializer(),
                idempotencyKey = UUID.randomUUID().toString(),
            ),
        )
    }

    /**
     * 刷新 Token（spec/api_spec_public.md → POST /public/v1/auth/refresh_token）：
     * App 端必须带非空 refresh_token + device_id；成功后会轮换 access/refresh 并写回 TokenStore。
     * 用于 WebSocket close 4001 后的自动恢复；刷新失败返回 10100 时调用方应清会话回登录页。
     */
    suspend fun refresh(): ApiResult<AuthSession> {
        val refreshToken = tokens.session.value?.refreshToken
        if (refreshToken.isNullOrBlank()) {
            return ApiResult.Err(ApiError.Network("登录状态已失效，请重新登录"))
        }
        val body = buildJsonObject {
            put("client_type", "app")
            put("device_id", tokens.deviceId())
            put("refresh_token", refreshToken)
        }
        return toSession(api.post(ApiEndpoint.REFRESH_TOKEN, body, AuthTokenPair.serializer()))
    }

    suspend fun signout(): ApiResult<Unit> {
        val result = api.postVoid(ApiEndpoint.SIGNOUT)
        tokens.clear()
        return result
    }

    private fun toSession(result: ApiResult<AuthTokenPair>): ApiResult<AuthSession> = when (result) {
        is ApiResult.Ok -> {
            val pair = result.data
            val userId = pair.userId
            val accessToken = pair.accessToken
            val refreshToken = pair.refreshToken
            if (userId.isNullOrBlank() || accessToken.isNullOrBlank() || refreshToken.isNullOrBlank()) {
                ApiResult.Err(ApiError.Network("登录响应缺少凭证字段"))
            } else {
                tokens.save(userId, accessToken, refreshToken)
                ApiResult.Ok(AuthSession(userId, accessToken, refreshToken))
            }
        }
        is ApiResult.Err -> result
    }
}

/**
 * 首页信息流：
 * - 为你推荐 = moments/search 热门模式（spec：无输入条件的热门瞬间，承接发现页首屏）
 * - 正在关注 = moments/fetch_following
 */
class FeedRepository(private val api: ApiClient) {
    suspend fun recommended(cursor: String?): ApiResult<Page<MomentItem>> = api.post(
        ApiEndpoint.MOMENTS_SEARCH,
        buildJsonObject {
            put("keyword", JsonNull)
            put("tag", JsonNull)
            put("sort_by", "popular")
            put("pagination", pagination(cursor))
        },
        Page.serializer(MomentItem.serializer()),
    )

    /** 瞬间详情（登录视角 MomentItem，含 is_favorited 查看者状态）。 */
    suspend fun openMoment(momentId: String): ApiResult<MomentItem> = api.post(
        ApiEndpoint.MOMENTS_OPEN,
        buildJsonObject { put("moment_id", momentId) },
        MomentItem.serializer(),
    )

    /**
     * 编辑瞬间（仅作者，spec/api_spec_moment.md → moments/update）。
     * Patch 语义：只提交 description，其余字段不传保持不变；返回更新后的 MomentItem。
     */
    /**
     * 编辑瞬间（仅作者，patch 语义）：只下发非 null 字段，未传字段保持不变。
     * description / privacyLevel 分别对应文案与可见范围（public/private）。
     */
    suspend fun updateMoment(
        momentId: String,
        description: String? = null,
        privacyLevel: String? = null,
        tags: List<String>? = null,
    ): ApiResult<MomentItem> = api.post(
        ApiEndpoint.MOMENTS_UPDATE,
        buildJsonObject {
            put("moment_id", momentId)
            if (description != null) put("description", description)
            if (privacyLevel != null) put("privacy_level", privacyLevel)
            // tags 传空数组表示清空（patch 语义：null 保持不变）
            if (tags != null) put("tags", buildJsonArray { tags.forEach { add(JsonPrimitive(it)) } })
        },
        MomentItem.serializer(),
    )

    /** 删除瞬间（仅作者，软删除；spec/api_spec_moment.md → moments/delete）。 */
    suspend fun deleteMoment(momentId: String): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.MOMENTS_DELETE,
        buildJsonObject { put("moment_id", momentId) },
    )

    /** 收藏 / 取消收藏（spec/api_spec_moment.md → moments/set_favorite，两态统一端点）。 */
    suspend fun setFavorite(momentId: String, active: Boolean): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.MOMENTS_SET_FAVORITE,
        buildJsonObject {
            put("moment_id", momentId)
            put("active", active)
        },
    )

    suspend fun following(cursor: String?): ApiResult<Page<MomentItem>> = api.post(
        ApiEndpoint.MOMENTS_FETCH_FOLLOWING,
        buildJsonObject {
            put("reference_point", JsonNull)
            put("pagination", pagination(cursor))
        },
        Page.serializer(MomentItem.serializer()),
    )
}

/** 附近枢纽：附近的人 / 附近群组 / 附近瞬间（距离排序由后端计算，客户端只展示）。 */
/**
 * 瞬间写操作（spec/api_spec_moment.md → moments/create）：发布器唯一入口。
 * location 必填；Android 无地图/设备定位，调用方传 ClientConfig 默认位置（与附近/发现同源）。
 */
class MomentsRepository(private val api: ApiClient) {
    suspend fun create(
        description: String,
        location: DefaultLocation,
        tags: List<String>,
        privacyLevel: String,
        imageFileIds: List<String>,
    ): ApiResult<MomentItem> = api.post(
        ApiEndpoint.MOMENTS_CREATE,
        buildJsonObject {
            put("description", description)
            put(
                "location",
                buildJsonObject {
                    put(
                        "point",
                        buildJsonObject {
                            put("latitude", location.point.latitude)
                            put("longitude", location.point.longitude)
                        },
                    )
                    put("location_name", location.locationName)
                },
            )
            put("privacy_level", privacyLevel)
            put("tags", buildJsonArray { tags.forEach { add(JsonPrimitive(it)) } })
            put("image_file_ids", buildJsonArray { imageFileIds.forEach { add(JsonPrimitive(it)) } })
            put("mentions", buildJsonArray { })
        },
        MomentItem.serializer(),
        idempotencyKey = UUID.randomUUID().toString(),
    )
}

/** 媒体上传三步链路（spec/api_spec_media.md）：presign_upload → PUT 直传 → complete_upload。 */
class MediaRepository(private val api: ApiClient) {
    private val uploadClient = OkHttpClient.Builder()
        .connectTimeout(Duration.ofSeconds(20))
        .readTimeout(Duration.ofSeconds(60))
        .writeTimeout(Duration.ofSeconds(120))
        .callTimeout(Duration.ofSeconds(180))
        .build()

    suspend fun presign(files: List<PresignFileInput>): ApiResult<PresignedUploads> = api.post(
        ApiEndpoint.MEDIA_PRESIGN_UPLOAD,
        buildJsonObject {
            put(
                "files",
                buildJsonArray {
                    files.forEach { file ->
                        add(
                            buildJsonObject {
                                put("file_name", file.fileName)
                                put("file_size", file.fileSize)
                                put("mime_type", file.mimeType)
                                put("category", file.category)
                                put("md5", file.md5)
                            },
                        )
                    }
                },
            )
        },
        PresignedUploads.serializer(),
    )

    suspend fun complete(uploadIds: List<String>): ApiResult<CompletedUploads> = api.post(
        ApiEndpoint.MEDIA_COMPLETE_UPLOAD,
        buildJsonObject {
            put("upload_ids", buildJsonArray { uploadIds.forEach { add(JsonPrimitive(it)) } })
        },
        CompletedUploads.serializer(),
    )

    /**
     * PUT 直传对象存储（预签名 URL + 响应下发的 headers，不附加 Authorization）。
     * 返回 Result：失败由调用方转成用户提示；onProgress 为 0..1 单文件进度。
     */
    suspend fun putObject(
        uploadUrl: String,
        headers: Map<String, String>,
        bytes: ByteArray,
        mimeType: String,
        onProgress: ((Float) -> Unit)? = null,
    ): Result<Unit> = withContext(Dispatchers.IO) {
        runCatching {
            val mediaType = (mimeType.takeIf { it.isNotBlank() } ?: "application/octet-stream").toMediaType()
            val requestBuilder = Request.Builder()
                .url(uploadUrl)
                .put(ProgressRequestBody(bytes, mediaType, onProgress))
            headers.forEach { (name, value) -> requestBuilder.header(name, value) }
            uploadClient.newCall(requestBuilder.build()).execute().use { response ->
                check(response.isSuccessful) { "上传失败（HTTP " + response.code + "）" }
            }
        }
    }

    /**
     * PUT 直传对象存储（流式，不把文件整份读入内存）；供 chat_file 等大文件使用。
     * contentLength 必须与 presign 申报的 file_size 一致，openStream 在写入时打开一次。
     */
    suspend fun putObjectStream(
        uploadUrl: String,
        headers: Map<String, String>,
        contentLength: Long,
        mimeType: String,
        onProgress: ((Float) -> Unit)? = null,
        openStream: () -> InputStream,
    ): Result<Unit> = withContext(Dispatchers.IO) {
        runCatching {
            val mediaType = (mimeType.takeIf { it.isNotBlank() } ?: "application/octet-stream").toMediaType()
            val requestBuilder = Request.Builder()
                .url(uploadUrl)
                .put(StreamRequestBody(contentLength, mediaType, onProgress, openStream))
            headers.forEach { (name, value) -> requestBuilder.header(name, value) }
            uploadClient.newCall(requestBuilder.build()).execute().use { response ->
                check(response.isSuccessful) { "上传失败（HTTP " + response.code + "）" }
            }
        }
    }
}

/** 带进度回调的请求体：分块写入，进度只用于 UI 提示。 */
private class ProgressRequestBody(
    private val bytes: ByteArray,
    private val mediaType: okhttp3.MediaType,
    private val onProgress: ((Float) -> Unit)?,
) : RequestBody() {
    override fun contentType(): okhttp3.MediaType = mediaType

    override fun contentLength(): Long = bytes.size.toLong()

    override fun writeTo(sink: BufferedSink) {
        val chunkSize = 16 * 1024
        var written = 0
        while (written < bytes.size) {
            val length = minOf(chunkSize, bytes.size - written)
            sink.write(bytes, written, length)
            written += length
            onProgress?.invoke(written.toFloat() / bytes.size.toFloat())
        }
    }
}

/** 流式请求体：从 InputStream 分块写入，支持 100MB 级聊天文件且不占用整份内存。 */
private class StreamRequestBody(
    private val length: Long,
    private val mediaType: okhttp3.MediaType,
    private val onProgress: ((Float) -> Unit)?,
    private val openStream: () -> InputStream,
) : RequestBody() {
    override fun contentType(): okhttp3.MediaType = mediaType

    override fun contentLength(): Long = length

    override fun writeTo(sink: BufferedSink) {
        val buffer = ByteArray(64 * 1024)
        var written = 0L
        openStream().use { input ->
            while (true) {
                val read = input.read(buffer)
                if (read < 0) break
                sink.write(buffer, 0, read)
                written += read
                if (length > 0) onProgress?.invoke((written.toFloat() / length.toFloat()).coerceIn(0f, 1f))
            }
        }
    }
}

class NearbyRepository(private val api: ApiClient) {
    /**
     * 附近的人（spec/api_spec_discovery.md：发现即 users/search + radius + sort_by=distance）。
     * tag 为兴趣标签筛选（全部传 null）；sortBy 支持 distance / newest。
     */
    suspend fun nearbyPeople(
        cursor: String?,
        tag: String? = null,
        sortBy: String = "distance",
        referencePoint: GeoPoint? = null,
    ): ApiResult<Page<UserItem>> = api.post(
        ApiEndpoint.USERS_SEARCH,
        buildJsonObject {
            put("keyword", JsonNull)
            put("tag", tag)
            put("radius", 5000)
            put("sort_by", sortBy)
            put("reference_point", referencePointJson(referencePoint))
            put("pagination", pagination(cursor))
        },
        Page.serializer(UserItem.serializer()),
    )

    /** 附近群组（groups/search + radius + sort_by=distance；tag 筛选同理）。 */
    suspend fun nearbyGroups(
        cursor: String?,
        tag: String? = null,
        sortBy: String = "distance",
        referencePoint: GeoPoint? = null,
    ): ApiResult<Page<GroupItem>> = api.post(
        ApiEndpoint.GROUPS_SEARCH,
        buildJsonObject {
            put("keyword", JsonNull)
            put("tag", tag)
            put("radius", 5000)
            put("sort_by", sortBy)
            put("reference_point", referencePointJson(referencePoint))
            put("pagination", pagination(cursor))
        },
        Page.serializer(GroupItem.serializer()),
    )

    suspend fun nearbyMoments(
        cursor: String?,
        referencePoint: GeoPoint? = null,
    ): ApiResult<Page<MomentItem>> = api.post(
        ApiEndpoint.MOMENTS_FETCH_NEARBY,
        buildJsonObject {
            put("radius", 5000)
            put("reference_point", referencePointJson(referencePoint))
            put("pagination", pagination(cursor))
        },
        Page.serializer(MomentItem.serializer()),
    )

    /**
     * 地图视口渲染（spec/api_spec_public.md → map/canvas）：服务端按视口跨度/密度
     * 返回 heatmap 或 hotspots（互斥）或 empty；dataType = 'user'|'group'|'moment'|null(全部)。
     */
    suspend fun mapCanvas(
        bounds: MapBounds,
        dataType: String? = null,
        cursor: String? = null,
    ): ApiResult<MapCanvasResponse> = api.post(
        ApiEndpoint.PUBLIC_MAP_CANVAS,
        buildJsonObject {
            put(
                "viewport",
                buildJsonObject {
                    put(
                        "bounds",
                        buildJsonObject {
                            put("north", bounds.north)
                            put("south", bounds.south)
                            put("east", bounds.east)
                            put("west", bounds.west)
                        },
                    )
                },
            )
            put(
                "filter",
                if (dataType == null) {
                    JsonNull
                } else {
                    buildJsonObject {
                        put("data_type", dataType)
                        put("time_range", JsonNull)
                    }
                },
            )
            put("pagination", pagination(cursor, 100))
        },
        MapCanvasResponse.serializer(),
    )
}


/** 附近类请求的 reference_point：锚点缺失时显式传 null（服务端回退链接口见 spec）。 */
private fun referencePointJson(point: GeoPoint?): JsonElement = if (point == null) {
    JsonNull
} else {
    buildJsonObject {
        put("latitude", point.latitude)
        put("longitude", point.longitude)
    }
}

/** 全局搜索：用户 / 瞬间 / 群组各自独立分页（spec/api_spec_discovery.md：不提供聚合端点）。 */
class SearchRepository(private val api: ApiClient) {
    suspend fun searchUsers(keyword: String, cursor: String?): ApiResult<Page<UserItem>> = api.post(
        ApiEndpoint.USERS_SEARCH,
        buildJsonObject {
            put("keyword", keyword)
            put("tag", JsonNull)
            put("radius", JsonNull)
            put("sort_by", "relevance")
            put("reference_point", JsonNull)
            put("pagination", pagination(cursor))
        },
        Page.serializer(UserItem.serializer()),
    )

    suspend fun searchMoments(keyword: String, cursor: String?): ApiResult<Page<MomentItem>> = api.post(
        ApiEndpoint.MOMENTS_SEARCH,
        buildJsonObject {
            put("keyword", keyword)
            put("tag", JsonNull)
            put("sort_by", "relevance")
            put("pagination", pagination(cursor))
        },
        Page.serializer(MomentItem.serializer()),
    )

    suspend fun searchGroups(keyword: String, cursor: String?): ApiResult<Page<GroupItem>> = api.post(
        ApiEndpoint.GROUPS_SEARCH,
        buildJsonObject {
            put("keyword", keyword)
            put("tag", JsonNull)
            put("radius", JsonNull)
            put("sort_by", "relevance")
            put("reference_point", JsonNull)
            put("pagination", pagination(cursor))
        },
        Page.serializer(GroupItem.serializer()),
    )
}

/** 消息：私聊会话列表 / 历史消息 / 发送文本消息。 */
class ChatRepository(private val api: ApiClient) {
    suspend fun conversations(cursor: String?): ApiResult<Page<PrivateChatItem>> = api.post(
        ApiEndpoint.PRIVATE_CHATS_FETCH,
        buildJsonObject { put("pagination", pagination(cursor)) },
        Page.serializer(PrivateChatItem.serializer()),
    )

    suspend fun messages(targetUserId: String, cursor: String?): ApiResult<Page<PrivateMessageItem>> = api.post(
        ApiEndpoint.PRIVATE_MESSAGES_FETCH,
        buildJsonObject {
            put("target_user_id", targetUserId)
            put("pagination", pagination(cursor))
        },
        Page.serializer(PrivateMessageItem.serializer()),
    )

    suspend fun sendTextMessage(targetUserId: String, content: String): ApiResult<MessageSentResult> = api.post(
        ApiEndpoint.PRIVATE_MESSAGES_SEND,
        buildJsonObject {
            put("target_user_id", targetUserId)
            put("type", "text")
            put("content", content)
            put("file_id", JsonNull)
            put("share_code", JsonNull)
            put("reply_to_message_id", JsonNull)
        },
        MessageSentResult.serializer(),
        idempotencyKey = UUID.randomUUID().toString(),
    )

    /** 发送私聊图片消息（file_id 来自 media/complete_upload，category=chat_image）。 */
    suspend fun sendImageMessage(targetUserId: String, fileId: String): ApiResult<MessageSentResult> = api.post(
        ApiEndpoint.PRIVATE_MESSAGES_SEND,
        buildJsonObject {
            put("target_user_id", targetUserId)
            put("type", "image")
            put("content", JsonNull)
            put("file_id", fileId)
            put("share_code", JsonNull)
            put("reply_to_message_id", JsonNull)
        },
        MessageSentResult.serializer(),
        idempotencyKey = UUID.randomUUID().toString(),
    )

    /** 发送私聊语音消息（file_id 来自 media/complete_upload，category=chat_audio）。 */
    suspend fun sendAudioMessage(targetUserId: String, fileId: String): ApiResult<MessageSentResult> = api.post(
        ApiEndpoint.PRIVATE_MESSAGES_SEND,
        buildJsonObject {
            put("target_user_id", targetUserId)
            put("type", "audio")
            put("content", JsonNull)
            put("file_id", fileId)
            put("share_code", JsonNull)
            put("reply_to_message_id", JsonNull)
        },
        MessageSentResult.serializer(),
        idempotencyKey = UUID.randomUUID().toString(),
    )

    /** 发送私聊视频消息（file_id 来自 media/complete_upload，category=chat_video）。 */
    suspend fun sendVideoMessage(targetUserId: String, fileId: String): ApiResult<MessageSentResult> = api.post(
        ApiEndpoint.PRIVATE_MESSAGES_SEND,
        buildJsonObject {
            put("target_user_id", targetUserId)
            put("type", "video")
            put("content", JsonNull)
            put("file_id", fileId)
            put("share_code", JsonNull)
            put("reply_to_message_id", JsonNull)
        },
        MessageSentResult.serializer(),
        idempotencyKey = UUID.randomUUID().toString(),
    )

    /** 发送私聊文件消息（file_id 来自 media/complete_upload，category=chat_file）。 */
    suspend fun sendFileMessage(targetUserId: String, fileId: String): ApiResult<MessageSentResult> = api.post(
        ApiEndpoint.PRIVATE_MESSAGES_SEND,
        buildJsonObject {
            put("target_user_id", targetUserId)
            put("type", "file")
            put("content", JsonNull)
            put("file_id", fileId)
            put("share_code", JsonNull)
            put("reply_to_message_id", JsonNull)
        },
        MessageSentResult.serializer(),
        idempotencyKey = UUID.randomUUID().toString(),
    )

    /** 撤回私聊消息（仅作者；时间窗由服务端 AppConfig 控制，超时 10400）。 */
    suspend fun recallPrivateMessage(messageId: String): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.PRIVATE_MESSAGES_RECALL,
        buildJsonObject { put("message_id", messageId) },
    )

    /** 撤回自己发送的群聊消息（仅作者；管理员撤回走 groups/admin/messages/recall）。 */
    suspend fun recallGroupMessage(messageId: String): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.GROUP_MESSAGES_RECALL,
        buildJsonObject { put("message_id", messageId) },
    )

    /** 私聊已读（spec/api_spec_chat.md → private_messages/mark_read：scope='session' 标记该会话全部已读，幂等）。 */
    suspend fun markPrivateRead(targetUserId: String): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.PRIVATE_MESSAGES_MARK_READ,
        buildJsonObject {
            put("scope", "session")
            put("target_user_id", targetUserId)
        },
    )

    // ── 群聊（spec/api_spec_chat.md → group_messages；历史 + 文本/图片发送）──

    suspend fun groupMessages(groupId: String, cursor: String?): ApiResult<Page<GroupMessageItem>> = api.post(
        ApiEndpoint.GROUP_MESSAGES_FETCH,
        buildJsonObject {
            put("group_id", groupId)
            put("pagination", pagination(cursor))
        },
        Page.serializer(GroupMessageItem.serializer()),
    )

    /** 群聊已读（spec/api_spec_chat.md → group_messages/mark_read：scope='group' 标记该群全部已读，幂等）。 */
    suspend fun markGroupRead(groupId: String): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.GROUP_MESSAGES_MARK_READ,
        buildJsonObject {
            put("scope", "group")
            put("group_id", groupId)
        },
    )

    /**
     * 发送群文本消息；mentions 为 @ 的 user_id 列表（spec/api_spec_chat.md：
     * 服务端对每个被 @ 的群成员推送 GroupMessageMentioned 并落库通知，非成员 ID 忽略）。
     */
    suspend fun sendGroupTextMessage(
        groupId: String,
        content: String,
        mentions: List<String> = emptyList(),
    ): ApiResult<MessageSentResult> = api.post(
        ApiEndpoint.GROUP_MESSAGES_SEND,
        buildJsonObject {
            put("group_id", groupId)
            put("type", "text")
            put("content", content)
            put("file_id", JsonNull)
            put("share_code", JsonNull)
            put("reply_to_message_id", JsonNull)
            put("mentions", buildJsonArray { mentions.distinct().forEach { add(JsonPrimitive(it)) } })
        },
        MessageSentResult.serializer(),
        idempotencyKey = UUID.randomUUID().toString(),
    )

    /** 发送群图片消息（file_id 来自 media/complete_upload，category=chat_image）。 */
    suspend fun sendGroupImageMessage(groupId: String, fileId: String): ApiResult<MessageSentResult> = api.post(
        ApiEndpoint.GROUP_MESSAGES_SEND,
        buildJsonObject {
            put("group_id", groupId)
            put("type", "image")
            put("content", JsonNull)
            put("file_id", fileId)
            put("share_code", JsonNull)
            put("reply_to_message_id", JsonNull)
            put("mentions", buildJsonArray { })
        },
        MessageSentResult.serializer(),
        idempotencyKey = UUID.randomUUID().toString(),
    )

    /** 发送群语音消息（file_id 来自 media/complete_upload，category=chat_audio）。 */
    suspend fun sendGroupAudioMessage(groupId: String, fileId: String): ApiResult<MessageSentResult> = api.post(
        ApiEndpoint.GROUP_MESSAGES_SEND,
        buildJsonObject {
            put("group_id", groupId)
            put("type", "audio")
            put("content", JsonNull)
            put("file_id", fileId)
            put("share_code", JsonNull)
            put("reply_to_message_id", JsonNull)
            put("mentions", buildJsonArray { })
        },
        MessageSentResult.serializer(),
        idempotencyKey = UUID.randomUUID().toString(),
    )

    /** 发送群视频消息（file_id 来自 media/complete_upload，category=chat_video）。 */
    suspend fun sendGroupVideoMessage(groupId: String, fileId: String): ApiResult<MessageSentResult> = api.post(
        ApiEndpoint.GROUP_MESSAGES_SEND,
        buildJsonObject {
            put("group_id", groupId)
            put("type", "video")
            put("content", JsonNull)
            put("file_id", fileId)
            put("share_code", JsonNull)
            put("reply_to_message_id", JsonNull)
            put("mentions", buildJsonArray { })
        },
        MessageSentResult.serializer(),
        idempotencyKey = UUID.randomUUID().toString(),
    )

    /** 发送群文件消息（file_id 来自 media/complete_upload，category=chat_file）。 */
    suspend fun sendGroupFileMessage(groupId: String, fileId: String): ApiResult<MessageSentResult> = api.post(
        ApiEndpoint.GROUP_MESSAGES_SEND,
        buildJsonObject {
            put("group_id", groupId)
            put("type", "file")
            put("content", JsonNull)
            put("file_id", fileId)
            put("share_code", JsonNull)
            put("reply_to_message_id", JsonNull)
            put("mentions", buildJsonArray { })
        },
        MessageSentResult.serializer(),
        idempotencyKey = UUID.randomUUID().toString(),
    )
}

/**
 * 评论：顶层列表（首屏 limit 50，cursor 透传）/ 创建（MUST 幂等）/ 评论点赞。
 * 契约见 spec/api_spec_comment.md 与 spec/api_spec_reaction.md；回复归组在 UI 层按 parent_id 完成。
 */
class CommentRepository(private val api: ApiClient) {
    suspend fun fetchComments(
        momentId: String,
        cursor: String?,
        limit: Int = COMMENT_PAGE_SIZE,
    ): ApiResult<Page<CommentItem>> = api.post(
        ApiEndpoint.COMMENTS_FETCH,
        buildJsonObject {
            put("moment_id", momentId)
            put("pagination", pagination(cursor, limit))
        },
        Page.serializer(CommentItem.serializer()),
    )

    /**
     * 创建评论/回复（spec：客户端必须携带 Idempotency-Key）。
     * parent_id 指向顶层评论（若指向二级回复，服务端自动挂到所属顶层评论）；mentions 为被 @ 的 user_id。
     */
    suspend fun createComment(
        momentId: String,
        content: String,
        parentId: String?,
        mentions: List<String>,
        idempotencyKey: String,
    ): ApiResult<CommentItem> = api.post(
        ApiEndpoint.COMMENTS_CREATE,
        buildJsonObject {
            put("moment_id", momentId)
            put("content", content)
            put("parent_id", parentId)
            put("mentions", buildJsonArray { mentions.forEach { add(JsonPrimitive(it)) } })
        },
        CommentItem.serializer(),
        idempotencyKey = idempotencyKey,
    )

    /**
     * 删除自己的评论（spec/comments/delete：软删除，仅评论作者；顶层删除后子评论保留）。
     * 平台审核删除走 admin 接口，不在客户端暴露。
     */
    suspend fun deleteComment(commentId: String): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.COMMENTS_DELETE,
        buildJsonObject { put("comment_id", commentId) },
    )

    /** 楼中楼全量回复（spec/comments/fetch_replies：时间正序；「查看全部 N 条回复」使用）。 */
    suspend fun fetchReplies(
        commentId: String,
        cursor: String? = null,
        limit: Int = REPLIES_PAGE_SIZE,
    ): ApiResult<Page<CommentItem>> = api.post(
        ApiEndpoint.COMMENTS_FETCH_REPLIES,
        buildJsonObject {
            put("comment_id", commentId)
            put("pagination", pagination(cursor, limit))
        },
        Page.serializer(CommentItem.serializer()),
    )

    /** 评论点赞/取消（reactions/set target_type=comment；emoji 须在服务端白名单内）。 */
    suspend fun setCommentReaction(commentId: String, active: Boolean): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.REACTIONS_SET,
        buildJsonObject {
            put("target_type", "comment")
            put("target_id", commentId)
            put("emoji", LIKE_EMOJI)
            put("active", active)
        },
    )

    companion object {
        /** 首屏评论页大小（任务契约：limit 50）。 */
        const val COMMENT_PAGE_SIZE = 50

        /** 展开全部回复页大小（与 Web 同口径 limit 100，一次取完常见楼层）。 */
        const val REPLIES_PAGE_SIZE = 100

        /** 点赞 emoji：服务端白名单默认含 👍（与 Web / 鸿蒙同口径）。 */
        const val LIKE_EMOJI = "👍"
    }
}

/**
 * 个人中心：账户资料（含仅自己可见的 email/phone/stats）+ 「我的内容」集合。
 * 我的瞬间/收藏/群组均为成员视角端点（spec：fetch_created 含 private 与软删占位、
 * fetch_favorites 含收藏时间、fetch_joined 含 viewer_state/last_message），
 * 不属于公共发现域，故统一收敛在资料仓库。
 */
class ProfileRepository(private val api: ApiClient) {
    suspend fun account(): ApiResult<AccountData> =
        api.post(ApiEndpoint.USERS_ACCOUNT, emptyObject(), AccountData.serializer())

    /** 我创建的瞬间（拥有者视角，spec/api_spec_moment.md → moments/fetch_created）。 */
    suspend fun createdMoments(cursor: String?): ApiResult<Page<MomentItem>> = api.post(
        ApiEndpoint.MOMENTS_FETCH_CREATED,
        buildJsonObject { put("pagination", pagination(cursor)) },
        Page.serializer(MomentItem.serializer()),
    )

    /** 我的收藏（spec → moments/fetch_favorites）。 */
    suspend fun favoriteMoments(cursor: String?): ApiResult<Page<MomentItem>> = api.post(
        ApiEndpoint.MOMENTS_FETCH_FAVORITES,
        buildJsonObject { put("pagination", pagination(cursor)) },
        Page.serializer(MomentItem.serializer()),
    )

    /**
     * 更新账户资料（spec → users/update_account，patch 语义：只下发要改的字段）。
     * avatar_file_id 依赖媒体上传链路（首版 TODO），本方法不涉及头像。
     */
    suspend fun updateAccount(
        nickname: String? = null,
        bio: String? = null,
        tags: List<String>? = null,
        gender: String? = null,
    ): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.USERS_UPDATE_ACCOUNT,
        buildJsonObject {
            if (nickname != null) put("nickname", nickname)
            if (bio != null) put("bio", bio)
            if (tags != null) {
                put(
                    "tags",
                    buildJsonArray {
                        tags.forEach { add(JsonPrimitive(it)) }
                    },
                )
            }
            if (gender != null) put("gender", gender)
        },
    )

    /** 好友列表（好友 = 双方互相关注；user_id 必传，拉自己时传自己的 id）。 */
    suspend fun friends(userId: String, cursor: String?): ApiResult<Page<RelatedUserItem>> =
        relationList(ApiEndpoint.USERS_FETCH_FRIENDS, userId, cursor)

    /** 关注列表（我关注的人）。 */
    suspend fun following(userId: String, cursor: String?): ApiResult<Page<RelatedUserItem>> =
        relationList(ApiEndpoint.USERS_FETCH_FOLLOWING, userId, cursor)

    /** 粉丝列表（关注我的人）。 */
    suspend fun followers(userId: String, cursor: String?): ApiResult<Page<RelatedUserItem>> =
        relationList(ApiEndpoint.USERS_FETCH_FOLLOWERS, userId, cursor)

    private suspend fun relationList(
        endpoint: String,
        userId: String,
        cursor: String?,
    ): ApiResult<Page<RelatedUserItem>> = api.post(
        endpoint,
        buildJsonObject {
            put("user_id", userId)
            put("pagination", pagination(cursor))
        },
        Page.serializer(RelatedUserItem.serializer()),
    )

    /** 关注 / 取关（spec/api_spec_user.md → users/set_follow：active 两态统一）。 */
    suspend fun setFollow(userId: String, active: Boolean): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.USERS_SET_FOLLOW,
        buildJsonObject {
            put("user_id", userId)
            put("active", active)
            put("source", JsonNull)
            put("source_moment_id", JsonNull)
        },
    )

    /** 我加入的群组（成员视角，spec/api_spec_group.md → groups/fetch_joined）。 */
    suspend fun joinedGroups(cursor: String?): ApiResult<Page<GroupItem>> = api.post(
        ApiEndpoint.GROUPS_FETCH_JOINED,
        buildJsonObject { put("pagination", pagination(cursor)) },
        Page.serializer(GroupItem.serializer()),
    )

    /** 他人主页资料（spec → users/open_the_door；10300 = 不存在或隐私隐藏）。 */
    suspend fun openUserDoor(userId: String, includeRecentMoments: Boolean = true): ApiResult<UserDoorResult> = api.post(
        ApiEndpoint.USERS_OPEN_THE_DOOR,
        buildJsonObject {
            put("user_id", userId)
            put("include_recent_moments", includeRecentMoments)
        },
        UserDoorResult.serializer(),
    )

    /** 某用户的瞬间历史（公开视角，翻页接在 recent_moments 之后）。 */
    suspend fun userMoments(userId: String, cursor: String?): ApiResult<Page<MomentItem>> = api.post(
        ApiEndpoint.USERS_FETCH_MOMENTS,
        buildJsonObject {
            put("user_id", userId)
            put("pagination", pagination(cursor))
        },
        Page.serializer(MomentItem.serializer()),
    )

    /** 设置好友备注（仅对已关注对象有效，spec → users/set_remark）。 */
    suspend fun setRemark(userId: String, remark: String): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.USERS_SET_REMARK,
        buildJsonObject {
            put("user_id", userId)
            put("remark", remark)
        },
    )

    /** 拉黑 / 解除拉黑（spec → users/set_block 两态统一）。 */
    suspend fun setBlock(userId: String, blocked: Boolean): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.USERS_SET_BLOCK,
        buildJsonObject {
            put("user_id", userId)
            // spec BlockRequest 字段为 active（历史实现误写 blocked，导致拉黑请求被服务端按默认 false 处理）
            put("active", blocked)
            put("reason", JsonNull)
        },
    )

    /** 我的拉黑列表（spec users/fetch_blocked，按 blocked_at 倒序）。 */
    suspend fun fetchBlocked(cursor: String?): ApiResult<Page<BlockedUserItem>> = api.post(
        ApiEndpoint.USERS_FETCH_BLOCKED,
        buildJsonObject { put("pagination", pagination(cursor)) },
        Page.serializer(BlockedUserItem.serializer()),
    )
}

/**
 * 群组详情域：详情 / 成员 / 公告 / 加入 / 退出。
 * 契约见 spec/api_spec_group.md（open_the_door、members/fetch、enter、exit）与
 * spec/api_spec_public.md（announcements/fetch：仅 public 版、请求只有 group_id、响应无分页）。
 * 距离/位置由后端计算，客户端只展示；加入/退出结果以服务端 status/错误码为准。
 */
class GroupRepository(private val api: ApiClient) {
    /** 群组详情（登录视角 GroupItem，含 viewer_state：null = 非成员）。 */
    suspend fun open(groupId: String): ApiResult<GroupItem> = api.post(
        ApiEndpoint.GROUPS_OPEN_THE_DOOR,
        buildJsonObject { put("group_id", groupId) },
        GroupItem.serializer(),
    )

    /**
     * 加入群组（智能路由：direct/password + requires_approval 由服务端判定）。
     * password 群需先收集口令；message 为审批群可选附言（本页不做输入，传 null）。
     * 错误码 10200/10300/10400 由 ApiClient 透传服务端 message。
     */
    suspend fun enter(
        groupId: String,
        password: String? = null,
        message: String? = null,
    ): ApiResult<EnterGroupResult> = api.post(
        ApiEndpoint.GROUPS_ENTER,
        buildJsonObject {
            put("group_id", groupId)
            put("password", password)
            put("message", message)
        },
        EnterGroupResult.serializer(),
    )

    /** 退出群组（群主调用会被服务端 10400 拒绝，客户端只透传 message）。 */
    suspend fun exit(groupId: String): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.GROUPS_EXIT,
        buildJsonObject { put("group_id", groupId) },
    )

    /** 成员列表（成员视角公开字段；按 joined_at 正序，客户端只做分页透传）。 */
    suspend fun members(
        groupId: String,
        cursor: String?,
        limit: Int = GROUP_MEMBERS_PAGE_SIZE,
    ): ApiResult<Page<GroupMemberItem>> = api.post(
        ApiEndpoint.GROUPS_MEMBERS_FETCH,
        buildJsonObject {
            put("group_id", groupId)
            put("role", JsonNull)
            put("pagination", pagination(cursor, limit))
        },
        Page.serializer(GroupMemberItem.serializer()),
    )

    /** 群公告列表（单次全量返回，无 cursor；置顶优先由服务端排序）。 */
    /** 封禁/解封成员（owner/admin；active=true 会移出群组并禁止再加入；群主保护 10400）。 */
    suspend fun setBan(
        groupId: String,
        userId: String,
        active: Boolean,
        reason: String? = null,
    ): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.GROUPS_ADMIN_MEMBERS_SET_BAN,
        buildJsonObject {
            put("group_id", groupId)
            put("user_id", userId)
            put("active", active)
            put("reason", reason)
        },
    )

    /** 封禁列表（owner/admin；普通成员 10200）。 */
    suspend fun bannedMembers(groupId: String): ApiResult<List<AdminBannedMemberItem>> {
        val result = api.post(
            ApiEndpoint.GROUPS_ADMIN_MEMBERS_FETCH_BANNED,
            buildJsonObject {
                put("group_id", groupId)
                put("pagination", pagination(null, 50))
            },
            Page.serializer(AdminBannedMemberItem.serializer()),
        )
        return when (result) {
            is ApiResult.Ok -> ApiResult.Ok(result.data.items)
            is ApiResult.Err -> result
        }
    }

    /** 邀请成员（owner/admin；被邀请者已是成员时返回 no-op 成功，inviteId 为 null）。 */
    suspend fun createInvite(groupId: String, userId: String, message: String? = null): ApiResult<GroupInvitation> = api.post(
        ApiEndpoint.GROUPS_ADMIN_INVITES_CREATE,
        buildJsonObject {
            put("group_id", groupId)
            put("user_id", userId)
            put("message", message)
        },
        GroupInvitation.serializer(),
    )

    /** 入群申请列表（群收件箱，默认 pending；owner/admin 守卫）。 */
    suspend fun applications(groupId: String, status: String = "pending"): ApiResult<List<GroupApplicationItem>> {
        val result = api.post(
            ApiEndpoint.GROUPS_ADMIN_APPLICATIONS_FETCH,
            buildJsonObject {
                put("group_id", groupId)
                put("status", status)
                put("pagination", pagination(null, 50))
            },
            Page.serializer(GroupApplicationItem.serializer()),
        )
        return when (result) {
            is ApiResult.Ok -> ApiResult.Ok(result.data.items)
            is ApiResult.Err -> result
        }
    }

    /** 同意入群申请（owner/admin）。 */
    suspend fun approveApplication(groupId: String, applicationId: String): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.GROUPS_ADMIN_APPLICATIONS_APPROVE,
        buildJsonObject {
            put("group_id", groupId)
            put("application_id", applicationId)
        },
    )

    /** 拒绝入群申请（reason 0-200 字符，可空）。 */
    suspend fun rejectApplication(groupId: String, applicationId: String, reason: String? = null): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.GROUPS_ADMIN_APPLICATIONS_REJECT,
        buildJsonObject {
            put("group_id", groupId)
            put("application_id", applicationId)
            put("reason", reason)
        },
    )

    /** 群设置（管理视角 GroupSettings，owner/admin 守卫）。 */
    suspend fun settings(groupId: String): ApiResult<GroupSettings> = api.post(
        ApiEndpoint.GROUPS_ADMIN_SETTINGS,
        buildJsonObject { put("group_id", groupId) },
        GroupSettings.serializer(),
    )

    /** 更新群设置（只提交本轮可编辑的四个字段，其余字段 null 不改）。 */
    suspend fun updateSettings(
        groupId: String,
        isSearchable: Boolean,
        membersSearchable: Boolean,
        requiresApproval: Boolean,
        maxMembers: Int,
        joinMethod: String? = null,
        password: String? = null,
        clearPassword: Boolean = false,
    ): ApiResult<Unit> {
        val result = api.post(
            ApiEndpoint.GROUPS_ADMIN_UPDATE,
            buildJsonObject {
                put("group_id", groupId)
                put("name", JsonNull)
                put("description", JsonNull)
                put("avatar_file_id", JsonNull)
                put("tags", JsonNull)
                // password 与 clear_password 互斥：有密码时只传 password，清除时只传 clear_password。
                put("password", password?.let(::JsonPrimitive) ?: JsonNull)
                put("clear_password", if (clearPassword) JsonPrimitive(true) else JsonNull)
                put("join_method", joinMethod?.let(::JsonPrimitive) ?: JsonNull)
                put("requires_approval", requiresApproval)
                put("next_activity_at", JsonNull)
                put("is_searchable", isSearchable)
                put("members_searchable", membersSearchable)
                put("max_members", maxMembers)
                put("location", JsonNull)
            },
            GroupItem.serializer(),
        )
        return when (result) {
            is ApiResult.Ok -> ApiResult.Ok(Unit)
            is ApiResult.Err -> result
        }
    }

    /** 转让群主（owner 专属；成功后原 owner 失去 owner 权限，自然去重）。 */
    suspend fun transferOwnership(groupId: String, newOwnerId: String): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.GROUPS_ADMIN_TRANSFER_OWNERSHIP,
        buildJsonObject {
            put("group_id", groupId)
            put("new_owner_id", newOwnerId)
        },
    )

    /** 设置成员角色（owner/admin 守卫；role 为 admin/member；服务端保护最后一个管理员）。 */
    suspend fun setMemberRole(groupId: String, userId: String, role: String): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.GROUPS_ADMIN_MEMBERS_SET_ROLE,
        buildJsonObject {
            put("group_id", groupId)
            put("user_id", userId)
            put("role", role)
        },
    )

    /** 设置/解除成员禁言（active=true 时 duration_minutes 必填，1..43200；群主保护由服务端返回 10400）。 */
    suspend fun setMemberMute(
        groupId: String,
        userId: String,
        active: Boolean,
        durationMinutes: Int? = null,
    ): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.GROUPS_ADMIN_MEMBERS_SET_MUTE,
        buildJsonObject {
            put("group_id", groupId)
            put("user_id", userId)
            put("active", active)
            put("duration_minutes", durationMinutes)
        },
    )

    /** 移除成员（owner/admin 守卫；群主保护由服务端返回 10400）。 */
    suspend fun removeMember(groupId: String, userId: String): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.GROUPS_ADMIN_MEMBERS_REMOVE,
        buildJsonObject {
            put("group_id", groupId)
            put("user_id", userId)
        },
    )

    /** 更新群名称/简介（spec/api_spec_group.md → groups/admin/update，owner/admin 守卫；其余字段传 null 不改）。 */
    suspend fun updateGroupInfo(groupId: String, name: String, description: String?): ApiResult<GroupItem> = api.post(
        ApiEndpoint.GROUPS_ADMIN_UPDATE,
        buildJsonObject {
            put("group_id", groupId)
            put("name", name)
            put("description", description)
            put("avatar_file_id", JsonNull)
            put("tags", JsonNull)
            put("password", JsonNull)
            put("clear_password", JsonNull)
            put("join_method", JsonNull)
            put("requires_approval", JsonNull)
            put("next_activity_at", JsonNull)
            put("is_searchable", JsonNull)
            put("members_searchable", JsonNull)
            put("max_members", JsonNull)
            put("location", JsonNull)
        },
        GroupItem.serializer(),
    )

    /** 发布/更新/删除群公告（spec/api_spec_group.md → groups/admin/announcements 的 create/update/delete，owner/admin 守卫）。 */
    suspend fun createAnnouncement(
        groupId: String,
        title: String,
        content: String,
        isPinned: Boolean,
    ): ApiResult<GroupAnnouncementItem> = api.post(
        ApiEndpoint.GROUPS_ADMIN_ANNOUNCEMENTS_CREATE,
        buildJsonObject {
            put("group_id", groupId)
            put("title", title)
            put("content", content)
            put("is_pinned", isPinned)
        },
        GroupAnnouncementItem.serializer(),
        idempotencyKey = UUID.randomUUID().toString(),
    )

    suspend fun updateAnnouncement(
        groupId: String,
        announcementId: String,
        title: String,
        content: String,
        isPinned: Boolean,
    ): ApiResult<GroupAnnouncementItem> = api.post(
        ApiEndpoint.GROUPS_ADMIN_ANNOUNCEMENTS_UPDATE,
        buildJsonObject {
            put("group_id", groupId)
            put("announcement_id", announcementId)
            put("title", title)
            put("content", content)
            put("is_pinned", isPinned)
        },
        GroupAnnouncementItem.serializer(),
    )

    suspend fun deleteAnnouncement(groupId: String, announcementId: String): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.GROUPS_ADMIN_ANNOUNCEMENTS_DELETE,
        buildJsonObject {
            put("group_id", groupId)
            put("announcement_id", announcementId)
        },
    )

    suspend fun announcements(groupId: String): ApiResult<List<GroupAnnouncementItem>> {
        val result = api.post(
            ApiEndpoint.GROUPS_ANNOUNCEMENTS_FETCH,
            buildJsonObject { put("group_id", groupId) },
            GroupAnnouncementList.serializer(),
        )
        return when (result) {
            is ApiResult.Ok -> ApiResult.Ok(result.data.items)
            is ApiResult.Err -> result
        }
    }

    companion object {
        /** 成员列表首屏页大小（任务契约：首屏 30）。 */
        const val GROUP_MEMBERS_PAGE_SIZE = 30
    }
}

/** 举报：我的举报记录（spec/api_spec_report.md → reports/fetch，created_at DESC）。 */
class ReportsRepository(private val api: ApiClient) {
    suspend fun fetch(
        status: String?,
        targetType: String? = null,
        cursor: String?,
    ): ApiResult<Page<ReportItem>> = api.post(
        ApiEndpoint.REPORTS_FETCH,
        buildJsonObject {
            put("status", status)
            put("target_type", targetType)
            put("pagination", pagination(cursor))
        },
        Page.serializer(ReportItem.serializer()),
    )

    /** 创建举报（spec/api_spec_report.md → reports/create；重复 pending 返回 is_duplicate=true 视为成功）。 */
    suspend fun create(
        targetType: String,
        targetId: String,
        reason: String,
        description: String? = null,
    ): ApiResult<ReportCreated> = api.post(
        ApiEndpoint.REPORTS_CREATE,
        buildJsonObject {
            put("target_type", targetType)
            put("target_id", targetId)
            put("reason", reason)
            put("description", description)
        },
        ReportCreated.serializer(),
    )
}

/**
 * 系统公开配置：只取 defaults.default_location 作为附近/地图等发现类请求的浏览锚点
 * （spec/api_spec_types.md「reference_point 解析规则」第 3 级）。
 */
class SystemRepository(private val api: ApiClient) {
    suspend fun clientConfig(): ApiResult<ClientConfig> =
        api.post(ApiEndpoint.SYSTEM_CLIENT_CONFIG, emptyObject(), ClientConfig.serializer())
}

/** 通知中心：列表 / 未读摘要 / 全部已读。 */
class NotifyRepository(private val api: ApiClient) {
    suspend fun fetch(cursor: String?): ApiResult<Page<NotifyItem>> = api.post(
        ApiEndpoint.NOTIFY_FETCH,
        buildJsonObject {
            put("type", JsonNull)
            put("is_read", JsonNull)
            put("pagination", pagination(cursor))
        },
        Page.serializer(NotifyItem.serializer()),
    )

    suspend fun unreadSummary(): ApiResult<UnreadSummary> =
        api.post(ApiEndpoint.NOTIFY_UNREAD_SUMMARY, emptyObject(), UnreadSummary.serializer())

    /** 把单条通知标记为已读（点击通知跳转时调用，spec notify/mark_read scope=ids）。 */
    suspend fun markRead(notifyId: String): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.NOTIFY_MARK_READ,
        buildJsonObject {
            put("scope", "ids")
            put("notify_ids", buildJsonArray { add(JsonPrimitive(notifyId)) })
        },
    )

    suspend fun markAllRead(): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.NOTIFY_MARK_READ,
        buildJsonObject {
            put("scope", "all")
            put("notify_ids", JsonNull)
        },
    )
}
