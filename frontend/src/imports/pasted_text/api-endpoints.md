认证 auth（12）
端点	认证	幂等
POST /api/v1/auth/register	无	MUST
POST /api/v1/auth/login	无	无
POST /api/v1/auth/logout	必须	无
POST /api/v1/auth/guest	无	MUST
POST /api/v1/auth/refresh_token	可选	无
POST /api/v1/auth/check_status	可选	无
POST /api/v1/auth/request_password_reset	无	MUST
POST /api/v1/auth/reset_password	无	MUST
POST /api/v1/auth/me/change_password	必须	MUST
POST /api/v1/auth/me/sessions/list	必须	无
POST /api/v1/auth/me/sessions/revoke	必须	无
POST /api/v1/auth/me/sessions/revoke_all	必须	无
用户 users（26）
端点	认证	幂等
POST /api/v1/users/search	可选	无
POST /api/v1/users/open_the_door	可选	无
POST /api/v1/users/profile_get	可选	无
POST /api/v1/users/list_friends	条件必须	无
POST /api/v1/users/list_following	条件必须	无
POST /api/v1/users/list_followers	条件必须	无
POST /api/v1/users/list_groups	可选	无
POST /api/v1/users/list_moments	可选	无
POST /api/v1/users/list_liked_moments	可选	无
POST /api/v1/users/me/profile/get	必须	无
POST /api/v1/users/me/profile/update	必须	幂等语义
POST /api/v1/users/me/location/update	必须	幂等语义
POST /api/v1/users/me/upgrade	必须	无
POST /api/v1/users/me/settings/get	必须	无
POST /api/v1/users/me/settings/update	必须	幂等语义
POST /api/v1/users/me/delete	必须	SHOULD
POST /api/v1/users/me/follow	必须	幂等语义
POST /api/v1/users/me/unfollow	必须	幂等语义
POST /api/v1/users/me/block	必须	幂等语义
POST /api/v1/users/me/unblock	必须	幂等语义
POST /api/v1/users/me/list_blocked	必须	无
POST /api/v1/users/me/set_remark	必须	幂等语义
POST /api/v1/users/me/email/request_change	必须	MUST ⏳
POST /api/v1/users/me/email/confirm_change	必须	MUST ⏳
POST /api/v1/users/me/phone/request_change	必须	MUST ⏳
POST /api/v1/users/me/phone/confirm_change	必须	MUST ⏳
瞬间 moments（12）
端点	认证	幂等
POST /api/v1/moments/search	可选	无
POST /api/v1/moments/open_the_door	可选	无
POST /api/v1/moments/create	必须	MUST
POST /api/v1/moments/update	必须	幂等语义
POST /api/v1/moments/delete	必须	幂等语义
POST /api/v1/moments/me/list	必须	无
POST /api/v1/moments/me/list_favorites	必须	无
POST /api/v1/moments/me/like	必须	幂等语义
POST /api/v1/moments/me/unlike	必须	幂等语义
POST /api/v1/moments/me/favorite	必须	幂等语义
POST /api/v1/moments/me/unfavorite	必须	幂等语义
POST /api/v1/moments/list_likes	可选	无
评论 comments（7）
端点	认证	幂等
POST /api/v1/comments/create	必须	MUST
POST /api/v1/comments/delete	必须	幂等语义
POST /api/v1/comments/list	可选	无
POST /api/v1/comments/list_replies	可选	无
POST /api/v1/comments/me/like	必须	幂等语义
POST /api/v1/comments/me/unlike	必须	幂等语义
POST /api/v1/comments/list_likes	可选	无
聊天 private_chats + private_messages + group_messages（16）
端点	认证	幂等
POST /api/v1/private_chats/me/list	必须	无
POST /api/v1/private_chats/open_the_door	必须	幂等语义
POST /api/v1/private_chats/me/update_settings	必须	幂等语义
POST /api/v1/private_chats/me/delete	必须	幂等语义
POST /api/v1/private_messages/send	必须	MUST
POST /api/v1/private_messages/get_history	必须	无
POST /api/v1/private_messages/update	必须	幂等语义
POST /api/v1/private_messages/recall	必须	幂等语义
POST /api/v1/private_messages/me/mark_as_read	必须	幂等语义
POST /api/v1/private_messages/me/mark_all_read	必须	幂等语义
POST /api/v1/group_messages/send	必须	MUST
POST /api/v1/group_messages/get_history	必须	无
POST /api/v1/group_messages/update	必须	幂等语义
POST /api/v1/group_messages/recall	必须	幂等语义
POST /api/v1/group_messages/me/mark_as_read	必须	幂等语义
POST /api/v1/group_messages/me/mark_all_read	必须	幂等语义
群组 groups（34）
端点	认证	幂等
POST /api/v1/groups/search	可选	无
POST /api/v1/groups/open_the_door	可选	无
POST /api/v1/groups/create	必须	MUST
POST /api/v1/groups/update	必须	幂等语义
POST /api/v1/groups/dismiss	必须	SHOULD
POST /api/v1/groups/members/list	可选	无
POST /api/v1/groups/members/search	可选	无
POST /api/v1/groups/members/invite	必须	MUST
POST /api/v1/groups/members/invites/list	必须	无
POST /api/v1/groups/members/invites/revoke	必须	幂等语义
POST /api/v1/groups/members/set_role	必须	幂等语义
POST /api/v1/groups/members/mute	必须	幂等语义
POST /api/v1/groups/members/unmute	必须	幂等语义
POST /api/v1/groups/members/ban	必须	MUST
POST /api/v1/groups/members/unban	必须	幂等语义
POST /api/v1/groups/members/list_banned	必须	无
POST /api/v1/groups/members/remove_member	必须	幂等语义
POST /api/v1/groups/applications/list	必须	无
POST /api/v1/groups/applications/approve	必须	MUST
POST /api/v1/groups/applications/reject	必须	MUST
POST /api/v1/groups/me/update_settings	必须	幂等语义
POST /api/v1/groups/me/join	必须	MUST
POST /api/v1/groups/me/transfer_ownership	必须	SHOULD
POST /api/v1/groups/announcements/create	必须	MUST
POST /api/v1/groups/announcements/list	可选	无
POST /api/v1/groups/announcements/update	必须	幂等语义
POST /api/v1/groups/announcements/delete	必须	幂等语义
POST /api/v1/groups/me/list	必须	无
POST /api/v1/groups/me/applications/list	必须	无
POST /api/v1/groups/me/applications/cancel	必须	幂等语义
POST /api/v1/groups/me/invites/list	必须	无
POST /api/v1/groups/me/invites/accept	必须	MUST
POST /api/v1/groups/me/invites/reject	必须	MUST
POST /api/v1/groups/me/leave	必须	SHOULD
信息流 feed（5）
端点	认证	幂等
POST /api/v1/feed/list_moments	必须	无
POST /api/v1/feed/list_groups	必须	无
POST /api/v1/feed/list_users	必须	无
POST /api/v1/feed/list_trending	可选	无
POST /api/v1/feed/me/dismiss	必须	幂等语义 ⏳
搜索 search（2）
端点	认证	幂等
POST /api/v1/search/suggest	可选	无
POST /api/v1/search/all	可选	无
地图 map（1）
端点	认证	幂等
POST /api/v1/map/canvas	可选	无
分享 share（4）
端点	认证	幂等
POST /api/v1/share/create	必须	MUST
POST /api/v1/share/get	可选	无
POST /api/v1/share/me/list	必须	无
POST /api/v1/share/revoke	必须	幂等语义
媒体 media（4）
端点	认证	幂等
POST /api/v1/media/presign_upload	必须	无
POST /api/v1/media/batch_presign_upload	必须	无
POST /api/v1/media/complete_upload	必须	MUST
POST /api/v1/media/complete_upload/batch	必须	无
举报 reports（2）
端点	认证	幂等
POST /api/v1/reports/create	必须	MUST
POST /api/v1/reports/me/list	必须	无
通知 notifications（4）
端点	认证	幂等
POST /api/v1/notifications/list	必须	无
POST /api/v1/notifications/unread_count	必须	无
POST /api/v1/notifications/mark_as_read	必须	幂等语义
POST /api/v1/notifications/mark_all_read	必须	幂等语义
系统 system（4）
端点	认证	幂等
POST /api/v1/system/get_meta	无	无
POST /api/v1/system/health	无	无
POST /api/v1/system/list_announcements	无	无
POST /api/v1/system/get_client_config	无	无
管理后台核心 admin.rbac + admin.audit + admin.app_config（14）
端点	认证	幂等
POST /api/v1/admin/rbac/get_me	必须(管理员)	无
POST /api/v1/admin/rbac/list_permissions	必须+admin.rbac.manage	无
POST /api/v1/admin/rbac/roles/list	必须+admin.rbac.manage	无
POST /api/v1/admin/rbac/roles/create	必须+admin.rbac.manage	MUST
POST /api/v1/admin/rbac/roles/update	必须+admin.rbac.manage	幂等语义
POST /api/v1/admin/rbac/roles/delete	必须+admin.rbac.manage	幂等语义
POST /api/v1/admin/rbac/roles/set_permissions	必须+admin.rbac.manage	幂等语义
POST /api/v1/admin/rbac/admins/list	必须+admin.rbac.manage	无
POST /api/v1/admin/rbac/admins/grant	必须+admin.rbac.manage	MUST
POST /api/v1/admin/rbac/admins/revoke	必须+admin.rbac.manage	幂等语义
POST /api/v1/admin/audit/list	必须+admin.audit.read	无
POST /api/v1/admin/audit/open_the_door	必须+admin.audit.read	无
POST /api/v1/admin/app_config/get	必须+admin.app_config.read	无
POST /api/v1/admin/app_config/update	必须+admin.app_config.write	幂等语义
管理后台审核 admin.users + admin.groups + admin.reports + admin.tags + admin.announcements（15）
端点	认证	幂等
POST /api/v1/admin/users/search	必须+admin.users.read	无
POST /api/v1/admin/users/open_the_door	必须+admin.users.read	无
POST /api/v1/admin/users/ban	必须+admin.users.write	MUST
POST /api/v1/admin/users/unban	必须+admin.users.write	幂等语义
POST /api/v1/admin/groups/search	必须+admin.groups.read	无
POST /api/v1/admin/groups/open_the_door	必须+admin.groups.read	无
POST /api/v1/admin/groups/dismiss	必须+admin.groups.write	MUST
POST /api/v1/admin/reports/search	必须+admin.reports.read	无
POST /api/v1/admin/reports/handle	必须+admin.reports.write	MUST
POST /api/v1/admin/tags/search	必须+admin.tags.read	无
POST /api/v1/admin/tags/delete	必须+admin.tags.write	幂等语义
POST /api/v1/admin/announcements/search	必须+admin.announcements.read	无
POST /api/v1/admin/announcements/create	必须+admin.announcements.write	MUST
POST /api/v1/admin/announcements/update	必须+admin.announcements.write	幂等语义
POST /api/v1/admin/announcements/delete	必须+admin.announcements.write	幂等语义
WebSocket
- 端点: ws(s):///ws/v1
- 客户端事件: Auth, Ping, SetSubscriptions, GetSubscriptions, PrivateTyping, GroupTyping
- 服务端自动推送: ConnectionEstablished, Pong, Error, SubscribeAck, SubscriptionsSnapshot, 私聊消息事件, 社交通知, 瞬间互动, 群组邀请, UserProfileUpdated, SystemAnnouncement, AccountBanned
- 用户状态订阅推送: UserOnline, UserOffline, PrivateUserTyping
- 群聊订阅推送: GroupMessageNew/Edited/Recalled, GroupUserTyping, 成员变动事件, 群管理事件