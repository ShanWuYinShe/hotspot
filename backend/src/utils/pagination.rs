use aes_gcm::{
    aead::{Aead, KeyInit},
    Aes256Gcm, Nonce,
};
use base64::{engine::general_purpose, Engine as _};
use once_cell::sync::Lazy;
use rand::prelude::*;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::env;

use crate::error::{ApiError, Result};

/// 分页返回类型：(数据列表, 下一页游标)
pub type Paginated<T> = crate::error::Result<(Vec<T>, Option<String>)>;

// 使用 JWT_SECRET 派生加密密钥
// 注意：这里使用 Lazy 确保只初始化一次
// 如果初始化失败，所有使用 CIPHER 的地方都会返回错误
static CIPHER: Lazy<Aes256Gcm> = Lazy::new(|| {
    let secret = env::var("JWT_SECRET").expect("JWT_SECRET must be set for cursor encryption");
    let mut hasher = Sha256::new();
    hasher.update(secret.as_bytes());
    let key = hasher.finalize();
    Aes256Gcm::new_from_slice(&key).expect("SHA-256 always produces valid AES-256 key length")
});

/// 分页游标工具
///
/// 用于生成和解析不透明的游标字符串 (Opaque Cursor)。
/// 游标通常是 (排序值, 唯一ID) 元组的编码字符串。
/// 为了保证不透明性，游标内容经过 AES-256-GCM 加密。
/// 解码游标字符串为指定类型
pub fn decode_cursor<T>(cursor: &str) -> Result<T>
where
    T: for<'de> Deserialize<'de>,
{
    // 1. Base64 解码
    let encrypted_bytes = general_purpose::URL_SAFE_NO_PAD
        .decode(cursor)
        .map_err(|_| ApiError::CursorFormatError("无效的游标格式".to_string()))?;

    // 2. 提取 Nonce (前12字节) 和 密文
    if encrypted_bytes.len() < 12 {
        return Err(ApiError::CursorFormatError("无效的游标数据".to_string()));
    }
    let (nonce_bytes, ciphertext) = encrypted_bytes.split_at(12);
    let nonce = Nonce::from_slice(nonce_bytes);

    // 3. 解密
    let plaintext = CIPHER
        .decrypt(nonce, ciphertext)
        .map_err(|_| ApiError::CursorInvalidOrExpired("游标解密失败".to_string()))?;

    // 4. JSON 反序列化
    let value = serde_json::from_slice(&plaintext)
        .map_err(|_| ApiError::CursorInvalidOrExpired("无效的游标内容".to_string()))?;

    Ok(value)
}

/// 解码游标，如为空则返回默认值
pub fn decode_cursor_or<T, F>(cursor: Option<String>, default: F) -> Result<T>
where
    T: for<'de> Deserialize<'de>,
    F: FnOnce() -> T,
{
    match cursor {
        Some(value) => decode_cursor(&value),
        None => Ok(default()),
    }
}
/// 编码值生成游标字符串
pub fn encode_cursor<T>(value: &T) -> Result<String>
where
    T: Serialize,
{
    // 1. JSON 序列化
    let plaintext = serde_json::to_vec(value)?;

    // 2. 生成随机 Nonce (12字节)
    let mut nonce_bytes = [0u8; 12];
    rand::rng().fill_bytes(&mut nonce_bytes);
    let nonce = Nonce::from_slice(&nonce_bytes);

    // 3. 加密
    let ciphertext = CIPHER
        .encrypt(nonce, plaintext.as_ref())
        .map_err(|_| ApiError::InternalError("Cursor encryption failed".to_string()))?;

    // 4. 拼接 Nonce + 密文
    let mut encrypted_bytes = nonce_bytes.to_vec();
    encrypted_bytes.extend_from_slice(&ciphertext);

    // 5. Base64 编码
    Ok(general_purpose::URL_SAFE_NO_PAD.encode(encrypted_bytes))
}

/// 通用游标类型定义
///
/// 针对常见的排序场景提供标准元组结构
/// 默认排序游标 (ID)
/// 适用于: ORDER BY id DESC
pub type IdCursor = i64;

/// 时间+ID排序游标 (Time, i64 ID)
/// 适用于: ORDER BY created_at DESC, id DESC
pub type TimeCursor = (chrono::DateTime<chrono::Utc>, i64);

/// 时间+UUID排序游标 (Time, Uuid ID)
/// 适用于: ORDER BY created_at DESC, uuid DESC
pub type TimeIdCursor = (chrono::DateTime<chrono::Utc>, uuid::Uuid);

/// 距离排序游标 (Distance, ID)
/// 适用于: ORDER BY distance ASC, id ASC
pub type DistanceCursor = (f64, i64);

/// 距离排序游标 (Distance, UUID ID)
/// 适用于: ORDER BY distance ASC, uuid ASC
pub type DistanceUuidCursor = (f64, uuid::Uuid);

/// 通用分页处理函数（使用 limit+1 模式）
///
/// # 参数
/// - `rows`: 查询结果（应该使用 `limit + 1` 查询）
/// - `limit`: 实际需要返回的条数
/// - `get_cursor_value`: 从项目中提取游标值的函数
///
/// # 返回
/// - `items`: 截取后的实际数据（最多 `limit` 条）
/// - `next_cursor`: 如果有更多数据则返回 `Some(cursor)`，否则返回 `None`
///
/// # 示例
/// ```rust
/// let rows = query.fetch_all().await?; // 查询了 limit + 1 条
/// let (items, next_cursor) = paginate_with_limit_plus_one(
///     rows,
///     limit,
///     |item| encode_cursor(&item.id)
/// )?;
/// ```
pub fn paginate_with_limit_plus_one<T, F>(
    rows: Vec<T>,
    limit: i32,
    get_cursor_value: F,
) -> Paginated<T>
where
    F: FnOnce(&T) -> Result<String>,
{
    let has_more = rows.len() > limit as usize;
    let items: Vec<T> = rows.into_iter().take(limit as usize).collect();

    let next_cursor = if has_more {
        items
            .last()
            .map(|item| get_cursor_value(item))
            .transpose()?
    } else {
        None
    };

    Ok((items, next_cursor))
}
