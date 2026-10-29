<script lang="ts">
	import { ShieldAlert } from '@lucide/svelte';
	import Button from '@/components/ui/button.svelte';

	// 账号封禁全屏覆盖（spec 错误码 10201）。由 auth store 在 bannedMessage
	// 非空时全局渲染，唯一出口是「知道了」——会话已在客户端丢弃。

	let { message, onAcknowledge }: { message: string | null; onAcknowledge: () => void } = $props();
</script>

<div class="fixed inset-0 z-[var(--cst-z-toast)] flex items-center justify-center p-6 bg-surface-primary" >
	<div class="flex flex-col items-center text-center gap-4"  style="max-width:360px">
		<div
			class="flex items-center justify-center rounded-full size-16"
			 style="background:var(--cst-status-danger-bg);"
		>
			<ShieldAlert size={30} color="var(--cst-status-danger)" strokeWidth={1.5} />
		</div>
		<!-- 标题字重对齐全站 h1-h4 基线（base.css = font-medium）；
		     font-semibold 只留给数字/统计、卡片主标题、激活态 -->
		<h1 role="alert" class="text-xl font-medium tracking-title text-text-primary" >
			账号已被封禁
		</h1>
		<p class="text-sm leading-relaxed text-text-secondary" >
			{message ?? '你的账号因违规操作被封禁。如有疑问，请联系客服。'}
		</p>
		<Button type="button" variant="primary" size="md" class="mt-2 px-8" onclick={onAcknowledge}>
			知道了
		</Button>
	</div>
</div>
