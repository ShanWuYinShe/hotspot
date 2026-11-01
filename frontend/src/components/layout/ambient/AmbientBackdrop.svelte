<script lang="ts">
	/**
	 * AmbientBackdrop — 页面氛围画布（纯装饰层）
	 *
	 * 设计定位：给页面注入现代、通透、高质感的环境光漫射与天光雾层。
	 * 彻底摒弃繁杂生硬的天际线/山林/等高线等具象图案，避免喧宾夺主。
	 * 依靠精细设计的天顶天光（sky）、中心微光（glow-a/b）、地平线漫射（horizon）
	 * 与聚焦暗角（vignette），让浅色与深色模式呈现开阔、干净的纯粹美感。
	 */

	let {
		variant = 'canvas',
		orientation = 'h',
		motion = false
	}: {
		variant?: 'canvas' | 'chrome';
		orientation?: 'h' | 'v';
		motion?: boolean;
	} = $props();
</script>

<div
	class="ambient-backdrop"
	class:ambient-backdrop--chrome={variant === 'chrome'}
	class:ambient-backdrop--chrome-h={variant === 'chrome' && orientation === 'h'}
	class:ambient-backdrop--chrome-v={variant === 'chrome' && orientation === 'v'}
	class:ambient-backdrop--motion={motion && variant !== 'chrome'}
	aria-hidden="true"
>
	<div class="ambient-stage">
		<div class="ambient-layer ambient-sky"></div>
		<!-- 天顶微光：纯净单层天光与柔和微漫射 -->
		<div class="ambient-layer ambient-glow ambient-glow--a"></div>
		<div class="ambient-layer ambient-glow ambient-glow--b"></div>

		{#if variant === 'canvas'}
			<div class="ambient-layer ambient-horizon"></div>
			<div class="ambient-layer ambient-vignette"></div>
		{/if}
	</div>
</div>
