import { toast } from '$lib/stores/toast.svelte';
import { t } from '$lib/i18n';

/**
 * Copy a share URL to clipboard.
 * Uses existing i18n keys for success/error toasts.
 */
export async function copyShareUrl(url: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(url);
    toast.success(t('share.linkCopied'));
  } catch {
    toast.error(t('share.copyFailed'));
  }
}
