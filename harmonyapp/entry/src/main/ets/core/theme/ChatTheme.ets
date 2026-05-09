/**
 * 聊天页面主题系统
 *
 * 提供聊天页面专用颜色 token 和主题管理。
 */

import { AppThemeManager } from './AppThemeManager';
import { AppThemePalette } from './AppThemeTypes';

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

export enum ThemeSettings {
  FOLLOW_SYSTEM = 'follow_system',
  LIGHT = 'light',
  DARK = 'dark',
}

export class ChatThemeManager {
  private static userSetting: ThemeSettings = ThemeSettings.FOLLOW_SYSTEM;

  static current(): ChatThemeColors {
    return ChatThemeManager.fromPalette(AppThemeManager.getSafeCurrentPalette(), ChatThemeManager.resolveIsDark());
  }

  static light(): ChatThemeColors {
    return ChatThemeManager.fromPalette(AppThemeManager.getPaletteForMode('vibrant', false), false);
  }

  static dark(): ChatThemeColors {
    return ChatThemeManager.fromPalette(AppThemeManager.getPaletteForMode('vibrant', true), true);
  }

  static setTheme(setting: ThemeSettings): void {
    ChatThemeManager.userSetting = setting;
  }

  static getThemeSetting(): ThemeSettings {
    return ChatThemeManager.userSetting;
  }

  private static fromPalette(palette: AppThemePalette, isDark: boolean): ChatThemeColors {
    return {
      background: palette.chatBackground,
      listBackground: palette.pageBackground,
      bubbleSelf: palette.accentBackground,
      bubbleSelfText: palette.accentText,
      bubblePeer: palette.surfaceBackground,
      bubblePeerText: palette.textPrimary,
      bubbleSystem: isDark ? palette.surfaceBackground : palette.surfaceSubtle,
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
