use std::sync::Arc;
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

    /// 批量发布事件
    pub fn publish_batch(&self, events: Vec<AppEvent>) {
        for event in events {
            self.publish(event);
        }
    }
}

impl Default for EventBus {
    fn default() -> Self {
        Self::new().0
    }
}

/// 事件总线管理器
pub struct EventBusManager {
    event_bus: Arc<EventBus>,
}

impl EventBusManager {
    pub fn new(event_bus: EventBus) -> Self {
        Self {
            event_bus: Arc::new(event_bus),
        }
    }

    pub fn get_bus(&self) -> Arc<EventBus> {
        self.event_bus.clone()
    }
}

