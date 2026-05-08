<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount, untrack } from 'svelte';
	import { groupsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth.svelte';
	import { websocketStore } from '$lib/websocket';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { useChatWsSubscription } from '$lib/hooks/use-chat-ws-subscription.svelte';
	import type { PageData } from './$types';
	import {
		DropdownMenu,
		DropdownMenuContent,
		DropdownMenuItem,
		DropdownMenuSeparator,
		DropdownMenuTrigger
	} from '$lib/components/ui/dropdown-menu';
	import * as Dialog from '$lib/components/ui/dialog';
	import type {
		GroupItem,
		GroupAnnouncementItem,
		GroupMemberItem,
		GroupApplicationItem,
		SettableGroupRole,
		GroupJoinMethod
	} from '$lib/types/api';
	import { formatDateTime } from '$lib/utils/format';
	import ArrowLeft from 'lucide-svelte/icons/arrow-left';
	import Users from 'lucide-svelte/icons/users';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import Lock from 'lucide-svelte/icons/lock';
	import Shield from 'lucide-svelte/icons/shield';
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import Plus from 'lucide-svelte/icons/plus';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import MoreHorizontal from 'lucide-svelte/icons/more-horizontal';
	import Pencil from 'lucide-svelte/icons/pencil';
	import Trash2 from 'lucide-svelte/icons/trash-2';
	import MicOff from 'lucide-svelte/icons/mic-off';
	import Mic from 'lucide-svelte/icons/mic';
	import UserMinus from 'lucide-svelte/icons/user-minus';
	import UserCog from 'lucide-svelte/icons/user-cog';

	let { data }: { data: PageData } = $props();

	const groupId = $derived(data.groupId);

	let groupData = $state<GroupItem | null>(null);
	const group = $derived(groupData ?? null);
	let settings = $state<{ join_method?: 'open' | 'approval' | 'invite_only' } | null>(null);
	let isJoining = $state(false);
	let isLeaving = $state(false);
	let showJoinDialog = $state(false);
	let joinPassword = $state('');
	let joinMessage = $state('');
	let settingsLoaded = $state(false);
	let currentTab = $state<'overview' | 'announcements' | 'members' | 'applications'>('overview');

	// groupId 变化时重置子数据加载标志，确保切换群组后重新加载公告和成员
	$effect(() => {
		groupId; // 依赖声明
		untrack(() => {
			groupData = data.groupData;
			announcementsLoaded = false;
			membersLoaded = false;
			applicationsLoaded = false;
			settingsLoaded = false;
			settings = null;
			currentTab = 'overview';
			announcementsPagination.reset();
			membersPagination.reset();
			applicationsPagination.reset();
		});
	});

	// 使用 usePagination hook 统一管理公告和成员的分页状态
	const announcementsPagination = usePagination<GroupAnnouncementItem>({
		initialLoading: false,
		uniqueKey: 'announcement_id'
	});
	let announcementsLoaded = $state(false);

	const membersPagination = usePagination<GroupMemberItem>({
		initialLoading: false,
		uniqueKey: (item) => item.user.user_id
	});
	let membersLoaded = $state(false);

	const applicationsPagination = usePagination<GroupApplicationItem>({
		initialLoading: false,
		uniqueKey: 'application_id'
	});
	let applicationsLoaded = $state(false);
	let handlingApplications = $state<Set<string>>(new Set());

	const canAccessGroupData = $derived(authStore.isAuthenticated && !!group?.my_state?.is_joined);
	const canManageGroup = $derived(
		authStore.isAuthenticated &&
			(group?.my_state?.role === 'owner' || group?.my_state?.role === 'admin')
	);
	function setApplicationHandling(applicationId: string, handling: boolean) {
		const next = new Set(handlingApplications);
		if (handling) next.add(applicationId);
		else next.delete(applicationId);
		handlingApplications = next;
	}

	function isApplicationHandling(applicationId: string) {
		return handlingApplications.has(applicationId);
	}

	const isOwner = $derived(group?.my_state?.role === 'owner');

	let showAnnouncementDialog = $state(false);
	let announcementTitle = $state('');
	let announcementContent = $state('');
	let announcementPinned = $state(false);
	let announcementSubmitting = $state(false);
	let editingAnnouncement = $state<GroupAnnouncementItem | null>(null);
	let showAnnouncementDeleteDialog = $state(false);
	let announcementToDelete = $state<GroupAnnouncementItem | null>(null);

	let showMuteDialog = $state(false);
	let muteTarget = $state<GroupMemberItem | null>(null);
	let muteDurationMinutes = $state('30');
	let muteSubmitting = $state(false);
	const GROUP_PASSWORD_MIN = 4;
	const GROUP_PASSWORD_MAX = 32;
	const GROUP_JOIN_MESSAGE_MAX = 200;
	const ANNOUNCEMENT_TITLE_MAX = 100;
	const ANNOUNCEMENT_CONTENT_MAX = 5000;
	const MUTE_DURATION_MIN = 1;
	const MUTE_DURATION_MAX = 43200;

	let showRoleDialog = $state(false);
	let roleTarget = $state<GroupMemberItem | null>(null);
	let roleSubmitting = $state(false);

	let showRemoveDialog = $state(false);
	let removeTarget = $state<GroupMemberItem | null>(null);
	let removeSubmitting = $state(false);

	let showLeaveDialog = $state(false);

	function getJoinMethodLabel(method?: GroupJoinMethod) {
		switch (method) {
			case 'open':
				return '开放加入';
			case 'approval':
				return '需要审核';
			case 'invite_only':
				return '仅限邀请';
			default:
				return '未公开';
		}
	}

	function getRoleLabel(role: GroupMemberItem['role']) {
		if (role === 'owner') return '群主';
		if (role === 'admin') return '管理员';
		return '成员';
	}

	const mutePresets = [
		{ label: '30 分钟', value: 30 },
		{ label: '2 小时', value: 120 },
		{ label: '1 天', value: 1440 },
		{ label: '7 天', value: 10080 }
	];

	function resetAnnouncementForm() {
		announcementTitle = '';
		announcementContent = '';
		announcementPinned = false;
		editingAnnouncement = null;
	}

	function closeOverlayDialogs() {
		showJoinDialog = false;
		showAnnouncementDialog = false;
		showAnnouncementDeleteDialog = false;
		showMuteDialog = false;
		showRoleDialog = false;
		showRemoveDialog = false;
		showLeaveDialog = false;
		announcementToDelete = null;
		muteTarget = null;
		roleTarget = null;
		removeTarget = null;
		resetAnnouncementForm();
	}

	function openAnnouncementDialog(announcement?: GroupAnnouncementItem) {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal();
			return;
		}

		closeOverlayDialogs();

		if (announcement) {
			editingAnnouncement = announcement;
			announcementTitle = announcement.title;
			announcementContent = announcement.content;
			announcementPinned = announcement.is_pinned;
		} else {
			resetAnnouncementForm();
		}
		showAnnouncementDialog = true;
	}

	function openDeleteAnnouncement(announcement: GroupAnnouncementItem) {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal();
			return;
		}
		closeOverlayDialogs();
		announcementToDelete = announcement;
		showAnnouncementDeleteDialog = true;
	}

	async function submitAnnouncement() {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal();
			return;
		}
		if (!group) return;

		const title = announcementTitle.trim();
		const content = announcementContent.trim();
		if (!title || !content) {
			toast.error('请填写公告标题和内容');
			return;
		}
		if (title.length > ANNOUNCEMENT_TITLE_MAX) {
			toast.error(`公告标题最多 ${ANNOUNCEMENT_TITLE_MAX} 字符`);
			return;
		}
		if (content.length > ANNOUNCEMENT_CONTENT_MAX) {
			toast.error(`公告内容最多 ${ANNOUNCEMENT_CONTENT_MAX} 字符`);
			return;
		}

		announcementSubmitting = true;
		try {
			if (editingAnnouncement) {
				const response = await groupsService.updateAnnouncement({
					group_id: groupId,
					announcement_id: editingAnnouncement.announcement_id,
					title,
					content,
					is_pinned: announcementPinned
				});
				if (isSuccess(response)) {
					// updateAnnouncement returns void, manually update item
					const updatedItem = {
						...editingAnnouncement,
						title,
						content,
						is_pinned: announcementPinned
					};
					announcementsPagination.updateItem((item) =>
						item.announcement_id === updatedItem.announcement_id ? updatedItem : null
					);
					toast.success('公告已更新');
					showAnnouncementDialog = false;
					resetAnnouncementForm();
					announcementsLoaded = true;
				}
			} else {
				const response = await groupsService.createAnnouncement({
					group_id: groupId,
					title,
					content,
					is_pinned: announcementPinned
				});
				if (isSuccess(response) && response.data) {
					announcementsPagination.addItem(response.data);
					toast.success('公告已发布');
					showAnnouncementDialog = false;
					resetAnnouncementForm();
					announcementsLoaded = true;
				} else {
					toast.error(response.message || '操作失败');
				}
			}
		} catch {
			toast.error('操作失败');
		} finally {
			announcementSubmitting = false;
		}
	}

	async function handleDeleteAnnouncement() {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal();
			return;
		}
		if (!announcementToDelete) return;

		const targetId = announcementToDelete.announcement_id;
		showAnnouncementDeleteDialog = false;
		announcementToDelete = null;
		try {
			const response = await groupsService.deleteAnnouncement({ group_id: groupId, announcement_id: targetId });
			if (isSuccess(response)) {
				// 从列表中删除公告
				announcementsPagination.removeItem((item) => item.announcement_id === targetId);
				toast.success('公告已删除');
			} else {
				toast.error(response.message || '删除失败');
			}
		} catch {
			toast.error('删除失败');
		}
	}

	function canManageMember(member: GroupMemberItem) {
		if (!canManageGroup) return false;
		if (member.user.user_id === authStore.currentUserId) return false;
		if (member.role === 'owner') return false;
		if (!isOwner && member.role === 'admin') return false;
		return true;
	}

	function canChangeRole(member: GroupMemberItem) {
		return canManageMember(member);
	}

	function openMuteDialog(member: GroupMemberItem) {
		if (!canManageMember(member)) return;
		closeOverlayDialogs();
		muteTarget = member;
		muteDurationMinutes = '30';
		showMuteDialog = true;
	}

	async function handleMuteMember() {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal();
			return;
		}
		if (!muteTarget) return;
		const targetUserId = muteTarget.user.user_id;
		const duration = Number(muteDurationMinutes);
		if (
			!Number.isInteger(duration) ||
			duration < MUTE_DURATION_MIN ||
			duration > MUTE_DURATION_MAX
		) {
			toast.error(`禁言时长需在 ${MUTE_DURATION_MIN}-${MUTE_DURATION_MAX} 分钟之间`);
			return;
		}

		muteSubmitting = true;
		try {
			const response = await groupsService.mute({
				group_id: groupId,
				user_id: targetUserId,
				duration_minutes: duration
			});
			if (isSuccess(response)) {
				// 计算禁言到期时间
				const mutedUntil = new Date(Date.now() + duration * 60 * 1000).toISOString();
				// 更新成员状态
				membersPagination.updateItem((item) =>
					item.user.user_id === targetUserId ? { ...item, muted_until: mutedUntil } : null
				);
				toast.success('已设置禁言');
				showMuteDialog = false;
				muteTarget = null;
			} else {
				toast.error(response.message || '操作失败');
			}
		} catch {
			toast.error('操作失败');
		} finally {
			muteSubmitting = false;
		}
	}

	async function handleUnmuteMember(member: GroupMemberItem) {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal();
			return;
		}
		if (!canManageMember(member)) return;

		try {
			const response = await groupsService.unmute({
				group_id: groupId,
				user_id: member.user.user_id
			});
			if (isSuccess(response)) {
				// 更新成员状态
				membersPagination.updateItem((item) =>
					item.user.user_id === member.user.user_id ? { ...item, muted_until: undefined } : null
				);
				toast.success('已解除禁言');
			} else {
				toast.error(response.message || '操作失败');
			}
		} catch {
			toast.error('操作失败');
		}
	}

	function openRoleDialog(member: GroupMemberItem) {
		if (!canChangeRole(member)) return;
		closeOverlayDialogs();
		roleTarget = member;
		showRoleDialog = true;
	}

	async function handleSetRole(role: SettableGroupRole) {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal();
			return;
		}
		if (!roleTarget) return;
		const targetUserId = roleTarget.user.user_id;

		roleSubmitting = true;
		try {
			const response = await groupsService.setRole({
				group_id: groupId,
				user_id: targetUserId,
				role
			});
			if (isSuccess(response)) {
				// 更新成员角色
				membersPagination.updateItem((item) =>
					item.user.user_id === targetUserId ? { ...item, role } : null
				);
				toast.success('成员角色已更新');
				showRoleDialog = false;
				roleTarget = null;
			} else {
				toast.error(response.message || '操作失败');
			}
		} catch {
			toast.error('操作失败');
		} finally {
			roleSubmitting = false;
		}
	}

	function openRemoveDialog(member: GroupMemberItem) {
		if (!canManageMember(member)) return;
		closeOverlayDialogs();
		removeTarget = member;
		showRemoveDialog = true;
	}

	async function handleRemoveMember() {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal();
			return;
		}
		if (!removeTarget) return;

		removeSubmitting = true;
		try {
			const response = await groupsService.removeMember({
				group_id: groupId,
				user_id: removeTarget.user.user_id
			});
			if (isSuccess(response)) {
				// 从成员列表中删除
				membersPagination.removeItem((item) => item.user.user_id === removeTarget!.user.user_id);
				if (groupData) {
					groupData = {
						...groupData,
						group: { ...groupData.group, members_count: Math.max(0, groupData.group.members_count - 1) }
					};
				}
				toast.success('已移除成员');
				showRemoveDialog = false;
				removeTarget = null;
			} else {
				toast.error(response.message || '操作失败');
			}
		} catch {
			toast.error('操作失败');
		} finally {
			removeSubmitting = false;
		}
	}

	// 刷新群组信息（加入/离开群组后使用）
	async function loadGroup() {
		try {
			const response = await groupsService.getGroup({ group_id: groupId });
			if (isSuccess(response) && response.data) {
				groupData = response.data;
				if (!response.data.my_state?.is_joined) {
					// 未加入群组，重置分页状态
					announcementsPagination.reset();
					membersPagination.reset();
					announcementsLoaded = false;
					membersLoaded = false;
				}
			} else {
				toast.error(response.message || '群组不存在');
			}
		} catch {
			toast.error('加载失败');
		}
	}

	async function loadSettings() {
		if (!authStore.isAuthenticated || settingsLoaded) return;
		if (!group) return;
		// Use join_method from group if available
		settings = { join_method: group.join_method };
		settingsLoaded = true;
	}

	// 加载公告（非分页，返回所有公告）
	async function loadAnnouncements(refresh = false) {
		if (!canAccessGroupData) return;

		// 使用 usePagination 的 load 方法，但包装非分页响应为分页格式
		await announcementsPagination.load(
			async () => {
				const response = await groupsService.getAnnouncements({ group_id: groupId });
				if (isSuccess(response) && response.data) {
					return {
						code: 0,
						data: {
							items: response.data.items,
							next_cursor: undefined
						}
					};
				}
				return {
					code: response.code,
					message: response.message,
					data: {
						items: [],
						next_cursor: undefined
					}
				};
			},
			refresh
		);

		if (!announcementsPagination.error) {
			announcementsLoaded = true;
		}
	}

	async function loadMembers(refresh = false) {
		if (!canAccessGroupData) return;

		await membersPagination.load(
			(cursor) =>
				groupsService.getMembers({
					group_id: groupId,
					pagination: { limit: 20, cursor }
				}),
			refresh
		);

		if (!membersPagination.error) {
			membersLoaded = true;
		}
	}

	async function loadApplications(refresh = false) {
		if (!canManageGroup) return;
		await applicationsPagination.load(
			(cursor) =>
				groupsService.getApplications({
					group_id: groupId,
					status: 'pending',
					pagination: { limit: 20, cursor }
				}),
			refresh
		);
		if (!applicationsPagination.error) {
			applicationsLoaded = true;
		}
	}

	async function handleApplicationDecision(
		application: GroupApplicationItem,
		action: 'approve' | 'reject'
	) {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal();
			return;
		}
		if (isApplicationHandling(application.application_id)) return;
		setApplicationHandling(application.application_id, true);
		try {
			const response =
				action === 'approve'
					? await groupsService.approveApplication({ application_id: application.application_id })
					: await groupsService.rejectApplication({ application_id: application.application_id });
			if (isSuccess(response)) {
				applicationsPagination.removeItem(
					(item) => item.application_id === application.application_id
				);
				if (action === 'approve' && groupData) {
					groupData = {
						...groupData,
						group: {
							...groupData.group,
							members_count: groupData.group.members_count + 1
						}
					};
				}
				toast.success(action === 'approve' ? '已批准入群申请' : '已拒绝入群申请');
			} else {
				toast.error(response.message || '操作失败');
			}
		} catch {
			toast.error('操作失败');
		} finally {
			setApplicationHandling(application.application_id, false);
		}
	}

	function openJoinDialog() {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal();
			return;
		}
		closeOverlayDialogs();
		joinPassword = '';
		joinMessage = '';
		showJoinDialog = true;
	}

	async function handleJoin() {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal();
			return;
		}

		const joinMethod = settings?.join_method ?? 'open';
		if (joinMethod === 'invite_only') {
			toast.error('该群组仅支持邀请加入');
			return;
		}
		const trimmedPassword = joinPassword.trim();
		const trimmedJoinMessage = joinMessage.trim();
		if (joinMethod === 'open' && group?.has_password && !trimmedPassword) {
			toast.error('请输入群组密码');
			return;
		}
		if (
			joinMethod === 'open' &&
			group?.has_password &&
			trimmedPassword &&
			(trimmedPassword.length < GROUP_PASSWORD_MIN || trimmedPassword.length > GROUP_PASSWORD_MAX)
		) {
			toast.error(`群组密码长度需在 ${GROUP_PASSWORD_MIN}-${GROUP_PASSWORD_MAX} 字符之间`);
			return;
		}
		if (joinMethod === 'approval' && trimmedJoinMessage.length > GROUP_JOIN_MESSAGE_MAX) {
			toast.error(`申请理由最多 ${GROUP_JOIN_MESSAGE_MAX} 字符`);
			return;
		}

		isJoining = true;
		try {
			const response = await groupsService.join({
				group_id: groupId,
				password: trimmedPassword || undefined,
				message: joinMethod === 'approval' ? (trimmedJoinMessage || undefined) : undefined
			});

			if (isSuccess(response)) {
				toast.success(joinMethod === 'approval' ? '已提交申请' : '已加入群组');
				showJoinDialog = false;
				settingsLoaded = false;
				await loadGroup();
				await loadSettings();
			} else {
				toast.error(response.message || '操作失败');
			}
		} catch {
			toast.error('操作失败');
		} finally {
			isJoining = false;
		}
	}

	function confirmLeave() {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal();
			return;
		}
		closeOverlayDialogs();
		showLeaveDialog = true;
	}

	async function handleLeave() {
		isLeaving = true;
		showLeaveDialog = false;
		try {
			const response = await groupsService.leave({ group_id: groupId });
			if (isSuccess(response)) {
				toast.success('已退出群组');
				settingsLoaded = false;
				await loadGroup();
				await loadSettings();
			} else {
				toast.error(response.message || '操作失败');
			}
		} catch {
			toast.error('操作失败');
		} finally {
			isLeaving = false;
		}
	}

	$effect(() => {
		if (currentTab === 'announcements' && canAccessGroupData && !announcementsLoaded) {
			loadAnnouncements(true);
		}
	});

	$effect(() => {
		if (currentTab === 'members' && canAccessGroupData && !membersLoaded) {
			loadMembers(true);
		}
	});

	$effect(() => {
		if (currentTab === 'applications' && canManageGroup && !applicationsLoaded) {
			loadApplications(true);
		}
	});

	onMount(() => {
		if (data.error) {
			toast.error(data.error);
		}
		loadSettings();
	});

	// WebSocket 群组订阅管理（进入页面时订阅，离开时取消）
	const wsGroupSubscription = useChatWsSubscription();

	$effect(() => {
		let cancelled = false;
		const connected = websocketStore.isConnected;
		const id = groupId;

		wsGroupSubscription.sync({
			connected,
			type: 'group',
			ids: [id],
			onSubscribeFailed: () => {
				if (cancelled) return;
				toast.warning('群组实时订阅失败，正在重试...');
			}
		});

		return () => {
			cancelled = true;
			wsGroupSubscription.cleanup('group', [id]);
		};
	});

	// WebSocket 事件处理（Svelte 5 $effect，自动清理）
	$effect(() => {
		return websocketStore.on('GroupMemberLeft', (payload) => {
			if (payload.group_id !== groupId) return;
			membersPagination.removeItem((item) => item.user.user_id === payload.user.user_id);
			if (groupData) {
				groupData = {
					...groupData,
					group: { ...groupData.group, members_count: payload.member_count }
				};
			}
		});
	});

	$effect(() => {
		return websocketStore.on('GroupMemberRemoved', (payload) => {
			if (payload.group_id !== groupId) return;
			membersPagination.removeItem((item) => item.user.user_id === payload.user.user_id);
			if (groupData) {
				groupData = {
					...groupData,
					group: { ...groupData.group, members_count: payload.member_count }
				};
			}
			if (payload.user.user_id === authStore.currentUserId) {
				loadGroup();
			}
		});
	});

	$effect(() => {
		return websocketStore.on('GroupMemberMuted', (payload) => {
			if (payload.group_id !== groupId) return;
			membersPagination.updateItem((item) => {
				if (item.user.user_id === payload.user.user_id) {
					return { ...item, muted_until: payload.muted_until };
				}
				return null;
			});
			if (payload.user.user_id === authStore.currentUserId) {
				loadGroup();
			}
		});
	});

	$effect(() => {
		return websocketStore.on('GroupMemberUnmuted', (payload) => {
			if (payload.group_id !== groupId) return;
			membersPagination.updateItem((item) => {
				if (item.user.user_id === payload.user.user_id) {
					return { ...item, muted_until: undefined };
				}
				return null;
			});
			if (payload.user.user_id === authStore.currentUserId) {
				loadGroup();
			}
		});
	});

	$effect(() => {
		return websocketStore.on('GroupUpdated', (payload) => {
			if (payload.group_id !== groupId || !groupData) return;
			groupData = {
				...groupData,
				group: {
					...groupData.group,
					...(payload.updated_fields.name && { name: payload.updated_fields.name }),
					...(payload.updated_fields.avatar_url && { avatar_url: payload.updated_fields.avatar_url })
				},
				...(payload.updated_fields.description && { description: payload.updated_fields.description })
			};
		});
	});
