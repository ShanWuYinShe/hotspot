/// 领域层模块
/// 
/// 包含纯粹的业务领域逻辑，不依赖任何外部框架或基础设施

pub mod entities;
pub mod values;
pub mod events;
pub mod errors;
pub mod constants;

// 重新导出核心领域类型
pub use entities::*;
pub use values::*;
pub use events::*;
pub use errors::*;
// 避免模糊重导出，明确导入常量
pub use constants::{
    location_constants::{DEFAULT_LATITUDE, DEFAULT_LONGITUDE, DEFAULT_LOCATION_NAME},
    group_roles::*,
};

