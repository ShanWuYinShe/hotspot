use std::sync::Arc;
use std::collections::HashSet;
use tracing::{info, warn};
use uuid;

use crate::repositories::{PgUserRepository, user_repository::UserRepository};
use crate::repositories::RepositoryResult;

/// 通知策略配置
#[derive(Debug, Clone)]
pub struct NotificationPolicy {
    /// 是否启用好友关系通知（用户状态变化）
    pub enable_friend_notifications: bool,
    /// 是否启用附近用户通知（用户状态变化）
    pub enable_nearby_notifications: bool,
    /// 附近用户通知的半径（公里）
    pub nearby_radius_km: f64,
    /// 是否启用全局广播（开发/测试用）
    pub enable_global_broadcast: bool,
}

/// 群组事件通知策略配置
#[derive(Debug, Clone)]
pub struct GroupNotificationPolicy {
    /// 是否启用群组加入/离开通知
    pub enable_member_join_leave_notifications: bool,
    /// 是否启用群组消息通知
    pub enable_group_message_notifications: bool,
    /// 是否启用群组设置变更通知
    pub enable_group_settings_notifications: bool,
}

impl Default for NotificationPolicy {
    fn default() -> Self {
        Self {
            enable_friend_notifications: true,
            enable_nearby_notifications: true,
            nearby_radius_km: 5.0, // 5公里内的用户
            enable_global_broadcast: false, // 默认关闭全局广播
        }
    }
}

impl Default for GroupNotificationPolicy {
    fn default() -> Self {
        Self {
            enable_member_join_leave_notifications: true,
            enable_group_message_notifications: true,
            enable_group_settings_notifications: true,
        }
    }
}

/// 智能通知策略服务
/// 根据用户关系（好友、位置）决定应该向哪些用户发送用户状态通知
/// 群组相关的通知由单独的群组通知服务处理
#[derive(Debug, Clone)]
pub struct NotificationPolicyService {
    user_repository: Arc<PgUserRepository>,
    policy: NotificationPolicy,
    group_policy: GroupNotificationPolicy,
}

impl NotificationPolicyService {
    pub fn new(
        user_repository: Arc<PgUserRepository>,
        policy: Option<NotificationPolicy>,
        group_policy: Option<GroupNotificationPolicy>,
    ) -> Self {
        Self {
            user_repository,
            policy: policy.unwrap_or_default(),
            group_policy: group_policy.unwrap_or_default(),
        }
    }

    /// 获取应该接收用户状态更新通知的用户ID列表
    /// 注意：这只包括好友和附近用户，不包括群组成员
    /// 群组相关的通知应该通过群组事件通知机制处理
    pub async fn get_presence_notification_targets(
        &self,
        user_id: i64,
    ) -> RepositoryResult<Vec<i64>> {
        let mut target_users = HashSet::new();

        // 如果启用全局广播（开发/测试模式）
        if self.policy.enable_global_broadcast {
            warn!("全局广播模式已启用，这将向所有用户发送通知！");
            // 返回空列表，让调用方处理全局广播
            return Ok(vec![]);
        }

        // 1. 获取好友列表
        if self.policy.enable_friend_notifications {
            match self.user_repository.find_user_friends(user_id).await {
                Ok(friends) => {
                    for friend in friends {
                        target_users.insert(friend.id);
                    }
                    info!("已添加 {} 个好友到状态通知目标", target_users.len());
                }
                Err(e) => {
                    warn!("获取好友列表失败: {:?}", e);
                }
            }
        }

        // 2. 获取附近用户
        if self.policy.enable_nearby_notifications {
            // TODO: 需要获取用户的当前位置才能查找附近用户
            // match self.user_repository.find_nearby_users(longitude, latitude, self.policy.nearby_radius_km, 0, 100, user_id).await {
            //     Ok((nearby_users, _, _)) => {
            //         let before_count = target_users.len();
            //         for user in nearby_users {
            //             target_users.insert(user.user_id);
            //         }
            //         info!("已添加 {} 个附近用户到状态通知目标", target_users.len() - before_count);
            //     }
            //     Err(e) => {
            //         warn!("获取附近用户失败: {:?}", e);
            //     }
            // }
            warn!("附近用户通知暂时禁用，需要用户位置信息");
        }

        let final_targets: Vec<i64> = target_users.into_iter().collect();
        info!("用户 {} 的状态更新将通知 {} 个目标用户（好友+附近用户）", user_id, final_targets.len());

        Ok(final_targets)
    }

    /// 检查是否应该向特定用户发送用户状态通知
    #[allow(dead_code)]
    pub async fn should_notify_user(
        &self,
        source_user_id: i64,
        target_user_id: i64,
    ) -> RepositoryResult<bool> {
        if source_user_id == target_user_id {
            return Ok(false);
        }

        let targets = self.get_presence_notification_targets(source_user_id).await?;
        Ok(targets.contains(&target_user_id))
    }

    /// 获取群组事件通知的目标用户（群组成员）
    /// 用于群组相关事件：新成员加入、成员离开、群组消息等
    #[allow(dead_code)]
    pub async fn get_group_event_notification_targets(
        &self,
        _user_id: i64,
        group_id: Option<uuid::Uuid>,
    ) -> RepositoryResult<Vec<i64>> {
        // 如果指定了群组ID，获取该群组的所有成员
        if let Some(_gid) = group_id {
            // TODO: 需要实现获取群组成员的方法
            return Ok(vec![]);  // 临时修复
            // return self.group_repository.get_group_members(gid, 0, 1000).await
            //     .map(|(members, _, _)| members.into_iter().map(|(user_id, _, _, _, _, _)| user_id).collect());
        }

        // 否则获取用户所在所有群组的成员
        // TODO: 需要实现获取用户群组成员的方法
        Ok(vec![])  // 临时修复
        // self.user_repository.find_group_members_for_user(user_id).await
    }

    /// 检查是否应该发送群组事件通知
    #[allow(dead_code)]
    pub fn should_send_group_notification(&self, event_type: &str) -> bool {
        match event_type {
            "member_join" | "member_leave" => self.group_policy.enable_member_join_leave_notifications,
            "group_message" => self.group_policy.enable_group_message_notifications,
            "group_settings" => self.group_policy.enable_group_settings_notifications,
            _ => false,
        }
    }

    /// 更新用户状态通知策略
    #[allow(dead_code)]
    pub fn update_policy(&mut self, new_policy: NotificationPolicy) {
        self.policy = new_policy;
        info!("用户状态通知策略已更新: {:?}", self.policy);
    }

    /// 更新群组事件通知策略
    #[allow(dead_code)]
    pub fn update_group_policy(&mut self, new_group_policy: GroupNotificationPolicy) {
        self.group_policy = new_group_policy;
        info!("群组事件通知策略已更新: {:?}", self.group_policy);
    }

    /// 获取当前用户状态通知策略
    pub fn get_policy(&self) -> &NotificationPolicy {
        &self.policy
    }

    /// 获取当前群组事件通知策略
    pub fn get_group_policy(&self) -> &GroupNotificationPolicy {
        &self.group_policy
    }
}
