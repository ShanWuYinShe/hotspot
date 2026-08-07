<script lang="ts">
	import { goto } from '$app/navigation';
	import { Users, ChevronRight, Clock, Shield, Search } from '@lucide/svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import { useMyGroups } from '@/lib/profile-data.svelte';
	import { ROUTES } from '@/lib/routes';

	// 我的群组分区（对齐 Figma Make ProfileSections SectionGroups）。
	// 横向媒体卡布局：固定宽左侧 banner + 右侧详情列（avatar+name+chevron+desc+members+lastActive），min-h 148px。
	// 纯内容组件：滚动与 gutter 由外层 StandardPageLayout/ContentColumn 承担，不自建容器。

	let searchQ = $state('');

	const { groups } = useMyGroups();
	const filtered = $derived(groups.filter((g) => searchQ === '' || g.name.toLowerCase().includes(searchQ.toLowerCase())));
	const totalOnline = $derived(groups.reduce((sum, g) => sum + (g.online ?? 0), 0));
</script>

<!-- 搜索 + 统计行（对齐 Figma SectionGroups header） -->
<div class="pb-3 pt-4">
	<SearchInput value={searchQ} onChange={(v) => (searchQ = v)} placeholder="搜索我的群组…" class="px-3.5 py-2.5" />
	<div class="mt-3 flex items-center justify-between">
		<p class="text-sm text-text-secondary">共 {groups.length} 个群组</p>
		<div class="flex items-center gap-1.5 text-sm text-status-online">
			<span class="rounded-full size-1.5"  style="background:var(--cst-status-online)"></span>
			{totalOnline} 人在线
		</div>
	</div>
</div>

<!-- 横向媒体卡网格（对齐 Figma：1/2 列，min-h 148px，左 banner + 右详情） -->
<div class="grid grid-cols-1 gap-3 pb-6 sm:grid-cols-2">
		{#each filtered as g (g.id)}
			<button
				type="button"
				class="card-hover tap-94 group flex min-h-37 w-full overflow-hidden rounded-relaxed border text-left outline-none focus-visible:ring-2 focus-visible:ring-accent-default"
				style="background:var(--cst-surface-primary);border-color:var(--cst-border-default);box-shadow:var(--shadow-neu-lg);"
				aria-label={`进入群组：${g.name}`}
				onclick={() => void goto(`${ROUTES.group(g.id)}?from=me`)}
			>
				<!-- 左侧 banner（固定宽 96-112px） -->
				<div class="relative w-24 shrink-0 overflow-hidden sm:w-28">
					<img src={g.banner} alt="" class="image-hover-frame__image h-full w-full object-cover" />
					<div class="absolute inset-0" style="background:var(--cst-surface-gallery-scrim);"></div>
					<div
						class="absolute left-2 top-2 rounded-full px-1.5 py-0.5 text-xxs"
						style="background:var(--cst-surface-gallery-label);color:var(--cst-content-inverse);backdrop-filter:blur(8px);"
					>
						{#if g.role === '管理员'}
							<Shield class="mr-0.5 inline-block size-2.25"  strokeWidth={2} />
						{/if}
						{g.role}
					</div>
					<div class="absolute bottom-2 left-2 flex items-center gap-1 text-content-inverse text-xxs">
						<span class="rounded-full size-1.5"  style="background:var(--cst-status-online)"></span>
						{g.online}
					</div>
				</div>

				<!-- 右侧详情列 -->
				<div class="flex min-w-0 flex-1 flex-col p-3">
					<div class="flex min-w-0 items-center gap-2">
						<img
							src={g.avatar}
							alt=""
							class="h-8 w-8 shrink-0 rounded-tight object-cover"
							style="border:1px solid var(--cst-border-default);"
						/>
						<p class="min-w-0 flex-1 truncate text-base font-medium text-text-primary">{g.name}</p>
						<ChevronRight class="shrink-0 text-text-secondary size-3.75"  strokeWidth={1.7} />
					</div>

					<p class="mt-2 truncate text-xs text-text-secondary">{g.desc}</p>

					<div class="mt-auto flex items-center justify-between pt-3 text-xs text-text-secondary">
						<span class="flex items-center gap-1">
							<Users class="size-3"  strokeWidth={1.5} />
							{g.members} 位成员
						</span>
						<span class="flex items-center gap-1">
							<Clock class="size-2.75"  strokeWidth={1.5} />
							{g.lastActive}
						</span>
					</div>
				</div>
			</button>
		{/each}
	</div>

	{#if filtered.length === 0}
		<div class="flex flex-col items-center justify-center gap-2 py-16">
			<Search class="text-text-secondary size-5.5"  strokeWidth={1.4} />
			<p class="text-base text-text-secondary">没有找到匹配的群组</p>
		</div>
	{/if}
