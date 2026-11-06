<script lang="ts">
	import { Flag, LogOut, X } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import ReportDialog from '@/components/overlays/ReportDialog.svelte';
	import Button from '@/components/ui/button.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import GroupAnnouncementDialog from '@/components/groups/GroupAnnouncementDialog.svelte';
	import MemberActionSheet from '@/components/groups/MemberActionSheet.svelte';
	import InviteMembersSheet from '@/components/groups/InviteMembersSheet.svelte';
	import GroupSettingsSheet from '@/components/groups/GroupSettingsSheet.svelte';
	import type { MemberAction } from '@/components/groups/MemberActionSheet.svelte';
	import type { UpdateGroupRequest } from '@/api/types';

	let {
		groupId,
		groupName,
		// 删除公告
		deleteAnnounceId,
		announcementDeleting,
		onCancelDeleteAnnounce,
		onConfirmDeleteAnnounce,
		// 退出群组
		leaveConfirmOpen,
		onCancelLeave,
		onConfirmLeave,
		// 更多菜单与举报
		moreOpen = $bindable(false),
		canReportGroup,
		reportOpen = $bindable(false),
		// 公告弹窗
		announceOpen = $bindable(false),
		announceMode,
		announceEditing,
		onSubmitAnnounce,
		// 成员管理操作
		actionMember = $bindable(null),
		memberActionPending,
		onMemberAction,
		// 邀请
		inviteOpen = $bindable(false),
		onInvited,
		// 设置
		settingsOpen = $bindable(false),
		rawGroup,
		isOwner,
		members,
		onSubmitSettings,
		onTransferOwnership,
		onDismissGroup
	}: {
		groupId: string;
		groupName: string;
		deleteAnnounceId: string | null;
		announcementDeleting: boolean;
		onCancelDeleteAnnounce: () => void;
		onConfirmDeleteAnnounce: () => void;
		leaveConfirmOpen: boolean;
		onCancelLeave: () => void;
		onConfirmLeave: () => void;
		moreOpen?: boolean;
		canReportGroup: boolean;
		reportOpen?: boolean;
		announceOpen?: boolean;
		announceMode: 'create' | 'edit';
		announceEditing: { id: string; title: string; content: string; isPinned: boolean } | null;
		onSubmitAnnounce: (title: string, content: string, isPinned: boolean) => Promise<boolean>;
		actionMember?: { userId: string; name: string; role: string; mutedUntil?: string } | null;
		memberActionPending: boolean;
		onMemberAction: (action: MemberAction, durationMinutes?: number) => void;
		inviteOpen?: boolean;
		onInvited: () => void;
		settingsOpen?: boolean;
		rawGroup: any;
		isOwner: boolean;
		members: any[];
		onSubmitSettings: (patch: Omit<UpdateGroupRequest, 'group_id'>) => Promise<boolean>;
		onTransferOwnership: (userId: string) => Promise<any>;
		onDismissGroup: () => Promise<any>;
	} = $props();
</script>

<ConfirmDialog
	open={deleteAnnounceId !== null}
	title="删除这条公告？"
	description="删除后无法恢复。"
	confirmLabel="确认删除"
	loading={announcementDeleting}
	onCancel={onCancelDeleteAnnounce}
	onConfirm={onConfirmDeleteAnnounce}
/>

<ConfirmDialog
	open={leaveConfirmOpen}
	title="退出群组「{groupName}」？"
	description="退出后将不再接收该群的聊天与公告。"
	icon={LogOut}
	confirmLabel="确认退出"
	onCancel={onCancelLeave}
	onConfirm={onConfirmLeave}
/>

<!-- 更多操作菜单（bottom sheet，对齐用户详情页「更多操作」交互模式） -->
<Dialog.Root open={moreOpen} onOpenChange={(next) => (next ? undefined : (moreOpen = false))}>
	<Dialog.Content side="bottom" floating class="p-5" aria-label={`${groupName} 的更多操作`}>
		<div class="flex items-center justify-between">
			<div>
				<p class="eyebrow">更多操作</p>
				<p class="mt-1 text-md text-text-primary">{groupName}</p>
			</div>
			<ActionIconButton icon={X} label="关闭" onClick={() => (moreOpen = false)} />
		</div>
		{#if canReportGroup}
			<Button
				type="button"
				variant="ghost"
				class="mt-4 w-full text-status-danger hover:text-text-status-danger-on-soft hover:bg-danger-bg"
				onclick={() => {
					moreOpen = false;
					reportOpen = true;
				}}
			>
				<Flag size={14} />
				举报群组
			</Button>
		{/if}
	</Dialog.Content>
</Dialog.Root>

<ReportDialog
	open={reportOpen}
	targetType="group"
	targetId={groupId}
	targetLabel="该群组"
	onClose={() => (reportOpen = false)}
/>

<GroupAnnouncementDialog
	open={announceOpen}
	mode={announceMode}
	initial={announceEditing ?? undefined}
	onClose={() => (announceOpen = false)}
	onSubmit={onSubmitAnnounce}
/>

<MemberActionSheet
	open={actionMember !== null}
	member={actionMember}
	loading={memberActionPending}
	onClose={() => (actionMember = null)}
	onSelect={onMemberAction}
/>

<InviteMembersSheet
	open={inviteOpen}
	groupId={groupId}
	onClose={() => (inviteOpen = false)}
	onInvited={onInvited}
/>

<GroupSettingsSheet
	open={settingsOpen}
	initial={rawGroup}
	{isOwner}
	{members}
	onClose={() => (settingsOpen = false)}
	onSubmit={onSubmitSettings}
	onTransferOwnership={onTransferOwnership}
	onDismiss={onDismissGroup}
/>
