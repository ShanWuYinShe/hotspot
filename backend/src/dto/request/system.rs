//! 系统健康检查 + 系统配置 + 公开公告

use serde::Deserialize;

#[derive(Debug, Deserialize)]
pub struct SystemHealthRequest {}

#[derive(Debug, Deserialize)]
pub struct FetchGlobalSettingsRequest {}

/// 公开系统公告列表请求（空结构体，返回条数由服务端 ResponseDefaults 控制）
#[derive(Debug, Deserialize)]
pub struct PublicAnnouncementListRequest {}

/// 系统元信息请求 - POST /api/v1/system/get_meta
#[derive(Debug, Deserialize)]
pub struct SystemGetMetaRequest {}
