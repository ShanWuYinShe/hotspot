<script lang="ts">
	/**
	 * AmbientBackdrop — 页面氛围画布（纯装饰层）
	 *
	 * 设计目的：给内容稀疏/空态的页面补「世界感」层次，避免大面积纯白/纯灰看起来空荡。
	 * 借鉴游戏场景的分层做法：天光（sky/sun/glow）→ 地平线雾（horizon）→
	 * 远景世界（stars/constellation/orbit 或 contours 或 ink mountains，底部带剪影）→
	 * 暗角（vignette）。全部压在正文内容之下（z-index:-1），不参与布局、不接收指针事件。
	 *
	 * 三主题各有一套「世界」语言（色值全部来自 themes/*.css 的 --cst-ambient-* token）：
	 *   vibrant 星际：星尘 + 星座连线 + 轨道环 + 城市天际线剪影 + 青蓝天光
	 *   forest  清野：等高线地形 + 暖阳 + 树林剪影 + 暖黄地平线
	 *   inkwash 水墨：三段墨山 + 朱砂圆相 + 飞白墨点 + 宣纸地平线
	 *
	 * 世界层按当前主题只渲染一套（主题由 stores/theme 驱动，避免三套 SVG 同时挂 DOM，
	 * 也避开 CSS 层叠（unlayered vs @layer）对显隐的干扰）。
	 * 几何/星点全部确定性生成，不做运行时随机，避免水合与截图抖动。
	 */
	import { theme } from '@/stores/theme.svelte';

	interface Star {
		x: number;
		y: number;
		r: number;
		o: number;
	}

	/** 确定性伪随机：同一构建下星点位置/大小稳定（避免每次渲染的视觉抖动）。 */
	function hash(n: number): number {
		const v = Math.sin(n * 127.1) * 43758.5453;
		return v - Math.floor(v);
	}

	/** 星尘：集中在上 2/3，越靠上越密（近地平线渐稀），大小/亮度两档错落。 */
	const STARS: Star[] = Array.from({ length: 84 }, (_, i) => ({
		x: Math.round(hash(i + 1) * 1600),
		y: Math.round(Math.pow(hash(i + 37), 1.3) * 600),
		r: Number((0.5 + Math.pow(hash(i + 73), 2) * 1.5).toFixed(1)),
		o: Number((0.16 + hash(i + 109) * 0.5).toFixed(2))
	}));

	/** Catmull-Rom → 三次贝塞尔，把折线点转成柔顺曲线（等高线用）。 */
	function smoothPath(points: Array<{ x: number; y: number }>): string {
		if (points.length < 2) return '';
		let d = `M${points[0].x} ${points[0].y}`;
		for (let i = 0; i < points.length - 1; i++) {
			const p0 = points[Math.max(0, i - 1)];
			const p1 = points[i];
			const p2 = points[i + 1];
			const p3 = points[Math.min(points.length - 1, i + 2)];
			const c1x = p1.x + (p2.x - p0.x) / 6;
			const c1y = p1.y + (p2.y - p0.y) / 6;
			const c2x = p2.x - (p3.x - p1.x) / 6;
			const c2y = p2.y - (p3.y - p1.y) / 6;
			d += ` C${c1x.toFixed(1)} ${c1y.toFixed(1)}, ${c2x.toFixed(1)} ${c2y.toFixed(1)}, ${p2.x} ${p2.y}`;
		}
		return d;
	}

	/** 等高线地形：9 条上下叠置的柔顺曲线，越往下振幅越大（近处地形起伏更明显）。 */
	const CONTOURS: string[] = Array.from({ length: 9 }, (_, i) => {
		const base = 470 + i * 46;
		const amp = 22 + i * 6;
		const points = Array.from({ length: 17 }, (_, s) => {
			const t = s / 16;
			return {
				x: Math.round(t * 1600),
				y: Math.round(
					base +
						Math.sin(t * Math.PI * 2 + i * 0.9) * amp +
						Math.sin(t * Math.PI * 4 + i * 1.7) * amp * 0.32
				)
			};
		});
		return smoothPath(points);
	});

	/** 星座连线：三段折线，端点刻意靠近星点群（不要求逐点吸附，避免解剖感过强）。 */
	const CONSTELLATIONS = [
		'M232 168 L318 214 L286 296 L372 330',
		'M1084 118 L1156 176 L1122 250 L1204 286',
		'M640 92 L700 148 L664 208'
	];

	/**
	 * 城市天际线：由 [x, 宽, 高] 生成楼群剪影路径（底部对齐 900）。
	 * 顶部加地面带让楼群连成一条城市轮廓（否则分离的矩形会被误读为骨架屏占位）；
	 * 高楼层加细天线，强化「城市」语义。
	 */
	function skyline(buildings: Array<[number, number, number]>, roofs = false): string {
		const base = 'M0 900 V883 H1600 V900 Z ';
		return (
			base +
			buildings
				.map(([x, w, h]) => {
					const body = `M${x} 900 V${900 - h} H${x + w} V900 Z`;
					if (!roofs || h < 150) return body;
					const antennaX = x + w * 0.5;
					return `${body} M${antennaX - 2} ${900 - h} V${900 - h - 24} H${antennaX + 2} V${900 - h} Z`;
				})
				.join(' ')
		);
	}

	/** 树林剪影：由 [x, 宽, 高] 生成针叶树路径（双峰，底部对齐 900）。 */
	function treeLine(trees: Array<[number, number, number]>): string {
		return trees
			.map(
				([x, w, h]) =>
					`M${x} 900 L${x + w * 0.34} ${900 - h * 0.52} L${x + w * 0.5} ${900 - h} ` +
					`L${x + w * 0.66} ${900 - h * 0.58} L${x + w} 900 Z`
			)
			.join(' ');
	}

	// 星际：远处低矮楼群 + 近处高低错落的天际线（数字化城市，克制不抢主体）
	const SKYLINE_FAR = skyline([
		[0, 46, 44], [54, 34, 58], [96, 60, 36], [164, 42, 62], [214, 70, 42],
		[292, 46, 30], [346, 56, 54], [410, 38, 66], [456, 66, 38], [530, 44, 52],
		[582, 52, 34], [642, 60, 60], [710, 40, 40], [758, 68, 30], [834, 46, 56],
		[888, 56, 44], [952, 40, 64], [1000, 66, 36], [1074, 44, 52], [1126, 62, 30],
		[1196, 48, 58], [1252, 66, 40], [1326, 42, 62], [1376, 56, 34], [1440, 64, 54],
		[1512, 44, 42], [1564, 36, 60]
	]);
	const SKYLINE_NEAR = skyline(
		[
			[24, 88, 72], [128, 70, 96], [212, 104, 64], [332, 78, 108], [426, 112, 78],
			[556, 84, 100], [656, 96, 58], [770, 74, 112], [858, 108, 74], [982, 82, 92],
			[1080, 100, 62], [1196, 78, 104], [1290, 92, 70], [1398, 84, 98], [1494, 88, 66]
		]
	);

	// 清野：远处矮树线 + 近处针叶林
	const TREES_FAR = treeLine([
		[0, 60, 84], [54, 46, 60], [104, 66, 96], [176, 50, 70], [232, 58, 104],
		[300, 44, 66], [352, 64, 90], [430, 48, 112], [486, 56, 74], [548, 62, 98],
		[626, 46, 64], [684, 60, 104], [756, 44, 78], [808, 58, 92], [882, 48, 68],
		[940, 62, 100], [1010, 44, 74], [1062, 58, 96], [1130, 48, 62], [1190, 64, 92],
		[1262, 44, 108], [1318, 60, 78], [1386, 48, 94], [1444, 58, 66], [1514, 62, 100]
	]);
	const TREES_NEAR = treeLine([
		[8, 84, 150], [104, 68, 108], [182, 96, 178], [292, 72, 120], [374, 88, 196],
		[474, 66, 132], [552, 92, 164], [660, 74, 116], [746, 88, 186], [846, 66, 138],
		[924, 96, 158], [1030, 72, 122], [1114, 86, 190], [1214, 68, 140], [1294, 92, 172],
		[1402, 74, 112], [1486, 96, 182]
	]);

	const world = $derived(theme.theme);
