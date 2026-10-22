<script lang="ts">
	import { goto } from '$app/navigation';
	import { Leaf } from '@lucide/svelte';
	import AuthForm from '@/components/auth/AuthForm.svelte';
	import AmbientBackdrop from '@/components/layout/ambient/AmbientBackdrop.svelte';
	import { ROUTES } from '@/lib/routes';
	import type { AuthMode } from '@/lib/auth-validation';

	// 全页认证路由 /login：登录 / 注册 / 访客 / 忘记密码+重置。
	// 路由级进入成功后回首页；经 requireAuth 进入时 auth store 已持有
	// pending action 并在成功后自动重放。
	// 卡片结构对齐 Figma Make LoginPage：rounded-lg 400px 卡片 + 品牌区（Leaf 圆标 +
	// 品牌 eyebrow + 标语），品牌名统一为站名「HotSpot」（与 <title> 一致）。
	// 品牌区文案随表单模式切换语境（注册页曾误用登录口吻，见 UI 走查）。
	let mode = $state<AuthMode>('login');
	const HEADINGS: Record<AuthMode, { title: string; subtitle: string }> = {
		login: { title: '回来看看附近的故事。', subtitle: '使用手机号、邮箱或账号登录。' },
		register: { title: '创建账号，探索附近。', subtitle: '一个账号即可发现周边的人、群组与故事。' },
		guest: { title: '随便逛逛附近的故事。', subtitle: '以访客身份浏览，随时可以绑定账号。' },
		forgot: { title: '找回你的账号。', subtitle: '输入注册时的手机号或邮箱，我们会发送验证码。' },
		reset: { title: '设置新密码。', subtitle: '验证码已发送，请输入并设置新密码。' }
	};
</script>
<svelte:head>
	<title>登录 · HotSpot</title>
</svelte:head>


<div class="relative isolate flex min-h-dvh justify-center bg-surface-page px-5 py-10">
	<!-- 氛围画布：认证页整屏留白大，用主题世界层（星尘/等高线/墨山）补足空荡感 -->
	<AmbientBackdrop />
	<!-- my-auto 替代 items-center：卡片高于视口时容器被撑开、整页可滚动（flex 居中溢出会硬裁底部） -->
	<section class="my-auto w-full max-w-100 border border-border-default rounded-lg p-6 sm:p-8 bg-surface-primary shadow-raised-md">
		<div class="flex items-center justify-center rounded-full size-11 shadow-raised-sm bg-accent-default text-text-on-accent">
			<Leaf class="size-5"  strokeWidth={1.75} />
		</div>
		<p class="mt-6 text-xxs uppercase tracking-eyebrow font-medium text-accent-default">HotSpot</p>
		<h1 class="mt-2 text-2xl font-medium tracking-title text-text-primary">{HEADINGS[mode].title}</h1>
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
	</section>
</div>
