// src/services/group_message_service.rs
// 群聊消息Service

use redis::aio::ConnectionManager;
use sqlx::PgPool;
use std::sync::Arc;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            AddGroupMessageReactionRequest, DeleteGroupMessageRequest,
            GetGroupMessageReactionsRequest, GetGroupMessagesRequest, GetMessageReadStatusRequest,
            MarkGroupMessagesReadRequest, RecallGroupMessageRequest, SendGroupMessageRequest,
            UpdateGroupMessageRequest,
        },
        response::{
            GroupMessageItem, MessageReactionItem, MessageReadItem, PaginatedData, PaginationMeta,
            SendGroupMessageData, UserInfo,
        },
    },
    error::{AppError, Result},
    repositories::{
        group_message_repo::GroupMessageRepository,
        group_member_repo::GroupMemberRepository,
        user_repo::UserRepository,
        chat_unread_repo::ChatUnreadRepository,
    },
    types::enums::{MessageType, MemberRole},
    utils::validation::Validator,
    EventBus,
};

#[derive(Clone)]
pub struct GroupMessageService {
    db: PgPool,
    #[allow(dead_code)]
    redis: ConnectionManager,
    #[allow(dead_code)]
    event_bus: Arc<EventBus>,
    message_repo: GroupMessageRepository,
    member_repo: GroupMemberRepository,
    user_repo: UserRepository,
    unread_repo: ChatUnreadRepository,
}

impl GroupMessageService {
    pub fn new(db: &PgPool, redis: &ConnectionManager, event_bus: Arc<EventBus>) -> Self {
        Self {
            message_repo: GroupMessageRepository::new(db.clone()),
            member_repo: GroupMemberRepository::new(db.clone()),
            user_repo: UserRepository::new(db.clone()),
            unread_repo: ChatUnreadRepository::new(db.clone()),
            db: db.clone(),
            redis: redis.clone(),
            event_bus,
        }
    }

