<script lang="ts">
	import imgHotSpot from '@/assets/hotspot-logo.png';

	// 启动等待动画：以 HotSpot 龙环 logo 为核心的纯 SVG + CSS 动画。
	// 三层结构：accent 渐变长弧+彗星光点（外环正向）、低透明虚线细环（反向慢转）、
	// 中心 logo 呼吸。prefers-reduced-motion 由全局 utilities.css 降级规则接管
	// （动画时长压到 0.01ms、iteration-count 1，自然定格为静态 logo 徽记）。
	// 色彩全部走主题 token（--cst-accent-default / --cst-border-default），
	// 明暗主题与 vibrant/forest 双主题自动适配。
</script>

<div class="boot-splash" role="status" aria-label="加载中">
	<svg viewBox="0 0 120 120" width="120" height="120" aria-hidden="true">
		<defs>
			<linearGradient id="boot-arc-grad" x1="0" y1="0" x2="1" y2="1">
				<stop offset="0%" style="stop-color: var(--cst-accent-default);" stop-opacity="0.95" />
				<stop offset="100%" style="stop-color: var(--cst-accent-default);" stop-opacity="0.08" />
			</linearGradient>
		</defs>

		<!-- 底环：静态轨道，提供弧线的「缺口」参照 -->
		<circle class="track" cx="60" cy="60" r="46" fill="none" stroke-width="3" />

		<!-- 外环：渐变长弧 + 彗星光点，正向旋转（视觉主运动） -->
		<g class="orbit">
			<circle
				cx="60"
				cy="60"
				r="46"
				fill="none"
				stroke="url(#boot-arc-grad)"
				stroke-width="3"
				stroke-linecap="round"
				stroke-dasharray="72 217"
			/>
			<circle class="comet" cx="60" cy="14" r="3" />
		</g>

		<!-- 内环：虚线细环反向慢转（层次感） -->
		<g class="orbit-rev">
			<circle
				cx="60"
				cy="60"
				r="36"
				fill="none"
				stroke-width="1.5"
				stroke-dasharray="2 6"
				class="dash-ring"
				stroke-linecap="round"
			/>
		</g>

		<!-- 中心 logo：呼吸缩放（龙环图与旋转弧同色系，视觉上像被环包裹） -->
		<image href={imgHotSpot} x="38" y="38" width="44" height="44" class="logo-breath" />
	</svg>
</div>

<style>
	.boot-splash {
		display: grid;
		place-items: center;
	}

	.track {
		stroke: var(--cst-border-default);
	}

	.comet {
		fill: var(--cst-accent-default);
		/* 彗星光点的发光尾焰：小面积 drop-shadow，代价可忽略 */
		filter: drop-shadow(0 0 5px var(--cst-accent-default));
	}

	.dash-ring {
		stroke: var(--cst-accent-default);
		opacity: 0.35;
	}

	/* SVG 内 CSS transform-origin 以 viewBox 坐标计（transform-box 默认 view-box） */
	.orbit {
		transform-origin: 60px 60px;
		animation: boot-spin 1.6s linear infinite;
	}

	.orbit-rev {
		transform-origin: 60px 60px;
		animation: boot-spin 5.2s linear infinite reverse;
	}

	.logo-breath {
		transform-origin: 60px 60px;
		animation: boot-breath 2.2s ease-in-out infinite;
	}

	@keyframes boot-spin {
		to {
			transform: rotate(360deg);
		}
	}

	@keyframes boot-breath {
		0%,
		100% {
			transform: scale(1);
			opacity: 0.92;
		}
		50% {
			transform: scale(1.07);
			opacity: 1;
		}
	}
</style>
