<script lang="ts">
	// 分享码落地页（spec api_spec_public.md share/open_the_door）：/share/[code] 统一解析
	// 鸿蒙端与 Web 端分享出去的分享码链接，按 payload 类型跳转到对应详情页。
	// 独立于 (app) 分组：落地页不要求登录、不挂导航 chrome（目标页自身再按需引导登录）。
	//
	// R13 品牌化：从「居中一列散字」收成一张品牌内容卡（品牌字标 + 状态图标砖 +
	// 单一主 CTA + 语境化次 CTA），与登录/404 共用同一套语言；跳转逻辑与
	// 过期文案不变。
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { Compass, Link2Off, MapPin, RotateCw } from '@lucide/svelte';
	import { openShareDoor } from '@/api';
	import Button from '@/components/ui/button.svelte';
	import AmbientBackdrop from '@/components/layout/ambient/AmbientBackdrop.svelte';
	import BrandLockup from '@/components/layout/brand/BrandLockup.svelte';
	import { ROUTES } from '@/lib/routes';
	import type { ShareDetail } from '@/api/types';

	let { params }: { params: { code: string } } = $props();

	let phase = $state<'loading' | 'done' | 'error'>('loading');
	// 网络异常态的「重试」入口：递增触发重新解析（r83）
	let retryKey = $state(0);
	let errorMessage = $state('');
	let detail = $state<ShareDetail | null>(null);

	const targetRoute = (d: ShareDetail): string | null => {
		const p = d.payload;
		if ('shared_user_id' in p) return ROUTES.user(p.shared_user_id);
		if ('shared_group_id' in p) return ROUTES.group(p.shared_group_id);
		if ('shared_moment_id' in p) return ROUTES.moment(p.shared_moment_id);
		return null; // location 分享无对应详情页，落地页直接展示
	};

	$effect(() => {
		const code = (page.params.code ?? '').trim();
		void retryKey; // 网络异常重试触发重新解析（r83）
		if (!code) {
			phase = 'error';
			errorMessage = '分享链接无效';
			return;
		}
		let cancelled = false;
		phase = 'loading';
		void (async () => {
			try {
				const res = await openShareDoor({ share_code: code });
				if (cancelled) return;
				if (res.code !== 0 || !res.data) {
					phase = 'error';
					errorMessage = res.code === 10300 ? '分享不存在或已过期' : '分享链接无效或已失效';
					return;
				}
				detail = res.data;
				phase = 'done';
				const target = targetRoute(res.data);
				// replace 落地页不留历史：返回键直接回到分享来源而非失效的中转页
				if (target) void goto(target, { replaceState: true });
			} catch {
				if (!cancelled) {
					phase = 'error';
					errorMessage = '网络异常，请稍后重试';
				}
			}
		})();
		return () => {
			cancelled = true;
		};
	});
</script>

<svelte:head>
	<title>分享落地 · HotSpot</title>
</svelte:head>

<!-- 本路由独立于 (app) 分组：外层没有 main 地标，需自备（走查 s3：全站唯一无 <main> 的路由） -->
<main id="main-content" class="relative isolate flex min-h-dvh flex-col items-center justify-center bg-surface-page px-5 py-10">
	<!-- 氛围画布：与登录/错误页同一套主题世界层，落地页不再是一块孤立白底 -->
	<AmbientBackdrop />

	<section class="relative w-full max-w-100 rounded-lg bg-surface-primary p-6 text-center shadow-raised-lg sm:p-8">
		<div class="flex justify-center">
			<BrandLockup size="md" />
		</div>

		<div class="mt-7 flex flex-col items-center">
			{#if phase === 'loading'}
				<span class="flex size-14 items-center justify-center rounded-full bg-accent-soft">
					<span class="spinner size-5" role="status" aria-label="正在解析分享"></span>
				</span>
				<p class="mt-4 text-sm text-text-secondary">正在解析分享…</p>
			{:else if phase === 'error'}
				<span class="flex size-14 items-center justify-center rounded-full bg-accent-soft text-accent-default">
					<Link2Off class="size-6" strokeWidth={1.5} aria-hidden="true" />
				</span>
				<p class="mt-4 text-xxs font-medium uppercase tracking-eyebrow text-accent-default">分享链接</p>
				<h1 class="mt-1.5 text-2xl font-semibold tracking-title text-text-primary">{errorMessage}</h1>
				<!-- 副说明补充容错语境(r83:强化引导与容错) -->
				<p class="mt-2 max-w-xs text-sm leading-6 text-text-secondary">
					{errorMessage.includes('网络') ? '网络不稳定，可稍后重试。' : '链接可能已被删除，或地址有误。'}
				</p>
				<div class="mt-6 flex flex-wrap items-center justify-center gap-3">
					{#if errorMessage.includes('网络')}
						<Button variant="soft" size="md" class="hit-slop" onclick={() => (retryKey += 1)}>
							<RotateCw class="size-4" strokeWidth={1.75} aria-hidden="true" />
							重试
						</Button>
					{/if}
					<Button variant="primary" size="md" class="hit-slop" onclick={() => void goto(ROUTES.home, { replaceState: true })}>
						<Compass class="size-4" strokeWidth={1.75} aria-hidden="true" />
						前往 HotSpot
					</Button>
				</div>
			{:else if detail}
				{@const p = detail.payload}
				{#if 'shared_location' in p}
					<span class="flex size-14 items-center justify-center rounded-full bg-accent-soft text-accent-default">
						<MapPin class="size-6" strokeWidth={1.5} aria-hidden="true" />
					</span>
					<p class="mt-4 text-xxs font-medium uppercase tracking-eyebrow text-accent-default">位置锚点</p>
					<h1 class="mt-1.5 text-2xl font-semibold tracking-title text-text-primary">{p.shared_location.location_name}</h1>
					<!-- point 缺失（旧数据/异常 payload）时不能整页白屏，降级只展示名称 -->
					{#if p.shared_location.point}
						<p class="mt-2 text-xs tabular-nums text-text-secondary">
							{p.shared_location.point.latitude.toFixed(5)}, {p.shared_location.point.longitude.toFixed(5)}
						</p>
					{/if}
					<p class="mt-2 max-w-xs text-sm leading-6 text-text-secondary">这是一个位置锚点分享，可在 HotSpot 地图里查看周边。</p>
					<Button variant="primary" size="md" class="mt-6" onclick={() => void goto(ROUTES.map, { replaceState: true })}>
						<MapPin class="size-4" strokeWidth={1.75} aria-hidden="true" />
						打开地图
					</Button>
				{:else}
					<p class="text-sm text-text-secondary" role="status">正在跳转…</p>
				{/if}
			{/if}
		</div>
	</section>

	<p class="relative mt-5 text-xs text-text-secondary">HotSpot · 让附近有温度</p>
</main>