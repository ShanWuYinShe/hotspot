<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { groupsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Input } from '$lib/components/ui/input';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Label } from '$lib/components/ui/label';
	import { Tabs, TabsContent, TabsList, TabsTrigger } from '$lib/components/ui/tabs';
	import {
		DropdownMenu,
		DropdownMenuContent,
		DropdownMenuItem,
		DropdownMenuSeparator,
		DropdownMenuTrigger
	} from '$lib/components/ui/dropdown-menu';
	import * as Dialog from '$lib/components/ui/dialog';
	import type {
		GroupCard,
		GroupSettings,
		GroupAnnouncementItem,
		GroupMemberItem,
		SetableGroupRole,
		GroupRole
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

	const groupId = $page.params.id!;

	let group = $state<GroupCard | null>(null);
	let settings = $state<GroupSettings | null>(null);
	let isLoading = $state(true);
	let isJoining = $state(false);
	let isLeaving = $state(false);
	let showJoinDialog = $state(false);
	let joinPassword = $state('');
	let joinMessage = $state('');
	let settingsLoaded = $state(false);
	let currentTab = $state<'overview' | 'announcements' | 'members'>('overview');

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

	const canAccessGroupData = $derived(authStore.isAuthenticated && !!group?.my_state?.is_joined);
	const canManageGroup = $derived(
		authStore.isAuthenticated &&
			(group?.my_state?.role === 'owner' || group?.my_state?.role === 'admin')
	);
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

	let showRoleDialog = $state(false);
	let roleTarget = $state<GroupMemberItem | null>(null);
	let roleSubmitting = $state(false);

	let showRemoveDialog = $state(false);
	let removeTarget = $state<GroupMemberItem | null>(null);
	let removeSubmitting = $state(false);

	let showLeaveDialog = $state(false);

	function getJoinMethodLabel(method?: GroupSettings['join_method']) {
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

	function openAnnouncementDialog(announcement?: GroupAnnouncementItem) {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal();
			return;
		}

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

		announcementSubmitting = true;
		try {
			const response = editingAnnouncement
				? await groupsService.updateAnnouncement({
					announcement_id: editingAnnouncement.announcement_id,
					title,
					content,
					is_pinned: announcementPinned
				})
				: await groupsService.createAnnouncement({
					group_chat_id: groupId,
					title,
					content,
					is_pinned: announcementPinned
				});

			if (isSuccess(response) && response.data) {
				const updated = response.data;
				if (editingAnnouncement) {
					// 更新已有公告
					announcementsPagination.updateItem((item) =>
						item.announcement_id === updated.announcement_id ? updated : null
					);
					toast.success('公告已更新');
				} else {
					// 添加新公告到列表开头
					announcementsPagination.addItem(updated);
					toast.success('公告已发布');
				}
				showAnnouncementDialog = false;
				resetAnnouncementForm();
				announcementsLoaded = true;
			} else {
				toast.error(response.message || '操作失败');
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
			const response = await groupsService.deleteAnnouncement({ announcement_id: targetId });
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
		return isOwner && member.role !== 'owner' && member.user.user_id !== authStore.currentUserId;
	}

	function openMuteDialog(member: GroupMemberItem) {
		if (!canManageMember(member)) return;
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
		const duration = Number(muteDurationMinutes);
		if (!Number.isFinite(duration) || duration <= 0) {
			toast.error('请输入有效的禁言时长');
			return;
		}

		muteSubmitting = true;
		try {
			const response = await groupsService.mute({
				group_chat_id: groupId,
				user_id: muteTarget.user.user_id,
				duration_minutes: duration
			});
			if (isSuccess(response) && response.data) {
				const updated = response.data;
				// 更新成员状态
				membersPagination.updateItem((item) =>
					item.user.user_id === updated.user.user_id ? updated : null
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
				group_chat_id: groupId,
				user_id: member.user.user_id
			});
			if (isSuccess(response) && response.data) {
				const updated = response.data;
				// 更新成员状态
				membersPagination.updateItem((item) =>
					item.user.user_id === updated.user.user_id ? updated : null
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
		roleTarget = member;
		showRoleDialog = true;
	}

	async function handleSetRole(role: SetableGroupRole) {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal();
			return;
		}
		if (!roleTarget) return;

		roleSubmitting = true;
		try {
			const response = await groupsService.setRole({
				group_chat_id: groupId,
				user_id: roleTarget.user.user_id,
				role
			});
			if (isSuccess(response) && response.data) {
				const updated = response.data;
				// 更新成员角色
				membersPagination.updateItem((item) =>
					item.user.user_id === updated.user.user_id ? updated : null
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
				group_chat_id: groupId,
				user_id: removeTarget.user.user_id
			});
			if (isSuccess(response)) {
				// 从成员列表中删除
				membersPagination.removeItem((item) => item.user.user_id === removeTarget!.user.user_id);
				if (group) {
					group = { ...group, members_count: Math.max(0, group.members_count - 1) };
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

	async function loadGroup() {
		isLoading = true;
		try {
			const response = await groupsService.getGroup({ group_chat_id: groupId });
			if (isSuccess(response) && response.data) {
				group = response.data;
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
		} finally {
			isLoading = false;
		}
	}

	async function loadSettings() {
		if (!authStore.isAuthenticated || settingsLoaded) return;
		try {
			const response = await groupsService.getSettings({ group_chat_id: groupId });
			if (isSuccess(response) && response.data) {
				settings = response.data;
			}
		} catch {
			// ignore
		} finally {
			settingsLoaded = true;
		}
	}

	// 使用 usePagination hook，简化分页逻辑
	async function loadAnnouncements(refresh = false) {
		if (!canAccessGroupData) return;

		await announcementsPagination.load(
			(cursor) =>
				groupsService.getAnnouncements({
					group_chat_id: groupId,
					limit: 10,
					cursor
				}),
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
					group_chat_id: groupId,
					limit: 20,
					cursor
				}),
			refresh
		);

		if (!membersPagination.error) {
			membersLoaded = true;
		}
	}

	function openJoinDialog() {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal();
			return;
		}
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

		isJoining = true;
		try {
			const response = joinMethod === 'approval'
				? await groupsService.apply({
					group_chat_id: groupId,
					message: joinMessage.trim() || undefined
				})
				: await groupsService.join({
					group_chat_id: groupId,
					password: joinPassword.trim() || undefined
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
		showLeaveDialog = true;
	}

	async function handleLeave() {
		isLeaving = true;
		showLeaveDialog = false;
		try {
			const response = await groupsService.leave({ group_chat_id: groupId });
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

	onMount(() => {
		loadGroup();
		loadSettings();

		// 监听群组成员离开事件
		const handleMemberLeft = (e: CustomEvent<{
			group_chat_id: string;
			user_id: number;
			member_count: number;
		}>) => {
			if (e.detail.group_chat_id === groupId) {
				// 从成员列表中移除
				membersPagination.removeItem((item) => item.user.user_id === e.detail.user_id);
				// 更新群组成员数量
				if (group) {
					group = { ...group, members_count: e.detail.member_count };
				}
			}
		};

		// 监听群组成员被移除事件
		const handleMemberRemoved = (e: CustomEvent<{
			group_chat_id: string;
			user_id: number;
			member_count: number;
			is_current_user: boolean;
		}>) => {
			if (e.detail.group_chat_id === groupId) {
				// 从成员列表中移除
				membersPagination.removeItem((item) => item.user.user_id === e.detail.user_id);
				// 更新群组成员数量
				if (group) {
					group = { ...group, members_count: e.detail.member_count };
				}
				// 如果被移除的是当前用户，重新加载群组信息
				if (e.detail.is_current_user) {
					loadGroup();
				}
			}
		};

		// 监听群组成员角色变更事件
		const handleMemberRoleChanged = (e: CustomEvent<{
			group_chat_id: string;
			user_id: number;
			old_role: string;
			new_role: string;
		}>) => {
			if (e.detail.group_chat_id === groupId) {
				// 更新成员角色
				membersPagination.updateItem((item) => {
					if (item.user.user_id === e.detail.user_id) {
						return { ...item, role: e.detail.new_role as GroupRole };
					}
					return null;
				});
				// 如果角色变更的是当前用户，重新加载群组信息以更新权限
				if (e.detail.user_id === authStore.currentUserId) {
					loadGroup();
				}
			}
		};

		// 监听群组成员被禁言事件
		const handleMemberMuted = (e: CustomEvent<{
			group_chat_id: string;
			user_id: number;
			muted_until?: string;
		}>) => {
			if (e.detail.group_chat_id === groupId) {
				// 更新成员禁言状态
				membersPagination.updateItem((item) => {
					if (item.user.user_id === e.detail.user_id) {
						return { ...item, muted_until: e.detail.muted_until };
					}
					return null;
				});
				// 如果被禁言的是当前用户，重新加载群组信息
				if (e.detail.user_id === authStore.currentUserId) {
					loadGroup();
				}
			}
		};

		// 监听群组成员解除禁言事件
		const handleMemberUnmuted = (e: CustomEvent<{
			group_chat_id: string;
			user_id: number;
		}>) => {
			if (e.detail.group_chat_id === groupId) {
				// 更新成员禁言状态
				membersPagination.updateItem((item) => {
					if (item.user.user_id === e.detail.user_id) {
						return { ...item, muted_until: undefined };
					}
					return null;
				});
				// 如果解除禁言的是当前用户，重新加载群组信息
				if (e.detail.user_id === authStore.currentUserId) {
					loadGroup();
				}
			}
		};

		// 监听群公告更新事件
		const handleAnnouncementUpdated = (e: CustomEvent<{
			group_chat_id: string;
			announcement_id: string;
			content: string;
		}>) => {
			if (e.detail.group_chat_id === groupId) {
				// 更新公告内容
				announcementsPagination.updateItem((item) => {
					if (item.announcement_id === e.detail.announcement_id) {
						return { ...item, content: e.detail.content };
					}
					return null;
				});
			}
		};

		// 监听群组信息更新事件
		const handleGroupUpdated = (e: CustomEvent<{
			group_chat_id: string;
			updated_fields: {
				name?: string;
				avatar_url?: string;
				description?: string;
			};
		}>) => {
			if (e.detail.group_chat_id === groupId && group) {
				// 更新群组信息
				group = {
					...group,
					...e.detail.updated_fields
				};
			}
		};

		// 注册事件监听器
		window.addEventListener('group-member-left', handleMemberLeft as EventListener);
		window.addEventListener('group-member-removed', handleMemberRemoved as EventListener);
		window.addEventListener('group-member-role-changed', handleMemberRoleChanged as EventListener);
		window.addEventListener('group-member-muted', handleMemberMuted as EventListener);
		window.addEventListener('group-member-unmuted', handleMemberUnmuted as EventListener);
		window.addEventListener('group-announcement-updated', handleAnnouncementUpdated as EventListener);
		window.addEventListener('group-updated', handleGroupUpdated as EventListener);

		// 清理函数
		return () => {
			window.removeEventListener('group-member-left', handleMemberLeft as EventListener);
			window.removeEventListener('group-member-removed', handleMemberRemoved as EventListener);
			window.removeEventListener('group-member-role-changed', handleMemberRoleChanged as EventListener);
			window.removeEventListener('group-member-muted', handleMemberMuted as EventListener);
			window.removeEventListener('group-member-unmuted', handleMemberUnmuted as EventListener);
			window.removeEventListener('group-announcement-updated', handleAnnouncementUpdated as EventListener);
			window.removeEventListener('group-updated', handleGroupUpdated as EventListener);
		};
	});
</script>

<svelte:head>
	<title>{group?.name ?? '群组详情'} - HotSpot</title>
</svelte:head>

<div class="mx-auto max-w-5xl space-y-6">
	<div class="flex items-center gap-4">
		<Button variant="ghost" size="icon" onclick={() => history.back()}>
			<ArrowLeft class="h-5 w-5" />
		</Button>
		<h1 class="text-2xl font-bold">群组详情</h1>
	</div>

	{#if isLoading}
		<Card>
			<CardContent class="p-6">
				<div class="flex flex-col items-center gap-4">
					<Skeleton class="h-24 w-24 rounded-full" />
					<Skeleton class="h-6 w-32" />
					<Skeleton class="h-4 w-48" />
					<div class="flex gap-8">
						<Skeleton class="h-12 w-20" />
						<Skeleton class="h-12 w-20" />
						<Skeleton class="h-12 w-20" />
					</div>
				</div>
			</CardContent>
		</Card>
	{:else if group}
		<Card class="border-0 bg-card/80 shadow-lg shadow-black/5 ring-1 ring-border/50">
			<CardContent class="space-y-6 p-6">
				<div class="flex flex-col gap-6 lg:flex-row lg:items-center lg:justify-between">
					<div class="flex flex-col gap-4 sm:flex-row sm:items-center">
						<Avatar class="h-24 w-24 ring-2 ring-primary/15">
							<AvatarImage src={group.avatar_url} alt={group.name} />
							<AvatarFallback class="text-2xl">{group.name.slice(0, 2)}</AvatarFallback>
						</Avatar>
						<div>
							<div class="flex items-center gap-2">
								<h2 class="text-2xl font-bold">{group.name}</h2>
								{#if group.has_password}
									<Lock class="h-4 w-4 text-muted-foreground" />
								{/if}
							</div>
							{#if group.description}
								<p class="mt-2 max-w-xl text-sm text-muted-foreground">{group.description}</p>
							{/if}
							{#if group.tags && group.tags.length > 0}
								<div class="mt-3 flex flex-wrap gap-2">
									{#each group.tags as tag}
										<Badge variant="secondary">#{tag}</Badge>
									{/each}
								</div>
							{/if}
						</div>
					</div>

					<div class="flex flex-wrap gap-3">
						{#if group.my_state?.is_joined}
							<Button onclick={() => goto(`/chat/groups/${groupId}`)}>
								进入群聊
							</Button>
							<Button variant="outline" disabled class="cursor-default">
								已加入
							</Button>
							<Button variant="outline" onclick={confirmLeave} disabled={isLeaving}>
								{#if isLeaving}
									<Loader2 class="mr-2 h-4 w-4 animate-spin" />
								{/if}
								退出群组
							</Button>
						{:else}
							<Button
								onclick={openJoinDialog}
								disabled={settings?.join_method === 'invite_only'}
							>
								{settings?.join_method === 'invite_only' ? '仅限邀请' : '加入群组'}
							</Button>
						{/if}
					</div>
				</div>

				<div class="grid gap-3 sm:grid-cols-3">
					<div class="rounded-2xl border border-border/60 bg-background/70 p-4 text-left">
						<div class="flex items-center gap-2 text-sm text-muted-foreground">
							<Users class="h-4 w-4" />
							成员数量
						</div>
						<div class="mt-2 text-2xl font-bold">{group.members_count}</div>
					</div>
					<div class="rounded-2xl border border-border/60 bg-background/70 p-4 text-left">
						<div class="flex items-center gap-2 text-sm text-muted-foreground">
							<Shield class="h-4 w-4" />
							加入方式
						</div>
						<div class="mt-2 text-sm font-semibold">
							{getJoinMethodLabel(settings?.join_method)}
						</div>
					</div>
					<div class="rounded-2xl border border-border/60 bg-background/70 p-4 text-left">
						<div class="flex items-center gap-2 text-sm text-muted-foreground">
							<MapPin class="h-4 w-4" />
							群组位置
						</div>
						<div class="mt-2 text-sm font-semibold">
							{group.location.location_name}
						</div>
					</div>
				</div>
			</CardContent>
		</Card>

		<Tabs value={currentTab} onValueChange={(value) => (currentTab = value as 'overview' | 'announcements' | 'members')}>
			<TabsList class="h-auto gap-1.5 rounded-full bg-card/80 p-1.5 shadow-md shadow-black/10 ring-1 ring-border/50 backdrop-blur-sm">
				<TabsTrigger value="overview" class="rounded-full">概览</TabsTrigger>
				<TabsTrigger value="announcements" class="rounded-full">公告</TabsTrigger>
				<TabsTrigger value="members" class="rounded-full">成员</TabsTrigger>
			</TabsList>

			<TabsContent value="overview" class="mt-6 space-y-4">
				<div class="grid gap-4 lg:grid-cols-[1.2fr_0.8fr]">
					<Card class="border-0 bg-card/70 shadow-sm shadow-black/5">
						<CardContent class="space-y-3 p-5">
							<h3 class="text-sm font-semibold">群组说明</h3>
							<p class="text-sm text-muted-foreground">
								{group.description || '暂无群组简介，欢迎加入后一起完善。'}
							</p>
							{#if group.tags && group.tags.length > 0}
								<div class="flex flex-wrap gap-2">
									{#each group.tags as tag}
										<Badge variant="secondary" class="rounded-full">#{tag}</Badge>
									{/each}
								</div>
							{/if}
						</CardContent>
					</Card>
					<Card class="border-0 bg-card/70 shadow-sm shadow-black/5">
						<CardContent class="space-y-3 p-5">
							<h3 class="text-sm font-semibold">加入须知</h3>
							<div class="space-y-2 text-sm text-muted-foreground">
								<div class="flex items-center justify-between">
									<span>加入方式</span>
									<span class="font-medium text-foreground">{getJoinMethodLabel(settings?.join_method)}</span>
								</div>
								<div class="flex items-center justify-between">
									<span>是否需要密码</span>
									<span class="font-medium text-foreground">{group.has_password ? '需要' : '不需要'}</span>
								</div>
								<div class="flex items-center justify-between">
									<span>当前位置</span>
									<span class="font-medium text-foreground">{group.location.location_name}</span>
								</div>
							</div>
							{#if !authStore.isAuthenticated}
								<Button variant="outline" size="sm" class="w-full rounded-full" onclick={() => uiStore.openLoginModal()}>
									登录后加入
								</Button>
							{:else if group.my_state?.is_joined}
								<div class="rounded-xl border border-border/50 bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
									你已加入该群组
								</div>
							{:else}
								<Button size="sm" class="w-full rounded-full" onclick={openJoinDialog}>
									申请加入
								</Button>
							{/if}
						</CardContent>
					</Card>
				</div>
			</TabsContent>

			<TabsContent value="announcements" class="mt-6 space-y-4">
				{#if !authStore.isAuthenticated}
					<Card class="border-0 bg-card/60 shadow-sm shadow-black/5">
						<CardContent class="flex flex-col items-center gap-3 py-10 text-sm text-muted-foreground">
							<p>登录后查看群公告</p>
							<Button variant="outline" size="sm" class="rounded-full" onclick={() => uiStore.openLoginModal()}>
								去登录
							</Button>
						</CardContent>
					</Card>
				{:else if !group.my_state?.is_joined}
					<Card class="border-0 bg-card/60 shadow-sm shadow-black/5">
						<CardContent class="flex flex-col items-center gap-3 py-10 text-sm text-muted-foreground">
							<p>加入群组后可查看公告</p>
							<Button size="sm" class="rounded-full" onclick={openJoinDialog}>
								申请加入
							</Button>
						</CardContent>
					</Card>
				{:else}
					<div class="flex flex-col gap-3 rounded-2xl border border-border/50 bg-card/60 p-4 shadow-sm shadow-black/5 sm:flex-row sm:items-center sm:justify-between">
						<div>
							<p class="text-sm font-semibold">群公告</p>
							<p class="mt-1 text-xs text-muted-foreground">重要通知与管理提示将展示在这里</p>
						</div>
						<div class="flex flex-wrap gap-2">
							<Button
								variant="outline"
								size="sm"
								class="rounded-full"
								onclick={() => loadAnnouncements(true)}
							disabled={announcementsPagination.loading}
						>
							<RefreshCw class="mr-1.5 h-3.5 w-3.5" />
							刷新
						</Button>
						{#if canManageGroup}
							<Button size="sm" class="rounded-full" onclick={() => openAnnouncementDialog()}>
								<Plus class="mr-1.5 h-3.5 w-3.5" />
								发布公告
							</Button>
						{/if}
					</div>
				</div>
				{#if announcementsPagination.loading && announcementsPagination.items.length === 0}
					<div class="space-y-3">
						{#each Array(3) as _}
							<Card class="border-0 bg-card/60 shadow-sm shadow-black/5">
								<CardContent class="space-y-2 p-4">
									<Skeleton class="h-4 w-40" />
									<Skeleton class="h-4 w-full" />
									<Skeleton class="h-3 w-32" />
								</CardContent>
							</Card>
						{/each}
					</div>
				{:else if announcementsPagination.items.length === 0}
					<Card class="border-0 bg-card/60 shadow-sm shadow-black/5">
						<CardContent class="py-10 text-center text-sm text-muted-foreground">
							暂无公告
						</CardContent>
					</Card>
				{:else}
					<div class="space-y-3">
						{#each announcementsPagination.items as announcement (announcement.announcement_id)}
								<Card class="border-0 bg-card/70 shadow-sm shadow-black/5">
									<CardContent class="p-4">
										<div class="flex items-start justify-between gap-3">
											<div class="space-y-2">
												<div class="flex items-center gap-2">
													<p class="font-medium">{announcement.title}</p>
													{#if announcement.is_pinned}
														<Badge variant="secondary" class="rounded-full text-[10px]">置顶</Badge>
													{/if}
												</div>
												<p class="whitespace-pre-wrap text-sm text-muted-foreground">{announcement.content}</p>
												<div class="text-xs text-muted-foreground">
													{announcement.created_by.nickname} · {formatDateTime(announcement.created_at)}
												</div>
											</div>
											{#if canManageGroup}
												<DropdownMenu>
													<DropdownMenuTrigger
														class="inline-flex h-8 w-8 items-center justify-center rounded-full border border-border/60 bg-background/80 text-foreground shadow-sm shadow-black/5 transition-colors hover:bg-accent/60"
														aria-label="公告操作"
													>
														<MoreHorizontal class="h-4 w-4" />
													</DropdownMenuTrigger>
													<DropdownMenuContent align="end" class="w-40">
														<DropdownMenuItem onclick={() => openAnnouncementDialog(announcement)}>
															<Pencil class="mr-2 h-4 w-4" />
															编辑公告
														</DropdownMenuItem>
														<DropdownMenuSeparator />
														<DropdownMenuItem onclick={() => openDeleteAnnouncement(announcement)}>
															<Trash2 class="mr-2 h-4 w-4 text-destructive" />
															删除公告
														</DropdownMenuItem>
													</DropdownMenuContent>
												</DropdownMenu>
											{/if}
										</div>
									</CardContent>
								</Card>
						{/each}
					</div>
					{#if announcementsPagination.hasMore}
						<div class="flex justify-center pt-2">
							<Button
								variant="outline"
								size="sm"
								class="rounded-full"
								onclick={() => loadAnnouncements(false)}
								disabled={announcementsPagination.loadingMore}
							>
								{#if announcementsPagination.loadingMore}
									<Loader2 class="mr-2 h-4 w-4 animate-spin" />
								{/if}
								加载更多
							</Button>
						</div>
					{/if}
					{/if}
					{#if announcementsPagination.hasMore}
						<div class="flex justify-center pt-2">
							<Button
								variant="outline"
								size="sm"
								class="rounded-full"
								onclick={() => loadAnnouncements(false)}
								disabled={announcementsPagination.loadingMore}
							>
								{#if announcementsPagination.loadingMore}
									<Loader2 class="mr-2 h-4 w-4 animate-spin" />
								{/if}
								加载更多
							</Button>
						</div>
					{/if}
				{/if}
			</TabsContent>

			<TabsContent value="members" class="mt-6 space-y-4">
				{#if !authStore.isAuthenticated}
					<Card class="border-0 bg-card/60 shadow-sm shadow-black/5">
						<CardContent class="flex flex-col items-center gap-3 py-10 text-sm text-muted-foreground">
							<p>登录后查看成员</p>
							<Button variant="outline" size="sm" class="rounded-full" onclick={() => uiStore.openLoginModal()}>
								去登录
							</Button>
						</CardContent>
					</Card>
				{:else if !group.my_state?.is_joined}
					<Card class="border-0 bg-card/60 shadow-sm shadow-black/5">
						<CardContent class="flex flex-col items-center gap-3 py-10 text-sm text-muted-foreground">
							<p>加入群组后可查看成员</p>
							<Button size="sm" class="rounded-full" onclick={openJoinDialog}>
								申请加入
							</Button>
						</CardContent>
					</Card>
				{:else}
					<div class="flex flex-col gap-3 rounded-2xl border border-border/50 bg-card/60 p-4 shadow-sm shadow-black/5 sm:flex-row sm:items-center sm:justify-between">
						<div>
							<p class="text-sm font-semibold">成员列表</p>
							<p class="mt-1 text-xs text-muted-foreground">支持群主/管理员进行禁言与角色管理</p>
						</div>
						<div class="flex flex-wrap gap-2">
						<Button
							variant="outline"
							size="sm"
							class="rounded-full"
							onclick={() => loadMembers(true)}
							disabled={membersPagination.loading}
						>
							<RefreshCw class="mr-1.5 h-3.5 w-3.5" />
							刷新
						</Button>
						{#if canManageGroup}
							<Badge variant="secondary" class="rounded-full px-3 py-1 text-[11px]">可管理成员</Badge>
						{/if}
					</div>
				</div>
				{#if membersPagination.loading && membersPagination.items.length === 0}
					<div class="columns-1 gap-3 sm:columns-2 lg:columns-3">
						{#each Array(6) as _}
							<div class="mb-3 break-inside-avoid">
								<Card class="border-0 bg-card/60 shadow-sm shadow-black/5">
									<CardContent class="flex items-center gap-3 p-4">
										<Skeleton class="h-10 w-10 rounded-full" />
										<div class="flex-1 space-y-2">
											<Skeleton class="h-3 w-24" />
											<Skeleton class="h-3 w-32" />
										</div>
									</CardContent>
								</Card>
							</div>
						{/each}
					</div>
				{:else if membersPagination.items.length === 0}
					<Card class="border-0 bg-card/60 shadow-sm shadow-black/5">
						<CardContent class="py-10 text-center text-sm text-muted-foreground">
							暂无成员
						</CardContent>
					</Card>
				{:else}
					<div class="columns-1 gap-3 sm:columns-2 lg:columns-3">
						{#each membersPagination.items as member (member.user.user_id)}
								<div class="mb-3 break-inside-avoid">
									<Card class="border-0 bg-card/70 shadow-sm shadow-black/5 transition-colors hover:bg-accent/40">
										<CardContent class="flex items-center gap-3 p-4">
											<a
												href="/users/{member.user.user_id}"
												class="flex min-w-0 flex-1 items-center gap-3"
											>
												<Avatar class="h-10 w-10">
													<AvatarImage src={member.user.avatar_url} alt={member.user.nickname} />
													<AvatarFallback>{member.user.nickname.slice(0, 2)}</AvatarFallback>
												</Avatar>
												<div class="min-w-0 flex-1">
													<div class="flex items-center gap-2">
														<p class="truncate font-medium">{member.user.nickname}</p>
														<Badge variant="secondary" class="rounded-full text-[10px]">{getRoleLabel(member.role)}</Badge>
													</div>
													<p class="text-xs text-muted-foreground">
														加入于 {formatDateTime(member.joined_at)}
													</p>
												{#if member.muted_until}
													<p class="text-xs text-destructive/80">
														禁言至 {formatDateTime(member.muted_until)}
													</p>
												{/if}
												</div>
											</a>
											<div class="flex items-center gap-2">
												{#if member.muted_until}
													<Badge variant="outline" class="rounded-full text-[10px] text-muted-foreground">已禁言</Badge>
												{/if}
												{#if canManageMember(member)}
													<DropdownMenu>
														<DropdownMenuTrigger
															class="inline-flex h-8 w-8 items-center justify-center rounded-full border border-border/60 bg-background/80 text-foreground shadow-sm shadow-black/5 transition-colors hover:bg-accent/60"
															aria-label="成员操作"
														>
															<MoreHorizontal class="h-4 w-4" />
														</DropdownMenuTrigger>
														<DropdownMenuContent align="end" class="w-40">
															{#if member.muted_until}
																<DropdownMenuItem onclick={() => handleUnmuteMember(member)}>
																	<Mic class="mr-2 h-4 w-4" />
																	解除禁言
																</DropdownMenuItem>
															{:else}
																<DropdownMenuItem onclick={() => openMuteDialog(member)}>
																	<MicOff class="mr-2 h-4 w-4" />
																	禁言成员
																</DropdownMenuItem>
															{/if}
															{#if canChangeRole(member)}
																<DropdownMenuSeparator />
																<DropdownMenuItem onclick={() => openRoleDialog(member)}>
																	<UserCog class="mr-2 h-4 w-4" />
																	角色管理
																</DropdownMenuItem>
															{/if}
															<DropdownMenuSeparator />
															<DropdownMenuItem onclick={() => openRemoveDialog(member)}>
																<UserMinus class="mr-2 h-4 w-4 text-destructive" />
																移除成员
															</DropdownMenuItem>
														</DropdownMenuContent>
													</DropdownMenu>
												{/if}
											</div>
										</CardContent>
									</Card>
								</div>
						{/each}
					</div>
					{#if membersPagination.hasMore}
						<div class="flex justify-center pt-2">
							<Button
								variant="outline"
								size="sm"
								class="rounded-full"
								onclick={() => loadMembers(false)}
								disabled={membersPagination.loadingMore}
							>
								{#if membersPagination.loadingMore}
									<Loader2 class="mr-2 h-4 w-4 animate-spin" />
								{/if}
								加载更多
							</Button>
						</div>
					{/if}
				{/if}
				{/if}
			</TabsContent>
		</Tabs>
	{:else}
		<Card>
			<CardContent class="py-12 text-center">
				<p class="text-muted-foreground">群组不存在或已被解散</p>
				<Button variant="outline" class="mt-4" onclick={() => goto('/groups/nearby')}>
					去探索群组
				</Button>
			</CardContent>
		</Card>
	{/if}
</div>

<Dialog.Root bind:open={showJoinDialog}>
	<Dialog.Content class="sm:max-w-lg">
		<Dialog.Header>
			<Dialog.Title>加入群组</Dialog.Title>
			<Dialog.Description>
				当前加入方式：{getJoinMethodLabel(settings?.join_method)}。
				{settings?.join_method === 'approval' ? '请填写申请理由，等待管理员审核。' : '填写信息后即可加入群组。'}
			</Dialog.Description>
		</Dialog.Header>
		<div class="space-y-4 py-4">
			{#if group?.has_password}
				<div class="space-y-2">
					<Label for="group-password">群组密码</Label>
					<Input
						id="group-password"
						type="password"
						placeholder="请输入群组密码"
						bind:value={joinPassword}
					/>
				</div>
			{/if}
			{#if settings?.join_method === 'approval'}
				<div class="space-y-2">
					<Label for="join-message">申请理由</Label>
					<Textarea
						id="join-message"
						placeholder="简单介绍一下你自己"
						rows={3}
						bind:value={joinMessage}
					/>
				</div>
			{/if}
		</div>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (showJoinDialog = false)}>取消</Button>
			<Button onclick={handleJoin} disabled={isJoining}>
				{#if isJoining}
					<Loader2 class="mr-2 h-4 w-4 animate-spin" />
				{/if}
				确认加入
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={showAnnouncementDialog}>
	<Dialog.Content class="sm:max-w-xl">
		<Dialog.Header>
			<Dialog.Title>{editingAnnouncement ? '编辑公告' : '发布公告'}</Dialog.Title>
			<Dialog.Description>发布给所有成员的重要通知与群规提醒。</Dialog.Description>
		</Dialog.Header>
		<div class="space-y-4 py-4">
			<div class="space-y-2">
				<Label for="announcement-title">公告标题</Label>
				<Input
					id="announcement-title"
					placeholder="输入公告标题"
					bind:value={announcementTitle}
				/>
			</div>
			<div class="space-y-2">
				<Label for="announcement-content">公告内容</Label>
				<Textarea
					id="announcement-content"
					rows={5}
					placeholder="输入公告内容"
					bind:value={announcementContent}
				/>
			</div>
			<div class="flex items-center justify-between rounded-xl border border-border/60 bg-muted/30 px-4 py-3 text-sm">
				<div>
					<p class="font-medium">置顶公告</p>
					<p class="text-xs text-muted-foreground">置顶后将优先展示给成员</p>
				</div>
				<Button
					variant={announcementPinned ? 'default' : 'outline'}
					size="sm"
					class="rounded-full"
					onclick={() => (announcementPinned = !announcementPinned)}
				>
					{announcementPinned ? '已置顶' : '未置顶'}
				</Button>
			</div>
		</div>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (showAnnouncementDialog = false)}>取消</Button>
			<Button onclick={submitAnnouncement} disabled={announcementSubmitting}>
				{#if announcementSubmitting}
					<Loader2 class="mr-2 h-4 w-4 animate-spin" />
				{/if}
				{editingAnnouncement ? '保存修改' : '发布公告'}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={showAnnouncementDeleteDialog}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>删除公告</Dialog.Title>
			<Dialog.Description>删除后成员将无法再看到该公告。</Dialog.Description>
		</Dialog.Header>
		<div class="rounded-xl border border-border/60 bg-muted/30 px-4 py-3 text-sm text-muted-foreground">
			{announcementToDelete?.title ?? '该公告'}
		</div>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (showAnnouncementDeleteDialog = false)}>取消</Button>
			<Button variant="destructive" onclick={handleDeleteAnnouncement}>确认删除</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={showMuteDialog}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>禁言成员</Dialog.Title>
			<Dialog.Description>被禁言成员仍可浏览，但无法发言。</Dialog.Description>
		</Dialog.Header>
		<div class="space-y-4 py-2">
			<div class="text-sm text-muted-foreground">
				成员：{muteTarget?.user.nickname ?? '-'}
			</div>
			<div class="grid grid-cols-2 gap-2">
				{#each mutePresets as preset}
					<Button
						variant={Number(muteDurationMinutes) === preset.value ? 'default' : 'outline'}
						size="sm"
						class="rounded-full"
						onclick={() => (muteDurationMinutes = String(preset.value))}
					>
						{preset.label}
					</Button>
				{/each}
			</div>
			<div class="space-y-2">
				<Label for="mute-duration">自定义时长（分钟）</Label>
				<Input
					id="mute-duration"
					type="number"
					min="1"
					bind:value={muteDurationMinutes}
				/>
			</div>
		</div>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (showMuteDialog = false)}>取消</Button>
			<Button onclick={handleMuteMember} disabled={muteSubmitting}>
				{#if muteSubmitting}
					<Loader2 class="mr-2 h-4 w-4 animate-spin" />
				{/if}
				确认禁言
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={showRoleDialog}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>角色管理</Dialog.Title>
			<Dialog.Description>仅群主可以调整管理员权限。</Dialog.Description>
		</Dialog.Header>
		<div class="space-y-4 py-2">
			<div class="text-sm text-muted-foreground">
				成员：{roleTarget?.user.nickname ?? '-'}
			</div>
			<div class="grid gap-2 sm:grid-cols-2">
				<Button
					variant={roleTarget?.role === 'admin' ? 'default' : 'outline'}
					class="rounded-full"
					disabled={roleTarget?.role === 'admin' || roleSubmitting}
					onclick={() => handleSetRole('admin')}
				>
					设为管理员
				</Button>
				<Button
					variant={roleTarget?.role === 'member' ? 'default' : 'outline'}
					class="rounded-full"
					disabled={roleTarget?.role === 'member' || roleSubmitting}
					onclick={() => handleSetRole('member')}
				>
					设为成员
				</Button>
			</div>
		</div>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (showRoleDialog = false)}>取消</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={showRemoveDialog}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>移除成员</Dialog.Title>
			<Dialog.Description>确认将该成员移出群组吗？</Dialog.Description>
		</Dialog.Header>
		<div class="rounded-xl border border-border/60 bg-muted/30 px-4 py-3 text-sm text-muted-foreground">
			{removeTarget?.user.nickname ?? '-'}
		</div>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (showRemoveDialog = false)}>取消</Button>
			<Button variant="destructive" onclick={handleRemoveMember} disabled={removeSubmitting}>
				{#if removeSubmitting}
					<Loader2 class="mr-2 h-4 w-4 animate-spin" />
				{/if}
				确认移除
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<!-- 退出群组确认对话框 -->
<Dialog.Root bind:open={showLeaveDialog}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>退出群组</Dialog.Title>
			<Dialog.Description>确认要退出该群组吗？退出后将无法查看群组内容。</Dialog.Description>
		</Dialog.Header>
		<div class="rounded-xl border border-border/60 bg-muted/30 px-4 py-3">
			<div class="text-sm font-medium">{group?.name ?? '群组'}</div>
			<div class="mt-1 text-xs text-muted-foreground">
				{group?.members_count ?? 0} 位成员
			</div>
		</div>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (showLeaveDialog = false)}>取消</Button>
			<Button variant="destructive" onclick={handleLeave} disabled={isLeaving}>
				{#if isLeaving}
					<Loader2 class="mr-2 h-4 w-4 animate-spin" />
				{/if}
				确认退出
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
