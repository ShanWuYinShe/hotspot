/// CQRS 命令模块
/// 
/// 定义系统中所有的写操作命令和命令处理器

pub mod user_commands;
pub mod group_commands;
pub mod message_commands;
pub mod moment_commands;

// 重新导出命令类型
pub use user_commands::*;
pub use group_commands::*;
pub use message_commands::*;
pub use moment_commands::*;

use crate::domain::DomainResult;
use async_trait::async_trait;

/// 命令处理器基础trait
#[async_trait]
pub trait CommandHandler<TCommand> {
    type Result;
    
    /// 处理命令
    async fn handle_command(&self, command: TCommand) -> DomainResult<Self::Result>;
}
