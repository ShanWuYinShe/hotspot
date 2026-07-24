<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount, untrack } from 'svelte';
  import { t } from '$lib/i18n';
  import { groupsService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { authStore } from '$lib/stores/auth.svelte';
  import { websocketStore } from '$lib/websocket';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
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
  import BackButton from '$lib/components/common/back-button.svelte';
  import Avatar from '$lib/components/common/avatar.svelte';
  import UserCard from '$lib/components/common/user-card.svelte';
  import { Textarea } from '$lib/components/ui/textarea';
  import Users from 'lucide-svelte/icons/users';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import Lock from 'lucide-svelte/icons/lock';
  import Shield from 'lucide-svelte/icons/shield';
  import Spinner from '$lib/components/common/spinner.svelte';
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

  // 群公告 / 成员列表为「认证可选」端点（见 spec/api_spec_group.md）：匿名与未加入
  // 用户也可查看公开字段。因此内容仅在群详情加载后即可浏览，敏感字段由服务端按可见性过滤。
  const canViewGroupContent = $derived(!!group);
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
        return t('groupDetail.joinMethodOpen');
      case 'approval':
        return t('groupDetail.joinMethodApproval');
      case 'invite_only':
        return t('groupDetail.joinMethodInviteOnly');
      default:
        return t('groupDetail.joinMethodUnknown');
    }
  }

  function getRoleLabel(role: GroupMemberItem['role']) {
    if (role === 'owner') return t('groupDetail.roleOwner');
    if (role === 'admin') return t('groupDetail.roleAdmin');
    return t('groupDetail.roleMember');
  }

  const mutePresets = $derived([
    { label: t('groupDetail.mutePreset30'), value: 30 },
    { label: t('groupDetail.mutePreset120'), value: 120 },
    { label: t('groupDetail.mutePreset1440'), value: 1440 },
    { label: t('groupDetail.mutePreset10080'), value: 10080 }
  ]);

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
      uiStore.openLoginModal();
      return;
    }
    closeOverlayDialogs();
    announcementToDelete = announcement;
    showAnnouncementDeleteDialog = true;
  }

  async function submitAnnouncement() {
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModal();
      return;
    }
    if (!group) return;

    const title = announcementTitle.trim();
    const content = announcementContent.trim();
    if (!title || !content) {
      toast.error(t('groupDetail.fillAnnouncementError'));
      return;
    }
    if (title.length > ANNOUNCEMENT_TITLE_MAX) {
      toast.error(t('groupDetail.announcementTitleMaxError', { count: ANNOUNCEMENT_TITLE_MAX }));
      return;
    }
    if (content.length > ANNOUNCEMENT_CONTENT_MAX) {
      toast.error(t('groupDetail.announcementContentMaxError', { count: ANNOUNCEMENT_CONTENT_MAX }));
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
          toast.success(t('groupDetail.announcementUpdated'));
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
          toast.success(t('groupDetail.announcementPublished'));
          showAnnouncementDialog = false;
          resetAnnouncementForm();
          announcementsLoaded = true;
        } else {
          toast.error(response.message || t('errors.operationFailed'));
        }
      }
    } catch {
      toast.error(t('errors.operationFailed'));
    } finally {
      announcementSubmitting = false;
    }
  }

  async function handleDeleteAnnouncement() {
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModal();
      return;
    }
    if (!announcementToDelete) return;

    const targetId = announcementToDelete.announcement_id;
    showAnnouncementDeleteDialog = false;
    announcementToDelete = null;
    try {
      const response = await groupsService.deleteAnnouncement({
        group_id: groupId,
        announcement_id: targetId
      });
      if (isSuccess(response)) {
        // 从列表中删除公告
        announcementsPagination.removeItem((item) => item.announcement_id === targetId);
        toast.success(t('groupDetail.announcementDeleted'));
      } else {
        toast.error(response.message || t('groupDetail.deleteFailed'));
      }
    } catch {
      toast.error(t('groupDetail.deleteFailed'));
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
      toast.error(t('groupDetail.muteDurationError', { min: MUTE_DURATION_MIN, max: MUTE_DURATION_MAX }));
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
        toast.success(t('groupDetail.muteSet'));
        showMuteDialog = false;
        muteTarget = null;
      } else {
        toast.error(response.message || t('errors.operationFailed'));
      }
    } catch {
      toast.error(t('errors.operationFailed'));
    } finally {
      muteSubmitting = false;
    }
  }

  async function handleUnmuteMember(member: GroupMemberItem) {
    if (!authStore.isAuthenticated) {
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
        toast.success(t('groupDetail.unmuted'));
      } else {
        toast.error(response.message || t('errors.operationFailed'));
      }
    } catch {
      toast.error(t('errors.operationFailed'));
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
        toast.success(t('groupDetail.roleUpdated'));
        showRoleDialog = false;
        roleTarget = null;
      } else {
        toast.error(response.message || t('errors.operationFailed'));
      }
    } catch {
      toast.error(t('errors.operationFailed'));
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
            group: {
              ...groupData.group,
              members_count: Math.max(0, groupData.group.members_count - 1)
            }
          };
        }
        toast.success(t('groupDetail.memberRemoved'));
        showRemoveDialog = false;
        removeTarget = null;
      } else {
        toast.error(response.message || t('errors.operationFailed'));
      }
    } catch {
      toast.error(t('errors.operationFailed'));
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
        toast.error(response.message || t('groupDetail.groupNotExist'));
      }
    } catch {
      toast.error(t('groupDetail.loadFailed'));
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
    if (!group) return;

    // 使用 usePagination 的 load 方法，但包装非分页响应为分页格式
    await announcementsPagination.load(async () => {
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
    }, refresh);

    if (!announcementsPagination.error) {
      announcementsLoaded = true;
    }
  }

  async function loadMembers(refresh = false) {
    if (!group) return;

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
        toast.success(action === 'approve' ? t('groupDetail.applicationApproved') : t('groupDetail.applicationRejected'));
      } else {
        toast.error(response.message || t('errors.operationFailed'));
      }
    } catch {
      toast.error(t('errors.operationFailed'));
    } finally {
      setApplicationHandling(application.application_id, false);
    }
  }

  function openJoinDialog() {
    if (!authStore.isAuthenticated) {
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
      uiStore.openLoginModal();
      return;
    }

    const joinMethod = settings?.join_method ?? 'open';
    if (joinMethod === 'invite_only') {
      toast.error(t('groupDetail.inviteOnlyJoinError'));
      return;
    }
    const trimmedPassword = joinPassword.trim();
    const trimmedJoinMessage = joinMessage.trim();
    if (joinMethod === 'open' && group?.has_password && !trimmedPassword) {
      toast.error(t('groupDetail.passwordRequired'));
      return;
    }
    if (
      joinMethod === 'open' &&
      group?.has_password &&
      trimmedPassword &&
      (trimmedPassword.length < GROUP_PASSWORD_MIN || trimmedPassword.length > GROUP_PASSWORD_MAX)
    ) {
      toast.error(t('groupDetail.passwordLengthError', { min: GROUP_PASSWORD_MIN, max: GROUP_PASSWORD_MAX }));
      return;
    }
    if (joinMethod === 'approval' && trimmedJoinMessage.length > GROUP_JOIN_MESSAGE_MAX) {
      toast.error(t('groupDetail.joinMessageMaxError', { count: GROUP_JOIN_MESSAGE_MAX }));
      return;
    }

    isJoining = true;
    try {
      const response = await groupsService.join({
        group_id: groupId,
        password: trimmedPassword || undefined,
        message: joinMethod === 'approval' ? trimmedJoinMessage || undefined : undefined
      });

      if (isSuccess(response)) {
        toast.success(joinMethod === 'approval' ? t('groupDetail.applicationSubmitted') : t('groupDetail.joinedGroup'));
        showJoinDialog = false;
        settingsLoaded = false;
        await loadGroup();
        await loadSettings();
      } else {
        toast.error(response.message || t('errors.operationFailed'));
      }
    } catch {
      toast.error(t('errors.operationFailed'));
    } finally {
      isJoining = false;
    }
  }

  function confirmLeave() {
    if (!authStore.isAuthenticated) {
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
        toast.success(t('groupDetail.leftGroup'));
        settingsLoaded = false;
        await loadGroup();
        await loadSettings();
      } else {
        toast.error(response.message || t('errors.operationFailed'));
      }
    } catch {
      toast.error(t('errors.operationFailed'));
    } finally {
      isLeaving = false;
    }
  }

  $effect(() => {
    if (currentTab === 'announcements' && canViewGroupContent && !announcementsLoaded) {
      loadAnnouncements(true);
    }
  });

  $effect(() => {
    if (currentTab === 'members' && canViewGroupContent && !membersLoaded) {
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

    const unsubscribeMemberLeft = websocketStore.on('GroupMemberLeft', (payload) => {
      if (payload.group_id !== groupId) return;
      membersPagination.removeItem((item) => item.user.user_id === payload.user.user_id);
      if (groupData) {
        groupData = {
          ...groupData,
          group: { ...groupData.group, members_count: payload.member_count }
        };
      }
    });

    const unsubscribeMemberRemoved = websocketStore.on('GroupMemberRemoved', (payload) => {
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

    const unsubscribeMemberMuted = websocketStore.on('GroupMemberMuted', (payload) => {
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

    const unsubscribeMemberUnmuted = websocketStore.on('GroupMemberUnmuted', (payload) => {
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

    const unsubscribeGroupUpdated = websocketStore.on('GroupUpdated', (payload) => {
      if (payload.group_id !== groupId || !groupData) return;
      groupData = {
        ...groupData,
        group: {
          ...groupData.group,
          ...(payload.updated_fields.name && { name: payload.updated_fields.name }),
          ...(payload.updated_fields.avatar_url && {
            avatar_url: payload.updated_fields.avatar_url
          })
        },
        ...(payload.updated_fields.description && {
          description: payload.updated_fields.description
        })
      };
    });

    return () => {
      unsubscribeMemberLeft();
      unsubscribeMemberRemoved();
      unsubscribeMemberMuted();
      unsubscribeMemberUnmuted();
      unsubscribeGroupUpdated();
    };
  });
</script>

<svelte:head>
  <title>{group?.group.name ?? t('groupDetail.pageTitle')} - HotSpot</title>
</svelte:head>

<!-- Sticky header — full-bleed, content aligned to the same narrow shell as the body -->
<div
  class="sticky top-0 z-10 page-pad-x pt-3 pb-3"
  style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
>
  <div class="page-shell flex items-center gap-4">
    <BackButton onclick={() => history.back()} size={36} />
    <div>
      <h1 class="text-h1" style="font-weight: 600; color: var(--text-primary)">{t('groupDetail.pageTitle')}</h1>
      <p class="text-meta" style="color: var(--text-secondary); margin-top: 2px">
        {t('groupDetail.pageSubtitle')}
      </p>
    </div>
  </div>
</div>

<div class="page-pad flex h-full flex-col overflow-y-auto no-scrollbar">
  <div class="page-shell space-y-5">
  {#if !group}
      <div
        class="rounded-[var(--radius-lg)] p-6"
        style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
      >
        <div class="flex flex-col items-center gap-4">
          <p style="color: var(--text-secondary);">
            {data.error || t('groupDetail.groupNotExist')}
          </p>
        </div>
      </div>
    {:else}
      <!-- Group profile card -->
      <div
        class="rounded-[var(--radius-lg)] p-6"
        style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
      >
        <div class="space-y-6">
          <div class="flex flex-col gap-6 lg:flex-row lg:items-center lg:justify-between">
            <div class="flex flex-col gap-4 sm:flex-row sm:items-center">
              <Avatar
                src={group.group.avatar_url}
                name={group.group.name}
                alt={group.group.name}
                size={96}
                type="group"
                class="opacity-90"
              />
              <div>
                <div class="flex items-center gap-2">
                  <h2
                    style="font-size: var(--text-display); font-weight: 700; color: var(--text-primary);"
                  >
                    {group.group.name}
                  </h2>
                  {#if group.has_password}
                    <Lock size={16} color="var(--text-secondary)" />
                  {/if}
                </div>
                {#if group.description}
                  <p
                    class="text-body-lg"
                    style="margin-top: 8px; max-width: 36rem; color: var(--text-secondary)"
                  >
                    {group.description}
                  </p>
                {/if}
                {#if group.tags && group.tags.length > 0}
                  <div class="mt-3 flex flex-wrap gap-2">
                    {#each group.tags as tag (tag)}
                      <span class="tag-chip">#{tag}</span>
                    {/each}
                  </div>
                {/if}
              </div>
            </div>

            <div class="flex flex-wrap gap-3">
              {#if group.my_state?.is_joined}
                <button
                  class="flex items-center justify-center rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
                  style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
                  onclick={() => goto(`/chat/groups/${groupId}`)}
                >
                  {t('groupDetail.enterChat')}
                </button>
                <button
                  class="flex items-center justify-center rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97] cursor-default"
                  style="border: 1px solid var(--border-theme); color: var(--brand); background: var(--surface-bg);"
                  disabled
                >
                  {t('groupDetail.joined')}
                </button>
                <button
                  class="flex items-center justify-center rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
                  style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
                  onclick={confirmLeave}
                  disabled={isLeaving}
                >
                  {#if isLeaving}
                    <Spinner size="sm" />
                  {/if}
                  {t('groupDetail.leaveGroup')}
                </button>
              {:else}
                <button
                  class="flex items-center justify-center rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
                  style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
                  onclick={openJoinDialog}
                  disabled={settings?.join_method === 'invite_only'}
                >
                  {settings?.join_method === 'invite_only' ? t('groupDetail.inviteOnly') : t('groupDetail.joinGroup')}
                </button>
              {/if}
            </div>
          </div>

          <!-- Stats row -->
          <div class="grid gap-3 sm:grid-cols-3">
            <div
              class="rounded-[var(--radius-md)] p-4 text-left"
              style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
            >
              <div
                class="text-body-lg flex items-center gap-2"
                style="color: var(--text-secondary)"
              >
                <Users size={16} />
                {t('groupDetail.membersCount')}
              </div>
              <div
                style="margin-top: 8px; font-size: var(--text-display); font-weight: 700; color: var(--text-primary);"
              >
                {group.group.members_count}
              </div>
            </div>
            <div
              class="rounded-[var(--radius-md)] p-4 text-left"
              style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
            >
              <div
                class="text-body-lg flex items-center gap-2"
                style="color: var(--text-secondary)"
              >
                <Shield size={16} />
                {t('groupDetail.joinMethod')}
              </div>
              <div
                class="text-body-lg"
                style="margin-top: 8px; font-weight: 600; color: var(--text-primary)"
              >
                {getJoinMethodLabel(settings?.join_method)}
              </div>
            </div>
            <div
              class="rounded-[var(--radius-md)] p-4 text-left"
              style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
            >
              <div
                class="text-body-lg flex items-center gap-2"
                style="color: var(--text-secondary)"
              >
                <MapPin size={16} />
                {t('groupDetail.groupLocation')}
              </div>
              <div
                class="text-body-lg"
                style="margin-top: 8px; font-weight: 600; color: var(--text-primary)"
              >
                {group.location.location_name}
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Tab bar -->
      <div class="flex flex-wrap gap-2">
        {#each [{ value: 'overview', label: t('groupDetail.tabOverview') }, { value: 'announcements', label: t('groupDetail.tabAnnouncements') }, { value: 'members', label: t('groupDetail.tabMembers') }, ...(canManageGroup ? [{ value: 'applications', label: t('groupDetail.tabApplications') }] : [])] as tab (tab.value)}
          <button
            class="filter-chip"
            class:filter-chip--active={currentTab === tab.value}
            onclick={() =>
              (currentTab = tab.value as 'overview' | 'announcements' | 'members' | 'applications')}
          >
            {tab.label}
          </button>
        {/each}
      </div>

      <!-- Tab: Overview -->
      {#if currentTab === 'overview'}
        <div class="grid gap-4 lg:grid-cols-[1.2fr_0.8fr]">
          <div
            class="rounded-[var(--radius-lg)] p-5"
            style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
          >
            <h3
              class="text-xs uppercase"
              style="letter-spacing: 0.14em; color: var(--text-secondary); margin-bottom: 12px"
            >
              {t('groupDetail.groupDescription')}
            </h3>
            <p class="text-body-lg" style="color: var(--text-secondary)">
              {group.description || t('groupDetail.descriptionEmpty')}
            </p>
            {#if group.tags && group.tags.length > 0}
              <div class="mt-3 flex flex-wrap gap-2">
                {#each group.tags as tag (tag)}
                  <span
                    class="text-caption rounded-full px-2.5 py-1"
                    style="color: var(--brand); background: var(--surface-bg); border: 1px solid var(--border-theme)"
                  >
                    #{tag}
                  </span>
                {/each}
              </div>
            {/if}
          </div>
          <div
            class="rounded-[var(--radius-lg)] p-5"
            style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
          >
            <h3
              class="text-xs uppercase"
              style="letter-spacing: 0.14em; color: var(--text-secondary); margin-bottom: 12px"
            >
              {t('groupDetail.joinNotice')}
            </h3>
            <div class="text-body-lg space-y-2" style="color: var(--text-secondary)">
              <div class="flex items-center justify-between">
                <span>{t('groupDetail.joinMethod')}</span>
                <span style="font-weight: 500; color: var(--text-primary);"
                  >{getJoinMethodLabel(settings?.join_method)}</span
                >
              </div>
              <div style="height: 1px; background: var(--border-theme);"></div>
              <div class="flex items-center justify-between">
                <span>{t('groupDetail.needPassword')}</span>
                <span style="font-weight: 500; color: var(--text-primary);"
                  >{group.has_password ? t('groupDetail.needPasswordYes') : t('groupDetail.needPasswordNo')}</span
                >
              </div>
              <div style="height: 1px; background: var(--border-theme);"></div>
              <div class="flex items-center justify-between">
                <span>{t('groupDetail.currentLocation')}</span>
                <span style="font-weight: 500; color: var(--text-primary);"
                  >{group.location.location_name}</span
                >
              </div>
            </div>
            <div class="mt-4">
              {#if !authStore.isAuthenticated}
                <button
                  class="w-full rounded-full py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
                  style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
                  onclick={() => uiStore.openLoginModal()}
                >
                  {t('groupDetail.loginToJoin')}
                </button>
              {:else if group.my_state?.is_joined}
                <div
                  class="text-meta rounded-[var(--radius-md)] px-3 py-2 text-center"
                  style="border: 1px solid var(--border-theme); color: var(--text-secondary)"
                >
                  {t('groupDetail.alreadyJoined')}
                </div>
              {:else}
                <button
                  class="w-full rounded-full py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
                  style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
                  onclick={openJoinDialog}
                >
                  {t('groupDetail.applyJoin')}
                </button>
              {/if}
            </div>
          </div>
        </div>
      {/if}

      <!-- Tab: Announcements (abbreviated for neumorphic style) -->
      {#if currentTab === 'announcements'}
        <!-- Announcements toolbar -->
        <div
          class="flex flex-col gap-3 rounded-[var(--radius-lg)] p-4 sm:flex-row sm:items-center sm:justify-between"
          style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
        >
          <div>
            <p class="text-body-lg" style="font-weight: 600; color: var(--text-primary)">{t('groupDetail.announcementsTitle')}</p>
            <p class="text-meta" style="margin-top: 4px; color: var(--text-secondary)">
              {t('groupDetail.announcementsDesc')}
            </p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button
              class="flex items-center gap-1.5 rounded-full px-4 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
              style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
              onclick={() => loadAnnouncements(true)}
              disabled={announcementsPagination.loading}
            >
              <RefreshCw size={14} />
              {t('common.refresh')}
            </button>
            {#if canManageGroup}
              <button
                class="flex items-center gap-1.5 rounded-full px-4 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
                style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
                onclick={() => openAnnouncementDialog()}
              >
                <Plus size={14} />
                {t('groupDetail.publishAnnouncement')}
              </button>
            {/if}
          </div>
        </div>
        {#if announcementsPagination.loading && announcementsPagination.items.length === 0}
          <div class="space-y-3">
            {#each Array(3) as _, i (i)}
              <div
                class="rounded-[var(--radius-lg)] p-4 space-y-2"
                style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
              >
                <div
                  class="rounded"
                  style="width: 160px; height: 14px; background: var(--border-theme);"
                ></div>
                <div
                  class="rounded"
                  style="width: 100%; height: 14px; background: var(--border-theme);"
                ></div>
                <div
                  class="rounded"
                  style="width: 128px; height: 12px; background: var(--border-theme);"
                ></div>
              </div>
            {/each}
          </div>
        {:else if announcementsPagination.items.length === 0}
          <div
            class="rounded-[var(--radius-lg)] py-10 text-center"
            style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
          >
            <p class="text-body-lg" style="color: var(--text-secondary)">{t('groupDetail.noAnnouncements')}</p>
          </div>
        {:else}
          <div class="space-y-3">
            {#each announcementsPagination.items as announcement (announcement.announcement_id)}
              <div
                class="rounded-[var(--radius-lg)] p-4"
                style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
              >
                <div class="flex items-start justify-between gap-3">
                  <div class="space-y-2">
                    <div class="flex items-center gap-2">
                      <p class="text-body-lg" style="font-weight: 500; color: var(--text-primary)">
                        {announcement.title}
                      </p>
                      {#if announcement.is_pinned}
                        <span
                          class="text-xs rounded-full px-2 py-0.5"
                          style="color: var(--brand); background: var(--surface-bg); border: 1px solid var(--border-theme)"
                          >{t('groupDetail.pinned')}</span
                        >
                      {/if}
                    </div>
                    <p
                      class="text-body-lg whitespace-pre-wrap"
                      style="color: var(--text-secondary)"
                    >
                      {announcement.content}
                    </p>
                    <div class="text-meta" style="color: var(--text-secondary)">
                      {formatDateTime(announcement.created_at)}
                    </div>
                  </div>
                  {#if canManageGroup}
                    <DropdownMenu>
                      <DropdownMenuTrigger
                        class="surface-flat inline-flex h-8 w-8 items-center justify-center rounded-full outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
                        aria-label={t('groupDetail.announcementActionsAria')}
                      >
                        <MoreHorizontal size={16} />
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" class="w-40">
                        <DropdownMenuItem onclick={() => openAnnouncementDialog(announcement)}>
                          <Pencil class="mr-2 h-4 w-4" />{t('groupDetail.editAnnouncement')}
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem onclick={() => openDeleteAnnouncement(announcement)}>
                          <Trash2 class="mr-2 h-4 w-4 text-destructive" />{t('groupDetail.deleteAnnouncement')}
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
              <button
                class="flex items-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
                style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
                onclick={() => loadAnnouncements(false)}
                disabled={announcementsPagination.loadingMore}
              >
                {#if announcementsPagination.loadingMore}<Spinner size="sm" />{/if}
                {t('common.loadMore')}
              </button>
            </div>
          {/if}
        {/if}
      {/if}

      <!-- Tab: Members (abbreviated for neumorphic style) -->
      {#if currentTab === 'members'}
        <div
          class="flex flex-col gap-3 rounded-[var(--radius-lg)] p-4 sm:flex-row sm:items-center sm:justify-between"
          style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
        >
          <div>
            <p class="text-body-lg" style="font-weight: 600; color: var(--text-primary)">
              {t('groupDetail.membersTitle')}
            </p>
            <p class="text-meta" style="margin-top: 4px; color: var(--text-secondary)">
              {t('groupDetail.membersDesc')}
            </p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button
              class="flex items-center gap-1.5 rounded-full px-4 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
              style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
              onclick={() => loadMembers(true)}
              disabled={membersPagination.loading}
            >
              <RefreshCw size={14} />{t('common.refresh')}
            </button>
            {#if canManageGroup}
              <span
                class="text-caption rounded-full px-3 py-1"
                style="color: var(--brand); background: var(--surface-bg); border: 1px solid var(--border-theme)"
                >{t('groupDetail.canManageMembers')}</span
              >
            {/if}
          </div>
        </div>
        {#if membersPagination.loading && membersPagination.items.length === 0}
          <div class="density-grid density-grid--compact">
            {#each Array(6) as _, i (i)}
              <div>
                <div
                  class="flex items-center gap-3 rounded-[var(--radius-lg)] p-4"
                  style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
                >
                  <div
                    class="rounded-full"
                    style="width: 40px; height: 40px; background: var(--border-theme);"
                  ></div>
                  <div class="flex-1 space-y-2">
                    <div
                      class="rounded"
                      style="width: 96px; height: 12px; background: var(--border-theme);"
                    ></div>
                    <div
                      class="rounded"
                      style="width: 128px; height: 12px; background: var(--border-theme);"
                    ></div>
                  </div>
                </div>
              </div>
            {/each}
          </div>
        {:else if membersPagination.items.length === 0}
          <div
            class="rounded-[var(--radius-lg)] py-10 text-center"
            style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
          >
            <p class="text-body-lg" style="color: var(--text-secondary)">{t('groupDetail.noMembers')}</p>
          </div>
        {:else}
          <div class="density-grid density-grid--compact">
            {#each membersPagination.items as member (member.user.user_id)}
              <UserCard
                variant="member-row"
                user={member.user}
                roleLabel={getRoleLabel(member.role)}
                subtitle={t('groupDetail.joinedAt', { time: formatDateTime(member.joined_at) })}
                mutedLabel={member.muted_until
                  ? t('groupDetail.mutedUntil', { time: formatDateTime(member.muted_until) })
                  : undefined}
              >
                {#snippet actions()}
                  {#if canManageMember(member)}
                    <DropdownMenu>
                      <DropdownMenuTrigger
                        class="surface-flat inline-flex h-8 w-8 items-center justify-center rounded-full outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
                        aria-label={t('groupDetail.memberActionsAria')}
                      >
                        <MoreHorizontal size={16} />
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" class="w-40">
                        {#if member.muted_until}
                          <DropdownMenuItem onclick={() => handleUnmuteMember(member)}>
                            <Mic class="mr-2 h-4 w-4" />{t('groupDetail.unmuteMember')}
                          </DropdownMenuItem>
                        {:else}
                          <DropdownMenuItem onclick={() => openMuteDialog(member)}>
                            <MicOff class="mr-2 h-4 w-4" />{t('groupDetail.muteMember')}
                          </DropdownMenuItem>
                        {/if}
                        {#if canChangeRole(member)}
                          <DropdownMenuSeparator />
                          <DropdownMenuItem onclick={() => openRoleDialog(member)}>
                            <UserCog class="mr-2 h-4 w-4" />{t('groupDetail.roleManagement')}
                          </DropdownMenuItem>
                        {/if}
                        <DropdownMenuSeparator />
                        <DropdownMenuItem onclick={() => openRemoveDialog(member)}>
                          <UserMinus class="mr-2 h-4 w-4 text-destructive" />{t('groupDetail.removeMember')}
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  {/if}
                {/snippet}
              </UserCard>
            {/each}
          </div>
          {#if membersPagination.hasMore}
            <div class="flex justify-center pt-2">
              <button
                class="flex items-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
                style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
                onclick={() => loadMembers(false)}
                disabled={membersPagination.loadingMore}
              >
                {#if membersPagination.loadingMore}<Spinner size="sm" />{/if}
                {t('common.loadMore')}
              </button>
            </div>
          {/if}
        {/if}
      {/if}

      <!-- Tab: Applications -->
      {#if canManageGroup && currentTab === 'applications'}
        <div class="space-y-4">
          <div
            class="flex flex-col gap-3 rounded-[var(--radius-lg)] p-4 sm:flex-row sm:items-center sm:justify-between"
            style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
          >
            <div>
              <p class="text-body-lg" style="font-weight: 600; color: var(--text-primary)">
                {t('groupDetail.applicationsTitle')}
              </p>
              <p class="text-meta" style="margin-top: 4px; color: var(--text-secondary)">
                {t('groupDetail.applicationsDesc')}
              </p>
            </div>
            <button
              class="flex items-center gap-1.5 rounded-full px-4 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
              style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
              onclick={() => loadApplications(true)}
              disabled={applicationsPagination.loading}
            >
              <RefreshCw size={14} />{t('common.refresh')}
            </button>
          </div>
          {#if applicationsPagination.loading && applicationsPagination.items.length === 0}
            <div class="space-y-3">
              {#each Array(3) as _, i (i)}
                <div
                  class="rounded-[var(--radius-lg)] p-4 space-y-2"
                  style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
                >
                  <div
                    class="rounded"
                    style="width: 128px; height: 14px; background: var(--border-theme);"
                  ></div>
                  <div
                    class="rounded"
                    style="width: 100%; height: 14px; background: var(--border-theme);"
                  ></div>
                  <div
                    class="rounded"
                    style="width: 144px; height: 12px; background: var(--border-theme);"
                  ></div>
                </div>
              {/each}
            </div>
          {:else if applicationsPagination.items.length === 0}
            <div
              class="rounded-[var(--radius-lg)] py-10 text-center"
              style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
            >
              <p class="text-body-lg" style="color: var(--text-secondary)">{t('groupDetail.noApplications')}</p>
            </div>
          {:else}
            <div class="space-y-3">
              {#each applicationsPagination.items as application (application.application_id)}
                <div
                  class="rounded-[var(--radius-lg)] p-4"
                  style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
                >
                  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                    <div class="min-w-0 space-y-1">
                      <div class="flex items-center gap-2">
                        <Avatar
                          src={application.applicant.avatar_url}
                          name={application.applicant.nickname}
                          alt={application.applicant.nickname}
                          size={32}
                        />
                        <p
                          class="text-body-lg truncate"
                          style="font-weight: 500; color: var(--text-primary)"
                        >
                          {application.applicant.nickname}
                        </p>
                      </div>
                      {#if application.message}
                        <p class="text-body-lg" style="color: var(--text-secondary)">
                          {application.message}
                        </p>
                      {/if}
                      <p class="text-meta" style="color: var(--text-secondary)">
                        {t('groupDetail.applicationTime', { time: formatDateTime(application.created_at) })}
                      </p>
                    </div>
                    <div class="flex flex-wrap gap-2">
                      <button
                        class="text-body rounded-full px-4 py-2 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
                        style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn)"
                        disabled={isApplicationHandling(application.application_id)}
                        onclick={() => handleApplicationDecision(application, 'approve')}
                      >
                        {isApplicationHandling(application.application_id) ? t('groupDetail.processing') : t('common.approve')}
                      </button>
                      <button
                        class="text-body rounded-full px-4 py-2 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
                        style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg)"
                        disabled={isApplicationHandling(application.application_id)}
                        onclick={() => handleApplicationDecision(application, 'reject')}
                      >
                        {t('common.reject')}
                      </button>
                    </div>
                  </div>
                </div>
              {/each}
            </div>
            {#if applicationsPagination.hasMore}
              <div class="flex justify-center pt-2">
                <button
                  class="flex items-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
                  style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
                  onclick={() => loadApplications(false)}
                  disabled={applicationsPagination.loadingMore}
                >
                  {#if applicationsPagination.loadingMore}<Spinner size="sm" />{/if}
                  {t('common.loadMore')}
                </button>
              </div>
            {/if}
          {/if}
        </div>
      {/if}
    {/if}
  </div>
</div>

<!-- Dialogs: kept with neumorphic inner styling -->
<Dialog.Root bind:open={showJoinDialog}>
  <Dialog.Content
    class="sm:max-w-lg border var(--surface-bg) backdrop-blur-md"
    style="border-color: var(--border-theme);"
  >
    <Dialog.Header>
      <Dialog.Title>{t('groupDetail.joinDialogTitle')}</Dialog.Title>
      <Dialog.Description>
        {settings?.join_method === 'approval'
          ? t('groupDetail.joinDialogDescApproval', { method: getJoinMethodLabel(settings?.join_method) })
          : t('groupDetail.joinDialogDescOpen', { method: getJoinMethodLabel(settings?.join_method) })}
      </Dialog.Description>
    </Dialog.Header>
    <div class="space-y-4 py-4">
      {#if group?.has_password}
        <div
          class="space-y-2 rounded-[var(--radius-md)] p-4"
          style="border: 1px solid var(--border-theme);"
        >
          <label
            for="group-password"
            class="text-body"
            style="font-weight: 500; color: var(--text-primary)">{t('groupDetail.joinDialogPasswordLabel')}</label
          >
          <input
            id="group-password"
            type="password"
            placeholder={t('groupDetail.joinDialogPasswordPlaceholder')}
            bind:value={joinPassword}
            maxlength={GROUP_PASSWORD_MAX}
            class="w-full rounded-full py-2.5 px-4 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
            style="background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-primary);"
          />
        </div>
      {/if}
      {#if settings?.join_method === 'approval'}
        <div
          class="space-y-2 rounded-[var(--radius-md)] p-4"
          style="border: 1px solid var(--border-theme);"
        >
          <label
            for="join-message"
            class="text-body"
            style="font-weight: 500; color: var(--text-primary)">{t('groupDetail.joinDialogMessageLabel')}</label
          >
          <Textarea
            id="join-message"
            size="md"
            placeholder={t('groupDetail.joinDialogMessagePlaceholder')}
            bind:value={joinMessage}
            maxlength={GROUP_JOIN_MESSAGE_MAX}
          />
        </div>
      {/if}
    </div>
    <Dialog.Footer>
      <button
        class="rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
        style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
        onclick={() => (showJoinDialog = false)}>{t('common.cancel')}</button
      >
      <button
        class="flex items-center justify-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
        style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
        onclick={handleJoin}
        disabled={isJoining}
      >
        {#if isJoining}<Spinner size="sm" />{/if}
        {t('groupDetail.confirmJoin')}
      </button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={showAnnouncementDialog}>
  <Dialog.Content
    class="sm:max-w-xl border var(--surface-bg) backdrop-blur-md"
    style="border-color: var(--border-theme);"
  >
    <Dialog.Header>
      <Dialog.Title>{editingAnnouncement ? t('groupDetail.announcementDialogTitleEdit') : t('groupDetail.announcementDialogTitleCreate')}</Dialog.Title>
      <Dialog.Description>{t('groupDetail.announcementDialogDesc')}</Dialog.Description>
    </Dialog.Header>
    <div class="space-y-4 py-4">
      <div
        class="space-y-2 rounded-[var(--radius-md)] p-4"
        style="border: 1px solid var(--border-theme);"
      >
        <label
          for="announcement-title"
          class="text-body"
          style="font-weight: 500; color: var(--text-primary)">{t('groupDetail.announcementTitleLabel')}</label
        >
        <input
          id="announcement-title"
          placeholder={t('groupDetail.announcementTitlePlaceholder')}
          bind:value={announcementTitle}
          maxlength={ANNOUNCEMENT_TITLE_MAX}
          class="w-full rounded-full py-2.5 px-4 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
          style="background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-primary);"
        />
      </div>
      <div
        class="space-y-2 rounded-[var(--radius-md)] p-4"
        style="border: 1px solid var(--border-theme);"
      >
        <label
          for="announcement-content"
          class="text-body"
          style="font-weight: 500; color: var(--text-primary)">{t('groupDetail.announcementContentLabel')}</label
        >
        <Textarea
          id="announcement-content"
          size="lg"
          placeholder={t('groupDetail.announcementContentPlaceholder')}
          bind:value={announcementContent}
          maxlength={ANNOUNCEMENT_CONTENT_MAX}
        />
      </div>
      <div
        class="flex items-center justify-between rounded-[var(--radius-md)] px-4 py-3"
        style="border: 1px solid var(--border-theme);"
      >
        <div>
          <p class="text-body-lg" style="font-weight: 500; color: var(--text-primary)">{t('groupDetail.pinAnnouncement')}</p>
          <p class="text-meta" style="color: var(--text-secondary)">{t('groupDetail.pinDesc')}</p>
        </div>
        <button
          class="rounded-full px-4 py-2 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
          style={announcementPinned
            ? 'background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);'
            : 'border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);'}
          onclick={() => (announcementPinned = !announcementPinned)}
        >
          {announcementPinned ? t('groupDetail.pinnedStatus') : t('groupDetail.unpinnedStatus')}
        </button>
      </div>
    </div>
    <Dialog.Footer>
      <button
        class="rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
        style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
        onclick={() => (showAnnouncementDialog = false)}>{t('common.cancel')}</button
      >
      <button
        class="flex items-center justify-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
        style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
        onclick={submitAnnouncement}
        disabled={announcementSubmitting}
      >
        {#if announcementSubmitting}<Spinner size="sm" />{/if}
        {editingAnnouncement ? t('groupDetail.saveChanges') : t('groupDetail.publishAnnouncement')}
      </button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={showAnnouncementDeleteDialog}>
  <Dialog.Content
    class="sm:max-w-md border var(--surface-bg) backdrop-blur-md"
    style="border-color: var(--border-theme);"
  >
    <Dialog.Header>
      <Dialog.Title>{t('groupDetail.deleteDialogTitle')}</Dialog.Title>
      <Dialog.Description>{t('groupDetail.deleteDialogDesc')}</Dialog.Description>
    </Dialog.Header>
    <div
      class="text-body-lg rounded-[var(--radius-md)] px-4 py-3"
      style="border: 1px solid var(--border-theme); color: var(--text-secondary)"
    >
      {announcementToDelete?.title ?? t('groupDetail.deleteDialogFallback')}
    </div>
    <Dialog.Footer>
      <button
        class="rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
        style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
        onclick={() => (showAnnouncementDeleteDialog = false)}>{t('common.cancel')}</button
      >
      <button
        class="rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
        style="background: var(--theme-error); color: var(--text-on-accent); box-shadow: 0 4px 14px var(--theme-error-border);"
        onclick={handleDeleteAnnouncement}>{t('groupDetail.confirmDelete')}</button
      >
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={showMuteDialog}>
  <Dialog.Content
    class="sm:max-w-md border var(--surface-bg) backdrop-blur-md"
    style="border-color: var(--border-theme);"
  >
    <Dialog.Header>
      <Dialog.Title>{t('groupDetail.muteDialogTitle')}</Dialog.Title>
      <Dialog.Description>{t('groupDetail.muteDialogDesc')}</Dialog.Description>
    </Dialog.Header>
    <div class="space-y-4 py-2">
      <div class="text-body-lg" style="color: var(--text-secondary)">
        {t('groupDetail.muteDialogMember', { nickname: muteTarget?.user.nickname ?? '-' })}
      </div>
      <div class="grid grid-cols-2 gap-2">
        {#each mutePresets as preset (preset.value)}
          <button
            class="rounded-full py-2 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
            style={Number(muteDurationMinutes) === preset.value
              ? 'background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);'
              : 'border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);'}
            onclick={() => (muteDurationMinutes = String(preset.value))}>{preset.label}</button
          >
        {/each}
      </div>
      <div class="space-y-2">
        <label
          for="mute-duration"
          class="text-body"
          style="font-weight: 500; color: var(--text-primary)">{t('groupDetail.muteDurationLabel')}</label
        >
        <input
          id="mute-duration"
          type="number"
          min={MUTE_DURATION_MIN}
          max={MUTE_DURATION_MAX}
          bind:value={muteDurationMinutes}
          class="w-full rounded-full py-2.5 px-4 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
          style="background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-primary);"
        />
      </div>
    </div>
    <Dialog.Footer>
      <button
        class="rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
        style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
        onclick={() => (showMuteDialog = false)}>{t('common.cancel')}</button
      >
      <button
        class="flex items-center justify-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
        style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
        onclick={handleMuteMember}
        disabled={muteSubmitting}
      >
        {#if muteSubmitting}<Spinner size="sm" />{/if}
        {t('groupDetail.confirmMute')}
      </button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={showRoleDialog}>
  <Dialog.Content
    class="sm:max-w-md border var(--surface-bg) backdrop-blur-md"
    style="border-color: var(--border-theme);"
  >
    <Dialog.Header>
      <Dialog.Title>{t('groupDetail.roleDialogTitle')}</Dialog.Title>
      <Dialog.Description>{t('groupDetail.roleDialogDesc')}</Dialog.Description>
    </Dialog.Header>
    <div class="space-y-4 py-2">
      <div class="text-body-lg" style="color: var(--text-secondary)">
        {t('groupDetail.roleDialogMember', { nickname: roleTarget?.user.nickname ?? '-' })}
      </div>
      <div class="grid gap-2 sm:grid-cols-2">
        <button
          class="rounded-full py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
          style={roleTarget?.role === 'admin'
            ? 'background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);'
            : 'border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);'}
          disabled={roleTarget?.role === 'admin' || roleSubmitting}
          onclick={() => handleSetRole('admin')}>{t('groupDetail.setAdmin')}</button
        >
        <button
          class="rounded-full py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
          style={roleTarget?.role === 'member'
            ? 'background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);'
            : 'border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);'}
          disabled={roleTarget?.role === 'member' || roleSubmitting}
          onclick={() => handleSetRole('member')}>{t('groupDetail.setMember')}</button
        >
      </div>
    </div>
    <Dialog.Footer>
      <button
        class="rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
        style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
        onclick={() => (showRoleDialog = false)}>{t('common.cancel')}</button
      >
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={showRemoveDialog}>
  <Dialog.Content
    class="sm:max-w-md border var(--surface-bg) backdrop-blur-md"
    style="border-color: var(--border-theme);"
  >
    <Dialog.Header>
      <Dialog.Title>{t('groupDetail.removeDialogTitle')}</Dialog.Title>
      <Dialog.Description>{t('groupDetail.removeDialogDesc')}</Dialog.Description>
    </Dialog.Header>
    <div
      class="text-body-lg rounded-[var(--radius-md)] px-4 py-3"
      style="border: 1px solid var(--border-theme); color: var(--text-secondary)"
    >
      {removeTarget?.user.nickname ?? '-'}
    </div>
    <Dialog.Footer>
      <button
        class="rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
        style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
        onclick={() => (showRemoveDialog = false)}>{t('common.cancel')}</button
      >
      <button
        class="flex items-center justify-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
        style="background: var(--theme-error); color: var(--text-on-accent); box-shadow: 0 4px 14px var(--theme-error-border);"
        onclick={handleRemoveMember}
        disabled={removeSubmitting}
      >
        {#if removeSubmitting}<Spinner size="sm" />{/if}
        {t('groupDetail.confirmRemove')}
      </button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={showLeaveDialog}>
  <Dialog.Content
    class="sm:max-w-md border var(--surface-bg) backdrop-blur-md"
    style="border-color: var(--border-theme);"
  >
    <Dialog.Header>
      <Dialog.Title>{t('groupDetail.leaveDialogTitle')}</Dialog.Title>
      <Dialog.Description>{t('groupDetail.leaveDialogDesc')}</Dialog.Description>
    </Dialog.Header>
    <div
      class="rounded-[var(--radius-md)] px-4 py-3"
      style="border: 1px solid var(--border-theme);"
    >
      <div class="text-body-lg" style="font-weight: 500; color: var(--text-primary)">
        {group?.group.name ?? t('groupDetail.leaveDialogFallback')}
      </div>
      <div class="text-meta" style="margin-top: 4px; color: var(--text-secondary)">
        {t('groupDetail.leaveDialogMemberCount', { count: group?.group.members_count ?? 0 })}
      </div>
    </div>
    <Dialog.Footer>
      <button
        class="rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
        style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
        onclick={() => (showLeaveDialog = false)}>{t('common.cancel')}</button
      >
      <button
        class="flex items-center justify-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
        style="background: var(--theme-error); color: var(--text-on-accent); box-shadow: 0 4px 14px var(--theme-error-border);"
        onclick={handleLeave}
        disabled={isLeaving}
      >
        {#if isLeaving}<Spinner size="sm" />{/if}
        {t('groupDetail.confirmLeave')}
      </button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
