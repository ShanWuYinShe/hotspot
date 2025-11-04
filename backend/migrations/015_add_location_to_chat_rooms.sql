-- 给 chat_rooms 表添加位置字段（用于群组位置）
ALTER TABLE chat_rooms
ADD COLUMN IF NOT EXISTS latitude DOUBLE PRECISION,
ADD COLUMN IF NOT EXISTS longitude DOUBLE PRECISION,
ADD COLUMN IF NOT EXISTS location_name VARCHAR(255);

-- 为位置字段创建索引（用于地理搜索）
CREATE INDEX IF NOT EXISTS idx_chat_rooms_location 
ON chat_rooms (latitude, longitude) 
WHERE latitude IS NOT NULL AND longitude IS NOT NULL;

-- 更新现有记录的位置（从 room_location_history 获取最新位置）
UPDATE chat_rooms cr
SET 
    latitude = rlh.latitude,
    longitude = rlh.longitude,
    location_name = rlh.location_name
FROM (
    SELECT DISTINCT ON (room_id) 
        room_id, latitude, longitude, location_name
    FROM room_location_history
    ORDER BY room_id, created_at DESC
) rlh
WHERE cr.room_id = rlh.room_id
  AND cr.room_type IN ('public_group'::room_type, 'private_group'::room_type)
  AND cr.latitude IS NULL;






