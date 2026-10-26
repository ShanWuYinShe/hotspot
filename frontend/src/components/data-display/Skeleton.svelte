<script lang="ts">
	import { cn } from '@/lib/utils';

	// Skeleton — 骨架屏占位块（唯一基础原语，纯装饰）。
	// 与鸿蒙 components/states/Skeleton 同口径：底色 --cst-surface-subtle
	// （鸿蒙 surfaceSubtle）+ 透明度脉冲（鸿蒙 SkeletonShimmer 1↔0.58，web 归并到
	// 最近 opacity 档 dim=0.60）。尺寸/形状一律由调用方用档位类提供
	// （size-* / aspect-* / rounded-*），因此桌面端只改栅格列数、移动端与鸿蒙
	// 保持同一几何节奏（行高/头像直径/文本条高度与宽度比例）。
	// 语义由外层 SkeletonList/SkeletonFeed 的 role=status + aria-busy 承担，
	// 本组件恒 aria-hidden，不进入读屏与焦点序列；减弱动效由 utilities.css 的
	// prefers-reduced-motion 全局规则瞬时化。
	let {
		class: className = '',
		style = ''
	}: {
		/** 占位块几何（size-* / h-* / aspect-* / rounded-* 档位）。 */
		class?: string;
		/** 布局几何例外：文本条高度/百分比宽度等需像素级对齐鸿蒙骨架时用
		 *  （非主题风格，AGENTS.md UI 原则 #4 已登记例外）。 */
		style?: string;
	} = $props();
</script>

<div data-skeleton aria-hidden="true" class={cn('skeleton-block', className)} {style}></div>
