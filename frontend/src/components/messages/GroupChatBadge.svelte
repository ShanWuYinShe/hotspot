<script lang="ts">
	import { Users } from '@lucide/svelte';

	// 群聊类型角标：叠加在群头像右下角，让「群聊 / 私聊」在会话列表与聊天头部一眼可分。
	// 仅靠头像形状（群组圆角矩形 / 私聊圆形）在有真实头像时不够显眼；补一个类型图标信号。
	// 与 OnlineDot 同位互斥：群聊显示本角标，私聊仍有在线绿点（两者都是头像右下角）。
	// 样式沿用 OnlineDot 的角标语言：绝对定位 + border-surface-primary 描边。

	let { size = 16 }: {
		/** 角标边长（px）：会话列表行 16 / 聊天头部 18。 */
		size?: number;
	} = $props();
</script>

<span
	data-group-chat-badge
	aria-hidden="true"
	class="absolute -bottom-0.5 -right-0.5 flex items-center justify-center rounded-full border-strong border-surface-primary bg-accent-default text-text-on-accent"
	style="width:{size}px;height:{size}px;"
>
	<Users class={size >= 18 ? 'size-3' : 'size-2.5'} strokeWidth={2.25} />
</span>
