/// 配置管理 - 统一的配置加载和验证
/// 
/// 配置特征：
/// - 环境变量支持
/// - 类型安全
/// - 验证和默认值
/// - 热重载支持

use serde::{Deserialize, Serialize};
use crate::infrastructure::errors::{InfrastructureError, ErrorContext, ContextualError};

/// 应用配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AppConfig {
    pub server: ServerConfig,
    pub database: DatabaseConfig,
    pub cache: CacheConfig,
    pub auth: AuthConfig,
    pub external_services: ExternalServicesConfig,
    pub logging: LoggingConfig,
    pub features: FeatureConfig,
}

/// 服务器配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServerConfig {
    pub host: String,
    pub port: u16,
    pub workers: Option<usize>,
    pub max_connections: u32,
    pub timeout_seconds: u64,
    pub cors_origins: Vec<String>,
}

/// 数据库配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DatabaseConfig {
    pub url: String,
    pub max_connections: u32,
    pub min_connections: u32,
    pub connection_timeout_seconds: u64,
    pub idle_timeout_seconds: u64,
    pub max_lifetime_seconds: u64,
    pub auto_migrate: bool,
}

/// 缓存配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CacheConfig {
    pub redis_url: Option<String>,
    pub memory_cache_size: usize,
    pub default_ttl_seconds: u64,
    pub max_key_size: usize,
    pub max_value_size: usize,
}

/// 认证配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthConfig {
    pub jwt_secret: String,
    pub jwt_expiry_seconds: u64,
    pub refresh_token_expiry_seconds: u64,
    pub password_min_length: usize,
    pub password_max_length: usize,
    pub session_timeout_seconds: u64,
}

/// 外部服务配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExternalServicesConfig {
    pub email: EmailServiceConfig,
    pub sms: SmsServiceConfig,
    pub push: PushServiceConfig,
    pub geolocation: GeolocationServiceConfig,
    pub file_storage: FileStorageConfig,
}

/// 邮件服务配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmailServiceConfig {
    pub enabled: bool,
    pub smtp_host: Option<String>,
    pub smtp_port: Option<u16>,
    pub smtp_username: Option<String>,
    pub smtp_password: Option<String>,
    pub from_address: Option<String>,
    pub from_name: Option<String>,
}

/// 短信服务配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SmsServiceConfig {
    pub enabled: bool,
    pub provider: Option<String>,
    pub api_key: Option<String>,
    pub api_secret: Option<String>,
}

/// 推送服务配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PushServiceConfig {
    pub enabled: bool,
    pub fcm_server_key: Option<String>,
    pub apns_key_id: Option<String>,
    pub apns_team_id: Option<String>,
    pub apns_bundle_id: Option<String>,
}

/// 地理位置服务配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GeolocationServiceConfig {
    pub enabled: bool,
    pub provider: Option<String>,
    pub api_key: Option<String>,
}

/// 文件存储配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FileStorageConfig {
    pub provider: String, // local, s3, etc.
    pub local_path: Option<String>,
    pub s3_bucket: Option<String>,
    pub s3_region: Option<String>,
    pub s3_access_key: Option<String>,
    pub s3_secret_key: Option<String>,
    pub max_file_size: u64,
    pub allowed_extensions: Vec<String>,
}

/// 日志配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LoggingConfig {
    pub level: String,
    pub format: String, // json, pretty
    pub output: String, // stdout, file
    pub file_path: Option<String>,
    pub max_file_size: Option<u64>,
    pub max_files: Option<u32>,
    pub structured: bool,
}

/// 功能开关配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeatureConfig {
    pub websocket_enabled: bool,
    pub job_queue_enabled: bool,
    pub cache_enabled: bool,
    pub metrics_enabled: bool,
    pub tracing_enabled: bool,
    pub rate_limiting_enabled: bool,
    pub maintenance_mode: bool,
}

