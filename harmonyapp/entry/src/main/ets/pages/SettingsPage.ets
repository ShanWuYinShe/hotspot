/**
 * 设置页面
 * 提供应用设置和偏好配置
 */

import { GlobalStateManager, GlobalState } from '../common/state/GlobalStateManager';
import { PreferencesUtil } from '../common/storage/PreferencesUtil';
import { SystemToast } from '../components/SystemToast';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF16;
const TAG: string = 'SettingsPage';

type SettingsValue = boolean | string | number | null;

interface SettingsItem {
  title: string;
  subtitle?: string;
  type: 'switch' | 'select' | 'action' | 'info';
  key?: string;
  value?: SettingsValue;
  options?: string[];
  action?: () => void;
}

interface SettingsGroup {
  title: string;
  items: SettingsItem[];
}

@Entry
@Component
struct SettingsPage {
  @State globalState: GlobalState = GlobalStateManager.getInstance().getState();
  private stateManager = GlobalStateManager.getInstance();
  @State settings: Record<string, SettingsValue> = {};
  @State isLoading: boolean = false;

  // 设置项配置
  private settingsGroups: SettingsGroup[] = [
    {
      title: '通知设置',
      items: [
        {
          title: '推送通知',
          subtitle: '接收新消息和活动通知',
          type: 'switch',
          key: 'push_notifications',
          value: true
        },
        {
          title: '声音提醒',
          subtitle: '消息到达时播放提示音',
          type: 'switch',
          key: 'sound_notifications',
          value: true
        },
        {
          title: '振动提醒',
          subtitle: '消息到达时振动提醒',
          type: 'switch',
          key: 'vibration_notifications',
          value: true
        }
      ]
    },
    {
      title: '隐私设置',
      items: [
        {
          title: '位置共享',
          subtitle: '允许其他用户看到我的位置',
          type: 'switch',
          key: 'location_sharing',
          value: true
        },
        {
          title: '在线状态',
          subtitle: '显示我的在线状态',
          type: 'switch',
          key: 'online_status',
          value: true
        },
        {
          title: '搜索可见性',
          subtitle: '允许其他用户搜索到我',
          type: 'switch',
          key: 'search_visibility',
          value: true
        }
      ]
    },
    {
      title: '显示设置',
      items: [
        {
          title: '主题模式',
          subtitle: '选择应用主题',
          type: 'select',
          key: 'theme_mode',
          value: 'auto',
          options: ['auto', 'light', 'dark']
        },
        {
          title: '字体大小',
          subtitle: '调整应用字体大小',
          type: 'select',
          key: 'font_size',
          value: 'medium',
          options: ['small', 'medium', 'large']
        },
        {
          title: '地图类型',
          subtitle: '选择默认地图样式',
          type: 'select',
          key: 'map_type',
          value: 'standard',
          options: ['standard', 'satellite', 'terrain']
        }
      ]
    },
    {
      title: '数据管理',
      items: [
        {
          title: '缓存大小',
          subtitle: '查看应用缓存使用情况',
          type: 'info',
          value: '计算中...'
        },
        {
          title: '清理缓存',
          subtitle: '清除应用缓存数据',
          type: 'action',
          action: () => this.clearCache()
        },
        {
          title: '导出数据',
          subtitle: '导出我的数据',
          type: 'action',
          action: () => this.exportData()
        }
      ]
    },
    {
      title: '关于',
      items: [
        {
          title: '应用版本',
          subtitle: 'GeoTalk v1.0.0',
          type: 'info'
        },
        {
          title: '检查更新',
          subtitle: '查看是否有新版本',
          type: 'action',
          action: () => this.checkUpdate()
        },
        {
          title: '用户协议',
          subtitle: '查看用户服务协议',
          type: 'action',
          action: () => this.showUserAgreement()
        },
        {
          title: '隐私政策',
          subtitle: '查看隐私保护政策',
          type: 'action',
          action: () => this.showPrivacyPolicy()
        }
      ]
    }
  ];

  async aboutToAppear() {
    hilog.info(DOMAIN_NUMBER, TAG, '设置页面初始化');
    
    // 订阅状态变化
    this.stateManager.subscribe('settings_page', (state: GlobalState) => {
      this.globalState = state;
    });

    // 加载设置
    await this.loadSettings();
    
    // 计算缓存大小
    await this.calculateCacheSize();
  }

