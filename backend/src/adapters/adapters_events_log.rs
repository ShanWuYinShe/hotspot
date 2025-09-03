/// 事件发布器日志适配器
/// 
/// 基于日志的事件发布实现，作为占位与最小可用实现，符合端口接口

use async_trait::async_trait;
use crate::domain::{DomainResult, UserEvent, GroupEvent, MessageEvent, MomentEvent};
use crate::ports::{EventPublisherPort, PublisherStats, DomainEventEnvelope};

#[derive(Clone, Default)]
pub struct LogEventPublisher;

#[async_trait]
impl EventPublisherPort for LogEventPublisher {
    async fn publish_user_event(&self, event: UserEvent) -> DomainResult<()> {
        log::info!("publish_user_event: {:?}", event);
        Ok(())
    }

    async fn publish_group_event(&self, event: GroupEvent) -> DomainResult<()> {
        log::info!("publish_group_event: {:?}", event);
        Ok(())
    }

    async fn publish_message_event(&self, event: MessageEvent) -> DomainResult<()> {
        log::info!("publish_message_event: {:?}", event);
        Ok(())
    }

    async fn publish_moment_event(&self, event: MomentEvent) -> DomainResult<()> {
        log::info!("publish_moment_event: {:?}", event);
        Ok(())
    }

    async fn publish_events(&self, events: Vec<DomainEventEnvelope>) -> DomainResult<()> {
        for e in events {
            log::info!("publish_event: type={} aggregate_id={}", e.event_type(), e.aggregate_id());
        }
        Ok(())
    }

    async fn get_publisher_stats(&self) -> DomainResult<PublisherStats> {
        Ok(PublisherStats {
            total_published: 0,
            failed_published: 0,
            by_type: Default::default(),
            avg_publish_latency_ms: 0.0,
            last_published_at: None,
        })
    }
}


