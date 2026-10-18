<script lang="ts">
	import { FlaskConical } from '@lucide/svelte';
	import { isMockEnabled, subscribe } from '@/mocks/toggle';

	// Dev-only Mock 数据指示徽标：mock 接管后全页数据均为假数据，此前没有任何
	// 视觉提示，多次把联调中的人误导（mock 用户/内容当成真数据排查半天）。
	// 仅 dev 构建渲染（import.meta.env.DEV），生产包无此节点；开关经 toggle
	// 订阅实时跟随（设置页/调试切换后 reload 即现）。
	// 位置：移动端左下——右下角与「加入」等转化按钮重叠（Gemini UI 审查 r62/r73；
	// pointer-events-none 只保证可点不保证不遮字）；桌面上移至 bottom-24，
	// 避让聊天输入栏发送按钮（第二轮审查 r87）。
	let mockOn = $state(false);
	$effect(() => {
		mockOn = isMockEnabled();
		return subscribe(() => (mockOn = isMockEnabled()));
	});
	const show = $derived(import.meta.env.DEV && mockOn);
</script>

{#if show}
	<div
		class="pointer-events-none fixed left-4 bottom-24 z-[var(--cst-z-toast)] md:left-auto md:right-4 md:bottom-24"
		role="status"
		aria-label="当前页面数据来自本地 Mock，非真实后端"
	>
		<!-- 默认只显示图标（24px 圆点），桌面 hover/focus 才展开「Mock 数据」文字：
		     原为常驻文字徽标（约 90×24），在移动端与弹层/列表正文叠压（全站复审
		     S4-05 / s3 P2-10）；信息不丢——aria-label/title 始终带完整说明。 -->
		<div
			title="开发联调提示：当前页面数据来自本地 Mock，非真实后端"
			class="group inline-flex items-center justify-center leading-none gap-1.5 h-6 w-6 px-0 md:w-auto md:px-2.5 rounded-full border border-status-warning-border bg-surface-primary/95 text-xxs font-medium text-status-warning shadow-raised-md backdrop-blur-md transition-all duration-standard md:group-hover:px-2.5"
		>
			<FlaskConical class="size-3 shrink-0" strokeWidth={2.2} />
			<span class="hidden md:inline max-w-0 overflow-hidden whitespace-nowrap opacity-0 transition-all duration-standard group-hover:max-w-20 group-hover:opacity-100">Mock 数据</span>
		</div>
	</div>
{/if}
