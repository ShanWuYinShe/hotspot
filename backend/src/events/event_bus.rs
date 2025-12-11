use tokio::sync::mpsc;
use tracing::{error, info};

use crate::events::events::AppEvent;

/// 事件总线
#[derive(Clone)]
pub struct EventBus {
    sender: mpsc::UnboundedSender<AppEvent>,
}

impl EventBus {
    /// 创建新的事件总线
    pub fn new() -> (Self, mpsc::UnboundedReceiver<AppEvent>) {
        let (sender, receiver) = mpsc::unbounded_channel();
        (Self { sender }, receiver)
    }

    /// 发布事件
    pub fn publish(&self, event: AppEvent) {
        if let Err(e) = self.sender.send(event.clone()) {
            error!("Failed to publish event: {:?}, error: {}", event, e);
        } else {
            info!("Event published: {}", event.event_type());
        }
    }
}

impl Default for EventBus {
    fn default() -> Self {
        Self::new().0
    }
}