/// 配置加载器
pub struct ConfigLoader;

impl ConfigLoader {
    /// 从环境变量加载配置
    pub fn load_from_env() -> Result<AppConfig, ContextualError> {
        let context = ErrorContext::new("load_config_from_env");
        
        let config = AppConfig {
            server: Self::load_server_config()?,
            database: Self::load_database_config()?,
            cache: Self::load_cache_config()?,
            auth: Self::load_auth_config()?,
            external_services: Self::load_external_services_config()?,
            logging: Self::load_logging_config()?,
            features: Self::load_feature_config()?,
        };
        
        // 验证配置
        Self::validate_config(&config)?;
        
        Ok(config)
    }
    
    /// 从文件加载配置
    pub fn load_from_file(path: &str) -> Result<AppConfig, ContextualError> {
        let context = ErrorContext::new("load_config_from_file")
            .with_metadata("file_path", path);
        
        let content = std::fs::read_to_string(path)
            .map_err(|e| {
                let infra_error = crate::infrastructure::errors::ErrorConverter::from_io_error(e, Some(path));
                ContextualError::new(infra_error, context.clone())
            })?;
        
        let config: AppConfig = if path.ends_with(".toml") {
            toml::from_str(&content)
        } else if path.ends_with(".yaml") || path.ends_with(".yml") {
            serde_yaml::from_str(&content)
        } else {
            serde_json::from_str(&content)
        }.map_err(|e| {
            let infra_error = InfrastructureError::Configuration {
                key: "config_file".to_string(),
                message: e.to_string(),
            };
            ContextualError::new(infra_error, context.clone())
        })?;
        
        Self::validate_config(&config)?;
        
        Ok(config)
    }
    
    /// 加载服务器配置
    fn load_server_config() -> Result<ServerConfig, ContextualError> {
        Ok(ServerConfig {
            host: std::env::var("SERVER_HOST").unwrap_or_else(|_| "0.0.0.0".to_string()),
            port: std::env::var("SERVER_PORT")
                .unwrap_or_else(|_| "8080".to_string())
                .parse()
                .map_err(|e| Self::env_parse_error("SERVER_PORT", &e.to_string()))?,
            workers: std::env::var("SERVER_WORKERS")
                .ok()
                .map(|v| v.parse().map_err(|e| Self::env_parse_error("SERVER_WORKERS", &e.to_string())))
                .transpose()?,
            max_connections: std::env::var("SERVER_MAX_CONNECTIONS")
                .unwrap_or_else(|_| "1000".to_string())
                .parse()
                .map_err(|e| Self::env_parse_error("SERVER_MAX_CONNECTIONS", &e.to_string()))?,
            timeout_seconds: std::env::var("SERVER_TIMEOUT_SECONDS")
                .unwrap_or_else(|_| "30".to_string())
                .parse()
                .map_err(|e| Self::env_parse_error("SERVER_TIMEOUT_SECONDS", &e.to_string()))?,
            cors_origins: std::env::var("CORS_ORIGINS")
                .unwrap_or_else(|_| "*".to_string())
                .split(',')
                .map(|s| s.trim().to_string())
                .collect(),
        })
    }
    
