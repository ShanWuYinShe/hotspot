<script lang="ts">
	import { MessageCircle, Plus } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import Button from '@/components/ui/button.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import type { ChatItem } from '@/lib/chat-list';

	// ChatEmptyPanel —— 桌面/宽平板分栏右栏的「未选会话」引导面板（R13）。
	// 取代旧实现（页面级 EmptyState + spotlight 装饰塞进窄面板）：大光环在窄面板里
	// 会被裁成断弧、且面板 70% 面积是空白。改为「引导 + 最近会话建议」：
	//   - 最近会话从页面已拉到的会话列表取前 3 条（纯前端，不新增 API）；
	//   - 点击建议行 = 选中该会话（与左栏行同一 selectChat 行为）；
	//   - 底部保留「发起新会话」入口，空面板动线闭环；
	//   - 小面板按空态规范不开 spotlight。

	let {
		recent,
		onSelect,
		onNewChat
	}: {
		recent: ChatItem[];
		onSelect: (chat: ChatItem) => void;
		onNewChat: () => void;
	} = $props();

	const suggestions = $derived(recent.slice(0, 3));
</script>

<div class="flex h-full items-center justify-center overflow-y-auto px-6 py-10 no-scrollbar">
	<div class="w-full max-w-sm">
		<div class="flex flex-col items-center text-center">
			<!-- 图标砖：软底圆形（空态语言，72 档在整页空态用；面板内收敛到 64 避免喧宾夺主） -->
			<div class="flex size-16 items-center justify-center rounded-full bg-accent-soft text-text-accent-on-soft">
				<MessageCircle class="size-7" strokeWidth={1.5} aria-hidden="true" />
			</div>
			<h2 class="mt-5 text-base font-semibold text-text-primary">选择一个会话</h2>
			<p class="mt-2 max-w-xs text-sm leading-relaxed text-text-secondary">
				从左侧选择会话继续聊天，或从最近会话快速进入。
			</p>
		</div>

		{#if suggestions.length > 0}
			<div class="mt-8">
				<SectionHead variant="eyebrow" title="最近会话" />
				<ListContainer bordered ariaLabel="最近会话建议">
					{#each suggestions as chat (chat.id)}
						<ListItem
							onClick={() => onSelect(chat)}
							ariaLabel={`打开与 ${chat.name} 的会话`}
							class="flex items-center gap-3 px-3 py-2.5"
						>
							<Avatar
								src={chat.avatar}
								name={chat.name}
								size={36}
								shape={chat.kind === 'group' ? 'rounded' : 'circle'}
								decorative
							/>
							<span class="min-w-0 flex-1">
								<span class="flex items-center justify-between gap-2">
									<span class="truncate text-md font-medium text-text-primary">{chat.name}</span>
									<span class="shrink-0 text-xs tabular-nums text-text-secondary">{chat.time}</span>
								</span>
								<span class="mt-0.5 block truncate text-sm text-text-secondary">
									{chat.lastMessage || '暂无消息'}
								</span>
							</span>
						</ListItem>
					{/each}
				</ListContainer>
			</div>
		{/if}

		<div class="mt-6 flex justify-center">
			<Button type="button" variant="soft" size="md" onclick={onNewChat}>
				<Plus class="size-4" strokeWidth={1.75} aria-hidden="true" />
				发起新会话
			</Button>
		</div>
	</div>
</div>
