/**
 * 系统分享服务
 * 利用 HarmonyOS UIAbility startAbility 调起系统分享面板
 */

import { common, Want } from '@kit.AbilityKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { WEB_BASE_URL } from '../../constants/AppConstants';

const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'SystemShareService';

/** 分享内容类型 */
export class ShareContent {
  title: string = '';
  description: string = '';
  url: string = '';
  imageUrl: string = '';
}

export class SystemShareService {
  private static instance: SystemShareService | null = null;

  static getInstance(): SystemShareService {
    if (!SystemShareService.instance) {
      SystemShareService.instance = new SystemShareService();
    }
    return SystemShareService.instance;
  }

  /** 分享文本内容到系统分享面板 */
  async shareText(context: common.UIAbilityContext, content: ShareContent): Promise<void> {
    try {
      const text = this.buildShareText(content);
      const want: Want = {
        action: 'ohos.want.action.sendData',
        type: 'text/plain',
        parameters: {
          'shareTitle': content.title,
          'shareData': text
        }
      };

      await context.startAbility(want);
      hilog.info(DOMAIN_NUMBER, TAG, `分享成功: ${content.title}`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `分享失败: ${JSON.stringify(error)}`);
      throw error instanceof Error ? error : new Error(String(error));
    }
  }

  private buildShareText(content: ShareContent): string {
    let text = `${content.title}\n${content.description}`;
    if (content.url) {
      text += `\n${content.url}`;
    }
    return text;
  }

  /** 构建用户名片分享内容 */
  static buildUserShareContent(nickname: string, bio: string = '', userId: string = ''): ShareContent {
    const content = new ShareContent();
    content.title = `来认识 ${nickname}`;
    content.description = bio || '快来 HotSpot 和附近的人交流吧！';
    content.url = userId ? `${WEB_BASE_URL}/users/${userId}` : '';
    return content;
  }

  /** 构建群组分享内容 */
  static buildGroupShareContent(groupName: string, description: string = '', groupId: string = ''): ShareContent {
    const content = new ShareContent();
    content.title = `加入群组「${groupName}」`;
    content.description = description || '在 HotSpot 发现有趣的附近群组';
    content.url = groupId ? `${WEB_BASE_URL}/groups/${groupId}` : '';
    return content;
  }

  /** 构建动态分享内容 */
  static buildMomentShareContent(text: string, author: string, momentId: string = ''): ShareContent {
    const content = new ShareContent();
    content.title = `${author} 的动态`;
    content.description = text.length > 100 ? text.substring(0, 100) + '...' : text;
    content.url = momentId ? `${WEB_BASE_URL}/moments/${momentId}` : '';
    return content;
  }
}
