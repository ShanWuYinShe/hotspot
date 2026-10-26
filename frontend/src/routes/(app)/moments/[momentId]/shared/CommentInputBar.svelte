<script lang="ts">
	import { Send } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import { auth } from '@/stores/auth.svelte';

	// CommentInputBar —— 详情页底部固定评论输入栏（Enter/按钮发送均走 onSend）。
	let {
		input = $bindable(''),
		inputEl = $bindable(null),
		onSend
	}: {
		input?: string;
		inputEl?: HTMLInputElement | null;
		onSend: () => void;
	} = $props();
</script>

<div
	class="shrink-0 flex items-center gap-2.5 bg-surface-primary border-t border-border-subtle/50 px-4 pt-3"
	style="padding-bottom:max(0.75rem, env(safe-area-inset-bottom));"
>
	<Avatar class="border border-border-subtle/60" src={auth.user?.avatar_url} name={auth.user?.nickname} size="md" />
	<div class="flex-1 flex items-center px-4 rounded-md input-surface h-9.5">
		<input
			bind:this={inputEl}
			bind:value={input}
			aria-label="评论输入框"
			onkeydown={(e) => {
				// IME 组合中的回车是选词确认，不能当发送（中文输入法致命）
				if (e.key === 'Enter' && !e.isComposing) onSend();
			}}
			placeholder="写下你的评论…"
			class="flex-1 bg-transparent text-text-primary text-md"
		/>
	</div>
	<button
		type="button"
		class="hover-btn hit-slop scale-btn flex items-center justify-center rounded-full border border-border-default focus-ring shrink-0 size-9 transition-all duration-standard ease-standard
			{input.trim() ? 'border-transparent bg-accent-default text-text-on-accent shadow-raised-sm hover:shadow-raised-md' : 'bg-surface-primary text-text-secondary shadow-raised-sm'}"
		onclick={() => onSend()}
		aria-label="发送评论"
		disabled={!input.trim()}
	>
		<Send class="size-3.5" style="transform:translateX(1px);" strokeWidth={2} />
	</button>
</div>
