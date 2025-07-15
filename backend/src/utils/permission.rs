use sqlx::PgPool;
use tracing::error;
use uuid::Uuid;

use crate::utils::response::AppError;
use crate::models::constants::group_roles;

/// 检查用户是否是群组成员
pub async fn check_group_member(
    db: &PgPool,
    group_id: &Uuid,
    user_id: i64,
) -> Result<bool, AppError> {
    let is_member = sqlx::query_scalar!(
        r#"
        SELECT EXISTS (
            SELECT 1 FROM group_members 
            WHERE group_id = $1 AND user_id = $2
        ) as "exists!"
        "#,
        group_id,
        user_id
    )
    .fetch_one(db)
    .await
    .map_err(|e| {
        error!("检查群组成员失败: {:?}", e);
        AppError::InternalServerError("检查群组成员权限失败".to_string())
    })?;

    Ok(is_member)
}

/// 验证用户是否群主或管理员
pub async fn check_group_admin(
    db: &PgPool,
    group_id: &Uuid,
    user_id: i64,
) -> Result<bool, AppError> {
    let is_admin = sqlx::query_scalar!(
        r#"
        SELECT EXISTS (
            SELECT 1 FROM group_members 
            WHERE group_id = $1 AND user_id = $2 AND role IN ($3, $4)
        ) as "exists!"
        "#,
        group_id,
        user_id,
        group_roles::ADMIN,
        group_roles::OWNER
    )
    .fetch_one(db)
    .await
    .map_err(|e| {
        error!("检查群组管理权限失败: {:?}", e);
        AppError::InternalServerError("检查群组管理权限失败".to_string())
    })?;

    Ok(is_admin)
}

/// 验证用户是否群主
pub async fn check_group_owner(
    db: &PgPool,
    group_id: &Uuid,
    user_id: i64,
) -> Result<bool, AppError> {
    let is_owner = sqlx::query_scalar!(
        r#"
        SELECT EXISTS (
            SELECT 1 FROM groups 
            WHERE group_id = $1 AND owner_id = $2
        ) as "exists!"
        "#,
        group_id,
        user_id
    )
    .fetch_one(db)
    .await
    .map_err(|e| {
        error!("检查群主权限失败: {:?}", e);
        AppError::InternalServerError("检查群主权限失败".to_string())
    })?;

    Ok(is_owner)
}

/// 检查用户是否有权操作签到（验证是否是该签到的创建者）
pub async fn check_checkin_owner(
    db: &PgPool,
    checkin_id: &Uuid,
    user_id: i64,
) -> Result<bool, AppError> {
    let is_owner = sqlx::query_scalar!(
        r#"
        SELECT EXISTS (
            SELECT 1 FROM checkins 
            WHERE checkin_id = $1 AND user_id = $2
        ) as "exists!"
        "#,
        checkin_id,
        user_id
    )
    .fetch_one(db)
    .await
    .map_err(|e| {
        error!("检查签到所有权失败: {:?}", e);
        AppError::InternalServerError("检查签到权限失败".to_string())
    })?;

    Ok(is_owner)
}

/// 检查用户是否有权操作群组消息（验证是否是该消息的发送者）
pub async fn check_group_message_owner(
    db: &PgPool,
    message_id: &Uuid,
    user_id: i64,
) -> Result<bool, AppError> {
    let is_owner = sqlx::query_scalar!(
        r#"
        SELECT EXISTS (
            SELECT 1 FROM group_messages 
            WHERE message_id = $1 AND sender_id = $2
        ) as "exists!"
        "#,
        message_id,
        user_id
    )
    .fetch_one(db)
    .await
    .map_err(|e| {
        error!("检查群组消息所有权失败: {:?}", e);
        AppError::InternalServerError("检查群组消息权限失败".to_string())
    })?;

    Ok(is_owner)
}

