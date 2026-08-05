<script lang="ts">
	import { ArrowLeft, MapPin, Clock, Users, Share2, Calendar, ChevronRight, Check, Star, X } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import SplitPageLayout from '@/components/layout/page/SplitPageLayout.svelte';
	import RailSection from '@/components/layout/desktop/RailSection.svelte';
	import RailUserRow from '@/components/layout/desktop/RailUserRow.svelte';
	import RailLocationMini from '@/components/layout/desktop/RailLocationMini.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import EmbeddedList from '@/components/data-display/EmbeddedList.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import { ROUTES } from '@/lib/routes';
	import { getEventDetail, getSimilarEvents } from '@/api';

	// 活动详情（对齐 Figma Make EventDetailPage）。
	// 全幅封面 + 报名 CTA + 发起人卡 + 参与者 sheet + 4 段桌面 rail。
	// 数据来自 MSW mock 端点（spec 尚未定义 events，后端契约就绪后接入真实端点）。

	interface Participant {
		id: string;
		name: string;
		avatar: string;
	}
	interface EventData {
		id: string;
		title: string;
		tag: string;
		color: string;
		bg: string;
		cover: string;
		description: string;
		location: string;
		date: string;
		time: string;
		host: { name: string; avatar: string };
		participants: Participant[];
		maxParticipants: number;
		distance: string;
		highlights: string[];
	}
	interface SimilarEvent {
		id: string;
		name: string;
		sub: string;
		avatar: string;
	}

	const eventIdParam = $derived(page.params.eventId ?? '');
	// spec: event_id 为 string，客户端不解析 ID 内容，直接透传使用。

	let ev = $state<EventData | null>(null);
	let similarEvents = $state<SimilarEvent[]>([]);
	let loading = $state(true);
	/** 空数据壳：仅避免 snippet 提升后对 ev 的空值收窄问题，不承载任何假数据。 */
	const EMPTY_EVENT: EventData = {
		id: '', title: '', tag: '', color: '', bg: '', cover: '', description: '',
		location: '', date: '', time: '', host: { name: '', avatar: '' },
		participants: [], maxParticipants: 0, distance: '', highlights: []
	};
	const safeEv = $derived(ev ?? EMPTY_EVENT);

	$effect(() => {
		let cancelled = false;
		void Promise.all([
			getEventDetail(eventIdParam).catch(() => ({ code: -1, data: null })),
			getSimilarEvents({ pagination: { limit: 3 } }).catch(() => ({ code: -1, data: { items: [] } }))
		]).then(([detailRes, similarRes]) => {
			if (cancelled) return;
			const detail = detailRes as unknown as { code: number; data: EventData | null };
			const similar = similarRes as unknown as { code: number; data: { items: SimilarEvent[] } | null };
			if (detail.code === 0 && detail.data) ev = detail.data;
			if (similar.code === 0 && similar.data) similarEvents = similar.data.items ?? [];
			loading = false;
		});
		return () => {
			cancelled = true;
		};
	});

	let joined = $state(false);
	let showParticipants = $state(false);

	const fillRate = $derived(ev ? Math.round((safeEv.participants.length / safeEv.maxParticipants) * 100) : 0);
	const evTone = (amount: number) => `color-mix(in srgb, ${safeEv.color || 'var(--accent-default)'} ${amount}%, transparent)`;
	const walkMinutes = $derived(ev ? Math.ceil(parseFloat(safeEv.distance) * 12) : 0);
</script>

