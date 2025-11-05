# HotSpot 部署运维文档

## 1. 系统架构

### 1.1 整体架构

HotSpot 采用前后端分离架构，通过 Nginx 作为反向代理和负载均衡器。

```
┌─────────────────────────────────────────────────────────────┐
│                        Client Layer                         │
│          (Web/Mobile App, WebSocket Clients)                │
└────────────────────────┬────────────────────────────────────┘
                         │ HTTPS / WSS
┌────────────────────────▼────────────────────────────────────┐
│                     Load Balancer                           │
│                  (Nginx / Traefik)                          │
└────────────────────────┬────────────────────────────────────┘
                         │
        ┌────────────────┴────────────────┐
        │                                 │
┌───────▼──────────┐            ┌────────▼─────────┐
│   HTTP Server    │            │  WebSocket Server │
│   (Axum/Tokio)   │            │   (Tungstenite)   │
└───────┬──────────┘            └────────┬──────────┘
        │                                │
        └────────────────┬───────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                  Application Layer                          │
│                  (HotSpot Backend)                          │
└────────────────────────┬────────────────────────────────────┘
                         │
        ┌────────────────┴────────────────┐
        │                                 │
┌───────▼──────────┐            ┌────────▼─────────┐
│   PostgreSQL     │            │      Redis        │
│   (主数据库)      │            │   (缓存/会话)      │
└──────────────────┘            └───────────────────┘
```

### 1.2 网络拓扑

**用户 API 访问流程**：
```
客户端 → https://api.example.com → 前端 Nginx → 后端服务 (8080)
```

**管理员 API 访问流程**：
```
管理员客户端（内网） → https://admin-api.example.com → 后端服务 (8080 或 8081)
```

**说明**：
- 用户 API 通过公网 Nginx 转发，提供负载均衡和 SSL 终止
- 管理员 API 使用独立域名，仅内网可访问，直接连接后端服务
- WebSocket 连接通过 Nginx 的 `proxy_pass` 升级为 WSS

## 2. 管理员 API 部署

### 2.1 独立域名配置

管理员 API 使用独立域名，通过内网 DNS 解析，确保只有内网用户可以访问。

**DNS 配置**：

| 域名 | DNS 类型 | 解析目标 | 说明 |
|---|---|---|---|
| `api.example.com` | 公网 DNS | 前端 Nginx 公网 IP | 用户 API |
| `admin-api.example.com` | 内网 DNS | 后端服务内网 IP（如 `192.168.1.100`） | 管理员 API |

**配置步骤**：

1. **配置内网 DNS**（如 dnsmasq、BIND）：
   ```
   # /etc/hosts 或 DNS 服务器配置
   192.168.1.100  admin-api.example.com
   ```

2. **验证 DNS 解析**：
   ```bash
   # 在内网机器上测试
   nslookup admin-api.example.com
   # 应返回内网 IP：192.168.1.100
   
   # 在公网机器上测试
   nslookup admin-api.example.com
   # 应无法解析或返回不同 IP
   ```

### 2.2 Nginx 配置

**用户 API 配置**（`/etc/nginx/sites-available/hotspot-api`）：

```nginx
upstream backend {
    server 192.168.1.100:8080;
    # 如果有多个后端实例，可以添加更多 server
    # server 192.168.1.101:8080;
}

server {
    listen 80;
    server_name api.example.com;
    
    # 重定向到 HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.example.com;
    
    # SSL 证书配置
    ssl_certificate /etc/letsencrypt/live/api.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.example.com/privkey.pem;
    
    # SSL 安全配置
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    
    # 日志配置
    access_log /var/log/nginx/hotspot-api-access.log;
    error_log /var/log/nginx/hotspot-api-error.log;
    
    # API 路由（不包括管理员 API）
    location /api/v1/public/ {
        proxy_pass http://backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    location /api/v1/protected/ {
        proxy_pass http://backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    # WebSocket 支持
    location /ws {
        proxy_pass http://backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket 超时配置
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }
    
    # 阻止访问管理员 API（防止误配置）
    location /api/v1/admin/ {
        return 403 "Access denied";
    }
}
```

