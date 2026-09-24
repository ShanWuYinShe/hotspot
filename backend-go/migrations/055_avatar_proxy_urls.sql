-- 055: 头像 URL 化——users/groups 的 avatar_url 由裸 file_id 统一为稳定代理路径。
--
-- 缺陷（线上实测）：users.UpdateMyProfile 与 groups Create/Update 此前把
-- avatar_file_id（形如 file_<hex>）直接存进 avatar_url，客户端 <img src>/
-- Image() 以其为相对路径请求落到 SPA 兜底 HTML，全站头像实际全部坏图
-- （首字母兜底）。修复后新写入统一为 /public/v1/media/avatars/<file_id>
-- （匿名 302 到签名读 URL，见 media service_avatar.go），本迁移把存量
-- 裸 file_id 行一次性修补为同形态。
--
-- 幂等：谓词 avatar_url LIKE 'file_%' 天然自检——转换后的值以 /public/ 开头，
-- 不再命中；重复执行为空操作。

UPDATE users
SET avatar_url = '/public/v1/media/avatars/' || avatar_url
WHERE avatar_url LIKE 'file_%';

UPDATE groups
SET avatar_url = '/public/v1/media/avatars/' || avatar_url
WHERE avatar_url LIKE 'file_%';
