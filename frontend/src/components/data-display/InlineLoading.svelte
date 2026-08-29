<script lang="ts">
	// InlineLoading —— 列表/内容区加载态（spinner + 文案），收敛各页手写的
	// 「spinner + 加载xxx…」样板；与 EmptyState/LoadError 同为三态标配组件。
	let { label = '加载中…', class: cls = '' }: { label?: string; class?: string } = $props();
</script>

<div class="flex flex-col items-center gap-3 py-20 {cls}" role="status" aria-label={label}>
	<div class="spinner size-7"></div>
	<p class="text-text-secondary text-sm">{label}</p>
</div>
