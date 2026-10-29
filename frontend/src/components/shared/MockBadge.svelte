<script lang="ts">
	import { FlaskConical } from '@lucide/svelte';
	import Badge from '@/components/data-display/Badge.svelte';
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
		<!-- 默认只显示图标（24px 圆点），桌面 hover 才展开「Mock 数据」文字：
		     原为常驻文字徽标（约 90×24），在移动端与弹层/列表正文叠压（全站复审
		     S4-05 / s3 P2-10）；信息不丢——外层 aria-label 始终带完整说明。
		     2026-10-29：外壳收敛到 data-display/Badge（warning 语义），本组件只保留
		     折叠/展开的布局几何与动效。 -->
		<Badge
			size="md"
			tone="warning"
			class="group h-6 w-6 gap-1.5 px-0 shadow-raised-md transition-all duration-standard md:w-auto md:px-2.5"
		>
			<FlaskConical class="size-3 shrink-0" strokeWidth={2.2} />
			<span class="hidden max-w-0 overflow-hidden whitespace-nowrap opacity-0 transition-all duration-standard md:inline md:group-hover:max-w-20 md:group-hover:opacity-100">Mock 数据</span>
		</Badge>
	</div>
{/if}
