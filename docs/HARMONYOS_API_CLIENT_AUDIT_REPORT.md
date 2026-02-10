# HarmonyOS API Client Audit Report

**Date:** 2025-02-10  
**Scope:** `ApiClient.ets`, `ApiEndpoints.ets` vs `spec/api_spec_*.mdc`  
**Source of Truth:** `spec/` directory

---

## Executive Summary

This audit compares every method in `ApiClient.ets` against the API specifications. Several issues were found: wrong response types, non-existent "delete" endpoints (correctly implemented as recall aliases), missing required request fields, and one typo in the endpoint constant name.

---

## 1. Critical Issues

### 1.1 `getGroupSettings` – Wrong Endpoint & Response Type

| Property | Value |
|----------|-------|
| **Method** | `getGroupSettings` |
| **Current endpoint** | `GROUP.OPEN_THE_DOOR` (`/api/v1/groups/open_the_door`) |
| **Current response type** | `ApiResponse<GroupSettingsData>` |
| **Spec** | No dedicated `groups/settings` endpoint |

**Why:** The spec only defines `POST /api/v1/groups/open_the_door`, which returns `ApiResponse<GroupOpenResult>` (`{ group: GroupItem; recent_messages: PaginatedData<GroupMessageItem> }`). There is no standalone group settings endpoint.

**Fix:** Either:
- Remove `getGroupSettings` and have callers use `getGroupDetail` and derive settings from `group`, or  
- Keep the method but change the return type to `ApiResponse<GroupOpenResult>` and have callers read `response.data.group` for settings.

---

### 1.2 `getRelationship` / `getUserStats` – Endpoint Reuse

| Property | Value |
|----------|-------|
| **Methods** | `getRelationship`, `getUserStats` |
| **Current endpoint** | `USER.OPEN_THE_DOOR` (`/api/v1/users/open_the_door`) |
| **Spec** | Only `UserOpenTheDoorRequest { user_id }` |

**Why:** The spec defines a single endpoint `users/open_the_door` that returns `UserItem` (including `relation` and `stats` when authenticated). There are no separate relationship or stats endpoints. The implementation reusing `open_the_door` is correct; the concern is only whether request types align.

**Request type:** Spec has `UserOpenTheDoorRequest { user_id: string }`.  
Client uses `GetRelationshipRequest { user_id: number }` and `UserStatsRequest { user_id: number }`. Both include `user_id`, so functionally OK if the backend accepts numeric IDs.

---

### 1.3 `sendPrivateMessage` / `sendGroupMessage` – Wrong Response Type

| Property | Value |
|----------|-------|
| **Methods** | `sendPrivateMessage`, `sendGroupMessage` |
| **Current response type** | `ApiResponse<PrivateMessageItem>` / `ApiResponse<GroupMessageItem>` |
| **Spec response type** | `ApiResponse<MessageSent>` |

**Spec:** Both send endpoints return `ApiResponse<MessageSent>` where:
```typescript
interface MessageSent {
  message_id: string;
  sent_at: string;
}
```

**Fix:** Update return types to `ApiResponse<MessageSent>` (or a local equivalent). The full message is delivered via WebSocket, so the client should not rely on receiving a full `PrivateMessageItem` / `GroupMessageItem` from the send APIs.

---

### 1.4 `getSessions` – Wrong Response Shape

| Property | Value |
|----------|-------|
| **Method** | `getSessions` |
| **Current response type** | `ApiResponse<PaginatedData<SessionItem>>` |
| **Spec response type** | `ApiResponse<{ items: SessionItem[] }>` |

**Spec:** "活跃会话数天然有限...无需分页" – sessions are returned as `{ items: SessionItem[] }`, not paginated.

**Fix:** Change return type to `ApiResponse<{ items: SessionItem[] }>`.

---

### 1.5 `getTrendingTags` – Wrong Response Shape

| Property | Value |
|----------|-------|
| **Method** | `getTrendingTags` |
| **Current response type** | `ApiResponse<TrendingTagsData>` (≈ `{ items: TagItem[] }`) |
| **Spec response type** | `ApiResponse<TagItem[]>` |

**Spec:** ` tags/list_trending` returns `ApiResponse<TagItem[]>` (a direct array), not `{ items: TagItem[] }`.

**Fix:** Change return type to `ApiResponse<TagItem[]>` and adjust callers to use `response.data` directly instead of `response.data.items`.

---

### 1.6 `fetchGlobalSettings` – Request & Response Mismatch

| Property | Value |
|----------|-------|
| **Method** | `fetchGlobalSettings` |
| **Endpoint** | `SYSTEM.GET_CLIENT_CONFIG` |
| **Current request** | `FetchGlobalSettingsRequest` (context, current_version) |
| **Current response** | `ApiResponse<FetchSettingsData>` (action, config?, latest_version) |
| **Spec request** | `{}` |
| **Spec response** | `ApiResponse<ClientConfig>` |