  aboutToDisappear() {
    // 取消订阅
    this.stateManager.unsubscribe('settings_page');
  }

  /**
   * 加载设置
   */
  private async loadSettings() {
    try {
      this.isLoading = true;
      
      // 从本地存储加载设置
      const savedSettings = await PreferencesUtil.getObject('app_settings') as Record<string, SettingsValue>;
      
      if (savedSettings) {
        this.settings = savedSettings;
        
        // 更新设置项的值
        this.settingsGroups.forEach(group => {
          group.items.forEach(item => {
            if (item.key && this.settings[item.key] !== undefined) {
              item.value = this.settings[item.key];
            }
          });
        });
      }
      
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载设置失败: ${error}`);
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * 保存设置
   */
  private async saveSettings() {
    try {
      await PreferencesUtil.setObject('app_settings', this.settings);
      hilog.info(DOMAIN_NUMBER, TAG, '设置已保存');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `保存设置失败: ${error}`);
      SystemToast.showError('保存设置失败', this.getUIContext());
    }
  }

  /**
   * 更新设置项
   */
  private async updateSetting(key: string, value: SettingsValue) {
    this.settings[key] = value;
    await this.saveSettings();
    
    // 应用设置变更
    this.applySettingChange(key, value);
  }

  /**
   * 应用设置变更
   */
  private applySettingChange(key: string, value: SettingsValue) {
    switch (key) {
      case 'theme_mode':
        // 应用主题变更
        SystemToast.showInfo(`主题已切换为: ${this.getThemeDisplayName(value as string)}`, this.getUIContext());
        break;
      case 'font_size':
        // 应用字体大小变更
        SystemToast.showInfo(`字体大小已调整为: ${this.getFontSizeDisplayName(value as string)}`, this.getUIContext());
        break;
      case 'location_sharing':
        // 位置共享设置变更
        SystemToast.showInfo((value as boolean) ? '已开启位置共享' : '已关闭位置共享', this.getUIContext());
        break;
      default:
        SystemToast.show('设置已更新', this.getUIContext());
        break;
    }
  }

  /**
   * 计算缓存大小
   */
  private async calculateCacheSize() {
    try {
      // 这里应该计算实际的缓存大小
      // 由于HarmonyOS的文件系统API限制，这里使用模拟数据
      setTimeout(() => {
        const cacheItem = this.settingsGroups
          .find(g => g.title === '数据管理')
          ?.items.find(i => i.title === '缓存大小');
        
        if (cacheItem) {
          cacheItem.value = '12.5 MB';
        }
      }, 1000);
      
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `计算缓存大小失败: ${error}`);
    }
  }

  /**
   * 清理缓存
   */
  private async clearCache(): Promise<void> {
    try {
      // 清理应用缓存
      // 这里应该清理实际的缓存数据
      
      SystemToast.show('正在清理缓存...', this.getUIContext());
      
      // 模拟清理过程
      setTimeout(() => {
        SystemToast.showSuccess('缓存清理完成', this.getUIContext());
        
        // 更新缓存大小显示
        const cacheItem = this.settingsGroups
          .find(g => g.title === '数据管理')
          ?.items.find(i => i.title === '缓存大小');
        
        if (cacheItem) {
          cacheItem.value = '0 MB';
        }
      }, 2000);
      
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `清理缓存失败: ${error}`);
      SystemToast.showError('清理缓存失败', this.getUIContext());
    }
  }

  /**
   * 导出数据
   */
  private async exportData(): Promise<void> {
    SystemToast.show('数据导出功能开发中...', this.getUIContext());
  }

  /**
   * 检查更新
   */
  private async checkUpdate(): Promise<void> {
    SystemToast.show('检查更新中...', this.getUIContext());
    
    // 模拟检查更新
    setTimeout(() => {
      SystemToast.showSuccess('当前已是最新版本', this.getUIContext());
    }, 2000);
  }

  /**
   * 显示用户协议
   */
  private showUserAgreement(): void {
    SystemToast.show('用户协议页面开发中...', this.getUIContext());
  }

  /**
   * 显示隐私政策
   */
  private showPrivacyPolicy(): void {
    SystemToast.show('隐私政策页面开发中...', this.getUIContext());
  }

  /**
   * 获取主题显示名称
   */
  private getThemeDisplayName(value: string): string {
    switch (value) {
      case 'auto': return '跟随系统';
      case 'light': return '浅色模式';
      case 'dark': return '深色模式';
      default: return value;
    }
  }

  /**
   * 获取字体大小显示名称
   */
  private getFontSizeDisplayName(value: string): string {
    switch (value) {
      case 'small': return '小';
      case 'medium': return '中';
      case 'large': return '大';
      default: return value;
    }
  }

  /**
   * 获取选项显示名称
   */
  private getOptionDisplayName(key: string, value: string): string {
    switch (key) {
      case 'theme_mode':
        return this.getThemeDisplayName(value);
      case 'font_size':
        return this.getFontSizeDisplayName(value);
      case 'map_type':
        switch (value) {
          case 'standard': return '标准';
          case 'satellite': return '卫星';
          case 'terrain': return '地形';
          default: return value;
        }
      default:
        return value;
    }
  }

  build() {
    Column() {
      // 顶部导航栏
      Row() {
        Button('←')
          .width(40)
          .height(40)
          .fontSize(20)
          .backgroundColor(Color.Transparent)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .onClick(() => {
            this.getUIContext().getRouter().back();
          })

        Text('设置')
          .fontSize(20)
          .fontWeight(FontWeight.Bold)
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        // 占位符，保持布局平衡
        Row()
          .width(40)
          .height(40)
      }
      .width('100%')
      .height(56)
      .padding({ left: 8, right: 8 })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // 设置列表
      if (this.isLoading) {
        Column() {
          LoadingProgress()
            .width(40)
            .height(40)
            .margin({ bottom: 16 })
          
          Text('加载设置中...')
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
        .alignItems(HorizontalAlign.Center)
      } else {
        List() {
          ForEach(this.settingsGroups, (group: SettingsGroup) => {
            // 分组标题
            ListItem() {
              Text(group.title)
                .fontSize(14)
                .fontColor($r('sys.color.ohos_id_color_emphasize'))
                .fontWeight(FontWeight.Medium)
                .padding({ left: 16, right: 16, top: 16, bottom: 8 })
            }

            // 分组项目
            ForEach(group.items, (item: SettingsItem) => {
              ListItem() {
                Row() {
                  Column() {
                    Text(item.title)
                      .fontSize(16)
                      .fontColor($r('sys.color.ohos_id_color_text_primary'))
                      .alignSelf(ItemAlign.Start)
                    
                    if (item.subtitle) {
                      Text(item.subtitle)
                        .fontSize(12)
                        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                        .margin({ top: 4 })
                        .alignSelf(ItemAlign.Start)
                    }
                  }
                  .alignItems(HorizontalAlign.Start)
                  .layoutWeight(1)

                  // 根据类型显示不同的控件
                  if (item.type === 'switch') {
                    Toggle({ type: ToggleType.Switch, isOn: (item.value as boolean) || false })
                      .onChange((isOn: boolean) => {
                        if (item.key) {
                          item.value = isOn;
                          this.updateSetting(item.key, isOn);
                        }
                      })
                  } else if (item.type === 'select') {
                    Row() {
                      Text(this.getOptionDisplayName(item.key || '', item.value as string || ''))
                        .fontSize(14)
                        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                        .margin({ right: 8 })
                      
                      Text('>')
                        .fontSize(16)
                        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                    }
                  } else if (item.type === 'info') {
                    Text(String(item.value || ''))
                      .fontSize(14)
                      .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                  } else if (item.type === 'action') {
                    Text('>')
                      .fontSize(16)
                      .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                  }
                }
                .width('100%')
                .padding(16)
                .onClick(() => {
                  if (item.type === 'action' && item.action) {
                    item.action();
                  } else if (item.type === 'select' && item.key && item.options) {
                    // 显示选择对话框
                    this.showSelectDialog(item);
                  }
                })
              }
              .backgroundColor($r('sys.color.ohos_id_color_background'))
            })

            // 分组间隔
            ListItem() {
              Row()
                .height(16)
            }
          })
        }
        .width('100%')
        .layoutWeight(1)
        .backgroundColor($r('sys.color.ohos_id_color_background_transparent'))
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_background_transparent'))
  }

  /**
   * 显示选择对话框
   */
  private showSelectDialog(item: SettingsItem) {
    if (!item.options || !item.key) return;

    // 这里应该显示选择对话框
    // 由于HarmonyOS的对话框API可能不同，这里使用Toast作为示例
    SystemToast.showInfo(`选择 ${item.title}: ${item.options.join(', ')}`, this.getUIContext());
  }
}