    /// 加载数据库配置
    fn load_database_config() -> Result<DatabaseConfig, ContextualError> {
        let url = std::env::var("DATABASE_URL")
            .map_err(|_| Self::env_missing_error("DATABASE_URL"))?;
        
        Ok(DatabaseConfig {
            url,
            max_connections: std::env::var("DATABASE_MAX_CONNECTIONS")
                .unwrap_or_else(|_| "20".to_string())
                .parse()
                .map_err(|e| Self::env_parse_error("DATABASE_MAX_CONNECTIONS", &e.to_string()))?,
            min_connections: std::env::var("DATABASE_MIN_CONNECTIONS")
                .unwrap_or_else(|_| "5".to_string())
                .parse()
                .map_err(|e| Self::env_parse_error("DATABASE_MIN_CONNECTIONS", &e.to_string()))?,
            connection_timeout_seconds: std::env::var("DATABASE_CONNECTION_TIMEOUT")
                .unwrap_or_else(|_| "30".to_string())
                .parse()
                .map_err(|e| Self::env_parse_error("DATABASE_CONNECTION_TIMEOUT", &e.to_string()))?,
            idle_timeout_seconds: std::env::var("DATABASE_IDLE_TIMEOUT")
                .unwrap_or_else(|_| "600".to_string())
                .parse()
                .map_err(|e| Self::env_parse_error("DATABASE_IDLE_TIMEOUT", &e.to_string()))?,
            max_lifetime_seconds: std::env::var("DATABASE_MAX_LIFETIME")
                .unwrap_or_else(|_| "3600".to_string())
                .parse()
                .map_err(|e| Self::env_parse_error("DATABASE_MAX_LIFETIME", &e.to_string()))?,
            auto_migrate: std::env::var("DATABASE_AUTO_MIGRATE")
                .unwrap_or_else(|_| "false".to_string())
                .parse()
                .map_err(|e| Self::env_parse_error("DATABASE_AUTO_MIGRATE", &e.to_string()))?,
        })
    }
    
    /// 加载缓存配置
    fn load_cache_config() -> Result<CacheConfig, ContextualError> {
        Ok(CacheConfig {
            redis_url: std::env::var("REDIS_URL").ok(),
            memory_cache_size: std::env::var("MEMORY_CACHE_SIZE")
                .unwrap_or_else(|_| "100".to_string())
                .parse::<usize>()
                .map_err(|e| Self::env_parse_error("MEMORY_CACHE_SIZE", &e.to_string()))?
                * 1024 * 1024, // MB to bytes
            default_ttl_seconds: std::env::var("CACHE_DEFAULT_TTL")
                .unwrap_or_else(|_| "300".to_string())
                .parse()
                .map_err(|e| Self::env_parse_error("CACHE_DEFAULT_TTL", &e.to_string()))?,
            max_key_size: std::env::var("CACHE_MAX_KEY_SIZE")
                .unwrap_or_else(|_| "1024".to_string())
                .parse()
                .map_err(|e| Self::env_parse_error("CACHE_MAX_KEY_SIZE", &e.to_string()))?,
            max_value_size: std::env::var("CACHE_MAX_VALUE_SIZE")
                .unwrap_or_else(|_| "1048576".to_string()) // 1MB
                .parse()
                .map_err(|e| Self::env_parse_error("CACHE_MAX_VALUE_SIZE", &e.to_string()))?,
        })
    }
    
    /// 加载认证配置
    fn load_auth_config() -> Result<AuthConfig, ContextualError> {
        let jwt_secret = std::env::var("JWT_SECRET")
            .map_err(|_| Self::env_missing_error("JWT_SECRET"))?;
        
        Ok(AuthConfig {
            jwt_secret,
            jwt_expiry_seconds: std::env::var("JWT_EXPIRY_SECONDS")
                .unwrap_or_else(|_| "3600".to_string()) // 1 hour
                .parse()
                .map_err(|e| Self::env_parse_error("JWT_EXPIRY_SECONDS", &e.to_string()))?,
            refresh_token_expiry_seconds: std::env::var("REFRESH_TOKEN_EXPIRY_SECONDS")
                .unwrap_or_else(|_| "2592000".to_string()) // 30 days
                .parse()
                .map_err(|e| Self::env_parse_error("REFRESH_TOKEN_EXPIRY_SECONDS", &e.to_string()))?,
            password_min_length: std::env::var("PASSWORD_MIN_LENGTH")
                .unwrap_or_else(|_| "8".to_string())
                .parse()
                .map_err(|e| Self::env_parse_error("PASSWORD_MIN_LENGTH", &e.to_string()))?,
            password_max_length: std::env::var("PASSWORD_MAX_LENGTH")
                .unwrap_or_else(|_| "128".to_string())
                .parse()
                .map_err(|e| Self::env_parse_error("PASSWORD_MAX_LENGTH", &e.to_string()))?,
            session_timeout_seconds: std::env::var("SESSION_TIMEOUT_SECONDS")
                .unwrap_or_else(|_| "86400".to_string()) // 24 hours
                .parse()
                .map_err(|e| Self::env_parse_error("SESSION_TIMEOUT_SECONDS", &e.to_string()))?,
        })
    }
    
