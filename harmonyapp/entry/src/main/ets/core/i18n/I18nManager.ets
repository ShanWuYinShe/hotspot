/**
 * 国际化管理器
 * 提供多语言支持和本地化功能
 */

import { common } from '@kit.AbilityKit';
import { i18n } from '@kit.LocalizationKit';
import { buffer } from '@kit.ArkTS'; // Added for buffer
import { createLogger } from '../utils/LogManager';
import { ErrorHandler, ErrorType, ErrorLevel } from '../utils/ErrorHandler';
import { BaseSingleton } from '../base/Singleton';
import { EventBus } from '../events/EventBus';
import { UnifiedStorageUtil } from '../utils/UnifiedStorageUtil';

const logger = createLogger('I18nManager');

/**
 * 支持的语言枚举
 */
export enum SupportedLanguage {
  ZH_CN = 'zh-CN',    // 简体中文
  ZH_TW = 'zh-TW',    // 繁体中文
  EN_US = 'en-US',    // 英语（美国）
  EN_GB = 'en-GB',    // 英语（英国）
  JA_JP = 'ja-JP',    // 日语
  KO_KR = 'ko-KR',    // 韩语
  FR_FR = 'fr-FR',    // 法语
  DE_DE = 'de-DE',    // 德语
  ES_ES = 'es-ES',    // 西班牙语
  RU_RU = 'ru-RU'     // 俄语
}

/**
 * 语言信息
 */
export interface LanguageInfo {
  code: SupportedLanguage;
  name: string;
  nativeName: string;
  flag: string;
  rtl: boolean;           // 是否从右到左
}

/**
 * 语言信息实现类
 */
class LanguageInfoImpl implements LanguageInfo {
  code: SupportedLanguage;
  name: string;
  nativeName: string;
  flag: string;
  rtl: boolean;
  
  constructor(
    code: SupportedLanguage,
    name: string,
    nativeName: string,
    flag: string,
    rtl: boolean
  ) {
    this.code = code;
    this.name = name;
    this.nativeName = nativeName;
    this.flag = flag;
    this.rtl = rtl;
  }
}

/**
 * 翻译键值对
 */
export class TranslationMap {
  private data: Map<string, string | TranslationMap> = new Map();
  
  set(key: string, value: string | TranslationMap): void {
    this.data.set(key, value);
  }
  
  get(key: string): string | TranslationMap | undefined {
    return this.data.get(key);
  }
  
  has(key: string): boolean {
    return this.data.has(key);
  }
  
  keys(): IterableIterator<string> {
    return this.data.keys();
  }
}

/**
 * 格式化参数
 */
export class FormatParams {
  private data: Map<string, string | number | Date> = new Map();
  
  set(key: string, value: string | number | Date): void {
    this.data.set(key, value);
  }
  
  get(key: string): string | number | Date | undefined {
    return this.data.get(key);
  }
  
  has(key: string): boolean {
    return this.data.has(key);
  }
  
  keys(): IterableIterator<string> {
    return this.data.keys();
  }
  
  entries(): IterableIterator<[string, string | number | Date]> {
    return this.data.entries();
  }
}

/**
 * 语言变化事件数据接口
 */
interface LanguageChangeData {
  oldLanguage: SupportedLanguage;
  newLanguage: SupportedLanguage;
}

/**
 * 语言变化事件数据实现类
 */
class LanguageChangeDataImpl implements LanguageChangeData {
  oldLanguage: SupportedLanguage;
  newLanguage: SupportedLanguage;

  constructor(oldLanguage: SupportedLanguage, newLanguage: SupportedLanguage) {
    this.oldLanguage = oldLanguage;
    this.newLanguage = newLanguage;
  }
}

/**
 * I18n重新加载事件数据接口
 */
interface I18nReloadData {
  language: SupportedLanguage;
}

/**
 * I18n重新加载事件数据实现类
 */
class I18nReloadDataImpl implements I18nReloadData {
  language: SupportedLanguage;

  constructor(language: SupportedLanguage) {
    this.language = language;
  }
}

/**
 * 日期时间格式选项
 */
export interface DateTimeFormatOptions {
  year?: 'numeric' | '2-digit';
  month?: 'numeric' | '2-digit' | 'long' | 'short' | 'narrow';
  day?: 'numeric' | '2-digit';
  hour?: 'numeric' | '2-digit';
  minute?: 'numeric' | '2-digit';
  second?: 'numeric' | '2-digit';
  weekday?: 'long' | 'short' | 'narrow';
  timeZone?: string;
  hour12?: boolean;
}

/**
 * 数字格式选项
 */
