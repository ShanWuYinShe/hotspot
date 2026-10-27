<script lang="ts">
	import { ArrowUp } from '@lucide/svelte';
	import { fade } from 'svelte/transition';
	import { motionDuration, prefersReducedMotion } from '@/lib/motion';

	// 回到顶部（Round 9）：滚动超过约 1.5 屏后出现在右下角（44px 触控），
	// 点击平滑回顶（reduced 直接跳），回到顶部后隐藏；键盘可达；固定定位不占文档流。
	let { scrollEl }: { scrollEl: () => HTMLElement | null } = $props();

	let visible = $state(false);
	// 点击回顶后 200ms 内保持隐藏，避免平滑滚动过程中回闪（reduced 直接跳同样适用）
	let hiding = false;

	$effect(() => {
		const el = scrollEl();
		if (!el) return;
		let raf = 0;
		const update = () => {
			raf = 0;
			if (hiding) return;
			visible = el.scrollTop > window.innerHeight * 1.5;
		};
		const onScroll = () => {
			if (!raf) raf = requestAnimationFrame(update);
		};
		el.addEventListener('scroll', onScroll, { passive: true });
		window.addEventListener('resize', onScroll, { passive: true });
		update();
		return () => {
			el.removeEventListener('scroll', onScroll);
			window.removeEventListener('resize', onScroll);
			if (raf) cancelAnimationFrame(raf);
		};
	});

	function backToTop() {
		const el = scrollEl();
		if (!el) return;
		hiding = true;
		visible = false; // 点击后立即隐藏（≤200ms 内不再回闪）
		el.scrollTo({ top: 0, behavior: prefersReducedMotion() ? 'auto' : 'smooth' });
		window.setTimeout(() => {
			hiding = false;
		}, 220);
	}
</script>

{#if visible}
	<button
		type="button"
		aria-label="回到顶部"
		title="回到顶部"
		onclick={backToTop}
		in:fade={{ duration: motionDuration(150) }}
		out:fade={{ duration: motionDuration(150) }}
		class="hover-btn focus-ring fixed right-4 bottom-24 z-[var(--cst-z-sticky)] flex size-11 items-center justify-center rounded-full border border-border-default bg-surface-primary/95 text-text-primary shadow-raised-md backdrop-blur-sm sm:bottom-6"
	>
		<ArrowUp class="size-5" strokeWidth={2} />
	</button>
{/if}
