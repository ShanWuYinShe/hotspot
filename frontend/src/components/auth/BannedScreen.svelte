<script lang="ts">
	import { ShieldAlert } from '@lucide/svelte';

	// 账号封禁全屏覆盖（spec 错误码 10201）。由 auth store 在 bannedMessage
	// 非空时全局渲染，唯一出口是「知道了」——会话已在客户端丢弃。

	let { message, onAcknowledge }: { message: string | null; onAcknowledge: () => void } = $props();
</script>

<div class="fixed inset-0 z-[var(--cst-z-toast)] flex items-center justify-center p-6" style="background:var(--cst-surface-primary);">
	<div class="flex flex-col items-center text-center gap-4"  style="max-width:360px">
		<div
			class="flex items-center justify-center rounded-full size-16"
			 style="background:var(--cst-color-danger-bg);"
		>
			<ShieldAlert size={30} color="var(--cst-status-danger)" strokeWidth={1.6} />
		</div>
		<h1 class="text-xl font-semibold tracking-tight" style="color:var(--cst-text-primary);">
			账号已被封禁
		</h1>
		<p class="text-sm leading-relaxed" style="color:var(--cst-text-secondary);">
			{message ?? '你的账号因违规操作被封禁。如有疑问，请联系客服。'}
		</p>
		<button
			type="button"
			class="hover-btn mt-2 rounded-sm px-8 py-2.5 text-sm font-medium"
			style="background:var(--cst-accent-default);color:var(--cst-text-on-accent);"
			onclick={onAcknowledge}
		>
			知道了
		</button>
	</div>
</div>
