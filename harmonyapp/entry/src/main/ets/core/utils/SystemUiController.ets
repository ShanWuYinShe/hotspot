import { SystemSurfaceStyle, SystemUiThemeStore } from './SystemUiThemeStore';

export class SystemUiController {
  static useSurface(style: SystemSurfaceStyle): void {
    SystemUiThemeStore.setStyle(style);
  }

  static resetToDefault(): void {
    SystemUiThemeStore.reset();
  }
}
