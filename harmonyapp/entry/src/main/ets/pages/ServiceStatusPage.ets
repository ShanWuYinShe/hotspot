/**
 * 服务状态监控页面
 * 实时显示所有智能服务的运行状态和统计信息
 */
import { ServiceManager, ServiceStatus } from '../common/services/ServiceManager';
import { SystemIcon } from '../components/SystemIcon';
import { AppIcon } from '../components/SystemIcon';
import { SystemToast } from '../common/utils/SystemToast';
import { AccessibilityManager } from '../common/accessibility/AccessibilityManager';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF21;
const TAG: string = 'ServiceStatusPage';

@Entry
@Component
struct ServiceStatusPage {
  @State private serviceStatus: Map<string, ServiceStatus> = new Map();
  @State private serviceStats: any = null;
  @State private isRefreshing: boolean = false;
  @State private autoRefresh: boolean = true;
  @State private selectedService: string | null = null;

  private serviceManager = ServiceManager.getInstance();
  private accessibilityManager = AccessibilityManager.getInstance();
  private refreshTimer: number = -1;

  aboutToAppear() {
    this.loadServiceStatus();
    this.setupStatusListener();
    
    if (this.autoRefresh) {
      this.startAutoRefresh();
    }
  }

  aboutToDisappear() {
    this.stopAutoRefresh();
  }

