# HotSpot 后端生产环境配置指南

## 概述

本文档提供了 HotSpot 后端服务在生产环境中的完整配置指南，包括所有必要的环境变量、安全设置和性能优化建议。

## 环境变量配置

### 基础应用配置

```bash
# 环境类型（必须设置为 production）
ENVIRONMENT=production

# 服务器配置
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# CORS 配置
CORS_ORIGINS=https://yourdomain.com,https://www.yourdomain.com
```

### 数据库配置

```bash
# PostgreSQL 连接
DATABASE_URL=postgresql://hotspot_user:secure_password@localhost:5432/hotspot_prod

# 连接池配置
DB_MAX_CONNECTIONS=20
DB_MIN_CONNECTIONS=5
DB_CONNECT_TIMEOUT=30
DB_IDLE_TIMEOUT=600
DB_MAX_LIFETIME=3600
DB_ACQUIRE_TIMEOUT=30
```

### 认证配置

```bash
# JWT 配置（使用强随机密钥）
JWT_SECRET=your_very_secure_jwt_secret_key_at_least_32_characters_long
JWT_ACCESS_EXPIRES_IN=3600      # 1小时
JWT_REFRESH_EXPIRES_IN=2592000  # 30天
```

### 缓存配置

```bash
# Redis 配置
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=your_redis_password
REDIS_DB=0

# 内存缓存
CACHE_MAX_SIZE=1000
CACHE_TTL=3600
```

### 日志配置

```bash
# 日志级别和格式
RUST_LOG=info,backend=info,tower_http=info
LOG_FORMAT=json
```

### 监控配置

```bash
# 性能监控
ENABLE_PERFORMANCE_MONITORING=true
ENABLE_ERROR_MONITORING=true
ENABLE_DB_OPTIMIZER=true
METRICS_COLLECTION_INTERVAL=60
```

### WebSocket 配置

```bash
# WebSocket 限制
WS_MAX_CONNECTIONS=10000
WS_CONNECTION_TIMEOUT=300
WS_HEARTBEAT_INTERVAL=30
```

### 任务队列配置

```bash
# 任务队列
ENABLE_JOBS=true
JOB_QUEUE_TYPE=redis
JOB_MAX_RETRIES=3
JOB_RETRY_DELAY=60
```

### 通知系统配置

```bash
# 通知策略
ENABLE_NOTIFICATIONS=true
ENABLE_FRIEND_NOTIFICATIONS=true
ENABLE_GROUP_NOTIFICATIONS=true
ENABLE_NEARBY_NOTIFICATIONS=true
NEARBY_RADIUS_KM=5.0
ENABLE_GLOBAL_BROADCAST=false
```

### 地理位置配置

```bash
# 默认位置
DEFAULT_LATITUDE=39.9042
DEFAULT_LONGITUDE=116.4074
DEFAULT_LOCATION_NAME=北京
MAX_SEARCH_RADIUS_KM=50.0
```

### 安全配置

```bash
# 安全设置
RATE_LIMIT_REQUESTS_PER_MINUTE=100
RATE_LIMIT_BURST_SIZE=20
REQUEST_TIMEOUT=30
FORCE_HTTPS=true
ENABLE_SECURITY_HEADERS=true
```

### 健康检查配置

```bash
# 健康检查
ENABLE_HEALTH_CHECKS=true
ENABLE_DETAILED_HEALTH_CHECKS=true
HEALTH_CHECK_TIMEOUT=10
```

## 部署清单

### 1. 系统要求

- **操作系统**: Ubuntu 20.04+ / CentOS 8+ / RHEL 8+
- **CPU**: 最少 2 核心，推荐 4+ 核心
- **内存**: 最少 4GB，推荐 8GB+
- **存储**: 最少 50GB SSD
- **网络**: 稳定的互联网连接

### 2. 依赖服务

- **PostgreSQL 14+**: 主数据库
- **Redis 6+**: 缓存和任务队列
- **Nginx**: 反向代理和负载均衡

### 3. 安全配置

#### 防火墙设置
```bash
# 只开放必要端口
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP
ufw allow 443/tcp   # HTTPS
ufw enable
```

