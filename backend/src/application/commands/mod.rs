/// 命令模块 - 重新导出所有命令类型
/// 命令特征
pub trait Command {
    /// 命令结果类型
    type Result;
}

// 用户命令
pub mod user_commands;
pub use user_commands::*;

// 群组命令
pub mod group_commands;
pub use group_commands::*;

// 消息命令
pub mod message_commands;
pub use message_commands::*;

// 瞬间命令
pub mod moment_commands;
pub use moment_commands::*;

// 额外的命令类型已合并到各自的模块中