export interface NumberFormatOptions {
  style?: 'decimal' | 'currency' | 'percent';
  currency?: string;
  minimumFractionDigits?: number;
  maximumFractionDigits?: number;
  useGrouping?: boolean;
}

/**
 * 国际化管理器
 */
export class I18nManager extends BaseSingleton {
  private static instance: I18nManager | null = null;
  private context: common.UIAbilityContext | null = null;
  private eventBus: EventBus = EventBus.getInstance();
  private storageUtil: UnifiedStorageUtil = UnifiedStorageUtil.getInstance();
  
  private currentLanguage: SupportedLanguage = SupportedLanguage.ZH_CN;
  private fallbackLanguage: SupportedLanguage = SupportedLanguage.EN_US;
  private translations: Map<SupportedLanguage, TranslationMap> = new Map();
  private isLoaded: boolean = false;

  // 支持的语言列表
  private readonly supportedLanguages: LanguageInfo[] = [
    new LanguageInfoImpl(
      SupportedLanguage.ZH_CN,
      'Chinese (Simplified)',
      '简体中文',
      '🇨🇳',
      false
    ),
    new LanguageInfoImpl(
      SupportedLanguage.ZH_TW,
      'Chinese (Traditional)',
      '繁體中文',
      '🇹🇼',
      false
    ),
    new LanguageInfoImpl(
      SupportedLanguage.EN_US,
      'English (US)',
      'English',
      '🇺🇸',
      false
    ),
    new LanguageInfoImpl(
      SupportedLanguage.EN_GB,
      'English (UK)',
      'English (UK)',
      '🇬🇧',
      false
    ),
    new LanguageInfoImpl(
      SupportedLanguage.JA_JP,
      'Japanese',
      '日本語',
      '🇯🇵',
      false
    ),
    new LanguageInfoImpl(
      SupportedLanguage.KO_KR,
      'Korean',
      '한국어',
      '🇰🇷',
      false
    ),
    new LanguageInfoImpl(
      SupportedLanguage.FR_FR,
      'French',
      'Français',
      '🇫🇷',
      false
    ),
    new LanguageInfoImpl(
      SupportedLanguage.DE_DE,
      'German',
      'Deutsch',
      '🇩🇪',
      false
    ),
    new LanguageInfoImpl(
      SupportedLanguage.ES_ES,
      'Spanish',
      'Español',
      '🇪🇸',
      false
    ),
    new LanguageInfoImpl(
      SupportedLanguage.RU_RU,
      'Russian',
      'Русский',
      '🇷🇺',
      false
    )
  ];

  // 内置翻译（核心功能）
  private readonly builtinTranslations: Map<SupportedLanguage, TranslationMap> = this.initializeBuiltinTranslations();

  /**
   * 初始化内置翻译
   */
  private initializeBuiltinTranslations(): Map<SupportedLanguage, TranslationMap> {
    const translations = new Map<SupportedLanguage, TranslationMap>();
    
    // 中文简体翻译
    const zhCnTranslations = new TranslationMap();
    const zhCnCommon = new TranslationMap();
    zhCnCommon.set('ok', '确定');
    zhCnCommon.set('cancel', '取消');
    zhCnCommon.set('confirm', '确认');
    zhCnCommon.set('delete', '删除');
    zhCnCommon.set('edit', '编辑');
    zhCnCommon.set('save', '保存');
    zhCnCommon.set('loading', '加载中...');
    zhCnCommon.set('error', '错误');
    zhCnCommon.set('success', '成功');
    zhCnCommon.set('warning', '警告');
    zhCnCommon.set('info', '信息');
    zhCnCommon.set('retry', '重试');
    zhCnCommon.set('refresh', '刷新');
    zhCnCommon.set('search', '搜索');
    zhCnCommon.set('filter', '筛选');
    zhCnCommon.set('sort', '排序');
    zhCnCommon.set('more', '更多');
    zhCnCommon.set('back', '返回');
    zhCnCommon.set('next', '下一步');
    zhCnCommon.set('previous', '上一步');
    zhCnCommon.set('finish', '完成');
    zhCnCommon.set('close', '关闭');
    zhCnTranslations.set('common', zhCnCommon);
    
    translations.set(SupportedLanguage.ZH_CN, zhCnTranslations);
    
    return translations;
  }

  // 内置翻译数据已经移动到 initializeBuiltinTranslations 方法中
  // 为了简化示例，这里只实现了中文简体的部分翻译
  // 实际应用中可以根据需要扩展更多语言和翻译内容
  
  // 所有翻译数据都已经被移除，以符合 ArkTS 要求
  // 实际应用中可以通过动态加载或配置文件来添加更多翻译
  // 这里仅保留基本的中文简体翻译作为示例