**管理员 API 不需要 Nginx 配置**，因为它直接通过内网 IP 访问后端服务。

### 2.3 SSL 证书配置

**用户 API SSL 证书**（公网域名）：

使用 Let's Encrypt 免费证书：
```bash
# 安装 certbot
sudo apt-get install certbot python3-certbot-nginx

# 获取证书
sudo certbot --nginx -d api.example.com

# 自动续期（添加到 crontab）
0 0 * * * certbot renew --quiet
```

**管理员 API SSL 证书**（内网域名）：

使用内网 CA 签发证书：

1. **创建内网 CA**（如果没有）：
   ```bash
   # 生成 CA 私钥
   openssl genrsa -out ca-key.pem 4096
   
   # 生成 CA 证书
   openssl req -new -x509 -days 3650 -key ca-key.pem -out ca-cert.pem \
     -subj "/C=CN/ST=Beijing/L=Beijing/O=HotSpot/CN=HotSpot Internal CA"
   ```

2. **为管理员 API 生成证书**：
   ```bash
   # 生成私钥
   openssl genrsa -out admin-api-key.pem 2048
   
   # 生成证书签名请求
   openssl req -new -key admin-api-key.pem -out admin-api.csr \
     -subj "/C=CN/ST=Beijing/L=Beijing/O=HotSpot/CN=admin-api.example.com"
   
   # 使用 CA 签发证书
   openssl x509 -req -in admin-api.csr -CA ca-cert.pem -CAkey ca-key.pem \
     -CAcreateserial -out admin-api-cert.pem -days 365
   ```

3. **配置后端服务使用证书**：
   ```bash
   # 将证书放到后端服务可访问的位置
   sudo mkdir -p /etc/hotspot/certs
   sudo cp admin-api-cert.pem /etc/hotspot/certs/
   sudo cp admin-api-key.pem /etc/hotspot/certs/
   sudo chmod 600 /etc/hotspot/certs/admin-api-key.pem
   ```

4. **在后端服务中配置环境变量**：
   ```bash
   # .env 文件
   ADMIN_DOMAIN=admin-api.example.com
   ADMIN_SSL_CERT=/etc/hotspot/certs/admin-api-cert.pem
   ADMIN_SSL_KEY=/etc/hotspot/certs/admin-api-key.pem
   ```

5. **客户端信任内网 CA**：
   ```bash
   # Linux
   sudo cp ca-cert.pem /usr/local/share/ca-certificates/hotspot-ca.crt
   sudo update-ca-certificates
   
   # macOS
   sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ca-cert.pem
   
   # Windows
   # 双击 ca-cert.pem，选择"安装证书" → "受信任的根证书颁发机构"
   ```

### 2.4 安全要求

**防火墙配置**：

```bash
# 允许公网访问用户 API（Nginx）
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# 限制管理员 API 访问（仅内网）
sudo ufw allow from 192.168.1.0/24 to any port 8080
sudo ufw allow from 192.168.1.0/24 to any port 8081

# 拒绝公网访问后端服务
sudo ufw deny 8080/tcp
sudo ufw deny 8081/tcp
```

**访问控制检查清单**：

- ✅ 管理员 API 必须使用 HTTPS
- ✅ 内网 DNS 确保 `admin-api.example.com` 只在内网可解析
- ✅ 配置防火墙规则，限制管理员 API 访问来源
- ✅ 前端 Nginx 不转发 `/api/v1/admin/*` 路由
- ✅ 后端服务记录所有配置变更日志（操作人、时间、变更内容）
- ✅ 定期审计管理员操作日志

## 3. 数据库部署

### 3.1 PostgreSQL 配置

**安装 PostgreSQL 15+**：

```bash
# Ubuntu/Debian
sudo apt-get install postgresql-15 postgresql-contrib-15 postgresql-15-postgis-3

# 启动服务
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

**创建数据库和用户**：

```bash
sudo -u postgres psql

