<script lang="ts">
	import { ArrowDown, MessageCircle } from '@lucide/svelte';
	import { fly } from 'svelte/transition';
	import { contentFade, motionDuration } from '@/lib/motion';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import ChatColumn from '@/components/messages/ChatColumn.svelte';
	import MessageBubble from '@/components/messages/MessageBubble.svelte';
	import ChatDatePill from '@/components/messages/ChatDatePill.svelte';
	import { inRecallWindow, type ChatController, type ChatMessage } from '@/lib/chat.svelte';

	// ChatMessageList —— 私聊/群聊共享的消息区（加载/错误/空态 + 日期分组气泡）。
	// 视图差异仅 canRecall 判定（群聊管理员可撤回他人消息），由 props 注入。

	// 消息列宽度与页头/输入栏同轴：统一由 ChatColumn 提供（禁止本文件再写 max-w-*）。

	let {
		chat,
		canRecall,
		showSenderNames = false,
		onReport,
		errorText = '消息加载失败，请检查网络后重试'
	}: {
		chat: ChatController;
		canRecall: (msg: ChatMessage) => boolean;
		/** 群聊= true：incoming 气泡上方显示发送者昵称 */
		showSenderNames?: boolean;
		/** 收到消息的举报入口（reports/create target_type='message'）；缺省不渲染 */
		onReport?: (messageId: string) => void;
		/** 加载失败文案；群聊可注入「未加入该群」等权限语义（默认网络失败） */
		errorText?: string;
	} = $props();

	// 「没有更多了」只在真正有意义时出现（R16 降噪）：
	// 首屏一次就返回全部消息（next_cursor=null）说明这个会话从未分页，顶部再写一句
	// 「没有更多了」只是噪声（用户实测「很怪」）；只有历史确实翻过页、翻到耗尽后，
	// 收尾提示才有信息量。切换会话（loading / messages 清空）时重置。
	let everPaginated = $state(false);
	$effect(() => {
		if (chat.loading || chat.messages.length === 0) everPaginated = false;
		else if (chat.hasMore) everPaginated = true;
	});

	// 渲染守卫（审计 P1-3）：后端/历史数据里存在 content 为空串的消息，直接渲染会得到一个
	// 只剩时间戳的「空气泡」，看起来像渲染失败。保留撤回消息（渲染为「消息已撤回」），
	// 其余 text 为空白的不生成气泡；空出来的日期分节一并收起，避免出现只有日期标签的孤儿分节。
	// 只影响渲染层：分页/未读/已读回执仍由 chat controller 维护。
	const visibleGroups = $derived(
		chat.dateGroups
			.map((group) => ({
				...group,
				messages: group.messages.filter((msg) => msg.recalled || (msg.text ?? '').trim().length > 0)
			}))
			.filter((group) => group.messages.length > 0)
	);

	// 入场语义（P1-2）：整列/每条气泡的静态 animate-fade-in 会让人一进会话就播 N 条淡入
	// （30 条历史 = 30 个动画），且历史上不该有入场。这里只给「实时到达」的那一条播一次
	// enterOnce：首屏/切会话的历史消息整体登记、不播；会话切换由 chat.loading 的
	// true→false 批次识别（加载中重置基线）。reduced-motion 由 enterOnce 内部跳过。
	let knownIds = new Set<string>();
	let initDone = false;
	let wasLoading = false;
	let freshId = $state('');
	$effect(() => {
		const loading = chat.loading;
		const ids = visibleGroups.flatMap((group) => group.messages.map((msg) => msg.id));
		if (!initDone) {
			if (loading) return; // 首屏加载中：等历史到达后整体登记
			knownIds = new Set(ids);
			initDone = true;
			wasLoading = false;
			return;
		}
		if (loading) {
			knownIds = new Set(ids);
			wasLoading = true;
			freshId = '';
			return;
		}
		if (wasLoading) {
			knownIds = new Set(ids);
			wasLoading = false;
			freshId = '';
			return;
		}
		const added = ids.filter((id) => !knownIds.has(id));
		knownIds = new Set(ids);
		if (added.length > 0) freshId = added[added.length - 1];
	});
</script>

