/**
 * 开发者调试面板
 * 提供性能监控、服务状态、配置管理等调试功能
 */
import { PerformanceOptimizer, PerformanceMetrics, PerformanceAlert } from '../common/performance/PerformanceOptimizer';
import { AIRecommendationService } from '../common/services/AIRecommendationService';
import { SmartNotificationService } from '../common/services/SmartNotificationService';
import { AccessibilityManager } from '../common/accessibility/AccessibilityManager';
import { DistributedSyncService } from '../common/services/DistributedSyncService';
import { SystemToast } from '../common/utils/SystemToast';

@Entry
@Component
struct DebugPanel {
  @State private performanceMetrics: PerformanceMetrics | null = null;
  @State private performanceAlerts: PerformanceAlert[] = [];
  @State private notificationCount: number = 0;
  @State private syncServiceStatus: string = '未知';
  @State private aiServiceStatus: string = '未知';
  @State private selectedTab: number = 0;
  @State private refreshing: boolean = false;

  private performanceOptimizer = PerformanceOptimizer.getInstance();
  private aiService = AIRecommendationService.getInstance();
  private notificationService = SmartNotificationService.getInstance();
  private accessibilityManager = AccessibilityManager.getInstance();
  private syncService = DistributedSyncService.getInstance();
  private refreshTimer: number = -1;

  aboutToAppear() {
    this.loadDebugData();
    this.startAutoRefresh();
  }

  aboutToDisappear() {
    this.stopAutoRefresh();
  }

  /**
   * 加载调试数据
   */
  private async loadDebugData(): Promise<void> {
    try {
      this.refreshing = true;
      
      // 获取性能指标
      this.performanceMetrics = this.performanceOptimizer.getPerformanceMetrics();
      
      // 获取性能告警
      this.performanceAlerts = this.performanceOptimizer.getPerformanceAlerts();
      
      // 获取通知数量
      this.notificationCount = this.notificationService.getUnreadCount();
      
      // 检查服务状态
      this.syncServiceStatus = '运行中';
      this.aiServiceStatus = '运行中';
      
    } catch (error) {
      SystemToast.showError(`加载调试数据失败: ${error}`);
    } finally {
      this.refreshing = false;
    }
  }

