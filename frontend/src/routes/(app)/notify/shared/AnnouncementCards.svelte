<script lang="ts">
	import { Megaphone, ExternalLink } from '@lucide/svelte';
	import Badge from '@/components/data-display/Badge.svelte';
	import Button from '@/components/ui/button.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import { formatMomentTime } from '@/lib/utils';
	import type { AnnounceSeverity, SystemAnnouncementItem } from '@/api/types';

	// 平台公告卡（/notify 顶部）：公开端点，游客也可见（登录后同时出现在通知流里）。
	// action_url 仅接受 http(s)：javascript:/data: 等一律降级为纯文本，防止注入式跳转。
	let { announcements }: { announcements: SystemAnnouncementItem[] } = $props();

	const SEVERITY_LABEL: Record<AnnounceSeverity, string> = {
		info: '公告',
		warning: '提醒',
		error: '重要'
	};
	// 与管理后台公告页同口径：info→neutral / warning→warning / error→danger（软底徽章）
	const SEVERITY_TONE: Record<AnnounceSeverity, 'neutral' | 'warning' | 'danger'> = {
		info: 'neutral',
		warning: 'warning',
		error: 'danger'
	};

	function safeUrl(url: string | null): string | null {
		if (!url) return null;
		try {
			const parsed = new URL(url);
			return parsed.protocol === 'http:' || parsed.protocol === 'https:' ? parsed.toString() : null;
		} catch {
			return null;
		}
	}

	function openUrl(url: string) {
		window.open(url, '_blank', 'noopener,noreferrer');
	}
</script>

{#if announcements.length > 0}
	<SectionHead variant="eyebrow" title="平台公告" icon={Megaphone} count={announcements.length} />
	<div class="mb-6 flex flex-col gap-2">
		{#each announcements as item (item.announcement_id)}
			{@const href = safeUrl(item.action_url)}
			<article class="rounded-lg border border-border-subtle bg-surface-primary px-4 py-3 shadow-raised-sm">
				<div class="flex items-center gap-2">
					<Badge tone={SEVERITY_TONE[item.severity]}>{SEVERITY_LABEL[item.severity]}</Badge>
					<h3 class="min-w-0 flex-1 truncate text-sm font-semibold text-text-primary">{item.title}</h3>
					<time class="shrink-0 text-xxs text-text-secondary" datetime={item.created_at}>{formatMomentTime(item.created_at)}</time>
				</div>
				<p class="mt-1.5 line-clamp-3 whitespace-pre-line text-xs leading-card text-text-secondary">{item.content}</p>
				{#if href}
					<div class="mt-2.5 flex justify-end">
						<Button
							type="button"
							variant="soft"
							size="sm"
							class="hit-slop"
							onclick={() => openUrl(href)}
						>
							<ExternalLink class="size-3.5" aria-hidden="true" />
							查看详情
						</Button>
					</div>
				{/if}
			</article>
		{/each}
	</div>
{/if}
