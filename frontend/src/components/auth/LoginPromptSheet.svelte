<script lang="ts">
	import { X } from '@lucide/svelte';
	import AuthForm from '@/components/auth/AuthForm.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import Button from '@/components/ui/button.svelte';
	import { platform } from '@/stores/platform.svelte';

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

	// 移动端保持拇指可达的底部 sheet；桌面/平板用居中模态，与全站表单弹层一致。
	const isMobile = $derived(platform.platform === 'mobile');
</script>

<Dialog.Root {open} onOpenChange={(next) => (next ? undefined : onClose())}>
	<Dialog.Content
		side={isMobile ? 'bottom' : 'center'}
		class={isMobile ? 'w-full px-5 pb-8 pt-3 sm:max-w-115' : 'w-[28rem] max-w-[calc(100vw-2rem)] p-6'}
	>
		{#if isMobile}
			<div class="mb-3 flex justify-center" aria-hidden="true">
				<span class="h-1 w-9 rounded-full bg-border-strong"></span>
			</div>
		{/if}
		<div class="mb-4 flex items-start justify-between">
			<div>
				<!-- 弹层标题字重对齐 h1-h4 基线（font-medium），不跟随数字/激活态的 semibold -->
				<Dialog.Title class="text-lg font-medium tracking-title text-text-primary">
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
				class="hover-action focus-ring hit-slop flex size-8 items-center justify-center rounded-full text-text-secondary transition-colors duration-rapid ease-standard hover:text-text-primary"
			>
				<X size={18} />
			</Dialog.Close>
		</div>
		<AuthForm compact initialMode="login" onSuccess={onSuccess} />
		<!-- 「随便逛逛」全站唯一语义：跳过登录/注册，去可浏览内容继续逛（不创建访客账号）。
		     弹层里当前页本身就是公开可浏览的，所以正确动作是关闭弹窗 + 丢弃待执行操作、留在原地，
		     而不是像整页登录引导面板那样跳首页（那一层当前页被登录墙挡住，只能回可浏览页）。 -->
		<Button type="button" variant="ghost" size="md" class="mt-4 w-full" onclick={onClose}>
			先随便逛逛
		</Button>
	</Dialog.Content>
</Dialog.Root>
