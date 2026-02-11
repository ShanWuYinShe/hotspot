/**
 * NFC 近场分享服务
 * 利用 HarmonyOS NFC 能力实现碰一碰分享用户名片/群组等
 */

import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'NfcShareService';

/** NFC 分享数据 */
export class NfcShareData {
  type: string = '';
  userId: string = '';
  nickname: string = '';
  bio: string = '';
  url: string = '';
}

export class NfcShareService {
  private static instance: NfcShareService | null = null;
  private currentShareData: NfcShareData | null = null;

  static getInstance(): NfcShareService {
    if (!NfcShareService.instance) {
      NfcShareService.instance = new NfcShareService();
    }
    return NfcShareService.instance;
  }

  /** 设置当前 NFC 分享数据 */
  setShareData(data: NfcShareData): void {
    this.currentShareData = data;
    hilog.info(DOMAIN_NUMBER, TAG, `设置 NFC 分享数据: type=${data.type}`);
  }

  /** 清除 NFC 分享数据 */
  clearShareData(): void {
    this.currentShareData = null;
    hilog.info(DOMAIN_NUMBER, TAG, '清除 NFC 分享数据');
  }

  /** 获取当前分享数据 */
  getShareData(): NfcShareData | null {
    return this.currentShareData;
  }

  /** 构建用户资料分享数据 */
  static buildUserProfileShare(userId: string, nickname: string, bio: string = '', avatarUrl: string = ''): NfcShareData {
    const data = new NfcShareData();
    data.type = 'user_profile';
    data.userId = userId;
    data.nickname = nickname;
    data.bio = bio;
    data.url = `https://geotalk.app/user/${userId}`;
    return data;
  }
}
