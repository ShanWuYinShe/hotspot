<script lang="ts">
	import { UserPlus, MoreHorizontal } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import Badge from '@/components/data-display/Badge.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import Button from '@/components/ui/button.svelte';
	import { goto } from '$app/navigation';
	import { ROUTES } from '$lib/routes';
	import type { GroupMember } from '@/lib/group-data.svelte';

	// MembersList —— 成员列表 + （管理者）添加成员入口；行点击进资料，
	// 管理按钮交由父级 MemberActionSheet 处理。
	// 分页：大群走 cursor 分页，尾部 LoadMoreButton 由父级 group-data 驱动
	//（审查 M5：此前一次全量拉取渲染）。
	//
	// R13：标题行改用全站 SectionHead（标题 + 计数 + 右侧动作），行语言统一为
	// 昵称 14px medium / 角色 11px（群主·管理员用 accent 淡底徽章，与普通成员区分）。

	/** 后端 role 值 → 中文展示（未收录角色回退原值）。 */
	const ROLE_LABELS: Record<string, string> = {
		owner: '群主',
		admin: '管理员',
		member: '成员'
	};
	const roleLabel = (role: string) => ROLE_LABELS[role] ?? role;

	let {
		members,
		total,
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
		/** 群成员总数（分页只加载首页时用于标题计数，缺省回退已加载条数）。 */
		total?: number;
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

<div class="mt-6">
	<div class="flex items-end justify-between gap-3">
		<SectionHead title="成员" count={total ?? members.length} class="mb-0 min-w-0 flex-1" />
		{#if canManageMembers}
			<Button
				type="button"
				variant="soft"
				size="sm"
				class="mb-1 shrink-0"
				onclick={onInvite}
			>
				<UserPlus size={13} strokeWidth={1.75} />
				添加成员
			</Button>
		{/if}
	</div>
	<!-- 空旷内容区 → List bordered（有边框列表契约）；列表行不设独立边框。
	     标题行→列表 12（同一区块内），列表与子 Tab 栏的 24 由标题行 mt-6 承担。 -->
	<ListContainer class="mt-3 divide-y divide-border-subtle/30">
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
					<span class="block truncate text-md font-medium text-text-primary">{member.name}</span>
					{#if member.role === 'owner' || member.role === 'admin'}
						<Badge tone="accent" class="mt-0.5">{roleLabel(member.role)}</Badge>
					{:else}
						<span class="mt-0.5 block text-xs text-text-secondary">{roleLabel(member.role)}</span>
					{/if}
				</span>
				{#if canManageMember(member)}
					<button
						type="button"
						onclick={(e) => { e.stopPropagation(); onManage(member); }}
						class="hit-slop hover-btn scale-btn flex size-8 shrink-0 items-center justify-center rounded-full text-text-secondary focus-ring hover:bg-interactive-hover"
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
	{:else if onLoadMore}
		<!-- 四态契约「到底」：无收尾提示时用户无法区分「已到底」与「加载卡住」
			（与 ListContainer 消费方一致，参见 data-display/ListEnd） -->
		<ListEnd />
	{/if}
</div>
