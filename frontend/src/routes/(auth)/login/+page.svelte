<script lang="ts">
	import { goto } from '$app/navigation';
	import { Leaf } from '@lucide/svelte';
	import AuthForm from '@/components/auth/AuthForm.svelte';
	import { ROUTES } from '@/lib/routes';

	// 全页认证路由 /login：登录 / 注册 / 访客 / 忘记密码+重置。
	// 路由级进入成功后回首页；经 requireAuth 进入时 auth store 已持有
	// pending action 并在成功后自动重放。
	// 卡片结构对齐 Figma Make LoginPage：直角 400px 卡片 + 品牌区（Leaf 圆标 +
	// 品牌 eyebrow + 标语），品牌名保留当前项目「热点」。
</script>

<div class="flex min-h-[100dvh] items-center justify-center bg-[var(--neu-bg)] px-5 py-10">
	<section class="w-full max-w-[400px] border border-[var(--neu-border)] p-6 sm:p-8" style="background:var(--neu-bg);">
		<div class="flex items-center justify-center rounded-full" style="width:44px;height:44px;background:var(--neu-accent-primary);color:white;box-shadow:0 4px 14px var(--neu-glow);">
			<Leaf style="width:20px;height:20px;" strokeWidth={1.8} />
		</div>
		<p class="mt-6 text-[11px] font-medium uppercase tracking-[0.14em] text-[var(--neu-accent-primary)]">热点</p>
		<h1 class="mt-2 text-2xl font-medium tracking-[-0.015em] text-[var(--neu-text)]">回来看看附近的故事。</h1>
		<p class="mt-2 text-sm leading-6 text-[var(--neu-text-muted)]">使用手机号、邮箱或账号登录。</p>

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
