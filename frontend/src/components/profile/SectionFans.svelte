<script lang="ts">
	import { goto } from '$app/navigation';
	import { UserPlus, Check } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import EmptyState from '@/components/shared/EmptyState.svelte';
	import DataDisplay, { type MetricItem } from '@/components/shared/DataDisplay.svelte';
	import { useRelationships } from '@/lib/profile-data.svelte';
	import { ROUTES } from '@/lib/routes';

	// 粉丝分区（迁移自旧版 ProfileSections 内 SectionFans/FanRow）。

	let { embedded = false }: { embedded?: boolean } = $props();

	let fanFilter = $state<'all' | 'mutual' | 'pending'>('all');

	const { followers } = useRelationships();
	let followedBack = $state<Set<number>>(new Set(followers.filter((f: any) => f.isMutual).map((f: any) => f.id)));

	const toggleFollow = (id: number) => {
		const next = new Set(followedBack);
		next.has(id) ? next.delete(id) : next.add(id);
		followedBack = next;
	};

	const filtered = $derived(
		followers.filter((f: any) => {
			const isMutual = followedBack.has(f.id);
			if (fanFilter === 'mutual') return isMutual;
			if (fanFilter === 'pending') return !isMutual;
			return true;
		})
	);
	const mutualCount = $derived([...followedBack].length);
	const pendingCount = $derived(followers.length - mutualCount);

	const fanLabel = (joinedDays: number) =>
		joinedDays <= 1 ? '今天' : joinedDays <= 7 ? `${joinedDays}天前` : `${Math.floor(joinedDays / 7)}周前`;
</script>

