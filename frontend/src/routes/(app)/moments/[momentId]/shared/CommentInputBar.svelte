<script lang="ts">
	import { Send } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import { auth } from '@/stores/auth.svelte';

	// CommentInputBar —— 瞬间详情的评论输入器（Enter/按钮发送均走 onSend）。
	// 2026-10-29 详情页改版：不再用「整条白底 + 硬边 border-t」的全宽底栏（用户反馈与本卡格格不入、
	// 移动端还与动作条叠成两条横杠）。改为同一输入器的两种登记形态：
	//   inline  桌面/平板：嵌在评论区内（软底浅槽），与卡片同圆角/同底色，不再悬浮成第三条带；
	//   floating 移动端：悬浮胶囊 dock（玻璃/白底 + 细边 + 投影 + 左右 inset + 安全区外），
	//            不再通栏贴边，视觉上属于画布而不是一条独立横条。
	// 输入框仍走输入族原语 .input-surface（39px 档），发送钮 36px 圆钮；空态不带投影。
	let {
		input = $bindable(''),
		inputEl = $bindable(null),
		onSend,
		variant = 'floating'
	}: {
		input?: string;
		inputEl?: HTMLInputElement | null;
		onSend: () => void;
		variant?: 'inline' | 'floating';
	} = $props();

	const canSend = $derived(input.trim().length > 0);
	const shell = $derived(
		variant === 'inline'
			? 'rounded-md bg-surface-subtle/40 p-2'
			: 'mx-3 mb-1 rounded-md border border-border-default/60 bg-surface-primary/95 px-3 py-2 shadow-raised-md backdrop-blur-md'
	);
</script>

<div class="flex items-center gap-2.5 {shell}">
	<Avatar src={auth.user?.avatar_url} name={auth.user?.nickname} size="md" />
	<div class="flex flex-1 items-center rounded-md input-surface px-3.5 py-2">
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
		class="hover-btn hit-slop scale-btn flex size-10 shrink-0 items-center justify-center rounded-full transition-all duration-standard ease-standard focus-ring
			{canSend ? 'bg-accent-default text-text-on-accent shadow-raised-sm hover:shadow-raised-md' : 'bg-surface-subtle text-text-secondary'}"
		onclick={() => onSend()}
		aria-label="发送评论"
		disabled={!canSend}
	>
		<Send class="size-4" style="transform:translateX(1px);" strokeWidth={2} />
	</button>
</div>
