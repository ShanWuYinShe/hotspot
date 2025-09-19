# 归档迁移文件

## 📁 目录说明

此目录包含已归档的旧版迁移文件，**不建议在生产环境中使用**。

### 🗂️ legacy_migrations/ - 旧版迁移文件

这些文件是项目早期的迁移文件，存在以下问题：

#### ❌ 已知问题
- **架构不统一**：消息系统使用过时的 `conversations` 架构
- **命名不规范**：不符合 `XxxRequest/XxxResponse` 命名规范
- **用户ID设计**：不支持用户类型编码规范
- **缺失功能**：缺少用户关系管理、内容安全、通知系统等功能
- **性能问题**：索引设计不够优化，缺少分区表支持

#### 📋 文件列表

| 文件名 | 说明 | 状态 |
|--------|------|------|
| `00_foundation.sql` | 基础设施和扩展 | ❌ 已废弃 |
| `01_users_core.sql` | 旧版用户表 | ❌ 不符合用户ID编码规范 |
| `02_groups_core.sql` | 旧版群组表 | ❌ 缺少统一消息架构支持 |
| `03_messages_core.sql` | 旧版消息表 | ❌ 使用过时的conversations架构 |
| `04_social_features.sql` | 旧版社交功能 | ❌ 功能不完整 |
| `05_system_performance.sql` | 旧版性能优化 | ❌ 索引设计过时 |
| `06_user_id_encoding_fix.sql` | 用户ID编码修复 | ✅ 已集成到新架构 |
| `07_soft_delete_triggers.sql` | 软删除触发器 | ✅ 已集成到新架构 |
| `08_unified_message_architecture.sql` | 统一消息架构 | ✅ 已集成到新架构 |

### ⚠️ 重要说明

1. **不要使用这些文件**：它们仅作为历史参考保留
2. **数据迁移**：如果需要从旧架构迁移数据，需要编写专门的迁移脚本
3. **学习参考**：可以作为架构演进的历史记录来学习

### 🚀 推荐做法

使用项目根目录下的 `01_complete_schema.sql` 进行全新部署：

```bash
# 推荐的部署方式
psql -h localhost -U postgres -d hotspot -f migrations/01_complete_schema.sql
```

## 📚 相关文档

- 查看 `../README.md` 了解当前推荐的迁移策略
- 查看 `../../.cursor/rules/backend_doc.mdc` 了解完整的数据库设计需求
