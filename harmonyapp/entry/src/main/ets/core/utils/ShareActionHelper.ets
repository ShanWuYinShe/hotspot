/**
 * 通用分享动作工具类
 * 封装 share_code 创建 + 系统分享面板调起的完整流程
 */
import { common } from '@kit.AbilityKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { ApiClient } from '../api/ApiClient';
import { SystemShareService, ShareContent } from '../services/SystemShareService';

const DOMAIN_NUMBER: number = 0xFF30;
const TAG: string = 'ShareActionHelper';

export class ShareActionHelper {
  /**
   * 创建分享并通过系统分享面板分发
   */
  static async shareResource(
    context: common.UIAbilityContext,
    shareType: string,
    payload: Record<string, Object>,
    title: string,
    description: string
  ): Promise<void> {
    try {
      const response = await ApiClient.createShare({
        share_type: shareType,
        payload: payload
      });

      if (response.code === 0 && response.data) {
        const shareCode = response.data.share_code;
        const shareUrl = `https://geotalk.app/share/${shareCode}`;

        const content = new ShareContent();
        content.title = title;
        content.description = description;
        content.url = shareUrl;

        await SystemShareService.getInstance().shareText(context, content);
      } else {
        hilog.error(DOMAIN_NUMBER, TAG, `Create share failed: ${response.message}`);
      }
    } catch (error) {
      const msg = error instanceof Error ? error.message : '';
      if (msg) {
        hilog.error(DOMAIN_NUMBER, TAG, `shareResource error: ${msg}`);
      }
    }
  }
}
