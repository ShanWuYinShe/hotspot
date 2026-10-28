<script lang="ts">
	import { Heart, MessageCircle, Bookmark, Share2 } from '@lucide/svelte';
	import { popOn } from '@/lib/motion';
	import { cn } from '@/lib/utils';

	// ImageCardActions — 图片内容卡「压在图片上的操作行」唯一实现（R14 卡片交互契约）。
	//
	// 首页信息流大卡（HomeFeedItem）、网格卡（MomentCard）、我的瞬间卡（ProfileMomentTile）
	// 等所有图片卡共用本组件，保证：
	// - 同一组图标尺寸（size-4.5 / strokeWidth 1.5）、间距（gap-6）、字号（text-sm tabular），
	//   不再出现「信息流卡 11px、网格卡 12px、有的卡不是按钮只能打开详情」的碎片化；
	// - 同一套 hover（hover-action 放大）/press（按压缩小）/focus（focus-ring）/计数弹跳
	//   （count-pop，仅激活方向）；
	// - 同一套状态语义（全站唯一口径）：点赞 = 危险红填充（压图态白色描边），
	//   收藏 = accent 填充，评论/分享 = 中性；填充形状 + 颜色双信号，不只靠色相。
	//
	// onImage：压在图片遮罩上（白字 + 白描边）；false = 无图 fallback/实底（主题色）。
	// likes/saved 传入的是「服务端基数」，组件按 liked 自行 +1 显示（调用方不要重复加）。

	let {
		likes,
		comments,
		shares = null,
		liked,
		saved,
		onImage = true,
		onLike,
		onComment,
		onSave,
		onShare,
		placeholderLabels = true,
		size = 'md',
		class: className = ''
	}: {
		/** 点赞数（服务端基数；组件按 liked 状态 +1 展示） */
		likes: number;
		comments: number;
		/** 分享数；null/0 时不显示数字（仅图标） */
		shares?: number | null;
		liked: boolean;
		saved: boolean;
		/** 压在图片上（白字反色）；false 为实底/无图回退（主题色） */
		onImage?: boolean;
		onLike: (next: boolean) => void;
		onComment: () => void;
		onSave: (next: boolean) => void;
		/** 分享：所有卡片都传（由 lib/moment-actions 的 shareMoment 实现） */
		onShare?: () => void;
		/** 计数为 0 时显示「赞/评论」文案（大卡友好）；false 时显示 0 */
		placeholderLabels?: boolean;
		/** 档位：md = 大卡（信息流/桌面瓦片）；sm = 手机 2 列窄卡（操作行必须四类操作都放下） */
		size?: 'md' | 'sm';
		class?: string;
	} = $props();

	// 点击驱动的弹跳：只在激活方向播（取消点赞不弹），reduced-motion 由 popOn 内部跳过。
	let likePop = $state(0);
	let savePop = $state(0);

	// 当前展示计数：点赞基数 + 本次乐观 +1（服务端回写后调用方会更新基数，视觉不跳）。
	const displayLikes = $derived(likes + (liked ? 1 : 0));

	// min-w-4.5：纯图标按钮（分享/收藏，sm 档图标 16px）视觉宽 16 时 hit-slop-44 的有效热区
	// 只有 42px（16+26）低于 44 下限；补到 18px 后 = 44px，触控门禁实测通过。
	const btnBase =
		'hover-action hit-slop-44 -my-2 flex min-w-4.5 shrink-0 items-center gap-1.5 py-2 focus-ring transition-colors';
	// 档位只影响间距/图标/字号（交互反馈完全一致）：sm 档 2 列窄卡四类操作仍不溢出。
	const groupGap = $derived(size === 'sm' ? 'gap-3' : 'gap-6');
	const iconSize = $derived(size === 'sm' ? 'size-4' : 'size-4.5');
	const labelSize = $derived(size === 'sm' ? 'text-xs' : 'text-sm');
	// 压图态：所有操作内容保持满亮度反色（对比度优先，不再叠 opacity-hover）；
	// 实底态：中性次级色，hover 提到正文色。
	const idleCls = $derived(onImage ? 'text-content-inverse' : 'text-text-secondary hover:text-text-primary');
	const activeLikeCls = $derived(onImage ? 'text-content-inverse' : 'text-status-danger');
	const activeSaveCls = $derived(onImage ? 'text-content-inverse' : 'text-accent-default');
</script>

<div
	class={cn('flex items-center justify-between gap-3', onImage ? 'text-content-inverse' : 'text-text-secondary', className)}
>
	<div class="flex shrink-0 items-center {groupGap}">
		<!-- 点赞：填充形状 + 颜色双信号（压图：红填充白描边；实底：红填充） -->
		<button
			type="button"
			class="{btnBase} {liked ? activeLikeCls : idleCls}"
			aria-label={liked ? '取消点赞' : '点赞'}
			aria-pressed={liked}
			onclick={(e) => {
				e.stopPropagation();
				if (!liked) likePop += 1;
				onLike(!liked);
			}}
		>
			<span class="inline-flex" use:popOn={likePop}>
				<Heart
					class="{iconSize} shrink-0 {liked
						? onImage
							? 'fill-status-danger text-content-inverse'
							: 'fill-status-danger text-status-danger'
						: ''}"
					strokeWidth={1.5}
					fill={!onImage && liked ? 'currentColor' : 'none'}
				/>
			</span>
			{#key displayLikes}
				<span class="count-pop {labelSize} font-medium tabular-nums">
					{displayLikes > 0 || !placeholderLabels ? displayLikes : '赞'}
				</span>
			{/key}
		</button>

		<!-- 评论：跳评论区 -->
		<button
			type="button"
			class="{btnBase} {idleCls}"
			aria-label="查看评论"
			onclick={(e) => {
				e.stopPropagation();
				onComment();
			}}
		>
			<MessageCircle class="{iconSize} shrink-0" strokeWidth={1.5} />
			<span class="{labelSize} font-medium tabular-nums">{comments > 0 || !placeholderLabels ? comments : '评论'}</span>
		</button>
	</div>

	<div class="flex shrink-0 items-center {groupGap}">
		{#if onShare}
			<button
				type="button"
				class="{btnBase} {idleCls}"
				aria-label="分享瞬间"
				onclick={(e) => {
					e.stopPropagation();
					onShare?.();
				}}
			>
				<Share2 class="{iconSize} shrink-0" strokeWidth={1.5} />
				{#if shares && shares > 0}
					<span class="text-sm font-medium tabular-nums">{shares}</span>
				{/if}
			</button>
		{/if}

		<!-- 收藏：accent 填充（压图：accent 填充白描边；实底：accent 填充） -->
		<button
			type="button"
			class="{btnBase} {saved ? activeSaveCls : idleCls}"
			aria-label={saved ? '取消收藏' : '收藏'}
			aria-pressed={saved}
			onclick={(e) => {
				e.stopPropagation();
				if (!saved) savePop += 1;
				onSave(!saved);
			}}
		>
			<span class="inline-flex" use:popOn={savePop}>
				<Bookmark
					class="{iconSize} shrink-0 {saved
						? onImage
							? 'fill-accent-default text-content-inverse'
							: 'fill-accent-default text-accent-default'
						: ''}"
					strokeWidth={1.5}
					fill={!onImage && saved ? 'currentColor' : 'none'}
				/>
			</span>
		</button>
	</div>
</div>