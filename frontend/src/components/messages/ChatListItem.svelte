<script lang="ts">
	import { Pin, BellOff, MoreHorizontal } from '@lucide/svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import type { ChatItem } from '@/lib/chat-list';

	// 会话列表项（R13 重排层级；R16 三态重构）：
	//   - 基于公共 ListItem 行组件（hover 填充 / tap 反馈 / 键盘 / aria）
	//   - 三态的非色相信号（灰度截图也能分辨，R16）：
	//       选中 = 左侧竖指示条（on-accent）+ accent 实底 + semibold 昵称
	//       未读 = 左侧短竖指示条（accent）+ 数字徽标 + semibold 昵称 + 行底淡 tint
	//       普通 = 无指示条 + medium 昵称；hover 由 ListItem 统一填充 + 本行文字/头像加强
	//   - selected 行内文字一律 full on-accent（旧实现 preview/time 叠 opacity 0.7/0.8，
	//     实底上对比度掉到 3.97/4.35，不满足 4.5；主次改由字号/字重表达）
	//   - 右缘不再并排堆「日期 / 未读 / ⋯」三种控件：日期固定在首行右端，
	//     未读徽标在次行右端；会话操作（置顶/免打扰/删除）收进**一个**菜单——
	//     桌面 hover/focus 显现 ⋯（desktop-reveal），移动端由列表级长按
	//     /右键触发（见 shared/row-long-press.ts），行上不常驻 ⋯，5 行列表更安静。
	//   - 置顶/免打扰改为昵称旁的静态身份标识（不再是行内按钮），不占右缘列。

	let {
		chat,
		isSelected = false,
		onClick,
		onMore,
		compact = false
	}: {
		chat: ChatItem;
		isSelected?: boolean;
		onClick: () => void;
		/** 打开会话操作菜单（置顶/免打扰/删除）；缺省不渲染入口 */
		onMore?: () => void;
		compact?: boolean;
	} = $props();

	const isGroup = $derived(chat.kind === 'group');
	const hasUnread = $derived(chat.unread > 0);
	const nameCls = $derived(
		isSelected ? 'text-text-on-accent' : 'text-text-primary group-hover:text-accent-default'
	);
	const previewCls = $derived(
		isSelected
			? 'text-text-on-accent'
			: hasUnread
				? 'text-text-primary'
				: 'text-text-secondary'
	);
	const metaCls = $derived(isSelected ? 'text-text-on-accent' : 'text-text-secondary');
	// 左侧指示条：选中=长条 on-accent；未读=短条 accent（与徽标同色，形状+色相同步）
	const indicatorCls = $derived(
		isSelected ? 'h-8 bg-text-on-accent' : 'h-6 bg-accent-default'
	);
</script>

<ListItem
	pressed={isSelected}
	onClick={onClick}
	ariaLabel={`打开与 ${chat.name} 的会话`}
	class="group flex select-none items-center gap-3 {compact ? 'px-3.5 py-2.5' : 'px-4 py-3'} {!isSelected && hasUnread ? 'bg-accent-soft' : ''}"
>
	{#if isSelected || hasUnread}
		<!-- 非色相态信号：选中/未读在行首各有一根竖指示条（灰度下仍靠「有/无条 + 长度」区分） -->
		<span
			aria-hidden="true"
			data-chat-indicator={isSelected ? 'selected' : 'unread'}
			class="absolute left-1 top-1/2 w-1 -translate-y-1/2 rounded-full {indicatorCls}"
		></span>
	{/if}
	<span class="relative shrink-0">
		<Avatar
			src={chat.avatar}
			name={chat.name}
			size={compact ? 40 : 44}
			shape={isGroup ? 'rounded' : 'circle'}
			decorative
			style={isSelected ? 'box-shadow:0 0 0 var(--cst-border-width-regular) var(--cst-border-inverse);' : ''}
		/>
		{#if chat.online}
			<OnlineDot size="md" />
		{/if}
	</span>

	<span class="min-w-0 flex-1 flex flex-col justify-center gap-1">
		<!-- 首行：昵称（+ 置顶/免打扰静态标识） / 日期 -->
		<span class="flex items-center justify-between gap-2">
			<span class="flex min-w-0 items-center gap-1.5">
				<span class="truncate text-md {hasUnread || isSelected ? 'font-semibold' : 'font-medium'} {nameCls}">
					{chat.name}
				</span>
				{#if chat.isPinned}
					<Pin
						class="size-3 shrink-0 {isSelected ? 'text-text-on-accent opacity-emphasis' : 'text-accent-default'}"
						strokeWidth={2}
						aria-hidden="true"
					/>
				{/if}
				{#if chat.isMuted}
					<BellOff
						class="size-3 shrink-0 {isSelected ? 'text-text-on-accent opacity-emphasis' : 'text-text-secondary'}"
						strokeWidth={2}
						aria-hidden="true"
					/>
				{/if}
			</span>
			<span class="shrink-0 text-xs tabular-nums {metaCls}">
				{chat.time}
			</span>
		</span>
		<!-- 次行：最后一条消息 / 未读徽标（+ 桌面 hover 显现的单一操作入口） -->
		<span class="flex items-center justify-between gap-2">
			<span class="truncate text-sm {previewCls}">
				{chat.lastMessage || '暂无消息'}
			</span>
			<span class="flex shrink-0 items-center gap-1">
				{#if hasUnread}
					<!-- 未读徽标：紧凑型 h-5 px-2 + text-xxs（AGENTS 徽标契约）；选中态反色 -->
					<span
						class="inline-flex h-5 min-w-5 items-center justify-center rounded-full px-2 text-xxs font-semibold leading-none tabular-nums {isSelected
							? 'bg-text-on-accent text-accent-default'
							: 'bg-accent-default text-text-on-accent'}"
					>
						{chat.unread}
					</span>
				{/if}
				{#if onMore}
					<!-- 唯一操作入口：桌面 hover/focus 显现；移动端走行级长按（不常驻） -->
					<button
						type="button"
						class="hover-btn scale-btn hit-slop-44 desktop-reveal -my-1 -mr-1 hidden shrink-0 items-center justify-center rounded-full p-1.5 text-text-secondary focus-ring md:flex"
						aria-label={`${chat.name} 的更多操作`}
						title="更多操作"
						onclick={(e) => {
							e.stopPropagation();
							onMore();
						}}
					>
						<MoreHorizontal class="size-4" strokeWidth={2} />
					</button>
				{/if}
			</span>
		</span>
	</span>
</ListItem>
