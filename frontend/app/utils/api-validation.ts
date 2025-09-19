/**
 * API 参数验证工具函数
 * 用于在发送请求前验证参数的有效性
 */

/**
 * 验证群组ID格式
 */
export const validateGroupId = (groupId: string): string => {
  if (!groupId || typeof groupId !== 'string') {
    throw new Error('群组ID不能为空');
  }
  
  const trimmed = groupId.trim();
  if (trimmed.length === 0) {
    throw new Error('群组ID不能为空');
  }
  
  // 简单的UUID格式验证
  const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
  if (!uuidRegex.test(trimmed)) {
    throw new Error('群组ID格式不正确');
  }
  
  return trimmed;
};

/**
 * 验证用户ID格式
 */
export const validateUserId = (userId: string): string => {
  if (!userId || typeof userId !== 'string') {
    throw new Error('用户ID不能为空');
  }
  
  const trimmed = userId.trim();
  if (trimmed.length === 0) {
    throw new Error('用户ID不能为空');
  }
  
  return trimmed;
};

/**
 * 验证文本内容长度和格式
 */
export const validateContent = (
  content: string, 
  minLength: number, 
  maxLength: number, 
  fieldName: string = '内容'
): string => {
  if (!content || typeof content !== 'string') {
    throw new Error(`${fieldName}不能为空`);
  }
  
  const trimmed = content.trim();
  if (trimmed.length < minLength) {
    throw new Error(`${fieldName}长度不能少于${minLength}个字符`);
  }
  
  if (trimmed.length > maxLength) {
    throw new Error(`${fieldName}长度不能超过${maxLength}个字符`);
  }
  
  return trimmed;
};

/**
 * 验证地理坐标
 */
export const validateCoordinates = (latitude: number, longitude: number): void => {
  if (typeof latitude !== 'number' || isNaN(latitude)) {
    throw new Error('纬度必须是有效数字');
  }
  
  if (typeof longitude !== 'number' || isNaN(longitude)) {
    throw new Error('经度必须是有效数字');
  }
  
  if (latitude < -90 || latitude > 90) {
    throw new Error('纬度必须在-90到90之间');
  }
  
  if (longitude < -180 || longitude > 180) {
    throw new Error('经度必须在-180到180之间');
  }
};

/**
 * 验证搜索半径
 */
export const validateRadius = (radius: number): number => {
  if (typeof radius !== 'number' || isNaN(radius)) {
    throw new Error('搜索半径必须是有效数字');
  }
  
  if (radius <= 0) {
    throw new Error('搜索半径必须大于0');
  }
  
  if (radius > 50000) { // 50km 限制
    throw new Error('搜索半径不能超过50公里');
  }
  
  return radius;
};

/**
 * 验证分页参数
 * cursor 应为 string 类型，符合游标分页规范
 */
export const validatePaginationParams = (params: {
  limit?: number;
  cursor?: string;
}): { limit: number; cursor?: string } => {
  const result: { limit: number; cursor?: string } = {
    limit: params.limit || 20
  };
  
  if (params.limit !== undefined) {
    if (typeof params.limit !== 'number' || isNaN(params.limit)) {
      throw new Error('分页限制必须是有效数字');
    }
    
    if (params.limit < 1) {
      throw new Error('分页限制必须大于0');
    }
    
    if (params.limit > 100) {
      throw new Error('分页限制不能超过100');
    }
    
    result.limit = params.limit;
  }
  
  if (params.cursor !== undefined) {
    if (typeof params.cursor === 'string') {
      if (params.cursor.trim() === '') {
        throw new Error('分页游标不能为空字符串');
      }
      result.cursor = params.cursor.trim();
    } else {
      throw new Error('分页游标必须是字符串类型');
    }
  }
  
  return result;
};

/**
 * 验证关键词搜索
 */
export const validateSearchKeyword = (keyword: string): string => {
  if (!keyword || typeof keyword !== 'string') {
    throw new Error('搜索关键词不能为空');
  }
  
  const trimmed = keyword.trim();
  if (trimmed.length === 0) {
    throw new Error('搜索关键词不能为空');
  }
  
  if (trimmed.length < 1) {
    throw new Error('搜索关键词长度不能少于1个字符');
  }
  
  if (trimmed.length > 50) {
    throw new Error('搜索关键词长度不能超过50个字符');
  }
  
  return trimmed;
};

/**
 * 验证登录ID（邮箱或手机号）
 */
export const validateLoginId = (loginId: string): string => {
  if (!loginId || typeof loginId !== 'string') {
    throw new Error('登录ID不能为空');
  }
  
  const trimmed = loginId.trim();
  if (trimmed.length === 0) {
    throw new Error('登录ID不能为空');
  }
  
  if (trimmed.length < 1 || trimmed.length > 100) {
    throw new Error('登录ID长度必须在1-100个字符之间');
  }
  
  // 简单的邮箱格式验证
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  // 简单的手机号格式验证（中国手机号）
  const phoneRegex = /^1[3-9]\d{9}$/;
  
  if (!emailRegex.test(trimmed) && !phoneRegex.test(trimmed)) {
    throw new Error('请输入有效的邮箱地址或手机号');
  }
  
  return trimmed;
};

/**
 * 验证密码
 */
export const validatePassword = (password: string): string => {
  if (!password || typeof password !== 'string') {
    throw new Error('密码不能为空');
  }
  
  if (password.length < 8 || password.length > 64) {
    throw new Error('密码长度必须在8-64个字符之间');
  }
  
  return password;
};

/**
 * 验证昵称
 */
export const validateNickname = (nickname: string): string => {
  if (!nickname || typeof nickname !== 'string') {
    throw new Error('昵称不能为空');
  }
  
  const trimmed = nickname.trim();
  if (trimmed.length === 0) {
    throw new Error('昵称不能为空');
  }
  
  if (trimmed.length < 1 || trimmed.length > 32) {
    throw new Error('昵称长度必须在1-32个字符之间');
  }
  
  return trimmed;
};