**Spec:** `get_client_config` accepts an empty body and returns `ClientConfig` directly.

**Fix:** Align request with spec (empty or minimal) and return type with `ApiResponse<ClientConfig>`. If the backend actually supports version-checking semantics, spec and implementation should be updated together.

---

### 1.7 `healthCheck` – Minor Response Type Mismatch

| Property | Value |
|----------|-------|
| **Method** | `healthCheck` |
| **Current response type** | `ApiResponse<HealthCheckData>` (status, version, timestamp) |
| **Spec response type** | `ApiResponse<HealthStatus>` (status, timestamp) |

**Spec:** `HealthStatus` has `status` and `timestamp`; no `version`.

**Fix:** Use `ApiResponse<HealthStatus>` or a type compatible with `{ status, timestamp }`. If the backend adds `version`, it can remain as an optional extension.

---

## 2. deletePrivateMessage / deleteGroupMessage vs Recall

### 2.1 Design

| Method | Endpoint | Spec |
|--------|----------|------|
| `recallPrivateMessage` | `PRIVATE_MESSAGES.RECALL` | ✅ Exists |
| `deletePrivateMessage` | `PRIVATE_MESSAGES.RECALL` | Uses same endpoint |
| `recallGroupMessage` | `GROUP_MESSAGES.RECALL` | ✅ Exists |
| `deleteGroupMessage` | `GROUP_MESSAGES.RECALL` | Uses same endpoint |

**Conclusion:** The spec only defines "recall" endpoints. There is no separate "delete" endpoint. Both `delete*` and `recall*` methods correctly call the same `recall` endpoint, so they are valid aliases. The naming may confuse callers; consider deprecating `delete*` or documenting that they call recall.

---

## 3. ApiEndpoints Typo

### 3.1 `CHAT` vs `CHAT`

**Finding:** `ApiEndpoints.CHAT` is used in `ApiClient.ets`. In `ApiEndpoints.ets` the exported constant is `CHAT`.

**Status:** No typo. The correct name is `CHAT`; the reference matches.

---

## 4. Request Type Mismatches

### 4.1 Feed Endpoints – Missing `reference_point`

**Spec:** `GetFeedMomentsRequest`, `GetFeedGroupsRequest`, `GetFeedUsersRequest` require `reference_point: GeoPoint`.

**Client types:** `GetFeedMomentsRequest`, `GetFeedGroupsRequest`, `GetFeedUsersRequest` in `api.ets` do not include `reference_point`.

**Fix:** Add `reference_point: GeoPoint` to all feed request types.

---

### 4.2 `suggestTags` – Wrong Request Type

| Property | Value |
|----------|-------|
| **Method** | `suggestTags` |
| **Endpoint** | `SEARCH.SUGGEST` |
| **Current request** | `SuggestTagsRequest` (prefix, limit?) |
| **Spec request** | `SearchSuggestRequest` (prefix, reference_point, types?) |

**Spec:** `search/suggest` requires `reference_point: GeoPoint` and optionally `types?: SearchSuggestType[]`.

**Fix:** Use a request type compatible with `SearchSuggestRequest` (including `reference_point`) for `suggestTags`. The tag-only use case can be expressed via `types: ['tag']`.

---

### 4.3 `CreateGroupRequest` – `location` Optional vs Required

**Spec:** `location: Location` is required in `CreateGroupRequest`.

**Client:** `CreateGroupRequest` has `location?: Location` (optional).

**Fix:** Make `location` required in `CreateGroupRequest` to match the spec.

---

### 4.4 `UserOpenTheDoorRequest` – `reference_point`

**Spec:** `UserOpenTheDoorRequest { user_id: string }` – no `reference_point`.

**Client:** `UserOpenTheDoorRequest { user_id: number; reference_point: GeoPoint }` – includes `reference_point`.

**Note:** Extra `reference_point` may be a backend extension. If the spec is authoritative, remove it; otherwise keep and document.

---

## 5. Response Type / Structure Notes

### 5.1 `createReport`

**Spec:** `ApiResponse<ReportCreated>` with `report_id`.  
**Client:** `CreateReportData` with `report_id`.  
**Conclusion:** Compatible.

---

### 5.2 `createGroupChat` / `getUserGroups`

**Spec:** `ApiResponse<GroupItem>`, `ApiResponse<PaginatedData<GroupItem>>`.  
**Client:** `ApiResponse<JoinedGroupItem>`, `ApiResponse<PaginatedData<JoinedGroupItem>>`.  
**Conclusion:** OK – `JoinedGroupItem extends GroupItem`.

---

### 5.3 `getPrivateChats`

