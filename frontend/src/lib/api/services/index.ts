/**
 * API 服务索引
 */

// 认证
export { authService } from './auth';

// 用户
export { usersService } from './users';

// 瞬间
export { momentsService } from './moments';

// 评论
export { commentsService } from './comments';

// 搜索
export { searchService } from './search';

// 信息流
export { feedService } from './feed';

// 私聊
export { privateChatsService, privateMessagesService } from './private-chats';

// 群组
export { groupsService, groupMessagesService } from './groups';

// 媒体
export { mediaService } from './media';

// 地图
export { mapService } from './map';

// 分享
export { shareService } from './share';

// 通知
export { notificationsService } from './notifications';

// 举报
export { reportsService } from './reports';

// 系统
export { systemService } from './system';

// 管理员
export {
  adminRbacService,
  adminUsersService,
  adminGroupsService,
  adminAnnouncementsService,
  adminAuditService,
  adminReportsService,
  adminTagsService,
  adminAppConfigService
} from './admin';
