<script lang="ts">
	import { UserPlus, MoreHorizontal } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import { goto } from '$app/navigation';
	import { ROUTES } from '@/lib/routes';
	import type { GroupMember } from '@/lib/group-data.svelte';

	// MembersList —— 成员列表 + （管理者）添加成员入口；行点击进资料，
	// 管理按钮交由父级 MemberActionSheet 处理。
	let {
		members,
		onlineMemberIds,
		canManageMembers,
		canManageMember,
		onInvite,
		onManage
	}: {
		members: GroupMember[];
		onlineMemberIds: Set<string>;
		canManageMembers: boolean;
		canManageMember: (m: GroupMember) => boolean;
		onInvite: () => void;
		onManage: (m: GroupMember) => void;
	} = $props();
</script>

{#if canManageMembers}
	<div class="mt-6 flex justify-end">
		<button type="button" onclick={onInvite} class="hover-btn flex items-center gap-1 rounded-full border border-border-default px-3.5 py-1.5 text-sm text-text-primary focus-ring">
			<UserPlus size={12} strokeWidth={1.75} />
			添加成员
		</button>
	</div>
{/if}
<!-- 空旷内容区 → List bordered（有边框列表契约）；列表行不设独立边框 -->
<ListContainer bordered class="mt-4">
	{#each members as member (member.id)}
		<ListItem
			onClick={() => {
				if (member.userId) void goto(ROUTES.user(member.userId));
			}}
			ariaLabel={`查看 ${member.name} 的资料`}
			class="flex list-row-standard items-center gap-3 px-3.5 py-3"
		>
			<div class="relative">
				<Avatar src={member.avatar} name={member.name} size={40} />
				{#if onlineMemberIds.has(member.userId)}
					<span class="absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full border-2 border-surface-primary bg-status-online"></span>
				{/if}
			</div>
			<span class="min-w-0 flex-1">
				<span class="block text-sm text-text-primary">{member.name}</span>
				<span class="text-xs text-text-secondary">{member.role}</span>
			</span>
			<span class="text-xs text-text-secondary">查看资料</span>
			{#if canManageMember(member)}
				<button
					type="button"
					onclick={(e) => { e.stopPropagation(); onManage(member); }}
					class="hit-slop flex h-7 w-7 items-center justify-center rounded-full text-text-secondary outline-none hover:bg-interactive-hover hover-btn scale-btn focus-visible:ring-2 focus-visible:ring-accent-default"
					aria-label={`管理 ${member.name}`}
				>
					<MoreHorizontal size={15} strokeWidth={2} />
				</button>
			{/if}
		</ListItem>
	{/each}
</ListContainer>
