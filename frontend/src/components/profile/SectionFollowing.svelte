<script lang="ts">
	import { goto } from '$app/navigation';
	import { UserPlus, Check, TrendingUp, Heart } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import BorderedList from '@/components/data-display/BorderedList.svelte';
	import InteractiveListRow from '@/components/navigation/InteractiveListRow.svelte';
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
		<div class="flex items-center justify-between px-[var(--page-gutter)] py-4 flex-shrink-0" style="border-bottom:1px solid var(--neu-border);">
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
	<div class="flex gap-1.5 px-[var(--page-gutter)] py-3 flex-shrink-0">
		{#each [{ id: 'recent', label: '最近关注' }, { id: 'active', label: '最活跃' }] as pill}
			{@const active = sortBy === pill.id}
			<button
				type="button"
				class="neu-hover-btn flex items-center gap-1.5 px-3 py-1 rounded-full outline-none neu-surface-ghost text-xs neu-tap-93 {active ? 'bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)] font-medium' : 'text-[var(--neu-text-muted)]'}"
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
		<div class="grid w-full grid-cols-1 gap-4 px-[var(--page-gutter)] pb-8 md:grid-cols-2 xl:grid-cols-3">
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
						<InteractiveListRow tone="card" class="flex items-center gap-3 p-3.5 cursor-pointer"
							onClick={() => void goto(`${ROUTES.user(f.id)}?from=relationships`)}
							ariaLabel={`查看 ${f.name} 的主页`}
							style={`opacity:${unfollowedIds.has(f.id) ? 0.55 : 1};`}
						>
							<Avatar ring src={f.avatar} name={f.name} size={48} />
							<div class="flex-1 min-w-0">
								<p class="text-[var(--neu-text)] text-[13px]">{f.name}</p>
								<p class="truncate text-[var(--neu-text-muted)] text-[11px] mt-px">{f.desc}</p>
								<p class="text-[var(--neu-accent-primary)] text-[10px] mt-px">{f.posts} 帖子</p>
							</div>
							<img src={f.latestImg} alt="" class="rounded-[8px] object-cover flex-shrink-0" style="width:44px;height:44px;opacity:0.85;" />
							<button
								type="button"
								class="neu-hover-btn flex items-center gap-1 px-3 py-1 rounded-full outline-none flex-shrink-0 text-[11px] neu-tap-93 {unfollowedIds.has(f.id) ? 'bg-[var(--neu-accent-primary)] text-white' : 'bg-transparent text-[var(--neu-text-muted)]'}"
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
						</InteractiveListRow>
					{/each}
				{/if}
		</div>
	{:else}
		<div class="px-[var(--page-gutter)] pb-6">
			{#if sorted.length === 0}
				<EmptyState title="还没有关注" description="去发现附近的人，找到有趣的创作者。">
					{#snippet icon()}
						<Heart size={18} />
					{/snippet}
				</EmptyState>
			{:else}
			<BorderedList>
					{#each sorted as f (f.id)}
						<InteractiveListRow class="flex items-center gap-3 p-3 cursor-pointer"
							onClick={() => void goto(`${ROUTES.user(f.id)}?from=relationships`)}
							ariaLabel={`查看 ${f.name} 的主页`}
							style={`opacity:${unfollowedIds.has(f.id) ? 0.55 : 1};`}
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
								class="neu-hover-btn flex items-center gap-1 px-3 py-1 rounded-full outline-none flex-shrink-0 text-[11px] neu-tap-93 {unfollowedIds.has(f.id) ? 'bg-[var(--neu-accent-primary)] text-white' : 'bg-transparent text-[var(--neu-text-muted)]'}"
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
						</InteractiveListRow>
					{/each}
				</BorderedList>
				{/if}
			</div>
		{/if}
	</div>
