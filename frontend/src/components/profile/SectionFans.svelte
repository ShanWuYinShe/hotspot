<script lang="ts">
	import { goto } from '$app/navigation';
	import { UserPlus, Check } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import BorderedList from '@/components/data-display/BorderedList.svelte';
	import InteractiveListRow from '@/components/navigation/InteractiveListRow.svelte';
	import MetricRow, { type MetricItem } from '@/components/data-display/MetricRow.svelte';
	import { useRelationships } from '@/lib/profile-data.svelte';
	import { ROUTES } from '@/lib/routes';

	// 粉丝分区。

	let { embedded = false }: { embedded?: boolean } = $props();

	let fanFilter = $state<'all' | 'mutual' | 'pending'>('all');

	const { followers } = useRelationships();
	let followedBack = $state<Set<string>>(new Set(followers.filter((f) => f.isMutual).map((f) => f.id)));

	const toggleFollow = (id: string) => {
		const next = new Set(followedBack);
		next.has(id) ? next.delete(id) : next.add(id);
		followedBack = next;
	};

	const filtered = $derived(
		followers.filter((f) => {
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
		<div class="flex items-center justify-between px-[var(--page-gutter)] py-4 shrink-0" style="border-bottom:1px solid var(--border-default);">
			<div>
				<div class="flex items-center gap-2.5">
					<span class="text-text-primary text-[15px]">粉丝</span>
					<span class="px-2 py-0.5 rounded-full bg-accent-default text-white text-xs">
						{followers.length} 人
					</span>
				</div>
				<p class="mt-1 text-text-secondary text-xs">关注你的人，优先处理待回关关系</p>
			</div>
			<div class="flex items-center gap-2">
				{#if pendingCount > 0}
					<span class="px-2 py-0.5 rounded-full text-status-warning text-xs" style="background:color-mix(in srgb, var(--status-warning) 13%, transparent);border:1px solid color-mix(in srgb, var(--status-warning) 24%, transparent);">
						{pendingCount} 待回关
					</span>
				{/if}
			</div>
		</div>
	{/if}

	<!-- Stats row — flat joined cells -->
	<div class="px-[var(--page-gutter)] py-3 shrink-0">
		<MetricRow
			items={[
				{ label: '总粉丝', value: followers.length },
				{ label: '互相关注', value: mutualCount, tone: 'positive' },
				{ label: '待回关', value: pendingCount, tone: 'warning' }
			]}
		/>
	</div>

	<!-- Filter pills — ghost -->
	<div class="flex gap-1.5 px-[var(--page-gutter)] pb-3 shrink-0">
		{#each [{ id: 'all', label: '全部' }, { id: 'mutual', label: '互相关注' }, { id: 'pending', label: '待回关' }] as pill}
			{@const active = fanFilter === pill.id}
			<button
				type="button"
				class="hover-btn px-3 py-1 rounded-full outline-none surface-ghost text-xs tap-93 {active ? 'bg-accent-soft text-accent-default font-medium' : 'text-text-secondary'}"
				style={active ? 'border-color:var(--accent-border);' : ''}
				onclick={() => (fanFilter = pill.id as 'all' | 'mutual' | 'pending')}
			>
				{pill.label}
			</button>
		{/each}
	</div>

		<!-- Fan list -->
	{#if embedded}
		<div class="grid w-full grid-cols-1 gap-4 px-[var(--page-gutter)] pb-8 md:grid-cols-2 xl:grid-cols-3">
			{#if filtered.length === 0}
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
						<InteractiveListRow tone="card" class="flex items-center gap-3 p-3.5 cursor-pointer"
							onClick={() => void goto(`${ROUTES.user(f.id)}?from=relationships`)}
							ariaLabel={`查看 ${f.name} 的主页`}
						>
							<Avatar ring src={f.avatar} name={f.name} size={48} />
							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-2">
									<span class="text-text-primary text-base">{f.name}</span>
									{#if isMutual}
										<span class="flex items-center gap-0.5 px-1.5 py-0.5 rounded-full text-status-online text-xxs" style="background:var(--color-online-bg);">
											<Check style="width:9px;height:9px;" strokeWidth={2.5} />
											互关
										</span>
									{/if}
								</div>
								<p class="text-text-secondary text-xs mt-0.5">
									{f.handle} · {fanLabel(f.joinedDays)}关注了你
								</p>
							</div>
							<button
								type="button"
								class="hover-btn flex items-center gap-1 px-3 py-1 rounded-full outline-none shrink-0 text-xs tap-93 {isMutual ? 'bg-transparent text-text-secondary' : 'bg-accent-default text-white'}"
								style={isMutual ? 'border:1px solid var(--border-strong);' : 'border:1px solid transparent;'}
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
						</InteractiveListRow>
					{/each}
				{/if}
		</div>
	{:else}
		<div class="px-[var(--page-gutter)] pb-6">
			{#if filtered.length === 0}
				<EmptyState title="还没有粉丝" description="分享你的瞬间，让更多人发现你。">
					{#snippet icon()}
						<UserPlus size={18} />
					{/snippet}
				</EmptyState>
			{:else}
			<BorderedList>
					{#each filtered as f (f.id)}
						{@const isMutual = followedBack.has(f.id)}
						<InteractiveListRow class="flex items-center gap-3 p-3 cursor-pointer"
							onClick={() => void goto(`${ROUTES.user(f.id)}?from=relationships`)}
							ariaLabel={`查看 ${f.name} 的主页`}
						>
							<Avatar ring src={f.avatar} name={f.name} size={42} />
							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-2">
									<span class="text-text-primary text-base">{f.name}</span>
									{#if isMutual}
										<span class="flex items-center gap-0.5 px-1.5 py-0.5 rounded-full text-status-online text-xxs" style="background:var(--color-online-bg);">
											<Check style="width:9px;height:9px;" strokeWidth={2.5} />
											互关
										</span>
									{/if}
								</div>
								<p class="text-text-secondary text-xs mt-0.5">
									{f.handle} · {fanLabel(f.joinedDays)}关注了你
								</p>
							</div>
							<button
								type="button"
								class="hover-btn flex items-center gap-1 px-3 py-1 rounded-full outline-none shrink-0 text-xs tap-93 {isMutual ? 'bg-transparent text-text-secondary' : 'bg-accent-default text-white'}"
								style={isMutual ? 'border:1px solid var(--border-strong);' : 'border:1px solid transparent;'}
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
						</InteractiveListRow>
					{/each}
				</BorderedList>
				{/if}
			</div>
		{/if}
	</div>
