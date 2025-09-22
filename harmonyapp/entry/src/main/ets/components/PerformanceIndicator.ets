/**
 * 性能指示器组件
 * 实时显示应用性能指标和状态
 */
import { PerformanceOptimizer, PerformanceMetrics, PerformanceAlert } from '../common/performance/PerformanceOptimizer';
import { SystemIcon } from './SystemIcon';
import { AppIcon } from '../common/constants/AppIcon';
import { SystemToast } from '../common/utils/SystemToast';
import { AccessibilityManager } from '../common/accessibility/AccessibilityManager';

@Component
export struct PerformanceIndicator {
  @Prop showDetails: boolean = false;
  @Prop compact: boolean = false;
  @State private metrics: PerformanceMetrics | null = null;
  @State private alerts: PerformanceAlert[] = [];
  @State private isExpanded: boolean = false;
  @State private refreshing: boolean = false;

  private performanceOptimizer = PerformanceOptimizer.getInstance();
  private accessibilityManager = AccessibilityManager.getInstance();
  private updateTimer: number = -1;

  aboutToAppear() {
    this.loadMetrics();
    this.setupPerformanceListener();
    this.startAutoUpdate();
  }

  aboutToDisappear() {
    this.stopAutoUpdate();
  }

  /**
   * 加载性能指标
   */
  private loadMetrics(): void {
    try {
      this.metrics = this.performanceOptimizer.getPerformanceMetrics();
      this.alerts = this.performanceOptimizer.getPerformanceAlerts().slice(0, 3);
    } catch (error) {
      SystemToast.showError(`加载性能数据失败: ${error}`);
    }
  }

  /**
   * 设置性能监听器
   */
  private setupPerformanceListener(): void {
    this.performanceOptimizer.onPerformanceAlert((alert: PerformanceAlert) => {
      this.loadMetrics();
      
      // 无障碍播报严重告警
      if (alert.level === 'critical') {
        this.accessibilityManager.announceText(`严重性能告警：${alert.message}`, 'high');
      }
    });
  }

  /**
   * 启动自动更新
   */
  private startAutoUpdate(): void {
    this.updateTimer = setInterval(() => {
      this.loadMetrics();
    }, 3000); // 每3秒更新一次
  }

  /**
   * 停止自动更新
   */
  private stopAutoUpdate(): void {
    if (this.updateTimer !== -1) {
      clearInterval(this.updateTimer);
      this.updateTimer = -1;
    }
  }

  /**
   * 手动刷新
   */
  private async onRefresh(): Promise<void> {
    this.refreshing = true;
    this.loadMetrics();
    
    // 模拟刷新延迟
    setTimeout(() => {
      this.refreshing = false;
      SystemToast.showSuccess('性能数据已刷新');
    }, 500);
  }

  /**
   * 清理缓存
   */
  private onClearCache(): void {
    try {
      this.performanceOptimizer.clearCache();
      SystemToast.showSuccess('缓存已清理');
      this.loadMetrics();
      
      // 无障碍播报
      this.accessibilityManager.announceText('缓存已清理，性能可能有所改善');
    } catch (error) {
      SystemToast.showError(`清理缓存失败: ${error}`);
    }
  }

  /**
   * 获取性能等级
   */
  private getPerformanceLevel(): { level: string; color: ResourceColor; icon: AppIcon } {
    if (!this.metrics) {
      return { level: '未知', color: Color.Gray, icon: AppIcon.QUESTION };
    }

    const { memoryUsage, errorRate, averageResponseTime } = this.metrics;
    
    // 综合评分
    let score = 100;
    score -= memoryUsage * 30; // 内存使用影响30分
    score -= errorRate * 40; // 错误率影响40分
    score -= Math.min(averageResponseTime / 1000, 1) * 30; // 响应时间影响30分

    if (score >= 80) {
      return { level: '优秀', color: Color.Green, icon: AppIcon.CHECK };
    } else if (score >= 60) {
      return { level: '良好', color: Color.Blue, icon: AppIcon.INFO };
    } else if (score >= 40) {
      return { level: '一般', color: Color.Orange, icon: AppIcon.WARNING };
    } else {
      return { level: '较差', color: Color.Red, icon: AppIcon.ERROR };
    }
  }

  /**
   * 获取指标颜色
   */
  private getMetricColor(value: number, thresholds: { good: number; warning: number }): ResourceColor {
    if (value <= thresholds.good) return Color.Green;
    if (value <= thresholds.warning) return Color.Orange;
    return Color.Red;
  }

