<script lang="ts">
	import { goto } from '$app/navigation';
	import Button from '@/components/ui/button.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import { avatarImageWidths, buildImageSrcSet, fadeImage, optimizedImageUrl } from '@/lib/images';
	import { TYPE_CONFIG, type NotifyViewItem } from './notify-data';

	// 通知行（对齐 Figma Make NotifyPage 行契约：ListItem
	// hover 背景填充 + tap opacity 0.84，无 scale 位移）。
	// R13：
	//   - 未读标识由「右侧呼吸光点」改为**左侧 3px 未读条**（静态、不抢注意力，
	//     与列表行的扫读方向一致）；「未读」语义保留在 aria-label + sr-only 文本；
	//   - 缩略图与头像统一 44px（同一行两个媒体单元不再一大一小）；
	//   - 接受/拒绝按钮统一行内关系 chip 档（主=accent 实底 / 次=软填），两个按钮
	//     都带 ≥44 触控热区。

	let {
		notif,
		actionPending = false,
		onRead,
		onRemove,
		onAccept,
		onReject
	}: {
		notif: NotifyViewItem;
		/** 邀请接受/拒绝请求进行中（按钮禁点防重复提交） */
		actionPending?: boolean;
		onRead: (id: string) => void;
		onRemove: (id: string) => void;
		/** 入群邀请通知的可选操作（inviteId 存在时渲染接受/拒绝按钮）。 */
		onAccept?: (inviteId: string) => void;
		onReject?: (inviteId: string) => void;
	} = $props();

	const cfg = $derived(TYPE_CONFIG[notif.type] ?? TYPE_CONFIG.reaction);
</script>

<ListItem
	onClick={() => {
		onRead(notif.id);
		// 点击跳转（payload 上下文推导的 href；行内接受/拒绝按钮已 stopPropagation）
		if (notif.href) void goto(notif.href);
	}}
	ariaLabel={`${notif.read ? '' : '未读，'}通知：${notif.user.name}${notif.text}`}
	class="relative flex items-center gap-x-3.5 px-4 py-3.5 transition-colors"
>
	<!-- 未读条：左侧 3px accent 竖条（状态条 1-3px 属已登记例外，不用圆角外框） -->
	{#if !notif.read}
		<span aria-hidden="true" class="absolute left-0 top-2.5 bottom-2.5 w-0.75 rounded-full bg-accent-default"></span>
	{/if}

	<!-- Avatar + icon badge -->
	<div class="relative shrink-0">
		{#if notif.user.avatar}
			<!-- 头像统一 shared/Avatar（占位/兜底/渐显内置）；类型角标外挂：
			     Avatar 包装是 overflow-hidden，18px 角标放内部会被圆角裁切 -->
			<Avatar src={notif.user.avatar} name={notif.user.name} size="lg" class="ring-1 ring-border-subtle/50" />
			<div
				class="absolute -bottom-0.5 -right-0.5 flex size-4.5 items-center justify-center rounded-full shadow-xs"
				style="background:{cfg.color};border:var(--cst-border-width-strong) solid var(--cst-surface-primary);"
			>
				<cfg.icon class="size-2.5 text-text-on-accent" strokeWidth={2.5} />
			</div>
		{:else}
			<div class="flex size-11 items-center justify-center rounded-full" style="background:{cfg.bg};">
				<cfg.icon class="size-4.5" style="color:{cfg.color};" strokeWidth={1.75} />
			</div>
		{/if}
	</div>

	<!-- Text -->
	<div class="min-w-0 flex-1">
		<p class="truncate text-text-primary text-sm">
			<span class={notif.read ? 'font-medium text-text-secondary' : 'font-semibold text-text-primary'}>{notif.user.name}</span>
			{' '}{notif.text}
		</p>
		<p class="mt-0.5 text-xs text-text-secondary">{notif.time}</p>
	</div>

	<!-- 动作组（入群邀请）/ 瞬间缩略图 / 未读语义 -->
	<div class="flex shrink-0 items-center gap-2.5">
		{#if notif.inviteId}
			<!-- 入群邀请操作：主=accent 实底 / 次=soft 软填（同档视觉重量，不再一实一虚）；
			     移动端两钮在行尾同一列竖排（不折到第二行悬空），sm+ 回到单行 -->
			<div class="flex shrink-0 flex-col items-stretch gap-1.5 sm:flex-row sm:items-center">
				<Button
					type="button"
					size="sm"
					variant="primary"
					shape="pill"
					class="hit-slop-44 px-3.5"
					disabled={actionPending}
					onclick={(e) => { e.stopPropagation(); onAccept?.(notif.inviteId!); }}
				>接受</Button>
				<Button
					type="button"
					size="sm"
					variant="soft"
					shape="pill"
					class="hit-slop-44 px-3.5"
					disabled={actionPending}
					onclick={(e) => { e.stopPropagation(); onReject?.(notif.inviteId!); }}
				>拒绝</Button>
			</div>
		{/if}
		{#if notif.momentImage}
			<!-- 缩略图：媒体容器去 1px 描边（边界由圆角 + 占位渐变 + 轻投影表达） -->
			<div class="media-placeholder size-11 overflow-hidden rounded-md shadow-xs">
				<img
					src={optimizedImageUrl(notif.momentImage, { width: 88 })}
					srcset={buildImageSrcSet(notif.momentImage, avatarImageWidths(44))}
					sizes="44px"
					alt=""
					loading="lazy"
					decoding="async"
					class="h-full w-full object-cover"
					use:fadeImage
				/>
			</div>
		{/if}
		{#if !notif.read}
			<span class="sr-only">未读</span>
		{/if}
	</div>
</ListItem>