    /// 加载外部服务配置
    fn load_external_services_config() -> Result<ExternalServicesConfig, ContextualError> {
        Ok(ExternalServicesConfig {
            email: EmailServiceConfig {
                enabled: std::env::var("EMAIL_ENABLED")
                    .unwrap_or_else(|_| "false".to_string())
                    .parse()
                    .unwrap_or(false),
                smtp_host: std::env::var("SMTP_HOST").ok(),
                smtp_port: std::env::var("SMTP_PORT").ok().and_then(|v| v.parse().ok()),
                smtp_username: std::env::var("SMTP_USERNAME").ok(),
                smtp_password: std::env::var("SMTP_PASSWORD").ok(),
                from_address: std::env::var("EMAIL_FROM_ADDRESS").ok(),
                from_name: std::env::var("EMAIL_FROM_NAME").ok(),
            },
            sms: SmsServiceConfig {
                enabled: std::env::var("SMS_ENABLED")
                    .unwrap_or_else(|_| "false".to_string())
                    .parse()
                    .unwrap_or(false),
                provider: std::env::var("SMS_PROVIDER").ok(),
                api_key: std::env::var("SMS_API_KEY").ok(),
                api_secret: std::env::var("SMS_API_SECRET").ok(),
            },
            push: PushServiceConfig {
                enabled: std::env::var("PUSH_ENABLED")
                    .unwrap_or_else(|_| "false".to_string())
                    .parse()
                    .unwrap_or(false),
                fcm_server_key: std::env::var("FCM_SERVER_KEY").ok(),
                apns_key_id: std::env::var("APNS_KEY_ID").ok(),
                apns_team_id: std::env::var("APNS_TEAM_ID").ok(),
                apns_bundle_id: std::env::var("APNS_BUNDLE_ID").ok(),
            },
            geolocation: GeolocationServiceConfig {
                enabled: std::env::var("GEOLOCATION_ENABLED")
                    .unwrap_or_else(|_| "false".to_string())
                    .parse()
                    .unwrap_or(false),
                provider: std::env::var("GEOLOCATION_PROVIDER").ok(),
                api_key: std::env::var("GEOLOCATION_API_KEY").ok(),
            },
            file_storage: FileStorageConfig {
                provider: std::env::var("FILE_STORAGE_PROVIDER")
                    .unwrap_or_else(|_| "local".to_string()),
                local_path: std::env::var("FILE_STORAGE_LOCAL_PATH").ok(),
                s3_bucket: std::env::var("S3_BUCKET").ok(),
                s3_region: std::env::var("S3_REGION").ok(),
                s3_access_key: std::env::var("S3_ACCESS_KEY").ok(),
                s3_secret_key: std::env::var("S3_SECRET_KEY").ok(),
                max_file_size: std::env::var("MAX_FILE_SIZE")
                    .unwrap_or_else(|_| "10485760".to_string()) // 10MB
                    .parse()
                    .unwrap_or(10485760),
                allowed_extensions: std::env::var("ALLOWED_FILE_EXTENSIONS")
                    .unwrap_or_else(|_| "jpg,jpeg,png,gif,pdf,doc,docx".to_string())
                    .split(',')
                    .map(|s| s.trim().to_string())
                    .collect(),
            },
        })
    }
    
