<script lang="ts">
	import { cn } from '@/lib/utils';
	import imgHotSpot from '@/assets/hotspot-logo.png';

	// BrandLockup — 品牌字标（Logo + HotSpot 字名）唯一实现。
	// R13 登录/注册、404、分享落地页统一用同一组标识，避免三处各拼一份大小/字重漂移。
	// 说明：字名已是文本，Logo 图对读屏是纯装饰（alt="" + aria-hidden），不重复播报品牌名。
	// 尺寸档：sm（移动页头 28）/ md（桌面左栏 36）/ lg（超大留白页 44）。

	let {
		size = 'md',
		class: className = ''
	}: {
		size?: 'sm' | 'md' | 'lg';
		class?: string;
	} = $props();

	const LOGO_CLASS: Record<'sm' | 'md' | 'lg', string> = {
		sm: 'size-7',
		md: 'size-9',
		lg: 'size-11'
	};
	const TEXT_CLASS: Record<'sm' | 'md' | 'lg', string> = {
		sm: 'text-lg',
		md: 'text-xl',
		lg: 'text-2xl'
	};
</script>

<span class={cn('inline-flex select-none items-center gap-2.5', className)}>
	<img
		src={imgHotSpot}
		alt=""
		aria-hidden="true"
		decoding="async"
		class={cn('shrink-0 rounded-full object-cover shadow-raised-sm', LOGO_CLASS[size])}
	/>
	<span class={cn('font-semibold tracking-title text-text-primary', TEXT_CLASS[size])}>HotSpot</span>
</span>
