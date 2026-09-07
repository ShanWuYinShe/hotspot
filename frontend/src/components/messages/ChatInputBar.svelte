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
<div
	class="shrink-0 flex items-center gap-2.5 bg-surface-chat border-t border-border-default px-4 pt-3"
	style="padding-bottom:max(0.75rem, env(safe-area-inset-bottom));"
>
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
		class="hover-btn scale-btn flex items-center justify-center rounded-full border border-border-default focus-ring shrink-0 transition-all duration-standard size-9
			{chat.inputVal.trim() && !chat.sending
			? 'border-transparent bg-accent-default text-text-on-accent shadow-raised-sm hover:shadow-raised-md'
			: 'bg-surface-primary text-text-secondary shadow-raised-sm'}
			{chat.sending ? 'pointer-events-none opacity-soft' : ''}"
		onclick={() => void chat.handleSend()}
		aria-label="发送"
		aria-busy={chat.sending || undefined}
		disabled={!chat.inputVal.trim() || chat.sending}
	>
		{#if chat.sending}
			<span class="spinner size-4" aria-hidden="true"></span>
		{:else}
			<Send class="size-3.5 {chat.inputVal.trim() ? '' : 'text-text-secondary'}" style="transform:translateX(1px);" strokeWidth={2} />
		{/if}
	</button>
</div>