</script>

<svelte:head>
	<title>{group?.group.name ?? '群组详情'} - HotSpot</title>
</svelte:head>

<div class="mx-auto max-w-5xl space-y-5">
	<!-- Sticky header -->
	<div
		class="sticky top-0 z-10 flex items-center gap-4 px-4 py-3"
		style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
	>
		<button
			class="flex items-center justify-center rounded-full outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
			style="width: 36px; height: 36px; background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);"
			onclick={() => history.back()}
			aria-label="返回"
		>
			<ArrowLeft size={18} color="var(--neu-text)" />
		</button>
		<div>
			<h1 style="font-size: 20px; font-weight: 600; color: var(--neu-text);">群组详情</h1>
			<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 2px;">查看群信息、成员与公告管理</p>
		</div>
	</div>

	{#if group && !wsGroupSubscription.state.isSubscribed}
		<div
			class="flex items-center gap-2 rounded-[16px] px-4 py-2"
			style="background: var(--neu-bg); box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-warning-border);"
		>
			<Loader2 size={14} class="animate-spin" style="color: var(--neu-warning);" />
			<span style="font-size: 14px; color: var(--neu-warning);">
				{#if wsGroupSubscription.state.isAttemptingSubscribe || !websocketStore.isConnected}
					群组实时事件连接中…
				{:else}
					群组实时事件订阅重试中…
				{/if}
			</span>
		</div>
	{/if}

	{#if !group}
		<div
			class="rounded-[24px] p-6"
			style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
		>
			<div class="flex flex-col items-center gap-4">
				<p style="color: var(--neu-text-muted);">
					{data.error || '群组不存在'}
				</p>
			</div>
		</div>
	{:else}
		<!-- Group profile card -->
		<div
			class="rounded-[24px] p-6"
			style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
		>
			<div class="space-y-6">
				<div class="flex flex-col gap-6 lg:flex-row lg:items-center lg:justify-between">
					<div class="flex flex-col gap-4 sm:flex-row sm:items-center">
						<div class="overflow-hidden rounded-full" style="width: 96px; height: 96px; border: 2px solid var(--neu-accent-primary); opacity: 0.9;">
							<img src={group.group.avatar_url} alt={group.group.name} class="h-full w-full object-cover" />
						</div>
						<div>
							<div class="flex items-center gap-2">
								<h2 style="font-size: 24px; font-weight: 700; color: var(--neu-text);">{group.group.name}</h2>
								{#if group.has_password}
									<Lock size={16} color="var(--neu-text-muted)" />
								{/if}
							</div>
							{#if group.description}
								<p style="margin-top: 8px; max-width: 36rem; font-size: 14px; color: var(--neu-text-muted);">{group.description}</p>
							{/if}
							{#if group.tags && group.tags.length > 0}
								<div class="mt-3 flex flex-wrap gap-2">
									{#each group.tags as tag (tag)}
										<span
											class="rounded-full px-2.5 py-1"
											style="font-size: 11px; color: var(--neu-accent-primary); background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);"
										>
											#{tag}
										</span>
									{/each}
								</div>
							{/if}
						</div>
					</div>

					<div class="flex flex-wrap gap-3">
						{#if group.my_state?.is_joined}
							<button
								class="flex items-center justify-center rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
								style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 14px var(--neu-glow);"
								onclick={() => goto(`/chat/groups/${groupId}`)}
							>
								进入群聊
							</button>
							<button
								class="flex items-center justify-center rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97] cursor-default"
								style="box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border); color: var(--neu-accent-primary); background: var(--neu-bg);"
								disabled
							>
								已加入
							</button>
							<button
								class="flex items-center justify-center rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
								style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
								onclick={confirmLeave}
								disabled={isLeaving}
							>
								{#if isLeaving}
									<Loader2 size={14} class="animate-spin" />
								{/if}
								退出群组
							</button>
						{:else}
							<button
								class="flex items-center justify-center rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
								style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 14px var(--neu-glow);"
								onclick={openJoinDialog}
								disabled={settings?.join_method === 'invite_only'}
							>
								{settings?.join_method === 'invite_only' ? '仅限邀请' : '加入群组'}
							</button>
						{/if}
					</div>
				</div>

				<!-- Stats row -->
				<div class="grid gap-3 sm:grid-cols-3">
					<div
						class="rounded-[18px] p-4 text-left"
						style="background: var(--neu-bg); box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);"
					>
						<div class="flex items-center gap-2" style="font-size: 14px; color: var(--neu-text-muted);">
							<Users size={16} />
							成员数量
						</div>
						<div style="margin-top: 8px; font-size: 24px; font-weight: 700; color: var(--neu-text);">{group.group.members_count}</div>
					</div>
					<div
						class="rounded-[18px] p-4 text-left"
						style="background: var(--neu-bg); box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);"
					>
						<div class="flex items-center gap-2" style="font-size: 14px; color: var(--neu-text-muted);">
							<Shield size={16} />
							加入方式
						</div>
						<div style="margin-top: 8px; font-size: 14px; font-weight: 600; color: var(--neu-text);">
							{getJoinMethodLabel(settings?.join_method)}
						</div>
					</div>
					<div
						class="rounded-[18px] p-4 text-left"
						style="background: var(--neu-bg); box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);"
					>
						<div class="flex items-center gap-2" style="font-size: 14px; color: var(--neu-text-muted);">
							<MapPin size={16} />
							群组位置
						</div>
						<div style="margin-top: 8px; font-size: 14px; font-weight: 600; color: var(--neu-text);">
							{group.location.location_name}
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- Tab bar -->
		<div class="flex flex-wrap gap-2">
			{#each [
				{ value: 'overview', label: '概览' },
				{ value: 'announcements', label: '公告' },
				{ value: 'members', label: '成员' },
				...(canManageGroup ? [{ value: 'applications', label: '申请' }] : [])
			] as tab (tab.value)}
				<button
					class="rounded-full px-4 py-2 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97] transition-all"
					style={currentTab === tab.value
						? 'background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 12px var(--neu-glow);'
						: 'background: var(--neu-bg); color: var(--neu-text-muted); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);'}
					onclick={() => (currentTab = tab.value as 'overview' | 'announcements' | 'members' | 'applications')}
				>
					<span style="font-size: 13px;">{tab.label}</span>
				</button>
			{/each}
		</div>

		<!-- Tab: Overview -->
		{#if currentTab === 'overview'}
			<div class="grid gap-4 lg:grid-cols-[1.2fr_0.8fr]">
				<div
					class="rounded-[24px] p-5"
					style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
				>
					<h3 class="uppercase" style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted); margin-bottom: 12px;">群组说明</h3>
					<p style="font-size: 14px; color: var(--neu-text-muted);">
						{group.description || '暂无群组简介，欢迎加入后一起完善。'}
					</p>
					{#if group.tags && group.tags.length > 0}
						<div class="mt-3 flex flex-wrap gap-2">
							{#each group.tags as tag (tag)}
								<span
									class="rounded-full px-2.5 py-1"
									style="font-size: 11px; color: var(--neu-accent-primary); background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);"
								>
									#{tag}
								</span>
							{/each}
						</div>
					{/if}
				</div>
				<div
					class="rounded-[24px] p-5"
					style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
				>
					<h3 class="uppercase" style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted); margin-bottom: 12px;">加入须知</h3>
					<div class="space-y-2" style="font-size: 14px; color: var(--neu-text-muted);">
						<div class="flex items-center justify-between">
							<span>加入方式</span>
							<span style="font-weight: 500; color: var(--neu-text);">{getJoinMethodLabel(settings?.join_method)}</span>
						</div>
						<div style="height: 1px; background: var(--neu-border);"></div>
						<div class="flex items-center justify-between">
							<span>是否需要密码</span>
							<span style="font-weight: 500; color: var(--neu-text);">{group.has_password ? '需要' : '不需要'}</span>
						</div>
						<div style="height: 1px; background: var(--neu-border);"></div>
						<div class="flex items-center justify-between">
							<span>当前位置</span>
							<span style="font-weight: 500; color: var(--neu-text);">{group.location.location_name}</span>
						</div>
					</div>
					<div class="mt-4">
						{#if !authStore.isAuthenticated}
							<button
								class="w-full rounded-full py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
								style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
								onclick={() => uiStore.openLoginModal()}
							>
								登录后加入
							</button>
						{:else if group.my_state?.is_joined}
							<div
								class="rounded-[14px] px-3 py-2 text-center"
								style="box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border); font-size: 12px; color: var(--neu-text-muted);"
							>
								你已加入该群组
							</div>
						{:else}
							<button
								class="w-full rounded-full py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
								style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 14px var(--neu-glow);"
								onclick={openJoinDialog}
							>
								申请加入
							</button>
						{/if}
					</div>
				</div>
			</div>
		{/if}

		<!-- Tab: Announcements (abbreviated for neumorphic style) -->
		{#if currentTab === 'announcements'}
			{#if !authStore.isAuthenticated}
				<div
					class="flex flex-col items-center gap-3 rounded-[24px] py-10"
					style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
				>
					<p style="font-size: 14px; color: var(--neu-text-muted);">登录后查看群公告</p>
					<button
						class="rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
						style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
						onclick={() => uiStore.openLoginModal()}
					>
						去登录
					</button>
				</div>
			{:else if !group.my_state?.is_joined}
				<div
					class="flex flex-col items-center gap-3 rounded-[24px] py-10"
					style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
				>
					<p style="font-size: 14px; color: var(--neu-text-muted);">加入群组后可查看公告</p>
					<button
						class="rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
						style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 14px var(--neu-glow);"
						onclick={openJoinDialog}
					>
						申请加入
					</button>
				</div>
			{:else}
				<!-- Announcements toolbar -->
				<div
					class="flex flex-col gap-3 rounded-[24px] p-4 sm:flex-row sm:items-center sm:justify-between"
					style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);"
				>
					<div>
						<p style="font-size: 14px; font-weight: 600; color: var(--neu-text);">群公告</p>
						<p style="margin-top: 4px; font-size: 12px; color: var(--neu-text-muted);">重要通知与管理提示将展示在这里</p>
					</div>
					<div class="flex flex-wrap gap-2">
						<button
							class="flex items-center gap-1.5 rounded-full px-4 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
							style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
							onclick={() => loadAnnouncements(true)}
							disabled={announcementsPagination.loading}
						>
							<RefreshCw size={14} />
							刷新
						</button>
						{#if canManageGroup}
							<button
								class="flex items-center gap-1.5 rounded-full px-4 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
								style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 14px var(--neu-glow);"
								onclick={() => openAnnouncementDialog()}
							>
								<Plus size={14} />
								发布公告
							</button>
						{/if}
					</div>
				</div>
				{#if announcementsPagination.loading && announcementsPagination.items.length === 0}
					<div class="space-y-3">
						{#each Array(3) as _, i (i)}
							<div
								class="rounded-[24px] p-4 space-y-2"
								style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
							>
								<div class="rounded" style="width: 160px; height: 14px; background: var(--neu-border);"></div>
								<div class="rounded" style="width: 100%; height: 14px; background: var(--neu-border);"></div>
								<div class="rounded" style="width: 128px; height: 12px; background: var(--neu-border);"></div>
							</div>
						{/each}
					</div>
				{:else if announcementsPagination.items.length === 0}
					<div
						class="rounded-[24px] py-10 text-center"
						style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
					>
						<p style="font-size: 14px; color: var(--neu-text-muted);">暂无公告</p>
					</div>
				{:else}
					<div class="space-y-3">
						{#each announcementsPagination.items as announcement (announcement.announcement_id)}
							<div
								class="rounded-[24px] p-4"
								style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
							>
								<div class="flex items-start justify-between gap-3">
									<div class="space-y-2">
										<div class="flex items-center gap-2">
											<p style="font-weight: 500; font-size: 15px; color: var(--neu-text);">{announcement.title}</p>
											{#if announcement.is_pinned}
												<span class="rounded-full px-2 py-0.5" style="font-size: 10px; color: var(--neu-accent-primary); background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);">置顶</span>
											{/if}
										</div>
										<p class="whitespace-pre-wrap" style="font-size: 14px; color: var(--neu-text-muted);">{announcement.content}</p>
										<div style="font-size: 12px; color: var(--neu-text-muted);">{formatDateTime(announcement.created_at)}</div>
									</div>
									{#if canManageGroup}
										<DropdownMenu>
											<DropdownMenuTrigger class="inline-flex h-8 w-8 items-center justify-center rounded-full outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text);" aria-label="公告操作">
												<MoreHorizontal size={16} />
											</DropdownMenuTrigger>
											<DropdownMenuContent align="end" class="w-40">
												<DropdownMenuItem onclick={() => openAnnouncementDialog(announcement)}>
													<Pencil size={16} style="margin-right: 8px;" />编辑公告
												</DropdownMenuItem>
												<DropdownMenuSeparator />
												<DropdownMenuItem onclick={() => openDeleteAnnouncement(announcement)}>
													<Trash2 size={16} style="margin-right: 8px; color: var(--neu-error);" />删除公告
												</DropdownMenuItem>
											</DropdownMenuContent>
										</DropdownMenu>
									{/if}
								</div>
							</div>
						{/each}
					</div>
					{#if announcementsPagination.hasMore}
						<div class="flex justify-center pt-2">
							<button class="flex items-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);" onclick={() => loadAnnouncements(false)} disabled={announcementsPagination.loadingMore}>
								{#if announcementsPagination.loadingMore}<Loader2 size={14} class="animate-spin" />{/if}
								加载更多
							</button>
						</div>
					{/if}
				{/if}
			{/if}
		{/if}

		<!-- Tab: Members (abbreviated for neumorphic style) -->
		{#if currentTab === 'members'}
			{#if !authStore.isAuthenticated}
				<div class="flex flex-col items-center gap-3 rounded-[24px] py-10" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);">
					<p style="font-size: 14px; color: var(--neu-text-muted);">登录后查看成员</p>
					<button class="rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);" onclick={() => uiStore.openLoginModal()}>去登录</button>
				</div>
			{:else if !group.my_state?.is_joined}
				<div class="flex flex-col items-center gap-3 rounded-[24px] py-10" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);">
					<p style="font-size: 14px; color: var(--neu-text-muted);">加入群组后可查看成员</p>
					<button class="rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 14px var(--neu-glow);" onclick={openJoinDialog}>申请加入</button>
				</div>
			{:else}
				<div class="flex flex-col gap-3 rounded-[24px] p-4 sm:flex-row sm:items-center sm:justify-between" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);">
					<div>
						<p style="font-size: 14px; font-weight: 600; color: var(--neu-text);">成员列表</p>
						<p style="margin-top: 4px; font-size: 12px; color: var(--neu-text-muted);">支持群主/管理员进行禁言与角色管理</p>
					</div>
					<div class="flex flex-wrap gap-2">
						<button class="flex items-center gap-1.5 rounded-full px-4 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);" onclick={() => loadMembers(true)} disabled={membersPagination.loading}>
							<RefreshCw size={14} />刷新
						</button>
						{#if canManageGroup}
							<span class="rounded-full px-3 py-1" style="font-size: 11px; color: var(--neu-accent-primary); background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);">可管理成员</span>
						{/if}
					</div>
				</div>
				{#if membersPagination.loading && membersPagination.items.length === 0}
					<div class="columns-1 gap-3 sm:columns-2 lg:columns-3">
						{#each Array(6) as _, i (i)}
							<div class="mb-3 break-inside-avoid">
								<div class="flex items-center gap-3 rounded-[24px] p-4" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);">
									<div class="rounded-full" style="width: 40px; height: 40px; background: var(--neu-border);"></div>
									<div class="flex-1 space-y-2">
										<div class="rounded" style="width: 96px; height: 12px; background: var(--neu-border);"></div>
										<div class="rounded" style="width: 128px; height: 12px; background: var(--neu-border);"></div>
									</div>
								</div>
							</div>
						{/each}
					</div>
				{:else if membersPagination.items.length === 0}
					<div class="rounded-[24px] py-10 text-center" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);">
						<p style="font-size: 14px; color: var(--neu-text-muted);">暂无成员</p>
					</div>
				{:else}
					<div class="columns-1 gap-3 sm:columns-2 lg:columns-3">
						{#each membersPagination.items as member (member.user.user_id)}
							<div class="mb-3 break-inside-avoid">
								<div class="flex items-center gap-3 rounded-[24px] p-4 transition-all" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);">
									<a href="/users/{member.user.user_id}" class="flex min-w-0 flex-1 items-center gap-3">
										<div class="overflow-hidden rounded-full" style="width: 40px; height: 40px; border: 1.5px solid var(--neu-border);">
											<img src={member.user.avatar_url} alt={member.user.nickname} class="h-full w-full object-cover" />
										</div>
										<div class="min-w-0 flex-1">
											<div class="flex items-center gap-2">
												<p class="truncate" style="font-weight: 500; font-size: 14px; color: var(--neu-text);">{member.user.nickname}</p>
												<span class="rounded-full px-2 py-0.5" style="font-size: 10px; color: var(--neu-accent-primary); background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);">{getRoleLabel(member.role)}</span>
											</div>
											<p style="font-size: 12px; color: var(--neu-text-muted);">加入于 {formatDateTime(member.joined_at)}</p>
											{#if member.muted_until}
												<p style="font-size: 12px; color: var(--neu-error); opacity: 0.8;">禁言至 {formatDateTime(member.muted_until)}</p>
											{/if}
										</div>
									</a>
									<div class="flex items-center gap-2">
										{#if member.muted_until}
											<span class="rounded-full px-2 py-0.5" style="font-size: 10px; color: var(--neu-text-muted); background: var(--neu-bg); box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);">已禁言</span>
										{/if}
										{#if canManageMember(member)}
											<DropdownMenu>
												<DropdownMenuTrigger class="inline-flex h-8 w-8 items-center justify-center rounded-full outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text);" aria-label="成员操作">
													<MoreHorizontal size={16} />
												</DropdownMenuTrigger>
												<DropdownMenuContent align="end" class="w-40">
													{#if member.muted_until}
														<DropdownMenuItem onclick={() => handleUnmuteMember(member)}><Mic size={16} style="margin-right: 8px;" />解除禁言</DropdownMenuItem>
													{:else}
														<DropdownMenuItem onclick={() => openMuteDialog(member)}><MicOff size={16} style="margin-right: 8px;" />禁言成员</DropdownMenuItem>
													{/if}
													{#if canChangeRole(member)}
														<DropdownMenuSeparator />
														<DropdownMenuItem onclick={() => openRoleDialog(member)}><UserCog size={16} style="margin-right: 8px;" />角色管理</DropdownMenuItem>
													{/if}
													<DropdownMenuSeparator />
													<DropdownMenuItem onclick={() => openRemoveDialog(member)}><UserMinus size={16} style="margin-right: 8px; color: var(--neu-error);" />移除成员</DropdownMenuItem>
												</DropdownMenuContent>
											</DropdownMenu>
										{/if}
									</div>
								</div>
							</div>
						{/each}
					</div>
					{#if membersPagination.hasMore}
						<div class="flex justify-center pt-2">
							<button class="flex items-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);" onclick={() => loadMembers(false)} disabled={membersPagination.loadingMore}>
								{#if membersPagination.loadingMore}<Loader2 size={14} class="animate-spin" />{/if}
								加载更多
							</button>
						</div>
					{/if}
				{/if}
			{/if}
		{/if}

		<!-- Tab: Applications -->
		{#if canManageGroup && currentTab === 'applications'}
			<div class="space-y-4">
				<div class="flex flex-col gap-3 rounded-[24px] p-4 sm:flex-row sm:items-center sm:justify-between" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);">
					<div>
						<p style="font-size: 14px; font-weight: 600; color: var(--neu-text);">待处理申请</p>
						<p style="margin-top: 4px; font-size: 12px; color: var(--neu-text-muted);">仅展示 pending 申请，处理后会自动移出列表</p>
					</div>
					<button class="flex items-center gap-1.5 rounded-full px-4 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);" onclick={() => loadApplications(true)} disabled={applicationsPagination.loading}>
						<RefreshCw size={14} />刷新
					</button>
				</div>
				{#if applicationsPagination.loading && applicationsPagination.items.length === 0}
					<div class="space-y-3">
						{#each Array(3) as _, i (i)}
							<div class="rounded-[24px] p-4 space-y-2" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);">
								<div class="rounded" style="width: 128px; height: 14px; background: var(--neu-border);"></div>
								<div class="rounded" style="width: 100%; height: 14px; background: var(--neu-border);"></div>
								<div class="rounded" style="width: 144px; height: 12px; background: var(--neu-border);"></div>
							</div>
						{/each}
					</div>
				{:else if applicationsPagination.items.length === 0}
					<div class="rounded-[24px] py-10 text-center" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);">
						<p style="font-size: 14px; color: var(--neu-text-muted);">暂无待处理申请</p>
					</div>
				{:else}
					<div class="space-y-3">
						{#each applicationsPagination.items as application (application.application_id)}
							<div class="rounded-[24px] p-4" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);">
								<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
									<div class="min-w-0 space-y-1">
										<div class="flex items-center gap-2">
											<div class="overflow-hidden rounded-full" style="width: 32px; height: 32px; border: 1.5px solid var(--neu-border);">
												<img src={application.applicant.avatar_url} alt={application.applicant.nickname} class="h-full w-full object-cover" />
											</div>
											<p class="truncate" style="font-size: 14px; font-weight: 500; color: var(--neu-text);">{application.applicant.nickname}</p>
										</div>
										{#if application.message}
											<p style="font-size: 14px; color: var(--neu-text-muted);">{application.message}</p>
										{/if}
										<p style="font-size: 12px; color: var(--neu-text-muted);">申请时间：{formatDateTime(application.created_at)}</p>
									</div>
									<div class="flex flex-wrap gap-2">
										<button class="rounded-full px-4 py-2 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 14px var(--neu-glow); font-size: 13px;" disabled={isApplicationHandling(application.application_id)} onclick={() => handleApplicationDecision(application, 'approve')}>
											{isApplicationHandling(application.application_id) ? '处理中...' : '批准'}
										</button>
										<button class="rounded-full px-4 py-2 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg); font-size: 13px;" disabled={isApplicationHandling(application.application_id)} onclick={() => handleApplicationDecision(application, 'reject')}>
											拒绝
										</button>
									</div>
								</div>
							</div>
						{/each}
					</div>
					{#if applicationsPagination.hasMore}
						<div class="flex justify-center pt-2">
							<button class="flex items-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);" onclick={() => loadApplications(false)} disabled={applicationsPagination.loadingMore}>
								{#if applicationsPagination.loadingMore}<Loader2 size={14} class="animate-spin" />{/if}
								加载更多
							</button>
						</div>
					{/if}
				{/if}
			</div>
		{/if}
	{/if}
</div>

<!-- Dialogs: kept with neumorphic inner styling -->
	<Dialog.Root bind:open={showJoinDialog}>
		<Dialog.Content class="sm:max-w-lg rounded-[24px] border var(--neu-bg) shadow-xl backdrop-blur-md" style="border-color: var(--neu-border);">
			<Dialog.Header>
				<Dialog.Title>加入群组</Dialog.Title>
				<Dialog.Description>
					当前加入方式：{getJoinMethodLabel(settings?.join_method)}。
					{settings?.join_method === 'approval' ? '请填写申请理由，等待管理员审核。' : '填写信息后即可加入群组。'}
				</Dialog.Description>
			</Dialog.Header>
			<div class="space-y-4 py-4">
				{#if group?.has_password}
					<div class="space-y-2 rounded-[18px] p-4" style="box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);">
						<label for="group-password" style="font-size: 13px; font-weight: 500; color: var(--neu-text);">群组密码</label>
						<input id="group-password" type="password" placeholder="请输入群组密码" bind:value={joinPassword} maxlength={GROUP_PASSWORD_MAX} class="w-full rounded-full py-2.5 px-4 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text);" />
					</div>
				{/if}
				{#if settings?.join_method === 'approval'}
					<div class="space-y-2 rounded-[18px] p-4" style="box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);">
						<label for="join-message" style="font-size: 13px; font-weight: 500; color: var(--neu-text);">申请理由</label>
						<textarea id="join-message" placeholder="简单介绍一下你自己" rows={3} bind:value={joinMessage} maxlength={GROUP_JOIN_MESSAGE_MAX} class="w-full resize-none rounded-[14px] p-3 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text);"></textarea>
					</div>
				{/if}
			</div>
			<Dialog.Footer>
				<button class="rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);" onclick={() => (showJoinDialog = false)}>取消</button>
				<button class="flex items-center justify-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 14px var(--neu-glow);" onclick={handleJoin} disabled={isJoining}>
					{#if isJoining}<Loader2 size={14} class="animate-spin" />{/if}
					确认加入
				</button>
			</Dialog.Footer>
		</Dialog.Content>
	</Dialog.Root>

	<Dialog.Root bind:open={showAnnouncementDialog}>
		<Dialog.Content class="sm:max-w-xl rounded-[24px] border var(--neu-bg) shadow-xl backdrop-blur-md" style="border-color: var(--neu-border);">
			<Dialog.Header>
				<Dialog.Title>{editingAnnouncement ? '编辑公告' : '发布公告'}</Dialog.Title>
				<Dialog.Description>发布给所有成员的重要通知与群规提醒。</Dialog.Description>
			</Dialog.Header>
			<div class="space-y-4 py-4">
				<div class="space-y-2 rounded-[18px] p-4" style="box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);">
					<label for="announcement-title" style="font-size: 13px; font-weight: 500; color: var(--neu-text);">公告标题</label>
					<input id="announcement-title" placeholder="输入公告标题" bind:value={announcementTitle} maxlength={ANNOUNCEMENT_TITLE_MAX} class="w-full rounded-full py-2.5 px-4 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text);" />
				</div>
				<div class="space-y-2 rounded-[18px] p-4" style="box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);">
					<label for="announcement-content" style="font-size: 13px; font-weight: 500; color: var(--neu-text);">公告内容</label>
					<textarea id="announcement-content" rows={5} placeholder="输入公告内容" bind:value={announcementContent} maxlength={ANNOUNCEMENT_CONTENT_MAX} class="w-full resize-none rounded-[14px] p-3 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text);"></textarea>
				</div>
				<div class="flex items-center justify-between rounded-[18px] px-4 py-3" style="box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);">
					<div>
						<p style="font-weight: 500; font-size: 14px; color: var(--neu-text);">置顶公告</p>
						<p style="font-size: 12px; color: var(--neu-text-muted);">置顶后将优先展示给成员</p>
					</div>
					<button class="rounded-full px-4 py-2 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style={announcementPinned ? 'background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 12px var(--neu-glow);' : 'box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);'} onclick={() => (announcementPinned = !announcementPinned)}>
						{announcementPinned ? '已置顶' : '未置顶'}
					</button>
				</div>
			</div>
			<Dialog.Footer>
				<button class="rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);" onclick={() => (showAnnouncementDialog = false)}>取消</button>
				<button class="flex items-center justify-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 14px var(--neu-glow);" onclick={submitAnnouncement} disabled={announcementSubmitting}>
					{#if announcementSubmitting}<Loader2 size={14} class="animate-spin" />{/if}
					{editingAnnouncement ? '保存修改' : '发布公告'}
				</button>
			</Dialog.Footer>
		</Dialog.Content>
	</Dialog.Root>

	<Dialog.Root bind:open={showAnnouncementDeleteDialog}>
		<Dialog.Content class="sm:max-w-md rounded-[24px] border var(--neu-bg) shadow-xl backdrop-blur-md" style="border-color: var(--neu-border);">
			<Dialog.Header>
				<Dialog.Title>删除公告</Dialog.Title>
				<Dialog.Description>删除后成员将无法再看到该公告。</Dialog.Description>
			</Dialog.Header>
			<div class="rounded-[18px] px-4 py-3" style="box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border); font-size: 14px; color: var(--neu-text-muted);">{announcementToDelete?.title ?? '该公告'}</div>
			<Dialog.Footer>
				<button class="rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);" onclick={() => (showAnnouncementDeleteDialog = false)}>取消</button>
				<button class="rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="background: var(--neu-error); color: var(--neu-active-text); box-shadow: 0 4px 14px var(--neu-error-border);" onclick={handleDeleteAnnouncement}>确认删除</button>
			</Dialog.Footer>
		</Dialog.Content>
	</Dialog.Root>

	<Dialog.Root bind:open={showMuteDialog}>
		<Dialog.Content class="sm:max-w-md rounded-[24px] border var(--neu-bg) shadow-xl backdrop-blur-md" style="border-color: var(--neu-border);">
			<Dialog.Header>
				<Dialog.Title>禁言成员</Dialog.Title>
				<Dialog.Description>被禁言成员仍可浏览，但无法发言。</Dialog.Description>
			</Dialog.Header>
			<div class="space-y-4 py-2">
				<div style="font-size: 14px; color: var(--neu-text-muted);">成员：{muteTarget?.user.nickname ?? '-'}</div>
				<div class="grid grid-cols-2 gap-2">
					{#each mutePresets as preset (preset.value)}
						<button class="rounded-full py-2 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style={Number(muteDurationMinutes) === preset.value ? 'background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 12px var(--neu-glow);' : 'box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);'} onclick={() => (muteDurationMinutes = String(preset.value))}>{preset.label}</button>
					{/each}
				</div>
				<div class="space-y-2">
					<label for="mute-duration" style="font-size: 13px; font-weight: 500; color: var(--neu-text);">自定义时长（分钟）</label>
					<input id="mute-duration" type="number" min={MUTE_DURATION_MIN} max={MUTE_DURATION_MAX} bind:value={muteDurationMinutes} class="w-full rounded-full py-2.5 px-4 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text);" />
				</div>
			</div>
			<Dialog.Footer>
				<button class="rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);" onclick={() => (showMuteDialog = false)}>取消</button>
				<button class="flex items-center justify-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 14px var(--neu-glow);" onclick={handleMuteMember} disabled={muteSubmitting}>
					{#if muteSubmitting}<Loader2 size={14} class="animate-spin" />{/if}
					确认禁言
				</button>
			</Dialog.Footer>
		</Dialog.Content>
	</Dialog.Root>

	<Dialog.Root bind:open={showRoleDialog}>
		<Dialog.Content class="sm:max-w-md rounded-[24px] border var(--neu-bg) shadow-xl backdrop-blur-md" style="border-color: var(--neu-border);">
			<Dialog.Header>
				<Dialog.Title>角色管理</Dialog.Title>
				<Dialog.Description>群主与管理员可调整成员角色；管理员不可操作管理员或群主。</Dialog.Description>
			</Dialog.Header>
			<div class="space-y-4 py-2">
				<div style="font-size: 14px; color: var(--neu-text-muted);">成员：{roleTarget?.user.nickname ?? '-'}</div>
				<div class="grid gap-2 sm:grid-cols-2">
					<button class="rounded-full py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style={roleTarget?.role === 'admin' ? 'background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 12px var(--neu-glow);' : 'box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);'} disabled={roleTarget?.role === 'admin' || roleSubmitting} onclick={() => handleSetRole('admin')}>设为管理员</button>
					<button class="rounded-full py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style={roleTarget?.role === 'member' ? 'background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 12px var(--neu-glow);' : 'box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);'} disabled={roleTarget?.role === 'member' || roleSubmitting} onclick={() => handleSetRole('member')}>设为成员</button>
				</div>
			</div>
			<Dialog.Footer>
				<button class="rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);" onclick={() => (showRoleDialog = false)}>取消</button>
			</Dialog.Footer>
		</Dialog.Content>
	</Dialog.Root>

	<Dialog.Root bind:open={showRemoveDialog}>
		<Dialog.Content class="sm:max-w-md rounded-[24px] border var(--neu-bg) shadow-xl backdrop-blur-md" style="border-color: var(--neu-border);">
			<Dialog.Header>
				<Dialog.Title>移除成员</Dialog.Title>
				<Dialog.Description>确认将该成员移出群组吗？</Dialog.Description>
			</Dialog.Header>
			<div class="rounded-[18px] px-4 py-3" style="box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border); font-size: 14px; color: var(--neu-text-muted);">{removeTarget?.user.nickname ?? '-'}</div>
			<Dialog.Footer>
				<button class="rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);" onclick={() => (showRemoveDialog = false)}>取消</button>
				<button class="flex items-center justify-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="background: var(--neu-error); color: var(--neu-active-text); box-shadow: 0 4px 14px var(--neu-error-border);" onclick={handleRemoveMember} disabled={removeSubmitting}>
					{#if removeSubmitting}<Loader2 size={14} class="animate-spin" />{/if}
					确认移除
				</button>
			</Dialog.Footer>
		</Dialog.Content>
	</Dialog.Root>

	<Dialog.Root bind:open={showLeaveDialog}>
		<Dialog.Content class="sm:max-w-md rounded-[24px] border var(--neu-bg) shadow-xl backdrop-blur-md" style="border-color: var(--neu-border);">
			<Dialog.Header>
				<Dialog.Title>退出群组</Dialog.Title>
				<Dialog.Description>确认要退出该群组吗？退出后将无法查看群组内容。</Dialog.Description>
			</Dialog.Header>
			<div class="rounded-[18px] px-4 py-3" style="box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);">
				<div style="font-size: 14px; font-weight: 500; color: var(--neu-text);">{group?.group.name ?? '群组'}</div>
				<div style="margin-top: 4px; font-size: 12px; color: var(--neu-text-muted);">{group?.group.members_count ?? 0} 位成员</div>
			</div>
			<Dialog.Footer>
				<button class="rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);" onclick={() => (showLeaveDialog = false)}>取消</button>
				<button class="flex items-center justify-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]" style="background: var(--neu-error); color: var(--neu-active-text); box-shadow: 0 4px 14px var(--neu-error-border);" onclick={handleLeave} disabled={isLeaving}>
					{#if isLeaving}<Loader2 size={14} class="animate-spin" />{/if}
					确认退出
				</button>
			</Dialog.Footer>
		</Dialog.Content>
	</Dialog.Root>
