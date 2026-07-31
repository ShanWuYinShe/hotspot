<script lang="ts">
	import { goto } from '$app/navigation';
	import { Plus, Users, ChevronRight, Clock, Shield } from '@lucide/svelte';
	import { platformStore } from '@/stores/platform.svelte';
	import { attachDismissibleLayer } from '@/lib/dismissible-layer';
	import CreateGroupSheet from '@/components/groups/CreateGroupSheet.svelte';
	import BspCardCanvas from '@/components/shared/BspCardCanvas.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import SearchInput from '@/components/shared/SearchInput.svelte';
	import { useMyGroups } from '@/lib/profile-data.svelte';
	import { ROUTES } from '@/lib/routes';

	// 我的群组分区（迁移自 React ProfileSections.tsx 内 SectionGroups/MyGroupCard）。

	let searchQ = $state('');
	let showCreate = $state(false);
	let visibleCount = $state(12);

	const { groups } = useMyGroups();
	const filtered = $derived(groups.filter((g) => searchQ === '' || g.name.toLowerCase().includes(searchQ.toLowerCase())));
	const isDesktop = $derived(platformStore.platform === 'desktop');

	$effect(() => {
		return attachDismissibleLayer({
			open: showCreate,
			onDismiss: () => (showCreate = false)
		});
	});
</script>