  private constructor() {
    super();
  }

  public static getInstance(): I18nManager {
    if (!I18nManager.instance) {
      I18nManager.instance = new I18nManager();
    }
    return I18nManager.instance;
  }

  public static async getInstanceAsync(): Promise<I18nManager> {
    const instance = I18nManager.getInstance();
    await instance.initialize();
    return instance;
  }

  protected async onInitialize(): Promise<void> {
    logger.info('国际化管理器初始化');
    const hasUserSetting = await this.loadSavedLanguage();
    if (!hasUserSetting) {
      await this.detectSystemLanguage();
    }
    await this.loadTranslations();
    this.isLoaded = true;
  }

  protected async onCleanup(): Promise<void> {
    logger.info('国际化管理器清理');
    this.translations.clear();
    this.context = null;
    I18nManager.instance = null;
  }

  /**
   * 设置上下文
   */
  public setContext(context: common.UIAbilityContext): void {
    this.context = context;
    logger.info('国际化管理器上下文已设置');
  }

  /**
   * 加载保存的语言设置
   */
  private async loadSavedLanguage(): Promise<boolean> {
    try {
      const savedLanguage = await this.storageUtil.getObject<SupportedLanguage>('user_language');

      if (savedLanguage && this.isSupportedLanguage(savedLanguage)) {
        this.currentLanguage = savedLanguage;
        logger.info(`加载保存的语言设置: ${savedLanguage}`);
        return true;
      }
      return false;
    } catch (error) {
      logger.error('加载保存的语言设置失败', error);
      return false;
    }
  }

  /**
   * 检测系统语言
   */
  private async detectSystemLanguage(): Promise<void> {
    try {
      // 获取系统语言
      const systemLocale = i18n.System.getSystemLanguage();
      const systemLanguage = this.mapSystemLocaleToSupported(systemLocale);

      if (systemLanguage) {
        logger.info(`检测到系统语言: ${systemLanguage} (原始值: ${systemLocale})`);
        this.currentLanguage = systemLanguage;
      }
    } catch (error) {
      logger.error('检测系统语言失败', error);
    }
  }

  /**
   * 映射系统区域设置到支持的语言
   */
  private mapSystemLocaleToSupported(systemLocale: string): SupportedLanguage | null {
    const localeMap: Record<string, SupportedLanguage> = {
      'zh-CN': SupportedLanguage.ZH_CN,
      'zh-Hans': SupportedLanguage.ZH_CN,
      'zh-TW': SupportedLanguage.ZH_TW,
      'zh-Hant': SupportedLanguage.ZH_TW,
      'en-US': SupportedLanguage.EN_US,
      'en-GB': SupportedLanguage.EN_GB,
      'en': SupportedLanguage.EN_US,
      'ja-JP': SupportedLanguage.JA_JP,
      'ja': SupportedLanguage.JA_JP,
      'ko-KR': SupportedLanguage.KO_KR,
      'ko': SupportedLanguage.KO_KR,
      'fr-FR': SupportedLanguage.FR_FR,
      'fr': SupportedLanguage.FR_FR,
      'de-DE': SupportedLanguage.DE_DE,
      'de': SupportedLanguage.DE_DE,
      'es-ES': SupportedLanguage.ES_ES,
      'es': SupportedLanguage.ES_ES,
      'ru-RU': SupportedLanguage.RU_RU,
      'ru': SupportedLanguage.RU_RU
    };

    return localeMap[systemLocale] || null;
  }

  /**
   * 加载翻译文件
   */
  private async loadTranslations(): Promise<void> {
    try {
      logger.info('加载翻译文件');

      // 加载内置翻译
      const builtinEntries = Array.from(this.builtinTranslations.entries());
      for (let i = 0; i < builtinEntries.length; i++) {
        const entry = builtinEntries[i];
        const language = entry[0];
        const translations = entry[1];
        this.translations.set(language, translations);
      }

      // 加载外部翻译文件（如果存在）
      await this.loadExternalTranslations();

      logger.info(`翻译文件加载完成，支持 ${this.translations.size} 种语言`);

    } catch (error) {
      logger.error('加载翻译文件失败', error);
      ErrorHandler.handle(error, 'I18nManager.loadTranslations', ErrorType.SYSTEM, ErrorLevel.MEDIUM);
    }
  }

