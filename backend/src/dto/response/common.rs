//! 通用基础类型与 API 响应格式

use serde::Serialize;

// ============================================================
// ID → string 序列化辅助
// ============================================================

/// 将 i64 序列化为 JSON string，反序列化时支持 string 或 number
/// （spec 要求所有 ID 在 API 层为 string 类型）
pub mod id_as_string {
    use serde::{self, Deserialize, Deserializer, Serializer};

    pub fn serialize<S>(id: &i64, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_str(&id.to_string())
    }

    pub fn deserialize<'de, D>(deserializer: D) -> Result<i64, D::Error>
    where
        D: Deserializer<'de>,
    {
        #[derive(Deserialize)]
        #[serde(untagged)]
        enum StringOrNumber {
            Str(String),
            Num(i64),
        }
        match StringOrNumber::deserialize(deserializer)? {
            StringOrNumber::Str(s) => s.parse::<i64>().map_err(serde::de::Error::custom),
            StringOrNumber::Num(n) => Ok(n),
        }
    }
}

/// 将 Option<Vec<i64>> 序列化为 JSON string 数组或 null
pub mod option_vec_id_as_string {
    use serde::{self, Serializer};

    pub fn serialize<S>(ids: &Option<Vec<i64>>, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        match ids {
            Some(ids) => {
                use serde::ser::SerializeSeq;
                let mut seq = serializer.serialize_seq(Some(ids.len()))?;
                for id in ids {
                    seq.serialize_element(&id.to_string())?;
                }
                seq.end()
            }
            None => serializer.serialize_none(),
        }
    }
}

// ============================================================
// 统一 API 响应格式
// ============================================================

/// 统一 API 响应格式 - API 文档: ApiResponse<T>
/// spec: { code, message?, data? }，request_id 通过 X-Request-Id 响应头传递
#[derive(Debug, Serialize)]
pub struct ApiResponse<T> {
    pub code: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data: Option<T>,
}

impl<T> ApiResponse<T> {
    pub fn success(data: T) -> Self {
        Self {
            code: 0,
            message: None,
            data: Some(data),
        }
    }

    pub fn error(code: i32, message: String) -> Self {
        Self {
            code,
            message: Some(message),
            data: None,
        }
    }
}

impl ApiResponse<()> {
    pub fn success_empty() -> Self {
        Self {
            code: 0,
            message: None,
            data: None,
        }
    }
}

// ============================================================
// 通用基础类型 (API 文档定义)
// ============================================================

/// 基础坐标点 (经纬度) - API 文档: GeoPoint
#[derive(Debug, Serialize, serde::Deserialize, Clone, Copy)]
pub struct GeoPoint {
    pub latitude: f64,
    pub longitude: f64,
}

/// 绝对位置 - API 文档: Location
/// 用途：请求/响应中直接回传客户端提供的坐标，无需距离/方位计算
#[derive(Debug, Serialize, serde::Deserialize, Clone)]
pub struct Location {
    pub point: GeoPoint,
    pub location_name: String,
}

/// 相对位置 - API 文档: RelativeLocation
/// 用途：所有 response 响应数据中的位置字段统一使用此类型
/// - 搜索/列表/Feed 场景：distance 和 bearing_degrees 有值（计算相对距离）
/// - 详情/Admin/用户自己的内容：distance 和 bearing_degrees 为 None（不需要距离）
/// 注意：如果 moment 没有位置信息，整个 location 字段应为 Option<RelativeLocation>
#[derive(Debug, Serialize, serde::Deserialize, Clone)]
pub struct RelativeLocation {
    pub point: GeoPoint,
    pub location_name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub distance: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bearing_degrees: Option<f64>,
}

impl RelativeLocation {
    /// 创建带距离和方位的相对位置（用于搜索/列表/Feed）
    pub fn with_distance(
        latitude: f64,
        longitude: f64,
        location_name: impl Into<String>,
        distance: f64,
        bearing_degrees: f64,
    ) -> Self {
        Self {
            point: GeoPoint {
                latitude,
                longitude,
            },
            location_name: location_name.into(),
            distance: Some(distance),
            bearing_degrees: Some(bearing_degrees),
        }
    }

    /// 创建无距离的相对位置（用于详情/自己的内容）
    pub fn without_distance(
        latitude: f64,
        longitude: f64,
        location_name: impl Into<String>,
    ) -> Self {
        Self {
            point: GeoPoint {
                latitude,
                longitude,
            },
            location_name: location_name.into(),
            distance: None,
            bearing_degrees: None,
        }
    }
}

/// 分页数据容器 - API 文档: PaginatedData<T>
/// next_cursor 为 None 时表示没有更多数据
#[derive(Debug, Serialize)]
pub struct PaginatedData<T> {
    pub items: Vec<T>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub next_cursor: Option<String>,
}
