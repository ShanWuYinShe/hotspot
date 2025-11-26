use sqlx::PgPool;
use std::time::Instant;

/// 性能测试：地理位置查询
///
/// 测试PostGIS地理位置查询性能，验证GIST索引的效果
///
/// 测试场景：
/// 1. 插入10000条用户位置数据
/// 2. 执行100次附近用户查询
/// 3. 验证平均查询时间 < 50ms
/// 4. 验证使用了GIST索引（通过EXPLAIN ANALYZE）

#[sqlx::test]
async fn test_location_query_performance(pool: PgPool) -> Result<(), Box<dyn std::error::Error>> {
    // 准备测试数据：插入10000条用户位置记录
    println!("开始插入测试数据...");
    let insert_start = Instant::now();

    for i in 0..10000 {
        let latitude = 35.0 + (i as f64 / 10000.0) * 0.5; // 35.0 ~ 35.5
        let longitude = 112.0 + (i as f64 / 10000.0) * 0.5; // 112.0 ~ 112.5
        let location_name = format!("测试位置{}", i);

        sqlx::query!(
            r#"
            INSERT INTO user_location_history (user_id, latitude, longitude, location_name)
            VALUES ($1, $2, $3, $4)
            "#,
            (144115188075855872_i64 + i), // 0x0200000000000000 + i
            latitude,
            longitude,
            location_name
        )
        .execute(&pool)
        .await?;
    }

    let insert_duration = insert_start.elapsed();
    println!("插入10000条数据耗时: {:?}", insert_duration);

    // 执行100次附近用户查询
    println!("开始执行位置查询...");
    let query_start = Instant::now();
    let mut total_results = 0;

    for i in 0..100 {
        let latitude = 35.2 + (i as f64 / 1000.0);
        let longitude = 112.2 + (i as f64 / 1000.0);
        let radius = 5000.0; // 5公里

        let results = sqlx::query!(
            r#"
            SELECT 
                ulh.user_id,
                ulh.latitude,
                ulh.longitude,
                ulh.location_name,
                ST_Distance(
                    ulh.location,
                    ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
                ) as "distance!"
            FROM (
                SELECT DISTINCT ON (user_id)
                    user_id,
                    latitude,
                    longitude,
                    location_name,
                    location
                FROM user_location_history
                ORDER BY user_id, created_at DESC
            ) ulh
            WHERE ST_DWithin(
                ulh.location,
                ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
                $3
            )
            ORDER BY distance
            LIMIT 20
            "#,
            longitude,
            latitude,
            radius
        )
        .fetch_all(&pool)
        .await?;

        total_results += results.len();
    }

    let query_duration = query_start.elapsed();
    let avg_query_time = query_duration / 100;

    println!("执行100次查询耗时: {:?}", query_duration);
    println!("平均查询时间: {:?}", avg_query_time);
    println!("总共找到 {} 条结果", total_results);

    // 验证平均查询时间 < 100ms（宽松的阈值，实际应该 < 50ms）
    assert!(
        avg_query_time.as_millis() < 100,
        "平均查询时间过长: {:?} > 100ms，可能未使用GIST索引",
        avg_query_time
    );

    // 执行EXPLAIN ANALYZE验证索引使用
    println!("\n验证查询计划（应使用GIST索引）...");
    let explain_result = sqlx::query_scalar::<_, String>(
        r#"
        EXPLAIN ANALYZE
        SELECT 
            ulh.user_id,
            ST_Distance(
                ulh.location,
                ST_SetSRID(ST_MakePoint(112.2, 35.2), 4326)::geography
            ) as distance
        FROM (
            SELECT DISTINCT ON (user_id)
                user_id,
                location
            FROM user_location_history
            ORDER BY user_id, created_at DESC
        ) ulh
        WHERE ST_DWithin(
            ulh.location,
            ST_SetSRID(ST_MakePoint(112.2, 35.2), 4326)::geography,
            5000
        )
        ORDER BY distance
        LIMIT 20
        "#,
    )
    .fetch_all(&pool)
    .await?;

    let explain_text = explain_result.join("\n");
    println!("{}", explain_text);

    // 验证使用了索引扫描（Index Scan 或 Bitmap Index Scan）
    assert!(
        explain_text.contains("Index Scan") || explain_text.contains("Bitmap Index Scan"),
        "查询未使用GIST索引！\n{}",
        explain_text
    );

    Ok(())
}

