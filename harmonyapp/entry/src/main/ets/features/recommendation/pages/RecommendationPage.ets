/**
 * 推荐页面
 * 展示推荐内容，使用原生 List 和 Grid 布局
 */

import { ApiClient } from '../../../core/api/ApiClient';
import { FeedUserItem, TrendingTagsData, TagData } from '../../../core/types/api';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { SystemToast } from '../../../components/SystemToast';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { DesignConstants } from '../../../constants/DesignConstants';


const DOMAIN_NUMBER: number = 0xFF09;
const TAG: string = 'RecommendationPage';

@Entry
@Component
struct RecommendationPage {
  @State isLoading: boolean = true;
  @State trendingTags: TagData[] = [];
  @State recommendedUsers: FeedUserItem[] = [];

  async aboutToAppear() {
    await this.loadData();
  }

  private async loadData() {
    this.isLoading = true;
    try {
      // 并行加载推荐数据
      const [tagsResponse, usersResponse] = await Promise.all([
        ApiClient.getTrendingTags({ limit: 10, timeframe: 'weekly' }),
        ApiClient.getFeedUsers({ limit: 5 })
      ]);

      if (tagsResponse.code === 0 && tagsResponse.data) {
        this.trendingTags = tagsResponse.data.items;
      }

      if (usersResponse.code === 0 && usersResponse.data) {
        this.recommendedUsers = usersResponse.data.items;
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载推荐数据失败: ${JSON.stringify(error)}`);
      SystemToast.showError('加载失败，请重试', this.getUIContext());
    } finally {
      this.isLoading = false;
    }
  }

  build() {
    Column() {
      // 标题栏
      Row() {
        Text('推荐')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Bold)
          .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
          .layoutWeight(1)
          .padding({ left: 16 })
      }
      .width('100%')
      .height(56)
      .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)

      if (this.isLoading) {
        SystemLoadingView({ message: '加载推荐内容...' })
      } else {
        List({ space: 16 }) {
          // 热门话题
          ListItem() {
            Column({ space: 12 }) {
              Text('热门话题')
                .fontSize(16)
                .fontWeight(FontWeight.Bold)
                .width('100%')
                .padding({ left: 8 })
              
              Flex({ wrap: FlexWrap.Wrap }) {
                if (this.trendingTags.length > 0) {
                  ForEach(this.trendingTags, (tag: TagData) => {
                    Text(`#${tag.name}`)
                      .padding(8)
                      .backgroundColor(DesignConstants.COLOR.TERTIARY_BG_LIGHT)
                      .borderRadius(16)
                      .margin({ right: 8, bottom: 8 })
                  })
                } else {
                  Text('暂无热门话题')
                    .fontSize(14)
                    .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
                    .padding(8)
                }
              }
              .padding({ left: 8, right: 8 })
            }
          }
          .width('100%')
          .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
          .padding(16)
          .borderRadius(DesignConstants.RADIUS.LG)


          // 推荐用户
          if (this.recommendedUsers.length > 0) {
            ListItemGroup({ header: this.ListHeader('推荐用户') }) {
              ForEach(this.recommendedUsers, (user: FeedUserItem) => {
                ListItem() {
                  Row() {
                    SystemAvatar({
                      avatarUrl: user.avatar_url,
                      nickname: user.nickname,
                      size: 40
                    })
                    .margin({ right: 12 })
                    
                    Column() {
                      Text(user.nickname)
                        .fontSize(16)
                        .fontWeight(FontWeight.Medium)
                      Text(user.reason || '可能会感兴趣')
                        .fontSize(12)
                        .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
                    }
                    .alignItems(HorizontalAlign.Start)
                    .layoutWeight(1)

                    Button('关注')
                      .type(ButtonType.Capsule)
                      .height(28)
                      .fontSize(12)
                      .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
                      .onClick(async () => {
                        try {
                          await ApiClient.followUser({ id: user.user_id });
                          SystemToast.showSuccess('已关注', this.getUIContext());
                        } catch (error) {
                          SystemToast.showError('关注失败', this.getUIContext());
                        }
                      })
                  }
                  .width('100%')
                  .padding(16)
                  .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
                }
              })
            }
            .borderRadius(DesignConstants.RADIUS.LG)
            .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
            .divider({ strokeWidth: 1, color: DesignConstants.COLOR.TERTIARY_BG_LIGHT, startMargin: 68, endMargin: 16 })
          }
        }
        .width('100%')
        .layoutWeight(1)
        .padding({ left: 16, right: 16, top: 16, bottom: 16 })
        .scrollBar(BarState.Auto)
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)
  }

  @Builder
  ListHeader(title: string) {
    Text(title)
      .fontSize(16)
      .fontWeight(FontWeight.Bold)
      .width('100%')
      .padding({ left: 8, bottom: 8, top: 8 })
      .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
  }
}
