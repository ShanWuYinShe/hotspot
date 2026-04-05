/**
 * 无障碍功能管理器
 * 提供完整的无障碍功能支持，包括屏幕阅读、高对比度、字体缩放等
 */
import { hilog } from '@kit.PerformanceAnalysisKit';
import { BusinessError } from '@kit.BasicServicesKit';
import { PreferencesUtil } from '../storage/PreferencesUtil';
import accessibility from '@ohos.accessibility';
// import config from '@ohos.accessibility.config'; // Module not available
import { bundleManager } from '@kit.AbilityKit';

export interface AccessibilityConfig {
  screenReaderEnabled: boolean;
  highContrastEnabled: boolean;
  largeTextEnabled: boolean;
  reduceMotionEnabled: boolean;
  voiceOverEnabled: boolean;
  magnificationEnabled: boolean;
  colorInversionEnabled: boolean;
  fontSize: 'small' | 'normal' | 'large' | 'extra_large';
  colorScheme: 'light' | 'dark' | 'high_contrast';
  animationDuration: 'normal' | 'slow' | 'disabled';
}

interface AccessibilityAttributes {
  label?: string;
  hint?: string;
  role?: string;
  value?: string;
  selected?: boolean;
  expanded?: boolean;
  disabled?: boolean;
}

interface AccessibilityAction {
  name: string;
  label: string;
  handler: () => void;
}

interface TouchTargetSize {
  width: number;
  height: number;
}

interface ColorContrast {
  normal: number;
  large: number;
}

interface AccessibilityReport {
  config: AccessibilityConfig;
  recommendations: string[];
  issues: string[];
}

interface RgbColor {
  r: number;
  g: number;
  b: number;
}

export interface AccessibilityEvent {
  type: 'focus' | 'click' | 'scroll' | 'text_change' | 'page_change';
  element: string;
  description: string;
  target?: string;
  data?: Record<string, Object>;
  timestamp: number;
}

export class AccessibilityManager {
  private static instance: AccessibilityManager;
  private config: AccessibilityConfig;
  private eventListeners: Map<string, ((event: AccessibilityEvent) => void)[]> = new Map();
  private readonly storageKey = 'accessibility_config';
  private bundleName: string = '';

  private constructor() {
    this.config = {
      screenReaderEnabled: false,
      highContrastEnabled: false,
      largeTextEnabled: false,
      reduceMotionEnabled: false,
      voiceOverEnabled: false,
      magnificationEnabled: false,
      colorInversionEnabled: false,
      fontSize: 'normal',
      colorScheme: 'light',
      animationDuration: 'normal'
    };
    
    this.loadConfig();
    this.detectSystemAccessibilitySettings();
    this.initBundleInfo();
  }

  private async initBundleInfo() {
    try {
      const bundleInfo = await bundleManager.getBundleInfoForSelf(bundleManager.BundleFlag.GET_BUNDLE_INFO_DEFAULT);
      this.bundleName = bundleInfo.name;
    } catch (error) {
      hilog.error(0x0000, 'AccessibilityManager', `获取BundleInfo失败: ${JSON.stringify(error)}`);
    }
  }

  public static getInstance(): AccessibilityManager {
    if (!AccessibilityManager.instance) {
      AccessibilityManager.instance = new AccessibilityManager();
    }
    return AccessibilityManager.instance;
  }

