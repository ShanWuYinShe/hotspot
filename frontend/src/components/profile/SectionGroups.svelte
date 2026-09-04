<script lang="ts">
	import { goto } from '$app/navigation';
	import { Users, ChevronRight, Clock, Shield } from '@lucide/svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
import InlineLoading from '@/components/data-display/InlineLoading.svelte';
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

<!-- 横向媒体卡网格（对齐 Figma：1/2 列，min-h 148px，左 banner + 右详情） -->
{#if loading}
	<InlineLoading class="py-16" label="加载我的群组…" />
{:else if needsAuth}
	<NeedsAuthNotice title="登录后查看" description="登录后即可查看你加入的群组" />
{:else if error}
	<LoadError message={error} onRetry={retry} />
{:else if filtered.length > 0}
	<div class="grid grid-cols-1 gap-3 pb-6 sm:grid-cols-2">
		{#each filtered as g (g.id)}
			<button
				type="button"
				class="card-hover tap-97 group flex min-h-37 w-full overflow-hidden rounded-md border border-border-default bg-surface-primary shadow-raised-md text-left outline-none focus-visible:ring-2 focus-visible:ring-accent-default"
				aria-label={`进入群组：${g.name}`}
				onclick={() => void goto(`${ROUTES.group(g.id)}?from=me`)}
			>
				<!-- 左侧 banner（固定宽 96-112px；无头像群组用 accent 渐变兜底，不渲染空 src img） -->
				<div class="relative w-24 shrink-0 overflow-hidden sm:w-28">
				{#if g.banner}
					<img src={g.banner} alt="" loading="lazy" class="h-full w-full object-cover" />
				{:else}
					<div class="surface-accent-gradient h-full w-full"></div>
				{/if}
					<div class="absolute inset-0" style="background:var(--cst-surface-gallery-scrim);"></div>
					<div
						class="absolute left-2 top-2 badge"
						style="background:var(--cst-surface-gallery-label);color:var(--cst-content-inverse);backdrop-filter:blur(var(--cst-blur-sm));"
					>
						{#if g.role === '管理员'}
							<Shield class="mr-0.5 inline-block size-2.5"  strokeWidth={2} />
						{/if}
						{g.role}
					</div>
					<div class="absolute bottom-2 left-2 flex items-center gap-1 text-content-inverse text-xxs">
						<span class="rounded-full size-1.5 bg-status-online"></span>
						{g.online}
					</div>
				</div>

				<!-- 右侧详情列 -->
					<div class="flex min-w-0 flex-1 flex-col p-3">
						<div class="flex min-w-0 items-center gap-2">
							{#if g.avatar}
								<img
									src={g.avatar}
									alt=""
									loading="lazy"
									class="h-8 w-8 shrink-0 rounded-sm border border-border-default object-cover"
								/>
							{:else}
								<div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-sm border border-border-default bg-accent-soft">
									<Users class="size-3.5 text-accent-default" strokeWidth={1.75} />
								</div>
							{/if}
							<p class="min-w-0 flex-1 truncate text-sm font-medium text-text-primary">{g.name}</p>
						<ChevronRight class="shrink-0 text-text-secondary size-4"  strokeWidth={1.75} />
					</div>

					<p class="mt-2 truncate text-xs text-text-secondary">{g.desc}</p>

					<div class="mt-auto flex items-center justify-between pt-3 text-xs text-text-secondary">
						<span class="flex items-center gap-1">
							<Users class="size-3"  strokeWidth={1.5} />
							{g.members} 位成员
						</span>
						<span class="flex items-center gap-1">
							<Clock class="size-3"  strokeWidth={1.5} />
							{g.lastActive}
						</span>
					</div>
				</div>
			</button>
		{/each}
		</div>

	{:else if searchQ}
		<!-- 搜索无结果与初始无群组区分语义 -->
		<EmptyState bare title="没有找到匹配的群组" description="换个关键词试试。" />
	{:else}
		<EmptyState bare title="还没有加入群组" description="去附近群组发现感兴趣的小组。" />
	{/if}
