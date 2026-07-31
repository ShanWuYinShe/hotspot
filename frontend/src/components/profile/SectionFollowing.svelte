<script lang="ts">
	import { goto } from '$app/navigation';
	import { UserPlus, Check, TrendingUp, Heart } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import EmptyState from '@/components/shared/EmptyState.svelte';
	import { useRelationships } from '@/lib/profile-data.svelte';
	import { ROUTES } from '@/lib/routes';

	// 关注分区（迁移自旧版 ProfileSections 内 SectionFollowing/FollowingRow）。

	let { embedded = false }: { embedded?: boolean } = $props();

	let sortBy = $state<'recent' | 'active'>('recent');
	let unfollowedIds = $state<Set<number>>(new Set());

	const { following } = useRelationships();

	const toggleUnfollow = (id: number) => {
		const next = new Set(unfollowedIds);
		next.has(id) ? next.delete(id) : next.add(id);
		unfollowedIds = next;
	};

	const sorted = $derived([...following].sort((a, b) => (sortBy === 'active' ? b.posts - a.posts : 0)));
</script>

<div class="flex-1 overflow-y-auto no-scrollbar flex flex-col">
	{#if !embedded}
		<!-- Header -->
		<div class="flex items-center justify-between px-5 py-4 flex-shrink-0" style="border-bottom:1px solid var(--neu-border);">
			<div>
				<div class="flex items-center gap-2.5">
					<span class="text-[var(--neu-text)] text-[15px]">关注</span>
					<span class="px-2 py-0.5 rounded-full bg-[var(--neu-accent-primary)] text-white text-[11px]">
						{following.length} 人
					</span>
				</div>
				<p class="mt-1 text-[var(--neu-text-muted)] text-[11px]">你主动订阅的动态与创作者</p>
			</div>
			<span class="rounded-full px-3 py-1 bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)] text-[11px]" style="border:1px solid var(--neu-accent-soft-border);">
				点击卡片查看主页
			</span>
		</div>
	{/if}

	<!-- Sort tabs — ghost chips -->
	<div class="flex gap-1.5 {embedded ? 'px-6 lg:px-8' : 'px-5'} py-4 flex-shrink-0">
		{#each [{ id: 'recent', label: '最近关注' }, { id: 'active', label: '最活跃' }] as pill}
			{@const active = sortBy === pill.id}
			<button
				type="button"
				class="neu-hover-btn flex items-center gap-1.5 px-3 py-1 rounded-full outline-none neu-surface-ghost text-xs {active ? 'bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)] font-medium' : 'text-[var(--neu-text-muted)]'}"
				style={active ? 'border-color:var(--neu-accent-soft-border);' : ''}
				onclick={() => (sortBy = pill.id as 'recent' | 'active')}
			>
				{#if pill.id === 'active'}
					<TrendingUp style="width:11px;height:11px;" strokeWidth={active ? 2 : 1.5} />
				{/if}
				{pill.label}
			</button>
		{/each}
	</div>

	<!-- Following list -->
	{#if embedded}
		<div class="grid w-full grid-cols-1 gap-4 px-6 pb-8 md:grid-cols-2 lg:px-8 xl:grid-cols-3">
			{#if sorted.length === 0}
				<div class="col-span-full">
					<EmptyState title="还没有关注" description="去发现附近的人，找到有趣的创作者。">
						{#snippet icon()}
							<Heart size={18} />
						{/snippet}
					</EmptyState>
				</div>
			{:else}
				{#each sorted as f (f.id)}
					<div
						class="neu-hover-card neu-row-hover flex items-center gap-3 rounded-[18px] p-3.5 cursor-pointer neu-focus-ring bg-[var(--neu-bg)]"
						style="box-shadow:5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light);border:1px solid color-mix(in srgb, var(--neu-accent) 10%, var(--neu-bg));opacity:{unfollowedIds.has(f.id) ? 0.55 : 1};"
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
							<p class="text-[var(--neu-text)] text-sm">{f.name}</p>
							<p class="truncate text-[var(--neu-text-muted)] text-[11px] mt-px">{f.desc}</p>
							<p class="text-[var(--neu-accent-primary)] text-[10px] mt-px">{f.posts} 帖子</p>
						</div>
						<img src={f.latestImg} alt="" class="rounded-[8px] object-cover flex-shrink-0" style="width:44px;height:44px;opacity:0.85;" />
						<button
							type="button"
							class="neu-hover-btn flex items-center gap-1 px-3 py-1 rounded-full outline-none flex-shrink-0 text-[11px] {unfollowedIds.has(f.id) ? 'bg-[var(--neu-accent-primary)] text-white' : 'bg-transparent text-[var(--neu-text-muted)]'}"
							style={unfollowedIds.has(f.id) ? 'border:1px solid transparent;' : 'border:1px solid var(--neu-border-pressed);'}
							onclick={(e) => {
								e.stopPropagation();
								toggleUnfollow(f.id);
							}}
						>
							{#if unfollowedIds.has(f.id)}
								<UserPlus style="width:10px;height:10px;" strokeWidth={2} />
								关注
							{:else}
								<Check style="width:10px;height:10px;" strokeWidth={2.5} />
								已关注
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
					{#each sorted as f (f.id)}
						<div
							class="neu-hover-bg neu-row-hover flex cursor-pointer items-center gap-3 p-3 outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-[var(--neu-accent-primary)]"
							style="background:var(--neu-bg);opacity:{unfollowedIds.has(f.id) ? 0.55 : 1};transition:background 0.18s ease;"
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
								<p class="text-[var(--neu-text)] text-[13px]">{f.name}</p>
								<p class="truncate text-[var(--neu-text-muted)] text-[11px] mt-px">{f.desc}</p>
								<p class="text-[var(--neu-accent-primary)] text-[10px] mt-px">{f.posts} 帖子</p>
							</div>
							<img src={f.latestImg} alt="" class="rounded-[8px] object-cover flex-shrink-0" style="width:36px;height:36px;opacity:0.85;" />
							<button
								type="button"
								class="neu-hover-btn flex items-center gap-1 px-3 py-1 rounded-full outline-none flex-shrink-0 text-[11px] {unfollowedIds.has(f.id) ? 'bg-[var(--neu-accent-primary)] text-white' : 'bg-transparent text-[var(--neu-text-muted)]'}"
								style={unfollowedIds.has(f.id) ? 'border:1px solid transparent;' : 'border:1px solid var(--neu-border-pressed);'}
								onclick={(e) => {
									e.stopPropagation();
									toggleUnfollow(f.id);
								}}
							>
								{#if unfollowedIds.has(f.id)}
									<UserPlus style="width:10px;height:10px;" strokeWidth={2} />
									关注
								{:else}
									<Check style="width:10px;height:10px;" strokeWidth={2.5} />
									已关注
								{/if}
							</button>
						</div>
					{/each}
				</div>
			</div>
		</div>
	{/if}
</div>
