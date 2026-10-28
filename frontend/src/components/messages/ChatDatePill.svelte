<script lang="ts">
	// ChatDatePill — 聊天日期分隔标签（私聊/群聊共用；文件名沿用历史命名）。
	// label 由 groupMessagesByDay 生成：今天 / 昨天 / M月D日（跨年带年份），
	// 每个日期分组前渲染一次，避免只看时分误判跨天消息顺序。
	//
	// R16 降噪：旧实现是「白底 + 投影 + rounded-full」的小胶囊，在聊天画布上像一个
	// 可点的按钮（用户实测「看起来很奇怪」）。改为中性轻量分节标签：无底色、无投影、
	// 无圆角控件外观，只保留 11px 次级文字 + 居中排布，让日期退成「阅读坐标」而非控件。

	let { label = '今天' }: { label?: string } = $props();
</script>

<div class="flex justify-center py-1.5" data-chat-date-label>
	<span class="text-xs font-medium text-text-secondary tabular-nums">
		{label}
	</span>
</div>
