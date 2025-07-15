use utoipa::OpenApi;

use crate::controllers::{
    checkin_controller,
    group_controller,
    health_controller,
    message_controller,
    user_controller,
};

#[derive(OpenApi)]
#[openapi(
    paths(
        health_controller::health_check,
        user_controller::register,
        user_controller::check_auth_status,
        group_controller::create_group,
        group_controller::join_group,
        group_controller::search_group_by_location,
        group_controller::search_group_by_name,
        group_controller::search_group_by_id,
        group_controller::group_members,
        group_controller::leave_group,
        group_controller::kick_member,
        group_controller::transfer_ownership,
        message_controller::send_group_message,
        message_controller::send_direct_message,
        message_controller::delete_group_message,
        message_controller::mark_direct_messages_read,
        checkin_controller::create_checkin,
        checkin_controller::delete_checkin,
        checkin_controller::search_checkin_by_id,
        checkin_controller::search_checkin_by_location,
        checkin_controller::search_checkin_by_tags,
        checkin_controller::like_checkin,
        checkin_controller::unlike_checkin,
    ),
    tags(
        (name = "用户", description = "用户管理相关API"),
        (name = "群组", description = "群组管理相关API"),
        (name = "消息", description = "消息相关API"),
        (name = "签到", description = "签到相关API"),
    )
)]
pub struct ApiDoc;
