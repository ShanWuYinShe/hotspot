import { DesignConstants } from '../../constants/DesignConstants';
import { AppThemeManager } from './AppThemeManager';

export interface ThemeCardVisualStyle {
  backgroundColor: ResourceColor;
  borderColor: ResourceColor;
  borderWidth: number;
  shadow: ShadowOptions;
  borderRadius: number;
  padding: number;
  minHeight?: number;
}

export interface ThemeButtonVisualStyle {
  backgroundColor: ResourceColor;
  textColor: ResourceColor;
  borderColor: ResourceColor;
  borderWidth: number;
  shadow: ShadowOptions;
  height: number;
  horizontalPadding: number;
}

export interface ThemeMetricVisualStyle extends ThemeCardVisualStyle {
  iconColor: ResourceColor;
  valueFontSize: number;
}

export interface ThemeProfileIdentityStyle {
  avatarSize: number;
  titleFontSize: number;
}

export interface ThemeProfileVisualStyle {
  card: ThemeCardVisualStyle;
  metricCard: ThemeMetricVisualStyle;
  primaryButton: ThemeButtonVisualStyle;
  identity: ThemeProfileIdentityStyle;
}

export class AppThemeVisuals {
  static getSurfaceShadow(themeId: string): ShadowOptions {
    return AppThemeManager.getPalette(themeId).surfaceShadow;
  }

  static getElevatedShadow(themeId: string): ShadowOptions {
    return AppThemeManager.getPalette(themeId).elevatedShadow;
  }

  static getAccentShadow(themeId: string): ShadowOptions {
    return AppThemeManager.getPalette(themeId).accentShadow;
  }

  static getControlShadow(themeId: string): ShadowOptions {
    return AppThemeManager.getPalette(themeId).controlShadow;
  }

  static getTabBarShadow(themeId: string): ShadowOptions {
    return AppThemeManager.getPalette(themeId).tabBarShadow;
  }

  static getOverlayStrong(themeId: string): string {
    return AppThemeManager.getPalette(themeId).overlayStrong;
  }

  static getSurfaceCardStyle(themeId: string): ThemeCardVisualStyle {
    const palette = AppThemeManager.getPalette(themeId);
    return {
      backgroundColor: palette.surfaceBackground,
      borderColor: palette.borderColor,
      borderWidth: 1,
      shadow: palette.surfaceShadow,
      borderRadius: AppThemeManager.isSoftGalleryTheme(themeId)
        ? DesignConstants.RADIUS.XXL + 4
        : DesignConstants.RADIUS.XXL,
      padding: AppThemeManager.isSoftGalleryTheme(themeId)
        ? DesignConstants.SPACING.LG + 2
        : DesignConstants.SPACING.LG
    };
  }

  static getProfileMetricCardStyle(themeId: string): ThemeMetricVisualStyle {
    const palette = AppThemeManager.getPalette(themeId);
    const baseStyle = AppThemeVisuals.getSurfaceCardStyle(themeId);
    return {
      ...baseStyle,
      padding: AppThemeManager.isSoftGalleryTheme(themeId)
        ? DesignConstants.SPACING.XL
        : DesignConstants.SPACING.LG,
      minHeight: AppThemeManager.isSoftGalleryTheme(themeId) ? 176 : 164,
      iconColor: palette.accentBackground,
      valueFontSize: AppThemeManager.isSoftGalleryTheme(themeId) ? 24 : 22
    };
  }

  static getProfilePrimaryButtonStyle(themeId: string): ThemeButtonVisualStyle {
    const palette = AppThemeManager.getPalette(themeId);
    return {
      backgroundColor: palette.accentBackground,
      textColor: palette.accentText,
      borderColor: palette.borderColor,
      borderWidth: 1,
      shadow: palette.accentShadow,
      height: AppThemeManager.isSoftGalleryTheme(themeId) ? 54 : 42,
      horizontalPadding: AppThemeManager.isSoftGalleryTheme(themeId)
        ? DesignConstants.SPACING.XXL + 8
        : DesignConstants.SPACING.XXL
    };
  }

  static getProfileIdentityStyle(themeId: string): ThemeProfileIdentityStyle {
    return {
      avatarSize: AppThemeManager.isSoftGalleryTheme(themeId) ? 116 : 96,
      titleFontSize: AppThemeManager.isSoftGalleryTheme(themeId)
        ? DesignConstants.FONT.SIZE.XXL
        : DesignConstants.FONT.SIZE.XL
    };
  }

  static getProfileVisualStyle(themeId: string): ThemeProfileVisualStyle {
    return {
      card: AppThemeVisuals.getSurfaceCardStyle(themeId),
      metricCard: AppThemeVisuals.getProfileMetricCardStyle(themeId),
      primaryButton: AppThemeVisuals.getProfilePrimaryButtonStyle(themeId),
      identity: AppThemeVisuals.getProfileIdentityStyle(themeId)
    };
  }
}
