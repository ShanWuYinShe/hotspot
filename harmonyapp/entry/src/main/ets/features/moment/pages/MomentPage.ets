/**
 * 极简生活瞬间页面
 * 使用HarmonyOS原生能力，专注核心功能
 * 优化：使用 LazyForEach 和 BasicDataSource 提升长列表性能
 * 优化：添加图片点击预览功能
 * 优化：使用 RDB 缓存实现离线浏览
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { PageNames } from '../../../constants/AppConstants';
import { LikeMomentRequest, MomentSearchItem, SearchMomentsByLocationRequest } from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { IntelligentLocationCard } from '../../../components/IntelligentLocationCard';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, SystemIcon } from '../../../components/SystemIcon';
import { SystemCard } from '../../../components/SystemCard';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { SystemToast } from '../../../components/SystemToast';
import { MomentCacheService } from '../../../core/database/MomentCacheService';

const DOMAIN_NUMBER: number = 0xFF05;
const TAG: string = 'MomentPage';

@Component
export struct MomentPage {
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  
  private stateManager = GlobalStateManager.getInstance();
  // 使用自定义 DataSource
  private momentsDataSource: BasicDataSource<MomentSearchItem> = new BasicDataSource([]);
  private momentCacheService = MomentCacheService.getInstance();
  
  @State isLoading: boolean = false;
  @State errorMessage: string = '';
  @State hasData: boolean = false; // 用于控制空状态显示

  async aboutToAppear() {
    await this.loadNearbyMoments();
  }

  aboutToDisappear() {
    // 
  }

  /**
   * 加载附近瞬间
   */
  private async loadNearbyMoments(forceRefresh: boolean = false) {
    if (!this.currentLocation && !forceRefresh) {
        // 无位置时尝试加载缓存
        await this.loadFromCache();
    }
    
    if (!this.currentLocation) {
      if (this.momentsDataSource.totalCount() === 0) {
        this.errorMessage = '请先获取位置信息';
      }
      return;
    }

    // 1. 先尝试加载缓存（如果是首次加载）
    if (!forceRefresh && this.momentsDataSource.totalCount() === 0) {
      await this.loadFromCache();
    }

    if (this.momentsDataSource.totalCount() === 0) {
        this.isLoading = true;
    }
    this.errorMessage = '';

    try {
      const request: SearchMomentsByLocationRequest = {
        reference_point: {
          latitude: this.currentLocation.latitude,
          longitude: this.currentLocation.longitude
        },
        radius: 2000,
        limit: 50
      };
      const response = await ApiClient.searchMomentsByLocation(request);
      
      if (response.code === 0 && response.data) {
        this.momentsDataSource.setData(response.data.items);
        this.hasData = response.data.items.length > 0;
        
        // 3. 异步更新缓存
        this.updateCache(response.data.items);
      } else {
        if (this.momentsDataSource.totalCount() === 0) {
           this.errorMessage = response.message || '加载瞬间失败';
           this.hasData = false;
        }
      }
    } catch (error) {
      if (this.momentsDataSource.totalCount() === 0) {
        this.errorMessage = error instanceof Error ? error.message : '加载瞬间失败';
        this.hasData = false;
      }
    } finally {
      this.isLoading = false;
    }
  }

  private async loadFromCache() {
      const cachedItems = await this.momentCacheService.getCachedMoments();
      if (cachedItems.length > 0) {
          this.momentsDataSource.setData(cachedItems);
          this.hasData = true;
          hilog.info(DOMAIN_NUMBER, TAG, `Loaded ${cachedItems.length} moments from cache`);
      }
  }

  private async updateCache(moments: MomentSearchItem[]) {
      await this.momentCacheService.cacheMoments(moments);
  }

  /**
   * 点赞瞬间
   */
  private async likeMoment(moment: MomentSearchItem, index: number) {
    const currentUser = this.stateManager.getCurrentUser();
    if (!currentUser) {
      SystemToast.showWarning('请先登录后再点赞', this.getUIContext());
      try {
        const routerInstance = this.getUIContext().getRouter();
        await routerInstance.pushUrl({
          url: `features/auth/pages/${PageNames.LOGIN}`,
          params: { reason: 'action_required', action: '点赞瞬间' }
        });
      } catch (error) {
        // ignore
      }
      return;
    }

    try {
      const request: LikeMomentRequest = {
        moment_id: moment.moment_id
      };
      const response = await ApiClient.likeMoment(request);
      
      if (response.code === 0) {
        // 更新本地状态 - 点赞成功后重新加载列表
        await this.loadNearbyMoments(true);
      } else {
        hilog.error(DOMAIN_NUMBER, TAG, `点赞失败: ${response.message}`);
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `点赞失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 预览图片
   */
  private previewImage(momentId: string, images: string[], index: number) {
    try {
      const routerInstance = this.getUIContext().getRouter();
      routerInstance.pushUrl({
        url: `features/moment/pages/${PageNames.PHOTO_PREVIEW}`,
        params: {
          momentId: momentId,
          images: images,
          index: index
        }
      });
    } catch (error) {
      SystemToast.showError('打开图片失败', this.getUIContext());
    }
  }

  build() {
    Column() {
      // 标题栏
      Row() {
        Text('附近瞬间')
          .fontSize(20)
          .fontWeight(FontWeight.Bold)
          .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
          .layoutWeight(1)

        Button('发布')
          .type(ButtonType.Capsule)
          .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
          .fontColor(Color.White)
          .fontSize(14)
          .padding({ left: 16, right: 16, top: 8, bottom: 8 })
          .onClick(async () => {
            try {
              const currentUser = this.stateManager.getCurrentUser();
              if (!currentUser) {
                SystemToast.showWarning('请先登录后再发布', this.getUIContext());
                const routerInstance = this.getUIContext().getRouter();
                await routerInstance.pushUrl({
                  url: `features/auth/pages/${PageNames.LOGIN}`,
                  params: { reason: 'action_required', action: '发布瞬间' }
                });
                return;
              }

              const routerInstance = this.getUIContext().getRouter();
              await routerInstance.pushUrl({ url: `features/moment/pages/${PageNames.CREATE_MOMENT}` });
            } catch (error) {
              hilog.error(DOMAIN_NUMBER, TAG, `跳转失败: ${JSON.stringify(error)}`);
            }
          })
      }
      .width('100%')
      .padding(16)
      .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)

      // 内容区域
      if (this.isLoading && !this.hasData) {
        SystemLoadingView({ message: '正在加载附近瞬间...' })
      } else if (!this.hasData && this.errorMessage) {
        // 内联错误视图
        Column({ space: 16 }) {
          Text('⚠️')
            .fontSize(48)
          
          Text(this.errorMessage)
            .fontSize(16)
            .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
            .textAlign(TextAlign.Center)
          
          Button('重试')
            .type(ButtonType.Capsule)
            .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
            .fontColor(Color.White)
            .onClick(() => this.loadNearbyMoments(true))
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
        .padding(24)
      } else if (!this.hasData) {
        // 内联空状态视图
        Refresh({ refreshing: $$this.isLoading, builder: this.RefreshBuilder() }) {
          Column({ space: 16 }) {
            Text('✨')
              .fontSize(48)
            
            Text('附近暂无瞬间\n快来分享第一个吧！')
              .fontSize(16)
              .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
              .textAlign(TextAlign.Center)
          }
          .width('100%')
          .height('100%')
          .justifyContent(FlexAlign.Center)
        }
        .onRefreshing(() => {
          this.loadNearbyMoments(true);
        })
      } else {
        Refresh({ refreshing: $$this.isLoading, builder: this.RefreshBuilder() }) {
          this.buildMomentList()
        }
        .onRefreshing(() => {
          this.loadNearbyMoments(true);
        })
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)
  }

  @Builder
  RefreshBuilder() {
    Row() {
      LoadingProgress().width(24).height(24)
      Text('正在刷新...').fontSize(14).margin({left: 8}).fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
    }
    .width('100%')
    .justifyContent(FlexAlign.Center)
    .padding(10)
  }

  @Builder
  private buildMomentList() {
    List({ space: 12 }) {
      LazyForEach(this.momentsDataSource, (moment: MomentSearchItem, index: number) => {
        ListItem() {
          this.buildMomentCard(moment, index)
        }
      }, (moment: MomentSearchItem) => moment.moment_id)
    }
    .width('100%')
    .layoutWeight(1)
    .padding({ left: 16, right: 16, top: 8, bottom: 16 })
    .scrollBar(BarState.Auto)
    .cachedCount(5) // 缓存5个项目
  }

  @Builder
  private buildMomentCard(moment: MomentSearchItem, index: number) {
    SystemCard({
      content: () => {
        this.MomentCardContent(moment, index);
      },
      cardBackgroundColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT
    })
  }


  @Builder
  private MomentCardContent(moment: MomentSearchItem, index: number) {
    // 使用独立的 LikeButton 组件管理点赞状态
    
    Column({ space: 12 }) {
      // 用户信息
      Row({ space: 12 }) {
        SystemAvatar({ 
          avatarUrl: moment.user.avatar_url || '',
          name: moment.user.nickname,
          avatarSize: 40
        })

        Column({ space: 2 }) {
          Text(moment.user.nickname)
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontWeight(FontWeight.Medium)
            .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)

          Text(this.formatTime(moment.created_at))
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)
      }
      .width('100%')

      // 瞬间内容
      Text(moment.description)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
        .lineHeight(24)

      // 图片展示 (如果有)
      if (moment.images && moment.images.length > 0) {
         Grid() {
           ForEach(moment.images.slice(0, 9), (imgUrl: string, imgIndex: number) => {
             GridItem() {
               Image(imgUrl)
                 .width('100%')
                 .height('100%')
                 .objectFit(ImageFit.Cover)
                 .borderRadius(DesignConstants.RADIUS.MD)
                 .backgroundColor(DesignConstants.COLOR.TERTIARY_BG_LIGHT)
                 .geometryTransition('moment_img_' + moment.moment_id + '_' + imgIndex)
                 .onClick(() => {
                    this.previewImage(moment.moment_id, moment.images || [], imgIndex);
                 })
             }
             .aspectRatio(1)
           })
         }
         .columnsTemplate('1fr 1fr 1fr')
         .columnsGap(8)
         .rowsGap(8)
         .height(this.calcImageGridHeight(moment.images.length))
         .width('100%')
       }

      // 标签
      if (moment.tags && moment.tags.length > 0) {
        Flex({ wrap: FlexWrap.Wrap }) {
          ForEach(moment.tags, (tag: string) => {
            // 内联标签组件
            Text(`#${tag}`)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor(DesignConstants.COLOR.BRAND_PRIMARY)
              .backgroundColor(DesignConstants.COLOR.TERTIARY_BG_LIGHT)
              .padding({ left: 12, right: 12, top: 6, bottom: 6 })
              .borderRadius(16)
              .margin({ right: 8, bottom: 8 })
          }, (tag: string) => tag)
        }
        .width('100%')
      }

      // 位置信息
      if (moment.location) {
        IntelligentLocationCard({
          latitude: moment.location.latitude,
          longitude: moment.location.longitude,
          locationName: moment.location.location_name,
          distance: moment.location.distance,
          altitude: 0 // 列表数据暂时不包含海拔信息，需详情接口获取
        })
      }

      // 互动区域
      Row({ space: 16 }) {
        Row({ space: 4 }) {
          LikeButton({ 
            momentId: moment.moment_id, 
            initialLikesCount: moment.likes_count,
            onLike: () => this.likeMoment(moment, index)
          })
        }

        Row({ space: 4 }) {
          SystemIcon({ icon: AppIcon.MESSAGE, iconSize: 20, iconColor: DesignConstants.COLOR.TEXT_SECONDARY_LIGHT })

          Text(moment.comments_count.toString())
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
        }

        Blank()

        Row({ space: 4 }) {
          SystemIcon({ icon: AppIcon.SHARE, iconSize: 20, iconColor: DesignConstants.COLOR.TEXT_SECONDARY_LIGHT })
          Text('分享')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
        }
      }
      .width('100%')
    }
    .width('100%')
    .padding(16)
  }
}

