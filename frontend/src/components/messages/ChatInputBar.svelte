<script lang="ts">
	import { Send, Smile } from '@lucide/svelte';
	import type { ChatController } from '@/lib/chat.svelte';

	// ChatInputBar —— 私聊/群聊共享的输入栏（发送错误提示 + 表情占位 + 输入框 + 发送钮）。
	// 表情按钮仅私聊布局展示（showEmojiButton，均为「暂未开放」占位）。
	let { chat, showEmojiButton = false }: { chat: ChatController; showEmojiButton?: boolean } = $props();
</script>

{#if chat.sendError}
	<p class="px-4 pb-1.5 text-status-warning text-xs">{chat.sendError}</p>
{/if}
<div class="shrink-0 flex items-center gap-2.5 px-4 py-3 bg-surface-primary border-t border-border-default">
	{#if showEmojiButton}
		<button
			type="button"
			class="hover-btn scale-btn flex items-center justify-center rounded-full border border-border-default bg-surface-primary size-9 shrink-0 shadow-raised-sm focus-ring disabled:opacity-muted"
			aria-label="表情"
			disabled
			title="暂未开放"
		>
			<Smile class="size-4 text-text-secondary" strokeWidth={1.5} />
		</button>
	{/if}

	<div class="flex-1 flex items-center px-4 rounded-md input-surface h-9.5">
		<input
			type="text"
			aria-label="消息输入框"
			bind:value={chat.inputVal}
			oninput={(e) => chat.handleInputChange(e)}
			onkeydown={(e) => {
				// IME 组合中的回车是选词确认，不能当发送（中文输入法致命）
				if (e.key === 'Enter' && !e.isComposing) void chat.handleSend();
			}}
			placeholder="发消息…"
			class="flex-1 bg-transparent text-text-primary text-sm"
		/>
	</div>

	<button
		type="button"
		class="hover-btn scale-btn flex items-center justify-center rounded-full focus-ring shrink-0 transition-all duration-standard {chat.inputVal.trim() ? 'bg-accent-default' : 'bg-surface-primary'} size-9"
		style="box-shadow:{chat.inputVal.trim() ? 'var(--shadow-glow-md)' : 'var(--shadow-raised-sm)'};border:var(--cst-border-width-hairline) solid var(--cst-border-default);"
		onclick={() => void chat.handleSend()}
		aria-label="发送"
		disabled={!chat.inputVal.trim()}
	>
		<Send class="size-3.5" style="color:{chat.inputVal.trim() ? 'var(--cst-text-on-accent)' : 'var(--cst-text-secondary)'};transform:translateX(1px);" strokeWidth={2} />
	</button>
</div>
