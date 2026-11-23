package com.hotspot.app.data.repo

import com.hotspot.app.data.api.ApiClient
import com.hotspot.app.data.api.ApiEndpoint
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.model.AdminPermissions
import com.hotspot.app.data.model.AdminAnnouncementItem
import com.hotspot.app.data.model.AdminGroupItem
import com.hotspot.app.data.model.AdminReportsPage
import com.hotspot.app.data.model.AdminUserItem
import com.hotspot.app.data.model.Page
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put

/**
 * 平台管理端仓库（/admin/api/v1）：当前仅接入 RBAC 身份与举报审核。
 * 权限由后端 RBAC 校验（非管理员返回 10200），客户端只做入口显隐。
 */
class AdminRepository(private val api: ApiClient) {
    suspend fun me(): ApiResult<AdminPermissions> = api.post(
        ApiEndpoint.ADMIN_RBAC_ME,
        ApiClient.emptyBody(),
        AdminPermissions.serializer(),
    )

    /** 举报列表（按状态过滤；status=null 表示全部）。 */
    suspend fun fetchReports(status: String?, cursor: String?): ApiResult<AdminReportsPage> = api.post(
        ApiEndpoint.ADMIN_REPORTS_FETCH,
        buildJsonObject {
            put("status", status?.let(::JsonPrimitive) ?: JsonNull)
            put("target_type", JsonNull)
            put("pagination", pagination(cursor))
        },
        AdminReportsPage.serializer(),
    )

    /** 搜索用户（平台管理）：nickname 模糊匹配；status 可选 normal/banned。 */
    suspend fun searchUsers(
        nickname: String?,
        status: String? = null,
        cursor: String? = null,
    ): ApiResult<Page<AdminUserItem>> = api.post(
        ApiEndpoint.ADMIN_USERS_SEARCH,
        buildJsonObject {
            put("user_id", JsonNull)
            put("nickname", nickname?.takeIf { it.isNotBlank() }?.let(::JsonPrimitive) ?: JsonNull)
            put("login_id", JsonNull)
            put("status", status?.let(::JsonPrimitive) ?: JsonNull)
            put("user_type", JsonNull)
            put("pagination", pagination(cursor))
        },
        Page.serializer(AdminUserItem.serializer()),
    )

    /** 系统公告列表（平台管理视角，含创建人/推送状态/阅读数）。 */
    suspend fun fetchAnnouncements(
        severity: String?,
        cursor: String? = null,
    ): ApiResult<Page<AdminAnnouncementItem>> = api.post(
        ApiEndpoint.ADMIN_ANNOUNCEMENTS_FETCH,
        buildJsonObject {
            put("severity", severity?.let(::JsonPrimitive) ?: JsonNull)
            put("pagination", pagination(cursor))
        },
        Page.serializer(AdminAnnouncementItem.serializer()),
    )

    /** 发布系统公告（sendPush 仅创建时可选）。 */
    suspend fun createAnnouncement(
        title: String,
        content: String,
        severity: String,
        actionUrl: String?,
        sendPush: Boolean,
    ): ApiResult<AdminAnnouncementItem> = api.post(
        ApiEndpoint.ADMIN_ANNOUNCEMENTS_CREATE,
        buildJsonObject {
            put("title", title)
            put("content", content)
            put("severity", severity)
            put("action_url", actionUrl?.takeIf { it.isNotBlank() }?.let(::JsonPrimitive) ?: JsonNull)
            put("send_push", sendPush)
        },
        AdminAnnouncementItem.serializer(),
        idempotencyKey = java.util.UUID.randomUUID().toString(),
    )

    /** 更新系统公告：只提交变化字段（action_url 传空串表示清除）。 */
    suspend fun updateAnnouncement(
        announcementId: String,
        title: String?,
        content: String?,
        severity: String?,
        actionUrl: String?,
        clearActionUrl: Boolean = false,
    ): ApiResult<AdminAnnouncementItem> = api.post(
        ApiEndpoint.ADMIN_ANNOUNCEMENTS_UPDATE,
        buildJsonObject {
            put("announcement_id", announcementId)
            put("title", title?.let(::JsonPrimitive) ?: JsonNull)
            put("content", content?.let(::JsonPrimitive) ?: JsonNull)
            put("severity", severity?.let(::JsonPrimitive) ?: JsonNull)
            put("action_url", if (clearActionUrl) JsonPrimitive("") else actionUrl?.let(::JsonPrimitive) ?: JsonNull)
        },
        AdminAnnouncementItem.serializer(),
    )

    /** 删除系统公告。 */
    suspend fun deleteAnnouncement(announcementId: String): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.ADMIN_ANNOUNCEMENTS_DELETE,
        buildJsonObject { put("announcement_id", announcementId) },
    )

    /** 搜索群组（平台管理）：名称模糊匹配。 */
    suspend fun searchGroups(
        name: String?,
        cursor: String? = null,
    ): ApiResult<Page<AdminGroupItem>> = api.post(
        ApiEndpoint.ADMIN_GROUPS_SEARCH,
        buildJsonObject {
            put("group_id", JsonNull)
            put("name", name?.takeIf { it.isNotBlank() }?.let(::JsonPrimitive) ?: JsonNull)
            put("owner_id", JsonNull)
            put("pagination", pagination(cursor))
        },
        Page.serializer(AdminGroupItem.serializer()),
    )

    /** 解散群组（平台管理）：reason 仅进审计日志，不推送给群成员。 */
    suspend fun dismissGroup(groupId: String, reason: String?): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.ADMIN_GROUPS_DISMISS,
        buildJsonObject {
            put("group_id", groupId)
            put("reason", reason?.takeIf { it.isNotBlank() }?.let(::JsonPrimitive) ?: JsonNull)
        },
    )

    /** 平台封禁/解封用户（两态统一）：durationDays null/0 = 永久。 */
    suspend fun setUserBan(
        userId: String,
        active: Boolean,
        reason: String?,
        durationDays: Int?,
    ): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.ADMIN_USERS_SET_BAN,
        buildJsonObject {
            put("user_id", userId)
            put("active", active)
            put("reason", reason?.takeIf { it.isNotBlank() }?.let(::JsonPrimitive) ?: JsonNull)
            put("duration_days", if (active) durationDays?.let(::JsonPrimitive) ?: JsonNull else JsonNull)
        },
    )

    /** 裁决举报：resolve=有效 / dismiss=驳回；banDurationDays 仅 resolve 时可选（0=永久）。 */
    suspend fun judge(
        reportId: String,
        action: String,
        note: String?,
        banDurationDays: Int? = null,
    ): ApiResult<Unit> = api.postVoid(
        ApiEndpoint.ADMIN_REPORTS_JUDGE,
        buildJsonObject {
            put("report_id", reportId)
            put("action", action)
            put("note", note?.takeIf { it.isNotBlank() }?.let(::JsonPrimitive) ?: JsonNull)
            put("ban_duration_days", banDurationDays?.let(::JsonPrimitive) ?: JsonNull)
        },
    )
}
