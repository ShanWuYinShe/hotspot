# HotSpot 后端集成测试

## 📋 测试概述

本目录包含 HotSpot 后端的集成测试和性能测试，用于验证核心功能和性能指标。

### 测试分类

1. **集成测试**（Integration Tests）
   - `integration_auth.rs` - 认证模块测试
   - `integration_user.rs` - 用户模块测试
   - `integration_message.rs` - 消息模块测试（待实现）
   - `integration_moment.rs` - 生活瞬间模块测试（待实现）

2. **性能测试**（Performance Tests）
   - `performance_location.rs` - 地理位置查询性能测试
   - `performance_table_separation.rs` - 表分离效果验证

## 🚀 运行测试

### 前置要求

1. **PostgreSQL 15+** 运行中，已安装 PostGIS 扩展
2. **Redis 7+** 运行中
3. 测试数据库配置（环境变量或 `.env.test` 文件）

### 环境配置

创建 `.env.test` 文件：

```bash
DATABASE_URL=postgresql://hotspot_user:your_password@localhost:5432/hotspot_test
REDIS_URL=redis://127.0.0.1:6379
JWT_SECRET=test_secret_key_for_integration_tests
```

### 运行所有测试

```bash
cd backend
cargo test --tests
```

### 运行特定测试

```bash
# 运行认证模块测试
cargo test --test integration_auth

# 运行用户模块测试
cargo test --test integration_user

# 运行性能测试
cargo test --test performance_location
```

### 运行单个测试用例

```bash
# 测试用户注册
cargo test --test integration_auth test_user_register

# 测试地理位置查询性能
cargo test --test performance_location test_location_query_performance -- --nocapture
```

## 📊 测试覆盖率

### 认证模块 (integration_auth.rs)

- ✅ 用户注册 - 正常流程
- ✅ 用户登录 - 正常流程
- ✅ Token 刷新 - 正常流程
- ✅ 重复注册 - 错误场景
- ✅ 错误密码登录 - 错误场景
- ⏳ 无效 Token - 待实现
- ⏳ Token 过期 - 待实现

### 用户模块 (integration_user.rs)

- ✅ 按昵称搜索用户
- ✅ 获取用户资料
- ✅ 更新用户资料
- ✅ 好友申请流程（发送、查看、接受）
- ⏳ 关注/取消关注 - 待实现
- ⏳ 拉黑/解除拉黑 - 待实现

### 性能测试 (performance_location.rs)

- ✅ 地理位置查询性能（10000条数据，100次查询）
- ✅ GIST 索引使用验证
- ✅ 表分离性能验证（未读计数 vs 成员列表）

## 🎯 性能基准

### 地理位置查询

- **数据量**：10,000 条用户位置记录
- **查询次数**：100 次
- **半径**：5 公里
- **预期性能**：
  - 平均查询时间 < 50ms
  - 必须使用 GIST 索引
  - 总查询时间 < 5秒

### 表分离效果

- **测试场景**：群组消息发送（更新未读计数）与成员列表查询并发
- **数据量**：100 个群组成员
- **操作次数**：100 次更新，100 次查询
- **预期性能**：
  - 未读计数更新：< 50ms/次
  - 成员列表查询：< 20ms/次
  - 两者相互独立，无锁竞争

## 🔍 测试策略

### 集成测试使用 SQLx Test

集成测试使用 `#[sqlx::test]` 宏，每个测试用例：

1. **自动创建测试数据库**
2. **运行所有迁移**（`migrations/` 目录）
3. **执行测试代码**
4. **自动清理数据**（事务回滚或删除数据库）

**优点**：
- 每个测试独立的数据库
- 测试间无干扰
- 无需手动清理数据

### 性能测试注意事项

1. **单独运行**：性能测试不应与其他测试并发运行
   ```bash
   cargo test --test performance_location -- --test-threads=1
   ```

2. **查看详细输出**：使用 `--nocapture` 查看性能指标
   ```bash
   cargo test --test performance_location -- --nocapture
   ```

3. **数据清理**：性能测试会插入大量数据，测试后会自动清理

## 🐛 故障排查

### 测试失败常见原因

1. **数据库连接失败**
   - 检查 PostgreSQL 是否运行
   - 验证 `DATABASE_URL` 配置
   - 确认用户权限

2. **PostGIS 扩展缺失**
   ```sql
   -- 手动安装扩展
   CREATE EXTENSION IF NOT EXISTS postgis;
   CREATE EXTENSION IF NOT EXISTS postgis_topology;
   ```

3. **Redis 连接失败**
   - 检查 Redis 是否运行
   - 验证 `REDIS_URL` 配置

4. **性能测试超时**
   - 检查是否创建了 GIST 索引
   ```sql
   -- 查看索引
   \d user_location_history
   ```

### 查看测试日志

```bash
# 启用 RUST_LOG
RUST_LOG=debug cargo test --test integration_auth -- --nocapture

# 仅查看错误
RUST_LOG=error cargo test
```

## 📝 编写新测试

### 集成测试模板

```rust
use sqlx::PgPool;

#[sqlx::test]
async fn test_your_feature(pool: PgPool) -> Result<(), Box<dyn std::error::Error>> {
    // 1. 准备测试数据
    let user_id = create_test_user(&pool).await?;
    
    // 2. 执行测试操作
    let result = your_function(&pool, user_id).await?;
    
    // 3. 验证结果
    assert_eq!(result, expected_value);
    
    Ok(())
}
```

### 性能测试模板

```rust
use std::time::Instant;

#[sqlx::test]
async fn test_performance(pool: PgPool) -> Result<(), Box<dyn std::error::Error>> {
    // 1. 插入测试数据
    insert_test_data(&pool, 10000).await?;
    
    // 2. 测量性能
    let start = Instant::now();
    for _ in 0..100 {
        perform_query(&pool).await?;
    }
    let duration = start.elapsed();
    
    // 3. 验证性能指标
    let avg_time = duration / 100;
    assert!(avg_time.as_millis() < 50, "查询过慢: {:?}", avg_time);
    
    Ok(())
}
```

## 🔗 相关资源

- [SQLx 文档](https://docs.rs/sqlx)
- [PostGIS 文档](https://postgis.net/docs/)
- [Cargo 测试指南](https://doc.rust-lang.org/cargo/commands/cargo-test.html)

---

**最后更新**：2025-11-05


