</script>

<div class="ambient-backdrop" aria-hidden="true">
	<div class="ambient-stage">
		<div class="ambient-layer ambient-sky"></div>
		{#if world === 'forest'}
			<div class="ambient-sun"></div>
		{/if}
		<div class="ambient-layer ambient-glow ambient-glow--a"></div>
		<div class="ambient-layer ambient-glow ambient-glow--b"></div>
		<div class="ambient-layer ambient-horizon"></div>

		{#if world === 'vibrant'}
			<!-- 星际：星尘 + 星座 + 轨道环 + 城市天际线 -->
			<svg
				class="ambient-world ambient-world--vibrant"
				viewBox="0 0 1600 900"
				preserveAspectRatio="xMidYMid slice"
				focusable="false"
			>
				<g class="ambient-stars" fill="currentColor">
					{#each STARS as star (star.x + '-' + star.y)}
						<circle cx={star.x} cy={star.y} r={star.r} opacity={star.o} />
					{/each}
				</g>
				<g
					class="ambient-constellation"
					fill="none"
					stroke="currentColor"
					stroke-width="1.1"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					{#each CONSTELLATIONS as d}<path {d} />{/each}
				</g>
				<g class="ambient-orbit" fill="none" stroke="currentColor" stroke-width="1">
					<ellipse cx="1290" cy="218" rx="260" ry="112" transform="rotate(-18 1290 218)" />
					<ellipse cx="1290" cy="218" rx="352" ry="158" transform="rotate(-18 1290 218)" opacity="0.55" />
				</g>
				<g class="ambient-scene ambient-scene--far" fill="currentColor">
					<path d={SKYLINE_FAR} />
				</g>
				<g class="ambient-scene ambient-scene--near" fill="currentColor">
					<path d={SKYLINE_NEAR} />
				</g>
			</svg>
		{:else if world === 'forest'}
			<!-- 清野：等高线地形 + 针叶林剪影 -->
			<svg
				class="ambient-world ambient-world--forest"
				viewBox="0 0 1600 900"
				preserveAspectRatio="xMidYMid slice"
				focusable="false"
			>
				<g
					class="ambient-contours"
					fill="none"
					stroke="currentColor"
					stroke-width="1.1"
					stroke-linecap="round"
				>
					{#each CONTOURS as d}<path {d} />{/each}
				</g>
				<g class="ambient-scene ambient-scene--far" fill="currentColor">
					<path d={TREES_FAR} />
				</g>
				<g class="ambient-scene ambient-scene--near" fill="currentColor">
					<path d={TREES_NEAR} />
				</g>
			</svg>
		{:else}
			<!-- 水墨：三段墨山 + 朱砂圆相 + 墨点 -->
			<svg
				class="ambient-world ambient-world--inkwash"
				viewBox="0 0 1600 900"
				preserveAspectRatio="xMidYMax slice"
				focusable="false"
			>
				<path
					class="ambient-scene ambient-scene--far"
					fill="currentColor"
					d="M0 900 L0 636 C120 600 220 660 340 612 S560 560 700 620 S940 572 1080 642 S1360 592 1600 662 L1600 900 Z"
				/>
				<path
					class="ambient-scene ambient-scene--mid"
					fill="currentColor"
					d="M0 900 L0 704 C160 662 260 722 420 682 S660 642 820 702 S1080 652 1240 712 S1460 672 1600 702 L1600 900 Z"
				/>
				<path
					class="ambient-scene ambient-scene--near"
					fill="currentColor"
					d="M0 900 L0 772 C200 742 320 802 520 772 S780 732 980 792 S1240 752 1400 802 S1520 782 1600 802 L1600 900 Z"
				/>
				<circle
					class="ambient-enso"
					fill="none"
					stroke="currentColor"
					stroke-width="9"
					stroke-linecap="round"
					stroke-dasharray="700 220"
					transform="rotate(-28 1290 250)"
					cx="1290"
					cy="250"
					r="146"
				/>
				<g class="ambient-ink-dots" fill="currentColor">
					<circle cx="246" cy="742" r="4" opacity="0.5" />
					<circle cx="284" cy="706" r="2.2" opacity="0.35" />
					<circle cx="196" cy="676" r="2.8" opacity="0.4" />
					<circle cx="1322" cy="812" r="3" opacity="0.3" />
				</g>
			</svg>
		{/if}

		<div class="ambient-layer ambient-vignette"></div>
	</div>
</div>
