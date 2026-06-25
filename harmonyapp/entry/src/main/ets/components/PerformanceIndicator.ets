/**
 * 性能指示器组件
 * 实时显示应用性能指标和状态
 */
import { PerformanceOptimizer, PerformanceMetrics, PerformanceAlert } from '../core/performance/PerformanceOptimizer';
import { SystemIcon, AppIcon } from './SystemIcon';
import { SystemToast } from './SystemToast';
import { AccessibilityManager } from '../core/accessibility/AccessibilityManager';
import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

interface PerformanceLevel {
  level: string;
  color: ResourceColor;
  icon: AppIcon;
}

interface MetricThresholds {
  good: number;
  warning: number;
}

@Component
export struct PerformanceIndicator {
  @Prop showDetails: boolean = false;
  @Prop compact: boolean = false;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';
  @State private metrics: PerformanceMetrics | null = null;
  @State private alerts: PerformanceAlert[] = [];
  @State private isExpanded: boolean = false;
  @State private refreshing: boolean = false;

  private performanceOptimizer = PerformanceOptimizer.getInstance();
  private accessibilityManager = AccessibilityManager.getInstance();
  private updateTimer: number = -1;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

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
      SystemToast.showError(`加载性能数据失败: ${error}`, this.getUIContext());
    }
  }

  /**
   * 设置性能监听器
   */
  private setupPerformanceListener(): void {
    // 使用轮询方式检查性能告警
    setInterval(() => {
      const alerts = this.performanceOptimizer.getPerformanceAlerts();
      if (alerts.length > 0) {
        const latestAlert = alerts[alerts.length - 1];
        this.loadMetrics();
        
        // 无障碍播报严重告警
        if (latestAlert.level === 'critical') {
          this.accessibilityManager.announceText(`严重性能告警：${latestAlert.message}`, 'high');
        }
      }
    }, 5000); // 每5秒检查一次
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
      SystemToast.showSuccess('性能数据已刷新', this.getUIContext());
    }, 500);
  }

  /**
   * 清理缓存
   */
  private onClearCache(): void {
    try {
      this.performanceOptimizer.clearCache();
      SystemToast.showSuccess('缓存已清理', this.getUIContext());
      this.loadMetrics();
      
      // 无障碍播报
      this.accessibilityManager.announceText('缓存已清理，性能可能有所改善');
    } catch (error) {
      SystemToast.showError(`清理缓存失败: ${error}`, this.getUIContext());
    }
  }

  /**
   * 获取性能等级
   */
  private getPerformanceLevel(): PerformanceLevel {
    if (!this.metrics) {
      const unknownLevel: PerformanceLevel = { level: '未知', color: this.getPalette().textSecondary, icon: AppIcon.QUESTION };
      return unknownLevel;
    }

    const memoryUsage = this.metrics.memoryUsage;
    const errorRate = this.metrics.errorRate;
    const averageResponseTime = this.metrics.averageResponseTime;
    
    // 综合评分
    let score = 100;
    score -= memoryUsage * 30; // 内存使用影响30分
    score -= errorRate * 40; // 错误率影响40分
    score -= Math.min(averageResponseTime / 1000, 1) * 30; // 响应时间影响30分

    if (score >= 80) {
      const excellentLevel: PerformanceLevel = { level: '优秀', color: this.getPalette().accentBackground, icon: AppIcon.CHECK };
      return excellentLevel;
    } else if (score >= 60) {
      const goodLevel: PerformanceLevel = { level: '良好', color: this.getPalette().textPrimary, icon: AppIcon.INFO };
      return goodLevel;
    } else if (score >= 40) {
      const averageLevel: PerformanceLevel = { level: '一般', color: this.getPalette().textSecondary, icon: AppIcon.WARNING };
      return averageLevel;
    } else {
      const poorLevel: PerformanceLevel = { level: '较差', color: this.getPalette().textTertiary, icon: AppIcon.ERROR };
      return poorLevel;
    }
  }

  /**
   * 获取指标颜色
   */
  private getMetricColor(value: number, thresholds: MetricThresholds): ResourceColor {
    if (value <= thresholds.good) return this.getPalette().accentBackground;
    if (value <= thresholds.warning) return this.getPalette().textSecondary;
    return this.getPalette().textTertiary;
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
        Row({ space: 4 }) {
          SystemIcon({
            icon: this.getPerformanceLevel().icon,
            iconSize: 16,
            iconColor: this.getPerformanceLevel().color
          })
          
          Text(this.getPerformanceLevel().level)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getPerformanceLevel().color)
        }
      }

      // 告警指示器
      if (this.alerts.length > 0) {
        Row({ space: 4 }) {
          SystemIcon({
            icon: AppIcon.WARNING,
            iconSize: 14,
            iconColor: this.getPalette().accentBackground
          })

          Text(this.alerts.length.toString())
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getPalette().accentBackground)
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
    .backgroundColor(this.getPalette().surfaceBackground)
    .borderRadius(DesignConstants.RADIUS.LG)
    .onClick(() => {
      this.isExpanded = !this.isExpanded;
    })
    // 展开的详细信息
    .bindPopup(this.isExpanded, {
      builder: this.buildPopupContent,
      placement: Placement.Bottom,
      maskColor: DesignConstants.COLOR.TRANSPARENT,
      popupColor: this.getPalette().surfaceBackground,
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
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getPalette().textPrimary)
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
            .color(this.getPalette().accentBackground)

          Text('刷新中...')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor(this.getPalette().textSecondary)
        }
      }

      // 性能概览
      if (this.metrics) {
        Row({ space: 12 }) {
          SystemIcon({
            icon: this.getPerformanceLevel().icon,
            iconSize: 32,
            iconColor: this.getPerformanceLevel().color
          })

          Column({ space: 4 }) {
            Text('整体性能')
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontColor(this.getPalette().textSecondary)
              .alignSelf(ItemAlign.Start)

            Text(this.getPerformanceLevel().level)
              .fontSize(DesignConstants.FONT.SIZE.XL)
              .fontWeight(FontWeight.Bold)
              .fontColor(this.getPerformanceLevel().color)
              .alignSelf(ItemAlign.Start)
          }
          .alignItems(HorizontalAlign.Start)
          .layoutWeight(1)
        }
        .width('100%')
        .padding(16)
        .backgroundColor(this.getPalette().surfaceBackground)
        .borderRadius(DesignConstants.RADIUS.MD)
      }

      // 详细指标
      if (this.metrics && this.showDetails) {
        Grid() {
          GridItem() {
            this.buildMetricCard(
              '内存使用',
              `${(this.metrics.memoryUsage * 100).toFixed(1)}%`,
              this.getMetricColor(this.metrics.memoryUsage, { good: 0.6, warning: 0.8 } as MetricThresholds),
              AppIcon.MEMORY
            )
          }

          GridItem() {
            this.buildMetricCard(
              '缓存命中率',
              `${(this.metrics.cacheHitRate * 100).toFixed(1)}%`,
              this.getMetricColor(1 - this.metrics.cacheHitRate, { good: 0.3, warning: 0.5 } as MetricThresholds),
              AppIcon.CACHE
            )
          }

          GridItem() {
            this.buildMetricCard(
              '响应时间',
              `${this.metrics.averageResponseTime.toFixed(0)}ms`,
              this.getMetricColor(this.metrics.averageResponseTime, { good: 500, warning: 1000 } as MetricThresholds),
              AppIcon.SPEED
            )
          }

          GridItem() {
            this.buildMetricCard(
              '错误率',
              `${(this.metrics.errorRate * 100).toFixed(1)}%`,
              this.getMetricColor(this.metrics.errorRate, { good: 0.01, warning: 0.05 } as MetricThresholds),
              AppIcon.ERROR
            )
          }

          // 扩展指标
          if (this.metrics.cpuUsage !== undefined) {
            GridItem() {
              this.buildMetricCard(
                'CPU使用',
                `${(this.metrics.cpuUsage * 100).toFixed(1)}%`,
                this.getMetricColor(this.metrics.cpuUsage, { good: 0.5, warning: 0.8 } as MetricThresholds),
                AppIcon.CPU
              )
            }
          }

          if (this.metrics.batteryLevel !== undefined) {
            GridItem() {
              this.buildMetricCard(
                '电池电量',
                `${(this.metrics.batteryLevel * 100).toFixed(0)}%`,
                this.getMetricColor(1 - this.metrics.batteryLevel, { good: 0.5, warning: 0.8 } as MetricThresholds),
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
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(FontWeight.Medium)
            .fontColor(this.getPalette().textPrimary)
            .alignSelf(ItemAlign.Start)

          ForEach(this.alerts, (alert: PerformanceAlert) => {
            Row({ space: 12 }) {
              SystemIcon({
                icon: alert.level === 'critical' ? AppIcon.ERROR : AppIcon.WARNING,
                iconSize: 20,
                iconColor: alert.level === 'critical' ? this.getPalette().textTertiary : this.getPalette().accentBackground
              })

              Column({ space: 4 }) {
                Text(alert.message)
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .fontColor(this.getPalette().textPrimary)
                  .alignSelf(ItemAlign.Start)

                Text(new Date(alert.timestamp).toLocaleString())
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(this.getPalette().textSecondary)
                  .alignSelf(ItemAlign.Start)
              }
              .alignItems(HorizontalAlign.Start)
              .layoutWeight(1)
            }
            .width('100%')
            .padding(12)
            .backgroundColor(this.getPalette().surfaceBackground)
            .borderRadius(DesignConstants.RADIUS.MD)
          }, (alert: PerformanceAlert) => `${alert.timestamp}_${alert.message}`)
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
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(FontWeight.Bold)
        .fontColor(color)

      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.getPalette().textSecondary)
        .textAlign(TextAlign.Center)
    }
    .width('100%')
    .padding(12)
    .backgroundColor(this.getPalette().surfaceBackground)
    .borderRadius(DesignConstants.RADIUS.MD)
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
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getPalette().textPrimary)

        Row() {
          Text('内存:')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getPalette().textSecondary)
            .width(60)

          Text(`${(this.metrics.memoryUsage * 100).toFixed(1)}%`)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getMetricColor(this.metrics.memoryUsage, { good: 0.6, warning: 0.8 } as MetricThresholds))
        }
        .width('100%')

        Row() {
          Text('响应:')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getPalette().textSecondary)
            .width(60)

          Text(`${this.metrics.averageResponseTime.toFixed(0)}ms`)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getMetricColor(this.metrics.averageResponseTime, { good: 500, warning: 1000 } as MetricThresholds))
        }
        .width('100%')

        if (this.alerts.length > 0) {
          Divider()
            .color(this.getPalette().borderColor)
            .margin({ top: 4, bottom: 4 })

          Text(`${this.alerts.length} 个告警`)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getPalette().accentBackground)
        }
      }
    }
    .width(200)
    .padding(12)
  }
}
