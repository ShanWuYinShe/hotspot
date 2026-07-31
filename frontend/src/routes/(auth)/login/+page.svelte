<script lang="ts">
	import { goto } from '$app/navigation';
	import AuthForm from '@/components/auth/AuthForm.svelte';
	import { ROUTES } from '@/lib/routes';

	// 全页认证路由 /login：登录 / 注册 / 访客 / 忘记密码+重置。
	// 路由级进入成功后回首页；经 requireAuth 进入时 auth store 已持有
	// pending action 并在成功后自动重放。
</script>

<div class="flex min-h-screen flex-col items-center justify-center px-5 py-10 bg-[var(--neu-bg)]">
	<div class="w-full max-w-[380px]">
		<div class="mb-7 text-center">
			<h1 class="text-[26px] font-semibold tracking-tight text-[var(--neu-text)]">
				热点
			</h1>
			<p class="mt-1.5 text-[13px] text-[var(--neu-text-muted)]">
				发现身边的人与故事
			</p>
		</div>
		<div class="rounded-[22px] px-5 py-6 bg-[var(--neu-surface,var(--neu-bg))] border border-[var(--neu-border)]">
			<AuthForm
				onSuccess={() => {
					// Route-level entry: go home. Prompt-driven entries replay their
					// pending action via auth store; either way, leave /login.
					void goto(ROUTES.home, { replaceState: true });
				}}
			/>
		</div>
	</div>
</div>
