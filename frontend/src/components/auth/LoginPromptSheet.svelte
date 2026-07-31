<script lang="ts">
	import { X } from '@lucide/svelte';
	import AuthForm from '@/components/auth/AuthForm.svelte';

	// requireAuth() 拦截匿名用户写操作时弹出的底部登录 Sheet。
	// 内嵌共享 AuthForm；登录成功后 auth store 自动重放延迟操作。

	let {
		open,
		actionLabel,
		onClose,
		onSuccess
	}: {
		open: boolean;
		actionLabel: string | null;
		onClose: () => void;
		onSuccess: () => void;
	} = $props();
</script>

{#if open}
	<div
		class="fixed inset-0 z-[var(--z-modal)]"
		style="background:rgba(0,0,0,.4);backdrop-filter:blur(2px);"
		role="presentation"
		onclick={onClose}
	></div>
	<div
		class="fixed bottom-0 left-1/2 z-[var(--z-modal)] w-full max-w-[460px] -translate-x-1/2 rounded-t-[28px] px-5 pb-8 pt-3"
		style="background:var(--neu-bg);box-shadow:0 -8px 30px rgba(0,0,0,.12);"
		role="dialog"
		aria-modal="true"
	>
		<div class="mb-3 flex justify-center" aria-hidden="true">
			<span class="h-1 w-9 rounded-full" style="background:var(--neu-border-pressed);"></span>
		</div>
		<div class="mb-4 flex items-start justify-between">
			<div>
				<h2 class="text-[17px] font-semibold tracking-tight" style="color:var(--neu-text);">
					登录后继续
				</h2>
				{#if actionLabel}
					<p class="mt-0.5 text-[12px]" style="color:var(--neu-text-muted);">
						请先登录后再「{actionLabel}」
					</p>
				{/if}
			</div>
			<button
				type="button"
				onclick={onClose}
				aria-label="关闭"
				class="flex items-center justify-center rounded-full transition-opacity hover:opacity-70"
				style="width:32px;height:32px;color:var(--neu-text-muted);"
			>
				<X size={18} />
			</button>
		</div>
		<AuthForm compact initialMode="login" onSuccess={onSuccess} />
	</div>
{/if}
