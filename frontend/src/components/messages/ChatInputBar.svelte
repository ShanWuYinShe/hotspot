<script lang="ts">
	import { Send } from '@lucide/svelte';
	import ChatColumn from '@/components/messages/ChatColumn.svelte';
	import type { ChatController } from '@/lib/chat.svelte';

	// ChatInputBar —— 私聊/群聊共享的输入栏（发送错误提示 + 输入框 + 发送钮）。
	// @ 提及（全站复审 s2-08）：仅群聊传入 mentionCandidates 时启用——输入 `@` 触发
	// 成员选择浮层，选中后插入 `@昵称 ` 并在发送时把对应 user_id 透传给
	// group_messages/send 的 mentions（spec api_spec_chat.md:135，最多 20 个）。
	// 私聊不传该 prop → 行为与之前完全一致。
	//
	// R16 内容轴：输入条整条铺满面板宽度（发丝线也是功能线，整条横贯），内部内容与
	// 消息列共用 ChatColumn —— 旧实现 max-w-3xl 居中，比消息列窄 128px（1920）/26px（1440），
	// 左缘错位是「消息页很怪」的主因之一。
	let {
		chat,
		mentionCandidates
	}: { chat: ChatController; showEmojiButton?: boolean; mentionCandidates?: { id: string; name: string }[] } =
		$props();

	/** 浮层开启时的查询词（`@` 之后、光标之前的片段）；null = 未开启 */
	let mentionQuery = $state<string | null>(null);
	let mentionIndex = $state(0);
	let inputEl = $state<HTMLInputElement | null>(null);

	const MENTION_MAX = 6; // 浮层最多展示条数（spec 上限 20 指发送的 mentions）
	const mentionList = $derived(
		mentionQuery === null || !mentionCandidates?.length
			? []
			: mentionCandidates
					.filter((m) => {
						const q = mentionQuery!.toLowerCase();
						return m.name.toLowerCase().includes(q) || m.id.toLowerCase().includes(q);
					})
					.slice(0, MENTION_MAX)
	);
	const mentionOpen = $derived(mentionQuery !== null && mentionList.length > 0);

	/** 从光标前的文本判断是否处于 `@查询` 状态 */
	const syncMentionQuery = () => {
		if (!mentionCandidates?.length) return;
		const el = inputEl;
		const caret = el?.selectionStart ?? chat.inputVal.length;
		const re = /(?:^|\s)@([^\s@]*)$/;
		let m = re.exec(chat.inputVal.slice(0, caret));
		// 兜底：程序化赋值（jsdom 测试/粘贴）时光标常停在 0，此时按整串尾部判定
		if (!m && caret === 0 && chat.inputVal.length > 0) m = re.exec(chat.inputVal);
		mentionQuery = m ? m[1] : null;
		mentionIndex = 0;
	};

	const applyMention = (name: string) => {
		const el = inputEl;
		const caret = el?.selectionStart || (mentionQuery !== null ? chat.inputVal.length : 0);
		const before = chat.inputVal.slice(0, caret);
		const after = chat.inputVal.slice(caret);
		const replaced = before.replace(/(^|\s)@([^\s@]*)$/, (_all, sp: string) => `${sp}@${name} `);
		chat.inputVal = replaced + after;
		mentionQuery = null;
		// 光标落在插入片段之后
		queueMicrotask(() => {
			const pos = replaced.length;
			el?.focus();
			el?.setSelectionRange(pos, pos);
		});
	};

	/** 发送时把文本里仍存在的 @昵称 映射回 user_id（编辑后已删除的不再计入） */
	const collectMentions = (text: string): string[] => {
		if (!mentionCandidates?.length || !text.includes('@')) return [];
		const ids = mentionCandidates.filter((m) => text.includes(`@${m.name}`)).map((m) => m.id);
		return [...new Set(ids)].slice(0, 20);
	};

	const send = () => {
		mentionQuery = null;
		void chat.handleSend(collectMentions(chat.inputVal.trim()) ?? undefined);
	};
</script>

<!-- 输入条：整条铺满聊天面板（发丝线是登记的功能线，不是页面框），
     内部内容经 ChatColumn 与消息列同轴；背景与聊天画布同色，工具栏与画布连成一体 -->