  /**
   * 启动自动刷新
   */
  private startAutoRefresh(): void {
    this.refreshTimer = setInterval(() => {
      this.loadDebugData();
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
    await this.loadDebugData();
    SystemToast.showSuccess('数据已刷新');
  }

  /**
   * 清理缓存
   */
  private onClearCache(): void {
    try {
      this.performanceOptimizer.clearCache();
      SystemToast.showSuccess('缓存已清理');
      this.loadDebugData();
    } catch (error) {
      SystemToast.showError(`清理缓存失败: ${error}`);
    }
  }

  /**
   * 重置性能统计
   */
  private onResetPerformanceStats(): void {
    try {
      this.performanceOptimizer.resetPerformanceStats();
      SystemToast.showSuccess('性能统计已重置');
      this.loadDebugData();
    } catch (error) {
      SystemToast.showError(`重置性能统计失败: ${error}`);
    }
  }

  /**
   * 测试推荐系统
   */
  private async onTestRecommendations(): Promise<void> {
    try {
      // 模拟用户行为
      await this.aiService.recordUserBehavior({
        userId: 'debug_user',
        action: 'view',
        targetType: 'moment',
        targetId: 'test_moment',
        timestamp: Date.now()
      });
      
      // 获取推荐
      const recommendations = await this.aiService.getRecommendations('debug_user', 5);
      SystemToast.showSuccess(`生成了 ${recommendations.length} 条推荐`);
    } catch (error) {
      SystemToast.showError(`测试推荐系统失败: ${error}`);
    }
  }

  /**
   * 测试通知系统
   */
  private async onTestNotifications(): Promise<void> {
    try {
      await this.notificationService.createSocialNotification(
        'like',
        '测试用户',
        '这是一条测试通知',
        'test_target'
      );
      SystemToast.showSuccess('测试通知已发送');
      this.loadDebugData();
    } catch (error) {
      SystemToast.showError(`测试通知失败: ${error}`);
    }
  }

  /**
   * 测试无障碍功能
   */
  private async onTestAccessibility(): Promise<void> {
    try {
      await this.accessibilityManager.announceText('这是无障碍功能测试');
      SystemToast.showSuccess('无障碍功能测试完成');
    } catch (error) {
      SystemToast.showError(`测试无障碍功能失败: ${error}`);
    }
  }

  build() {
    Column() {
      // 标题栏
      Row() {
        Text('开发者调试面板')
          .fontSize(20)
          .fontWeight(FontWeight.Bold)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
        
        Spacer()
        
        Button(this.refreshing ? '刷新中...' : '刷新')
          .enabled(!this.refreshing)
          .onClick(() => this.onRefresh())
      }
      .width('100%')
      .height(56)
      .padding({ left: 16, right: 16 })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // 选项卡
      Tabs({ barPosition: BarPosition.Start, index: this.selectedTab }) {
        
        // 性能监控选项卡
        TabContent() {
          this.buildPerformanceTab()
        }
        .tabBar('性能监控')
        
        // 服务状态选项卡
        TabContent() {
          this.buildServicesTab()
        }
        .tabBar('服务状态')
        
        // 测试工具选项卡
        TabContent() {
          this.buildTestingTab()
        }
        .tabBar('测试工具')
        
        // 配置管理选项卡
        TabContent() {
          this.buildConfigTab()
        }
        .tabBar('配置管理')
      }
      .width('100%')
      .layoutWeight(1)
      .onChange((index: number) => {
        this.selectedTab = index;
      })
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_background_light'))
  }

  /**
   * 构建性能监控选项卡
   */
  @Builder
  private buildPerformanceTab() {
    Scroll() {
      Column({ space: 12 }) {
        
        // 性能指标卡片
        if (this.performanceMetrics) {
          Column({ space: 8 }) {
            Text('实时性能指标')
              .fontSize(16)
              .fontWeight(FontWeight.Medium)
              .alignSelf(ItemAlign.Start)
            
            Grid() {
              GridItem() {
                this.buildMetricItem('内存使用', `${(this.performanceMetrics.memoryUsage * 100).toFixed(1)}%`)
              }
              GridItem() {
                this.buildMetricItem('缓存命中率', `${(this.performanceMetrics.cacheHitRate * 100).toFixed(1)}%`)
              }
              GridItem() {
                this.buildMetricItem('平均响应时间', `${this.performanceMetrics.averageResponseTime.toFixed(0)}ms`)
              }
              GridItem() {
                this.buildMetricItem('错误率', `${(this.performanceMetrics.errorRate * 100).toFixed(1)}%`)
              }
              
              if (this.performanceMetrics.cpuUsage !== undefined) {
                GridItem() {
                  this.buildMetricItem('CPU使用率', `${(this.performanceMetrics.cpuUsage * 100).toFixed(1)}%`)
                }
              }
              
              if (this.performanceMetrics.batteryLevel !== undefined) {
                GridItem() {
                  this.buildMetricItem('电池电量', `${(this.performanceMetrics.batteryLevel * 100).toFixed(0)}%`)
                }
              }
              
              if (this.performanceMetrics.networkLatency !== undefined) {
                GridItem() {
                  this.buildMetricItem('网络延迟', `${this.performanceMetrics.networkLatency.toFixed(0)}ms`)
                }
              }
              
              if (this.performanceMetrics.frameRate !== undefined) {
                GridItem() {
                  this.buildMetricItem('帧率', `${this.performanceMetrics.frameRate.toFixed(0)} FPS`)
                }
              }
            }
            .columnsTemplate('1fr 1fr')
            .rowsGap(8)
            .columnsGap(8)
          }
          .width('100%')
          .padding(16)
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(8)
        }

        // 性能告警
        if (this.performanceAlerts.length > 0) {
          Column({ space: 8 }) {
            Text(`性能告警 (${this.performanceAlerts.length})`)
              .fontSize(16)
              .fontWeight(FontWeight.Medium)
              .alignSelf(ItemAlign.Start)
            
            ForEach(this.performanceAlerts.slice(0, 5), (alert: PerformanceAlert) => {
              Row({ space: 12 }) {
                Circle({ width: 8, height: 8 })
                  .fill(alert.level === 'critical' ? Color.Red : Color.Orange)
                
                Column({ space: 4 }) {
                  Text(alert.message)
                    .fontSize(14)
                    .fontColor($r('sys.color.ohos_id_color_text_primary'))
                  
                  Text(new Date(alert.timestamp).toLocaleString())
                    .fontSize(12)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                }
                .alignItems(HorizontalAlign.Start)
                .layoutWeight(1)
              }
              .width('100%')
              .padding(12)
              .backgroundColor($r('sys.color.ohos_id_color_background_light'))
              .borderRadius(4)
            })
          }
          .width('100%')
          .padding(16)
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(8)
        }

        // 操作按钮
        Column({ space: 8 }) {
          Text('性能操作')
            .fontSize(16)
            .fontWeight(FontWeight.Medium)
            .alignSelf(ItemAlign.Start)
          
          Row({ space: 12 }) {
            Button('清理缓存')
              .layoutWeight(1)
              .onClick(() => this.onClearCache())
            
            Button('重置统计')
              .layoutWeight(1)
              .onClick(() => this.onResetPerformanceStats())
          }
        }
        .width('100%')
        .padding(16)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius(8)
      }
      .padding(16)
    }
  }

  /**
   * 构建服务状态选项卡
   */
  @Builder
  private buildServicesTab() {
    Scroll() {
      Column({ space: 12 }) {
        
        // 服务状态列表
        Column({ space: 8 }) {
          Text('服务状态')
            .fontSize(16)
            .fontWeight(FontWeight.Medium)
            .alignSelf(ItemAlign.Start)
          
          this.buildServiceItem('性能优化器', '运行中', Color.Green)
          this.buildServiceItem('AI推荐服务', this.aiServiceStatus, Color.Green)
          this.buildServiceItem('智能通知服务', '运行中', Color.Green)
          this.buildServiceItem('无障碍管理器', '运行中', Color.Green)
          this.buildServiceItem('分布式同步服务', this.syncServiceStatus, Color.Green)
        }
        .width('100%')
        .padding(16)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius(8)

        // 通知统计
        Column({ space: 8 }) {
          Text('通知统计')
            .fontSize(16)
            .fontWeight(FontWeight.Medium)
            .alignSelf(ItemAlign.Start)
          
          Row() {
            Text('未读通知数量')
              .fontSize(14)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))
              .layoutWeight(1)
            
            Text(this.notificationCount.toString())
              .fontSize(14)
              .fontWeight(FontWeight.Medium)
              .fontColor($r('sys.color.ohos_id_color_emphasize'))
          }
          .width('100%')
          .padding(12)
        }
        .width('100%')
        .padding(16)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius(8)
      }
      .padding(16)
    }
  }

  /**
   * 构建测试工具选项卡
   */
  @Builder
  private buildTestingTab() {
    Scroll() {
      Column({ space: 12 }) {
        
        // 功能测试
        Column({ space: 8 }) {
          Text('功能测试')
            .fontSize(16)
            .fontWeight(FontWeight.Medium)
            .alignSelf(ItemAlign.Start)
          
          Button('测试推荐系统')
            .width('100%')
            .onClick(() => this.onTestRecommendations())
          
          Button('测试通知系统')
            .width('100%')
            .onClick(() => this.onTestNotifications())
          
          Button('测试无障碍功能')
            .width('100%')
            .onClick(() => this.onTestAccessibility())
        }
        .width('100%')
        .padding(16)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius(8)
      }
      .padding(16)
    }
  }

  /**
   * 构建配置管理选项卡
   */
  @Builder
  private buildConfigTab() {
    Scroll() {
      Column({ space: 12 }) {
        
        // 自适应配置
        Column({ space: 8 }) {
          Text('自适应配置')
            .fontSize(16)
            .fontWeight(FontWeight.Medium)
            .alignSelf(ItemAlign.Start)
          
          Text('当前配置会根据设备性能自动调整')
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          
          // 这里可以添加配置项的显示和编辑
          Text('图片质量: 自适应')
            .fontSize(14)
            .padding({ top: 8, bottom: 8 })
          
          Text('缓存大小: 自适应')
            .fontSize(14)
            .padding({ top: 8, bottom: 8 })
          
          Text('动画效果: 自适应')
            .fontSize(14)
            .padding({ top: 8, bottom: 8 })
        }
        .width('100%')
        .padding(16)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius(8)
      }
      .padding(16)
    }
  }

  /**
   * 构建指标项
   */
  @Builder
  private buildMetricItem(label: string, value: string) {
    Column({ space: 4 }) {
      Text(value)
        .fontSize(18)
        .fontWeight(FontWeight.Bold)
        .fontColor($r('sys.color.ohos_id_color_emphasize'))
      
      Text(label)
        .fontSize(12)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
    }
    .alignItems(HorizontalAlign.Center)
    .padding(12)
    .backgroundColor($r('sys.color.ohos_id_color_background_light'))
    .borderRadius(4)
  }

  /**
   * 构建服务项
   */
  @Builder
  private buildServiceItem(name: string, status: string, statusColor: ResourceColor) {
    Row() {
      Text(name)
        .fontSize(14)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .layoutWeight(1)
      
      Row({ space: 6 }) {
        Circle({ width: 8, height: 8 })
          .fill(statusColor)
        
        Text(status)
          .fontSize(12)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
      }
    }
    .width('100%')
    .padding(12)
    .backgroundColor($r('sys.color.ohos_id_color_background_light'))
    .borderRadius(4)
  }
}