  build() {
    if (this.compact) {
      // 紧凑模式
      this.buildCompactView()
    } else {
      // 完整模式
      this.buildFullView()
    }
  }

  /**
   * 构建紧凑视图
   */
  @Builder
  private buildCompactView() {
    Row({ space: 8 }) {
      // 性能等级指示器
      if (this.metrics) {
        const performance = this.getPerformanceLevel();
        
        Row({ space: 4 }) {
          SystemIcon({
            icon: performance.icon,
            iconSize: 16,
            iconColor: performance.color
          })
          
          Text(performance.level)
            .fontSize(12)
            .fontColor(performance.color)
        }
      }

      // 告警指示器
      if (this.alerts.length > 0) {
        Row({ space: 4 }) {
          SystemIcon({
            icon: AppIcon.WARNING,
            iconSize: 14,
            iconColor: Color.Orange
          })
          
          Text(this.alerts.length.toString())
            .fontSize(11)
            .fontColor(Color.Orange)
        }
      }

      // 展开按钮
      SystemIcon({
        icon: this.isExpanded ? AppIcon.ARROW_UP : AppIcon.ARROW_DOWN,
        iconSize: 14,
        onIconClick: () => {
          this.isExpanded = !this.isExpanded;
        }
      })
    }
    .padding({ left: 8, right: 8, top: 4, bottom: 4 })
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(12)
    .onClick(() => {
      this.isExpanded = !this.isExpanded;
    })
    // 展开的详细信息
    .bindPopup(this.isExpanded, {
      builder: this.buildPopupContent,
      placement: Placement.Bottom,
      maskColor: Color.Transparent,
      popupColor: $r('sys.color.ohos_id_color_background'),
      enableArrow: true,
      showInSubWindow: false
    })
    // 无障碍属性
    .accessibilityGroup(true)
    .accessibilityText(this.metrics ? `当前性能等级：${this.getPerformanceLevel().level}` : '性能数据加载中')
    .accessibilityDescription(this.alerts.length > 0 ? `有${this.alerts.length}个性能告警` : '性能正常')
  }

