/// JavaScript安全类型定义和转换工具
/// 
/// JavaScript的Number类型只能安全表示到2^53-1（约9千万亿），
/// 而Rust的i64可以表示到2^63-1。为了确保前后端数据一致性，
/// 我们需要将64位整数序列化为字符串传输给前端。
use serde::{Deserialize, Deserializer, Serialize, Serializer};
use std::fmt;
use std::str::FromStr;

/// JavaScript安全的64位整数类型
/// 
/// 这个类型在序列化时会转换为字符串，反序列化时会从字符串解析为i64
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub struct JsSafeI64(pub i64);

impl JsSafeI64 {
    /// 创建新的JsSafeI64
    pub fn new(value: i64) -> Self {
        Self(value)
    }
    
    /// 获取内部的i64值
    pub fn value(self) -> i64 {
        self.0
    }
    
    /// 从字符串解析
    pub fn from_str(s: &str) -> Result<Self, std::num::ParseIntError> {
        Ok(Self(s.parse()?))
    }
    
    /// 转换为字符串
    pub fn to_string_value(self) -> String {
        self.0.to_string()
    }
}

impl From<i64> for JsSafeI64 {
    fn from(value: i64) -> Self {
        Self(value)
    }
}

impl From<JsSafeI64> for i64 {
    fn from(value: JsSafeI64) -> Self {
        value.0
    }
}

impl fmt::Display for JsSafeI64 {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.0)
    }
}

impl FromStr for JsSafeI64 {
    type Err = std::num::ParseIntError;
    
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        Ok(Self(s.parse()?))
    }
}

// 序列化为字符串
impl Serialize for JsSafeI64 {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_str(&self.0.to_string())
    }
}

// 从字符串或数字反序列化
impl<'de> Deserialize<'de> for JsSafeI64 {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct JsSafeI64Visitor;
        
        impl<'de> serde::de::Visitor<'de> for JsSafeI64Visitor {
            type Value = JsSafeI64;
            
            fn expecting(&self, formatter: &mut fmt::Formatter) -> fmt::Result {
                formatter.write_str("a string or integer representing a 64-bit integer")
            }
            
            fn visit_str<E>(self, value: &str) -> Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                value.parse::<i64>()
                    .map(JsSafeI64)
                    .map_err(serde::de::Error::custom)
            }
            
            fn visit_i64<E>(self, value: i64) -> Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                Ok(JsSafeI64(value))
            }
            
            fn visit_u64<E>(self, value: u64) -> Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                if value <= i64::MAX as u64 {
                    Ok(JsSafeI64(value as i64))
                } else {
                    Err(serde::de::Error::custom("u64 value too large for i64"))
                }
            }
        }
        
        deserializer.deserialize_any(JsSafeI64Visitor)
    }
}

/// JavaScript安全的可选64位整数类型
pub type JsSafeOptionalI64 = Option<JsSafeI64>;

/// 转换工具函数
pub mod convert {
    use super::JsSafeI64;
    
    /// 将i64向量转换为JsSafeI64向量
    pub fn i64_vec_to_js_safe(vec: Vec<i64>) -> Vec<JsSafeI64> {
        vec.into_iter().map(JsSafeI64::from).collect()
    }
    
    /// 将JsSafeI64向量转换为i64向量
    pub fn js_safe_vec_to_i64(vec: Vec<JsSafeI64>) -> Vec<i64> {
        vec.into_iter().map(|x| x.value()).collect()
    }
    
    /// 将Option<i64>转换为Option<JsSafeI64>
    pub fn optional_i64_to_js_safe(opt: Option<i64>) -> Option<JsSafeI64> {
        opt.map(JsSafeI64::from)
    }
    
    /// 将Option<JsSafeI64>转换为Option<i64>
    pub fn optional_js_safe_to_i64(opt: Option<JsSafeI64>) -> Option<i64> {
        opt.map(|x| x.value())
    }
}

/// 用于数据库查询的转换宏
#[macro_export]
macro_rules! js_safe_query {
    ($field:expr) => {
        $crate::utils::js_safe_types::JsSafeI64::from($field)
    };
}

/// 用于从JsSafeI64获取数据库值的转换宏
#[macro_export]
macro_rules! db_value {
    ($js_safe:expr) => {
        $js_safe.value()
    };
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json;
    
    #[test]
    fn test_js_safe_i64_serialization() {
        let value = JsSafeI64::new(1234567890123456789);
        let json = serde_json::to_string(&value).unwrap();
        assert_eq!(json, "\"1234567890123456789\"");
    }
    
    #[test]
    fn test_js_safe_i64_deserialization_from_string() {
        let json = "\"1234567890123456789\"";
        let value: JsSafeI64 = serde_json::from_str(json).unwrap();
        assert_eq!(value.value(), 1234567890123456789);
    }
    
    #[test]
    fn test_js_safe_i64_deserialization_from_number() {
        let json = "123456789";
        let value: JsSafeI64 = serde_json::from_str(json).unwrap();
        assert_eq!(value.value(), 123456789);
    }
    
    #[test]
    fn test_js_safe_i64_conversion() {
        let original: i64 = 9876543210;
        let js_safe = JsSafeI64::from(original);
        let converted: i64 = js_safe.into();
        assert_eq!(original, converted);
    }
    
    #[test]
    fn test_optional_conversion() {
        let original = Some(1234567890i64);
        let js_safe = convert::optional_i64_to_js_safe(original);
        let converted = convert::optional_js_safe_to_i64(js_safe);
        assert_eq!(original, converted);
    }
    
    #[test]
    fn test_vector_conversion() {
        let original = vec![1, 2, 3, 4, 5];
        let js_safe = convert::i64_vec_to_js_safe(original.clone());
        let converted = convert::js_safe_vec_to_i64(js_safe);
        assert_eq!(original, converted);
    }
    
    #[test]
    fn test_large_number_serialization() {
        // 测试超过JavaScript安全整数范围的数字
        let large_number = 9007199254740992i64; // 2^53
        let js_safe = JsSafeI64::new(large_number);
        let json = serde_json::to_string(&js_safe).unwrap();
        assert_eq!(json, "\"9007199254740992\"");
        
        let deserialized: JsSafeI64 = serde_json::from_str(&json).unwrap();
        assert_eq!(deserialized.value(), large_number);
    }
}
