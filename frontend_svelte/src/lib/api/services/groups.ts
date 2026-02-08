/**
 * 群组服务
 */

import { post, paginate } from '../client';
import { GROUPS, GROUP_MESSAGES } from '../endpoints';
import type {
	ApplicationStatus,
	GroupSearchItem,
	GroupSuggestions,
	GroupInfo,
	GroupSettings,
	GroupMemberItem,
	GroupAnnouncementItem,
	GroupApplicationItem,
	GroupMessageItem,
	MessageLocation,
	MessageReactionItem,
	MessageReadItem,
	GeoPoint,
	Location,
	SendMessageType,
	ReactionType,
	SetableGroupRole,
	UserInfo
} from '$lib/types/api';

export const groupsService = {
	// ==================== 搜索 ====================

	/**
	 * 关键词搜索群组
	 */
	searchByKeyword(params: { reference_point: GeoPoint; kw: string; limit?: number; cursor?: string }) {
		return paginate<GroupSearchItem>(GROUPS.SEARCH_BY_KEYWORD, params);
	},

	/**
	 * 群组搜索建议
	 */
	suggest(params: { prefix: string; limit?: number }) {
		return post<GroupSuggestions>(GROUPS.SUGGEST, params);
	},

	/**
	 * 标签搜索群组
	 */
	searchByTag(params: { reference_point: GeoPoint; tag: string; limit?: number; cursor?: string }) {
		return paginate<GroupSearchItem>(GROUPS.SEARCH_BY_TAG, params);
	},

	/**
	 * 位置搜索群组
	 */
	searchByLocation(params: {
		reference_point: GeoPoint;
		radius: number;
		kw?: string;
		limit?: number;
		cursor?: string;
	}) {
		return paginate<GroupSearchItem>(GROUPS.SEARCH_BY_LOCATION, params);
	},

	// ==================== 群组信息 ====================

	/**
	 * 获取群组详情
	 */
	getGroup(data: { group_id: string }) {
		return post<GroupSearchItem>(GROUPS.OPEN_THE_DOOR, data);
	},

	/**
	 * 获取群组设置
	 */
	getSettings(data: { group_id: string }) {
		return post<GroupSettings>(GROUPS.SETTINGS, data);
	},

	// ==================== 群组操作 ====================

	/**
	 * 创建群组
	 */
	create(data: {
		name: string;
		description?: string;
		avatar_file_id?: string;
		tags?: string[];
		password?: string;
		join_method?: 'open' | 'approval' | 'invite_only';
		is_searchable?: boolean;
		max_members?: number;
		location?: Location;
	}) {
		return post<void>(GROUPS.CREATE, data);
	},

	/**
	 * 更新群组
	 */
	update(data: {
		group_id: string;
		name?: string;
		description?: string;
		avatar_file_id?: string;
		tags?: string[];
		password?: string;
		join_method?: 'open' | 'approval' | 'invite_only';
		is_searchable?: boolean;
		max_members?: number;
		location?: Location;
	}) {
		return post<void>(GROUPS.UPDATE, data);
	},

	/**
	 * 更新我的群组设置
	 */
	updateMySettings(data: { group_id: string; is_muted?: boolean; is_pinned?: boolean }) {
		return post<void>(GROUPS.UPDATE_MY_SETTINGS, data);
	},

	/**
	 * 解散群组
	 */
	delete(data: { group_id: string }) {
		return post<void>(GROUPS.DELETE, data);
	},

	/**
	 * 转让群主
	 */
	transferOwnership(data: { group_id: string; new_owner_id: number }) {
		return post<void>(GROUPS.TRANSFER_OWNERSHIP, data);
	},

	// ==================== 成员管理 ====================

	/**
	 * 获取成员列表
	 */
	getMembers(params: { group_id: string; limit?: number; cursor?: string }) {
		return paginate<GroupMemberItem>(GROUPS.MEMBERS.LIST, params);
	},

	/**
	 * 邀请成员
	 */
	invite(data: { group_id: string; user_ids: number[] }) {
		return post<void>(GROUPS.MEMBERS.INVITE, data);
	},

	/**
	 * 直接加入（开放群）
	 */
	join(data: { group_id: string; password?: string }) {
		return post<void>(GROUPS.MEMBERS.JOIN, data);
	},

	/**
	 * 申请加入
	 */
	apply(data: { group_id: string; message?: string }) {
		return post<void>(GROUPS.MEMBERS.APPLY, data);
	},

	/**
	 * 设置成员角色
	 */
	setRole(data: { group_id: string; user_id: number; role: SetableGroupRole }) {
		return post<GroupMemberItem>(GROUPS.MEMBERS.SET_ROLE, data);
	},

	/**
	 * 禁言成员
	 */
	mute(data: { group_id: string; user_id: number; duration_minutes: number }) {
		return post<GroupMemberItem>(GROUPS.MEMBERS.MUTE, data);
	},

	/**
	 * 解除禁言
	 */
	unmute(data: { group_id: string; user_id: number }) {
		return post<GroupMemberItem>(GROUPS.MEMBERS.UNMUTE, data);
	},

	/**
	 * 封禁成员
	 */
	ban(data: { group_id: string; user_id: number; reason?: string }) {
		return post<void>(GROUPS.MEMBERS.BAN, data);
	},

	/**
	 * 解除封禁
	 */
	unban(data: { group_id: string; user_id: number }) {
		return post<void>(GROUPS.MEMBERS.UNBAN, data);
	},

	/**
	 * 获取封禁列表
	 */
	getBanned(params: { group_id: string; limit?: number; cursor?: string }) {
		return paginate<{ user: UserInfo; banned_at: string; reason?: string }>(
			GROUPS.MEMBERS.BANNED,
			params
		);
	},

	/**
	 * 移除成员
	 */
	removeMember(data: { group_id: string; user_id: number }) {
		return post<void>(GROUPS.MEMBERS.REMOVE_MEMBER, data);
	},

	// ==================== 申请管理 ====================

	/**
	 * 获取入群申请列表
	 */
	getApplications(params: { group_id: string; limit?: number; cursor?: string }) {
		return paginate<GroupApplicationItem>(GROUPS.APPLICATIONS.LIST, params);
	},

	/**
	 * 批准入群申请
	 */
	approveApplication(data: { application_id: string }) {
		return post<GroupApplicationItem>(GROUPS.APPLICATIONS.APPROVE, data);
	},

	/**
	 * 拒绝入群申请
	 */
	rejectApplication(data: { application_id: string; reason?: string }) {
		return post<GroupApplicationItem>(GROUPS.APPLICATIONS.REJECT, data);
	},

	// ==================== 群公告 ====================

	/**
	 * 获取群公告列表
	 */
	getAnnouncements(params: { group_id: string; limit?: number; cursor?: string }) {
		return paginate<GroupAnnouncementItem>(GROUPS.ANNOUNCEMENTS.LIST, params);
	},

	/**
	 * 发布群公告
	 */
	createAnnouncement(data: { group_id: string; title: string; content: string; is_pinned?: boolean }) {
		return post<GroupAnnouncementItem>(GROUPS.ANNOUNCEMENTS.CREATE, data);
	},

	/**
	 * 更新群公告
	 */
	updateAnnouncement(data: { announcement_id: string; title?: string; content?: string; is_pinned?: boolean }) {
		return post<GroupAnnouncementItem>(GROUPS.ANNOUNCEMENTS.UPDATE, data);
	},

	/**
	 * 删除群公告
	 */
	deleteAnnouncement(data: { announcement_id: string }) {
		return post<void>(GROUPS.ANNOUNCEMENTS.DELETE, data);
	},

	// ==================== 当前用户 ====================

	/**
	 * 获取我加入的群组列表
	 */
	getMyGroups(params?: { limit?: number; cursor?: string }) {
		return paginate<GroupSearchItem>(GROUPS.ME.LIST, params);
	},

	/**
	 * 取消入群申请
	 */
	cancelApplication(data: { group_id: string }) {
		return post<void>(GROUPS.ME.CANCEL_APPLICATION, data);
	},

	/**
	 * 获取收到的群组邀请
	 */
	getInvites(params?: { limit?: number; cursor?: string; status?: ApplicationStatus | 'all' }) {
		return paginate<{
			invite_id: string;
			group: GroupInfo;
			inviter: UserInfo;
			message?: string;
			status: ApplicationStatus;
			created_at: string;
			expires_at?: string;
		}>(GROUPS.ME.INVITES, params);
	},

	/**
	 * 接受群组邀请
	 */
	acceptInvite(data: { invite_id: string }) {
		return post<void>(GROUPS.ME.ACCEPT_INVITE, data);
	},

	/**
	 * 拒绝群组邀请
	 */
	rejectInvite(data: { invite_id: string }) {
		return post<void>(GROUPS.ME.REJECT_INVITE, data);
	},

	/**
	 * 退出群组
	 */
	leave(data: { group_id: string }) {
		return post<void>(GROUPS.ME.LEAVE, data);
	}
};

