<script lang="ts">
	import { X, UserPlus, MessageCircle, Users, Check } from '@lucide/svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';

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

	const filteredSuggested = $derived(
		SUGGESTED.filter((u) => !searchQ || u.name.toLowerCase().includes(searchQ.toLowerCase()) || u.subtitle.includes(searchQ))
	);
	const filteredContacts = $derived(CONTACTS.filter((c) => !searchQ || c.name.toLowerCase().includes(searchQ.toLowerCase())));
</script>

<div class="flex flex-col h-full bg-[var(--surface-primary)]">
	<!-- Handle -->
	<div class="flex justify-center pt-3 pb-1 flex-shrink-0">
		<div class="rounded-full bg-[var(--border-default)]" style="width:36px;height:4px;"></div>
	</div>

	<!-- Header -->
	<div class="flex items-center justify-between px-5 py-3 flex-shrink-0" style="border-bottom:1px solid var(--border-default);">
		<h2 class="text-[var(--text-primary)]" style="font-size:17px;">
			{tab === 'add' ? '添加好友' : '新建会话'}
		</h2>
		<button
			type="button"
			class="hover-btn scale-btn flex items-center justify-center rounded-full outline-none bg-[var(--surface-primary)]"
			style="width:32px;height:32px;box-shadow:3px 3px 7px var(--elevation-shadow-dark), -3px -3px 7px var(--elevation-shadow-light);border:1px solid var(--border-default);"
			onclick={onClose}
		>
			<X style="width:14px;height:14px;color:var(--text-secondary);" strokeWidth={2} />
		</button>
	</div>

	<!-- Tab switcher -->
	<div class="flex px-5 pt-3 pb-2 gap-2 flex-shrink-0">
		{#each [{ id: 'add', icon: UserPlus, label: '添加好友' }, { id: 'chat', icon: MessageCircle, label: '开始私聊' }] as t}
			<button
				type="button"
				class="hover-btn flex items-center gap-2 px-4 py-2 rounded-full outline-none {tab === t.id ? 'bg-[var(--accent-default)] text-white' : 'bg-[var(--surface-primary)] text-[var(--text-secondary)]'}"
				style={tab === t.id
					? 'font-size:13px;font-weight:500;box-shadow:0 4px 12px var(--glow-accent);'
					: 'font-size:13px;box-shadow:3px 3px 7px var(--elevation-shadow-dark), -3px -3px 7px var(--elevation-shadow-light);border:1px solid var(--border-default);'}
				onclick={() => {
					tab = t.id as Tab;
					searchQ = '';
				}}
			>
				<t.icon style="width:13px;height:13px;" strokeWidth={1.75} />
				{t.label}
			</button>
		{/each}
	</div>

	<!-- Search bar — 复用 SearchInput 统一原语 -->
	<div class="px-5 pb-3 flex-shrink-0">
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
			<div class="animate-fade-in flex flex-col gap-1">
				<p class="uppercase mb-2 text-[var(--text-secondary)]" style="font-size:10px;letter-spacing:0.14em;">
					附近推荐
				</p>
				{#each filteredSuggested as u (u.id)}
					<div class="flex items-center gap-3.5 px-3 py-3 rounded-relaxed row-hover" style="border:1px solid var(--border-default);">
						<img src={u.avatar} alt={u.name} class="rounded-full object-cover flex-shrink-0" style="width:44px;height:44px;border:1.5px solid var(--border-default);" />
						<div class="flex-1 min-w-0">
							<p class="text-[var(--text-primary)]" style="font-size:14px;">{u.name}</p>
							<p class="text-[var(--text-secondary)]" style="font-size:11px;">{u.subtitle}</p>
						</div>
						<button
							type="button"
							class="hover-btn tap-92 flex items-center gap-1.5 px-3 py-1.5 rounded-full outline-none flex-shrink-0 {addedIds.has(u.id) ? 'bg-[var(--surface-primary)] text-[var(--accent-default)]' : 'bg-[var(--accent-default)] text-white'}"
							style={addedIds.has(u.id)
								? 'box-shadow:inset 2px 2px 5px var(--elevation-shadow-dark), inset -2px -2px 5px var(--elevation-shadow-light);border:1px solid var(--border-strong);font-size:11px;'
								: 'font-size:11px;box-shadow:0 3px 8px var(--glow-accent);'}
							onclick={() => {
								const n = new Set(addedIds);
								n.has(u.id) ? n.delete(u.id) : n.add(u.id);
								addedIds = n;
							}}
						>
							{#if addedIds.has(u.id)}
								<Check style="width:10px;height:10px;" strokeWidth={2.5} />
								已添加
							{:else}
								<UserPlus style="width:10px;height:10px;" strokeWidth={2} />
								添加
							{/if}
						</button>
					</div>
				{/each}
			</div>
		{:else}
			<div class="animate-fade-in flex flex-col gap-1">
				<!-- Create group chat row -->
				<button
					type="button"
					class="hover-card flex items-center gap-3.5 px-3 py-3 rounded-relaxed w-full outline-none text-left mb-3 bg-[var(--surface-primary)]"
					style="box-shadow:5px 5px 12px var(--elevation-shadow-dark), -5px -5px 12px var(--elevation-shadow-light);border:1px solid var(--border-default);"
					onclick={() => {
						onClose();
						onCreateGroup?.();
					}}
				>
					<div class="flex items-center justify-center rounded-full flex-shrink-0 bg-[var(--accent-default)]" style="width:44px;height:44px;box-shadow:0 3px 8px var(--glow-accent);">
						<Users style="width:20px;height:20px;color:var(--text-on-accent);" strokeWidth={1.75} />
					</div>
					<div>
						<p class="text-[var(--text-primary)]" style="font-size:14px;font-weight:500;">创建群聊</p>
						<p class="text-[var(--text-secondary)]" style="font-size:11px;">邀请好友一起聊</p>
					</div>
				</button>

				<p class="uppercase mb-2 text-[var(--text-secondary)]" style="font-size:10px;letter-spacing:0.14em;">
					联系人
				</p>
				{#each filteredContacts as c (c.id)}
					<button
						type="button"
						class="flex items-center gap-3.5 px-3 py-3 rounded-relaxed cursor-pointer w-full text-left row-hover"
						style="border:1px solid var(--border-default);"
						onclick={() => {
							onNewChat?.(c.id);
							onClose();
						}}
					>
						<div class="relative flex-shrink-0">
							<img src={c.avatar} alt={c.name} class="rounded-full object-cover" style="width:44px;height:44px;border:1.5px solid var(--border-default);" />
							{#if c.online}
								<span class="absolute bottom-0 right-0 rounded-full bg-[var(--status-online)]" style="width:10px;height:10px;border:2px solid var(--surface-primary);"></span>
							{/if}
						</div>
						<div class="flex-1 min-w-0">
							<p class="text-[var(--text-primary)]" style="font-size:14px;">{c.name}</p>
							<p class="{c.online ? 'text-[var(--status-online)]' : 'text-[var(--text-secondary)]'}" style="font-size:11px;">
								{c.online ? '在线' : '离线'}
							</p>
						</div>
						<MessageCircle style="width:16px;height:16px;color:var(--text-secondary);flex-shrink:0;" strokeWidth={1.5} />
					</button>
				{/each}
			</div>
		{/if}
	</div>
</div>
