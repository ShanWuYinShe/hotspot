<script lang="ts">
	import { Users } from '@lucide/svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
	import SkeletonFeed from '@/components/data-display/SkeletonFeed.svelte';
	import ProfileGroupTile from '@/components/profile/ProfileGroupTile.svelte';
	import SectionGroupApplications from '@/components/profile/SectionGroupApplications.svelte';
	import { useMyGroups } from '@/lib/profile-data.svelte';

	// 我的群组分区（R13 改横排列表）：
	// 「我的群组」是列表型内容，原桌面 5 列图片网格在只有 2 个群时右侧留出成片空场；
	// 改为居中限宽的横排卡列表（图左 40% + 信息右），并把管理入口（… → 退出群组）
	// 收在每张卡右上（R13 评审 P1-7 批准）。退出成功后从本地列表移除，不重拉。

	let searchQ = $state('');
	let removedIds = $state(new Set<string>());

	// getter 对象解构会拿到非响应式快照，$derived 桥接
	const myGroups = useMyGroups();
	const groups = $derived(myGroups.groups);
	const loading = $derived(myGroups.loading);
	const error = $derived(myGroups.error);
	const needsAuth = $derived(myGroups.needsAuth);
	const retry = myGroups.retry;
	const visibleGroups = $derived(groups.filter((g) => !removedIds.has(g.id)));
	const filtered = $derived(
		visibleGroups.filter((g) => searchQ === '' || g.name.toLowerCase().includes(searchQ.toLowerCase()))
	);
	const totalOnline = $derived(visibleGroups.reduce((sum, g) => sum + (g.online ?? 0), 0));

	function handleLeft(id: string) {
		removedIds = new Set([...removedIds, id]);
	}
</script>

<!-- 工具行（统计 + 搜索）：与下方列表同宽居中，避免宽屏下工具行与列表左右不齐 -->
{#if !needsAuth && !loading}
	<div class="mx-auto mb-4 flex w-full max-w-3xl flex-col gap-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
		<p class="shrink-0 text-sm text-text-secondary">
			共 {visibleGroups.length} 个群组
			<span class="mx-2 text-text-secondary opacity-dim" aria-hidden="true">·</span>
			<span class="inline-flex items-center gap-1.5 {totalOnline > 0 ? 'text-status-online' : 'text-text-secondary'}">
				{#if totalOnline > 0}
					<span class="size-1.5 rounded-full bg-status-online"></span>
				{/if}
				{totalOnline} 人在线
			</span>
		</p>
		<SearchInput value={searchQ} onChange={(v) => (searchQ = v)} placeholder="搜索我的群组…" class="w-full px-3 py-2.5 sm:w-72 sm:shrink-0" />
	</div>
{/if}

{#if loading}
	<!-- 首屏骨架屏：与真实横排群组卡同构（图左 40% + 右侧两行信息），骨架→内容零位移 -->
	<div class="mx-auto w-full max-w-3xl">
		<SkeletonFeed variant="card" horizontal count={4} class="grid grid-cols-1 gap-3" label="加载我的群组…" />
	</div>
{:else if needsAuth}
	<NeedsAuthNotice spotlight title="登录后查看" description="登录后即可查看你加入的群组" />
{:else if error}
	<LoadError message={error} onRetry={retry} />
{:else if filtered.length > 0}
	<div class="mx-auto flex w-full max-w-3xl flex-col gap-3">
		{#each filtered as g (g.id)}
			<ProfileGroupTile group={g} onLeft={() => handleLeft(g.id)} />
		{/each}
	</div>

	{:else if searchQ}
		<!-- 搜索无结果与初始无群组区分语义 -->
		<EmptyState
			iconTone="accent"
			class="py-16"
			title="没有找到匹配的群组"
			description="换个关键词试试，或者去附近群组发现新的小圈子。"
		>
			{#snippet icon()}
				<Users class="size-7" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	{:else}
		<EmptyState
			spotlight
			iconTone="accent"
			title="还没有加入群组"
			description="去附近群组发现感兴趣的小组，加入后就能在这里快速进入。"
		>
			{#snippet icon()}
				<Users class="size-7" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	{/if}

	<!-- 我的入群申请进度（pending 可撤回；未登录时区块自隐藏） -->
	<div class="mx-auto w-full max-w-3xl">
		<SectionGroupApplications />
	</div>
