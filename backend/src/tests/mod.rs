// 测试模块
// 
// 所有测试已适配新架构并启用

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
