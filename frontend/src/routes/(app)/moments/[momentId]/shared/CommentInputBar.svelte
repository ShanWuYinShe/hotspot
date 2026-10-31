<script lang="ts">
	import { Send } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import { auth } from '@/stores/auth.svelte';

	// CommentInputBar —— 瞬间详情的评论输入器（Enter/按钮发送均走 onSend）。
	// 2026-10-31 移动端体验升级：
	//   inline 桌面/平板：嵌在评论区内（软底浅槽），与卡片同圆角/同底色；
	//   docked 移动端：通栏贴底底栏（全宽贴边 + 顶部分割线 + 白底毛玻璃 + 底部实底封死安全区），
	//          彻底废除悬浮胶囊 dock，避免滚动时字句从缝隙中穿插漏光。
	let {
		input = $bindable(''),
		inputEl = $bindable(null),
		onSend,
		variant = 'docked'
	}: {
		input?: string;
		inputEl?: HTMLInputElement | null;
		onSend: () => void;
		variant?: 'inline' | 'docked' | 'floating';
	} = $props();

	const canSend = $derived(input.trim().length > 0);
	const shell = $derived(
		variant === 'inline'
			? 'rounded-md bg-surface-subtle/40 p-2'
			: 'w-full px-3 py-2 flex items-center gap-2.5'
	);
</script>

<div class="flex items-center gap-2.5 {shell}">
	<Avatar src={auth.user?.avatar_url} name={auth.user?.nickname} size="md" />
	<div class="flex flex-1 items-center rounded-full input-surface px-3.5 py-1.5">
		<input
			bind:this={inputEl}
			bind:value={input}
			aria-label="评论输入框"
			onkeydown={(e) => {
				// IME 组合中的回车是选词确认，不能当发送（中文输入法致命）
				if (e.key === 'Enter' && !e.isComposing) onSend();
			}}
			placeholder="写下你的评论…"
			class="flex-1 bg-transparent text-md text-text-primary placeholder:text-text-secondary"
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
		<Send class="size-4" style="transform:translateX(1px);" strokeWidth={2} />
	</button>
</div>
