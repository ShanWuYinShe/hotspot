<script lang="ts">
	import { shareService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { MapPin, Share2, Copy, Check, Loader2, UsersRound } from 'lucide-svelte';
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogTitle,
		DialogDescription
	} from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import { Label } from '$lib/components/ui/label';
	import { Input } from '$lib/components/ui/input';

	interface Props {
		open?: boolean;
		latitude: number;
		longitude: number;
		locationName: string;
		onClose?: () => void;
	}

	let { open = $bindable(false), latitude, longitude, locationName, onClose }: Props = $props();

	// 表单数据
	let isSubmitting = $state(false);
	let shareCode = $state<string | null>(null);
	let shareUrl = $state<string | null>(null);
	let isCopied = $state(false);

	// 复制分享链接
	async function copyToClipboard() {
		if (!shareUrl) return;
		try {
			await navigator.clipboard.writeText(shareUrl);
			isCopied = true;
			toast.success('分享链接已复制到剪贴板');
			setTimeout(() => {
				isCopied = false;
			}, 2000);
		} catch (err) {
			toast.error('复制失败，请手动复制');
		}
	}

	// 提交创建分享
	async function handleSubmit() {
		isSubmitting = true;
		try {
			const response = await shareService.create({
				share_type: 'location',
				payload: {
					shared_location: {
						point: {
							latitude,
							longitude
						},
						location_name: locationName
					}
				}
			});

			if (isSuccess(response) && response.data) {
				const shareCreated = response.data;
				shareCode = shareCreated.share_code;
				shareUrl = `${window.location.origin}/share/${shareCreated.share_code}`;
				toast.success('位置分享创建成功！');
			} else {
				toast.error('创建失败', response.message);
			}
		} catch (error) {
			toast.error('网络错误', '请检查网络连接');
		} finally {
			isSubmitting = false;
		}
	}

	// 关闭并重置
	function handleClose() {
		if (!isSubmitting) {
			// 重置表单
			shareCode = null;
			shareUrl = null;
			isCopied = false;

			open = false;
			onClose?.();
		}
	}
</script>

<Dialog bind:open onOpenChange={(newOpen) => !newOpen && handleClose()}>
	<DialogContent class="max-w-lg rounded-3xl border border-border/60 bg-card/95 shadow-xl backdrop-blur-md">
		{#if !shareCode}
			<!-- 创建表单 -->
			<DialogHeader>
				<DialogTitle class="flex items-center gap-3">
					<div class="flex h-10 w-10 items-center justify-center rounded-2xl border border-border/50 bg-primary/10 text-primary">
						<Share2 class="h-5 w-5" />
					</div>
					<div>分享此位置</div>
				</DialogTitle>
				<DialogDescription>生成分享链接，让朋友快速定位到这个区域</DialogDescription>
			</DialogHeader>

			<div class="mt-4 space-y-4">
				<!-- 位置信息展示 -->
				<div class="space-y-2 rounded-2xl border border-border/50 bg-background/65 p-4">
					<div class="flex items-center gap-2 font-medium">
						<MapPin class="h-5 w-5 text-primary" />
						{locationName}
					</div>
					<div class="text-xs text-muted-foreground">
						{latitude.toFixed(6)}, {longitude.toFixed(6)}
					</div>
				</div>

				<!-- 说明 -->
				<div class="space-y-2 rounded-2xl border border-primary/25 bg-primary/10 p-4 text-sm">
					<p class="font-medium text-primary flex items-center gap-2">
						<UsersRound class="h-4 w-4" />
						接收者可以
					</p>
					<ul class="space-y-1.5 text-muted-foreground">
						<li>• 一键设置浏览锚点到这个位置</li>
						<li>• 查看附近的人、群组和动态</li>
						<li>• 开始探索这个区域的社交圈</li>
					</ul>
				</div>

				<!-- 操作按钮 -->
				<Button class="h-11 w-full rounded-2xl" onclick={handleSubmit} disabled={isSubmitting}>
					{#if isSubmitting}
						<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						生成中...
					{:else}
						<Share2 class="mr-2 h-4 w-4" />
						生成分享链接
					{/if}
				</Button>
			</div>
		{:else}
			<!-- 分享成功结果 -->
			<DialogHeader>
				<DialogTitle class="flex items-center gap-3 text-emerald-600">
					<div class="flex h-10 w-10 items-center justify-center rounded-2xl border border-emerald-200 bg-emerald-50 text-emerald-600 dark:border-emerald-500/40 dark:bg-emerald-500/15">
						<Check class="h-5 w-5" />
					</div>
					<div>分享链接已生成！</div>
				</DialogTitle>
				<DialogDescription>复制链接发送给朋友</DialogDescription>
			</DialogHeader>

			<div class="mt-4 space-y-4">
				<!-- 分享链接 -->
				<div class="space-y-2 rounded-2xl border border-border/50 bg-background/65 p-4">
					<Label class="text-sm text-muted-foreground">分享链接</Label>
					<div class="flex gap-2">
						<Input
							value={shareUrl ?? ''}
							readonly
							class="h-11 flex-1 rounded-2xl border-border/60 bg-background/75 font-mono text-sm"
						/>
						<Button variant="outline" class="h-11 rounded-2xl border-border/60 bg-background/70" onclick={copyToClipboard}>
							{#if isCopied}
								<Check class="h-4 w-4" />
							{:else}
								<Copy class="h-4 w-4" />
							{/if}
						</Button>
					</div>
				</div>

				<Button class="h-11 w-full rounded-2xl" onclick={handleClose}>
					完成
				</Button>
			</div>
		{/if}
	</DialogContent>
</Dialog>