/// 检查用户是否有权操作私聊消息（验证是否是该消息的发送者）
pub async fn check_direct_message_owner(
    db: &PgPool,
    message_id: &Uuid,
    user_id: i64,
) -> Result<bool, AppError> {
    let is_owner = sqlx::query_scalar!(
        r#"
        SELECT EXISTS (
            SELECT 1 FROM direct_messages 
            WHERE message_id = $1 AND sender_id = $2
        ) as "exists!"
        "#,
        message_id,
        user_id
    )
    .fetch_one(db)
    .await
    .map_err(|e| {
        error!("检查私聊消息所有权失败: {:?}", e);
        AppError::InternalServerError("检查私聊消息权限失败".to_string())
    })?;

    Ok(is_owner)
}

/// 检查用户是否是对话的参与者
pub async fn check_conversation_member(
    db: &PgPool,
    conversation_id: &Uuid,
    user_id: i64,
) -> Result<bool, AppError> {
    let is_member = sqlx::query_scalar!(
        r#"
        SELECT EXISTS (
            SELECT 1 FROM conversations 
            WHERE conversation_id = $1 AND (user1_id = $2 OR user2_id = $2)
        ) as "exists!"
        "#,
        conversation_id,
        user_id
    )
    .fetch_one(db)
    .await
    .map_err(|e| {
        error!("检查对话成员权限失败: {:?}", e);
        AppError::InternalServerError("检查对话权限失败".to_string())
    })?;

    Ok(is_member)
}

/// 要求用户是群组成员，否则返回错误
pub async fn require_group_member(
    db: &PgPool,
    group_id: &Uuid,
    user_id: i64,
) -> Result<(), AppError> {
    let is_member = check_group_member(db, group_id, user_id).await?;
    if !is_member {
        return Err(AppError::Forbidden("您不是该群组的成员".to_string()));
    }
    Ok(())
}

/// 要求用户是群组管理员，否则返回错误
pub async fn require_group_admin(
    db: &PgPool,
    group_id: &Uuid,
    user_id: i64,
) -> Result<(), AppError> {
    let is_admin = check_group_admin(db, group_id, user_id).await?;
    if !is_admin {
        return Err(AppError::Forbidden("您不是该群组的管理员".to_string()));
    }
    Ok(())
}

/// 要求用户是群主，否则返回错误
pub async fn require_group_owner(
    db: &PgPool,
    group_id: &Uuid,
    user_id: i64,
) -> Result<(), AppError> {
    let is_owner = check_group_owner(db, group_id, user_id).await?;
    if !is_owner {
        return Err(AppError::Forbidden("您不是该群组的群主".to_string()));
    }
    Ok(())
}

/// 要求用户是签到创建者，否则返回错误
pub async fn require_checkin_owner(
    db: &PgPool,
    checkin_id: &Uuid,
    user_id: i64,
) -> Result<(), AppError> {
    let is_owner = check_checkin_owner(db, checkin_id, user_id).await?;
    if !is_owner {
        return Err(AppError::Forbidden("您不是该签到的创建者".to_string()));
    }
    Ok(())
}

/// 要求用户是群组消息发送者，否则返回错误
pub async fn require_group_message_owner(
    db: &PgPool,
    message_id: &Uuid,
    user_id: i64,
) -> Result<(), AppError> {
    let is_owner = check_group_message_owner(db, message_id, user_id).await?;
    if !is_owner {
        return Err(AppError::Forbidden("您不是该群组消息的发送者".to_string()));
    }
    Ok(())
}

/// 要求用户是私聊消息发送者，否则返回错误
pub async fn require_direct_message_owner(
    db: &PgPool,
    message_id: &Uuid,
    user_id: i64,
) -> Result<(), AppError> {
    let is_owner = check_direct_message_owner(db, message_id, user_id).await?;
    if !is_owner {
        return Err(AppError::Forbidden("您不是该私聊消息的发送者".to_string()));
    }
    Ok(())
}

/// 要求用户是对话成员，否则返回错误
pub async fn require_conversation_member(
    db: &PgPool,
    conversation_id: &Uuid,
    user_id: i64,
) -> Result<(), AppError> {
    let is_member = check_conversation_member(db, conversation_id, user_id).await?;
    if !is_member {
        return Err(AppError::Forbidden("您不是该对话的参与者".to_string()));
    }
    Ok(())
}
