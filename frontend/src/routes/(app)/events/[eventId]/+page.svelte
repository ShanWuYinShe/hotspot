<script lang="ts">
	import { ArrowLeft, MapPin, Clock, Users, Share2, Calendar, ChevronRight, Check, Star, X } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { platformStore } from '@/stores/platform.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import DesktopDetailSplit from '@/components/layout/desktop/DesktopDetailSplit.svelte';
	import RailSection from '@/components/layout/desktop/RailSection.svelte';
	import RailUserRow from '@/components/layout/desktop/RailUserRow.svelte';
	import RailLocationMini from '@/components/layout/desktop/RailLocationMini.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import { ROUTES } from '@/lib/routes';

	// 活动详情（对齐 Figma Make EventDetailPage）。
	// 全幅封面 + 报名 CTA + 发起人卡 + 参与者 sheet + 4 段桌面 rail。
	// 后端无活动端点，先用 mock（与 Figma EVENTS 对齐），后续接入。

	interface Participant {
		id: number;
		name: string;
		avatar: string;
	}
	interface EventData {
		id: number;
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

	const SIMILAR_EVENTS = [
		{ id: 2, name: '户部山周末弹唱', sub: '周日 14:00 · 0.7km', avatar: 'https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=120' },
		{ id: 3, name: '回龙窝夜读会', sub: '周四 19:30 · 1.1km', avatar: 'https://images.unsplash.com/photo-1774125384717-5c5a05ce8e9c?w=120' },
		{ id: 4, name: '老街胶片散步', sub: '周六 09:00 · 1.5km', avatar: 'https://images.unsplash.com/photo-1759734282634-b4958ea9a6e0?w=120' }
	];

	const EVENTS: Record<number, EventData> = {
		1: {
			id: 1,
			title: '湖边晨咖小聚',
			tag: '社交',
			color: 'var(--semantic-warning)',
			bg: 'color-mix(in srgb, var(--semantic-warning) 12%, transparent)',
			cover: 'https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=1080',
			description:
				'每周六早上九点，我们聚集在老街手冲馆，分享一周的所见所闻。无论是设计师、程序员还是艺术家，都欢迎加入！带上你的好奇心就够了。',
			location: '老街手冲馆·徐州市鼓楼区',
			date: '2026年4月18日 周六',
			time: '09:00 - 11:00',
			host: { name: '南枝', avatar: 'https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80' },
			participants: [
				{ id: 1, name: '鹿鸣', avatar: 'https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80' },
				{ id: 2, name: '阿禾', avatar: 'https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80' },
				{ id: 3, name: '野原', avatar: 'https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80' },
				{ id: 4, name: 'Mika', avatar: 'https://images.unsplash.com/photo-1758613172074-7275c3455034?w=80' }
			],
			maxParticipants: 20,
			distance: '0.3km',
			highlights: ['免费咖啡 ☕', '结识新朋友', '分享创意', '每周固定']
		},
		2: {
			id: 2,
			title: '户部山周末弹唱',
			tag: '音乐',
			color: 'var(--neu-accent-primary)',
			bg: 'var(--neu-accent-soft)',
			cover: 'https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=1080',
			description:
				'在城市的中心，用音乐连接每一颗心。本次户部山周末弹唱将邀请本地独立乐队和歌手进行现场表演，免费入场，欢迎所有人来感受音乐的力量。',
			location: '彭城广场·徐州市云龙区',
			date: '2026年4月19日 周日',
			time: '14:00 - 20:00',
			host: { name: '本地向导', avatar: 'https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80' },
			participants: [
				{ id: 1, name: '林一', avatar: 'https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80' },
				{ id: 2, name: '小满', avatar: 'https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80' },
				{ id: 3, name: '阿川', avatar: 'https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80' }
			],
			maxParticipants: 200,
			distance: '0.7km',
			highlights: ['免费入场 🎸', '现场表演', '多个乐队', '本地艺术家']
		}
	};

	const DEFAULT_EVENT = EVENTS[1];

	const eventIdParam = $derived(page.params.eventId);
	const ev = $derived(EVENTS[Number(eventIdParam)] ?? DEFAULT_EVENT);
	const isDesktop = $derived(platformStore.platform === 'desktop');

	let joined = $state(false);
	let showParticipants = $state(false);

	const fillRate = $derived(Math.round((ev.participants.length / ev.maxParticipants) * 100));
	const evTone = (amount: number) => `color-mix(in srgb, ${ev.color} ${amount}%, transparent)`;
	const walkMinutes = $derived(Math.ceil(parseInt(ev.distance) * 12));
</script>

<!-- 主内容（桌面用 DesktopDetailSplit，移动/平板用单列） -->
{#snippet mainContent()}
	<div class="flex flex-col h-full" style="background-color:var(--neu-bg);">
		<!-- 可滚动区 -->
		<div class="no-scrollbar flex-1 overflow-y-auto">
			<!-- 全幅封面 -->
			<div class="relative flex-shrink-0 overflow-hidden" style="height:clamp(200px, 32vw, 280px);">
				<img src={ev.cover} alt={ev.title} class="absolute inset-0 h-full w-full object-cover" />
				<div class="absolute inset-0" style="background:linear-gradient(to bottom, transparent 30%, {ev.bg} 100%);"></div>

				<!-- 返回 -->
				<div class="absolute top-4 left-4 flex gap-2">
					<ActionIconButton icon={ArrowLeft} label="返回" onClick={() => history.back()} variant="overlay" tone="inverse" />
				</div>
				<div class="absolute top-4 right-4">
					<ActionIconButton icon={Share2} label="分享活动" variant="overlay" tone="inverse" />
				</div>

				<!-- 标签 -->
				<div class="absolute bottom-4 left-4">
					<span class="rounded-full px-3 py-1 text-white text-[12px]" style="background:{ev.color};box-shadow:0 3px 10px {evTone(50)};">{ev.tag}</span>
				</div>
			</div>

			<!-- 内容 -->
			<div class="tablet-content-fill px-5 pb-6 pt-4" style="max-width:640px;margin-left:auto;margin-right:auto;width:100%;">
				<h1 class="text-[var(--neu-text)] text-[24px] mb-2">{ev.title}</h1>

				<!-- meta 行 -->
				<div class="mb-5 flex flex-col gap-2">
					{#each [{ icon: Calendar, text: ev.date }, { icon: Clock, text: ev.time }, { icon: MapPin, text: ev.location }] as meta}
						<div class="flex items-center gap-2.5">
							<div class="flex flex-shrink-0 items-center justify-center rounded-full" style="width:30px;height:30px;background:color-mix(in srgb, {ev.color} 14%, transparent);">
								<meta.icon style="width:13px;height:13px;color:{ev.color};" strokeWidth={1.75} />
							</div>
							<span class="text-[13px] text-[var(--neu-text-muted)]">{meta.text}</span>
						</div>
					{/each}
				</div>

				<!-- 发起人 -->
				<div class="mb-4 flex items-center gap-3 rounded-panel px-4 py-3.5" style="background:var(--neu-bg);box-shadow:5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light);border:1px solid var(--neu-border);">
					<div class="flex-shrink-0 overflow-hidden rounded-full" style="width:40px;height:40px;border:2px solid {ev.color};">
						<img src={ev.host.avatar} alt={ev.host.name} class="h-full w-full object-cover" />
					</div>
					<div class="flex-1">
						<p class="text-[13px] text-[var(--neu-text)]">{ev.host.name}</p>
						<div class="flex items-center gap-1">
							<Star style="width:10px;height:10px;color:var(--semantic-warning);" fill="var(--semantic-warning)" strokeWidth={1.5} />
							<span class="text-[11px] text-[var(--neu-text-muted)]">活动发起人</span>
						</div>
					</div>
					<button type="button" class="neu-hover-btn rounded-full px-3 py-1.5 text-white text-[12px] outline-none" style="background:{ev.color};box-shadow:0 3px 10px {evTone(40)};">关注</button>
				</div>

				<!-- 描述 -->
				<p class="mb-5 text-[14px] leading-[1.75] text-[var(--neu-text-muted)]">{ev.description}</p>

				<!-- 亮点 -->
				<div class="mb-5 grid gap-2.5" style="grid-template-columns:repeat(auto-fill, minmax(180px, 1fr));">
					{#each ev.highlights as h}
						<div class="flex items-center gap-2.5 rounded-control px-3.5 py-3" style="background:var(--neu-bg);box-shadow:4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light);border:1px solid var(--neu-border);">
							<div class="flex flex-shrink-0 items-center justify-center rounded-full" style="width:24px;height:24px;background:color-mix(in srgb, {ev.color} 14%, transparent);">
								<Check style="width:11px;height:11px;color:{ev.color};" strokeWidth={2.5} />
							</div>
							<span class="text-[12px] text-[var(--neu-text)]">{h}</span>
						</div>
					{/each}
				</div>

				<!-- 参与者 -->
				<div class="mb-5">
					<div class="mb-3 flex items-center justify-between">
						<p class="text-[14px] text-[var(--neu-text)]">参与者 · {ev.participants.length}/{ev.maxParticipants}</p>
						<button type="button" class="neu-hover-btn flex items-center gap-0.5 outline-none" style="color:var(--neu-accent-primary);" onclick={() => (showParticipants = true)}>
							<span class="text-[12px]">查看全部</span>
							<ChevronRight style="width:13px;height:13px;" strokeWidth={1.5} />
						</button>
					</div>

					<!-- 进度条 -->
					<div class="mb-3 overflow-hidden rounded-full" style="height:6px;background:var(--neu-border);">
						<div style="width:{fillRate}%;height:100%;background:{ev.color};border-radius:3px;transition:width 0.8s ease-out;"></div>
					</div>

					<!-- 头像栈 -->
					<div class="flex items-center gap-1">
						{#each ev.participants as p, i (p.id)}
							<div class="overflow-hidden rounded-full" style="width:36px;height:36px;border:2px solid var(--neu-bg);margin-left:{i > 0 ? '-10px' : '0'};z-index:{ev.participants.length - i};">
								<img src={p.avatar} alt={p.name} class="h-full w-full object-cover" />
							</div>
						{/each}
						{#if ev.maxParticipants - ev.participants.length > 0}
							<div class="flex items-center justify-center rounded-full" style="width:36px;height:36px;border:2px solid var(--neu-bg);margin-left:-10px;background:{evTone(18)};">
								<span class="text-[11px] font-semibold" style="color:{ev.color};">+{ev.maxParticipants - ev.participants.length}</span>
							</div>
						{/if}
						<span class="ml-2 text-[12px] text-[var(--neu-text-muted)]">已报名</span>
					</div>
				</div>

				<!-- 距离信息 -->
				<div class="flex items-center gap-2 rounded-control px-4 py-3" style="background:{evTone(10)};border:1px solid {evTone(25)};">
					<MapPin style="width:14px;height:14px;color:{ev.color};" strokeWidth={1.5} />
					<span class="text-[13px] text-[var(--neu-text-muted)]">距你 {ev.distance} · 步行约 {walkMinutes} 分钟</span>
				</div>
			</div>
		</div>

		<!-- sticky 底部 CTA -->
		<div class="tablet-content-fill flex-shrink-0 px-5 pb-6 pt-3" style="background-color:var(--neu-bg);border-top:1px solid var(--neu-border);max-width:640px;margin-left:auto;margin-right:auto;width:100%;">
			<button
				type="button"
				onclick={() => (joined = !joined)}
				class="neu-hover-btn flex w-full items-center justify-center gap-2 rounded-feature py-4 text-[15px] outline-none transition-all duration-300"
				style={joined
					? `background:var(--neu-bg);border:2px solid ${ev.color};color:${ev.color};box-shadow:0 0 0 4px ${evTone(18)};`
					: `background:${ev.color};color:white;box-shadow:0 8px 24px ${evTone(45)};border:none;`}
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

<!-- 桌面 rail -->
{#snippet railContent()}
	<RailSection title="活动发起人">
		<RailUserRow avatar={ev.host.avatar} name={ev.host.name} sub="活动发起人 · 关注 3.2k">
			{#snippet trailing()}
				<button type="button" class="neu-hover-btn flex-shrink-0 rounded-full px-2.5 py-1 text-white text-[11px] outline-none" style="background:{ev.color};">关注</button>
			{/snippet}
		</RailUserRow>
	</RailSection>

	<RailSection title="活动地点">
		<RailLocationMini name={ev.location} desc={`距你 ${ev.distance}`} />
	</RailSection>

	<RailSection title="近期报名" action="全部" onAction={() => (showParticipants = true)}>
		<div class="flex flex-col gap-3">
			{#each ev.participants.slice(0, 4) as p (p.id)}
				<RailUserRow avatar={p.avatar} name={p.name} sub="已报名" />
			{/each}
		</div>
	</RailSection>

	<RailSection title="同类活动" action="更多" onAction={() => void goto(ROUTES.nearbyMoments)}>
		<div class="flex flex-col gap-3">
			{#each SIMILAR_EVENTS as s (s.id)}
				<RailUserRow avatar={s.avatar} name={s.name} sub={s.sub} onClick={() => void goto(ROUTES.event(s.id))} />
			{/each}
		</div>
	</RailSection>
{/snippet}

{#if isDesktop}
	<DesktopDetailSplit>
		{#snippet main()}{@render mainContent()}{/snippet}
		{#snippet rail()}{@render railContent()}{/snippet}
	</DesktopDetailSplit>
{:else}
	{@render mainContent()}
{/if}

<!-- 参与者名单 sheet -->
<Dialog.Root open={showParticipants} onOpenChange={(next) => (next ? undefined : (showParticipants = false))}>
	<Dialog.Content side="bottom" class="overflow-hidden" style="max-width:520px;margin-left:auto;margin-right:auto;left:16px;right:16px;">
		<header class="flex items-center gap-3 px-5 py-4" style="border-bottom:1px solid var(--neu-border);">
			<div class="flex-1">
				<p class="text-[16px] font-semibold text-[var(--neu-text)]">报名名单</p>
				<p class="mt-0.5 text-[11px] text-[var(--neu-text-muted)]">{ev.participants.length} 人已报名 · 还可加入 {ev.maxParticipants - ev.participants.length} 人</p>
			</div>
			<Dialog.Close class="neu-surface-ghost flex items-center justify-center rounded-full outline-none" style="width:32px;height:32px;" aria-label="关闭报名名单">
				<X size={16} color="var(--neu-text-muted)" />
			</Dialog.Close>
		</header>
		<div class="no-scrollbar max-h-[52dvh] overflow-y-auto px-5 py-3">
			<div class="flex items-center gap-3 py-3" style="border-bottom:1px solid var(--neu-border);">
				<img src={ev.host.avatar} alt={ev.host.name} class="h-10 w-10 rounded-full object-cover" style="border:2px solid {ev.color};" />
				<div class="flex-1">
					<p class="text-[13px] font-semibold text-[var(--neu-text)]">{ev.host.name}</p>
					<p class="mt-0.5 text-[11px] text-[var(--neu-text-muted)]">活动发起人</p>
				</div>
				<span class="rounded-full px-2 py-1 text-[10px]" style="color:{ev.color};background:{evTone(12)};">发起人</span>
			</div>
			{#each ev.participants as participant, index (participant.id)}
				<div class="flex items-center gap-3 py-3" style="border-bottom:{index < ev.participants.length - 1 ? '1px solid var(--neu-border)' : 'none'};">
					<img src={participant.avatar} alt={participant.name} class="h-10 w-10 rounded-full object-cover" />
					<div class="flex-1">
						<p class="text-[13px] text-[var(--neu-text)]">{participant.name}</p>
						<p class="mt-0.5 text-[11px] text-[var(--neu-text-muted)]">已报名 · 等待一起见面</p>
					</div>
					<Check size={15} color={ev.color} />
				</div>
			{/each}
		</div>
		<div class="px-5 pb-5 pt-3" style="border-top:1px solid var(--neu-border);">
			<button type="button" onclick={() => (showParticipants = false)} class="w-full rounded-full py-3 text-[13px] font-semibold outline-none" style="background:{evTone(14)};color:{ev.color};">知道了</button>
		</div>
	</Dialog.Content>
</Dialog.Root>
