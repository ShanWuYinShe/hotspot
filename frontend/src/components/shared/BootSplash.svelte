<script lang="ts">
	import imgHotSpot from '@/assets/hotspot-logo.png';

	// 启动等待动画：以 HotSpot 龙环 logo 为核心的纯 SVG + CSS 动画。
	// 从全屏游龙 SVG 提取的三层精华（token 化重述，非原样搬运）：
	// 1. 拖尾流光：外环长弧渐变 accent（尾）→ warning 鎏金（首），对应原 inkTrail
	//    的红→金→亮白走向；亮白首端在浅色主题下无对比，改以 warning 收尾。
	// 2. 星火粒子：主彗星（accent）+ 对侧余烬小点（warning），对应原龙首粒子流，
	//    收敛为同组双光点，不另起动画（零新增重绘成本）。
	// 3. 龙首呼吸辉光：中心高清透明 logo 缩放呼吸 + accent 光晕，对应原龙首的
	//    呼吸/摆动与 strongGlow（旋转摆动省略：启动位观感求稳，呼吸已足够）。
	// 舍弃：全屏 S 轨迹 animateMotion、太极底纹、feGaussianBlur 辉光滤镜——
	// 前两者与居中启动位形态冲突，后者全屏重绘代价高，CSS drop-shadow 足够。
	// prefers-reduced-motion 由全局 utilities.css 降级规则接管（CSS 动画压到
	// 0.01ms、iteration-count 1，自然定格为静态 logo 徽记）。
	// 色彩全部走主题 token，明暗与 vibrant/forest/inkwash 三主题自动适配。
</script>

<div class="boot-splash" role="status" aria-label="加载中">
	<svg viewBox="0 0 120 120" width="120" height="120" aria-hidden="true">
		<defs>
			<linearGradient id="boot-arc-grad" x1="0" y1="0" x2="1" y2="1">
				<stop offset="0%" style="stop-color: var(--cst-accent-default);" stop-opacity="0.08" />
				<stop offset="55%" style="stop-color: var(--cst-accent-default);" stop-opacity="0.9" />
				<stop offset="82%" style="stop-color: var(--cst-status-warning);" stop-opacity="1" />
				<stop offset="100%" style="stop-color: var(--cst-status-warning);" stop-opacity="1" />
			</linearGradient>
		</defs>

		<!-- 底环：静态轨道，提供弧线的「缺口」参照 -->
		<circle class="track" cx="60" cy="60" r="46" fill="none" stroke-width="3" />

		<!-- 外环：流光长弧 + 主彗星 + 对侧余烬，正向旋转（视觉主运动） -->
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
			<circle class="ember" cx="60" cy="106" r="2" />
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

		<!-- 中心 logo：高清透明龙环 + 呼吸 + 辉光 -->
		<image href={imgHotSpot} x="34" y="34" width="52" height="52" class="logo-breath" />
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

	.ember {
		fill: var(--cst-status-warning);
		filter: drop-shadow(0 0 4px var(--cst-status-warning));
		opacity: 0.9;
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
		/* 辉光底座：--cst-glow-accent 在浅色模式是全透明，故用实色 accent；
		   6px 小半径仅包裹 logo 剪影，浅色下亦克制 */
		filter: drop-shadow(0 0 6px var(--cst-accent-default));
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
