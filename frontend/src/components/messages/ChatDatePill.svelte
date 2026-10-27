<script lang="ts">
	// ChatDatePill — 聊天日期分隔胶囊（私聊/群聊共用）。
	// label 由 groupMessagesByDay 生成：今天 / 昨天 / M月D日（跨年带年份），
	// 每个日期分组前渲染一次，避免只看时分误判跨天消息顺序。
	// R13：去掉 1px 描边（安静精致）——用软底 + 极轻投影在聊天画布上承担分隔，
	// 与消息气泡同层但不竞争注意力。

	let { label = '今天' }: { label?: string } = $props();
</script>

<div class="mb-2 flex justify-center">
	<span
		class="inline-flex h-5 items-center justify-center rounded-full bg-surface-primary px-2.5 text-xxs font-medium leading-none tracking-label text-text-secondary shadow-xs"
	>
		{label}
	</span>
</div>
