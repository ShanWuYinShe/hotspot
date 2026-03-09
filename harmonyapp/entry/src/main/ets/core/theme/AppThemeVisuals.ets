import { AppThemeManager } from './AppThemeManager';

export class AppThemeVisuals {
  static getSurfaceShadow(themeId: string): ShadowOptions {
    return AppThemeManager.getPaletteById(themeId).surfaceShadow;
  }

  static getElevatedShadow(themeId: string): ShadowOptions {
    return AppThemeManager.getPaletteById(themeId).elevatedShadow;
  }

  static getAccentShadow(themeId: string): ShadowOptions {
    return AppThemeManager.getPaletteById(themeId).accentShadow;
  }

  static getControlShadow(themeId: string): ShadowOptions {
    return AppThemeManager.getPaletteById(themeId).controlShadow;
  }

  static getTabBarShadow(themeId: string): ShadowOptions {
    return AppThemeManager.getPaletteById(themeId).tabBarShadow;
  }

  static getOverlayStrong(themeId: string): string {
    return AppThemeManager.getPaletteById(themeId).overlayStrong;
  }
}