  /**
   * 加载外部翻译文件
   */
  private async loadExternalTranslations(): Promise<void> {
    try {
      if (!this.context) {
        logger.warn('Context not available, skipping external translations');
        return;
      }

      // 尝试从 rawfile 加载翻译文件
      const resourceManager = this.context.resourceManager;
      const rawFileDir = 'i18n'; // 假设翻译文件放在 rawfile/i18n 目录下
      
      try {
        const files = await resourceManager.getRawFileList(rawFileDir);
        
        for (const file of files) {
          if (file.endsWith('.json')) {
            const content = await resourceManager.getRawFileContent(`${rawFileDir}/${file}`);
            const jsonStr = buffer.from(content).toString('utf-8');
            const translations = JSON.parse(jsonStr);
            
            // 解析文件名获取语言代码，例如 zh-CN.json
            const langCode = file.replace('.json', '');
            if (this.isSupportedLanguage(langCode as SupportedLanguage)) {
               this.mergeTranslations(langCode as SupportedLanguage, translations);
               logger.info(`Loaded external translations for ${langCode}`);
            }
          }
        }
      } catch (e) {
         // 目录可能不存在，忽略
         logger.debug(`No external translation directory found: ${rawFileDir}`);
      }

    } catch (error) {
      logger.warn('加载外部翻译文件失败', error);
    }
  }

  private mergeTranslations(language: SupportedLanguage, newTranslations: object): void {
      let existingMap = this.translations.get(language);
      if (!existingMap) {
          existingMap = new TranslationMap();
          this.translations.set(language, existingMap);
      }
      this.mergeRecursive(existingMap, newTranslations);
  }

  private mergeRecursive(map: TranslationMap, obj: object): void {
      const objRecord = obj as Record<string, Object>;
      const keys = Object.keys(objRecord);
      for (let i = 0; i < keys.length; i++) {
          const key = keys[i];
          const value = objRecord[key];
          if (value !== null && typeof value === 'object') {
              let childMap = map.get(key);
              if (!(childMap instanceof TranslationMap)) {
                  childMap = new TranslationMap();
                  map.set(key, childMap);
              }
              this.mergeRecursive(childMap as TranslationMap, value);
          } else if (typeof value === 'string') {
              map.set(key, value);
          }
      }
  }

  /**
   * 设置当前语言
   */
  public async setLanguage(language: SupportedLanguage): Promise<void> {
    try {
      if (!this.isSupportedLanguage(language)) {
        throw new Error(`不支持的语言: ${language}`);
      }

      const oldLanguage = this.currentLanguage;
      this.currentLanguage = language;

      // 保存语言设置
      await this.storageUtil.setObject('user_language', language);

      logger.info(`语言已切换: ${oldLanguage} -> ${language}`);

      // 触发语言变化事件
      const eventData = new LanguageChangeDataImpl(oldLanguage, language);
      this.eventBus.emit('languageChanged', eventData);

    } catch (error) {
      logger.error('设置语言失败', error);
      ErrorHandler.handle(error, 'I18nManager.setLanguage', ErrorType.SYSTEM, ErrorLevel.HIGH);
      throw new Error(String(error));
    }
  }

  /**
   * 获取当前语言
   */
  public getCurrentLanguage(): SupportedLanguage {
    return this.currentLanguage;
  }

  /**
   * 获取支持的语言列表
   */
  public getSupportedLanguages(): LanguageInfo[] {
    return [...this.supportedLanguages];
  }

  /**
   * 获取语言信息
   */
  public getLanguageInfo(language: SupportedLanguage): LanguageInfo | null {
    return this.supportedLanguages.find(lang => lang.code === language) || null;
  }

  /**
   * 翻译文本
   */
  public t(key: string, params?: FormatParams): string {
    try {
      if (!this.isLoaded) {
        logger.warn('翻译尚未加载完成，返回原始键值');
        return key;
      }

      // 尝试获取当前语言的翻译
      let translation = this.getTranslation(key, this.currentLanguage);
      
      // 如果当前语言没有翻译，尝试回退语言
      if (!translation && this.currentLanguage !== this.fallbackLanguage) {
        translation = this.getTranslation(key, this.fallbackLanguage);
      }

      // 如果仍然没有翻译，返回键值
      if (!translation) {
        logger.warn(`翻译缺失: ${key} (${this.currentLanguage})`);
        return key;
      }

      // 如果有参数，进行格式化
      if (params) {
        return this.formatTranslation(translation, params);
      }

      return translation;

    } catch (error) {
      logger.error(`翻译失败: ${key}`, error);
      return key;
    }
  }

  /**
   * 获取翻译
   */
  private getTranslation(key: string, language: SupportedLanguage): string | null {
    const translations = this.translations.get(language);
    if (!translations) {
      return null;
    }

    const keys = key.split('.');
    let current: Object | null = translations as Object;

    for (const k of keys) {
      if (current && typeof current === 'object') {
        const obj = current as Record<string, Object>;
        if (obj[k] !== undefined) {
          current = obj[k];
        } else {
          return null;
        }
      } else {
        return null;
      }
    }

    return typeof current === 'string' ? current : null;
  }