-- 创建用户
CREATE USER hotspot WITH PASSWORD 'your_secure_password';

-- 创建数据库
CREATE DATABASE hotspot OWNER hotspot;

-- 启用 PostGIS 扩展
\c hotspot
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 授予权限
GRANT ALL PRIVILEGES ON DATABASE hotspot TO hotspot;
```

**配置连接**（`/etc/postgresql/15/main/postgresql.conf`）：

```ini
# 监听地址
listen_addresses = 'localhost,192.168.1.100'

# 最大连接数
max_connections = 200

# 共享缓冲区（建议为系统内存的 25%）
shared_buffers = 2GB

# 工作内存
work_mem = 16MB

# 维护工作内存
maintenance_work_mem = 512MB

# 有效缓存大小（建议为系统内存的 50-75%）
effective_cache_size = 6GB

# WAL 配置
wal_level = replica
max_wal_size = 2GB
min_wal_size = 1GB
```

**配置访问控制**（`/etc/postgresql/15/main/pg_hba.conf`）：

```
# 本地连接
local   all             all                                     peer

# 内网连接
host    hotspot         hotspot         192.168.1.0/24          scram-sha-256

# 禁止公网连接
host    all             all             0.0.0.0/0               reject
```

**运行数据库迁移**：

```bash
cd /path/to/hotspot/backend

# 设置数据库连接
export DATABASE_URL="postgresql://hotspot:your_secure_password@localhost/hotspot"

# 运行迁移
sqlx migrate run
```

### 3.2 Redis 配置

**安装 Redis 7+**：

```bash
# Ubuntu/Debian
sudo apt-get install redis-server

# 启动服务
sudo systemctl start redis-server
sudo systemctl enable redis-server
```

**配置 Redis**（`/etc/redis/redis.conf`）：

```ini
# 绑定地址
bind 127.0.0.1 192.168.1.100

# 端口
port 6379

# 密码认证
requirepass your_redis_password

# 最大内存（根据系统配置调整）
maxmemory 1gb

# 内存淘汰策略
maxmemory-policy allkeys-lru

# 持久化配置
save 900 1
save 300 10
save 60 10000

# AOF 持久化
appendonly yes
appendfsync everysec
```

**测试连接**：

```bash
redis-cli -h 192.168.1.100 -a your_redis_password ping
# 应返回 PONG
```

### 3.3 数据库备份

**PostgreSQL 备份脚本**（`/usr/local/bin/backup-hotspot-db.sh`）：

```bash
#!/bin/bash

BACKUP_DIR="/var/backups/hotspot"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/hotspot_$TIMESTAMP.sql.gz"

# 创建备份目录
mkdir -p $BACKUP_DIR

# 执行备份
pg_dump -U hotspot -h localhost hotspot | gzip > $BACKUP_FILE

# 删除 7 天前的备份
find $BACKUP_DIR -name "hotspot_*.sql.gz" -mtime +7 -delete

echo "Backup completed: $BACKUP_FILE"
```

**添加到 crontab**：

```bash
# 每天凌晨 2 点备份
0 2 * * * /usr/local/bin/backup-hotspot-db.sh
```

**Redis 备份**：

Redis 的 RDB 和 AOF 文件会自动保存在 `/var/lib/redis/` 目录下，定期备份该目录即可。

```bash
# 手动触发 RDB 快照
redis-cli -a your_redis_password BGSAVE

# 备份 RDB 文件
cp /var/lib/redis/dump.rdb /var/backups/hotspot/redis_$(date +%Y%m%d_%H%M%S).rdb
```

## 4. 后端服务部署

### 4.1 编译后端服务

```bash
cd /path/to/hotspot/backend

# 编译 Release 版本
cargo build --release

# 编译后的二进制文件位于
# target/release/hotspot-backend
```

### 4.2 配置环境变量

创建 `.env` 文件：

```bash
# 数据库配置
DATABASE_URL=postgresql://hotspot:your_secure_password@localhost/hotspot

