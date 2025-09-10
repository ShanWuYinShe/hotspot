/// 位置相关常量
/// 
/// 定义默认位置、搜索半径等地理位置相关的常量
/// 默认纬度 - 南天门 (35.16)
pub const DEFAULT_LATITUDE: f64 = 35.16;

/// 默认经度 - 南天门 (112.68)
pub const DEFAULT_LONGITUDE: f64 = 112.68;

/// 默认位置名称
pub const DEFAULT_LOCATION_NAME: &str = "南天门";

/// 搜索半径相关常量
pub mod radius {
    /// 附近用户搜索默认半径（米）
    pub const DEFAULT_NEARBY_RADIUS_METERS: f64 = 5000.0; // 5公里
    
    /// 附近群组搜索默认半径（米）
    pub const DEFAULT_GROUP_RADIUS_METERS: f64 = 10000.0; // 10公里
    
    /// 最大搜索半径（米）
    pub const MAX_SEARCH_RADIUS_METERS: f64 = 50000.0; // 50公里
    
    /// 附近用户搜索默认半径（公里）
    pub const DEFAULT_NEARBY_RADIUS_KM: f64 = DEFAULT_NEARBY_RADIUS_METERS / 1000.0;
    
    /// 附近群组搜索默认半径（公里）
    pub const DEFAULT_GROUP_RADIUS_KM: f64 = DEFAULT_GROUP_RADIUS_METERS / 1000.0;
    
    /// 最大搜索半径（公里）
    pub const MAX_SEARCH_RADIUS_KM: f64 = MAX_SEARCH_RADIUS_METERS / 1000.0;
}

/// 通知半径相关常量
pub mod notification {
    /// 默认通知半径（公里）
    pub const DEFAULT_NOTIFICATION_RADIUS_KM: f64 = 5.0;
    
    /// 最大通知半径（公里）
    pub const MAX_NOTIFICATION_RADIUS_KM: f64 = 50.0;
    
    /// 最小通知半径（公里）
    pub const MIN_NOTIFICATION_RADIUS_KM: f64 = 0.1;
}