#### SSL/TLS 配置
- 使用 Let's Encrypt 或商业 SSL 证书
- 配置 HTTPS 重定向
- 启用 HSTS 头

#### 数据库安全
- 创建专用数据库用户
- 限制数据库访问权限
- 启用数据库连接加密

### 4. 性能优化

#### 数据库优化
```sql
-- PostgreSQL 配置优化
shared_buffers = 256MB
effective_cache_size = 1GB
work_mem = 4MB
maintenance_work_mem = 64MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
```

#### 系统优化
```bash
# 增加文件描述符限制
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# 优化网络参数
echo "net.core.somaxconn = 65536" >> /etc/sysctl.conf
echo "net.ipv4.tcp_max_syn_backlog = 65536" >> /etc/sysctl.conf
```

### 5. 监控和日志

#### 日志管理
- 配置日志轮转（logrotate）
- 设置日志保留策略
- 监控日志文件大小

#### 系统监控
- CPU、内存、磁盘使用率
- 数据库连接数和性能
- WebSocket 连接数
- API 响应时间

#### 告警配置
```bash
# 告警阈值
CPU_ALERT_THRESHOLD=80
MEMORY_ALERT_THRESHOLD=85
DISK_ALERT_THRESHOLD=90
DB_CONNECTION_ALERT_THRESHOLD=15
```

### 6. 备份策略

#### 数据库备份
```bash
# 每日自动备份脚本
#!/bin/bash
BACKUP_DIR="/var/backups/hotspot"
DATE=$(date +%Y%m%d_%H%M%S)
pg_dump hotspot_prod > "$BACKUP_DIR/hotspot_$DATE.sql"
find $BACKUP_DIR -name "hotspot_*.sql" -mtime +30 -delete
```

#### 配置文件备份
- 定期备份环境配置文件
- 版本控制配置变更
- 测试备份恢复流程

### 7. 部署流程

#### 1. 环境准备
```bash
# 创建应用用户
useradd -m -s /bin/bash hotspot
usermod -aG sudo hotspot

# 创建应用目录
mkdir -p /opt/hotspot
chown hotspot:hotspot /opt/hotspot
```

#### 2. 应用部署
```bash
# 编译发布版本
cargo build --release

# 复制二进制文件
cp target/release/backend /opt/hotspot/
chmod +x /opt/hotspot/backend

# 创建配置文件
cp .env.production /opt/hotspot/.env
```

#### 3. 系统服务配置
```ini
# /etc/systemd/system/hotspot.service
[Unit]
Description=HotSpot Backend Service
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=hotspot
WorkingDirectory=/opt/hotspot
ExecStart=/opt/hotspot/backend
Restart=always
RestartSec=10
Environment=RUST_LOG=info

[Install]
WantedBy=multi-user.target
```

#### 4. 启动服务
```bash
systemctl daemon-reload
systemctl enable hotspot
systemctl start hotspot
systemctl status hotspot
```

### 8. 维护和更新

#### 定期维护任务
- 数据库统计信息更新
- 日志文件清理
- 系统安全更新
- 依赖包更新

#### 更新流程
1. 备份当前版本
2. 测试新版本
3. 滚动更新部署
4. 验证服务状态
5. 回滚计划准备

### 9. 故障排除

#### 常见问题
1. **数据库连接失败**
   - 检查数据库服务状态
   - 验证连接字符串
   - 检查防火墙设置

2. **内存不足**
   - 调整连接池大小
   - 优化缓存配置
   - 增加系统内存

3. **WebSocket 连接问题**
   - 检查代理配置
   - 验证防火墙规则
   - 监控连接数限制

#### 日志分析
```bash
# 查看应用日志
journalctl -u hotspot -f

# 查看错误日志
grep ERROR /var/log/hotspot/app.log

# 监控性能指标
curl http://localhost:8080/health/detailed
```

### 10. 安全最佳实践

1. **定期安全审计**
2. **及时更新依赖**
3. **最小权限原则**
4. **网络隔离**
5. **数据加密**
6. **访问日志监控**
7. **入侵检测**
8. **定期渗透测试**

## 联系支持

如有部署或配置问题，请联系技术支持团队。

---

*最后更新: 2024年1月*
