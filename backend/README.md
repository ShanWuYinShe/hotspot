# HotSpot 后端服务

基于 Rust 和 PostgreSQL 构建的地理位置社交平台后端服务，采用现代化的端口适配器架构（Hexagonal Architecture）。

## 🏗️ 架构概览

### 新架构设计

HotSpot 后端采用端口适配器架构，实现了清晰的分层设计：

```
┌─────────────────────────────────────────────────────────────┐
│                    Presentation Layer                        │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │  HTTP Controllers│  │  WebSocket      │  │  Middleware     │ │
│  │                 │  │  Handlers       │  │                 │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                         │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │  User Service   │  │  Group Service  │  │  Message Service│ │
│  │                 │  │                 │  │                 │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                      Domain Layer                           │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │  Entities       │  │  Value Objects  │  │  Domain Events  │ │
│  │                 │  │                 │  │                 │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                      Ports Layer                            │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │  Repository     │  │  Cache Port     │  │  Job Dispatcher │ │
│  │  Ports          │  │                 │  │  Port           │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                    Adapters Layer                           │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │  PostgreSQL     │  │  Memory Cache   │  │  Lite Job Queue │ │
│  │  Repository     │  │                 │  │                 │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### 核心特性

- **🔧 端口适配器架构**: 清晰的分层设计，易于测试和维护
- **🎯 领域驱动设计**: 业务逻辑封装在领域层
- **⚡ CQRS模式**: 命令查询分离，优化性能
- **🔄 实时通信**: WebSocket支持实时消息
- **📍 地理位置**: PostGIS集成的高性能地理查询
- **🔀 异步任务**: 基于tokio的异步处理
- **💾 缓存系统**: 多层缓存策略
- **📊 监控指标**: 完整的健康检查和指标收集
- **🔒 类型安全**: 强类型系统和泛型依赖注入

## 🚀 快速开始

### 环境要求

- Rust 1.70+
- PostgreSQL 14+ (with PostGIS extension)
- Redis (可选，用于缓存)

### 安装和运行

1. **克隆项目**
```bash
git clone <repository-url>
cd hotspot/backend
```

2. **设置环境变量**
```bash
cp .env.example .env
# 编辑 .env 文件，设置数据库连接等配置
```

3. **运行数据库迁移**
```bash
# 确保PostgreSQL运行并创建了数据库
sqlx migrate run
```

4. **启动服务**
```bash
cargo run
```

服务将在 `http://localhost:8080` 启动。

### 环境变量配置

| 变量名 | 描述 | 默认值 |
|--------|------|--------|
| `DATABASE_URL` | PostgreSQL连接字符串 | 必需 |
| `JWT_SECRET` | JWT签名密钥 | 开发环境有默认值 |
| `REDIS_URL` | Redis连接字符串 | 可选 |
| `HOST` | 服务监听地址 | `0.0.0.0` |
| `PORT` | 服务监听端口 | `8080` |

## 📦 项目结构

```
backend/
├── src/
│   ├── adapters/           # 适配器层 - 外部系统集成
│   │   ├── adapters_pg/    # PostgreSQL适配器
│   │   ├── adapters_cache_memory.rs  # 内存缓存适配器
│   │   └── adapters_jobs_lite.rs     # 轻量级任务队列
│   ├── application/        # 应用层 - 用例和应用服务
│   │   ├── commands/       # 命令处理器
│   │   ├── queries/        # 查询处理器
│   │   └── services/       # 应用服务
│   ├── bootstrap/          # 启动层 - 依赖注入和配置
│   │   ├── app_state.rs    # 应用状态管理
│   │   ├── config_adapter.rs # 配置适配器
│   │   └── dependency_injection.rs # 依赖注入
│   ├── domain/             # 领域层 - 核心业务逻辑
│   │   ├── entities/       # 实体
│   │   ├── values/         # 值对象
│   │   └── events/         # 领域事件
│   ├── ports/              # 端口层 - 抽象接口
│   │   ├── repository_ports.rs # 仓储接口
│   │   ├── cache_ports.rs      # 缓存接口
│   │   └── job_dispatcher_ports.rs # 任务调度接口
│   ├── presentation/       # 表现层 - HTTP和WebSocket
│   │   ├── http/           # HTTP处理
│   │   │   ├── controllers/ # 控制器
│   │   │   ├── middleware/  # 中间件
│   │   │   └── routes.rs    # 路由定义
│   │   └── ws/             # WebSocket处理
│   ├── utils/              # 工具函数
│   └── main.rs             # 应用入口
├── migrations/             # 数据库迁移文件
└── Cargo.toml             # 项目配置
```

## 🔌 API 端点

### 健康检查
- `POST /health` - 基础健康检查
- `POST /health/detailed` - 详细组件状态
- `POST /ready` - 就绪状态检查
- `POST /alive` - 存活状态检查

