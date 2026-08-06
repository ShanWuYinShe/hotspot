<script lang="ts">
	import { ArrowLeft, MapPin, Clock, Users, Share2, Calendar, ChevronRight, Check, Star, X } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import SplitPageLayout from '@/components/layout/page/SplitPageLayout.svelte';
	import RailSection from '@/components/rail/RailSection.svelte';
	import RailUserRow from '@/components/rail/RailUserRow.svelte';
	import RailLocationMini from '@/components/rail/RailLocationMini.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import ListContainer from '@/components/list/ListContainer.svelte';
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
		id: '', title: '', tag: '', cover: '', description: '',
		location: '', date: '', time: '', host: { name: '', avatar: '' },
		participants: [], maxParticipants: 0, distance: '', highlights: []
	};
	const safeEv = $derived(ev ?? EMPTY_EVENT);

	// 活动色：数据层只返回语义 tag，视觉色由前端映射到 design token 档位
	// （主色 main / 淡底 bg / 淡边框 border / 光晕 glow / 反色文字 on）。
	// 参照 notifications TYPE_CONFIG 模式；缺省 tag 回退 accent 族。
	const EVENT_TONE: Record<string, { main: string; bg: string; border: string; glow: string; on: string }> = {
		社交: { main: 'var(--cst-status-warning)', bg: 'var(--cst-status-warning-bg)', border: 'var(--cst-status-warning-border)', glow: 'var(--cst-status-warning-glow)', on: 'var(--cst-text-on-status)' },
		音乐: { main: 'var(--cst-accent-default)', bg: 'var(--cst-accent-soft-bg)', border: 'var(--cst-accent-border-soft)', glow: 'var(--cst-accent-glow-soft)', on: 'var(--cst-text-on-accent)' },
		户外: { main: 'var(--cst-marker-group)', bg: 'var(--cst-marker-group-bg)', border: 'var(--cst-marker-group-border)', glow: 'var(--cst-marker-group-glow)', on: 'var(--cst-text-on-status)' },
		阅读: { main: 'var(--cst-status-online)', bg: 'var(--cst-status-online-bg)', border: 'var(--cst-status-online-border)', glow: 'var(--cst-status-online-glow)', on: 'var(--cst-text-on-status)' },
		市集: { main: 'var(--cst-status-danger)', bg: 'var(--cst-status-danger-bg)', border: 'var(--cst-status-danger-border)', glow: 'var(--cst-status-danger-glow)', on: 'var(--cst-text-on-status)' }
	};
	const DEFAULT_TONE = EVENT_TONE['音乐'];
	const tone = $derived(EVENT_TONE[safeEv.tag] ?? DEFAULT_TONE);

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
	const walkMinutes = $derived(ev && safeEv.distance ? Math.ceil(parseFloat(safeEv.distance) * 12) : 0);
</script>

