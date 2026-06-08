export interface SystemSurfaceStyle {
  baseColor: string;
  overlayOpacity?: number;
  preferDarkContent?: boolean;
}

type SystemSurfaceListener = (style: SystemSurfaceStyle) => void;

const DEFAULT_SYSTEM_SURFACE_STYLE: SystemSurfaceStyle = {
  baseColor: '#D0CBC3',
  overlayOpacity: 1,
  preferDarkContent: true
};

export class SystemUiThemeStore {
  private static listeners: SystemSurfaceListener[] = [];
  private static currentStyle: SystemSurfaceStyle = DEFAULT_SYSTEM_SURFACE_STYLE;

  static getStyle(): SystemSurfaceStyle {
    return {
      baseColor: SystemUiThemeStore.currentStyle.baseColor,
      overlayOpacity: SystemUiThemeStore.currentStyle.overlayOpacity,
      preferDarkContent: SystemUiThemeStore.currentStyle.preferDarkContent
    };
  }

  static setStyle(style: SystemSurfaceStyle): void {
    SystemUiThemeStore.currentStyle = {
      baseColor: style.baseColor ?? SystemUiThemeStore.currentStyle.baseColor,
      overlayOpacity: style.overlayOpacity ?? 1,
      preferDarkContent: style.preferDarkContent ?? SystemUiThemeStore.currentStyle.preferDarkContent
    };
    SystemUiThemeStore.notify();
  }

  static reset(): void {
    SystemUiThemeStore.currentStyle = {
      baseColor: DEFAULT_SYSTEM_SURFACE_STYLE.baseColor,
      overlayOpacity: DEFAULT_SYSTEM_SURFACE_STYLE.overlayOpacity,
      preferDarkContent: DEFAULT_SYSTEM_SURFACE_STYLE.preferDarkContent
    };
    SystemUiThemeStore.notify();
  }

  static subscribe(listener: SystemSurfaceListener): () => void {
    SystemUiThemeStore.listeners.push(listener);
    return () => {
      SystemUiThemeStore.listeners = SystemUiThemeStore.listeners.filter((item) => item !== listener);
    };
  }

  private static notify(): void {
    const snapshot: SystemSurfaceStyle = {
      baseColor: SystemUiThemeStore.currentStyle.baseColor,
      overlayOpacity: SystemUiThemeStore.currentStyle.overlayOpacity,
      preferDarkContent: SystemUiThemeStore.currentStyle.preferDarkContent
    };
    for (const listener of SystemUiThemeStore.listeners) {
      listener(snapshot);
    }
  }
}