    /// 发送群聊消息
    pub async fn send(
        &self,
        req: &SendGroupMessageRequest,
        user_id: i64,
    ) -> Result<SendGroupMessageData> {
        // 1. 参数验证
        Validator::validate_length(&req.content, "消息内容", 1, 10000)?;

        let group_chat_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
        
        let message_type = MessageType::from_str(&req.r#type)
            .ok_or_else(|| AppError::BadRequest("无效的消息类型".to_string()))?;

        let reply_to_message_id = if let Some(ref id) = req.reply_to {
            Some(Uuid::parse_str(id).map_err(|_| AppError::BadRequest("无效的回复消息ID".to_string()))?)
        } else {
            None
        };

        // 解析 mentions 中的 user_id
        let mentions = if let Some(ref mention_ids) = req.mentions {
            let mut ids = Vec::new();
            for id_str in mention_ids {
                if let Ok(id) = id_str.parse::<i64>() {
                    ids.push(id);
                }
            }
            if ids.is_empty() { None } else { Some(ids) }
        } else {
            None
        };

        // 2. 验证用户是否是群成员
        let member = self.member_repo.find_by_user_and_group(group_chat_id, user_id).await?
            .ok_or_else(|| AppError::Forbidden("您不是此群组的成员".to_string()))?;

        // 检查是否被禁言
        if let Some(muted_until) = member.muted_until {
            if muted_until > chrono::Utc::now() {
                return Err(AppError::Forbidden("您已被禁言，无法发送消息".to_string()));
            }
        }

        // 3. 保存到数据库
        let created = self.message_repo.create(
            group_chat_id,
            user_id,
            message_type,
            req.content.clone(),
            reply_to_message_id,
            mentions,
        ).await?;

        // 4. 更新群聊最后活跃时间（通过更新updated_at字段）
        sqlx::query!(
            r#"UPDATE group_chats SET updated_at = NOW() WHERE group_chat_id = $1"#,
            group_chat_id
        )
        .execute(&self.db)
        .await?;

        Ok(SendGroupMessageData {
            id: created.message_id.to_string(),
            sent_at: created.sent_at.to_rfc3339(),
        })
    }

    /// 获取群聊消息历史
    pub async fn get_history(
        &self,
        req: &GetGroupMessagesRequest,
        user_id: i64,
    ) -> Result<PaginatedData<GroupMessageItem>> {
        // 1. 参数验证
        let limit = req.limit.unwrap_or(20).min(100) as i64;
        let group_chat_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        // 2. 验证用户是否是群成员
        if self.member_repo.find_by_user_and_group(group_chat_id, user_id).await?.is_none() {
            return Err(AppError::Forbidden("您不是此群组的成员".to_string()));
        }

        // 3. 解析游标
        let cursor = req.cursor.as_ref()
            .map(|c| Uuid::parse_str(c))
            .transpose()
            .map_err(|_| AppError::BadRequest("无效的游标".to_string()))?;

        // 4. 查询消息历史
        let messages = self.message_repo.list_by_group(group_chat_id, limit + 1, cursor).await?;

        let has_more = messages.len() > limit as usize;
        let messages: Vec<_> = messages.into_iter().take(limit as usize).collect();
        let next_cursor = if has_more {
            messages.last().map(|m| m.cursor_id.to_string())
        } else {
            None
        };

        // 5. 批量获取发送者信息
        let sender_ids: Vec<i64> = messages.iter().map(|m| m.sender_id).collect();
        let users = self.user_repo.find_by_ids(&sender_ids).await?;
        let user_map: std::collections::HashMap<i64, _> = users.into_iter()
            .map(|u| (u.user_id, u))
            .collect();

        // 6. 转换为DTO
        let items: Vec<GroupMessageItem> = messages
            .iter()
            .map(|m| {
                let user = user_map.get(&m.sender_id);
                GroupMessageItem {
                    id: m.message_id.to_string(),
                    r#type: m.r#type.to_string(),
                    content: m.content.clone(),
                    sender: UserInfo {
                        user_id: m.sender_id.to_string(),
                        nickname: user.map(|u| u.nickname.clone()).unwrap_or_else(|| "未知用户".to_string()),
                        avatar_url: user.and_then(|u| u.avatar_url.clone()),
                    },
                    reply_to: m.reply_to_message_id.map(|id| id.to_string()),
                    mentions: m
                        .mentions
                        .as_ref()
                        .map(|ids| ids.iter().map(|id| id.to_string()).collect()),
                    sent_at: m.sent_at.to_rfc3339(),
                    edited_at: m.edited_at.map(|t| t.to_rfc3339()),
                    is_recalled: m.is_recalled,
                    recalled_by: m.recalled_by.map(|id| id.to_string()),
                    recalled_at: m.recalled_at.map(|t| t.to_rfc3339()),
                    read_count: None, // 已读人数需要单独查询
                }
            })
            .collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }

    /// 编辑群聊消息
    pub async fn update(&self, req: &UpdateGroupMessageRequest, user_id: i64) -> Result<()> {
        // 1. 参数验证
        Validator::validate_length(&req.content, "消息内容", 1, 10000)?;
        let message_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 2. 查询消息
        let message = self.message_repo
            .find_by_id(message_id)
            .await?
            .ok_or_else(|| AppError::NotFound("消息不存在".to_string()))?;

        // 3. 验证权限（只能编辑自己的消息）
        if message.sender_id != user_id {
            return Err(AppError::Forbidden("无权编辑此消息".to_string()));
        }

        // 4. 验证消息状态
        if message.is_recalled {
            return Err(AppError::BadRequest("已撤回的消息无法编辑".to_string()));
        }

        // 5. 更新消息
        self.message_repo.update_content(message_id, req.content.clone()).await?;

        Ok(())
    }

    /// 撤回群聊消息
    pub async fn recall(&self, req: &RecallGroupMessageRequest, user_id: i64) -> Result<()> {
        let message_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 1. 查询消息
        let message = self.message_repo
            .find_by_id(message_id)
            .await?
            .ok_or_else(|| AppError::NotFound("消息不存在".to_string()))?;

        // 2. 验证权限
        let can_recall = if message.sender_id == user_id {
            // 发送者可以撤回自己的消息（2分钟内）
            let now = chrono::Utc::now();
            let elapsed = now.signed_duration_since(message.sent_at);
            elapsed.num_minutes() < 2
        } else {
            // 检查是否是管理员/群主
            let member = self.member_repo.find_by_user_and_group(message.group_chat_id, user_id).await?;
            member.map(|m| m.role == MemberRole::Owner || m.role == MemberRole::Admin).unwrap_or(false)
        };

        if !can_recall {
            return Err(AppError::Forbidden("无权撤回此消息".to_string()));
        }

        // 3. 验证消息状态
        if message.is_recalled {
            return Err(AppError::BadRequest("消息已被撤回".to_string()));
        }

        // 4. 撤回消息
        self.message_repo.recall(message_id, user_id).await?;

        Ok(())
    }

    /// 删除群聊消息
    pub async fn delete(&self, req: &DeleteGroupMessageRequest, user_id: i64) -> Result<()> {
        let message_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 1. 查询消息
        let message = self.message_repo
            .find_by_id(message_id)
            .await?
            .ok_or_else(|| AppError::NotFound("消息不存在".to_string()))?;

        // 2. 验证权限
        let for_everyone = req.for_everyone.unwrap_or(false);

        if for_everyone {
            // 为所有人删除（仅管理员/群主）
            let member = self.member_repo.find_by_user_and_group(message.group_chat_id, user_id).await?;
            let is_admin = member.map(|m| m.role == MemberRole::Owner || m.role == MemberRole::Admin).unwrap_or(false);
            if !is_admin {
                return Err(AppError::Forbidden("无权为所有人删除消息".to_string()));
            }
        } else {
            // 仅为自己删除（需要是群成员）
            if self.member_repo.find_by_user_and_group(message.group_chat_id, user_id).await?.is_none() {
                return Err(AppError::Forbidden("您不是此群组的成员".to_string()));
            }
        }

        // 3. 删除消息
        self.message_repo.soft_delete(message_id).await?;

        Ok(())
    }

    /// 标记群聊消息已读
    pub async fn mark_as_read(
        &self,
        req: &MarkGroupMessagesReadRequest,
        user_id: i64,
    ) -> Result<()> {
        let group_chat_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        // 验证用户是否是群成员
        if self.member_repo.find_by_user_and_group(group_chat_id, user_id).await?.is_none() {
            return Err(AppError::Forbidden("您不是此群组的成员".to_string()));
        }

        // 解析截止消息ID
        let up_to = req.up_to.as_ref()
            .map(|id| Uuid::parse_str(id))
            .transpose()
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 更新已读状态
        self.unread_repo.mark_as_read("group_chat", &group_chat_id.to_string(), user_id, up_to).await?;

        Ok(())
    }

    /// 获取消息已读状态
    pub async fn get_read_status(
        &self,
        req: &GetMessageReadStatusRequest,
        user_id: i64,
    ) -> Result<PaginatedData<MessageReadItem>> {
        let limit = req.limit.unwrap_or(20).min(100);
        let message_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 获取消息
        let message = self.message_repo.find_by_id(message_id).await?
            .ok_or_else(|| AppError::NotFound("消息不存在".to_string()))?;

        // 验证用户是否是群成员
        if self.member_repo.find_by_user_and_group(message.group_chat_id, user_id).await?.is_none() {
            return Err(AppError::Forbidden("您不是此群组的成员".to_string()));
        }

        // 查询已读状态
        let read_users = self.message_repo.get_read_status(message_id).await?;

        // 批量获取用户信息
        let users = self.user_repo.find_by_ids(&read_users).await?;
        let user_map: std::collections::HashMap<i64, _> = users.into_iter()
            .map(|u| (u.user_id, u))
            .collect();

        // 转换为DTO
        let items: Vec<MessageReadItem> = read_users.iter().map(|&uid| {
            let user = user_map.get(&uid);
            MessageReadItem {
                user: UserInfo {
                    user_id: uid.to_string(),
                    nickname: user.map(|u| u.nickname.clone()).unwrap_or_else(|| "未知用户".to_string()),
                    avatar_url: user.and_then(|u| u.avatar_url.clone()),
                },
                read_at: chrono::Utc::now().to_rfc3339(), // 实际应从数据库获取
            }
        }).take(limit as usize).collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: false,
                next_cursor: None,
            },
        })
    }

    /// 添加消息反应
    /// 注意：需要先创建 group_message_reactions 表
    pub async fn add_reaction(
        &self,
        req: &AddGroupMessageReactionRequest,
        user_id: i64,
    ) -> Result<()> {
        let message_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 获取消息
        let message = self.message_repo.find_by_id(message_id).await?
            .ok_or_else(|| AppError::NotFound("消息不存在".to_string()))?;

        // 验证用户是否是群成员
        if self.member_repo.find_by_user_and_group(message.group_chat_id, user_id).await?.is_none() {
            return Err(AppError::Forbidden("您不是此群组的成员".to_string()));
        }

        // 验证操作类型
        if req.action != "add" && req.action != "remove" {
            return Err(AppError::BadRequest("无效的操作类型".to_string()));
        }

        // 暂时返回成功，等待数据库表创建后实现
        // TODO: 创建 group_message_reactions 表后启用数据库操作
        Ok(())
    }

    /// 获取消息反应列表
    /// 注意：需要先创建 group_message_reactions 表
    pub async fn get_reactions(
        &self,
        req: &GetGroupMessageReactionsRequest,
        user_id: i64,
    ) -> Result<PaginatedData<MessageReactionItem>> {
        let message_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 获取消息
        let message = self.message_repo.find_by_id(message_id).await?
            .ok_or_else(|| AppError::NotFound("消息不存在".to_string()))?;

        // 验证用户是否是群成员
        if self.member_repo.find_by_user_and_group(message.group_chat_id, user_id).await?.is_none() {
            return Err(AppError::Forbidden("您不是此群组的成员".to_string()));
        }

        // 暂时返回空列表，等待数据库表创建后实现
        // TODO: 创建 group_message_reactions 表后启用数据库查询
        Ok(PaginatedData {
            items: vec![],
            pagination: PaginationMeta {
                has_more: false,
                next_cursor: None,
            },
        })
    }
}