<ChatColumn class="relative flex min-h-0 flex-1 flex-col">
	<!-- 消息区是独立滚动容器（Round 10：分页 prepend 保视口） -->
	<!-- 容器保留唯一一条入场淡入（contentFade：仅「显示过加载态 → 内容到达」时播 110ms，
	     首屏缓存命中/无加载态不播）；气泡自身不再单独入场。 -->
	<div
		bind:this={chat.scrollEl}
		onscroll={chat.handleScroll}
		use:contentFade={chat.loading}
		class="min-h-0 flex-1 overflow-y-auto no-scrollbar"
		style="overscroll-behavior:contain;"
	>
		{#if chat.loading}
			<!-- 加载态统一 InlineLoading（全站三态标配，不再自绘 spinner） -->
			<InlineLoading label="加载消息…" class="h-full justify-center" />
		{:else if chat.loadError}
			<!-- 错误态统一 LoadError（与全站一致），不再各页手搓文案 + 按钮 -->
			<LoadError message={errorText} onRetry={() => void chat.reload()} class="h-full" />
		{:else if visibleGroups.length === 0}
			<!-- 空会话（含「全部消息内容为空」的脏数据场景）：统一 EmptyState 空态 -->
			<EmptyState class="h-full" title="还没有消息" description="打个招呼，开始这段对话吧" bare>
				{#snippet icon()}
					<!-- bare 空态裸图标统一 28px（style skill 已登记口径） -->
					<MessageCircle class="size-7 text-text-secondary" strokeWidth={1.5} aria-hidden="true" />
				{/snippet}
			</EmptyState>
		{:else}
			<!-- 短会话底对齐（R16）：消息贴住输入条、空白留在上方（Telegram/iMessage/WhatsApp 同款），
			     消除「消息挤在顶部、下方一大片空场」的断裂感；内容超过一屏时 min-h-full 自然失效、正常滚动。
			     上方 pt-3 / 下方 pb-4 是滚动到顶/底时的呼吸余量。 -->
			<div class="flex min-h-full flex-col justify-end gap-2 px-4 pb-4 pt-3">
				{#if chat.loadingMoreFailed}
					<!-- 分页失败可见化（layout 第 6 条）：轻量重试入口，不遮蔽已加载消息 -->
					<div class="flex items-center justify-center gap-2 py-3 text-sm text-text-secondary">
						<span>更早的消息加载失败</span>
						<button type="button" class="link-btn" onclick={() => void chat.loadMore()}>重试</button>
					</div>
				{:else if chat.loadingMore}
					<InlineLoading label="加载更早消息…" class="py-3" />
				{:else if !chat.hasMore && everPaginated}
					<!-- 历史耗尽：ListEnd 收尾（layout 第 6 条），区分「已到顶」与「卡住」 -->
					<ListEnd />
				{/if}
				{#each visibleGroups as group, gi (group.label)}
					<!-- 每个日期分节：中性日期标签 + 气泡组；分节之间比组内多一档留白（12+8） -->
					<div class="flex flex-col gap-2 {gi > 0 ? 'mt-2' : ''}">
						<ChatDatePill label={group.label} />
						{#each group.messages as msg (msg.id)}
							<MessageBubble
								text={msg.text}
								sent={msg.sent}
								time={msg.time}
								senderName={showSenderNames ? msg.senderName : ''}
								showCheck={msg.id === chat.lastSentId}
								messageId={msg.id}
								enter={msg.id === freshId}
								recalled={msg.recalled}
								editedAt={msg.editedAt}
								canRecall={canRecall(msg)}
								canEdit={msg.sent}
								canReport={!msg.sent && Boolean(onReport)}
								onRecall={(id) => void chat.handleRecall(id)}
								onEdit={(id, content) => void chat.handleEdit(id, content)}
								onReport={(id) => onReport?.(id)}
							/>
						{/each}
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<!-- 回到底部（Round 10）：上滑超过 200px 才出现；无新消息时只有箭头，
	     有新消息时显示「N 条新消息」。点击滚底 + 清计数 + 恢复自动滚动。 -->
	{#if chat.showScrollDown && !chat.loading}
		<button
			type="button"
			aria-label={chat.newMessageCount > 0
				? `回到底部，${chat.newMessageCount > 99 ? '99+' : chat.newMessageCount} 条新消息`
				: '回到底部'}
			title="回到底部"
			onclick={() => chat.scrollToBottom()}
			in:fly={{ y: 8, duration: motionDuration(150) }}
			out:fly={{ y: 8, duration: motionDuration(150) }}
			class="absolute right-14 bottom-3 z-[var(--cst-z-sticky)] flex h-11 min-w-11 items-center justify-center gap-1.5 rounded-full border border-border-default bg-surface-primary/95 px-3 text-xs font-medium text-text-primary shadow-raised-md backdrop-blur-sm focus-ring"
		>
			<ArrowDown class="size-4 shrink-0" strokeWidth={2} />
			{#if chat.newMessageCount > 0}
				<span aria-live="polite">{chat.newMessageCount > 99 ? '99+' : chat.newMessageCount} 条新消息</span>
			{/if}
		</button>
	{/if}
</ChatColumn>
