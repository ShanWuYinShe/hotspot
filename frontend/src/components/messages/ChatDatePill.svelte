<script lang="ts">
	// ChatDatePill — 聊天时间分隔胶囊（消息列表顶部的「今天」标记，私聊/群聊共用）。
	// 后续接入服务端日期分节时以 label 传入对应日期文案。

	let { label = '今天' }: { label?: string } = $props();
</script>

<div class="mb-2 flex justify-center">
	<span
		class="rounded-full border border-border-default bg-surface-primary px-3 py-1 text-xxs tracking-wide text-text-secondary opacity-80"
	>
		{label}
	</span>
</div>
