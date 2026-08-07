<script lang="ts">
	import { goto } from '$app/navigation';
	import { Leaf } from '@lucide/svelte';
	import AuthForm from '@/components/auth/AuthForm.svelte';
	import { ROUTES } from '@/lib/routes';

	// 全页认证路由 /login：登录 / 注册 / 访客 / 忘记密码+重置。
	// 路由级进入成功后回首页；经 requireAuth 进入时 auth store 已持有
	// pending action 并在成功后自动重放。
	// 卡片结构对齐 Figma Make LoginPage：rounded-panel 400px 卡片 + 品牌区（Leaf 圆标 +
	// 品牌 eyebrow + 标语），品牌名保留当前项目「热点」。
</script>

<div class="flex min-h-dvh items-center justify-center bg-surface-primary px-5 py-10">
	<section class="w-full max-w-100 border border-border-default rounded-panel p-6 sm:p-8" style="background:var(--cst-surface-primary);">
		<div class="flex items-center justify-center rounded-full size-11 shadow-[0_4px_14px_var(--cst-glow-accent)]"   style="background:var(--cst-accent-default);color:var(--cst-text-on-accent)">
			<Leaf class="size-5"  strokeWidth={1.8} />
		</div>
		<p class="mt-6 text-xs font-medium uppercase tracking-[0.14em] text-accent-default">热点</p>
		<h1 class="mt-2 text-2xl font-medium tracking-[-0.015em] text-text-primary">回来看看附近的故事。</h1>
		<p class="mt-2 text-sm leading-6 text-text-secondary">使用手机号、邮箱或账号登录。</p>

		<div class="mt-7">
			<AuthForm
				onSuccess={() => {
					// Route-level entry: go home. Prompt-driven entries replay their
					// pending action via auth store; either way, leave /login.
					void goto(ROUTES.home, { replaceState: true });
				}}
			/>
		</div>
	</section>
</div>