<div class="flex-1 overflow-y-auto no-scrollbar flex flex-col">
	<!-- Section header -->
	<div class="flex items-center justify-between px-5 py-4 flex-shrink-0" style="border-bottom:1px solid var(--neu-border);">
		<div class="flex items-center gap-2.5">
			<span class="text-[var(--neu-text)] text-[15px]">我的群组</span>
			<span class="px-2 py-0.5 rounded-full bg-[var(--neu-accent-primary)] text-white text-[11px]">
				{groups.length} 个
			</span>
		</div>
		<button
			type="button"
			class="neu-hover-btn flex items-center gap-1.5 px-3.5 py-2 rounded-full outline-none bg-[var(--neu-accent-primary)] text-white text-xs"
			style="box-shadow:0 4px 12px var(--neu-glow);"
			onclick={() => (showCreate = true)}
		>
			<Plus style="width:13px;height:13px;" strokeWidth={2.5} />
			创建群组
		</button>
	</div>

	<!-- Search -->
	<div class="px-5 pt-3 pb-2 flex-shrink-0">
		<SearchInput value={searchQ} onChange={(v) => (searchQ = v)} placeholder="搜索群组…" compact />
	</div>

	<!-- Group cards -->
	{#if isDesktop}
		<div class="px-5 pb-6">
			<BspCardCanvas
				items={filtered.slice(0, visibleCount)}
				gap={24}
				minRatio={0.68}
				maxRatio={1.48}
				minCardWidth={160}
				minCardHeight={188}
				minCanvasHeight={520}
				ariaLabel="我的群组的不规则满铺布局"
				hasMore={visibleCount < filtered.length}
				onLoadMore={() => (visibleCount = Math.min(visibleCount + 8, filtered.length))}
			>
				{#snippet renderItem(g: any, index: number)}
					<button
						type="button"
						class="neu-hover-card flex h-full w-full flex-col rounded-[20px] overflow-hidden cursor-pointer neu-card-hover neu-focus-ring bg-[var(--neu-bg)] text-left"
						style="box-shadow:6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light);border:1px solid color-mix(in srgb, var(--neu-accent) 10%, var(--neu-bg));"
						onclick={() => void goto(`${ROUTES.group(g.id)}?from=me`)}
					>
						<!-- Banner -->
						<div class="relative flex-1 min-h-0" style="min-height:clamp(60px, 10vw, 80px);">
							<img src={g.banner} alt={g.name} class="w-full h-full object-cover" />
							<div class="absolute inset-0" style="background:linear-gradient(to bottom, transparent 30%, rgba(0,0,0,0.6) 100%);"></div>
							<div class="absolute bottom-2 left-3 flex items-center gap-1.5">
								<span class="px-2 py-0.5 rounded-full text-[10px] {g.role === '管理员' ? 'bg-[var(--neu-accent-primary)]' : 'bg-[rgba(0,0,0,0.5)]'} text-white" style="backdrop-filter:blur(8px);">
									{#if g.role === '管理员'}
										<Shield style="width:8px;height:8px;display:inline;margin-right:2px;" strokeWidth={2} />
									{/if}
									{g.role}
								</span>
							</div>
							<div class="absolute bottom-2 right-3 flex items-center gap-1">
								<div class="bg-[var(--semantic-online)]" style="width:7px;height:7px;border-radius:50%;"></div>
								<span class="text-white text-[10px]">{g.online} 在线</span>
							</div>
						</div>
						<!-- Info -->
						<div class="flex items-center gap-3 p-3.5">
							<Avatar shape="rounded" src={g.avatar} name={g.name} size={44} style="margin-top:-22px;border:2px solid var(--neu-bg);box-shadow:0 4px 12px rgba(0,0,0,0.15);" />
							<div class="flex-1 min-w-0">
								<p class="text-[var(--neu-text)] text-sm">{g.name}</p>
								<p class="text-[var(--neu-text-muted)] text-[11px] mt-0.5">{g.desc}</p>
							</div>
							<div class="flex flex-col items-end gap-1.5 flex-shrink-0">
								<div class="flex items-center gap-1">
									<Users style="width:11px;height:11px;color:var(--neu-text-muted);" strokeWidth={1.5} />
									<span class="text-[var(--neu-text-muted)] text-[11px]">{g.members}</span>
								</div>
								<span class="flex items-center gap-1 px-2.5 py-1 rounded-full bg-[var(--neu-accent-primary)] text-white text-[11px]" style="box-shadow:0 3px 8px var(--neu-glow);">
									进入
									<ChevronRight style="width:10px;height:10px;" strokeWidth={2} />
								</span>
							</div>
						</div>
						<!-- Last active -->
						<div class="px-3.5 pb-3 flex items-center gap-1.5" style="border-top:1px solid var(--neu-border);padding-top:8px;margin-top:-4px;">
							<Clock style="width:11px;height:11px;color:var(--neu-text-muted);" strokeWidth={1.5} />
							<span class="text-[var(--neu-text-muted)] text-[11px]">最近活跃 {g.lastActive}</span>
						</div>
					</button>
				{/snippet}
			</BspCardCanvas>
		</div>
	{:else}
		<div class="flex flex-col gap-3 px-5 pb-6">
			{#each filtered as g (g.id)}
				<div
					class="neu-hover-card flex h-full w-full flex-col rounded-[20px] overflow-hidden cursor-pointer neu-card-hover neu-focus-ring bg-[var(--neu-bg)]"
					style="box-shadow:6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light);border:1px solid color-mix(in srgb, var(--neu-accent) 10%, var(--neu-bg));"
					role="button"
					tabindex="0"
					onclick={() => void goto(`${ROUTES.group(g.id)}?from=me`)}
					onkeydown={(e) => {
						if (e.key === 'Enter' || e.key === ' ') {
							e.preventDefault();
							void goto(`${ROUTES.group(g.id)}?from=me`);
						}
					}}
				>
					<!-- Banner -->
					<div class="relative flex-1 min-h-0" style="min-height:clamp(60px, 10vw, 80px);">
						<img src={g.banner} alt={g.name} class="w-full h-full object-cover" />
						<div class="absolute inset-0" style="background:linear-gradient(to bottom, transparent 30%, rgba(0,0,0,0.6) 100%);"></div>
						<div class="absolute bottom-2 left-3 flex items-center gap-1.5">
							<span class="px-2 py-0.5 rounded-full text-[10px] {g.role === '管理员' ? 'bg-[var(--neu-accent-primary)]' : 'bg-[rgba(0,0,0,0.5)]'} text-white" style="backdrop-filter:blur(8px);">
								{#if g.role === '管理员'}
									<Shield style="width:8px;height:8px;display:inline;margin-right:2px;" strokeWidth={2} />
								{/if}
								{g.role}
							</span>
						</div>
						<div class="absolute bottom-2 right-3 flex items-center gap-1">
							<div class="bg-[var(--semantic-online)]" style="width:7px;height:7px;border-radius:50%;"></div>
							<span class="text-white text-[10px]">{g.online} 在线</span>
						</div>
					</div>
					<!-- Info -->
					<div class="flex items-center gap-3 p-3.5">
						<Avatar shape="rounded" src={g.avatar} name={g.name} size={44} style="margin-top:-22px;border:2px solid var(--neu-bg);box-shadow:0 4px 12px rgba(0,0,0,0.15);" />
						<div class="flex-1 min-w-0">
							<p class="text-[var(--neu-text)] text-sm">{g.name}</p>
							<p class="text-[var(--neu-text-muted)] text-[11px] mt-0.5">{g.desc}</p>
						</div>
						<div class="flex flex-col items-end gap-1.5 flex-shrink-0">
							<div class="flex items-center gap-1">
								<Users style="width:11px;height:11px;color:var(--neu-text-muted);" strokeWidth={1.5} />
								<span class="text-[var(--neu-text-muted)] text-[11px]">{g.members}</span>
							</div>
							<button
								type="button"
								class="neu-hover-btn flex items-center gap-1 px-2.5 py-1 rounded-full outline-none bg-[var(--neu-accent-primary)] text-white text-[11px]"
								style="box-shadow:0 3px 8px var(--neu-glow);"
								onclick={(e) => {
									e.stopPropagation();
									void goto(`${ROUTES.group(g.id)}?from=me`);
								}}
							>
								进入
								<ChevronRight style="width:10px;height:10px;" strokeWidth={2} />
							</button>
						</div>
					</div>
					<!-- Last active -->
					<div class="px-3.5 pb-3 flex items-center gap-1.5" style="border-top:1px solid var(--neu-border);padding-top:8px;margin-top:-4px;">
						<Clock style="width:11px;height:11px;color:var(--neu-text-muted);" strokeWidth={1.5} />
						<span class="text-[var(--neu-text-muted)] text-[11px]">最近活跃 {g.lastActive}</span>
					</div>
				</div>
			{/each}
		</div>
	{/if}

	{#if showCreate}
		<div class="fixed inset-0 z-40 bg-[rgba(0,0,0,0.35)]" style="backdrop-filter:blur(3px);" role="presentation" onclick={() => (showCreate = false)}></div>
		<div
			class="fixed bottom-0 left-0 right-0 z-50 overflow-hidden bg-[var(--neu-bg)]"
			role="dialog"
			aria-modal="true"
			aria-label="创建群组"
			style="height:92%;border-radius:28px 28px 0 0;box-shadow:0 -8px 40px rgba(0,0,0,0.18);"
		>
			<CreateGroupSheet onClose={() => (showCreate = false)} />
		</div>
	{/if}
</div>
