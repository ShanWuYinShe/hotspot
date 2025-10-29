use std::env;

#[derive(Debug, Clone)]
pub struct Config {
    pub database_url: String,
    pub redis_url: String,
    pub jwt_secret: String,
    pub jwt_access_token_expiration: i64,
    pub jwt_refresh_token_expiration: i64,
    pub server_host: String,
    pub server_port: u16,
    pub max_upload_size: usize,
    pub upload_dir: String,
    pub rate_limit_global_per_hour: u32,
    pub rate_limit_user_per_hour: u32,
}

impl Config {
    pub fn from_env() -> Result<Self, String> {
        dotenvy::dotenv().ok();

        Ok(Config {
            database_url: env::var("DATABASE_URL")
                .map_err(|_| "DATABASE_URL must be set".to_string())?,
            redis_url: env::var("REDIS_URL")
                .unwrap_or_else(|_| "redis://localhost:6379".to_string()),
            jwt_secret: env::var("JWT_SECRET")
                .map_err(|_| "JWT_SECRET must be set".to_string())?,
            jwt_access_token_expiration: env::var("JWT_ACCESS_TOKEN_EXPIRATION")
                .unwrap_or_else(|_| "3600".to_string())
                .parse()
                .map_err(|_| "Invalid JWT_ACCESS_TOKEN_EXPIRATION".to_string())?,
            jwt_refresh_token_expiration: env::var("JWT_REFRESH_TOKEN_EXPIRATION")
                .unwrap_or_else(|_| "2592000".to_string())
                .parse()
                .map_err(|_| "Invalid JWT_REFRESH_TOKEN_EXPIRATION".to_string())?,
            server_host: env::var("SERVER_HOST")
                .unwrap_or_else(|_| "0.0.0.0".to_string()),
            server_port: env::var("SERVER_PORT")
                .unwrap_or_else(|_| "8080".to_string())
                .parse()
                .map_err(|_| "Invalid SERVER_PORT".to_string())?,
            max_upload_size: env::var("MAX_UPLOAD_SIZE")
                .unwrap_or_else(|_| "10485760".to_string())
                .parse()
                .map_err(|_| "Invalid MAX_UPLOAD_SIZE".to_string())?,
            upload_dir: env::var("UPLOAD_DIR")
                .unwrap_or_else(|_| "./uploads".to_string()),
            rate_limit_global_per_hour: env::var("RATE_LIMIT_GLOBAL_PER_HOUR")
                .unwrap_or_else(|_| "1000".to_string())
                .parse()
                .map_err(|_| "Invalid RATE_LIMIT_GLOBAL_PER_HOUR".to_string())?,
            rate_limit_user_per_hour: env::var("RATE_LIMIT_USER_PER_HOUR")
                .unwrap_or_else(|_| "5000".to_string())
                .parse()
                .map_err(|_| "Invalid RATE_LIMIT_USER_PER_HOUR".to_string())?,
        })
    }

    pub fn server_addr(&self) -> String {
        format!("{}:{}", self.server_host, self.server_port)
    }
}