    /// 加载日志配置
    fn load_logging_config() -> Result<LoggingConfig, ContextualError> {
        Ok(LoggingConfig {
            level: std::env::var("LOG_LEVEL").unwrap_or_else(|_| "info".to_string()),
            format: std::env::var("LOG_FORMAT").unwrap_or_else(|_| "json".to_string()),
            output: std::env::var("LOG_OUTPUT").unwrap_or_else(|_| "stdout".to_string()),
            file_path: std::env::var("LOG_FILE_PATH").ok(),
            max_file_size: std::env::var("LOG_MAX_FILE_SIZE").ok().and_then(|v| v.parse().ok()),
            max_files: std::env::var("LOG_MAX_FILES").ok().and_then(|v| v.parse().ok()),
            structured: std::env::var("LOG_STRUCTURED")
                .unwrap_or_else(|_| "true".to_string())
                .parse()
                .unwrap_or(true),
        })
    }
    
    /// 加载功能开关配置
    fn load_feature_config() -> Result<FeatureConfig, ContextualError> {
        Ok(FeatureConfig {
            websocket_enabled: std::env::var("WEBSOCKET_ENABLED")
                .unwrap_or_else(|_| "true".to_string())
                .parse()
                .unwrap_or(true),
            job_queue_enabled: std::env::var("JOB_QUEUE_ENABLED")
                .unwrap_or_else(|_| "true".to_string())
                .parse()
                .unwrap_or(true),
            cache_enabled: std::env::var("CACHE_ENABLED")
                .unwrap_or_else(|_| "true".to_string())
                .parse()
                .unwrap_or(true),
            metrics_enabled: std::env::var("METRICS_ENABLED")
                .unwrap_or_else(|_| "true".to_string())
                .parse()
                .unwrap_or(true),
            tracing_enabled: std::env::var("TRACING_ENABLED")
                .unwrap_or_else(|_| "true".to_string())
                .parse()
                .unwrap_or(true),
            rate_limiting_enabled: std::env::var("RATE_LIMITING_ENABLED")
                .unwrap_or_else(|_| "true".to_string())
                .parse()
                .unwrap_or(true),
            maintenance_mode: std::env::var("MAINTENANCE_MODE")
                .unwrap_or_else(|_| "false".to_string())
                .parse()
                .unwrap_or(false),
        })
    }
    
    /// 验证配置
    fn validate_config(config: &AppConfig) -> Result<(), ContextualError> {
        let context = ErrorContext::new("validate_config");
        
        // 验证端口范围
        if config.server.port == 0 || config.server.port > 65535 {
            return Err(ContextualError::new(
                InfrastructureError::Configuration {
                    key: "server.port".to_string(),
                    message: "port must be between 1 and 65535".to_string(),
                },
                context,
            ));
        }
        
        // 验证数据库URL
        if config.database.url.is_empty() {
            return Err(ContextualError::new(
                InfrastructureError::Configuration {
                    key: "database.url".to_string(),
                    message: "database URL cannot be empty".to_string(),
                },
                context,
            ));
        }
        
        // 验证JWT密钥长度
        if config.auth.jwt_secret.len() < 32 {
            return Err(ContextualError::new(
                InfrastructureError::Configuration {
                    key: "auth.jwt_secret".to_string(),
                    message: "JWT secret must be at least 32 characters".to_string(),
                },
                context,
            ));
        }
        
        // 验证密码长度配置
        if config.auth.password_min_length > config.auth.password_max_length {
            return Err(ContextualError::new(
                InfrastructureError::Configuration {
                    key: "auth.password_length".to_string(),
                    message: "min_length cannot be greater than max_length".to_string(),
                },
                context,
            ));
        }
        
        Ok(())
    }
    
