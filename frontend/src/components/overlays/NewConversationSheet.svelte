<script lang="ts">
	import { UserPlus, MessageCircle, Users, Check } from '@lucide/svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import SheetShell from '@/components/overlays/SheetShell.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';

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
	<!-- Tab switcher -->
	<div class="flex px-5 pt-3 pb-2 gap-2 shrink-0">
		{#each [{ id: 'add', icon: UserPlus, label: '添加好友' }, { id: 'chat', icon: MessageCircle, label: '开始私聊' }] as t}
			<button
				type="button"
				class="hover-btn flex items-center gap-2 rounded-full px-4 py-2 text-sm outline-none {tab === t.id
					? 'bg-accent-default text-text-on-accent font-medium shadow-glow-md'
					: 'border border-border-default bg-surface-primary text-text-secondary shadow-raised-sm'}"
				onclick={() => {
					tab = t.id as Tab;
					searchQ = '';
				}}
			>
				<t.icon class="size-3.25"  strokeWidth={1.75} />
				{t.label}
			</button>
		{/each}
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
				{#each filteredSuggested as u (u.id)}
					<ListItem
						tone="card"
						onClick={() => toggleAdd(u.id)}
						ariaLabel={addedIds.has(u.id) ? `取消添加 ${u.name}` : `添加 ${u.name}`}
						class="flex items-center gap-3.5 p-3"
					>
						<img src={u.avatar} alt={u.name} class="rounded-full object-cover shrink-0 size-11"  style="border:1.5px solid var(--cst-border-default)" />
						<div class="flex-1 min-w-0">
							<p class="text-text-primary text-sm" >{u.name}</p>
							<p class="text-text-secondary text-xs" >{u.subtitle}</p>
						</div>
						<button
							type="button"
							class="hover-btn tap-92 flex items-center gap-1.5 px-3 py-1.5 rounded-full outline-none shrink-0 text-xs {addedIds.has(u.id) ? 'bg-surface-primary text-accent-default' : 'bg-accent-default text-text-on-accent'}"
							style={addedIds.has(u.id)
								? 'box-shadow:var(--shadow-inset-sm);border:1px solid var(--cst-border-strong);'
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
						<Users class="size-5"  style="color:var(--cst-text-on-accent)" strokeWidth={1.75} />
					</div>
					<div>
						<p class="text-text-primary text-sm font-medium">创建群聊</p>
						<p class="text-text-secondary text-xs" >邀请好友一起聊</p>
					</div>
				</ListItem>

				<p class="eyebrow mb-2">
					联系人
				</p>
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
							<img src={c.avatar} alt={c.name} class="rounded-full object-cover size-11"  style="border:1.5px solid var(--cst-border-default)" />
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
						<MessageCircle class="size-4 shrink-0"  style="color:var(--cst-text-secondary)" strokeWidth={1.5} />
					</ListItem>
				{/each}
			</div>
		{/if}
	</div>
</SheetShell>
