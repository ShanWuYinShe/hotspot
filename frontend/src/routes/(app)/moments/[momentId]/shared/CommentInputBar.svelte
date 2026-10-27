<script lang="ts">
	import { Send } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import { auth } from '@/stores/auth.svelte';

	// CommentInputBar —— 详情页底部固定评论输入栏（Enter/按钮发送均走 onSend）。
	// R13：与消息区的关系收敛为「一条发丝线 + 画布色」；输入框 40 高、发送钮 36 圆钮，
	// 空内容时不带投影（安静态），有内容才浮起 accent 实底。
	let {
		input = $bindable(''),
		inputEl = $bindable(null),
		onSend
	}: {
		input?: string;
		inputEl?: HTMLInputElement | null;
		onSend: () => void;
	} = $props();

	const canSend = $derived(input.trim().length > 0);
</script>

<div
	class="shrink-0 flex items-center gap-2.5 bg-surface-primary border-t border-border-subtle/50 px-[var(--cst-page-gutter)] pt-3"
	style="padding-bottom:max(0.75rem, env(safe-area-inset-bottom));"
>
	<Avatar src={auth.user?.avatar_url} name={auth.user?.nickname} size="md" />
	<div class="flex h-10 flex-1 items-center rounded-md input-surface px-4">
		<input
			bind:this={inputEl}
			bind:value={input}
			aria-label="评论输入框"
			onkeydown={(e) => {
				// IME 组合中的回车是选词确认，不能当发送（中文输入法致命）
				if (e.key === 'Enter' && !e.isComposing) onSend();
			}}
			placeholder="写下你的评论…"
			class="flex-1 bg-transparent text-md text-text-primary"
		/>
	</div>
	<button
		type="button"
		class="hover-btn hit-slop scale-btn flex size-9 shrink-0 items-center justify-center rounded-full transition-all duration-standard ease-standard focus-ring
			{canSend ? 'bg-accent-default text-text-on-accent shadow-raised-sm hover:shadow-raised-md' : 'bg-surface-subtle text-text-secondary'}"
		onclick={() => onSend()}
		aria-label="发送评论"
		disabled={!canSend}
	>
		<Send class="size-3.5" style="transform:translateX(1px);" strokeWidth={2} />
	</button>
</div>
