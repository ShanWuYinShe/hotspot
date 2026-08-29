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

<div class="shrink-0 flex items-center gap-2.5 px-4 py-3 bg-surface-primary border-t border-border-default">
	<Avatar class="border border-border-default" src={auth.user?.avatar_url} name={auth.user?.nickname} size={30} />
	<div class="flex-1 flex items-center px-4 rounded-md input-surface h-9.5">
		<input
			bind:this={inputEl}
			bind:value={input}
			onkeydown={(e) => {
				// IME 组合中的回车是选词确认，不能当发送（中文输入法致命）
				if (e.key === 'Enter' && !e.isComposing) onSend();
			}}
			placeholder="写下你的评论…"
			class="flex-1 bg-transparent text-text-primary text-sm"
		/>
	</div>
	<button
		type="button"
		class="hover-btn scale-btn tap-88 flex items-center justify-center rounded-full focus-ring shrink-0 size-9" style="background:{input.trim() ? 'var(--cst-accent-default)' : 'var(--cst-surface-primary)'};box-shadow:{input.trim() ? 'var(--shadow-glow-md)' : 'var(--shadow-raised-sm)'};border:var(--cst-border-width-hairline) solid var(--cst-border-default);transition:all var(--cst-motion-standard) var(--cst-ease-standard);"
		onclick={() => onSend()}
	>
		<Send class="size-3.5" style="color:{input.trim() ? 'var(--cst-text-on-accent)' : 'var(--cst-text-secondary)'};transform:translateX(1px);" strokeWidth={2} />
	</button>
</div>