{#if ev}
<SplitPageLayout contentClassName="pb-6">
	<!-- 全幅封面（负 margin 抵消 SplitPageLayout 的 gutter） -->
	<div class="relative shrink-0 overflow-hidden -mx-[var(--page-gutter)]" style="height:clamp(200px, 32vw, 280px);">
				<img src={safeEv.cover} alt={safeEv.title} class="absolute inset-0 h-full w-full object-cover" />
				<div class="absolute inset-0" style="background:linear-gradient(to bottom, transparent 30%, {safeEv.bg} 100%);"></div>

				<!-- 返回 -->
				<div class="absolute top-4 left-4 flex gap-2">
					<ActionIconButton icon={ArrowLeft} label="返回" onClick={() => history.back()} variant="overlay" tone="inverse" />
				</div>
				<div class="absolute top-4 right-4">
					<ActionIconButton icon={Share2} label="分享活动" variant="overlay" tone="inverse" />
				</div>

				<!-- 标签 -->
				<div class="absolute bottom-4 left-4">
					<span class="rounded-full px-3 py-1 text-white text-sm" style="background:{safeEv.color};box-shadow:0 3px 10px {evTone(50)};">{safeEv.tag}</span>
				</div>
			</div>

	<!-- 内容（gutter 由 SplitPageLayout 提供） -->
	<div class="pt-5">
				<h1 class="text-text-primary text-[24px] mb-3">{safeEv.title}</h1>

				<!-- meta 行 -->
				<div class="mb-6 flex flex-col gap-2.5">
					{#each [{ icon: Calendar, text: safeEv.date }, { icon: Clock, text: safeEv.time }, { icon: MapPin, text: safeEv.location }] as meta}
						<div class="flex items-center gap-2.5">
							<div class="flex shrink-0 items-center justify-center rounded-full" style="width:30px;height:30px;background:color-mix(in srgb, {safeEv.color} 14%, transparent);">
								<meta.icon style="width:13px;height:13px;color:{safeEv.color};" strokeWidth={1.75} />
							</div>
							<span class="text-base text-text-secondary">{meta.text}</span>
						</div>
					{/each}
				</div>

				<!-- 发起人 -->
				<div class="mb-4 flex items-center gap-3 rounded-panel px-4 py-3.5" style="background:var(--surface-primary);box-shadow:var(--shadow-neu-lg);border:1px solid var(--border-default);">
					<div class="shrink-0 overflow-hidden rounded-full" style="width:40px;height:40px;border:2px solid {safeEv.color};">
						<img src={safeEv.host.avatar} alt={safeEv.host.name} class="h-full w-full object-cover" />
					</div>
					<div class="flex-1">
						<p class="text-base text-text-primary">{safeEv.host.name}</p>
						<div class="flex items-center gap-1">
							<Star style="width:10px;height:10px;color:var(--status-warning);" fill="var(--status-warning)" strokeWidth={1.5} />
							<span class="text-xs text-text-secondary">活动发起人</span>
						</div>
					</div>
					<button type="button" class="hover-btn tap-93 rounded-full px-3 py-1.5 text-white text-sm outline-none" style="background:{safeEv.color};box-shadow:0 3px 10px {evTone(40)};">关注</button>
				</div>

				<!-- 描述 -->
				<p class="mb-5 text-[14px] leading-[1.75] text-text-secondary">{safeEv.description}</p>

				<!-- 亮点 -->
				<div class="mb-5 grid gap-2.5" style="grid-template-columns:repeat(auto-fill, minmax(180px, 1fr));">
					{#each safeEv.highlights as h}
						<div class="flex items-center gap-2.5 rounded-control px-3.5 py-3" style="background:var(--surface-primary);box-shadow:var(--shadow-neu-md);border:1px solid var(--border-default);">
							<div class="flex shrink-0 items-center justify-center rounded-full" style="width:24px;height:24px;background:color-mix(in srgb, {safeEv.color} 14%, transparent);">
								<Check style="width:11px;height:11px;color:{safeEv.color};" strokeWidth={2.5} />
							</div>
							<span class="text-sm text-text-primary">{h}</span>
						</div>
					{/each}
				</div>

				<!-- 参与者 -->
				<div class="mb-5">
					<div class="mb-3 flex items-center justify-between">
						<p class="text-[14px] text-text-primary">参与者 · {safeEv.participants.length}/{safeEv.maxParticipants}</p>
						<button type="button" class="hover-btn tap-93 flex items-center gap-0.5 outline-none" style="color:var(--accent-default);" onclick={() => (showParticipants = true)}>
							<span class="text-sm">查看全部</span>
							<ChevronRight style="width:13px;height:13px;" strokeWidth={1.5} />
						</button>
					</div>

					<!-- 进度条 -->
					<div class="mb-3 overflow-hidden rounded-full" style="height:6px;background:var(--border-default);">
						<div style="width:{fillRate}%;height:100%;background:{safeEv.color};border-radius:3px;transition:width 0.8s ease-out;"></div>
					</div>

					<!-- 头像栈 -->
					<div class="flex items-center gap-1">
						{#each safeEv.participants as p, i (p.id)}
							<div class="overflow-hidden rounded-full" style="width:36px;height:36px;border:2px solid var(--surface-primary);margin-left:{i > 0 ? '-10px' : '0'};z-index:{safeEv.participants.length - i};">
								<img src={p.avatar} alt={p.name} class="h-full w-full object-cover" />
							</div>
						{/each}
						{#if safeEv.maxParticipants - safeEv.participants.length > 0}
							<div class="flex items-center justify-center rounded-full" style="width:36px;height:36px;border:2px solid var(--surface-primary);margin-left:-10px;background:{evTone(18)};">
								<span class="text-xs font-semibold" style="color:{safeEv.color};">+{safeEv.maxParticipants - safeEv.participants.length}</span>
							</div>
						{/if}
						<span class="ml-2 text-sm text-text-secondary">已报名</span>
					</div>
				</div>

				<!-- 距离信息 -->
				<div class="flex items-center gap-2 rounded-control px-4 py-3" style="background:{evTone(10)};border:1px solid {evTone(25)};">
					<MapPin style="width:14px;height:14px;color:{safeEv.color};" strokeWidth={1.5} />
					<span class="text-base text-text-secondary">距你 {safeEv.distance} · 步行约 {walkMinutes} 分钟</span>
				</div>
	</div>

	{#snippet footer()}
		<!-- sticky 底部 CTA（按钮保持 640px 阅读宽度，页面布局全宽） -->
		<div class="shrink-0 px-[var(--page-gutter)] pb-6 pt-3" style="background-color:var(--surface-primary);border-top:1px solid var(--border-default);">
			<div class="mx-auto w-full" style="max-width:640px;">
			<button
				type="button"
				onclick={() => (joined = !joined)}
				class="hover-btn tap-97 flex w-full items-center justify-center gap-2 rounded-feature py-4 text-[15px] outline-none transition-all duration-300"
				style={joined
					? `background:var(--surface-primary);border:2px solid ${safeEv.color};color:${safeEv.color};box-shadow:0 0 0 4px ${evTone(18)};`
					: `background:${safeEv.color};color:white;box-shadow:0 8px 24px ${evTone(45)};border:none;`}
			>
				{#if joined}
					<Check style="width:18px;height:18px;" strokeWidth={2.5} />
					已报名 · 点击取消
				{:else}
					<Users style="width:18px;height:18px;" strokeWidth={2} />
					立即报名参加
				{/if}
			</button>
			</div>
		</div>
	{/snippet}

	{#snippet rail()}
	{#if ev}
	<RailSection title="活动发起人">
		<RailUserRow avatar={safeEv.host.avatar} name={safeEv.host.name} sub="活动发起人">
			{#snippet trailing()}
				<button type="button" class="hover-btn shrink-0 rounded-full px-2.5 py-1 text-white text-xs outline-none" style="background:{safeEv.color};">关注</button>
			{/snippet}
		</RailUserRow>
	</RailSection>

	<RailSection title="活动地点">
		<RailLocationMini name={safeEv.location} desc={`距你 ${safeEv.distance}`} />
	</RailSection>

	<RailSection title="近期报名" action="全部" onAction={() => (showParticipants = true)}>
		<div class="flex flex-col gap-3">
			{#each safeEv.participants.slice(0, 4) as p (p.id)}
				<RailUserRow avatar={p.avatar} name={p.name} sub="已报名" />
			{/each}
		</div>
	</RailSection>

	<RailSection title="同类活动" action="更多" onAction={() => void goto(ROUTES.nearbyMoments)}>
		<div class="flex flex-col gap-3">
			{#each similarEvents as s (s.id)}
				<RailUserRow avatar={s.avatar} name={s.name} sub={s.sub} onClick={() => void goto(ROUTES.event(s.id))} />
			{/each}
		</div>
	</RailSection>
	{/if}
	{/snippet}
</SplitPageLayout>
{:else}
	<div class="flex h-full w-full items-center justify-center">
		{#if loading}
			<EmptyState title="活动加载中…" />
		{:else}
			<EmptyState title="活动不存在" description="换个活动看看吧" />
		{/if}
	</div>
{/if}

<!-- 参与者名单 sheet -->
{#if ev}
<Dialog.Root open={showParticipants} onOpenChange={(next) => (next ? undefined : (showParticipants = false))}>
	<Dialog.Content side="bottom" class="overflow-hidden" style="max-width:520px;margin-left:auto;margin-right:auto;left:16px;right:16px;">
		<header class="flex items-center gap-3 px-5 py-4" style="border-bottom:1px solid var(--border-default);">
			<div class="flex-1">
				<p class="text-[16px] font-semibold text-text-primary">报名名单</p>
				<p class="mt-0.5 text-xs text-text-secondary">{safeEv.participants.length} 人已报名 · 还可加入 {safeEv.maxParticipants - safeEv.participants.length} 人</p>
			</div>
			<Dialog.Close class="surface-ghost flex items-center justify-center rounded-full outline-none" style="width:32px;height:32px;" aria-label="关闭报名名单">
				<X size={16} color="var(--text-secondary)" />
			</Dialog.Close>
		</header>
		<div class="no-scrollbar max-h-[52dvh] overflow-y-auto px-5 py-3">
			<div class="flex items-center gap-3 py-3" style="border-bottom:1px solid var(--border-default);">
				<img src={safeEv.host.avatar} alt={safeEv.host.name} class="h-10 w-10 rounded-full object-cover" style="border:2px solid {safeEv.color};" />
				<div class="flex-1">
					<p class="text-base font-semibold text-text-primary">{safeEv.host.name}</p>
					<p class="mt-0.5 text-xs text-text-secondary">活动发起人</p>
				</div>
				<span class="rounded-full px-2 py-1 text-xxs" style="color:{safeEv.color};background:{evTone(12)};">发起人</span>
			</div>
			<EmbeddedList>
			{#each safeEv.participants as participant (participant.id)}
				<div class="flex items-center gap-3 py-3">
					<img src={participant.avatar} alt={participant.name} class="h-10 w-10 rounded-full object-cover" />
					<div class="flex-1">
						<p class="text-base text-text-primary">{participant.name}</p>
						<p class="mt-0.5 text-xs text-text-secondary">已报名 · 等待一起见面</p>
					</div>
					<Check size={15} color={safeEv.color} />
				</div>
			{/each}
			</EmbeddedList>
		</div>
		<div class="px-5 pb-5 pt-3" style="border-top:1px solid var(--border-default);">
			<button type="button" onclick={() => (showParticipants = false)} class="w-full rounded-full py-3 text-base font-semibold outline-none" style="background:{evTone(14)};color:{safeEv.color};">知道了</button>
		</div>
	</Dialog.Content>
</Dialog.Root>
{/if}