# Redis 配置
REDIS_URL=redis://:your_redis_password@localhost:6379

# JWT 密钥（使用强随机字符串）
JWT_SECRET=your_jwt_secret_key_here

# 服务器配置
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# 管理员 API 配置
ADMIN_DOMAIN=admin-api.example.com
ADMIN_SSL_CERT=/etc/hotspot/certs/admin-api-cert.pem
ADMIN_SSL_KEY=/etc/hotspot/certs/admin-api-key.pem

# 日志级别
RUST_LOG=info

# 文件上传配置
UPLOAD_DIR=/var/hotspot/uploads
MAX_UPLOAD_SIZE=10485760  # 10MB
```

### 4.3 Systemd 服务配置

创建 systemd 服务文件（`/etc/systemd/system/hotspot-backend.service`）：

```ini
[Unit]
Description=HotSpot Backend Service
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=hotspot
Group=hotspot
WorkingDirectory=/opt/hotspot/backend
Environment="RUST_LOG=info"
EnvironmentFile=/opt/hotspot/backend/.env
ExecStart=/opt/hotspot/backend/target/release/hotspot-backend
Restart=always
RestartSec=5

# 安全配置
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/hotspot/uploads /var/log/hotspot

[Install]
WantedBy=multi-user.target
```

**启动服务**：

```bash
# 重新加载 systemd 配置
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start hotspot-backend

# 设置开机自启
sudo systemctl enable hotspot-backend

# 查看状态
sudo systemctl status hotspot-backend

# 查看日志
sudo journalctl -u hotspot-backend -f
```

## 5. 监控与日志

### 5.1 日志配置

**应用日志**：

后端服务使用 `tracing` 库记录日志，日志输出到标准输出，由 systemd 收集。

查看日志：
```bash
# 实时查看日志
sudo journalctl -u hotspot-backend -f

# 查看最近 100 条日志
sudo journalctl -u hotspot-backend -n 100

# 查看特定时间段的日志
sudo journalctl -u hotspot-backend --since "2025-10-30 00:00:00" --until "2025-10-30 23:59:59"
```

**Nginx 日志**：

- 访问日志：`/var/log/nginx/hotspot-api-access.log`
- 错误日志：`/var/log/nginx/hotspot-api-error.log`

**日志轮转**（`/etc/logrotate.d/hotspot`）：

```
/var/log/nginx/hotspot-*.log {
    daily
    rotate 30
    compress
    delaycompress
    notifempty
    create 0640 www-data adm
    sharedscripts
    postrotate
        [ -f /var/run/nginx.pid ] && kill -USR1 `cat /var/run/nginx.pid`
    endscript
}
```

### 5.2 性能监控

**系统监控工具**：

- **htop**：实时查看系统资源使用情况
- **iotop**：监控磁盘 I/O
- **nethogs**：监控网络流量

**数据库监控**：

```bash
# PostgreSQL 连接数
psql -U hotspot -c "SELECT count(*) FROM pg_stat_activity;"

# 慢查询日志
# 在 postgresql.conf 中启用
log_min_duration_statement = 1000  # 记录超过 1 秒的查询

# 查看慢查询
sudo tail -f /var/log/postgresql/postgresql-15-main.log | grep "duration:"
```

**Redis 监控**：

```bash
# 实时监控 Redis 命令
redis-cli -a your_redis_password monitor

# 查看 Redis 信息
redis-cli -a your_redis_password info

# 查看内存使用
redis-cli -a your_redis_password info memory
```

### 5.3 告警配置

**磁盘空间告警**：

```bash
# 添加到 crontab
0 * * * * /usr/local/bin/check-disk-space.sh
```

**check-disk-space.sh**：
```bash
#!/bin/bash

THRESHOLD=80
USAGE=$(df / | tail -1 | awk '{print $5}' | sed 's/%//')

if [ $USAGE -gt $THRESHOLD ]; then
    echo "Disk usage is above $THRESHOLD%: $USAGE%" | mail -s "HotSpot Disk Alert" admin@example.com
