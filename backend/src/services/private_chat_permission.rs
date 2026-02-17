use crate::error::Result;
use crate::repositories::user::user_repo::UserRow;
use crate::repositories::user_blocks_repo::UserBlocksRepository;
use crate::repositories::user_follows_repo::UserFollowsRepository;
use crate::repositories::user_settings_repo::{UserSettingsRepository, UserSettingsRow};
use crate::types::enums::{CantMessageReason, PrivateChatPermission, UserStatus};
use tokio::try_join;

#[derive(Clone)]
pub struct PrivateChatPermissionResolver {
    settings_repo: UserSettingsRepository,
    follows_repo: UserFollowsRepository,
    blocks_repo: UserBlocksRepository,
}

#[derive(Debug, Clone)]
pub struct PermissionCheck {
    pub can_message: bool,
    pub reason: Option<CantMessageReason>,
    pub is_mutual_friend: bool,
}

impl PrivateChatPermissionResolver {
    pub fn new(
        settings_repo: UserSettingsRepository,
        follows_repo: UserFollowsRepository,
        blocks_repo: UserBlocksRepository,
    ) -> Self {
        Self {
            settings_repo,
            follows_repo,
            blocks_repo,
        }
    }

    pub async fn evaluate(
        &self,
        current_user_id: i64,
        target_user_id: i64,
        target_user: Option<&UserRow>,
    ) -> Result<PermissionCheck> {
        if current_user_id == target_user_id {
            return Ok(PermissionCheck {
                can_message: false,
                reason: Some(CantMessageReason::Privacy),
                is_mutual_friend: false,
            });
        }

        if let Some(user) = target_user {
            if user.status == UserStatus::Banned {
                return Ok(PermissionCheck {
                    can_message: false,
                    reason: Some(CantMessageReason::AccountSuspended),
                    is_mutual_friend: false,
                });
            }
        } else {
            return Ok(PermissionCheck {
                can_message: false,
                reason: Some(CantMessageReason::AccountSuspended),
                is_mutual_friend: false,
            });
        }

        if self
            .blocks_repo
            .is_blocked(target_user_id, current_user_id)
            .await?
            || self
                .blocks_repo
                .is_blocked(current_user_id, target_user_id)
                .await?
        {
            return Ok(PermissionCheck {
                can_message: false,
                reason: Some(CantMessageReason::Privacy),
                is_mutual_friend: false,
            });
        }

        let settings = self
            .settings_repo
            .find_by_user_id(target_user_id)
            .await?
            .map(|row| row);
        let permission = parse_permission(settings.as_ref());

        let (follows_other, other_follows) = try_join!(
            self.follows_repo
                .is_following(current_user_id, target_user_id),
            self.follows_repo
                .is_following(target_user_id, current_user_id)
        )?;
        let is_mutual_friend = follows_other && other_follows;

        match permission {
            PrivateChatPermission::Everyone => Ok(PermissionCheck {
                can_message: true,
                reason: None,
                is_mutual_friend,
            }),
            PrivateChatPermission::Nobody => Ok(PermissionCheck {
                can_message: false,
                reason: Some(CantMessageReason::Privacy),
                is_mutual_friend,
            }),
            PrivateChatPermission::Friends => {
                if is_mutual_friend {
                    Ok(PermissionCheck {
                        can_message: true,
                        reason: None,
                        is_mutual_friend: true,
                    })
                } else {
                    Ok(PermissionCheck {
                        can_message: false,
                        reason: Some(CantMessageReason::Privacy),
                        is_mutual_friend: false,
                    })
                }
            }
        }
    }
}

fn parse_permission(settings: Option<&UserSettingsRow>) -> PrivateChatPermission {
    settings
        .and_then(|s| s.privacy_settings.get("allow_private_chat"))
        .and_then(|v| v.as_str())
        .and_then(|value| value.parse::<PrivateChatPermission>().ok())
        .unwrap_or(PrivateChatPermission::Everyone)
}