  /**
   * 格式化翻译文本
   */
  private formatTranslation(template: string, params: FormatParams): string {
    let result = template;

    const paramEntries = Array.from(params.entries());
    for (let i = 0; i < paramEntries.length; i++) {
      const entry = paramEntries[i];
      const key = entry[0];
      const value = entry[1];
      const placeholder = `{${key}}`;
      const replacement = String(value);
      result = result.replace(new RegExp(placeholder, 'g'), replacement);
    }

    return result;
  }

  /**
   * 格式化日期时间
   */
  public formatDateTime(
    date: Date,
    options?: DateTimeFormatOptions,
    language?: SupportedLanguage
  ): string {
    try {
      const locale = language || this.currentLanguage;
      const formatter = new Intl.DateTimeFormat(locale, options as any);
      return formatter.format(date);

    } catch (error) {
      logger.error('格式化日期时间失败', error);
      return date.toLocaleString();
    }
  }

  /**
   * 格式化数字
   */
  public formatNumber(
    number: number,
    options?: NumberFormatOptions,
    language?: SupportedLanguage
  ): string {
    try {
      const locale = language || this.currentLanguage;
      const formatter = new Intl.NumberFormat(locale, options as any);
      return formatter.format(number);

    } catch (error) {
      logger.error('格式化数字失败', error);
      return number.toString();
    }
  }

  /**
   * 格式化货币
   */
  public formatCurrency(
    amount: number,
    currency: string = 'CNY',
    language?: SupportedLanguage
  ): string {
    return this.formatNumber(amount, {
      style: 'currency',
      currency: currency
    }, language);
  }

  /**
   * 格式化百分比
   */
  public formatPercent(
    value: number,
    language?: SupportedLanguage
  ): string {
    return this.formatNumber(value, {
      style: 'percent'
    }, language);
  }

  /**
   * 获取相对时间
   */
  public getRelativeTime(date: Date, language?: SupportedLanguage): string {
    try {
      const locale = language || this.currentLanguage;
      const now = new Date();
      const diffMs = now.getTime() - date.getTime();
      const diffMinutes = Math.floor(diffMs / (1000 * 60));
      const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
      const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

      if (diffMinutes < 1) {
        return this.t('time.justNow');
      } else if (diffMinutes < 60) {
        const params = new FormatParams();
        params.set('count', diffMinutes as number);
        return this.t('time.minutesAgo', params);
      } else if (diffHours < 24) {
        const params = new FormatParams();
        params.set('count', diffHours as number);
        return this.t('time.hoursAgo', params);
      } else if (diffDays < 7) {
        const params = new FormatParams();
        params.set('count', diffDays as number);
        return this.t('time.daysAgo', params);
      } else {
        return this.formatDateTime(date, {
          year: 'numeric',
          month: 'short',
          day: 'numeric'
        }, locale);
      }

    } catch (error) {
      logger.error('获取相对时间失败', error);
      return date.toLocaleDateString();
    }
  }

  /**
   * 检查是否支持指定语言
   */
  private isSupportedLanguage(language: Object): boolean {
    const supportedValues = [
      SupportedLanguage.ZH_CN,
      SupportedLanguage.ZH_TW,
      SupportedLanguage.EN_US,
      SupportedLanguage.EN_GB,
      SupportedLanguage.JA_JP,
      SupportedLanguage.KO_KR,
      SupportedLanguage.FR_FR,
      SupportedLanguage.DE_DE,
      SupportedLanguage.ES_ES,
      SupportedLanguage.RU_RU
    ];
    return supportedValues.includes(language as SupportedLanguage);
  }

  /**
   * 获取当前语言的文本方向
   */
  public getTextDirection(): 'ltr' | 'rtl' {
    const languageInfo = this.getLanguageInfo(this.currentLanguage);
    return languageInfo?.rtl ? 'rtl' : 'ltr';
  }

  /**
   * 检查是否已加载完成
   */
  public getLoadedStatus(): boolean {
    return this.isLoaded;
  }

  /**
   * 重新加载翻译
   */
  public async reload(): Promise<void> {
    logger.info('重新加载国际化');
    this.isLoaded = false;
    this.translations.clear();
    await this.loadTranslations();
    this.isLoaded = true;
    
    const reloadData = new I18nReloadDataImpl(this.currentLanguage);
    this.eventBus.emit('i18nReloaded', reloadData);
  }
}
