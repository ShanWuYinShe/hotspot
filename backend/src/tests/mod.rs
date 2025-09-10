/// HotSpot 测试模块
/// 
/// 功能职责：
/// - 提供完整的测试套件和测试基础设施
/// - 实现单元测试、集成测试、端到端测试
/// - 提供测试数据的创建、清理和管理功能
/// - 实现性能测试和压力测试的基础设施
/// - 支持测试环境的隔离和并行执行
/// 
/// 测试类型：
/// - 单元测试：针对单个函数和组件的独立测试
/// - 集成测试：模块间协作和数据流的测试
/// - 架构测试：验证系统架构和设计原则的遵守
/// - 性能测试：高并发和大数据量的性能验证
/// - API 测试：完整的 HTTP API 接口测试覆盖
/// 
/// 测试工具：
/// - 数据库测试工具：测试数据库的创建和清理
/// - Mock 服务：外部依赖的模拟和打桩
/// - 测试工具函数：通用的测试数据生成和验证
/// - 性能监控：测试执行时间和资源使用的监控
/// 
/// 测试策略：
/// - 测试驱动开发：先写测试再实现功能
/// - 测试金字塔：单元测试 > 集成测试 > E2E 测试
/// - 测试隔离：每个测试用例独立运行，不相互影响
/// - 数据清理：测试后自动清理测试数据和资源
/// 
/// 所有测试已适配新架构并启用
// 核心测试模块
pub mod admin_controller_test;        // ✅ 管理员控制器测试
pub mod test_user_id_generation;     // ✅ 用户ID生成测试
pub mod basic_integration_test;      // ✅ 基础集成测试
pub mod architecture_integration_test; // ✅ 新架构集成测试

// 集成测试模块
pub mod ports_integration_test;        // ✅ 端口架构集成测试
pub mod integration_test;              // ✅ 系统集成测试

// 功能测试模块
pub mod geolocation_test;              // ✅ 地理位置计算测试
pub mod group_permission_test;         // ✅ 群组权限管理测试

// 性能测试模块
pub mod websocket_performance_test;    // ✅ WebSocket性能和压力测试