<div class="flex-1 overflow-y-auto no-scrollbar flex flex-col">
	{#if !embedded}
		<!-- Header -->
		<div class="flex items-center justify-between px-5 py-4 flex-shrink-0" style="border-bottom:1px solid var(--neu-border);">
			<div>
				<div class="flex items-center gap-2.5">
					<span class="text-[var(--neu-text)] text-[15px]">粉丝</span>
					<span class="px-2 py-0.5 rounded-full bg-[var(--neu-accent-primary)] text-white text-[11px]">
						{followers.length} 人
					</span>
				</div>
				<p class="mt-1 text-[var(--neu-text-muted)] text-[11px]">关注你的人，优先处理待回关关系</p>
			</div>
			<div class="flex items-center gap-2">
				{#if pendingCount > 0}
					<span class="px-2 py-0.5 rounded-full text-[var(--semantic-warning)] text-[11px]" style="background:color-mix(in srgb, var(--semantic-warning) 13%, transparent);border:1px solid color-mix(in srgb, var(--semantic-warning) 24%, transparent);">
						{pendingCount} 待回关
					</span>
				{/if}
			</div>
		</div>
	{/if}

	<!-- Stats row — flat joined cells -->
	<div class="{embedded ? 'px-6 lg:px-8' : 'px-5'} py-4 flex-shrink-0">
		<DataDisplay
			items={[
				{ label: '总粉丝', value: followers.length },
				{ label: '互相关注', value: mutualCount, tone: 'positive' },
				{ label: '待回关', value: pendingCount, tone: 'warning' }
			]}
		/>
	</div>

	<!-- Filter pills — ghost -->
	<div class="flex gap-1.5 {embedded ? 'px-6 lg:px-8' : 'px-5'} pb-4 flex-shrink-0">
		{#each [{ id: 'all', label: '全部' }, { id: 'mutual', label: '互相关注' }, { id: 'pending', label: '待回关' }] as pill}
			{@const active = fanFilter === pill.id}
			<button
				type="button"
				class="neu-hover-btn px-3 py-1 rounded-full outline-none neu-surface-ghost text-xs {active ? 'bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)] font-medium' : 'text-[var(--neu-text-muted)]'}"
				style={active ? 'border-color:var(--neu-accent-soft-border);' : ''}
				onclick={() => (fanFilter = pill.id as 'all' | 'mutual' | 'pending')}
			>
				{pill.label}
			</button>
		{/each}
	</div>

	<!-- Fan list -->
	{#if embedded}
		<div class="grid w-full grid-cols-1 gap-4 px-6 pb-8 md:grid-cols-2 lg:px-8 xl:grid-cols-3">
			{#if followers.length === 0}
				<div class="col-span-full">
					<EmptyState title="还没有粉丝" description="分享你的瞬间，让更多人发现你。">
						{#snippet icon()}
							<UserPlus size={18} />
						{/snippet}
					</EmptyState>
				</div>
			{:else}
				{#each filtered as f (f.id)}
					{@const isMutual = followedBack.has(f.id)}
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
						<Avatar ring src={f.avatar} name={f.name} size={48} />
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2">
								<span class="text-[var(--neu-text)] text-sm">{f.name}</span>
								{#if isMutual}
									<span class="flex items-center gap-0.5 px-1.5 py-0.5 rounded-full text-[var(--semantic-online)] text-[10px]" style="background:color-mix(in srgb, var(--semantic-online) 12%, transparent);">
										<Check style="width:9px;height:9px;" strokeWidth={2.5} />
										互关
									</span>
								{/if}
							</div>
							<p class="text-[var(--neu-text-muted)] text-[11px] mt-0.5">
								{f.handle} · {fanLabel(f.joinedDays)}关注了你
							</p>
						</div>
						<button
							type="button"
							class="neu-hover-btn flex items-center gap-1 px-3 py-1 rounded-full outline-none flex-shrink-0 text-[11px] {isMutual ? 'bg-transparent text-[var(--neu-text-muted)]' : 'bg-[var(--neu-accent-primary)] text-white'}"
							style={isMutual ? 'border:1px solid var(--neu-border-pressed);' : 'border:1px solid transparent;'}
							onclick={(e) => {
								e.stopPropagation();
								toggleFollow(f.id);
							}}
						>
							{#if isMutual}
								<Check style="width:10px;height:10px;" strokeWidth={2.5} />
								互关
							{:else}
								<UserPlus style="width:10px;height:10px;" strokeWidth={2} />
								回关
							{/if}
						</button>
					</div>
				{/each}
			{/if}
		</div>
	{:else}
		<div class="px-5 pb-6">
			<div class="rounded-[16px] flex flex-col overflow-hidden" style="gap:1px;background:var(--neu-border);box-shadow:0 0 0 1px color-mix(in srgb, var(--neu-accent) 10%, var(--neu-bg));">
				<div style="overflow:hidden;border-radius:15px;">
					{#each filtered as f (f.id)}
						{@const isMutual = followedBack.has(f.id)}
						<div
							class="neu-hover-bg neu-row-hover flex cursor-pointer items-center gap-3 p-3 outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-[var(--neu-accent-primary)]"
							style="background:var(--neu-bg);transition:background 0.18s ease;"
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
							<Avatar ring src={f.avatar} name={f.name} size={42} />
							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-2">
									<span class="text-[var(--neu-text)] text-[13px]">{f.name}</span>
									{#if isMutual}
										<span class="flex items-center gap-0.5 px-1.5 py-0.5 rounded-full text-[var(--semantic-online)] text-[10px]" style="background:color-mix(in srgb, var(--semantic-online) 12%, transparent);">
											<Check style="width:9px;height:9px;" strokeWidth={2.5} />
											互关
										</span>
									{/if}
								</div>
								<p class="text-[var(--neu-text-muted)] text-[11px] mt-0.5">
									{f.handle} · {fanLabel(f.joinedDays)}关注了你
								</p>
							</div>
							<button
								type="button"
								class="neu-hover-btn flex items-center gap-1 px-3 py-1 rounded-full outline-none flex-shrink-0 text-[11px] {isMutual ? 'bg-transparent text-[var(--neu-text-muted)]' : 'bg-[var(--neu-accent-primary)] text-white'}"
								style={isMutual ? 'border:1px solid var(--neu-border-pressed);' : 'border:1px solid transparent;'}
								onclick={(e) => {
									e.stopPropagation();
									toggleFollow(f.id);
								}}
							>
								{#if isMutual}
									<Check style="width:10px;height:10px;" strokeWidth={2.5} />
									互关
								{:else}
									<UserPlus style="width:10px;height:10px;" strokeWidth={2} />
									回关
								{/if}
							</button>
						</div>
					{/each}
				</div>
			</div>
		</div>
	{/if}
</div>