    /// 创建环境变量缺失错误
    fn env_missing_error(var: &str) -> ContextualError {
        let context = ErrorContext::new("load_env_var")
            .with_metadata("variable", var);
        let infra_error = InfrastructureError::Environment {
            var: var.to_string(),
            message: "required environment variable is missing".to_string(),
        };
        ContextualError::new(infra_error, context)
    }
    
    /// 创建环境变量解析错误
    fn env_parse_error(var: &str, message: &str) -> ContextualError {
        let context = ErrorContext::new("parse_env_var")
            .with_metadata("variable", var);
        let infra_error = InfrastructureError::Environment {
            var: var.to_string(),
            message: message.to_string(),
        };
        ContextualError::new(infra_error, context)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::env;

    #[test]
    fn test_config_validation() {
        let mut config = AppConfig {
            server: ServerConfig {
                host: "localhost".to_string(),
                port: 8080,
                workers: None,
                max_connections: 1000,
                timeout_seconds: 30,
                cors_origins: vec!["*".to_string()],
            },
            database: DatabaseConfig {
                url: "postgresql://localhost/test".to_string(),
                max_connections: 20,
                min_connections: 5,
                connection_timeout_seconds: 30,
                idle_timeout_seconds: 600,
                max_lifetime_seconds: 3600,
                auto_migrate: false,
            },
            cache: CacheConfig {
                redis_url: None,
                memory_cache_size: 104857600, // 100MB
                default_ttl_seconds: 300,
                max_key_size: 1024,
                max_value_size: 1048576,
            },
            auth: AuthConfig {
                jwt_secret: "a".repeat(32), // 32 characters
                jwt_expiry_seconds: 3600,
                refresh_token_expiry_seconds: 2592000,
                password_min_length: 8,
                password_max_length: 128,
                session_timeout_seconds: 86400,
            },
            external_services: ExternalServicesConfig {
                email: EmailServiceConfig {
                    enabled: false,
                    smtp_host: None,
                    smtp_port: None,
                    smtp_username: None,
                    smtp_password: None,
                    from_address: None,
                    from_name: None,
                },
                sms: SmsServiceConfig {
                    enabled: false,
                    provider: None,
                    api_key: None,
                    api_secret: None,
                },
                push: PushServiceConfig {
                    enabled: false,
                    fcm_server_key: None,
                    apns_key_id: None,
                    apns_team_id: None,
                    apns_bundle_id: None,
                },
                geolocation: GeolocationServiceConfig {
                    enabled: false,
                    provider: None,
                    api_key: None,
                },
                file_storage: FileStorageConfig {
                    provider: "local".to_string(),
                    local_path: Some("/tmp".to_string()),
                    s3_bucket: None,
                    s3_region: None,
                    s3_access_key: None,
                    s3_secret_key: None,
                    max_file_size: 10485760,
                    allowed_extensions: vec!["jpg".to_string(), "png".to_string()],
                },
            },
            logging: LoggingConfig {
                level: "info".to_string(),
                format: "json".to_string(),
                output: "stdout".to_string(),
                file_path: None,
                max_file_size: None,
                max_files: None,
                structured: true,
            },
            features: FeatureConfig {
                websocket_enabled: true,
                job_queue_enabled: true,
                cache_enabled: true,
                metrics_enabled: true,
                tracing_enabled: true,
                rate_limiting_enabled: true,
                maintenance_mode: false,
            },
        };
        
        // 有效配置应该通过验证
        assert!(ConfigLoader::validate_config(&config).is_ok());
        
        // 无效端口应该失败
        config.server.port = 0;
        assert!(ConfigLoader::validate_config(&config).is_err());
        config.server.port = 8080;
        
        // 空数据库URL应该失败
        config.database.url = "".to_string();
        assert!(ConfigLoader::validate_config(&config).is_err());
        config.database.url = "postgresql://localhost/test".to_string();
        
        // 短JWT密钥应该失败
        config.auth.jwt_secret = "short".to_string();
        assert!(ConfigLoader::validate_config(&config).is_err());
    }
}
