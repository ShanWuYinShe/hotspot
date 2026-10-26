<script lang="ts">
	import { goto } from '$app/navigation';
	import { platform } from '@/stores/platform.svelte';
	import { Users, ChevronRight, Clock, Shield } from '@lucide/svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
import SkeletonFeed from '@/components/data-display/SkeletonFeed.svelte';
	import NearbyImageCard from '@/components/data-display/NearbyImageCard.svelte';
	import SectionGroupApplications from '@/components/profile/SectionGroupApplications.svelte';
	import { useMyGroups } from '@/lib/profile-data.svelte';
	import { ROUTES } from '@/lib/routes';

	// 我的群组分区（对齐 Figma Make ProfileSections SectionGroups）。
	// 手机为横向紧凑卡（图左 40% + 信息右，h-32）：2 个群不再独占整屏；
	// 平板/桌面为 4:3 图片卡网格。纯内容组件：滚动与 gutter 由外层
	// StandardPageLayout/ContentColumn 承担，不自建容器。

	const device = $derived(platform.platform);
	const isMobile = $derived(device === 'mobile');

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
     纯内容组件：顶部间距由外层 ContentColumn 的 pt-4 承担，不自建 pt 避免双重叠加。
     未登录隐藏（Gemini 审查 r14：登录引导与不可用控件并存）。 -->
{#if !needsAuth}
<!-- 统计(左) + 搜索(右限宽)：桌面同行；移动端搜索独占一行（固定 w-72 同行在 390 视口横向溢出被裁） -->
<div class="pb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
	<p class="text-sm text-text-secondary shrink-0">共 {groups.length} 个群组<!-- 分隔符原用 text-border-strong（border 色 token 当正文色，s3 P2-1）→ 文字色 + 弱化 -->
	<span class="mx-2 text-text-secondary opacity-dim" aria-hidden="true">·</span><!-- 零在线弱化为次要灰、不亮绿点( Gemini 审查 r14:零数据高亮绿造成视觉误导 ) --><span class="inline-flex items-center gap-1.5 {totalOnline > 0 ? 'text-status-online' : 'text-text-secondary'}">
		{#if totalOnline > 0}
			<span class="rounded-full size-1.5 bg-status-online"></span>
		{/if}
		{totalOnline} 人在线
	</span></p>
	<SearchInput value={searchQ} onChange={(v) => (searchQ = v)} placeholder="搜索我的群组…" class="w-full px-3 py-2.5 sm:w-72 sm:shrink-0" />
</div>
{/if}

<!-- 统一群组卡片网格：对齐全站图片优先 NearbyImageCard 原语与响应式多列布局。
     与下方入群申请区块的间距由后者的 mt-6 单侧提供，网格不再叠 pb（否则叠加成 48）。 -->
{#if loading}
	<!-- 首屏骨架屏：移动端横排紧凑卡（h-32），平板/桌面 4:3 卡网格，与真实群组卡同构 -->
	<SkeletonFeed
		variant="card"
		horizontal={isMobile}
		count={isMobile ? 4 : 8}
		class={isMobile
			? 'grid grid-cols-1 gap-4'
			: 'grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5'}
		label="加载我的群组…"
	/>
{:else if needsAuth}
	<NeedsAuthNotice spotlight title="登录后查看" description="登录后即可查看你加入的群组" />
{:else if error}
	<LoadError message={error} onRetry={retry} />
{:else if filtered.length > 0}
	<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
		{#each filtered as g (g.id)}
			<NearbyImageCard
				imageSrc={g.banner || g.avatar}
				imageAlt={g.name}
				onClick={() => void goto(`${ROUTES.group(g.id)}?from=me`)}
				horizontal={isMobile}
				class={isMobile ? 'h-32 rounded-md' : 'aspect-[4/3] rounded-md'}
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

				<!-- 右上：在线人数徽章（手机横向卡的信息面板占满右半，移至 footer meta 行） -->
				{#if !isMobile}
					<div
						class="absolute top-2.5 right-2.5 z-[2] inline-flex h-5 items-center justify-center gap-1 px-2.5 rounded-full bg-surface-overlay-strong text-content-inverse text-xxs font-medium leading-none"
						style="backdrop-filter:blur(var(--cst-blur-sm));"
					>
						<span class="rounded-full size-1.5 bg-status-online"></span>
						<span>{g.online} 在线</span>
					</div>
				{/if}

				{#snippet footer()}
					<div class="px-3 pb-2.5 pt-2.5">
						<div class="flex items-center justify-between gap-2">
							<p class="truncate text-sm font-semibold text-text-primary">{g.name}</p>
							<ChevronRight class="size-3.5 shrink-0 text-text-secondary" strokeWidth={1.75} />
						</div>
						{#if g.desc}
							<p class="mt-1 truncate text-xs text-text-secondary">{g.desc}</p>
						{/if}
						<div class="mt-2 flex items-center justify-between gap-2 text-xxs text-text-secondary">
							<span class="flex items-center gap-1">
								<Users class="size-3" strokeWidth={1.5} />
								{g.members} 位成员
							</span>
							{#if isMobile}
								<span class="flex items-center gap-1">
									<span class="rounded-full size-1.5 bg-status-online"></span>
									{g.online} 在线
								</span>
							{:else if g.lastActive}
								<span class="flex items-center gap-1">
									<Clock class="size-3" strokeWidth={1.5} />
									{g.lastActive}
								</span>
							{/if}
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
		<EmptyState spotlight bare title="还没有加入群组" description="去附近群组发现感兴趣的小组。" />
	{/if}

	<!-- 我的入群申请进度（pending 可撤回；未登录时区块自隐藏） -->
	<SectionGroupApplications />