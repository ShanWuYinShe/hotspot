/**
 * 创建生活瞬间页面
 * 用户可以发布带有地理位置的生活瞬间
 */

import { GlobalStateManager, GlobalState } from '../common/state/GlobalStateManager';
import { ApiClient } from '../common/api/ApiClient';
import { LocationUtil } from '../common/location/LocationUtil';
import { SystemToast } from '../components/SystemToast';
import { IconButton, AppIcon, IconSize } from '../components/SystemIcon';
import { selectImages, capturePhoto } from '../common/utils/MediaUtils';
import { common } from '@kit.AbilityKit';
import { LocationInfo } from '../common/location/LocationUtil';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF06;
const TAG: string = 'CreateMomentPage';

@Entry
@Component
struct CreateMomentPage {
  @State globalState: GlobalState = GlobalStateManager.getInstance().getState();
  private stateManager = GlobalStateManager.getInstance();
  @State description: string = '';
  @State tags: string[] = [];
  @State currentTag: string = '';
  @State imageUrls: string[] = [];
  @State isPublishing: boolean = false;
  @State currentLocation: LocationInfo | null = null;

  async aboutToAppear() {
    // 订阅状态变化
    this.stateManager.subscribe('create_moment_page', (state: GlobalState) => {
      this.globalState = state;
    });

    // 请求媒体权限
    await this.requestPermissions();

    // 获取当前位置
    await this.getCurrentLocation();
  }

