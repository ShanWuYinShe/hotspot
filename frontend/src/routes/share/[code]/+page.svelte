<script lang="ts">
	// 分享码落地页（spec api_spec_public.md share/open_the_door）：/share/[code] 统一解析
	// 鸿蒙端与 Web 端分享出去的分享码链接，按 payload 类型跳转到对应详情页。
	// 独立于 (app) 分组：落地页不要求登录、不挂导航 chrome（目标页自身再按需引导登录）。
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { openShareDoor } from '@/api';
	import { ROUTES } from '@/lib/routes';
	import type { ShareDetail } from '@/api/types';

	let { params }: { params: { code: string } } = $props();

	let phase = $state<'loading' | 'done' | 'error'>('loading');
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
					errorMessage = res.code === 10300 ? '分享不存在或已过期' : res.message || '分享解析失败';
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

<div class="flex min-h-dvh flex-col items-center justify-center gap-4 bg-surface-page px-6 text-center">
	<p class="eyebrow">HotSpot</p>
	{#if phase === 'loading'}
		<p class="text-md text-text-secondary" role="status">正在解析分享…</p>
	{:else if phase === 'error'}
		<p class="text-md text-text-primary">{errorMessage}</p>
		<button
			type="button"
			onclick={() => goto(ROUTES.home, { replaceState: true })}
			class="rounded-full bg-accent-soft px-5 py-2 text-sm font-medium text-accent-default outline-none transition-colors hover:bg-interactive-active focus-visible:ring-2 focus-visible:ring-accent-default"
		>
			前往 HotSpot
		</button>
	{:else if detail}
		{@const p = detail.payload}
		{#if 'shared_location' in p}
			<div class="flex flex-col items-center gap-2">
				<p class="text-lg font-semibold tracking-title text-text-primary">{p.shared_location.location_name}</p>
				<p class="text-xs tabular-nums text-text-secondary">
					{p.shared_location.point.latitude.toFixed(5)}, {p.shared_location.point.longitude.toFixed(5)}
				</p>
				<p class="text-sm text-text-secondary">这是一个位置锚点分享，请在 HotSpot App 内查看</p>
			</div>
			<button
				type="button"
				onclick={() => goto(ROUTES.map, { replaceState: true })}
				class="rounded-full bg-accent-soft px-5 py-2 text-sm font-medium text-accent-default outline-none transition-colors hover:bg-interactive-active focus-visible:ring-2 focus-visible:ring-accent-default"
			>
				打开地图
			</button>
		{:else}
			<p class="text-md text-text-secondary" role="status">正在跳转…</p>
		{/if}
	{/if}
</div>
