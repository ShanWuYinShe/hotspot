<script lang="ts">
	// ChatDatePill — 聊天日期分隔胶囊（私聊/群聊共用）。
	// label 由 groupMessagesByDay 生成：今天 / 昨天 / M月D日（跨年带年份），
	// 每个日期分组前渲染一次，避免只看时分误判跨天消息顺序。

	let { label = '今天' }: { label?: string } = $props();
</script>

<div class="mb-2 flex justify-center">
	<span
		class="rounded-full border border-border-default bg-surface-primary px-3 py-1 text-xxs tracking-wide text-text-secondary opacity-emphasis"
	>
		{label}
	</span>
</div>
