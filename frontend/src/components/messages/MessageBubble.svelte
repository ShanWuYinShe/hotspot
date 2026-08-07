<script lang="ts">
	import { CheckCheck } from '@lucide/svelte';

	// MessageBubble — 聊天气泡单元（私聊 ChatView / 群聊 GroupChatView 共用）。
	// 发送方向由 sent 决定（右对齐 + accent 填充 + 圆角尾角在右上；收到侧对称）。
	// showCheck 在最后一条已发消息上渲染已读双勾。

	let {
		text,
		sent,
		time,
		showCheck = false
	}: {
		text: string;
		sent: boolean;
		time: string;
		showCheck?: boolean;
	} = $props();
</script>

<div class="animate-fade-in flex {sent ? 'justify-end' : 'justify-start'}">
		<div
			class="{sent ? 'bg-accent-default shadow-glow-md' : 'bg-surface-primary shadow-raised-sm border border-border-default'} max-w-[76%] px-3.5 py-2.5 {sent
				? 'rounded-[var(--cst-shape-md)_var(--cst-shape-sm)_var(--cst-shape-md)_var(--cst-shape-md)]'
				: 'rounded-[var(--cst-shape-sm)_var(--cst-shape-md)_var(--cst-shape-md)_var(--cst-shape-md)]'}"
		>
			<p class="text-sm leading-card {sent ? 'text-text-on-accent' : 'text-text-primary'}">
				{text}
			</p>
			<p
				class="mt-1 flex items-center gap-0.75 text-xxs {sent ? 'justify-end text-text-on-accent opacity-70' : 'justify-start text-text-secondary'}"
			>
			{time}
			{#if sent && showCheck}
				<CheckCheck class="size-3"  style="opacity:0.8" strokeWidth={2} />
			{/if}
		</p>
	</div>
</div>
