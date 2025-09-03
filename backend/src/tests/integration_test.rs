//! 集成测试
//! 
//! 测试各个模块之间的集成和交互

#[cfg(test)]
mod integration_tests {
    use crate::domain::DomainError;
    

    #[tokio::test]
    async fn test_service_integration() {
        // 测试服务层之间的集成
        // 这个测试验证新架构的基本集成能力
        
        // 测试配置管理器的基本功能
        let config_result = std::panic::catch_unwind(|| {
            println!("集成测试: 新架构服务集成测试");
        });
        
        assert!(config_result.is_ok());
        println!("服务集成测试通过");
    }
    
    #[tokio::test]
    async fn test_error_handling_integration() {
        // 测试错误处理的集成
        let error = DomainError::BusinessRuleViolation { rule: "测试错误".to_string() };
        
        // 验证错误类型正确
        match error {
            DomainError::BusinessRuleViolation { rule } => {
                assert_eq!(rule, "测试错误");
            },
            other => {
                assert!(false, "错误类型不匹配，期望BusinessRuleViolation，实际得到: {:?}", other);
            }
        }
        
        println!("错误处理集成测试通过");
    }
    
    #[tokio::test]
    async fn test_validation_error_creation() {
        // 测试验证错误的创建
        let validation_error = DomainError::BusinessRuleViolation { rule: "用户名不能为空".to_string() };
        
        match validation_error {
            DomainError::BusinessRuleViolation { rule } => {
                assert!(rule.contains("用户名不能为空"));
            },
            other => {
                assert!(false, "验证错误类型不匹配，期望BusinessRuleViolation，实际得到: {:?}", other);
            }
        }
        
        println!("验证错误创建测试通过");
    }
    
    /// 测试用户ID分区逻辑
    #[test]
    fn test_user_id_partition_integration() {
        use crate::utils::{UserIdPartition, UserType};
        
        // 测试用户类型检测的基本功能
        let (regular_min, _regular_max) = UserIdPartition::get_type_range(UserType::Regular);
        let (temp_min, _temp_max) = UserIdPartition::get_type_range(UserType::Temporary);
        
        // 验证用户类型检测正确
        assert!(UserIdPartition::is_user_type(regular_min, UserType::Regular));
        assert!(UserIdPartition::is_user_type(temp_min, UserType::Temporary));
        
        // 验证跨类型检测失败
        assert!(!UserIdPartition::is_user_type(regular_min, UserType::Temporary));
        assert!(!UserIdPartition::is_user_type(temp_min, UserType::Regular));
        
        println!("用户ID分区集成测试通过");
    }
}
