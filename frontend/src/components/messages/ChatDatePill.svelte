<script lang="ts">
	// ChatDatePill — 聊天日期分隔胶囊（私聊/群聊共用）。
	// label 由 groupMessagesByDay 生成：今天 / 昨天 / M月D日（跨年带年份），
	// 每个日期分组前渲染一次，避免只看时分误判跨天消息顺序。

	let { label = '今天' }: { label?: string } = $props();
</script>

<div class="mb-2 flex justify-center">
	<span
		class="inline-flex h-5 items-center justify-center rounded-full border border-border-default bg-surface-primary px-2.5 text-xxs font-medium leading-none tracking-label text-text-secondary opacity-emphasis"
	>
		{label}
	</span>
</div>