<div
	class="shrink-0 border-t border-border-subtle bg-surface-chat"
	style="padding-bottom:max(0.75rem, env(safe-area-inset-bottom));"
>
	<ChatColumn class="px-4 pt-2.5">
		{#if chat.sendError}
			<p class="pb-1.5 text-status-danger text-xs">{chat.sendError}</p>
		{/if}
		<div class="flex items-center gap-2.5">
			<div class="relative flex h-11 flex-1 items-center rounded-md input-surface px-3.5">
				{#if mentionOpen}
					<!-- 提及选择浮层：贴在输入框上方（输入栏在底部），键盘 ↑↓ 选择、Enter 确认、Esc 关闭 -->
					<div
						class="absolute bottom-full left-0 right-0 mb-2 max-h-56 overflow-y-auto rounded-lg border border-border-default bg-surface-primary p-1 shadow-raised-md"
						id="mention-listbox"
						role="listbox"
						aria-label="提及群成员"
					>
						{#each mentionList as m, i (m.id)}
							<button
								type="button"
								id={`mention-opt-${m.id}`}
								role="option"
								aria-selected={i === mentionIndex}
								class="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-sm focus-ring {i ===
								mentionIndex
									? 'bg-accent-soft text-text-accent-on-soft'
									: 'text-text-primary hover:bg-interactive-hover'}"
								onmousedown={(e) => {
									e.preventDefault(); // 别让输入框失焦
									applyMention(m.name);
								}}
							>
								<span class="truncate">{m.name}</span>
								<span class="ml-auto shrink-0 font-mono text-xxs text-text-secondary">{m.id}</span>
							</button>
						{/each}
					</div>
				{/if}
				<input
					type="text"
					aria-label="消息输入框"
					role="combobox"
					aria-expanded={mentionOpen}
					aria-controls={mentionOpen ? 'mention-listbox' : undefined}
					aria-activedescendant={mentionOpen ? `mention-opt-${mentionList[mentionIndex]?.id}` : undefined}
					aria-autocomplete="list"
					bind:this={inputEl}
					bind:value={chat.inputVal}
					oninput={(e) => {
						chat.handleInputChange(e);
						syncMentionQuery();
					}}
					onkeydown={(e) => {
						if (mentionOpen) {
							if (e.key === 'ArrowDown') {
								e.preventDefault();
								mentionIndex = (mentionIndex + 1) % mentionList.length;
								return;
							}
							if (e.key === 'ArrowUp') {
								e.preventDefault();
								mentionIndex = (mentionIndex - 1 + mentionList.length) % mentionList.length;
								return;
							}
							if (e.key === 'Enter' && !e.isComposing) {
								e.preventDefault();
								applyMention(mentionList[mentionIndex].name);
								return;
							}
							if (e.key === 'Escape') {
								e.preventDefault();
								mentionQuery = null;
								return;
							}
						}
						// IME 组合中的回车是选词确认，不能当发送（中文输入法致命）
						if (e.key === 'Enter' && !e.isComposing) send();
					}}
					placeholder="发消息…"
					class="flex-1 bg-transparent text-text-primary text-md"
				/>
			</div>

			<button
				type="button"
				class="hover-btn hit-slop scale-btn flex size-10 shrink-0 items-center justify-center rounded-full border transition-colors duration-standard focus-ring
					{chat.inputVal.trim() && !chat.sending
					? 'border-transparent bg-accent-default text-text-on-accent shadow-raised-sm'
					: 'border-border-default bg-surface-primary text-text-secondary shadow-raised-sm'}
					{chat.sending ? 'pointer-events-none opacity-soft' : ''}"
				onclick={send}
				aria-label="发送"
				aria-busy={chat.sending || undefined}
				disabled={!chat.inputVal.trim() || chat.sending}
			>
				{#if chat.sending}
					<span class="spinner size-4" aria-hidden="true"></span>
				{:else}
					<Send class="size-4 translate-x-px {chat.inputVal.trim() ? '' : 'text-text-secondary'}" strokeWidth={2} />
				{/if}
			</button>
		</div>
	</ChatColumn>
</div>