fi
```

**服务健康检查**：

```bash
# 添加到 crontab
*/5 * * * * /usr/local/bin/check-service-health.sh
```

**check-service-health.sh**：
```bash
#!/bin/bash

# 检查后端服务
if ! systemctl is-active --quiet hotspot-backend; then
    echo "HotSpot backend is down!" | mail -s "HotSpot Service Alert" admin@example.com
    systemctl start hotspot-backend
fi

# 检查 API 健康
if ! curl -f http://localhost:8080/api/v1/public/system/health > /dev/null 2>&1; then
    echo "HotSpot API health check failed!" | mail -s "HotSpot API Alert" admin@example.com
fi
```

## 6. 安全加固

### 6.1 系统安全

**更新系统**：
```bash
sudo apt-get update
sudo apt-get upgrade -y
```

**配置防火墙**：
```bash
# 启用 UFW
sudo ufw enable

# 允许 SSH（修改为非标准端口更安全）
sudo ufw allow 22/tcp

# 允许 HTTP/HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# 限制内网访问后端服务
sudo ufw allow from 192.168.1.0/24 to any port 8080

# 查看规则
sudo ufw status
```

**禁用不必要的服务**：
```bash
sudo systemctl disable bluetooth
sudo systemctl disable cups
```

### 6.2 应用安全

**环境变量保护**：
```bash
# 确保 .env 文件权限正确
chmod 600 /opt/hotspot/backend/.env
chown hotspot:hotspot /opt/hotspot/backend/.env
```

**定期更新依赖**：
```bash
cd /opt/hotspot/backend
cargo update
cargo audit
```

**限制文件上传**：
- 在 Nginx 中限制请求体大小
- 验证文件类型和大小
- 使用独立的存储服务（如 S3）

### 6.3 数据库安全

**定期更新密码**：
```sql
ALTER USER hotspot WITH PASSWORD 'new_secure_password';
```

**限制数据库权限**：
```sql
-- 只授予必要的权限
REVOKE ALL ON DATABASE hotspot FROM PUBLIC;
GRANT CONNECT ON DATABASE hotspot TO hotspot;
GRANT ALL ON ALL TABLES IN SCHEMA public TO hotspot;
```

**启用 SSL 连接**：
```ini
# postgresql.conf
ssl = on
ssl_cert_file = '/etc/postgresql/15/main/server.crt'
ssl_key_file = '/etc/postgresql/15/main/server.key'
```

## 7. 故障排查

### 7.1 常见问题

**问题 1：后端服务无法启动**

检查步骤：
```bash
# 查看服务状态
sudo systemctl status hotspot-backend

# 查看详细日志
sudo journalctl -u hotspot-backend -n 100

# 检查端口占用
sudo netstat -tlnp | grep 8080

# 检查数据库连接
psql -U hotspot -h localhost -d hotspot
```

**问题 2：数据库连接失败**

检查步骤：
```bash
# 检查 PostgreSQL 服务状态
sudo systemctl status postgresql

# 检查连接配置
cat /etc/postgresql/15/main/pg_hba.conf

# 测试连接
psql -U hotspot -h localhost -d hotspot
```

**问题 3：Redis 连接失败**

检查步骤：
```bash
# 检查 Redis 服务状态
sudo systemctl status redis-server

# 测试连接
redis-cli -h localhost -a your_redis_password ping

# 查看 Redis 日志
sudo tail -f /var/log/redis/redis-server.log
```

**问题 4：管理员 API 无法访问**

检查步骤：
```bash
# 检查 DNS 解析
nslookup admin-api.example.com

# 检查防火墙规则
sudo ufw status

# 检查 SSL 证书
openssl s_client -connect admin-api.example.com:8080 -showcerts

# 检查后端日志
sudo journalctl -u hotspot-backend -f
```

### 7.2 性能优化

**数据库优化**：

```sql
-- 分析表统计信息
ANALYZE;

