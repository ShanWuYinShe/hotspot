import { DesignConstants } from '../../constants/DesignConstants';

export interface SystemSurfaceStyle {
  baseColor: string;
  overlayOpacity?: number;
  preferDarkContent?: boolean;
}

type SystemSurfaceListener = (style: SystemSurfaceStyle) => void;

export class SystemUiThemeStore {
  private static listeners: SystemSurfaceListener[] = [];
  private static currentStyle: SystemSurfaceStyle = {
    baseColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT,
    overlayOpacity: 1,
    preferDarkContent: true
  };

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
      baseColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT,
      overlayOpacity: 1,
      preferDarkContent: true
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
