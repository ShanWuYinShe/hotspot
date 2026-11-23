<script lang="ts">
	import { UserPlus, MoreHorizontal } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import InfiniteSentinel from '@/components/list/InfiniteSentinel.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import Badge from '@/components/data-display/Badge.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import Button from '@/components/ui/button.svelte';
	import { goto } from '$app/navigation';
	import { ROUTES } from '$lib/routes';
	import type { GroupMember } from '@/lib/group-data.svelte';
	import { searchGroupMembers } from '@/api/domains/groups';
	import type { GroupMemberItem } from '@/api/types';

	// MembersList —— 成员列表 + （管理者）添加成员入口；行点击进资料，
	// 管理按钮交由父级 MemberActionSheet 处理。
	// 分页：大群走 cursor 分页，尾部 InfiniteSentinel 滚到底自动续载（由父级 group-data 驱动）
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
		groupId,
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
		/** 当前群 id：成员搜索（groups/members/search）使用。 */
		groupId: string;
		canManageMembers: boolean;
		canManageMember: (m: GroupMember) => boolean;
		hasMore?: boolean;
		loadingMore?: boolean;
		onInvite: () => void;
		onManage: (m: GroupMember) => void;
		onLoadMore?: () => void;
	} = $props();
	// 成员搜索（契约 groups/members/search 支持 keyword）：350ms 防抖，搜索模式独立分页/空态，
	// 清空关键词恢复父级分页列表；搜索只按昵称命中，不影响角色徽章与行点击进资料。
	let query = $state('');
	let searchItems = $state<GroupMember[]>([]);
	let searchCursor = $state<string | null>(null);
	let searchLoading = $state(false);
	let searchLoadingMore = $state(false);
	let searchError = $state('');
	let searchSeq = 0;

	const shownMembers = $derived(query.trim() ? searchItems : members);
	const shownCount = $derived(query.trim() ? searchItems.length : (total ?? members.length));

	function mapMember(item: GroupMemberItem, index: number): GroupMember {
		return {
			id: index + 1,
			userId: item.user?.user_id ?? '',
			name: item.user?.nickname ?? '',
			role: item.role ?? 'member',
			avatar: item.user?.avatar_url ?? '',
			presence: item.presence_status === 'online' ? 'online' : null
		};
	}

	async function runSearch(keyword: string) {
		const seq = ++searchSeq;
		searchLoading = true;
		searchError = '';
		try {
			const res = await searchGroupMembers({ group_id: groupId, keyword, pagination: { limit: 20 } });
			if (seq !== searchSeq) return;
			if (res.code === 0) {
				searchItems = (res.data?.items ?? []).map(mapMember);
				searchCursor = res.data?.next_cursor ?? null;
			} else {
				searchError = res.message || '搜索失败';
				searchItems = [];
				searchCursor = null;
			}
		} catch {
			if (seq === searchSeq) {
				searchError = '搜索失败';
				searchItems = [];
				searchCursor = null;
			}
		} finally {
			if (seq === searchSeq) searchLoading = false;
		}
	}

	$effect(() => {
		const keyword = query.trim();
		if (!keyword) {
			searchSeq++;
			searchItems = [];
			searchCursor = null;
			searchError = '';
			searchLoading = false;
			return;
		}
		const timer = setTimeout(() => void runSearch(keyword), 350);
		return () => clearTimeout(timer);
	});

	async function loadMoreSearch() {
		const cursor = searchCursor;
		if (!cursor || searchLoadingMore) return;
		searchLoadingMore = true;
		try {
			const res = await searchGroupMembers({
				group_id: groupId,
				keyword: query.trim(),
				pagination: { limit: 20, cursor }
			});
			if (res.code === 0) {
				const start = searchItems.length;
				searchItems = [...searchItems, ...(res.data?.items ?? []).map((item, i) => mapMember(item, start + i))];
				searchCursor = res.data?.next_cursor ?? null;
			} else {
				searchError = res.message || '加载更多失败';
			}
		} catch {
			searchError = '加载更多失败';
		} finally {
			searchLoadingMore = false;
		}
	}
</script>

<div class="mt-6">
	<div class="flex items-end justify-between gap-3">
		<SectionHead title="成员" count={shownCount} class="mb-0 min-w-0 flex-1" />
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
	<div class="mt-3">
		<SearchInput
			value={query}
			onChange={(value) => (query = value)}
			placeholder="搜索成员…"
			ariaLabel="搜索群成员"
		/>
	</div>
	{#if query.trim() && searchLoading}
		<InlineLoading label="正在搜索成员…" class="mt-4" />
	{:else if query.trim() && searchError}
		<div class="mt-4 flex flex-col items-center gap-3 py-8">
			<p class="text-sm text-text-secondary">{searchError}</p>
			<Button type="button" variant="soft" size="sm" onclick={() => void runSearch(query.trim())}>
				重试
			</Button>
		</div>
	{:else if query.trim() && shownMembers.length === 0}
		<EmptyState title="没有找到成员" description="换个昵称关键词试试，或清空搜索。" class="mt-4" bare />
	{:else}
	<!-- 开放页面区域 → ListContainer bordered（内容软岛：surface-island + shadow-xs + rounded-lg(20) + p-2，
	     行透明、行间离岛边 8）；列表行不设独立边框，禁止再手写 divide-y 分隔线。
	     标题行→列表 12（同一区块内），列表与子 Tab 栏的 24 由标题行 mt-6 承担。 -->
	<ListContainer bordered class="mt-3">
		{#each shownMembers as member (member.userId || member.id)}
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
					<ActionIconButton
						icon={MoreHorizontal}
						label={`管理 ${member.name}`}
						size={32}
						variant="ghost"
						tone="muted"
						onClick={() => onManage(member)}
					/>
				{/if}
			</ListItem>
		{/each}
	</ListContainer>
	{#if query.trim()}
		{#if searchCursor}
			<div class="mt-4 flex justify-center">
				<Button
					type="button"
					variant="soft"
					size="sm"
					loading={searchLoadingMore}
					onclick={() => void loadMoreSearch()}
				>
					{searchLoadingMore ? '加载中…' : '加载更多成员'}
				</Button>
			</div>
		{:else}
			<ListEnd />
		{/if}
	{:else if hasMore && onLoadMore}
		<div class="mt-4">
			<!-- 大群成员列表滚到底自动续载，不再需要手动点按钮 -->
			<InfiniteSentinel hasMore={hasMore} loading={loadingMore} onLoadMore={onLoadMore} />
		</div>
	{:else if onLoadMore}
		<!-- 四态契约「到底」：无收尾提示时用户无法区分「已到底」与「加载卡住」
			（与 ListContainer 消费方一致，参见 data-display/ListEnd） -->
		<ListEnd />
	{/if}
	{/if}
</div>