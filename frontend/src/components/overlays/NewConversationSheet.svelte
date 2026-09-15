<script lang="ts">
	import { goto } from '$app/navigation';
	import { UserPlus, MessageCircle, Users } from '@lucide/svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import SheetShell from '@/components/overlays/SheetShell.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import { useRelationships } from '@/lib/profile-data.svelte';
	import { ROUTES } from '@/lib/routes';

	// 新会话底部 Sheet：开始私聊（联系人）/ 添加好友（去附近发现）/ 创建群聊入口。
	// 联系人 = 好友 ∪ 关注 ∪ 粉丝 去重（useRelationships 统一数据源），
	// 替代原硬编码空数组（「开始私聊」tab 永远空态的死 UI）。

	type Tab = 'add' | 'chat';

	let {
		defaultTab = 'chat',
		onClose,
		onNewChat,
		onCreateGroup
	}: {
		defaultTab?: Tab;
		onClose: () => void;
		onNewChat?: (contactId: string) => void;
		onCreateGroup?: () => void;
	} = $props();

	const rel = useRelationships();
	// 好友优先，其次关注/粉丝；按 id 去重
	const CONTACTS = $derived.by(() => {
		const seen = new Set<string>();
		const list: { id: string; name: string; avatar: string }[] = [];
		for (const f of [...rel.friends, ...rel.following, ...rel.followers]) {
			if (seen.has(f.id)) continue;
			seen.add(f.id);
			list.push({ id: f.id, name: f.name, avatar: f.avatar });
		}
		return list;
	});

	// defaultTab 仅在打开时生效（此后切换走 tab 按钮），有意只读取一次初始值。
	// svelte-ignore state_referenced_locally
	let tab = $state<Tab>(defaultTab);
	let searchQ = $state('');

	const filteredContacts = $derived(CONTACTS.filter((c) => !searchQ || c.name.toLowerCase().includes(searchQ.toLowerCase())));
</script>

<SheetShell title={tab === 'add' ? '添加好友' : '新建会话'} {onClose}>
	<!-- Tab switcher — 统一 TabBar 原语（禁止手写 tab 变体），保留原回调 -->
	<div class="shrink-0 px-5 pt-3 pb-2">
		<TabBar
			variant="segmented"
			items={[
				{ id: 'chat', label: '开始私聊', icon: MessageCircle },
				{ id: 'add', label: '添加好友', icon: UserPlus }
			]}
			value={tab}
			onChange={(id) => {
				tab = id as Tab;
				searchQ = '';
			}}
		/>
	</div>

	<!-- Search bar — 复用 SearchInput 统一原语 -->
	<div class="px-5 pb-3 shrink-0">
		<SearchInput
			value={searchQ}
			onChange={(v) => (searchQ = v)}
			class="px-3.5 py-2.5"
			placeholder={tab === 'add' ? '搜索用户名或手机号…' : '搜索联系人…'}
		/>
	</div>

	<!-- Content -->
	<div class="flex-1 overflow-y-auto no-scrollbar px-5 pb-6">
		{#if tab === 'add'}
			<!-- 添加好友：无站内推荐数据源，直接引导到附近的人发现页 -->
			<div class="animate-fade-in flex flex-col gap-2">
				<p class="eyebrow mb-2">添加好友</p>
				<EmptyState
					bare
					title={searchQ.trim() ? '站内暂不支持搜索加人' : '去附近认识新朋友'}
					description={searchQ.trim() ? '换个关键词试试，或到附近的人里逛逛' : '发现 5km 内有趣的人，关注后即可发起私聊'}
				>
					{#snippet action()}
						<button
							type="button"
							class="hover-btn rounded-full border border-border-default px-4 py-2 text-xs text-text-primary focus-ring hover:border-border-strong"
							onclick={() => {
								onClose();
								void goto(ROUTES.nearbyPeople);
							}}
						>
							去附近看看
						</button>
					{/snippet}
				</EmptyState>
			</div>
		{:else if rel.loading}
			<InlineLoading label="加载联系人…" />
		{:else}
			<div class="animate-fade-in flex flex-col gap-2">
				<!-- Create group chat row -->
				<ListItem
					tone="card"
					onClick={() => {
						onClose();
						onCreateGroup?.();
					}}
					ariaLabel="创建群聊"
					class="mb-3 flex items-center gap-3.5 p-3"
				>
					<div class="flex items-center justify-center rounded-full shrink-0 bg-accent-default size-11">
						<Users class="size-5 text-text-on-accent" strokeWidth={1.75} />
					</div>
					<div>
						<p class="text-text-primary text-sm font-medium">创建群聊</p>
						<p class="text-text-secondary text-xs">邀请好友一起聊</p>
					</div>
				</ListItem>

				<p class="eyebrow mb-2">联系人</p>
				{#if filteredContacts.length > 0}
					<!-- 联系人集合用一张 bordered 列表卡（行间 1px 分隔），不用逐行独立卡 -->
					<ListContainer bordered ariaLabel="联系人">
						{#each filteredContacts as c (c.id)}
							<ListItem
								onClick={() => {
									onNewChat?.(c.id);
									onClose();
								}}
								ariaLabel={`与 ${c.name} 开始私聊`}
								class="flex items-center gap-3.5 px-4 py-3"
							>
								<Avatar src={c.avatar} name={c.name} size="lg" />
								<div class="flex-1 min-w-0">
									<p class="text-text-primary text-sm">{c.name}</p>
									<p class="text-text-secondary text-xs">发起新会话</p>
								</div>
								<MessageCircle class="size-4 shrink-0 text-text-secondary" strokeWidth={1.5} />
							</ListItem>
						{/each}
					</ListContainer>
				{:else}
					<EmptyState
						bare
						title={searchQ.trim() ? '未找到联系人' : '暂无联系人'}
						description={searchQ.trim() ? '换个关键词试试' : '先添加好友，即可在这里发起私聊'}
					>
						{#snippet action()}
							<button
								type="button"
								class="hover-btn rounded-full border border-border-default px-4 py-2 text-xs text-text-primary focus-ring hover:border-border-strong"
								onclick={() => {
									tab = 'add';
									searchQ = '';
								}}
							>
								去添加好友
							</button>
						{/snippet}
					</EmptyState>
				{/if}
			</div>
		{/if}
	</div>
</SheetShell>