  /**
   * 检测系统无障碍设置
   */
  private async detectSystemAccessibilitySettings(): Promise<void> {
    try {
      // 检测系统是否启用了无障碍服务
      const systemSettings = await this.getSystemAccessibilitySettings();
      
      if (systemSettings.screenReaderEnabled) {
        this.config.screenReaderEnabled = true;
        this.config.voiceOverEnabled = true;
      }
      
      if (systemSettings.highContrastEnabled) {
        this.config.highContrastEnabled = true;
        this.config.colorScheme = 'high_contrast';
      }
      
      if (systemSettings.largeTextEnabled) {
        this.config.largeTextEnabled = true;
        this.config.fontSize = 'large';
      }
      
      if (systemSettings.reduceMotionEnabled) {
        this.config.reduceMotionEnabled = true;
        this.config.animationDuration = 'disabled';
      }

      if (systemSettings.colorInversionEnabled) {
        this.config.colorInversionEnabled = true;
      }
      
      await this.saveConfig();
      
      hilog.info(0x0000, 'AccessibilityManager', '系统无障碍设置检测完成');
    } catch (error) {
      hilog.warn(0x0000, 'AccessibilityManager', `检测系统无障碍设置失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 获取系统无障碍设置
   */
  private async getSystemAccessibilitySettings(): Promise<Partial<AccessibilityConfig>> {
    const settings: Partial<AccessibilityConfig> = {};
    try {
      // @ohos.accessibility.config module not available - commented out
      // // 获取高对比度文本状态
      // settings.highContrastEnabled = await config.highContrastText.get();
      // 
      // // 获取颜色反转状态
      // settings.colorInversionEnabled = await config.invertColor.get();
      // 
      // // 获取动画开启状态 (animationOff 为 true 表示减少动画)
      // settings.reduceMotionEnabled = await config.animationOff.get();
      // 
      // // 检测屏幕阅读器是否开启
      // // isOpenTouchGuide deprecated, use config.touchGuideState
      // settings.screenReaderEnabled = await config.touchGuideState.get();
      // 
      // // 这里的 largeTextEnabled 没有直接的 API，通常由 system.configuration 获取 fontScale
      // try {
      //   const configuration = await config.get();
      //   if (configuration && configuration.fontScale) {
      //      settings.largeTextEnabled = configuration.fontScale > 1.0;
      //   }
      // } catch (e) {
      //   // 忽略获取配置失败
      // }
      
    } catch (error) {
      hilog.error(0x0000, 'AccessibilityManager', `获取系统无障碍设置失败: ${JSON.stringify(error)}`);
    }
    return settings;
  }

  /**
   * 获取当前无障碍配置
   */
  public getConfig(): AccessibilityConfig {
    const configCopy: AccessibilityConfig = {
      screenReaderEnabled: this.config.screenReaderEnabled,
      highContrastEnabled: this.config.highContrastEnabled,
      largeTextEnabled: this.config.largeTextEnabled,
      reduceMotionEnabled: this.config.reduceMotionEnabled,
      voiceOverEnabled: this.config.voiceOverEnabled,
      magnificationEnabled: this.config.magnificationEnabled,
      colorInversionEnabled: this.config.colorInversionEnabled,
      fontSize: this.config.fontSize,
      colorScheme: this.config.colorScheme,
      animationDuration: this.config.animationDuration
    };
    return configCopy;
  }

  /**
   * 更新无障碍配置
   */
  public async updateConfig(updates: Partial<AccessibilityConfig>): Promise<void> {
    const oldConfig: AccessibilityConfig = {
      screenReaderEnabled: this.config.screenReaderEnabled,
      highContrastEnabled: this.config.highContrastEnabled,
      largeTextEnabled: this.config.largeTextEnabled,
      reduceMotionEnabled: this.config.reduceMotionEnabled,
      voiceOverEnabled: this.config.voiceOverEnabled,
      magnificationEnabled: this.config.magnificationEnabled,
      colorInversionEnabled: this.config.colorInversionEnabled,
      fontSize: this.config.fontSize,
      colorScheme: this.config.colorScheme,
      animationDuration: this.config.animationDuration
    };
    
    // 手动合并配置
    this.config = {
      screenReaderEnabled: updates.screenReaderEnabled !== undefined ? updates.screenReaderEnabled : this.config.screenReaderEnabled,
      highContrastEnabled: updates.highContrastEnabled !== undefined ? updates.highContrastEnabled : this.config.highContrastEnabled,
      largeTextEnabled: updates.largeTextEnabled !== undefined ? updates.largeTextEnabled : this.config.largeTextEnabled,
      reduceMotionEnabled: updates.reduceMotionEnabled !== undefined ? updates.reduceMotionEnabled : this.config.reduceMotionEnabled,
      voiceOverEnabled: updates.voiceOverEnabled !== undefined ? updates.voiceOverEnabled : this.config.voiceOverEnabled,
      magnificationEnabled: updates.magnificationEnabled !== undefined ? updates.magnificationEnabled : this.config.magnificationEnabled,
      colorInversionEnabled: updates.colorInversionEnabled !== undefined ? updates.colorInversionEnabled : this.config.colorInversionEnabled,
      fontSize: updates.fontSize !== undefined ? updates.fontSize : this.config.fontSize,
      colorScheme: updates.colorScheme !== undefined ? updates.colorScheme : this.config.colorScheme,
      animationDuration: updates.animationDuration !== undefined ? updates.animationDuration : this.config.animationDuration
    };
    
    // 应用配置变更
    await this.applyConfigChanges(oldConfig, this.config);
    
    // 保存配置
    await this.saveConfig();
    
    // 触发配置变更事件
    this.emitEvent({
      type: 'page_change',
      element: 'accessibility_config',
      description: '无障碍配置已更新',
      timestamp: Date.now()
    });
    
    hilog.info(0x0000, 'AccessibilityManager', `无障碍配置已更新: ${JSON.stringify(updates)}`);
  }

  /**
   * 应用配置变更
   */
  private async applyConfigChanges(oldConfig: AccessibilityConfig, newConfig: AccessibilityConfig): Promise<void> {
    // 字体大小变更
    if (oldConfig.fontSize !== newConfig.fontSize) {
      await this.applyFontSizeChange(newConfig.fontSize);
    }
    
    // 颜色方案变更
    if (oldConfig.colorScheme !== newConfig.colorScheme) {
      await this.applyColorSchemeChange(newConfig.colorScheme);
    }
    
    // 动画设置变更
    if (oldConfig.animationDuration !== newConfig.animationDuration) {
      await this.applyAnimationDurationChange(newConfig.animationDuration);
    }
    
    // 屏幕阅读器变更
    if (oldConfig.screenReaderEnabled !== newConfig.screenReaderEnabled) {
      await this.applyScreenReaderChange(newConfig.screenReaderEnabled);
    }
  }

  /**
   * 应用字体大小变更
   */
  private async applyFontSizeChange(fontSize: string): Promise<void> {
    try {
      // 更新 AppStorage 中的字体设置，供 UI 组件订阅使用
      AppStorage.setOrCreate('accessibility_fontSize', fontSize);
      
      const fontSizeMap: Record<string, string> = {
        'small': '0.875rem',
        'normal': '1rem',
        'large': '1.25rem',
        'extra_large': '1.5rem'
      };
      
      const targetSize = fontSizeMap[fontSize] || '1rem';
      hilog.info(0x0000, 'AccessibilityManager', `应用字体大小变更: ${fontSize} -> ${targetSize}`);
    } catch (error) {
      hilog.error(0x0000, 'AccessibilityManager', `应用字体大小变更失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 应用颜色方案变更
   */
  private async applyColorSchemeChange(colorScheme: string): Promise<void> {
    try {
      // 更新 AppStorage 中的颜色方案设置
      AppStorage.setOrCreate('accessibility_colorScheme', colorScheme);
      hilog.info(0x0000, 'AccessibilityManager', `应用颜色方案变更: ${colorScheme}`);
    } catch (error) {
      hilog.error(0x0000, 'AccessibilityManager', `应用颜色方案变更失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 应用动画持续时间变更
   */
  private async applyAnimationDurationChange(animationDuration: string): Promise<void> {
    try {
      // 更新 AppStorage 中的动画持续时间设置
      AppStorage.setOrCreate('accessibility_animationDuration', animationDuration);
      
      const durationMap: Record<string, number> = {
        'normal': 1.0,
        'slow': 2.0,
        'disabled': 0.0
      };
      
      const targetDuration = durationMap[animationDuration] || 1.0;
      hilog.info(0x0000, 'AccessibilityManager', `应用动画持续时间变更: ${animationDuration} -> ${targetDuration}x`);
    } catch (error) {
      hilog.error(0x0000, 'AccessibilityManager', `应用动画持续时间变更失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 应用屏幕阅读器变更
   */
  private async applyScreenReaderChange(enabled: boolean): Promise<void> {
    try {
      if (enabled) {
        // 启用屏幕阅读器功能
        hilog.info(0x0000, 'AccessibilityManager', '启用屏幕阅读器功能');
      } else {
        // 禁用屏幕阅读器功能
        hilog.info(0x0000, 'AccessibilityManager', '禁用屏幕阅读器功能');
      }
    } catch (error) {
      hilog.error(0x0000, 'AccessibilityManager', `应用屏幕阅读器变更失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 语音播报文本
   */
  public async announceText(text: string, priority: 'low' | 'medium' | 'high' = 'medium'): Promise<void> {
    if (!this.config.screenReaderEnabled && !this.config.voiceOverEnabled) {
      return;
    }
    
    try {
      if (!this.bundleName) {
        await this.initBundleInfo();
      }

      // 发送无障碍事件进行语音播报
      const eventInfo: accessibility.EventInfo = {
        type: 'announce' as accessibility.EventType,
        bundleName: this.bundleName,
        triggerAction: 'common',
        textAnnouncedForAccessibility: text,
      };
      
      await accessibility.sendAccessibilityEvent(eventInfo);
      
      hilog.info(0x0000, 'AccessibilityManager', `语音播报 [${priority}]: ${text}`);
      
      // 触发内部事件
      this.emitEvent({
        type: 'text_change',
        element: 'voice_announcement',
        description: text,
        timestamp: Date.now()
      });
    } catch (error) {
      hilog.error(0x0000, 'AccessibilityManager', `语音播报失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 设置元素的无障碍属性
   */
  public setAccessibilityAttributes(elementId: string, attributes: AccessibilityAttributes): void {
    try {
      // 注意：HarmonyOS ArkUI 声明式开发中，无障碍属性通常直接绑定在组件上
      // 动态设置任意元素的属性可能不支持，建议使用状态变量绑定组件属性
      hilog.info(0x0000, 'AccessibilityManager', `请求设置无障碍属性 (仅日志): ${elementId}: ${JSON.stringify(attributes)}`);
    } catch (error) {
      hilog.error(0x0000, 'AccessibilityManager', `设置无障碍属性失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 焦点管理
   */
  public async focusElement(elementId: string, announce: boolean = true): Promise<void> {
    try {
      // 调用 HarmonyOS API 设置焦点
      focusControl.requestFocus(elementId);
      hilog.info(0x0000, 'AccessibilityManager', `设置焦点到元素: ${elementId}`);
      
      if (announce && (this.config.screenReaderEnabled || this.config.voiceOverEnabled)) {
        await this.announceText(`焦点移动到 ${elementId}`);
      }
      
      // 触发焦点事件
      const event: AccessibilityEvent = {
        type: 'focus',
        element: elementId,
        description: `焦点移动到 ${elementId}`,
        timestamp: Date.now()
      };
      this.emitEvent(event);
    } catch (error) {
      hilog.error(0x0000, 'AccessibilityManager', `设置焦点失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 获取推荐的触摸目标大小
   */
  public getRecommendedTouchTargetSize(): TouchTargetSize {
    // 根据无障碍指南，触摸目标最小应为 44dp x 44dp
    const baseSize = 44;
    
    // 根据字体大小调整
    let multiplier = 1.0;
    switch (this.config.fontSize) {
      case 'large':
        multiplier = 1.25;
        break;
      case 'extra_large':
        multiplier = 1.5;
        break;
    }
    
    const size = Math.round(baseSize * multiplier);
    const result: TouchTargetSize = { width: size, height: size };
    return result;
  }

  /**
   * 获取推荐的颜色对比度
   */
  public getRecommendedColorContrast(): ColorContrast {
    if (this.config.highContrastEnabled) {
      const aaaLevel: ColorContrast = { normal: 7.0, large: 4.5 }; // AAA 级别
      return aaaLevel;
    } else {
      const aaLevel: ColorContrast = { normal: 4.5, large: 3.0 }; // AA 级别
      return aaLevel;
    }
  }

  /**
   * 检查颜色对比度是否符合要求
   */
  public checkColorContrast(foreground: string, background: string, isLargeText: boolean = false): boolean {
    try {
      // 实现真实的颜色对比度计算算法
      const contrast = this.calculateColorContrast(foreground, background);
      const requiredContrast = this.getRecommendedColorContrast();
      const threshold = isLargeText ? requiredContrast.large : requiredContrast.normal;
      
      return contrast >= threshold;
    } catch (error) {
      hilog.error(0x0000, 'AccessibilityManager', '颜色对比度检查失败，返回false');
      return false;
    }
  }

  /**
   * 计算两个颜色之间的对比度
   */
  private calculateColorContrast(foreground: string, background: string): number {
    try {
      // 将颜色转换为RGB值
      const fgRgb = this.hexToRgb(foreground);
      const bgRgb = this.hexToRgb(background);
      
      if (!fgRgb || !bgRgb) {
        return 0;
      }
      
      // 计算相对亮度
      const fgLuminance = this.getRelativeLuminance(fgRgb);
      const bgLuminance = this.getRelativeLuminance(bgRgb);
      
      // 计算对比度
      const lighter = Math.max(fgLuminance, bgLuminance);
      const darker = Math.min(fgLuminance, bgLuminance);
      
      return (lighter + 0.05) / (darker + 0.05);
    } catch (error) {
      hilog.error(0x0000, 'AccessibilityManager', '计算颜色对比度失败');
      return 0;
    }
  }

  /**
   * 将十六进制颜色转换为RGB
   */
  private hexToRgb(hex: string): RgbColor | null {
    const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
    return result ? {
      r: parseInt(result[1], 16),
      g: parseInt(result[2], 16),
      b: parseInt(result[3], 16)
    } as RgbColor : null;
  }

  /**
   * 计算相对亮度
   */
  private getRelativeLuminance(rgb: RgbColor): number {
    const rsRGB = rgb.r / 255;
    const gsRGB = rgb.g / 255;
    const bsRGB = rgb.b / 255;

    const r = rsRGB <= 0.03928 ? rsRGB / 12.92 : Math.pow((rsRGB + 0.055) / 1.055, 2.4);
    const g = gsRGB <= 0.03928 ? gsRGB / 12.92 : Math.pow((gsRGB + 0.055) / 1.055, 2.4);
    const b = bsRGB <= 0.03928 ? bsRGB / 12.92 : Math.pow((bsRGB + 0.055) / 1.055, 2.4);

    return 0.2126 * r + 0.7152 * g + 0.0722 * b;
  }

  /**
   * 获取动画持续时间倍数
   */
  public getAnimationDurationMultiplier(): number {
    switch (this.config.animationDuration) {
      case 'slow':
        return 2.0;
      case 'disabled':
        return 0.0;
      default:
        return 1.0;
    }
  }

  /**
   * 检查是否应该减少动画
   */
  public shouldReduceMotion(): boolean {
    return this.config.reduceMotionEnabled || this.config.animationDuration === 'disabled';
  }

  /**
   * 注册事件监听器
   */
  public addEventListener(eventType: string, listener: (event: AccessibilityEvent) => void): void {
    if (!this.eventListeners.has(eventType)) {
      this.eventListeners.set(eventType, []);
    }
    this.eventListeners.get(eventType)!.push(listener);
  }

  /**
   * 移除事件监听器
   */
  public removeEventListener(eventType: string, listener: (event: AccessibilityEvent) => void): void {
    const listeners = this.eventListeners.get(eventType);
    if (listeners) {
      const index = listeners.indexOf(listener);
      if (index > -1) {
        listeners.splice(index, 1);
      }
    }
  }

  /**
   * 触发事件
   */
  private emitEvent(event: AccessibilityEvent): void {
    const listeners = this.eventListeners.get(event.type);
    if (listeners) {
      listeners.forEach(listener => {
        try {
          listener(event);
        } catch (error) {
          hilog.error(0x0000, 'AccessibilityManager', `事件监听器执行失败: ${JSON.stringify(error)}`);
        }
      });
    }
  }

  /**
   * 生成无障碍报告
   */
  public generateAccessibilityReport(): AccessibilityReport {
    const recommendations: string[] = [];
    const issues: string[] = [];
    
    // 检查配置并生成建议
    if (!this.config.screenReaderEnabled && !this.config.voiceOverEnabled) {
      recommendations.push('考虑启用屏幕阅读器以获得更好的无障碍体验');
    }
    
    if (this.config.fontSize === 'small') {
      recommendations.push('建议使用更大的字体以提高可读性');
    }
    
    if (!this.config.highContrastEnabled && this.config.colorScheme === 'light') {
      recommendations.push('在光线不足的环境中考虑使用深色模式或高对比度模式');
    }
    
    if (!this.config.reduceMotionEnabled && this.config.animationDuration === 'normal') {
      recommendations.push('如果您对动画敏感，可以考虑减少动画效果');
    }
    
    const report: AccessibilityReport = {
      config: this.config,
      recommendations,
      issues
    };
    return report;
  }

  /**
   * 保存配置到本地存储
   */
  private async saveConfig(): Promise<void> {
    try {
      await PreferencesUtil.setObject(this.storageKey, this.config);
    } catch (error) {
      hilog.error(0x0000, 'AccessibilityManager', `保存无障碍配置失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 从本地存储加载配置
   */
  private async loadConfig(): Promise<void> {
    try {
      const savedConfig = await PreferencesUtil.getObject<AccessibilityConfig>(this.storageKey);
      if (savedConfig) {
        // 手动合并配置对象
        const mergedConfig: AccessibilityConfig = {
          screenReaderEnabled: savedConfig.screenReaderEnabled ?? this.config.screenReaderEnabled,
          highContrastEnabled: savedConfig.highContrastEnabled ?? this.config.highContrastEnabled,
          largeTextEnabled: savedConfig.largeTextEnabled ?? this.config.largeTextEnabled,
          reduceMotionEnabled: savedConfig.reduceMotionEnabled ?? this.config.reduceMotionEnabled,
          voiceOverEnabled: savedConfig.voiceOverEnabled ?? this.config.voiceOverEnabled,
          magnificationEnabled: savedConfig.magnificationEnabled ?? this.config.magnificationEnabled,
          colorInversionEnabled: savedConfig.colorInversionEnabled ?? this.config.colorInversionEnabled,
          fontSize: savedConfig.fontSize ?? this.config.fontSize,
          colorScheme: savedConfig.colorScheme ?? this.config.colorScheme,
          animationDuration: savedConfig.animationDuration ?? this.config.animationDuration
        };
        this.config = mergedConfig;
      }
    } catch (error) {
      hilog.warn(0x0000, 'AccessibilityManager', '加载无障碍配置失败，使用默认配置');
    }
  }

  /**
   * 重置配置为默认值
   */
  public async resetToDefaults(): Promise<void> {
    this.config = {
      screenReaderEnabled: false,
      highContrastEnabled: false,
      largeTextEnabled: false,
      reduceMotionEnabled: false,
      voiceOverEnabled: false,
      magnificationEnabled: false,
      colorInversionEnabled: false,
      fontSize: 'normal',
      colorScheme: 'light',
      animationDuration: 'normal'
    };
    
    await this.saveConfig();
    
    hilog.info(0x0000, 'AccessibilityManager', '无障碍配置已重置为默认值');
  }
}
