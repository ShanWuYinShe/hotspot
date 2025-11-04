-- 插入系统用户
-- 注释：
-- 216172782113783809 = 0x0300000000000001 (系统通知)
-- 216172782113783810 = 0x0300000000000002 (安全中心)
-- 216172782113783811 = 0x0300000000000003 (运营团队)

-- 步骤1: 插入系统用户（不包含位置信息）
INSERT INTO users (
    user_id, 
    nickname, 
    privacy_level, 
    location_sharing, 
    allow_follow,
    created_at
) VALUES 
    (216172782113783809, '系统通知', 'public', FALSE, FALSE, NOW()),
    (216172782113783810, '安全中心', 'public', FALSE, FALSE, NOW()),
    (216172782113783811, '运营团队', 'public', FALSE, FALSE, NOW())
ON CONFLICT (user_id) DO NOTHING;

-- 步骤2: 为系统用户插入默认位置（南天门）到位置历史表
-- 注意：location 字段由触发器自动生成，无需手动插入
INSERT INTO user_location_history (
    user_id,
    latitude,
    longitude,
    location_name
) VALUES 
    (216172782113783809, 35.154922, 112.682524, '南天门'),
    (216172782113783810, 35.154922, 112.682524, '南天门'),
    (216172782113783811, 35.154922, 112.682524, '南天门');