  /**
   * 加载服务状态
   */
  private loadServiceStatus(): void {
    try {
      this.serviceStatus = this.serviceManager.getServiceStatus();
      this.serviceStats = this.serviceManager.getServiceStats();
      
      hilog.info(DOMAIN_NUMBER, TAG, `加载了 ${this.serviceStatus.size} 个服务状态`);
    } catch (error) {
      SystemToast.showError(`加载服务状态失败: ${error}`);
      hilog.error(DOMAIN_NUMBER, TAG, `加载服务状态失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 设置状态监听器
   */
  private setupStatusListener(): void {
    this.serviceManager.addStatusListener((status: Map<string, ServiceStatus>) => {
      this.serviceStatus = status;
      this.serviceStats = this.serviceManager.getServiceStats();
      
      // 无障碍播报状态变化
      if (this.serviceStats) {
        this.accessibilityManager.announceText(
          `服务状态更新：${this.serviceStats.running} 个服务运行中`
        );
      }
    });
  }

  /**
   * 启动自动刷新
   */
  private startAutoRefresh(): void {
    if (this.refreshTimer !== -1) return;
    
    this.refreshTimer = setInterval(() => {
      this.loadServiceStatus();
    }, 5000); // 每5秒刷新一次
  }

  /**
   * 停止自动刷新
   */
  private stopAutoRefresh(): void {
    if (this.refreshTimer !== -1) {
      clearInterval(this.refreshTimer);
      this.refreshTimer = -1;
    }
  }

  /**
   * 手动刷新
   */
  private async onRefresh(): Promise<void> {
    this.isRefreshing = true;
    this.loadServiceStatus();
    
    setTimeout(() => {
      this.isRefreshing = false;
      SystemToast.showSuccess('服务状态已刷新');
    }, 500);
  }

  /**
   * 切换自动刷新
   */
  private toggleAutoRefresh(): void {
    this.autoRefresh = !this.autoRefresh;
    
    if (this.autoRefresh) {
      this.startAutoRefresh();
    } else {
      this.stopAutoRefresh();
    }
    
    SystemToast.showInfo(`自动刷新已${this.autoRefresh ? '开启' : '关闭'}`);
  }

  /**
   * 重启服务
   */
  private async restartService(serviceName: string): Promise<void> {
    try {
      SystemToast.showInfo(`正在重启服务: ${serviceName}`);
      
      // 停止服务
      await this.serviceManager.stopService(serviceName);
      
      // 等待一秒后重新初始化
      setTimeout(async () => {
        await this.serviceManager.initializeServices();
        SystemToast.showSuccess(`服务 ${serviceName} 重启成功`);
        
        // 无障碍播报
        await this.accessibilityManager.announceText(`服务 ${serviceName} 已重启`);
      }, 1000);
      
    } catch (error) {
      SystemToast.showError(`重启服务失败: ${error}`);
    }
  }

  /**
   * 获取状态颜色
   */
  private getStatusColor(status: ServiceStatus['status']): ResourceColor {
    const colorMap = {
      'running': Color.Green,
      'stopped': Color.Gray,
      'error': Color.Red,
      'initializing': Color.Orange
    };
    return colorMap[status];
  }

  /**
   * 获取状态图标
   */
  private getStatusIcon(status: ServiceStatus['status']): AppIcon {
    const iconMap = {
      'running': AppIcon.CHECK,
      'stopped': AppIcon.PAUSE,
      'error': AppIcon.ERROR,
      'initializing': AppIcon.LOADING
    };
    return iconMap[status];
  }

  /**
   * 获取状态文本
   */
  private getStatusText(status: ServiceStatus['status']): string {
    const textMap = {
      'running': '运行中',
      'stopped': '已停止',
      'error': '错误',
      'initializing': '初始化中'
    };
    return textMap[status];
  }

  /**
   * 返回上一页
   */
  private onBack(): void {
    const routerInstance = this.getUIContext().getRouter();
    routerInstance.back();
  }

  build() {
    Column() {
      // 标题栏
      Row() {
        SystemIcon({
          icon: AppIcon.BACK,
          iconSize: 24,
          onIconClick: () => this.onBack()
        })

        Text('服务状态监控')
          .fontSize(20)
          .fontWeight(FontWeight.Bold)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .margin({ left: 16 })
          .layoutWeight(1)

        // 操作按钮
        Row({ space: 12 }) {
          SystemIcon({
            icon: this.autoRefresh ? AppIcon.PAUSE : AppIcon.PLAY,
            iconSize: 20,
            onIconClick: () => this.toggleAutoRefresh()
          })

          SystemIcon({
            icon: AppIcon.REFRESH,
            iconSize: 20,
            onIconClick: () => this.onRefresh()
          })
        }
      }
      .width('100%')
      .height(56)
      .padding({ left: 16, right: 16 })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // 服务统计概览
      if (this.serviceStats) {
        Row({ space: 16 }) {
          this.buildStatCard('总计', this.serviceStats.total.toString(), AppIcon.ALL, Color.Blue)
          this.buildStatCard('运行', this.serviceStats.running.toString(), AppIcon.CHECK, Color.Green)
          this.buildStatCard('错误', this.serviceStats.error.toString(), AppIcon.ERROR, Color.Red)
          this.buildStatCard('停止', this.serviceStats.stopped.toString(), AppIcon.PAUSE, Color.Gray)
        }
        .width('100%')
        .padding(16)
      }

      // 刷新状态指示
      if (this.isRefreshing) {
        Row({ space: 8 }) {
          LoadingProgress()
            .width(16)
            .height(16)
            .color($r('sys.color.ohos_id_color_emphasize'))
          
          Text('刷新中...')
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .padding({ bottom: 16 })
      }

      // 服务列表
      if (this.serviceStatus.size === 0) {
        Column() {
          SystemIcon({
            icon: AppIcon.EMPTY,
            iconSize: 64,
            iconColor: $r('sys.color.ohos_id_color_text_tertiary')
          })
          
          Text('暂无服务信息')
            .fontSize(16)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .margin({ top: 16 })
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
      } else {
        List({ space: 8 }) {
          ForEach(Array.from(this.serviceStatus.entries()), ([name, status]: [string, ServiceStatus]) => {
            ListItem() {
              this.buildServiceItem(name, status)
            }
          })
        }
        .width('100%')
        .layoutWeight(1)
        .padding({ left: 16, right: 16, bottom: 16 })
        .scrollBar(BarState.Auto)
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_background_light'))
    // 无障碍属性
    .accessibilityGroup(true)
    .accessibilityText('服务状态监控页面')
    .accessibilityDescription(this.serviceStats 
      ? `当前有${this.serviceStats.total}个服务，${this.serviceStats.running}个运行中`
      : '服务状态加载中')
  }

  /**
   * 构建统计卡片
   */
  @Builder
  private buildStatCard(title: string, value: string, icon: AppIcon, color: ResourceColor) {
    Column({ space: 8 }) {
      SystemIcon({
        icon: icon,
        iconSize: 24,
        iconColor: color
      })

      Text(value)
        .fontSize(20)
        .fontWeight(FontWeight.Bold)
        .fontColor(color)

      Text(title)
        .fontSize(12)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
    }
    .layoutWeight(1)
    .padding(12)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(8)
    .justifyContent(FlexAlign.Center)
    // 无障碍属性
    .accessibilityGroup(true)
    .accessibilityText(`${title}：${value}个服务`)
  }

  /**
   * 构建服务项
   */
  @Builder
  private buildServiceItem(name: string, status: ServiceStatus) {
    Column({ space: 12 }) {
      // 服务基本信息
      Row({ space: 12 }) {
        // 状态指示器
        Stack() {
          SystemIcon({
            icon: this.getStatusIcon(status.status),
            iconSize: 24,
            iconColor: this.getStatusColor(status.status)
          })

          // 如果是初始化状态，显示加载动画
          if (status.status === 'initializing') {
            LoadingProgress()
              .width(32)
              .height(32)
              .color(this.getStatusColor(status.status))
          }
        }
        .width(32)
        .height(32)

        // 服务信息
        Column({ space: 4 }) {
          Text(name)
            .fontSize(16)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .alignSelf(ItemAlign.Start)

          Text(this.getStatusText(status.status))
            .fontSize(14)
            .fontColor(this.getStatusColor(status.status))
            .alignSelf(ItemAlign.Start)

          Text(`最后更新: ${new Date(status.lastUpdate).toLocaleString()}`)
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
            .alignSelf(ItemAlign.Start)
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)

        // 操作按钮
        if (status.status === 'error' || status.status === 'stopped') {
          Button('重启')
            .type(ButtonType.Capsule)
            .fontSize(12)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .padding({ left: 12, right: 12, top: 4, bottom: 4 })
            .onClick(() => this.restartService(name))
        }
      }
      .width('100%')

      // 错误信息
      if (status.errorMessage) {
        Row({ space: 8 }) {
          SystemIcon({
            icon: AppIcon.WARNING,
            iconSize: 16,
            iconColor: Color.Orange
          })

          Text(status.errorMessage)
            .fontSize(12)
            .fontColor(Color.Red)
            .layoutWeight(1)
        }
        .width('100%')
        .padding(8)
        .backgroundColor($r('sys.color.ohos_id_color_background_light'))
        .borderRadius(4)
      }

      // 元数据信息（可展开）
      if (status.metadata && this.selectedService === name) {
        Column({ space: 4 }) {
          Text('详细信息')
            .fontSize(14)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .alignSelf(ItemAlign.Start)

          ForEach(Object.entries(status.metadata), ([key, value]) => {
            Row() {
              Text(`${key}:`)
                .fontSize(12)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                .width(100)

              Text(String(value))
                .fontSize(12)
                .fontColor($r('sys.color.ohos_id_color_text_primary'))
                .layoutWeight(1)
            }
            .width('100%')
          })
        }
        .width('100%')
        .padding(8)
        .backgroundColor($r('sys.color.ohos_id_color_background_light'))
        .borderRadius(4)
      }

      // 展开/收起按钮
      if (status.metadata) {
        Row() {
          Spacer()
          
          Button(this.selectedService === name ? '收起' : '详情')
            .type(ButtonType.Normal)
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_emphasize'))
            .backgroundColor(Color.Transparent)
            .onClick(() => {
              this.selectedService = this.selectedService === name ? null : name;
            })
        }
        .width('100%')
      }
    }
    .width('100%')
    .padding(16)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(8)
    // 无障碍属性
    .accessibilityGroup(true)
    .accessibilityText(`服务${name}，状态：${this.getStatusText(status.status)}`)
    .accessibilityDescription(status.errorMessage 
      ? `错误信息：${status.errorMessage}` 
      : `最后更新时间：${new Date(status.lastUpdate).toLocaleString()}`)
  }
}