-- 重建索引
REINDEX DATABASE hotspot;

-- 清理死元组
VACUUM FULL;
```

**Redis 优化**：

```bash
# 清理过期键
redis-cli -a your_redis_password --scan --pattern "*" | xargs redis-cli -a your_redis_password DEL

# 查看慢查询
redis-cli -a your_redis_password slowlog get 10
```

**应用优化**：

- 启用 HTTP/2
- 配置 Gzip 压缩
- 使用 CDN 加速静态资源
- 优化数据库查询（使用 EXPLAIN ANALYZE）
- 增加 Redis 缓存命中率

## 8. 扩展与高可用

### 8.1 水平扩展

**后端服务扩展**：

1. 部署多个后端实例
2. 在 Nginx 中配置负载均衡
3. 使用 Redis 共享会话

**Nginx 负载均衡配置**：

```nginx
upstream backend {
    least_conn;  # 最少连接算法
    server 192.168.1.100:8080 weight=1;
    server 192.168.1.101:8080 weight=1;
    server 192.168.1.102:8080 weight=1;
}
```

### 8.2 数据库高可用

**PostgreSQL 主从复制**：

1. 配置主服务器
2. 配置从服务器
3. 使用 pgpool-II 或 HAProxy 进行读写分离

**Redis 高可用**：

使用 Redis Sentinel 或 Redis Cluster：

```bash
# 启动 Redis Sentinel
redis-sentinel /etc/redis/sentinel.conf
```

### 8.3 容器化部署

**Docker Compose 示例**：

```yaml
version: '3.8'

services:
  backend:
    build: ./backend
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgresql://hotspot:password@postgres:5432/hotspot
      - REDIS_URL=redis://:password@redis:6379
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgis/postgis:15-3.3
    environment:
      - POSTGRES_USER=hotspot
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=hotspot
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass password
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```

## 9. 附录

### 9.1 环境变量参考

| 变量名 | 说明 | 示例值 |
|---|---|---|
| `DATABASE_URL` | PostgreSQL 连接字符串 | `postgresql://user:pass@host/db` |
| `REDIS_URL` | Redis 连接字符串 | `redis://:pass@host:6379` |
| `JWT_SECRET` | JWT 签名密钥 | `your_secret_key` |
| `SERVER_HOST` | 服务器监听地址 | `0.0.0.0` |
| `SERVER_PORT` | 服务器监听端口 | `8080` |
| `ADMIN_DOMAIN` | 管理员 API 域名 | `admin-api.example.com` |
| `ADMIN_SSL_CERT` | 管理员 API SSL 证书路径 | `/etc/hotspot/certs/cert.pem` |
| `ADMIN_SSL_KEY` | 管理员 API SSL 私钥路径 | `/etc/hotspot/certs/key.pem` |
| `RUST_LOG` | 日志级别 | `info` / `debug` / `warn` |
| `UPLOAD_DIR` | 文件上传目录 | `/var/hotspot/uploads` |
| `MAX_UPLOAD_SIZE` | 最大上传大小（字节） | `10485760` (10MB) |

### 9.2 端口参考

| 端口 | 服务 | 说明 |
|---|---|---|
| 80 | Nginx | HTTP（重定向到 HTTPS） |
| 443 | Nginx | HTTPS（用户 API） |
| 8080 | Backend | 后端服务（内网） |
| 8081 | Backend | 管理员 API（可选，内网） |
| 5432 | PostgreSQL | 数据库（内网） |
| 6379 | Redis | 缓存（内网） |

### 9.3 相关文档

- [HotSpot API 规范](../.cursor/rules/api_spec.mdc)
- [后端开发规范](../.cursor/rules/backend_dev_rule.mdc)
- [后端开发需求](../.cursor/rules/backend_requirements.mdc)
- [PostgreSQL 官方文档](https://www.postgresql.org/docs/)
- [Redis 官方文档](https://redis.io/documentation)
- [Nginx 官方文档](https://nginx.org/en/docs/)











