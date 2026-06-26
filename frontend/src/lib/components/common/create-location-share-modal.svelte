<script lang="ts">
  import { shareService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { toast } from '$lib/stores/toast.svelte';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import Share2 from 'lucide-svelte/icons/share-2';
  import Copy from 'lucide-svelte/icons/copy';
  import Check from 'lucide-svelte/icons/check';
  import Loader2 from 'lucide-svelte/icons/loader-2';
  import UsersRound from 'lucide-svelte/icons/users-round';
  import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription
  } from '$lib/components/ui/dialog';

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
  <DialogContent
    class="max-w-lg"
    style="background: var(--surface-bg); border: 1px solid var(--border-theme); box-shadow: var(--shadow-lg);"
  >
    {#if !shareCode}
      <!-- 创建表单 -->
      <DialogHeader>
        <DialogTitle class="flex items-center gap-3">
          <div
            class="flex items-center justify-center rounded-full"
            style="width: 40px; height: 40px; box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
          >
            <Share2 style="width: 20px; height: 20px; color: var(--brand);" />
          </div>
          <div style="color: var(--text-primary);">分享此位置</div>
        </DialogTitle>
        <DialogDescription style="color: var(--text-secondary);"
          >生成分享链接，让朋友快速定位到这个区域</DialogDescription
        >
      </DialogHeader>

      <div class="mt-4 space-y-4">
        <!-- 位置信息展示 -->
        <div
          class="rounded-[var(--radius-lg)] p-4 space-y-2"
          style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
        >
          <div
            class="flex items-center gap-2"
            style="font-weight: 500; color: var(--text-primary);"
          >
            <MapPin style="width: 20px; height: 20px; color: var(--brand);" />
            {locationName}
          </div>
          <div class="text-meta" style="color: var(--text-secondary)">
            接收者将看到以此位置为中心的内容推荐。
          </div>
        </div>

        <!-- 说明 -->
        <div
          class="text-body rounded-[var(--radius-lg)] p-4 space-y-2"
          style="background: var(--surface-bg); border: 1px solid var(--theme-overlay-subtle)"
        >
          <p class="flex items-center gap-2" style="font-weight: 500; color: var(--brand);">
            <UsersRound style="width: 16px; height: 16px;" />
            接收者可以
          </p>
          <ul class="space-y-1.5" style="color: var(--text-secondary);">
            <li>• 一键设置浏览锚点到这个位置</li>
            <li>• 查看附近的人、群组和动态</li>
            <li>• 开始探索这个区域的社交圈</li>
          </ul>
        </div>

        <!-- 操作按钮 -->
        <button
          class="text-body rounded-full px-5 py-2.5 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97] w-full"
          style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn)"
          onclick={handleSubmit}
          disabled={isSubmitting}
        >
          {#if isSubmitting}
            <Loader2 class="mr-2 inline-block" style="width: 16px; height: 16px;" />
            生成中...
          {:else}
            <Share2 class="mr-2 inline-block" style="width: 16px; height: 16px;" />
            生成分享链接
          {/if}
        </button>
      </div>
    {:else}
      <!-- 分享成功结果 -->
      <DialogHeader>
        <DialogTitle class="flex items-center gap-3" style="color: var(--success);">
          <div
            class="flex items-center justify-center rounded-full"
            style="width: 40px; height: 40px; box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
          >
            <Check style="width: 20px; height: 20px; color: var(--success);" />
          </div>
          <div>分享链接已生成！</div>
        </DialogTitle>
        <DialogDescription style="color: var(--text-secondary);"
          >复制链接发送给朋友</DialogDescription
        >
      </DialogHeader>

      <div class="mt-4 space-y-4">
        <!-- 分享链接 -->
        <div
          class="rounded-[var(--radius-lg)] p-4 space-y-2"
          style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
        >
          <label for="share-link-url" class="text-meta" style="color: var(--text-secondary)"
            >分享链接</label
          >
          <div class="flex gap-2">
            <input
              id="share-link-url"
              value={shareUrl ?? ''}
              readonly
              class="text-meta field-inset rounded-[var(--radius-md)] px-4 py-2.5 outline-none transition-all duration-150 flex-1"
              style="background: var(--surface-bg); color: var(--text-primary); font-family: monospace"
            />
            <button
              class="rounded-[var(--radius-md)] px-3 py-2.5 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97] flex items-center justify-center"
              style="background: var(--surface-bg); color: var(--text-primary); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
              onclick={copyToClipboard}
            >
              {#if isCopied}
                <Check style="width: 16px; height: 16px; color: var(--success);" />
              {:else}
                <Copy style="width: 16px; height: 16px;" />
              {/if}
            </button>
          </div>
        </div>

        <button
          class="text-body rounded-full px-5 py-2.5 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97] w-full"
          style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn)"
          onclick={handleClose}
        >
          完成
        </button>
      </div>
    {/if}
  </DialogContent>
</Dialog>
