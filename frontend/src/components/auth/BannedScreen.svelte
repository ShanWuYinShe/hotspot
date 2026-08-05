<script lang="ts">
	import { ShieldAlert } from '@lucide/svelte';

	// 账号封禁全屏覆盖（spec 错误码 10201）。由 auth store 在 bannedMessage
	// 非空时全局渲染，唯一出口是「知道了」——会话已在客户端丢弃。

	let { message, onAcknowledge }: { message: string | null; onAcknowledge: () => void } = $props();
</script>

<div class="fixed inset-0 z-[var(--z-toast)] flex items-center justify-center p-6" style="background:var(--surface-primary);">
	<div class="flex flex-col items-center text-center" style="max-width:360px;gap:16px;">
		<div
			class="flex items-center justify-center rounded-full"
			style="width:64px;height:64px;background:var(--color-danger-bg);"
		>
			<ShieldAlert size={30} color="var(--status-danger)" strokeWidth={1.6} />
		</div>
		<h1 class="text-[20px] font-semibold tracking-tight" style="color:var(--text-primary);">
			账号已被封禁
		</h1>
		<p class="text-[14px] leading-relaxed" style="color:var(--text-secondary);">
			{message ?? '你的账号因违规操作被封禁。如有疑问，请联系客服。'}
		</p>
		<button
			type="button"
			class="hover-btn mt-2 rounded-xl px-8 py-2.5 text-[15px] font-medium"
			style="background:var(--accent-default);color:white;"
			onclick={onAcknowledge}
		>
			知道了
		</button>
	</div>
</div>
