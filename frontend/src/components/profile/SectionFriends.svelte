<script lang="ts">
	import { goto } from '$app/navigation';
	import { MessageCircle } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import TagPill from '@/components/shared/TagPill.svelte';
	import SearchInput from '@/components/shared/SearchInput.svelte';
	import { useRelationships } from '@/lib/profile-data.svelte';
	import { ROUTES } from '@/lib/routes';

	// 好友分区（迁移自 React ProfileSections.tsx 内 SectionFriends）。

	let { embedded = false }: { embedded?: boolean } = $props();

	let searchQ = $state('');
	let activeFilter = $state<'all' | 'online' | 'offline'>('all');

	const { friends, loading } = useRelationships();

	const filtered = $derived(
		friends
			.filter((f) => activeFilter === 'all' || (activeFilter === 'online' ? f.online : !f.online))
			.filter((f) => searchQ === '' || f.name.toLowerCase().includes(searchQ.toLowerCase()) || f.handle.includes(searchQ))
	);
	const onlineCount = $derived(friends.filter((f) => f.online).length);
</script>

<div class="flex-1 overflow-y-auto no-scrollbar flex flex-col">
	{#if !embedded}
		<!-- Header -->
		<div class="flex items-center justify-between px-5 py-4 flex-shrink-0" style="border-bottom:1px solid var(--neu-border);">
			<div>
				<div class="flex items-center gap-2.5">
					<span class="text-[var(--neu-text)] text-[15px]">好友</span>
					<div class="flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px]" style="background:color-mix(in srgb, var(--semantic-online) 13%, transparent);border:1px solid color-mix(in srgb, var(--semantic-online) 24%, transparent);">
						<div class="bg-[var(--semantic-online)]" style="width:6px;height:6px;border-radius:50%;"></div>
						<span class="text-[var(--semantic-online)]">{onlineCount} 在线</span>
					</div>
				</div>
				<p class="mt-1 text-[var(--neu-text-muted)] text-[11px]">双向关系，可直接开启对话</p>
			</div>
			<span class="text-[var(--neu-text-muted)] text-xs">{friends.length} 位好友</span>
		</div>
	{/if}

	<!-- Search -->
	<div class="{embedded ? 'px-6 lg:px-8' : 'px-5'} pt-4 pb-2 flex-shrink-0">
		<SearchInput value={searchQ} onChange={(v) => (searchQ = v)} placeholder="搜索好友…" compact />
	</div>

	<!-- Filter pills -->
	<div class="flex gap-2 {embedded ? 'px-6 lg:px-8' : 'px-5'} pb-4 flex-shrink-0">
		{#each [{ id: 'all', label: '全部' }, { id: 'online', label: '在线' }, { id: 'offline', label: '离线' }] as pill}
			<button
				type="button"
				class="neu-hover-btn px-3 py-1.5 rounded-full outline-none text-xs {activeFilter === pill.id ? 'bg-[var(--neu-accent-primary)] text-white' : 'bg-[var(--neu-bg)] text-[var(--neu-text-muted)]'}"
				style={activeFilter === pill.id ? 'box-shadow:0 3px 8px var(--neu-glow);' : 'border:1px solid var(--neu-border);'}
				onclick={() => (activeFilter = pill.id as 'all' | 'online' | 'offline')}
			>
				{pill.label}
			</button>
		{/each}
	</div>

	<!-- Friend list -->
	<div class={embedded
		? 'grid w-full grid-cols-1 gap-4 px-6 pb-8 md:grid-cols-2 lg:px-8 xl:grid-cols-3'
		: 'flex flex-col gap-3 px-5 pb-6'}>
		{#each filtered as f (f.id)}
			<div
				class="neu-hover-card neu-row-hover flex items-center gap-3 rounded-[18px] p-3.5 cursor-pointer neu-focus-ring bg-[var(--neu-bg)]"
				style="box-shadow:5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light);border:1px solid color-mix(in srgb, var(--neu-accent) 10%, var(--neu-bg));"
				role="button"
				tabindex="0"
				onclick={() => void goto(`${ROUTES.user(f.id)}?from=relationships`)}
				onkeydown={(e) => {
					if (e.key === 'Enter' || e.key === ' ') {
						e.preventDefault();
						void goto(`${ROUTES.user(f.id)}?from=relationships`);
					}
				}}
			>
				<Avatar ring src={f.avatar} name={f.name} size={48}>
					{#snippet badge()}
						<span class="absolute bottom-0 right-0 rounded-full {f.online ? 'bg-[var(--semantic-online)]' : 'bg-[var(--neu-text-muted)]'}" style="width:12px;height:12px;border:2.5px solid var(--neu-bg);opacity:{f.online ? 1 : 0.5};"></span>
					{/snippet}
				</Avatar>
				<div class="flex-1 min-w-0">
					<div class="flex items-center gap-2">
						<span class="text-[var(--neu-text)] text-sm">{f.name}</span>
						<TagPill size="sm">{f.tag}</TagPill>
					</div>
					<p class="text-[var(--neu-text-muted)] text-xs mt-0.5">
						{f.handle} · {f.mutual} 共同好友
					</p>
				</div>
				<div class="flex flex-col gap-1.5 flex-shrink-0">
					<button
						type="button"
						class="neu-hover-btn flex items-center justify-center gap-1 px-2.5 py-1.5 rounded-full outline-none bg-[var(--neu-accent-primary)] text-white text-[11px]"
						style="box-shadow:0 3px 8px var(--neu-glow);"
						onclick={(e) => {
							e.stopPropagation();
							void goto(ROUTES.message(f.id));
						}}
					>
						<MessageCircle style="width:11px;height:11px;" strokeWidth={2} />
						发消息
					</button>
					<button
						type="button"
						class="neu-hover-btn flex items-center justify-center gap-1 px-2.5 py-1.5 rounded-full outline-none bg-[var(--neu-bg)] text-[var(--neu-text-muted)] text-[11px]"
						style="border:1px solid var(--neu-border);box-shadow:3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light);"
						onclick={(e) => {
							e.stopPropagation();
							void goto(`${ROUTES.user(f.id)}?from=relationships`);
						}}
					>
						查看主页
					</button>
				</div>
			</div>
		{/each}
	</div>
</div>
