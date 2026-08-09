<script lang="ts">
	import { X } from '@lucide/svelte';
	import AuthForm from '@/components/auth/AuthForm.svelte';
	import * as Dialog from '@/components/ui/dialog';

	// requireAuth() 拦截匿名用户写操作时弹出的底部登录 Sheet。
	// 内嵌共享 AuthForm；登录成功后 auth store 自动重放延迟操作。
	// 弹层行为（Escape / 外部点击 / 焦点圈 / 滚动锁）由 bits-ui Dialog 提供。

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

<Dialog.Root {open} onOpenChange={(next) => (next ? undefined : onClose())}>
	<Dialog.Content
		side="bottom"
		class="left-1/2 w-full max-w-115 -translate-x-1/2 px-5 pb-8 pt-3"
	>
		<div class="mb-3 flex justify-center" aria-hidden="true">
			<span class="h-1 w-9 rounded-full bg-border-strong"></span>
		</div>
		<div class="mb-4 flex items-start justify-between">
			<div>
				<Dialog.Title class="text-lg font-semibold tracking-tight text-text-primary" >
					登录后继续
				</Dialog.Title>
				{#if actionLabel}
					<p class="mt-0.5 text-sm text-text-secondary" >
						请先登录后再「{actionLabel}」
					</p>
				{/if}
			</div>
			<Dialog.Close
				aria-label="关闭"
				class="flex items-center justify-center rounded-full transition-opacity hover:opacity-70 size-8 text-text-secondary"
				 
			>
				<X size={18} />
			</Dialog.Close>
		</div>
		<AuthForm compact initialMode="login" onSuccess={onSuccess} />
	</Dialog.Content>
</Dialog.Root>
