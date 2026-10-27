<script lang="ts">
	import { goto } from '$app/navigation';
	import { Compass, MessageCircle, Sparkles } from '@lucide/svelte';
	import AuthForm from '@/components/auth/AuthForm.svelte';
	import AmbientBackdrop from '@/components/layout/ambient/AmbientBackdrop.svelte';
	import BrandLockup from '@/components/layout/brand/BrandLockup.svelte';
	import { untrack } from 'svelte';
	import { page } from '$app/state';
	import { ROUTES } from '@/lib/routes';
	import type { AuthMode } from '@/lib/auth-validation';

	// 全页认证路由 /login：登录 / 注册 / 访客 / 忘记密码+重置。
	// 路由级进入成功后回首页；经 requireAuth 进入时 auth store 已持有
	// pending action 并在成功后自动重放。
	//
	// R13 品牌页：桌面（≥lg）双栏——左品牌叙事（brand 字标 + 一句主张 + 3 条价值点 +
	// 底部细字）直接坐在氛围画布上，右表单卡限宽 400 垂直居中；平板/移动单列，
	// 页头保留品牌字标，标题层级收进表单卡。视觉母题只用既有 AmbientBackdrop 与
	// accent 语义 token，不额外铺第二套背景。
	// 卡片结构由页面承载，表单控件仍全部来自 AuthForm（不改提交/校验/验证码逻辑）。
	//
	// 路由复用：/login 与 /register 共用本组件（AuthPage），注册态也支持
	// /login?mode=register 深链（外部分享/文档里常见的注册入口不再落 404，走查 P1）。
	let { initialMode = 'login' }: { initialMode?: AuthMode } = $props();
	const MODES: AuthMode[] = ['login', 'register', 'guest', 'forgot', 'reset'];
	const queryMode = page.url.searchParams.get('mode');
	// 初始模式只在挂载时解析一次（后续切换由 AuthForm 内部控制）：untrack 显式声明
	// 「读一次 prop」的意图，避免 svelte-check 的 state_referenced_locally 告警。
	let mode = $state<AuthMode>(
		untrack(() =>
			queryMode && (MODES as string[]).includes(queryMode) ? (queryMode as AuthMode) : initialMode
		)
	);
	const HEADINGS: Record<AuthMode, { eyebrow: string; title: string; subtitle: string }> = {
		login: { eyebrow: '欢迎回来', title: '回来看看附近的故事。', subtitle: '使用手机号、邮箱或账号登录。' },
		register: { eyebrow: '加入 HotSpot', title: '创建账号，探索附近。', subtitle: '一个账号即可发现周边的人、群组与故事。' },
		guest: { eyebrow: '访客模式', title: '随便逛逛附近的故事。', subtitle: '以访客身份浏览，随时可以绑定账号。' },
		forgot: { eyebrow: '找回账号', title: '找回你的账号。', subtitle: '输入注册时的手机号或邮箱，我们会发送验证码。' },
		reset: { eyebrow: '重置密码', title: '设置新密码。', subtitle: '验证码已发送，请输入并设置新密码。' }
	};
	const VALUE_POINTS = [
		{ icon: Compass, title: '发现附近', text: '身边的人、群组与瞬间，都发生在几百米内。' },
		{ icon: MessageCircle, title: '遇见同频', text: '一个招呼、一次同行，让附近不再陌生。' },
		{ icon: Sparkles, title: '记录此刻', text: '把街角的温度分享给刚好路过的人。' }
	];
	const pageTitle = $derived(mode === 'register' ? '注册' : mode === 'guest' ? '访客浏览' : '登录');
</script>

<svelte:head>
	<title>{pageTitle} · HotSpot</title>
</svelte:head>

<main class="relative isolate min-h-dvh bg-surface-page">
	<!-- 氛围画布：认证页大留白，用主题世界层（星尘/等高线/墨山）补足空荡感 -->
	<AmbientBackdrop />

	<div class="mx-auto grid min-h-dvh w-full lg:grid-cols-[1.08fr_1fr]">
		<!-- 左：品牌叙事（桌面 ≥lg）。纯文本排版 + 既有氛围画布，不再叠加装饰层。 -->
		<section class="relative hidden flex-col justify-between px-14 py-12 lg:flex xl:px-20">
			<BrandLockup size="md" />

			<div class="max-w-lg">
				<p class="text-xxs font-medium uppercase tracking-eyebrow text-accent-default">城市社交画布</p>
				<p class="mt-4 text-4xl font-semibold leading-tight tracking-title text-text-primary">
					在附近，<br />遇见同频的人。
				</p>
				<p class="mt-4 max-w-md text-md leading-copy text-text-secondary">
					HotSpot 把城市切成一个个可被看见的附近：谁在附近、正在发生什么、有没有人和你同频。
				</p>
				<ul class="mt-10 flex max-w-md flex-col gap-5">
					{#each VALUE_POINTS as point (point.title)}
						<li class="flex items-start gap-3.5">
							<span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-accent-soft text-accent-default">
								<point.icon class="size-4.5" strokeWidth={1.75} aria-hidden="true" />
							</span>
							<span class="min-w-0">
								<span class="block text-md font-medium text-text-primary">{point.title}</span>
								<span class="mt-0.5 block text-xs leading-relaxed text-text-secondary">{point.text}</span>
							</span>
						</li>
					{/each}
				</ul>
			</div>

			<p class="text-xs text-text-secondary">© {new Date().getFullYear()} HotSpot · 让附近有温度</p>
		</section>

		<!-- 右：表单（桌面）/ 整页（平板·移动） -->
		<section class="flex items-center justify-center px-5 py-10 sm:px-8 lg:py-12">
			<div class="w-full max-w-100">
				<!-- 平板/移动页头：品牌标识（桌面由左栏承担，避免重复） -->
				<div class="mb-7 flex items-center justify-between lg:hidden">
					<BrandLockup size="sm" />
					<span class="text-xs text-text-secondary">城市社交画布</span>
				</div>

				<!-- 表单卡用 surface-primary（不是 island）：forest/inkwash 的 island 偏灰，
				     会把这个「第一眼入口」读成禁用面板；白色主表面 + lg 投影更符合入口卡语义。 -->
				<div class="rounded-lg bg-surface-primary p-6 shadow-raised-lg sm:p-8">
					<p class="text-xxs font-medium uppercase tracking-eyebrow text-accent-default">{HEADINGS[mode].eyebrow}</p>
					<h1 class="mt-2 text-2xl font-semibold tracking-title text-text-primary">{HEADINGS[mode].title}</h1>
					<p class="mt-2 text-sm leading-6 text-text-secondary">{HEADINGS[mode].subtitle}</p>

					<div class="mt-7">
						<AuthForm
							initialMode={mode}
							onModeChange={(m) => (mode = m)}
							onSuccess={() => {
								// Route-level entry: go home. Prompt-driven entries replay their
								// pending action via auth store; either way, leave /login.
								void goto(ROUTES.home, { replaceState: true });
							}}
						/>
					</div>
				</div>

				<p class="mt-5 text-center text-xs text-text-secondary lg:hidden">让附近有温度 · 随时可以退出</p>
			</div>
		</section>
	</div>
</main>