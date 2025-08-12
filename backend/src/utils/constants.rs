//! 通用常量

/// 默认纬度 - 南天门 (35.16)
pub const DEFAULT_LATITUDE: f64 = 35.16;

/// 默认经度 - 南天门 (112.68)
pub const DEFAULT_LONGITUDE: f64 = 112.68;

/// 默认位置名称
pub const DEFAULT_LOCATION_NAME: &str = "南天门";

/// 应用默认值常量
pub mod defaults {
    /// 默认用户昵称
    pub const DEFAULT_USER_NICKNAME: &str = "用户";
    pub const DEFAULT_TEMP_USER_NICKNAME: &str = "临时用户";

    /// 默认分页设置
    pub const DEFAULT_PAGE_SIZE: i64 = 10;
    #[allow(dead_code)]
    pub const DEFAULT_LARGE_PAGE_SIZE: i64 = 20;
    pub const MAX_PAGE_SIZE: i64 = 50;
    #[allow(dead_code)]
    pub const MAX_LARGE_PAGE_SIZE: i64 = 100;

    /// 搜索限制
    pub const MAX_SEARCH_RADIUS_METERS: f64 = 10000.0; // 10公里
    pub const MAX_SEARCH_KEYWORD_LENGTH: usize = 50;

    /// 用户设置默认值
    #[allow(dead_code)]
    pub const DEFAULT_QUIET_HOURS_START: &str = "22:00";
    #[allow(dead_code)]
    pub const DEFAULT_QUIET_HOURS_END: &str = "08:00";
    #[allow(dead_code)]
    pub const DEFAULT_NOTIFICATION_RADIUS_KM: f64 = 5.0;
}

pub mod validation {
    /// 验证限制常量
    pub const MIN_PASSWORD_LENGTH: usize = 6;
    pub const MAX_PASSWORD_LENGTH: usize = 20;
    pub const MAX_LOGIN_ID_LENGTH: usize = 50;
    pub const MAX_NICKNAME_LENGTH: usize = 50;
    #[allow(dead_code)]
    pub const MAX_LOCATION_NAME_LENGTH: usize = 100;
}

pub mod messages {
    /// 系统消息模板
    pub const WELCOME_TEMP_USER_MESSAGE: &str = "欢迎临时用户 {} 使用 GeoTalk！";
}