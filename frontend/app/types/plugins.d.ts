/**
 * 插件系统全局类型定义
 * - 统一管理所有插件的类型定义
 * - 提供 TypeScript 智能提示和类型检查
 * - 确保插件 API 的类型安全
 */

import type { DeviceCapabilities } from "~/plugins/02.device-capabilities.client";

declare global {
  // 插件全局实例
  var $deviceCapabilities: DeviceCapabilities;
  var $smartCache: import("~/plugins/03.smart-cache.client").SmartCacheManager;
  var $performanceMonitor: import("~/plugins/05.performance-monitor.client").PerformanceMonitor;
  var $progressiveEnhancement: import("~/plugins/06.progressive-enhancement.client").ProgressiveEnhancement;
  var $systemIntegration: import("~/plugins/07.system-integration.client").SystemIntegrationManager;

  // Nuxt 插件注入的类型
  interface NuxtApp {
    $deviceCapabilities: DeviceCapabilities;
    $smartCache: import("~/plugins/03.smart-cache.client").SmartCacheManager;
    $performanceMonitor: import("~/plugins/05.performance-monitor.client").PerformanceMonitor;
    $progressiveEnhancement: import("~/plugins/06.progressive-enhancement.client").ProgressiveEnhancement;
    $systemIntegration: import("~/plugins/07.system-integration.client").SystemIntegrationManager;

    // 便捷方法
    $canUse: (feature: keyof DeviceCapabilities) => boolean;
    $isHighDPI: () => boolean;
    $isSlowNetwork: () => boolean;
    $hasFastNetwork: () => boolean;

    $cache: {
      set: <T>(key: string, data: T, options?: any) => Promise<void>;
      get: <T>(key: string) => Promise<T | null>;
      stats: () => any;
    };

    $getPerformanceReport: () => any;
    $markPerformance: (name: string, detail?: any) => void;

    $canUseFeature: (feature: string) => boolean;
    $getPerformanceProfile: () => "high" | "medium" | "low";

    $installPWA: () => Promise<boolean>;
    $updatePWA: () => void;
    $share: (data: any) => Promise<boolean>;
    $copyText: (text: string) => Promise<boolean>;
    $notify: (title: string, options?: NotificationOptions) => Promise<boolean>;
    $setBadge: (count?: number) => Promise<boolean>;
    $pickFiles: (options?: any) => Promise<File[]>;
    $getCamera: (
      constraints?: MediaStreamConstraints,
    ) => Promise<MediaStream | null>;
    $getMicrophone: (
      constraints?: MediaStreamConstraints,
    ) => Promise<MediaStream | null>;
    $shareScreen: () => Promise<MediaStream | null>;
  }
}

// 自定义事件类型
declare global {
  interface WindowEventMap {
    // PWA 事件
    pwaInstallable: CustomEvent<{ canInstall: boolean }>;
    pwaInstalled: CustomEvent;
    pwaUpdateAvailable: CustomEvent;

    // 系统事件
    filesDropped: CustomEvent<{
      files: File[];
      position: { x: number; y: number };
    }>;
    lowMemory: CustomEvent;

    // 网络事件
    connectionChange: CustomEvent<{ effectiveType: string; downlink: number }>;
  }
}

declare global {
  function definePageMeta(meta: any): void;
}

export {};
