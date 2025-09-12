/// 事件发布器日志适配器
/// 
/// 基于日志的事件发布实现，作为占位与最小可用实现，符合端口接口
use async_trait::async_trait;
use crate::domain::{DomainResult, UserEvent, GroupEvent, MessageEvent, MomentEvent};
use crate::ports::{EventPublisher, PublisherStats, DomainEventEnvelope};

#[derive(Clone, Default)]
pub struct LogEventPublisher;

#[async_trait]
impl EventPublisher for LogEventPublisher {
    async fn publish(&self, event: &crate::domain::DomainEvent) -> DomainResult<()> {
        log::info!("publish_event: {:?}", event);
        Ok(())
    }
    
    async fn publish_batch(&self, events: &[crate::domain::DomainEvent]) -> DomainResult<()> {
        for event in events {
            log::info!("publish_batch_event: {:?}", event);
        }
        Ok(())
    }
    
    async fn publish_and_wait(&self, event: &crate::domain::DomainEvent) -> DomainResult<()> {
        log::info!("publish_and_wait_event: {:?}", event);
        Ok(())
    }
}

impl LogEventPublisher {
    #[allow(dead_code)]
    async fn publish_user_event(&self, event: UserEvent) -> DomainResult<()> {
        log::info!("publish_user_event: {:?}", event);
        Ok(())
    }

    #[allow(dead_code)]
    async fn publish_group_event(&self, event: GroupEvent) -> DomainResult<()> {
        log::info!("publish_group_event: {:?}", event);
        Ok(())
    }

    #[allow(dead_code)]
    async fn publish_message_event(&self, event: MessageEvent) -> DomainResult<()> {
        log::info!("publish_message_event: {:?}", event);
        Ok(())
    }

    #[allow(dead_code)]
    async fn publish_moment_event(&self, event: MomentEvent) -> DomainResult<()> {
        log::info!("publish_moment_event: {:?}", event);
        Ok(())
    }

    #[allow(dead_code)]
    async fn publish_events(&self, events: Vec<DomainEventEnvelope>) -> DomainResult<()> {
        for e in events {
            log::info!("publish_event: type={} aggregate_id={}", e.event_type, e.aggregate_id);
        }
        Ok(())
    }

    #[allow(dead_code)]
    async fn get_publisher_stats(&self) -> DomainResult<PublisherStats> {
        Ok(        PublisherStats {
            published_count: 0,
            failed_count: 0,
            avg_publish_time_ms: 0.0,
        })
    }
}


