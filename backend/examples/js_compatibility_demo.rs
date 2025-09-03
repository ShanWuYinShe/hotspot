/// JavaScript兼容性演示
/// 
/// 展示如何使用JsSafeI64类型确保前后端数据一致性

use backend::utils::JsSafeI64;
use serde_json;

fn main() {
    println!("🚀 JavaScript兼容性演示");
    println!("{}", "=".repeat(50));
    
    // 演示1: 基本序列化和反序列化
    println!("\n📦 1. 基本序列化和反序列化");
    let large_id = JsSafeI64::new(9007199254740992); // 2^53，JavaScript安全整数边界
    let json = serde_json::to_string(&large_id).unwrap();
    println!("原始值: {}", large_id.value());
    println!("序列化为JSON: {}", json);
    
    let deserialized: JsSafeI64 = serde_json::from_str(&json).unwrap();
    println!("反序列化后: {}", deserialized.value());
    println!("数据一致性: {}", large_id == deserialized);
    
    // 演示2: 超大数字处理
    println!("\n🔢 2. 超大数字处理");
    let very_large_id = JsSafeI64::new(i64::MAX);
    let json = serde_json::to_string(&very_large_id).unwrap();
    println!("最大i64值: {}", very_large_id.value());
    println!("序列化为JSON: {}", json);
    
    // 演示3: API响应示例
    println!("\n📡 3. API响应示例");
    
    #[derive(serde::Serialize)]
    struct UserResponse {
        user_id: JsSafeI64,
        nickname: String,
        follower_count: JsSafeI64,
    }
    
    let user = UserResponse {
        user_id: JsSafeI64::new(1234567890123456789),
        nickname: "测试用户".to_string(),
        follower_count: JsSafeI64::new(999999999999),
    };
    
    let json = serde_json::to_string_pretty(&user).unwrap();
    println!("用户响应JSON:");
    println!("{}", json);
    
    // 演示4: WebSocket消息示例
    println!("\n🔌 4. WebSocket消息示例");
    
    #[derive(serde::Serialize)]
    struct WebSocketMessage {
        message_type: String,
        sender_id: JsSafeI64,
        timestamp: JsSafeI64,
        content: String,
    }
    
    let ws_msg = WebSocketMessage {
        message_type: "chat".to_string(),
        sender_id: JsSafeI64::new(987654321098765432),
        timestamp: JsSafeI64::new(1703123456789),
        content: "Hello, World!".to_string(),
    };
    
    let json = serde_json::to_string_pretty(&ws_msg).unwrap();
    println!("WebSocket消息JSON:");
    println!("{}", json);
    
    // 演示5: 分页响应示例
    println!("\n📄 5. 分页响应示例");
    
    #[derive(serde::Serialize)]
    struct PaginationMeta {
        has_more: bool,
        next_cursor: Option<JsSafeI64>,
    }
    
    #[derive(serde::Serialize)]
    struct PaginatedResponse<T> {
        items: Vec<T>,
        pagination: PaginationMeta,
    }
    
    #[derive(serde::Serialize)]
    struct MessageItem {
        id: JsSafeI64,
        content: String,
        sender_id: JsSafeI64,
    }
    
    let messages = vec![
        MessageItem {
            id: JsSafeI64::new(1001),
            content: "第一条消息".to_string(),
            sender_id: JsSafeI64::new(123456789012345),
        },
        MessageItem {
            id: JsSafeI64::new(1002),
            content: "第二条消息".to_string(),
            sender_id: JsSafeI64::new(98765432109876),
        },
    ];
    
    let response = PaginatedResponse {
        items: messages,
        pagination: PaginationMeta {
            has_more: true,
            next_cursor: Some(JsSafeI64::new(1002)),
        },
    };
    
    let json = serde_json::to_string_pretty(&response).unwrap();
    println!("分页响应JSON:");
    println!("{}", json);
    
    // 演示6: 从字符串解析
    println!("\n🔄 6. 从字符串解析");
    let json_input = r#""9223372036854775807""#; // i64::MAX as string
    let parsed: JsSafeI64 = serde_json::from_str(json_input).unwrap();
    println!("输入JSON: {}", json_input);
    println!("解析结果: {}", parsed.value());
    
    // 演示7: 从数字解析（兼容性）
    println!("\n🔢 7. 从数字解析（兼容性）");
    let json_number = "123456789";
    let parsed_num: JsSafeI64 = serde_json::from_str(json_number).unwrap();
    println!("输入JSON数字: {}", json_number);
    println!("解析结果: {}", parsed_num.value());
    
    println!("\n✅ 所有演示完成！");
    println!("📝 关键要点:");
    println!("   • 所有64位整数都序列化为字符串");
    println!("   • 支持从字符串和数字反序列化");
    println!("   • 确保JavaScript前端能正确处理大整数");
    println!("   • 保持后端数据库的i64类型不变");
}
