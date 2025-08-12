use uuid::Uuid;
use serde::{Serialize, Deserialize};
use crate::ws::message_types::{RoomType, RoomId};

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct UserId(pub i64);

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct GroupId(pub Uuid);

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct ConversationId(pub Uuid);

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct MessageId(pub Uuid);

impl From<i64> for UserId { fn from(v: i64) -> Self { UserId(v) } }
impl From<UserId> for i64 { fn from(v: UserId) -> Self { v.0 } }

impl From<Uuid> for GroupId { fn from(v: Uuid) -> Self { GroupId(v) } }
impl From<GroupId> for Uuid { fn from(v: GroupId) -> Self { v.0 } }

impl From<Uuid> for ConversationId { fn from(v: Uuid) -> Self { ConversationId(v) } }
impl From<ConversationId> for Uuid { fn from(v: ConversationId) -> Self { v.0 } }

impl From<Uuid> for MessageId { fn from(v: Uuid) -> Self { MessageId(v) } }
impl From<MessageId> for Uuid { fn from(v: MessageId) -> Self { v.0 } }

// Display implementations for logging
impl std::fmt::Display for UserId {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result { write!(f, "{}", self.0) }
}
impl std::fmt::Display for GroupId {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result { write!(f, "{}", self.0) }
}
impl std::fmt::Display for ConversationId {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result { write!(f, "{}", self.0) }
}
impl std::fmt::Display for MessageId {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result { write!(f, "{}", self.0) }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RoomKey {
    pub room_type: RoomType,
    pub room_id: Uuid,
}

impl RoomKey {
    pub fn new(room_type: RoomType, room_id: Uuid) -> Self { Self { room_type, room_id } }
}

impl std::fmt::Display for RoomKey {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self.room_type {
            RoomType::Group => write!(f, "group:{}", self.room_id),
            RoomType::Conversation => write!(f, "conversation:{}", self.room_id),
        }
    }
}

impl From<(RoomType, Uuid)> for RoomKey {
    fn from(value: (RoomType, Uuid)) -> Self { Self { room_type: value.0, room_id: value.1 } }
}

impl From<RoomKey> for RoomId {
    fn from(value: RoomKey) -> Self {
        match value.room_type {
            RoomType::Group => RoomId::Group(value.room_id),
            RoomType::Conversation => RoomId::Conversation(value.room_id),
        }
    }
}


