<script lang="ts">
	import { MessageCircle } from '@lucide/svelte';

	// ChatEmptyPanel —— 桌面/宽平板分栏右栏的「未选会话」引导面板。
	// 会话入口统一由左栏承载（完整列表 + 页头「新会话」），此处只做引导：
	// 不再重复列出「最近会话」，也不重复放「发起新会话」按钮。
</script>

<div class="flex h-full items-center justify-center px-6 py-10">
	<div class="flex max-w-sm flex-col items-center text-center">
		<!-- 图标砖：软底圆形（空态语言）；小面板按空态规范不开 spotlight -->
		<div class="flex size-16 items-center justify-center rounded-full bg-accent-soft text-text-accent-on-soft">
			<MessageCircle class="size-7" strokeWidth={1.5} aria-hidden="true" />
		</div>
		<h2 class="mt-5 text-md font-semibold text-text-primary">选择一个会话</h2>
		<p class="mt-2 max-w-xs text-sm leading-relaxed text-text-secondary">
			从左侧列表选择会话继续聊天。
		</p>
	</div>
</div>