### 用户管理
- `POST /api/v1/users/register` - 用户注册
- `POST /api/v1/users/temp` - 创建临时用户
- `POST /api/v1/users/{user_id}/upgrade` - 升级临时用户
- `POST /api/v1/users/{user_id}/profile` - 更新用户资料
- `POST /api/v1/users/nearby` - 查找附近用户

### 群组管理
- `POST /api/v1/groups/create` - 创建群组
- `POST /api/v1/groups/{group_id}/join` - 加入群组
- `POST /api/v1/groups/{group_id}/leave` - 离开群组
- `POST /api/v1/groups/nearby` - 查找附近群组

### 消息系统
- `POST /api/v1/messages/send` - 发送消息
- `POST /api/v1/messages/history` - 获取消息历史
- WebSocket: `/ws` - 实时消息连接

### 瞬间分享
- `POST /api/v1/moments/create` - 发布瞬间
- `POST /api/v1/moments/nearby` - 获取附近瞬间
- `POST /api/v1/moments/{moment_id}` - 获取瞬间详情

## 🧪 测试

运行测试：
```bash
# 运行所有测试
cargo test

# 运行特定测试
cargo test test_user_creation

# 运行集成测试
cargo test --test integration_tests
```

## 📊 监控和指标

### 健康检查端点

系统提供多层次的健康检查：

1. **基础健康检查** (`/health`)
   - 返回服务基本状态
   - 包含版本信息和时间戳

2. **详细健康检查** (`/health/detailed`)
   - 检查数据库连接状态
   - 检查缓存系统状态
   - 检查任务队列状态
   - 提供详细的性能指标

3. **就绪检查** (`/ready`)
   - 验证关键依赖是否可用
   - 适用于Kubernetes就绪探针

4. **存活检查** (`/alive`)
   - 验证应用进程是否正常
   - 适用于Kubernetes存活探针

### 指标收集

- 数据库连接池指标
- 缓存命中率统计
- 任务队列处理统计
- HTTP请求响应时间
- WebSocket连接数量

## 🔧 开发指南

### 添加新功能

1. **定义领域模型** - 在 `domain/` 中添加实体和值对象
2. **创建端口接口** - 在 `ports/` 中定义抽象接口
3. **实现适配器** - 在 `adapters/` 中实现具体功能
4. **编写应用服务** - 在 `application/services/` 中编写业务逻辑
5. **添加控制器** - 在 `presentation/http/controllers/` 中处理HTTP请求
6. **更新路由** - 在 `presentation/http/routes.rs` 中添加路由

### 代码规范

- 使用 `cargo fmt` 格式化代码
- 使用 `cargo clippy` 进行代码检查
- 遵循 Rust 命名约定
- 为公共API编写文档注释
- 编写单元测试和集成测试

### 数据库迁移

```bash
# 创建新迁移
sqlx migrate add <migration_name>

# 运行迁移
sqlx migrate run

# 回滚迁移
sqlx migrate revert
```

## 🚀 部署

### Docker部署

```dockerfile
# Dockerfile示例
FROM rust:1.70 as builder
WORKDIR /app
COPY . .
RUN cargo build --release

FROM debian:bullseye-slim
RUN apt-get update && apt-get install -y ca-certificates
COPY --from=builder /app/target/release/backend /usr/local/bin/backend
CMD ["backend"]
```

### Kubernetes部署

```yaml
# k8s-deployment.yaml示例
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hotspot-backend
spec:
  replicas: 3
  selector:
    matchLabels:
      app: hotspot-backend
  template:
    metadata:
      labels:
        app: hotspot-backend
    spec:
      containers:
      - name: backend
        image: hotspot-backend:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: hotspot-secrets
              key: database-url
        livenessProbe:
          httpPost:
            path: /alive
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpPost:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

## 🤝 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 打开 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🆘 故障排除

### 常见问题

1. **数据库连接失败**
   - 检查 `DATABASE_URL` 环境变量
   - 确保PostgreSQL服务正在运行
   - 验证数据库用户权限

2. **编译错误**
   - 更新 Rust 版本: `rustup update`
   - 清理构建缓存: `cargo clean`
   - 检查依赖版本兼容性

3. **测试失败**
   - 确保测试数据库可用
   - 运行数据库迁移
   - 检查测试环境配置

### 性能调优

1. **数据库优化**
   - 调整连接池大小
   - 添加适当的索引
   - 优化查询语句

2. **缓存策略**
   - 启用Redis缓存
   - 调整缓存TTL
   - 监控缓存命中率

3. **并发处理**
   - 调整tokio运行时配置
   - 优化异步任务处理
   - 监控系统资源使用

---

📧 如有问题，请通过 [Issues](../../issues) 联系我们。