export const groupMessagesService = {
	/**
	 * 发送群消息
	 */
	send(data: {
		group_id: string;
		type: SendMessageType;
		content?: string;
		file_id?: string;
		location?: MessageLocation;
		reply_to_message_id?: string;
		mentions?: number[];
		idempotencyKey?: string;
	}) {
		const { idempotencyKey, ...payload } = data;
		return post<void>(GROUP_MESSAGES.SEND, payload, { idempotencyKey });
	},

	/**
	 * 获取消息历史
	 */
	getHistory(params: {
		group_id: string;
		limit?: number;
		cursor?: string;
		include_system?: boolean;
	}) {
		return paginate<GroupMessageItem>(GROUP_MESSAGES.HISTORY, params);
	},

	/**
	 * 编辑消息
	 */
	update(data: { message_id: string; content: string }) {
		return post<void>(GROUP_MESSAGES.UPDATE, data);
	},

	/**
	 * 撤回消息
	 */
	recall(data: { message_id: string }) {
		return post<void>(GROUP_MESSAGES.RECALL, data);
	},

	/**
	 * 删除消息（仅自己可见）
	 */
	delete(data: { message_id: string }) {
		return post<void>(GROUP_MESSAGES.DELETE, data);
	},

	/**
	 * 标记已读
	 */
	markAsRead(data: { group_id: string; up_to_message_id?: string }) {
		return post<void>(GROUP_MESSAGES.MARK_AS_READ, data);
	},

	/**
	 * 获取消息已读状态
	 */
	getReadStatus(params: { message_id: string; limit?: number; cursor?: string }) {
		return paginate<MessageReadItem>(GROUP_MESSAGES.READ_STATUS, params);
	},

	/**
	 * 添加消息反应
	 */
	addReaction(data: { message_id: string; type: ReactionType }) {
		return post<void>(GROUP_MESSAGES.ADD_REACTION, data);
	},

	/**
	 * 移除消息反应
	 */
	removeReaction(data: { message_id: string; type: ReactionType }) {
		return post<void>(GROUP_MESSAGES.REMOVE_REACTION, data);
	},

	/**
	 * 获取消息反应列表
	 */
	getReactions(params: { message_id: string; limit?: number; cursor?: string }) {
		return paginate<MessageReactionItem>(GROUP_MESSAGES.REACTIONS, params);
	}
};