{#if ev}
<SplitPageLayout contentClassName="pb-6">
	<!-- 全幅封面（负 margin 抵消 SplitPageLayout 的 gutter） -->
	<div class="relative shrink-0 overflow-hidden -mx-[var(--cst-page-gutter)]" style="height:clamp(200px, 32vw, 280px);">
				<img src={safeEv.cover} alt={safeEv.title} class="absolute inset-0 h-full w-full object-cover" />
				<div class="absolute inset-0" style="background:linear-gradient(to bottom, transparent 30%, {tone.bg} 100%);"></div>

				<!-- 返回 -->
				<div class="absolute top-4 left-4 flex gap-2">
					<ActionIconButton icon={ArrowLeft} label="返回" onClick={() => history.back()} variant="overlay" tone="inverse" />
				</div>
				<div class="absolute top-4 right-4">
					<ActionIconButton icon={Share2} label="分享活动" variant="overlay" tone="inverse" />
				</div>

				<!-- 标签 -->
				<div class="absolute bottom-4 left-4">
					<span class="rounded-full px-3 py-1 text-sm" style="color:{tone.on};background:{tone.main};box-shadow:0 3px 10px {tone.glow};">{safeEv.tag}</span>
				</div>
			</div>

	<!-- 内容（gutter 由 SplitPageLayout 提供） -->
	<div class="pt-5">
				<h1 class="text-text-primary text-2xl mb-3">{safeEv.title}</h1>

				<!-- meta 行 -->
				<div class="mb-6 flex flex-col gap-2.5">
					{#each [{ icon: Calendar, text: safeEv.date }, { icon: Clock, text: safeEv.time }, { icon: MapPin, text: safeEv.location }] as meta}
						<div class="flex items-center gap-2.5">
							<div class="flex shrink-0 items-center justify-center rounded-full" style="width:30px;height:30px;background:{tone.bg};">
								<meta.icon style="width:13px;height:13px;color:{tone.main};" strokeWidth={1.75} />
							</div>
							<span class="text-base text-text-secondary">{meta.text}</span>
						</div>
					{/each}
				</div>

				<!-- 发起人 -->
				<div class="mb-4 flex items-center gap-3 rounded-panel px-4 py-3.5 shadow-neu-lg border border-border-default"  style="background:var(--cst-surface-primary)">
					<div class="shrink-0 overflow-hidden rounded-full" style="width:40px;height:40px;border:2px solid {tone.main};">
						<img src={safeEv.host.avatar} alt={safeEv.host.name} class="h-full w-full object-cover" />
					</div>
					<div class="flex-1">
						<p class="text-base text-text-primary">{safeEv.host.name}</p>
						<div class="flex items-center gap-1">
							<Star class="size-2.5"  style="color:var(--cst-status-warning)" fill="var(--cst-status-warning)" strokeWidth={1.5} />
							<span class="text-xs text-text-secondary">活动发起人</span>
						</div>
					</div>
					<!-- TODO: 关注后端 users/follow 已实现（POST /api/v1/users/follow），前端接线待完成 -->
					<button type="button" class="hover-btn tap-93 rounded-full px-3 py-1.5 text-sm outline-none" style="color:{tone.on};background:{tone.main};box-shadow:0 3px 10px {tone.glow};">关注</button>
				</div>

				<!-- 描述 -->
				<p class="mb-5 text-base leading-[1.75] text-text-secondary">{safeEv.description}</p>

				<!-- 亮点 -->
				<div class="mb-5 grid gap-2.5" style="grid-template-columns:repeat(auto-fill, minmax(180px, 1fr));">
					{#each safeEv.highlights as h}
						<div class="flex items-center gap-2.5 rounded-control px-3.5 py-3 shadow-neu-md border border-border-default"  style="background:var(--cst-surface-primary)">
							<div class="flex shrink-0 items-center justify-center rounded-full" style="width:24px;height:24px;background:{tone.bg};">
								<Check style="width:11px;height:11px;color:{tone.main};" strokeWidth={2.5} />
							</div>
							<span class="text-sm text-text-primary">{h}</span>
						</div>
					{/each}
				</div>

				<!-- 参与者 -->
				<div class="mb-5">
					<div class="mb-3 flex items-center justify-between">
						<p class="text-base text-text-primary">参与者 · {safeEv.participants.length}/{safeEv.maxParticipants}</p>
						<button type="button" class="hover-btn tap-93 flex items-center gap-0.5 outline-none" style="color:var(--cst-accent-default);" onclick={() => (showParticipants = true)}>
							<span class="text-sm">查看全部</span>
							<ChevronRight class="size-3.25"  strokeWidth={1.5} />
						</button>
					</div>

					<!-- 进度条 -->
					<div class="mb-3 overflow-hidden rounded-full h-1.5"  style="background:var(--cst-border-default)">
						<div style="width:{fillRate}%;height:100%;background:{tone.main};border-radius:3px;transition:width 0.8s ease-out;"></div>
					</div>

					<!-- 头像栈 -->
					<div class="flex items-center gap-1">
						{#each safeEv.participants as p, i (p.id)}
							<div class="overflow-hidden rounded-full" style="width:36px;height:36px;border:2px solid var(--cst-surface-primary);margin-left:{i > 0 ? '-10px' : '0'};z-index:{safeEv.participants.length - i};">
								<img src={p.avatar} alt={p.name} class="h-full w-full object-cover" />
							</div>
						{/each}
						{#if safeEv.maxParticipants - safeEv.participants.length > 0}
							<div class="flex items-center justify-center rounded-full" style="width:36px;height:36px;border:2px solid var(--cst-surface-primary);margin-left:-10px;background:{tone.bg};">
								<span class="text-xs font-semibold" style="color:{tone.main};">+{safeEv.maxParticipants - safeEv.participants.length}</span>
							</div>
						{/if}
						<span class="ml-2 text-sm text-text-secondary">已报名</span>
					</div>
				</div>

				<!-- 距离信息 -->
				<div class="flex items-center gap-2 rounded-control px-4 py-3" style="background:{tone.bg};border:1px solid {tone.border};">
					<MapPin style="width:14px;height:14px;color:{tone.main};" strokeWidth={1.5} />
					<span class="text-base text-text-secondary">距你 {safeEv.distance} · 步行约 {walkMinutes} 分钟</span>
				</div>
	</div>

	{#snippet footer()}
		<!-- sticky 底部 CTA（按钮保持 640px 阅读宽度，页面布局全宽） -->
		<div class="shrink-0 px-[var(--cst-page-gutter)] pb-6 pt-3 border-t border-border-default"  style="background-color:var(--cst-surface-primary)">
			<div class="mx-auto w-full" style="max-width:640px;">
			<button
				type="button"
				onclick={() => (joined = !joined)}
				class="hover-btn tap-97 flex w-full items-center justify-center gap-2 rounded-feature py-4 text-base outline-none transition-all duration-300"
				style={joined
					? `background:var(--cst-surface-primary);border:2px solid ${tone.main};color:${tone.main};box-shadow:0 0 0 4px ${tone.bg};`
					: `background:${tone.main};color:var(--cst-text-on-status);box-shadow:0 8px 24px ${tone.glow};border:none;`}
			>
				{#if joined}
					<Check class="size-4.5"  strokeWidth={2.5} />
					已报名 · 点击取消
				{:else}
					<Users class="size-4.5"  strokeWidth={2} />
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
				<!-- TODO: 关注后端 users/follow 已实现（POST /api/v1/users/follow），前端接线待完成 -->
				<button type="button" class="hover-btn shrink-0 rounded-full px-2.5 py-1 text-xs outline-none" style="color:{tone.on};background:{tone.main};">关注</button>
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
		<header class="flex items-center gap-3 px-5 py-4 border-b border-border-default" >
			<div class="flex-1">
				<p class="text-[16px] font-semibold text-text-primary">报名名单</p>
				<p class="mt-0.5 text-xs text-text-secondary">{safeEv.participants.length} 人已报名 · 还可加入 {safeEv.maxParticipants - safeEv.participants.length} 人</p>
			</div>
			<Dialog.Close class="surface-ghost flex items-center justify-center rounded-full outline-none size-8"  aria-label="关闭报名名单">
				<X size={16} color="var(--cst-text-secondary)" />
			</Dialog.Close>
		</header>
		<div class="no-scrollbar max-h-[52dvh] overflow-y-auto px-5 py-3">
			<div class="flex items-center gap-3 py-3 border-b border-border-default" >
				<img src={safeEv.host.avatar} alt={safeEv.host.name} class="h-10 w-10 rounded-full object-cover" style="border:2px solid {tone.main};" />
				<div class="flex-1">
					<p class="text-base font-semibold text-text-primary">{safeEv.host.name}</p>
					<p class="mt-0.5 text-xs text-text-secondary">活动发起人</p>
				</div>
				<span class="rounded-full px-2 py-1 text-xxs" style="color:{tone.main};background:{tone.bg};">发起人</span>
			</div>
			<ListContainer>
			{#each safeEv.participants as participant (participant.id)}
				<div class="flex items-center gap-3 py-3">
					<img src={participant.avatar} alt={participant.name} class="h-10 w-10 rounded-full object-cover" />
					<div class="flex-1">
						<p class="text-base text-text-primary">{participant.name}</p>
						<p class="mt-0.5 text-xs text-text-secondary">已报名 · 等待一起见面</p>
					</div>
					<Check size={15} color={tone.main} />
				</div>
			{/each}
			</ListContainer>
		</div>
		<div class="px-5 pb-5 pt-3 border-t border-border-default" >
			<button type="button" onclick={() => (showParticipants = false)} class="w-full rounded-full py-3 text-base font-semibold outline-none" style="background:{tone.bg};color:{tone.main};">知道了</button>
		</div>
	</Dialog.Content>
</Dialog.Root>
{/if}
