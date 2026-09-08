<script lang="ts">
	import { goto } from '$app/navigation';
	import { Users, ChevronRight, Clock, Shield } from '@lucide/svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import NearbyImageCard from '@/components/data-display/NearbyImageCard.svelte';
	import { useMyGroups } from '@/lib/profile-data.svelte';
	import { ROUTES } from '@/lib/routes';

	// 我的群组分区（对齐 Figma Make ProfileSections SectionGroups）。
	// 横向媒体卡布局：固定宽左侧 banner + 右侧详情列（avatar+name+chevron+desc+members+lastActive），min-h 148px。
	// 纯内容组件：滚动与 gutter 由外层 StandardPageLayout/ContentColumn 承担，不自建容器。

	let searchQ = $state('');

	// getter 对象解构会拿到非响应式快照，$derived 桥接
	const myGroups = useMyGroups();
	const groups = $derived(myGroups.groups);
	const loading = $derived(myGroups.loading);
	const error = $derived(myGroups.error);
const needsAuth = $derived(myGroups.needsAuth);
	const retry = myGroups.retry;
	const filtered = $derived(groups.filter((g) => searchQ === '' || g.name.toLowerCase().includes(searchQ.toLowerCase())));
	const totalOnline = $derived(groups.reduce((sum, g) => sum + (g.online ?? 0), 0));
</script>

<!-- 搜索 + 统计行（对齐 Figma SectionGroups header）。
     纯内容组件：顶部间距由外层 ContentColumn 的 pt-4 承担，不自建 pt 避免双重叠加。 -->
<div class="pb-3">
	<SearchInput value={searchQ} onChange={(v) => (searchQ = v)} placeholder="搜索我的群组…" class="px-3.5 py-2.5" />
	<div class="mt-3 flex items-center justify-between">
		<p class="text-sm text-text-secondary">共 {groups.length} 个群组</p>
		<div class="flex items-center gap-1.5 text-sm text-status-online">
			<span class="rounded-full size-1.5 bg-status-online"></span>
			{totalOnline} 人在线
		</div>
	</div>
</div>

<!-- 统一群组卡片网格：对齐全站图片优先 NearbyImageCard 原语与响应式多列布局 -->
{#if loading}
	<InlineLoading class="py-16" label="加载我的群组…" />
{:else if needsAuth}
	<NeedsAuthNotice title="登录后查看" description="登录后即可查看你加入的群组" />
{:else if error}
	<LoadError message={error} onRetry={retry} />
{:else if filtered.length > 0}
	<div class="grid grid-cols-1 gap-4 pb-6 sm:grid-cols-2 lg:grid-cols-3">
		{#each filtered as g (g.id)}
			<NearbyImageCard
				imageSrc={g.banner || g.avatar}
				imageAlt={g.name}
				onClick={() => void goto(`${ROUTES.group(g.id)}?from=me`)}
				class="aspect-[4/3] rounded-md"
			>
				{#snippet placeholder()}
					<span class="text-accent-default text-3xl font-semibold">{Array.from(g.name)[0] ?? '群'}</span>
				{/snippet}

				<!-- 左上：角色身份徽章 -->
				<div
					class="absolute top-2.5 left-2.5 z-[2] inline-flex h-5 items-center justify-center gap-1 px-2.5 rounded-full bg-surface-overlay-strong text-content-inverse text-xxs font-medium leading-none"
					style="backdrop-filter:blur(var(--cst-blur-sm));"
				>
					{#if g.role === '管理员'}
						<Shield class="size-2.5 mr-0.5 text-accent-default inline" strokeWidth={2} />
					{/if}
					{g.role}
				</div>

				<!-- 右上：在线人数徽章 -->
				<div
					class="absolute top-2.5 right-2.5 z-[2] inline-flex h-5 items-center justify-center gap-1 px-2.5 rounded-full bg-surface-overlay-strong text-content-inverse text-xxs font-medium leading-none"
					style="backdrop-filter:blur(var(--cst-blur-sm));"
				>
					<span class="rounded-full size-1.5 bg-status-online"></span>
					<span>{g.online} 在线</span>
				</div>

				{#snippet footer()}
					<div class="px-3 pb-2.5 pt-2.5">
						<div class="flex items-center justify-between gap-2">
							<p class="truncate text-sm font-semibold text-text-primary">{g.name}</p>
							<ChevronRight class="size-3.5 shrink-0 text-text-secondary" strokeWidth={1.75} />
						</div>
						{#if g.desc}
							<p class="mt-1 truncate text-xs text-text-secondary">{g.desc}</p>
						{/if}
						<div class="mt-2 flex items-center justify-between text-xxs text-text-secondary">
							<span class="flex items-center gap-1">
								<Users class="size-3" strokeWidth={1.5} />
								{g.members} 位成员
							</span>
							<span class="flex items-center gap-1">
								<Clock class="size-3" strokeWidth={1.5} />
								{g.lastActive}
							</span>
						</div>
					</div>
				{/snippet}
			</NearbyImageCard>
		{/each}
	</div>

	{:else if searchQ}
		<!-- 搜索无结果与初始无群组区分语义 -->
		<EmptyState bare title="没有找到匹配的群组" description="换个关键词试试。" />
	{:else}
		<EmptyState bare title="还没有加入群组" description="去附近群组发现感兴趣的小组。" />
	{/if}
