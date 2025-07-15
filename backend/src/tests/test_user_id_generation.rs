#[cfg(test)]
mod user_id_generation_tests {
    use crate::utils::{UserIdPartition, UserType};

    #[tokio::test]
    async fn test_user_type_detection() {
        // 直接使用 UserIdPartition 的静态方法测试用户类型检测
        // 不需要创建 UserService 实例

        // 获取实际的范围
        let (regular_min, regular_max) = UserIdPartition::get_type_range(UserType::Regular);
        let (temp_min, temp_max) = UserIdPartition::get_type_range(UserType::Temporary);

        // 测试正式用户
        assert!(UserIdPartition::is_user_type(regular_min, UserType::Regular));
        assert!(UserIdPartition::is_user_type(regular_max, UserType::Regular));
        assert!(!UserIdPartition::is_user_type(regular_min, UserType::Temporary));

        // 测试临时用户
        assert!(UserIdPartition::is_user_type(temp_min, UserType::Temporary));
        assert!(UserIdPartition::is_user_type(temp_max, UserType::Temporary));
        assert!(!UserIdPartition::is_user_type(temp_min, UserType::Regular));
    }

    #[tokio::test]
    async fn test_user_id_generation_logic() {
        // 测试用户ID生成逻辑（不需要真实的数据库）
        
        // 获取实际的范围
        let (regular_min, regular_max) = UserIdPartition::get_type_range(UserType::Regular);
        let (temp_min, temp_max) = UserIdPartition::get_type_range(UserType::Temporary);
        
        // 测试正式用户ID生成
        let regular_id_1 = UserIdPartition::generate_user_id(UserType::Regular, 1).unwrap();
        let regular_id_2 = UserIdPartition::generate_user_id(UserType::Regular, 2).unwrap();
        
        assert!(regular_id_1 >= regular_min && regular_id_1 <= regular_max, "正式用户ID应该在正确范围内");
        assert!(regular_id_2 >= regular_min && regular_id_2 <= regular_max, "正式用户ID应该在正确范围内");
        assert!(regular_id_2 > regular_id_1, "正式用户ID应该递增");
        
        // 测试临时用户ID生成
        let temp_id_1 = UserIdPartition::generate_user_id(UserType::Temporary, 1).unwrap();
        let temp_id_2 = UserIdPartition::generate_user_id(UserType::Temporary, 2).unwrap();
        
        assert!(temp_id_1 >= temp_min && temp_id_1 <= temp_max, "临时用户ID应该在正确范围内");
        assert!(temp_id_2 >= temp_min && temp_id_2 <= temp_max, "临时用户ID应该在正确范围内");
        assert!(temp_id_2 > temp_id_1, "临时用户ID应该递增");
        
        // 测试管理员用户ID生成
        let admin_id = UserIdPartition::generate_user_id(UserType::Admin, 1).unwrap();
        let (admin_min, admin_max) = UserIdPartition::get_type_range(UserType::Admin);
        assert!(admin_id >= admin_min && admin_id <= admin_max, "管理员ID应该在正确范围内");
        
        println!(
            "正式用户ID: {} -> {}, 临时用户ID: {} -> {}, 管理员ID: {}",
            regular_id_1, regular_id_2, temp_id_1, temp_id_2, admin_id
        );
    }

    #[test]
    fn test_user_type_ranges() {
        // 测试用户类型范围
        let regular_range = UserIdPartition::get_type_range(UserType::Regular);
        let temp_range = UserIdPartition::get_type_range(UserType::Temporary);
        let admin_range = UserIdPartition::get_type_range(UserType::Admin);
        let system_range = UserIdPartition::get_type_range(UserType::System);
        let test_range = UserIdPartition::get_type_range(UserType::Test);
        
        // 验证范围不重叠
        assert!(regular_range.1 < temp_range.0, "正式用户和临时用户范围不应重叠");
        assert!(temp_range.1 < admin_range.0, "临时用户和管理员范围不应重叠");
        assert!(admin_range.1 < system_range.0, "管理员和系统用户范围不应重叠");
        assert!(system_range.1 < test_range.0, "系统用户和测试用户范围不应重叠");
        
        // 验证范围大小
        let regular_size = regular_range.1 - regular_range.0 + 1;
        let temp_size = temp_range.1 - temp_range.0 + 1;
        
        assert!(regular_size > 0, "正式用户范围应该大于0");
        assert!(temp_size > 0, "临时用户范围应该大于0");
        
        println!("正式用户范围: {:?}, 临时用户范围: {:?}", regular_range, temp_range);
        println!("管理员范围: {:?}, 系统用户范围: {:?}, 测试用户范围: {:?}", admin_range, system_range, test_range);
    }

    #[test]
    fn test_sequence_generation() {
        // 测试序列号生成逻辑
        
        // 测试从起始位置开始的序列号
        let start_regular = UserIdPartition::get_type_range(UserType::Regular).0;
        let next_seq_1 = UserIdPartition::next_sequence_for_type(UserType::Regular, start_regular).unwrap();
        assert_eq!(next_seq_1, 1, "第一个序列号应该是1");
        
        let start_temp = UserIdPartition::get_type_range(UserType::Temporary).0;
        let next_seq_2 = UserIdPartition::next_sequence_for_type(UserType::Temporary, start_temp).unwrap();
        assert_eq!(next_seq_2, 1, "第一个序列号应该是1");
        
        // 测试从已有ID继续的序列号
        let current_id = start_regular + 5; // 假设已经有5个用户
        let next_seq_3 = UserIdPartition::next_sequence_for_type(UserType::Regular, current_id).unwrap();
        assert_eq!(next_seq_3, 6, "下一个序列号应该是6");
    }

    #[test]
    fn test_id_parsing() {
        // 测试ID解析功能
        
        // 测试正式用户ID解析
        let regular_id = UserIdPartition::generate_user_id(UserType::Regular, 12345).unwrap();
        let (parsed_type, parsed_sequence) = UserIdPartition::parse_user_id(regular_id).unwrap();
        assert_eq!(parsed_type, UserType::Regular, "应该正确解析为正式用户类型");
        assert_eq!(parsed_sequence, 12345, "应该正确解析序列号");
        
        // 测试临时用户ID解析
        let temp_id = UserIdPartition::generate_user_id(UserType::Temporary, 67890).unwrap();
        let (parsed_type_2, parsed_sequence_2) = UserIdPartition::parse_user_id(temp_id).unwrap();
        assert_eq!(parsed_type_2, UserType::Temporary, "应该正确解析为临时用户类型");
        assert_eq!(parsed_sequence_2, 67890, "应该正确解析序列号");
    }
}
