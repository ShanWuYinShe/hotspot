/**
 * 聊天页面主题系统
 *
 * 提供聊天页面专用颜色 token 和主题管理。
 * 颜色通过 $r('app.color.chat_*') 资源引用自动跟随系统深浅模式，
 * ChatThemeManager 额外提供程序化访问，预留用户自定义主题能力。
 */

import { AppThemeManager } from './AppThemeManager';

// ==================== 聊天主题色彩 Token ====================

export interface ChatThemeColors {
  /** 页面背景 */
  background: string;
  /** 消息列表区域背景 */
  listBackground: string;
  /** 自己的消息气泡 */
  bubbleSelf: string;
  /** 自己的气泡文字 */
  bubbleSelfText: string;
  /** 对方的消息气泡 */
  bubblePeer: string;
  /** 对方的气泡文字（跟随系统文字色，此处不硬编码） */
  bubblePeerText: string;
  /** 系统消息背景 */
  bubbleSystem: string;
  /** 输入栏背景 */
  inputBarBg: string;
  /** 输入框背景 */
  inputFieldBg: string;
  /** 输入栏顶部边框 */
  inputBorder: string;
  /** 发送按钮激活态背景 */
  sendButtonActive: string;
  /** 发送按钮激活态文字 */
  sendButtonActiveText: string;
  /** 发送按钮禁用态背景 */
  sendButtonDisabled: string;
  /** 标题栏背景 */
  titleBarBg: string;
  /** 标题栏边框 */
  titleBarBorder: string;
  /** 分割线 */
  divider: string;
}

// ==================== 主题预设值 ====================

const LIGHT_THEME: ChatThemeColors = {
  background: '#F5F6FA',
  listBackground: '#EEF0F5',
  bubbleSelf: '#6B8CFF',
  bubbleSelfText: '#FFFFFF',
  bubblePeer: '#FFFFFF',
  bubblePeerText: '#111827',
  bubbleSystem: '#FFFFFF',
  inputBarBg: '#FFFFFF',
  inputFieldBg: '#F2F2F2',
  inputBorder: '#E8EAF0',
  sendButtonActive: '#6B8CFF',
  sendButtonActiveText: '#FFFFFF',
  sendButtonDisabled: '#F3F4F6',
  titleBarBg: '#FFFFFF',
  titleBarBorder: '#F3F4F6',
  divider: '#F3F4F6',
};

const DARK_THEME: ChatThemeColors = {
  background: '#11131D',
  listBackground: '#181C28',
  bubbleSelf: '#5F83E8',
  bubbleSelfText: '#FFFFFF',
  bubblePeer: '#242A38',
  bubblePeerText: '#F9FAFB',
  bubbleSystem: '#1C2030',
  inputBarBg: '#11151D',
  inputFieldBg: '#252B38',
  inputBorder: '#2D3544',
  sendButtonActive: '#5F83E8',
  sendButtonActiveText: '#FFFFFF',
  sendButtonDisabled: '#222733',
  titleBarBg: '#171A20',
  titleBarBorder: '#2D3544',
  divider: '#28303D',
};

// ==================== 主题设置 ====================

export enum ThemeSettings {
  FOLLOW_SYSTEM = 'follow_system',
  LIGHT = 'light',
  DARK = 'dark',
}

// ==================== ChatThemeManager ====================

export class ChatThemeManager {
  private static userSetting: ThemeSettings = ThemeSettings.FOLLOW_SYSTEM;

  /**
   * 获取当前聊天主题色彩。
   * 根据用户设置和系统深浅模式返回对应主题。
   */
  static current(): ChatThemeColors {
    const palette = AppThemeManager.getSafeCurrentPalette();
    if (palette.id === 'tactile_minimal_light') {
      return {
        background: '#EFE8DE',
        listBackground: '#F3ECE3',
        bubbleSelf: '#D7CCBC',
        bubbleSelfText: '#6F6256',
        bubblePeer: '#FBF7F1',
        bubblePeerText: '#5F564D',
        bubbleSystem: '#F6F0E7',
        inputBarBg: '#F5F0E8',
        inputFieldBg: '#E9E1D6',
        inputBorder: '#D9D0C4',
        sendButtonActive: '#D7CCBC',
        sendButtonActiveText: '#6F6256',
        sendButtonDisabled: '#E4DBCF',
        titleBarBg: '#F5F0E8',
        titleBarBorder: '#D9D0C4',
        divider: '#D9D0C4'
      };
    }
    if (palette.id === 'zen_artistic_light') {
      return {
        background: '#CFDADD',
        listBackground: '#D8E2E5',
        bubbleSelf: '#ECE2D7',
        bubbleSelfText: '#7A695B',
        bubblePeer: '#EEF3F4',
        bubblePeerText: '#536066',
        bubbleSystem: '#E2EAEC',
        inputBarBg: '#DDE6E8',
        inputFieldBg: '#C9D3D6',
        inputBorder: '#BFCBD0',
        sendButtonActive: '#ECE2D7',
        sendButtonActiveText: '#7A695B',
        sendButtonDisabled: '#D6DFE1',
        titleBarBg: '#DDE6E8',
        titleBarBorder: '#BFCBD0',
        divider: '#BFCBD0'
      };
    }
    const isDark = ChatThemeManager.resolveIsDark();
    return isDark ? DARK_THEME : LIGHT_THEME;
  }

  /** 获取浅色主题 */
  static light(): ChatThemeColors {
    return LIGHT_THEME;
  }

  /** 获取深色主题 */
  static dark(): ChatThemeColors {
    return DARK_THEME;
  }

  /** 设置用户偏好 */
  static setTheme(setting: ThemeSettings): void {
    ChatThemeManager.userSetting = setting;
  }

  /** 获取当前用户偏好 */
  static getThemeSetting(): ThemeSettings {
    return ChatThemeManager.userSetting;
  }

  private static resolveIsDark(): boolean {
    switch (ChatThemeManager.userSetting) {
      case ThemeSettings.LIGHT:
        return false;
      case ThemeSettings.DARK:
        return true;
      case ThemeSettings.FOLLOW_SYSTEM:
      default: {
        // 读取 EntryAbility 写入 AppStorage 的系统深浅模式标志
        const isDark = AppStorage.get<boolean>('isDarkMode');
        return isDark === true;
      }
    }
  }
}