/**
 * 独立的点赞按钮组件，用于管理单个瞬间的点赞状态
 */
@Component
struct LikeButton {
  @Prop momentId: string;
  @State isLiked: boolean = false;
  @State likesCount: number = 0;
  @Prop initialLikesCount: number = 0;
  onLike?: () => void;

  async aboutToAppear() {
    this.likesCount = this.initialLikesCount;
  }

  build() {
    Row({ space: 4 }) {
      SystemIcon({ 
        icon: this.isLiked ? AppIcon.HEART_FILL : AppIcon.HEART, 
        iconSize: 20, 
        iconColor: this.isLiked ? DesignConstants.COLOR.ERROR : DesignConstants.COLOR.TEXT_SECONDARY_LIGHT,
        clickable: true,
        onIconClick: () => {
          // 乐观更新
          this.isLiked = !this.isLiked;
          this.likesCount += this.isLiked ? 1 : -1;
          if (this.onLike) {
            this.onLike();
          }
        }
      })

      Text(this.likesCount.toString())
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontColor(this.isLiked ? DesignConstants.COLOR.ERROR : DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
    }
  }

  private calcImageGridHeight(count: number): number {
    const rows = Math.ceil(Math.min(count, 9) / 3);
    // 假设每个 GridItem 宽度大约 (Screen - padding) / 3
    // 这里简单给个估计值，或者使用 aspectRatio 固定比例
    // 实际上 Grid 使用 aspectRatio(1) 后高度会自动撑开，但 Grid 容器可能需要显式高度
    // 如果 Grid 不设置高度，在 Column 中可能显示异常? 
    // 实测 Grid 设置 columnsTemplate 后高度默认自适应内容，但在 Scroll/List 中表现各异
    // 这里给个估算高度: 假设宽度约 100vp
    return rows * 110; 
  }

  /**
   * 格式化时间
   */
  private formatTime(timeStr: string): string {
    const time = new Date(timeStr);
    const now = new Date();
    const diff = now.getTime() - time.getTime();
    
    const minutes = Math.floor(diff / (1000 * 60));
    const hours = Math.floor(diff / (1000 * 60 * 60));
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (minutes < 1) return '刚刚';
    if (minutes < 60) return `${minutes}分钟前`;
    if (hours < 24) return `${hours}小时前`;
    if (days < 7) return `${days}天前`;
    
    return time.toLocaleDateString();
  }
}
