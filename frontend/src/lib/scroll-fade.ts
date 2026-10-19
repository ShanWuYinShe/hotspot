// 横向滚动「还有内容」渐隐提示 —— 全站唯一实现（Svelte action）。
//
// 背景（全站复审 C1）：移动端后台 10 页的模块导航在 390px 下溢出 328px、表格溢出
// 27–217px，而容器多为 `no-scrollbar`（滚动条本身也被隐掉），用户没有任何「右侧还有
// 内容」的线索，被裁的内容看起来像渲染残缺。
//
// 机制：只在**真正溢出的一侧**加 `.scroll-fade-left/right`（见 styles/theme/utilities.css），
// 用 mask 做几何裁剪。之所以不是「常驻渐隐」：到头了还渐隐会变成假提示。
// 之所以用 mask 而不是经典 CSS scroll-shadow：表格表头（TH）自带不透明底色，会把
// 容器背景上的阴影盖住；mask 作用在最终渲染结果上，与容器内元素有无底色无关。
//
// 用法：`<div class="overflow-x-auto" use:scrollFade>…</div>`
export function scrollFade(node: HTMLElement) {
	const sync = () => {
		const max = node.scrollWidth - node.clientWidth;
		// 2px 容差：亚像素/取整误差不应当成「还能滚」
		node.classList.toggle('scroll-fade-left', node.scrollLeft > 2);
		node.classList.toggle('scroll-fade-right', max - node.scrollLeft > 2);
	};

	sync();
	node.addEventListener('scroll', sync, { passive: true });
	// ResizeObserver 在 jsdom 下不存在（组件测试环境）
	//
	// 观察**两侧**：
	// - 容器自身尺寸变化（视口旋转、栅格列数变化）；
	// - **子内容变化**（MutationObserver）——这是实测踩过的坑：标签/筛选胶囊多为异步
	//   拉取后填充，内容变长时**容器尺寸不变**，只观察容器不会触发 → 溢出却无提示
	//   （全站复审收尾回归在 /nearby/moments 移动端抓到：溢出 46px、提示缺失）。
	let ro: ResizeObserver | undefined;
	if (typeof ResizeObserver !== 'undefined') {
		ro = new ResizeObserver(sync);
		ro.observe(node);
	}
	let mo: MutationObserver | undefined;
	if (typeof MutationObserver !== 'undefined') {
		mo = new MutationObserver(sync);
		mo.observe(node, { childList: true, subtree: true, characterData: true });
	}

	return {
		update() {
			sync();
		},
		destroy() {
			node.removeEventListener('scroll', sync);
			ro?.disconnect();
			mo?.disconnect();
		}
	};
}
