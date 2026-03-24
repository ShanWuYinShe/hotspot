/**
 * 聊天页面主题系统
 *
 * 提供聊天页面专用颜色 token 和主题管理。
 */

import { AppThemeManager } from './AppThemeManager';

export interface ChatThemeColors {
  background: string;
  listBackground: string;
  bubbleSelf: string;
  bubbleSelfText: string;
  bubblePeer: string;
  bubblePeerText: string;
  bubbleSystem: string;
  inputBarBg: string;
  inputFieldBg: string;
  inputBorder: string;
  sendButtonActive: string;
  sendButtonActiveText: string;
  sendButtonDisabled: string;
  titleBarBg: string;
  titleBarBorder: string;
  divider: string;
}

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

export enum ThemeSettings {
  FOLLOW_SYSTEM = 'follow_system',
  LIGHT = 'light',
  DARK = 'dark',
}

export class ChatThemeManager {
  private static userSetting: ThemeSettings = ThemeSettings.FOLLOW_SYSTEM;

  static current(): ChatThemeColors {
    const palette = AppThemeManager.getSafeCurrentPalette();
    if (palette.id === 'default_clean_light' || palette.id === 'default_clean_dark') {
      return ChatThemeManager.resolveIsDark() ? DARK_THEME : LIGHT_THEME;
    }

    return {
      background: palette.chatBackground,
      listBackground: palette.pageBackground,
      bubbleSelf: palette.accentBackground,
      bubbleSelfText: palette.accentText,
      bubblePeer: palette.surfaceBackground,
      bubblePeerText: palette.textPrimary,
      bubbleSystem: palette.surfaceSubtle,
      inputBarBg: palette.surfaceBackground,
      inputFieldBg: palette.surfaceSubtle,
      inputBorder: palette.borderColor,
      sendButtonActive: palette.accentBackground,
      sendButtonActiveText: palette.accentText,
      sendButtonDisabled: palette.buttonSecondaryBackground,
      titleBarBg: palette.surfaceBackground,
      titleBarBorder: palette.borderColor,
      divider: palette.borderColor
    };
  }

  static light(): ChatThemeColors {
    return LIGHT_THEME;
  }

  static dark(): ChatThemeColors {
    return DARK_THEME;
  }

  static setTheme(setting: ThemeSettings): void {
    ChatThemeManager.userSetting = setting;
  }

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
        const isDark = AppStorage.get<boolean>('isDarkMode');
        return isDark === true;
      }
    }
  }
}
