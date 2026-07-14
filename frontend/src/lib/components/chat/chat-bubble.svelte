<script lang="ts" module>
  import type { Snippet } from 'svelte';

  interface ChatMessageMedia {
    url: string;
  }

  interface Props {
    /** 是否为自己发送(决定配色与尖角方向) */
    isSelf: boolean;
    /** 消息类型:text / image / share / ... */
    type?: string;
    /** 文本内容 */
    content?: string;
    /** 图片媒体(image 类型) */
    media?: ChatMessageMedia | null;
    /** 是否撤回 */
    isRecalled?: boolean;
    /** 时间戳文案;传入则在气泡内底部渲染(单聊风格),不传则由调用方在外部布局(群聊风格) */
    time?: string;
    /** 已读回执(仅 isSelf + time 时生效) */
    readReceipt?: boolean;
    /** 图片点击预览回调;传入则图片渲染为可点 button,否则纯 img */
    onImageClick?: (url: string) => void;
    /** 额外 class */
    class?: string;
    /** 自定义内容(如 share 卡片),在 type 非 text/image/recalled 时渲染 */
    children?: Snippet;
  }
</script>

<script lang="ts">
  import { CheckCheck } from 'lucide-svelte';
  import { t } from '$lib/i18n';

  let {
    isSelf,
    type = 'text',
    content = '',
    media = null,
    isRecalled = false,
    time,
    readReceipt = false,
    onImageClick,
    class: className = '',
    children
  }: Props = $props();
</script>

<!--
  统一聊天气泡本体:圆角 var(--radius-md) + 4px 尖角(指向头像侧)、
  padding(px-4 py-2.5 / 图片 p-1)、配色(自己 brand+shadow-brand-btn / 对方 surface-bg+border)。
  单聊/群聊共用;头像、昵称、时间戳外部布局由调用方决定(单聊 time 传入→气泡内,群聊 time 不传→气泡外)。
-->
<div
  class="relative {type === 'image' ? 'p-1' : 'px-4 py-2.5'} {className}"
  style={isSelf
    ? 'background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn); border-radius: var(--radius-md) var(--radius-md) 4px var(--radius-md);'
    : 'background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-primary); border-radius: var(--radius-md) var(--radius-md) var(--radius-md) 4px;'}
>
  {#if isRecalled}
    <p class="text-body" style="font-style: italic; opacity: 0.7">{t('chat.messageRecalled')}</p>
  {:else if type === 'text'}
    <p class="text-body whitespace-pre-wrap wrap-break-word" style="line-height: 1.55">
      {content}
    </p>
  {:else if type === 'image' && media}
    {#if onImageClick}
      <button
        type="button"
        class="block overflow-hidden rounded-[var(--radius-md)] border-none bg-transparent p-0 shadow-none outline-none"
        onclick={() => onImageClick(media!.url)}
        aria-label={t('chat.viewImageAria')}
      >
        <img
          src={media.url}
          alt={t('chat.image')}
          loading="lazy"
          decoding="async"
          class="max-h-[300px] max-w-[250px] object-cover"
        />
      </button>
    {:else}
      <img
        src={media.url}
        alt={t('chat.image')}
        loading="lazy"
        decoding="async"
        class="max-w-[250px] rounded-[var(--radius-md)]"
      />
    {/if}
  {:else if children}
    {@render children()}
  {:else}
    <p class="text-body" style="opacity: 0.7">[{type}]</p>
  {/if}

  {#if time}
    <p
      class="text-xs"
      style="margin-top: 4px; color: {isSelf
        ? 'color-mix(in srgb, var(--text-on-accent) 65%, transparent)'
        : 'var(--text-secondary)'}; display: flex; align-items: center; gap: 3px; justify-content: {isSelf
        ? 'flex-end'
        : 'flex-start'}"
    >
      {time}
      {#if readReceipt}
        <span style="display: inline-flex; width: 12px; height: 12px; opacity: 0.8;">
          <CheckCheck size={12} color="var(--text-on-accent)" strokeWidth={2} />
        </span>
      {/if}
    </p>
  {/if}
</div>
