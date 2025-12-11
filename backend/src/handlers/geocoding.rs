//! 地理编码服务 Handler
//!
//! 提供逆地理编码功能，将坐标转换为地址
//! 使用高德地图 Web 服务 API（Key 安全保存在后端）

use axum::{extract::State, Json};
use serde::{Deserialize, Serialize};

use crate::{
    dto::response::ApiResponse,
    error::{AppError, Result},
    state::AppState,
};

/// 逆地理编码请求
#[derive(Debug, Deserialize)]
pub struct ReverseGeocodeRequest {
    pub latitude: f64,
    pub longitude: f64,
}

/// 逆地理编码响应数据
#[derive(Debug, Serialize)]
pub struct ReverseGeocodeData {
    /// 完整地址
    pub address: String,
    /// 简短名称（适合显示）
    pub short_name: String,
    /// 详细地址组件
    pub components: AddressComponents,
}

/// 地址组件
#[derive(Debug, Serialize, Default)]
pub struct AddressComponents {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub country: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub province: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub city: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub district: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub street: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub building: Option<String>,
}

/// 高德地图逆地理编码 API 响应
#[derive(Debug, Deserialize)]
struct AmapResponse {
    status: String,
    #[serde(default)]
    info: String,
    regeocode: Option<AmapRegeocode>,
}

#[derive(Debug, Deserialize)]
struct AmapRegeocode {
    formatted_address: Option<String>,
    #[serde(rename = "addressComponent")]
    address_component: Option<AmapAddressComponent>,
}

#[derive(Debug, Deserialize)]
struct AmapAddressComponent {
    country: Option<String>,
    province: Option<StringOrVec>,
    city: Option<StringOrVec>,
    district: Option<StringOrVec>,
    street: Option<StringOrVec>,
    building: Option<AmapBuilding>,
    neighborhood: Option<AmapNeighborhood>,
}

/// 高德返回的建筑信息
#[derive(Debug, Deserialize)]
struct AmapBuilding {
    name: Option<StringOrVec>,
}

/// 高德返回的社区信息
#[derive(Debug, Deserialize)]
struct AmapNeighborhood {
    name: Option<StringOrVec>,
}

/// 高德 API 有时返回字符串，有时返回数组，需要特殊处理
#[derive(Debug, Deserialize)]
#[serde(untagged)]
enum StringOrVec {
    String(String),
    Vec(Vec<String>),
}

impl StringOrVec {
    fn to_option_string(&self) -> Option<String> {
        match self {
            StringOrVec::String(s) if !s.is_empty() && s != "[]" => Some(s.clone()),
            StringOrVec::Vec(v) if !v.is_empty() => v.first().cloned(),
            _ => None,
        }
    }
}

/// 逆地理编码
/// 
/// POST /api/v1/public/system/reverse_geocode
pub async fn reverse_geocode(
    State(state): State<AppState>,
    Json(req): Json<ReverseGeocodeRequest>,
) -> Result<Json<ApiResponse<ReverseGeocodeData>>> {
    // 验证坐标范围
    if req.latitude < -90.0 || req.latitude > 90.0 {
        return Err(AppError::BadRequest("纬度必须在 -90 到 90 之间".to_string()));
    }
    if req.longitude < -180.0 || req.longitude > 180.0 {
        return Err(AppError::BadRequest("经度必须在 -180 到 180 之间".to_string()));
    }

    // 获取高德 Key
    let amap_key = state.config.amap_key.as_ref().ok_or_else(|| {
        tracing::warn!("高德地图 API Key 未配置，使用坐标作为位置名称");
        AppError::InternalError("地理编码服务未配置".to_string())
    })?;

    // 调用高德逆地理编码 API
    let result = call_amap_reverse_geocode(amap_key, req.latitude, req.longitude).await?;
    
    Ok(Json(ApiResponse::success(result)))
}

/// 调用高德逆地理编码 API
async fn call_amap_reverse_geocode(
    key: &str,
    latitude: f64,
    longitude: f64,
) -> Result<ReverseGeocodeData> {
    // 高德坐标格式：经度,纬度
    let location = format!("{:.6},{:.6}", longitude, latitude);
    
    let url = format!(
        "https://restapi.amap.com/v3/geocode/regeo?key={}&location={}&extensions=base&output=json",
        key, location
    );

    let client = reqwest::Client::builder()
        .timeout(std::time::Duration::from_secs(5))
        .build()
        .map_err(|e| AppError::InternalError(format!("HTTP 客户端创建失败: {}", e)))?;

    let response = client
        .get(&url)
        .send()
        .await
        .map_err(|e| {
            tracing::error!("高德 API 请求失败: {}", e);
            AppError::InternalError("地理编码服务请求失败".to_string())
        })?;

    if !response.status().is_success() {
        tracing::error!("高德 API 返回错误状态: {}", response.status());
        return Err(AppError::InternalError("地理编码服务返回错误".to_string()));
    }

    let amap_response: AmapResponse = response
        .json()
        .await
        .map_err(|e| {
            tracing::error!("高德 API 响应解析失败: {}", e);
            AppError::InternalError("地理编码服务响应解析失败".to_string())
        })?;

    if amap_response.status != "1" {
        tracing::error!("高德 API 业务错误: {}", amap_response.info);
        return Err(AppError::InternalError(format!("地理编码失败: {}", amap_response.info)));
    }

    // 解析响应
    let regeocode = amap_response.regeocode.ok_or_else(|| {
        AppError::InternalError("地理编码结果为空".to_string())
    })?;

    let address = regeocode.formatted_address.unwrap_or_default();
    let components = parse_address_components(regeocode.address_component);
    let short_name = generate_short_name(&components, &address);

    Ok(ReverseGeocodeData {
        address,
        short_name,
        components,
    })
}

/// 解析地址组件
fn parse_address_components(component: Option<AmapAddressComponent>) -> AddressComponents {
    let Some(c) = component else {
        return AddressComponents::default();
    };

    AddressComponents {
        country: c.country,
        province: c.province.and_then(|v| v.to_option_string()),
        city: c.city.and_then(|v| v.to_option_string()),
        district: c.district.and_then(|v| v.to_option_string()),
        street: c.street.and_then(|v| v.to_option_string()),
        building: c.building
            .and_then(|b| b.name)
            .and_then(|v| v.to_option_string())
            .or_else(|| {
                c.neighborhood
                    .and_then(|n| n.name)
                    .and_then(|v| v.to_option_string())
            }),
    }
}

/// 生成简短的位置名称
fn generate_short_name(components: &AddressComponents, full_address: &str) -> String {
    // 优先使用：建筑/社区 > 街道 > 区 > 城市
    if let Some(ref building) = components.building {
        if !building.is_empty() {
            return building.clone();
        }
    }

    // 组合：区 + 街道
    let mut parts = Vec::new();
    if let Some(ref district) = components.district {
        if !district.is_empty() {
            parts.push(district.as_str());
        }
    }
    if let Some(ref street) = components.street {
        if !street.is_empty() {
            parts.push(street.as_str());
        }
    }
    if !parts.is_empty() {
        return parts.join("");
    }

    // 使用城市
    if let Some(ref city) = components.city {
        if !city.is_empty() {
            return city.clone();
        }
    }

    // 使用省份
    if let Some(ref province) = components.province {
        if !province.is_empty() {
            return province.clone();
        }
    }

    // 最后使用完整地址的前 20 个字符
    if !full_address.is_empty() {
        let chars: Vec<char> = full_address.chars().collect();
        let len = chars.len().min(20);
        return chars[..len].iter().collect();
    }

    "未知位置".to_string()
}




