import Bell from 'lucide-svelte/icons/bell';
import BellDot from 'lucide-svelte/icons/bell-dot';
import Heart from 'lucide-svelte/icons/heart';
import Info from 'lucide-svelte/icons/info';
import MessageCircle from 'lucide-svelte/icons/message-circle';
import UserPlus from 'lucide-svelte/icons/user-plus';
import Users from 'lucide-svelte/icons/users';
import { formatRelativeTime } from '$lib/utils/format';
import { translateWithLocale } from '$lib/i18n';
import type { Locale, TranslationKey } from '$lib/i18n';
import type { NotificationItem } from '$lib/types/api';

const notificationTitleKeys: Record<NotificationItem['type'], TranslationKey> = {
	new_follower: 'notificationDropdown.titles.newFollower',
	mutual_follow: 'notificationDropdown.titles.mutualFollow',
	moment_liked: 'notificationDropdown.titles.momentLiked',
	moment_commented: 'notificationDropdown.titles.momentCommented',
	comment_liked: 'notificationDropdown.titles.commentLiked',
	comment_reply: 'notificationDropdown.titles.commentReply',
	mentioned: 'notificationDropdown.titles.mentioned',
	group_invite_received: 'notificationDropdown.titles.groupInviteReceived',
	group_invite_accepted: 'notificationDropdown.titles.groupInviteAccepted',
	group_application_received: 'notificationDropdown.titles.groupApplicationReceived',
	group_application_accepted: 'notificationDropdown.titles.groupApplicationAccepted',
	system_announcement: 'notificationDropdown.titles.systemAnnouncement'
};

export function getNotificationIcon(type: NotificationItem['type']) {
	switch (type) {
		case 'new_follower':
		case 'mutual_follow':
			return UserPlus;
		case 'moment_liked':
		case 'comment_liked':
			return Heart;
		case 'moment_commented':
		case 'comment_reply':
		case 'mentioned':
			return MessageCircle;
		case 'group_invite_received':
		case 'group_invite_accepted':
		case 'group_application_received':
		case 'group_application_accepted':
			return Users;
		case 'system_announcement':
			return Info;
		default:
			return Bell;
	}
}

export function getNotificationBadgeIcon(unreadCount: number) {
	return unreadCount > 0 ? BellDot : Bell;
}

export function getNotificationTitle(item: NotificationItem, locale: Locale): string {
	return translateWithLocale(locale, notificationTitleKeys[item.payload.type] ?? 'notificationDropdown.titles.default');
}

export function getNotificationContent(item: NotificationItem, locale: Locale): string {
	const p = item.payload;
	switch (p.type) {
		case 'new_follower':
			return translateWithLocale(locale, 'notificationDropdown.content.newFollower', { nickname: p.actor.nickname });
		case 'mutual_follow':
			return translateWithLocale(locale, 'notificationDropdown.content.mutualFollow', { nickname: p.actor.nickname });
		case 'moment_liked':
			return translateWithLocale(locale, 'notificationDropdown.content.momentLiked', { nickname: p.actor.nickname });
		case 'moment_commented':
			return translateWithLocale(locale, 'notificationDropdown.content.momentCommented', {
				nickname: p.actor.nickname,
				preview: p.comment_preview
			});
		case 'comment_liked':
			return translateWithLocale(locale, 'notificationDropdown.content.commentLiked', { nickname: p.actor.nickname });
		case 'comment_reply':
			return translateWithLocale(locale, 'notificationDropdown.content.commentReply', {
				nickname: p.actor.nickname,
				preview: p.replied_comment_preview
			});
		case 'mentioned':
			return translateWithLocale(locale, 'notificationDropdown.content.mentioned', { nickname: p.actor.nickname });
		case 'group_invite_received':
			return translateWithLocale(locale, 'notificationDropdown.content.groupInviteReceived', {
				nickname: p.actor.nickname,
				groupName: p.group_name
			});
		case 'group_invite_accepted':
			return translateWithLocale(locale, 'notificationDropdown.content.groupInviteAccepted', {
				nickname: p.user.nickname,
				groupName: p.group_name
			});
		case 'group_application_received':
			return translateWithLocale(locale, 'notificationDropdown.content.groupApplicationReceived', {
				nickname: p.applicant.nickname,
				groupName: p.group_name
			});
		case 'group_application_accepted':
			return translateWithLocale(locale, 'notificationDropdown.content.groupApplicationAccepted', {
				groupName: p.group_name
			});
		case 'system_announcement':
			return p.severity === 'error'
				? translateWithLocale(locale, 'notificationDropdown.content.announcementImportant')
				: p.severity === 'warning'
					? translateWithLocale(locale, 'notificationDropdown.content.announcementWarning')
					: translateWithLocale(locale, 'notificationDropdown.content.announcementDefault');
		default:
			return '';
	}
}

export function getNotificationLink(item: NotificationItem): string {
	const p = item.payload;
	switch (p.type) {
		case 'new_follower':
		case 'mutual_follow':
			return `/users/${p.actor.user_id}`;
		case 'moment_liked':
		case 'moment_commented':
		case 'comment_liked':
		case 'comment_reply':
		case 'mentioned':
			return `/moments/${p.moment_id}`;
		case 'group_invite_received':
		case 'group_invite_accepted':
		case 'group_application_received':
		case 'group_application_accepted':
			return `/groups/${p.group_id}`;
		case 'system_announcement':
			return '';
		default:
			return '';
	}
}

export function formatNotificationTimestamp(timestamp: string, locale: Locale): string {
	return formatRelativeTime(timestamp, locale, {
		fallback: 'date',
		dateOptions: { month: 'short', day: 'numeric' }
	});
}