**Spec:** `ApiResponse<PaginatedData<PrivateChatItem>>`.  
**Client:** `ApiResponse<PaginatedData<PrivateChatListItem>>`.  
**Conclusion:** Fields should be aligned. `PrivateChatItem` includes `can_message`, `cant_message_reason`, `presence_status`, etc. Ensure `PrivateChatListItem` matches or can be safely mapped.

---

### 5.4 `applyToGroup`

**Spec:** `groups/me/join` returns `ApiResponse<JoinGroupResult>` with `status: 'joined' | 'pending_approval'` and optional `application`.

**Client:** Returns `ApiResponse<ApplyToGroupData>` (`application_id` only).

**Conclusion:** Inconsistent. When `status === 'pending_approval'`, `JoinGroupResult.application` contains `application_id`. The client may assume that case and only extract `application_id`. Prefer aligning the return type with `ApiResponse<JoinGroupResult>`.

---

## 6. Location Structure (Potential Cross-Cutting Issue)

**Spec (`api_spec_types.mdc`):**
```typescript
interface Location {
  point: GeoPoint;   // { latitude, longitude }
  location_name: string;
}
```

**Client (`api.ets`):**
```typescript
interface Location extends GeoPoint {
  location_name: string;
}
// => { latitude, longitude, location_name } (flat)
```

**Conclusion:** Spec uses `point: GeoPoint`; client uses a flat structure. If the backend expects the nested form, all requests/responses using `Location` may be wrong. This should be confirmed against the backend DTOs and corrected if needed.

---

## 7. Confirmed Correct Methods

The following methods are correctly implemented against the spec:

- **Auth:** `register`, `login`, `createTempUser`, `refreshToken`, `checkAuthStatus`, `logout`, `revokeAllSessions`, `changePassword`, `revokeSession`
- **User search:** `searchUsersByKeyword`, `searchUsersByTag`, `searchUsersByLocation`, `getUserProfile`
- **User lists:** `getUserFriendsList`, `getUserFollowingList`, `getUserFollowersList`, `getUserGroups`, `getUserLikedMoments`
- **User me:** `getMyProfile`, `updateProfile`, `updateLocation`, `upgradeAccount`, `getSettings`, `updateSettings`, `deleteAccount`, `followUser`, `unfollowUser`, `blockUser`, `unblockUser`, `getFriendsList`, `getFollowingList`, `getFollowersList`, `getBlockedList`, `setFriendRemark`
- **Moments:** `searchMoments*`, `getMomentDetail`, `getUserMoments`, `getMomentComments`, `getMomentLikes`, `createMoment`, `updateMoment`, `deleteMoment`, `likeMoment`, `unlikeMoment`, `favoriteMoment`, `unfavoriteMoment`, `getMyMoments`, `getMyFavorites`
- **Comments:** `getComments`, `getReplies`, `getCommentLikes`, `createComment`, `deleteComment`, `likeComment`, `unlikeComment`
- **Private chats:** `getPrivateChats`, `privateChatOpenTheDoor`, `updatePrivateChatSettings`, `deletePrivateChat`
- **Groups:** `searchGroups*`, `getGroupDetail`, `getGroupMembers`, `createGroupChat`, `updateGroupChat`, `deleteGroupChat`, `inviteMember`, `joinGroup`, `applyToGroup`, member management, applications, announcements
- **Chat messages:** `getPrivateMessages`, `getGroupMessages`, `updatePrivateMessage`, `updateGroupMessage`, `recallPrivateMessage`, `recallGroupMessage`, `markPrivateMessagesRead`, `markGroupMessagesRead`
- **Reports:** `createReport`, `getMyReports`
- **Feed:** `getFeedMoments`, `getFeedGroups`, `getFeedUsers`, `getFeedTrending`, `feedFeedback`
- **Tags:** `getTrendingTags` (shape fix still needed per §1.5)
- **Map:** `renderMap`
- **Share:** `createShare`, `getShare`, `applyShare`, `getMyShares`, `revokeShare`
- **Media:** `presignUpload`, `batchPresignUpload`, `completeUpload`
- **System:** `healthCheck`, `getSystemMeta`, `getSystemAnnouncements`

---

## 8. Recommended Fix Priority

1. **High:** `getGroupSettings` (endpoint/response), `sendPrivateMessage` / `sendGroupMessage` (response type)
2. **High:** Feed request types (`reference_point`), `suggestTags` request type
3. **Medium:** `getSessions` response type, `getTrendingTags` response shape
4. **Medium:** `fetchGlobalSettings` request/response alignment
5. **Medium:** `applyToGroup` return type
6. **Low:** `healthCheck` response type, `deletePrivateMessage` / `deleteGroupMessage` documentation
7. **Review:** Location structure if backend expects nested `point`

---

## 9. ApiEndpoints Path Verification

All paths in `ApiEndpoints.ets` were checked against the spec and match the documented endpoints.