  /**
   * 构建完整视图
   */
  @Builder
  private buildFullView() {
    Column({ space: 16 }) {
      // 标题栏
      Row() {
        Text('性能监控')
          .fontSize(18)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)

        Row({ space: 12 }) {
          SystemIcon({
            icon: AppIcon.REFRESH,
            iconSize: 20,
            onIconClick: () => this.onRefresh()
          })

          SystemIcon({
            icon: AppIcon.CLEAN,
            iconSize: 20,
            onIconClick: () => this.onClearCache()
          })
        }
      }
      .width('100%')

      if (this.refreshing) {
        Row({ space: 8 }) {
          LoadingProgress()
            .width(16)
            .height(16)
            .color($r('sys.color.ohos_id_color_emphasize'))
          
          Text('刷新中...')
            .fontSize(14)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
      }

      // 性能概览
      if (this.metrics) {
        const performance = this.getPerformanceLevel();
        
        Row({ space: 12 }) {
          SystemIcon({
            icon: performance.icon,
            iconSize: 32,
            iconColor: performance.color
          })

          Column({ space: 4 }) {
            Text('整体性能')
              .fontSize(14)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              .alignSelf(ItemAlign.Start)

            Text(performance.level)
              .fontSize(20)
              .fontWeight(FontWeight.Bold)
              .fontColor(performance.color)
              .alignSelf(ItemAlign.Start)
          }
          .alignItems(HorizontalAlign.Start)
          .layoutWeight(1)
        }
        .width('100%')
        .padding(16)
        .backgroundColor($r('sys.color.ohos_id_color_background'))
        .borderRadius(8)
      }

      // 详细指标
      if (this.metrics && this.showDetails) {
        Grid() {
          GridItem() {
            this.buildMetricCard(
              '内存使用',
              `${(this.metrics.memoryUsage * 100).toFixed(1)}%`,
              this.getMetricColor(this.metrics.memoryUsage, { good: 0.6, warning: 0.8 }),
              AppIcon.MEMORY
            )
          }

          GridItem() {
            this.buildMetricCard(
              '缓存命中率',
              `${(this.metrics.cacheHitRate * 100).toFixed(1)}%`,
              this.getMetricColor(1 - this.metrics.cacheHitRate, { good: 0.3, warning: 0.5 }),
              AppIcon.CACHE
            )
          }

          GridItem() {
            this.buildMetricCard(
              '响应时间',
              `${this.metrics.averageResponseTime.toFixed(0)}ms`,
              this.getMetricColor(this.metrics.averageResponseTime, { good: 500, warning: 1000 }),
              AppIcon.SPEED
            )
          }

          GridItem() {
            this.buildMetricCard(
              '错误率',
              `${(this.metrics.errorRate * 100).toFixed(1)}%`,
              this.getMetricColor(this.metrics.errorRate, { good: 0.01, warning: 0.05 }),
              AppIcon.ERROR
            )
          }

          // 扩展指标
          if (this.metrics.cpuUsage !== undefined) {
            GridItem() {
              this.buildMetricCard(
                'CPU使用',
                `${(this.metrics.cpuUsage * 100).toFixed(1)}%`,
                this.getMetricColor(this.metrics.cpuUsage, { good: 0.5, warning: 0.8 }),
                AppIcon.CPU
              )
            }
          }

          if (this.metrics.batteryLevel !== undefined) {
            GridItem() {
              this.buildMetricCard(
                '电池电量',
                `${(this.metrics.batteryLevel * 100).toFixed(0)}%`,
                this.getMetricColor(1 - this.metrics.batteryLevel, { good: 0.5, warning: 0.8 }),
                AppIcon.BATTERY
              )
            }
          }
        }
        .columnsTemplate('1fr 1fr')
        .rowsGap(12)
        .columnsGap(12)
        .width('100%')
      }

      // 性能告警
      if (this.alerts.length > 0) {
        Column({ space: 8 }) {
          Text(`性能告警 (${this.alerts.length})`)
            .fontSize(16)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .alignSelf(ItemAlign.Start)

          ForEach(this.alerts, (alert: PerformanceAlert) => {
            Row({ space: 12 }) {
              SystemIcon({
                icon: alert.level === 'critical' ? AppIcon.ERROR : AppIcon.WARNING,
                iconSize: 20,
                iconColor: alert.level === 'critical' ? Color.Red : Color.Orange
              })

              Column({ space: 4 }) {
                Text(alert.message)
                  .fontSize(14)
                  .fontColor($r('sys.color.ohos_id_color_text_primary'))
                  .alignSelf(ItemAlign.Start)

                Text(new Date(alert.timestamp).toLocaleString())
                  .fontSize(12)
                  .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                  .alignSelf(ItemAlign.Start)
              }
              .alignItems(HorizontalAlign.Start)
              .layoutWeight(1)
            }
            .width('100%')
            .padding(12)
            .backgroundColor($r('sys.color.ohos_id_color_background'))
            .borderRadius(8)
          })
        }
        .width('100%')
      }
    }
    .width('100%')
    .padding(16)
  }

  /**
   * 构建指标卡片
   */
  @Builder
  private buildMetricCard(title: string, value: string, color: ResourceColor, icon: AppIcon) {
    Column({ space: 8 }) {
      SystemIcon({
        icon: icon,
        iconSize: 24,
        iconColor: color
      })

      Text(value)
        .fontSize(18)
        .fontWeight(FontWeight.Bold)
        .fontColor(color)

      Text(title)
        .fontSize(12)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        .textAlign(TextAlign.Center)
    }
    .width('100%')
    .padding(12)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(8)
    .justifyContent(FlexAlign.Center)
    // 无障碍属性
    .accessibilityGroup(true)
    .accessibilityText(`${title}：${value}`)
  }

  /**
   * 构建弹窗内容
   */
  @Builder
  private buildPopupContent() {
    Column({ space: 8 }) {
      if (this.metrics) {
        Text('性能概览')
          .fontSize(14)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))

        Row() {
          Text('内存:')
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .width(60)

          Text(`${(this.metrics.memoryUsage * 100).toFixed(1)}%`)
            .fontSize(12)
            .fontColor(this.getMetricColor(this.metrics.memoryUsage, { good: 0.6, warning: 0.8 }))
        }
        .width('100%')

        Row() {
          Text('响应:')
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .width(60)

          Text(`${this.metrics.averageResponseTime.toFixed(0)}ms`)
            .fontSize(12)
            .fontColor(this.getMetricColor(this.metrics.averageResponseTime, { good: 500, warning: 1000 }))
        }
        .width('100%')

        if (this.alerts.length > 0) {
          Divider()
            .color($r('sys.color.ohos_id_color_list_separator'))
            .margin({ top: 4, bottom: 4 })

          Text(`${this.alerts.length} 个告警`)
            .fontSize(12)
            .fontColor(Color.Orange)
        }
      }
    }
    .width(200)
    .padding(12)
  }
}
