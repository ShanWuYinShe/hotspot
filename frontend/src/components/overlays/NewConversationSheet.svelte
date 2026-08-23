<script lang="ts">
	import { goto } from '$app/navigation';
	import { UserPlus, MessageCircle, Users, Check } from '@lucide/svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import SheetShell from '@/components/overlays/SheetShell.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import { ROUTES } from '@/lib/routes';

	// 新会话底部 Sheet：添加好友 / 开始私聊 / 创建群聊入口。

	type Tab = 'add' | 'chat';

	const SUGGESTED: { id: number; name: string; subtitle: string; avatar: string; added: boolean }[] = [];
	const CONTACTS: { id: number; name: string; avatar: string; online: boolean }[] = [];

	let {
		defaultTab = 'chat',
		onClose,
		onNewChat,
		onCreateGroup
	}: {
		defaultTab?: Tab;
		onClose: () => void;
		onNewChat?: (contactId: number) => void;
		onCreateGroup?: () => void;
	} = $props();

	let tab = $state<Tab>(initialTab());

	// defaultTab 仅在打开时生效（此后切换走 tab 按钮），有意只读取一次初始值。
	function initialTab(): Tab {
		return defaultTab;
	}
let searchQ = $state('');
let addedIds = $state<Set<number>>(new Set());

// 行整体与「添加」按钮共用同一切换逻辑（按钮 stopPropagation 避免双击切换）
function toggleAdd(id: number) {
	const n = new Set(addedIds);
	n.has(id) ? n.delete(id) : n.add(id);
	addedIds = n;
}

	const filteredSuggested = $derived(
		SUGGESTED.filter((u) => !searchQ || u.name.toLowerCase().includes(searchQ.toLowerCase()) || u.subtitle.includes(searchQ))
	);
	const filteredContacts = $derived(CONTACTS.filter((c) => !searchQ || c.name.toLowerCase().includes(searchQ.toLowerCase())));
</script>

<SheetShell title={tab === 'add' ? '添加好友' : '新建会话'} {onClose}>
	<!-- Tab switcher — 统一 TabBar 原语（禁止手写 tab 变体），保留原回调 -->
	<div class="shrink-0 px-5 pt-3 pb-2">
		<TabBar
			items={[
				{ id: 'add', label: '添加好友', icon: UserPlus },
				{ id: 'chat', label: '开始私聊', icon: MessageCircle }
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
			<div class="animate-fade-in flex flex-col gap-2">
				<p class="eyebrow mb-2">
					附近推荐
				</p>
				{#if filteredSuggested.length > 0}
				{#each filteredSuggested as u (u.id)}
					<ListItem
						tone="card"
						onClick={() => toggleAdd(u.id)}
						ariaLabel={addedIds.has(u.id) ? `取消添加 ${u.name}` : `添加 ${u.name}`}
						class="flex items-center gap-3.5 p-3"
					>
						<Avatar src={u.avatar} name={u.name} size={44} class="shrink-0 border" style="border-color:var(--cst-border-default);border-width:var(--cst-border-width-regular);" />
						<div class="flex-1 min-w-0">
							<p class="text-text-primary text-sm" >{u.name}</p>
							<p class="text-text-secondary text-xs" >{u.subtitle}</p>
						</div>
						<button
							type="button"
							class="hover-btn tap-92 flex items-center gap-1.5 px-3 py-1.5 rounded-full focus-ring shrink-0 text-xs {addedIds.has(u.id) ? 'bg-surface-primary text-accent-default' : 'bg-accent-default text-text-on-accent'}"
							style={addedIds.has(u.id)
								? 'box-shadow:var(--shadow-inset-sm);border:var(--cst-border-width-hairline) solid var(--cst-border-strong);'
								: 'box-shadow:var(--shadow-glow-sm);'}
							onclick={(e) => {
								e.stopPropagation();
								toggleAdd(u.id);
							}}
						>
							{#if addedIds.has(u.id)}
								<Check class="size-2.5"  strokeWidth={2.5} />
								已添加
							{:else}
								<UserPlus class="size-2.5"  strokeWidth={2} />
								添加
							{/if}
						</button>
					</ListItem>
				{/each}
				{:else}
					<EmptyState
						bare
						title={searchQ.trim() ? '未找到相关用户' : '暂无推荐好友'}
						description={searchQ.trim() ? '换个关键词试试' : '去附近的人里认识新朋友'}
					>
						{#snippet action()}
							<button
								type="button"
								class="hover-btn rounded-full border border-border-default px-4 py-2 text-xs text-text-primary focus-ring"
								onclick={() => {
									onClose();
									void goto(ROUTES.nearbyPeople);
								}}
							>
								去附近看看
							</button>
						{/snippet}
					</EmptyState>
				{/if}
			</div>
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
					<div class="flex items-center justify-center rounded-full shrink-0 bg-accent-default size-11 shadow-glow-sm"  >
						<Users class="size-5 text-text-on-accent"   strokeWidth={1.75} />
					</div>
					<div>
						<p class="text-text-primary text-sm font-medium">创建群聊</p>
						<p class="text-text-secondary text-xs" >邀请好友一起聊</p>
					</div>
				</ListItem>

				<p class="eyebrow mb-2">
					联系人
				</p>
				{#if filteredContacts.length > 0}
				{#each filteredContacts as c (c.id)}
					<ListItem
						tone="card"
						onClick={() => {
							onNewChat?.(c.id);
							onClose();
						}}
						ariaLabel={`与 ${c.name} 开始私聊`}
						class="flex items-center gap-3.5 p-3"
					>
						<div class="relative shrink-0">
							<Avatar src={c.avatar} name={c.name} size={44} class="border" style="border-color:var(--cst-border-default);border-width:var(--cst-border-width-regular);" />
							{#if c.online}
								<OnlineDot size="md" />
							{/if}
						</div>
						<div class="flex-1 min-w-0">
							<p class="text-text-primary text-sm" >{c.name}</p>
							<p class="{c.online ? 'text-status-online' : 'text-text-secondary'} text-xs" >
								{c.online ? '在线' : '离线'}
							</p>
						</div>
						<MessageCircle class="size-4 shrink-0 text-text-secondary"   strokeWidth={1.5} />
					</ListItem>
				{/each}
				{:else}
					<EmptyState
						bare
						title={searchQ.trim() ? '未找到联系人' : '暂无联系人'}
						description={searchQ.trim() ? '换个关键词试试' : '先添加好友，即可在这里发起私聊'}
					>
						{#snippet action()}
							<button
								type="button"
								class="hover-btn rounded-full border border-border-default px-4 py-2 text-xs text-text-primary focus-ring"
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
