<script lang="ts">
	import { Heart, Users } from '@lucide/svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import { ROUTES } from '@/lib/routes';
	import { formatCardTime } from '@/lib/utils';
	import type { GroupItem, MomentItem } from '@/api/types';

	// W06：他人主页的「群组 / 反应过的瞬间」两个集合分区。
	// 数据由 useUserProfileData 并行加载；10200 privacy_setting 表示对方未公开，
	// 此时用 SectionBlocked 明说原因，而不是展示成"空列表"。
	let {
		groups,
		reactedMoments,
		groupsHidden,
		reactedHidden,
		loading,
		userName
	}: {
		groups: GroupItem[];
		reactedMoments: MomentItem[];
		groupsHidden: boolean;
		reactedHidden: boolean;
		loading: boolean;
		userName: string;
	} = $props();

	const visibleGroups = $derived(groups.slice(0, 6));
	const visibleMoments = $derived(reactedMoments.slice(0, 6));
</script>

<section class="mt-6 grid gap-6 lg:grid-cols-2" data-user-collections>
	<div>
		<h2 class="mb-2 flex items-center gap-2 text-sm font-medium text-text-primary">
			<Users class="size-4 text-accent-default" strokeWidth={1.75} />
			群组
			{#if !groupsHidden && !loading}
				<span class="text-text-secondary">{groups.length}</span>
			{/if}
		</h2>
		{#if loading}
			<InlineLoading label="加载群组…" />
		{:else if groupsHidden}
			<p class="rounded-md bg-surface-subtle px-3 py-2 text-sm text-text-secondary">对方未公开群组列表。</p>
		{:else if visibleGroups.length === 0}
			<p class="rounded-md bg-surface-subtle px-3 py-2 text-sm text-text-secondary">{userName} 还没有加入群组。</p>
		{:else}
			<ul class="grid gap-2">
				{#each visibleGroups as item (item.group.group_id)}
					<li>
						<a
							href={ROUTES.group(item.group.group_id)}
							class="hover-btn flex items-center gap-3 rounded-md bg-surface-primary px-3 py-2 shadow-xs outline-none focus-visible:ring-2 focus-visible:ring-accent-default"
						>
							<span class="flex size-9 shrink-0 items-center justify-center overflow-hidden rounded-full bg-surface-subtle">
								{#if item.group.avatar_url}
									<img src={item.group.avatar_url} alt="" class="size-full object-cover" loading="lazy" />
								{:else}
									<Users class="size-4 text-text-secondary" strokeWidth={1.75} />
								{/if}
							</span>
							<span class="min-w-0 flex-1">
								<span class="block truncate text-sm font-medium text-text-primary">{item.group.name}</span>
								<span class="block text-xs text-text-secondary">{item.group.members_count} 名成员</span>
							</span>
						</a>
					</li>
				{/each}
			</ul>
		{/if}
	</div>

	<div>
		<h2 class="mb-2 flex items-center gap-2 text-sm font-medium text-text-primary">
			<Heart class="size-4 text-accent-default" strokeWidth={1.75} />
			反应过的瞬间
			{#if !reactedHidden && !loading}
				<span class="text-text-secondary">{reactedMoments.length}</span>
			{/if}
		</h2>
		{#if loading}
			<InlineLoading label="加载瞬间…" />
		{:else if reactedHidden}
			<p class="rounded-md bg-surface-subtle px-3 py-2 text-sm text-text-secondary">对方未公开反应记录。</p>
		{:else if visibleMoments.length === 0}
			<p class="rounded-md bg-surface-subtle px-3 py-2 text-sm text-text-secondary">{userName} 还没有反应过的瞬间。</p>
		{:else}
			<ul class="grid grid-cols-2 gap-2 sm:grid-cols-3">
				{#each visibleMoments as moment (moment.moment_id)}
					<li>
						<a
							href={ROUTES.moment(moment.moment_id)}
							class="hover-btn block overflow-hidden rounded-md bg-surface-primary shadow-xs outline-none focus-visible:ring-2 focus-visible:ring-accent-default"
						>
							<span class="block aspect-square overflow-hidden bg-surface-subtle">
								{#if moment.images?.[0]?.url}
									<img src={moment.images[0].url} alt="" class="size-full object-cover" loading="lazy" />
								{:else}
									<span class="flex size-full items-center justify-center text-text-secondary">
										<Heart class="size-5" strokeWidth={1.5} />
									</span>
								{/if}
							</span>
							<span class="block px-2 py-1.5">
								<span class="line-clamp-2 text-xs text-text-secondary">{moment.description || '无文字内容'}</span>
								<span class="mt-0.5 block text-[11px] text-text-secondary">{formatCardTime(moment.created_at)}</span>
							</span>
						</a>
					</li>
				{/each}
			</ul>
		{/if}
	</div>
</section>
