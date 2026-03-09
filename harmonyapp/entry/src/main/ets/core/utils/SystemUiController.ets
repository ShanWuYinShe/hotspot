import { SystemSurfaceStyle, SystemUiThemeStore } from './SystemUiThemeStore';
import { AppThemeManager } from '../theme/AppThemeManager';

export class SystemUiController {
  static useSurface(style: SystemSurfaceStyle): void {
    SystemUiThemeStore.setStyle(style);
  }

  static resetToDefault(): void {
    SystemUiThemeStore.reset();
  }

  static syncWithCurrentTheme(): void {
    SystemUiThemeStore.setStyle(AppThemeManager.getCurrentSurfaceStyle());
  }
}
