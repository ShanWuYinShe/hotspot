<script lang="ts">
	import { UserPlus, MoreHorizontal } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import { goto } from '$app/navigation';
	import { ROUTES } from '@/lib/routes';
	import type { GroupMember } from '@/lib/group-data.svelte';

	// MembersList —— 成员列表 + （管理者）添加成员入口；行点击进资料，
	// 管理按钮交由父级 MemberActionSheet 处理。
	// 分页：大群走 cursor 分页，尾部 LoadMoreButton 由父级 group-data 驱动
	//（审查 M5：此前一次全量拉取渲染）。

	/** 后端 role 值 → 中文展示（未收录角色回退原值）。 */
	const ROLE_LABELS: Record<string, string> = {
		owner: '群主',
		admin: '管理员',
		member: '成员'
	};
	const roleLabel = (role: string) => ROLE_LABELS[role] ?? role;

	let {
		members,
		onlineMemberIds,
		canManageMembers,
		canManageMember,
		hasMore = false,
		loadingMore = false,
		onInvite,
		onManage,
		onLoadMore
	}: {
		members: GroupMember[];
		onlineMemberIds: Set<string>;
		canManageMembers: boolean;
		canManageMember: (m: GroupMember) => boolean;
		hasMore?: boolean;
		loadingMore?: boolean;
		onInvite: () => void;
		onManage: (m: GroupMember) => void;
		onLoadMore?: () => void;
	} = $props();
</script>

{#if canManageMembers}
	<div class="mt-6 flex justify-end">
		<button type="button" onclick={onInvite} class="hover-btn hit-slop flex items-center gap-1 rounded-full border border-border-default px-3.5 py-2 text-sm text-text-primary focus-ring hover:border-border-strong">
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
				<Avatar src={member.avatar} name={member.name} size="lg" />
				{#if onlineMemberIds.has(member.userId)}
					<OnlineDot size="md" />
				{/if}
			</div>
			<span class="min-w-0 flex-1">
				<span class="block text-sm text-text-primary">{member.name}</span>
				<span class="text-xs text-text-secondary">{roleLabel(member.role)}</span>
			</span>
			{#if canManageMember(member)}
				<button
					type="button"
					onclick={(e) => { e.stopPropagation(); onManage(member); }}
					class="hit-slop hover-btn scale-btn flex h-8 w-8 items-center justify-center rounded-full text-text-secondary outline-none hover:bg-interactive-hover focus-visible:ring-2 focus-visible:ring-accent-default"
					aria-label={`管理 ${member.name}`}
				>
					<MoreHorizontal size={15} strokeWidth={2} />
				</button>
			{/if}
		</ListItem>
	{/each}
</ListContainer>
{#if hasMore && onLoadMore}
	<div class="mt-4">
		<LoadMoreButton label="加载更多成员" loading={loadingMore} onLoad={onLoadMore} />
	</div>
{/if}