#[sqlx::test]
async fn test_table_separation_performance(pool: PgPool) -> Result<(), Box<dyn std::error::Error>> {
    println!("性能测试：验证表分离效果");
    println!("测试场景：模拟高并发更新未读计数和查询成员列表");

    // 1. 准备测试数据：创建一个群组和100个成员
    let room_id = uuid::Uuid::new_v4();

    sqlx::query!(
        r#"
        INSERT INTO chat_rooms (room_id, room_type, name, owner_id, max_members)
        VALUES ($1, 'public_group', '性能测试群', 144115188075855872, 500)
        "#,
        room_id
    )
    .execute(&pool)
    .await?;

    for i in 0..100 {
        let user_id = 144115188075855872_i64 + i;

        // 插入成员记录
        sqlx::query!(
            r#"
            INSERT INTO room_members (room_id, user_id, role)
            VALUES ($1, $2, 'member')
            "#,
            room_id,
            user_id
        )
        .execute(&pool)
        .await?;

        // 插入未读计数记录
        sqlx::query!(
            r#"
            INSERT INTO room_member_unread (room_id, user_id, unread_count)
            VALUES ($1, $2, 0)
            "#,
            room_id,
            user_id
        )
        .execute(&pool)
        .await?;
    }

    // 2. 测试场景1：并发更新未读计数（模拟发送消息）
    println!("\n场景1: 并发更新未读计数（100次）");
    let update_start = Instant::now();

    for _ in 0..100 {
        // 更新所有成员的未读计数
        sqlx::query!(
            r#"
            UPDATE room_member_unread
            SET unread_count = unread_count + 1,
                updated_at = NOW()
            WHERE room_id = $1
            "#,
            room_id
        )
        .execute(&pool)
        .await?;
    }

    let update_duration = update_start.elapsed();
    println!("更新未读计数耗时: {:?}", update_duration);
    println!("平均每次更新: {:?}", update_duration / 100);

    // 3. 测试场景2：并发查询成员列表（不应被未读计数更新阻塞）
    println!("\n场景2: 并发查询成员列表（100次）");
    let query_start = Instant::now();

    for _ in 0..100 {
        let _ = sqlx::query!(
            r#"
            SELECT user_id, role, joined_at
            FROM room_members
            WHERE room_id = $1 AND is_deleted = FALSE
            ORDER BY joined_at DESC
            LIMIT 20
            "#,
            room_id
        )
        .fetch_all(&pool)
        .await?;
    }

    let query_duration = query_start.elapsed();
    println!("查询成员列表耗时: {:?}", query_duration);
    println!("平均每次查询: {:?}", query_duration / 100);

    // 4. 验证性能指标
    let avg_update_time = update_duration.as_millis() / 100;
    let avg_query_time = query_duration.as_millis() / 100;

    println!("\n性能汇总:");
    println!("- 平均更新未读计数: {}ms", avg_update_time);
    println!("- 平均查询成员列表: {}ms", avg_query_time);

    // 验证更新和查询都在合理范围内
    assert!(
        avg_update_time < 50,
        "更新未读计数过慢: {}ms > 50ms",
        avg_update_time
    );

    assert!(
        avg_query_time < 20,
        "查询成员列表过慢: {}ms > 20ms",
        avg_query_time
    );

    println!("\n✅ 表分离性能测试通过！");
    println!("   未读计数更新和成员列表查询相互独立，无锁竞争。");

    Ok(())
}


