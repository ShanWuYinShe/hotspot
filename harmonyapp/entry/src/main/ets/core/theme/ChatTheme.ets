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

export class ChatThemeManager {
  /** 聊天页专用色聚合：全部由当前主题 palette 派生，无独立色值 */
  static current(): ChatThemeColors {
    return ChatThemeManager.fromPalette(
      AppThemeManager.getSafeCurrentPalette(),
      AppStorage.get<boolean>('isDarkMode') === true
    );
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

}