  /**
   * 请求媒体权限
   */
  private async requestPermissions(): Promise<void> {
    try {
      // 临时注释，等待用户提供正确的媒体权限实现
      // const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      // if (context) {
      //   await requestMediaPermissions(context);
      // }
      hilog.info(DOMAIN_NUMBER, TAG, '媒体权限请求功能待实现');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `请求权限失败: ${JSON.stringify(error)}`);
    }
  }

  aboutToDisappear() {
    this.stateManager.unsubscribe('create_moment_page');
  }

  /**
   * 获取当前位置
   */
  private async getCurrentLocation(): Promise<void> {
    try {
      if (this.globalState.currentLocation) {
        this.currentLocation = {
          latitude: this.globalState.currentLocation.latitude,
          longitude: this.globalState.currentLocation.longitude,
          locationName: this.globalState.currentLocation.locationName || '南天门'
        };
      } else {
        const location = await LocationUtil.getCurrentLocation();
        if (location) {
          this.currentLocation = {
            latitude: location.latitude,
            longitude: location.longitude,
            locationName: location.locationName || '南天门'
          };
        }
      }
    } catch (error) {
      SystemToast.showError('获取位置信息失败', this.getUIContext());
    }
  }

  /**
   * 添加标签
   */
  private addTag(): void {
    const tag = this.currentTag.trim();
    if (!tag) {
      return;
    }

    if (this.tags.length >= 10) {
      SystemToast.show('最多只能添加10个标签', this.getUIContext());
      return;
    }

    if (this.tags.includes(tag)) {
      SystemToast.show('标签已存在', this.getUIContext());
      return;
    }

    this.tags.push(tag);
    this.currentTag = '';
  }

  /**
   * 删除标签
   */
  private removeTag(index: number) {
    this.tags.splice(index, 1);
  }

  /**
   * 选择图片 - 使用HarmonyOS原生图片选择器
   */
  private async selectImages(): Promise<void> {
    try {
      const maxCount = 9 - this.imageUrls.length;
      if (maxCount <= 0) {
        SystemToast.show('最多只能选择9张图片', this.getUIContext());
        return;
      }

      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const selectedImages = await selectImages(context, maxCount);
      for (const imageUri of selectedImages) {
        this.imageUrls.push(imageUri);
      }
      
      if (selectedImages.length > 0) {
        SystemToast.show(`已选择${selectedImages.length}张图片`, this.getUIContext());
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `选择图片失败: ${JSON.stringify(error)}`);
      SystemToast.show('选择图片失败，请重试', this.getUIContext());
    }
  }

  /**
   * 拍照 - 使用HarmonyOS原生相机
   */
  private async capturePhoto(): Promise<void> {
    try {
      if (this.imageUrls.length >= 9) {
        SystemToast.show('最多只能选择9张图片', this.getUIContext());
        return;
      }

      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const capturedImageUri = await capturePhoto(context);
      if (capturedImageUri) {
        this.imageUrls.push(capturedImageUri);
        SystemToast.show('拍照成功', this.getUIContext());
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `拍照失败: ${JSON.stringify(error)}`);
      SystemToast.show('拍照失败，请重试', this.getUIContext());
    }
  }

  /**
   * 删除图片
   */
  private removeImage(index: number) {
    this.imageUrls.splice(index, 1);
  }

  /**
   * 发布瞬间
   */
  private async publishMoment(): Promise<void> {
    // 验证输入
    if (!this.description.trim()) {
      SystemToast.show('请输入瞬间描述', this.getUIContext());
      return;
    }

    if (!this.currentLocation) {
      SystemToast.show('请先获取位置信息', this.getUIContext());
      return;
    }

    this.isPublishing = true;
    try {
      const response = await ApiClient.createMoment({
        description: this.description.trim(),
        latitude: this.currentLocation.latitude,
        longitude: this.currentLocation.longitude,
        location_name: this.currentLocation.locationName || "南天门",
        tags: this.tags.length > 0 ? this.tags : undefined,
        images: this.imageUrls.length > 0 ? this.imageUrls : undefined
      });

      if (response.code === 0) {
        SystemToast.showSuccess('瞬间发布成功', this.getUIContext());

        // 返回上一页
        const routerInstance = this.getUIContext().getRouter();
        routerInstance.back();
      } else {
        SystemToast.showError(response.msg || '发布瞬间失败', this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError('发布瞬间失败', this.getUIContext());
    } finally {
      this.isPublishing = false;
    }
  }

  /**
   * 刷新位置
   */
  private async refreshLocation(): Promise<void> {
    try {
      const location = await LocationUtil.getCurrentLocation();
      if (location) {
        this.currentLocation = {
          latitude: location.latitude,
          longitude: location.longitude,
          locationName: location.locationName || '南天门'
        };
        SystemToast.show('位置已更新', this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError('获取位置失败', this.getUIContext());
    }
  }

  /**
   * 构建标签项
   */
  @Builder
  buildTagItem(tag: string, index: number) {
    Row() {
      Text(`#${tag}`)
        .fontSize(12)
        .fontColor($r('sys.color.ohos_id_color_emphasize'))

      IconButton({
        icon: AppIcon.CLOSE,
        iconSize: IconSize.SMALL,
        bgColor: '#00000000',
        paddingSize: 2,
        onButtonClick: (): void => this.removeTag(index)
      })
      .margin({ left: 4 })
    }
    .padding({
      left: 8,
      right: 4,
      top: 4,
      bottom: 4
    })
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
    .borderRadius(12)
    .margin({ right: 8, bottom: 8 })
  }

  /**
   * 构建图片项
   */
  @Builder
  buildImageItem(imageUrl: string, index: number) {
    Stack() {
      Image(imageUrl)
        .width(80)
        .height(80)
        .borderRadius(8)
        .objectFit(ImageFit.Cover)
        .backgroundColor($r('sys.color.ohos_id_color_sub_background'))

      Button() {
        Image($r('sys.media.ohos_ic_public_cancel'))
          .width(16)
          .height(16)
          .fillColor(Color.White)
      }
      .type(ButtonType.Circle)
      .backgroundColor('rgba(0,0,0,0.6)')
      .width(24)
      .height(24)
      .onClick(() => this.removeImage(index))
    }
    .width(80)
    .height(80)
    .margin({ right: 8, bottom: 8 })
    .alignContent(Alignment.TopEnd)
  }

  build() {
    Column() {
      // 标题栏
      Row() {
        IconButton({
          icon: AppIcon.BACK,
          iconSize: IconSize.LARGE,
          bgColor: '#00000000',
          onButtonClick: () => {
            const routerInstance = this.getUIContext().getRouter();
            routerInstance.back();
          }
        })

        Text('发布瞬间')
          .fontSize(18)
          .fontWeight(FontWeight.Medium)
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        Button('发布')
          .type(ButtonType.Normal)
          .backgroundColor('#00000000')
          .fontColor($r('sys.color.ohos_id_color_emphasize'))
          .fontSize(16)
          .enabled(!this.isPublishing)
          .onClick(() => this.publishMoment())
      }
      .width('100%')
      .height(56)
      .padding({ left: 16, right: 16 })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      Scroll() {
        Column() {
          // 内容输入
          Column() {
            Text('分享你的生活瞬间')
              .fontSize(16)
              .fontWeight(FontWeight.Medium)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))
              .width('100%')
              .margin({ bottom: 16 })

            TextArea({ placeholder: '写下你想分享的内容...' })
              .onChange((value: string) => {
                this.description = value;
              })
              .maxLength(500)
              .height(120)
              .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
              .borderRadius(8)
          }
          .width('100%')
          .padding(16)
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(12)
          .margin({ bottom: 16 })

          // 图片选择
          Column() {
            Row() {
              Text('添加图片')
                .fontSize(16)
                .fontWeight(FontWeight.Medium)
                .fontColor($r('sys.color.ohos_id_color_text_primary'))
                .layoutWeight(1)

              Text(`${this.imageUrls.length}/9`)
                .fontSize(12)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            }
            .width('100%')
            .margin({ bottom: 16 })

            // 图片网格
            Flex({ wrap: FlexWrap.Wrap }) {
              // 已选择的图片
              ForEach(this.imageUrls, (imageUrl: string, index: number) => {
                this.buildImageItem(imageUrl, index)
              })

              // 添加图片按钮
              if (this.imageUrls.length < 9) {
                Button() {
                  Column() {
                    Image($r('sys.media.ohos_ic_public_add'))
                      .width(24)
                      .height(24)
                      .fillColor($r('sys.color.ohos_id_color_text_secondary'))

                    Text('相册')
                      .fontSize(10)
                      .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                      .margin({ top: 4 })
                  }
                }
                .type(ButtonType.Normal)
                .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
                .width(80)
                .height(80)
                .borderRadius(8)
                .onClick(() => this.selectImages())
              }

              // 拍照按钮
              if (this.imageUrls.length < 9) {
                Button() {
                  Column() {
                    Image($r('sys.media.ohos_ic_public_camera'))
                      .width(24)
                      .height(24)
                      .fillColor($r('sys.color.ohos_id_color_text_secondary'))

                    Text('拍照')
                      .fontSize(10)
                      .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                      .margin({ top: 4 })
                  }
                }
                .type(ButtonType.Normal)
                .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
                .width(80)
                .height(80)
                .borderRadius(8)
                .onClick(() => this.capturePhoto())
              }
            }
            .width('100%')
          }
          .width('100%')
          .padding(16)
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(12)
          .margin({ bottom: 16 })

          // 标签管理
          Column() {
            Text('添加标签')
              .fontSize(16)
              .fontWeight(FontWeight.Medium)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))
              .width('100%')
              .margin({ bottom: 16 })

            // 标签输入
            Row() {
              TextInput({ placeholder: '输入标签名称', text: this.currentTag })
                .onChange((value: string) => {
                  this.currentTag = value;
                })
                .layoutWeight(1)
                .maxLength(20)

              Button('添加')
                .type(ButtonType.Capsule)
                .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
                .fontColor(Color.White)
                .fontSize(14)
                .onClick(() => this.addTag())
                .margin({ left: 8 })
            }
            .width('100%')
            .margin({ bottom: 16 })

            // 已添加的标签
            if (this.tags.length > 0) {
              Flex({ wrap: FlexWrap.Wrap }) {
                ForEach(this.tags, (tag: string, index: number) => {
                  this.buildTagItem(tag, index)
                })
              }
              .width('100%')
            }
          }
          .width('100%')
          .padding(16)
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(12)
          .margin({ bottom: 16 })

          // 位置信息
          Column() {
            Row() {
              Text('位置信息')
                .fontSize(16)
                .fontWeight(FontWeight.Medium)
                .fontColor($r('sys.color.ohos_id_color_text_primary'))
                .layoutWeight(1)

              Button('刷新')
                .type(ButtonType.Normal)
                .backgroundColor(Color.Transparent)
                .fontColor($r('sys.color.ohos_id_color_emphasize'))
                .fontSize(14)
                .onClick(() => this.refreshLocation())
            }
            .width('100%')
            .margin({ bottom: 16 })

            if (this.currentLocation) {
              Row() {
                Image($r('sys.media.ohos_ic_public_location'))
                  .width(20)
                  .height(20)
                  .fillColor($r('sys.color.ohos_id_color_emphasize'))

                Column() {
                  Text(this.currentLocation.locationName)
                    .fontSize(14)
                    .fontColor($r('sys.color.ohos_id_color_text_primary'))
                    .width('100%')

                  Text(`${this.currentLocation.latitude.toFixed(6)}, ${this.currentLocation.longitude.toFixed(6)}`)
                    .fontSize(12)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                    .width('100%')
                    .margin({ top: 2 })
                }
                .alignItems(HorizontalAlign.Start)
                .layoutWeight(1)
                .margin({ left: 8 })
              }
              .width('100%')
              .padding(12)
              .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
              .borderRadius(8)
            } else {
              Text('正在获取位置信息...')
                .fontSize(14)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                .width('100%')
                .textAlign(TextAlign.Center)
                .padding(20)
            }
          }
          .width('100%')
          .padding(16)
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(12)
          .margin({ bottom: 16 })

          // 发布按钮
          Button('发布瞬间')
            .type(ButtonType.Capsule)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .width('100%')
            .height(44)
            .enabled(!this.isPublishing)
            .onClick(() => this.publishMoment())
            .margin({ bottom: 32 })
        }
        .width('100%')
        .padding(16)
      }
      .layoutWeight(1)
      .scrollBar(BarState.Auto)
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }
}
