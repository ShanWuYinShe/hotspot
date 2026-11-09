<script lang="ts">
	import { Send } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { cn } from '@/lib/utils';

	// CommentInputBar —— 瞬间详情的评论输入器（Enter/按钮发送均走 onSend）。
	// 视觉契约：全端 docked（由 SplitPageLayout footer 承载，桌面 sticky bottom-0 /
	// 移动 fixed），通栏底栏（横向 padding + 顶部分割线 + 毛玻璃实底），
	// 不再提供 inline 内嵌形态——评论输入器始终常驻贴底，随时可发。
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
	// 桌面 docked 栏与内容卡同宽后，内部左右 padding 对齐内容列 gutter（28px），
	// 头像/发送钮与评论内容轴齐平；平板/手机保持通栏 12px。
	const shell = 'w-full px-3 xl:px-[var(--cst-page-gutter)] py-2 flex items-center gap-2.5';
</script>

<div class={cn('flex items-center', shell)}>
	<Avatar src={auth.user?.avatar_url} name={auth.user?.nickname} size="md" class="shrink-0" />
	<div class="group flex flex-1 items-center rounded-full bg-surface-subtle/50 hover:bg-surface-subtle/70 focus-within:bg-surface-primary focus-within:ring-2 focus-within:ring-accent-default/30 border border-border-default/50 px-3.5 sm:px-4 py-1.5 sm:py-2 transition-all">
		<input
			bind:this={inputEl}
			bind:value={input}
			aria-label="评论输入框"
			onkeydown={(e) => {
				// IME 组合中的回车是选词确认，不能当发送（中文输入法致命）
				if (e.key === 'Enter' && !e.isComposing) onSend();
			}}
			placeholder="写下你的评论…"
			class="flex-1 bg-transparent text-sm sm:text-md text-text-primary placeholder:text-text-secondary focus:outline-none"
		/>
	</div>
	<button
		type="button"
		class="hover-btn hit-slop scale-btn flex size-9 shrink-0 items-center justify-center rounded-full transition-all duration-standard ease-standard focus-ring
			{canSend ? 'bg-accent-default text-text-on-accent shadow-raised-sm hover:shadow-raised-md active:scale-95' : 'bg-surface-subtle text-text-secondary cursor-not-allowed'}"
		onclick={() => onSend()}
		aria-label="发送评论"
		disabled={!canSend}
	>
		<Send class="size-4" style="transform:translateX(1px);" strokeWidth={2} />
	</button>
</div>
