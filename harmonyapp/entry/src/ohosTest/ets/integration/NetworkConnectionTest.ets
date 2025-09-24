/**
 * 网络连接测试
 * 用于验证后端服务器连接和API功能
 */

import { describe, beforeAll, beforeEach, afterEach, afterAll, it, expect } from '@ohos/hypium';
import { HttpUtil } from '../../../main/ets/common/network/HttpUtil';
import { ApiClient } from '../../../main/ets/common/api/ApiClient';
import { API_BASE_URL } from '../../../main/ets/constants/AppConstants';

export default function NetworkConnectionTest() {
  describe('网络连接测试', () => {
    
    beforeAll(() => {
      console.info('开始网络连接测试套件');
    });

    afterAll(() => {
      console.info('网络连接测试套件完成');
    });

    it('应该能够连接到服务器', async () => {
      try {
        const result = await HttpUtil.testConnection();
        expect(result.success).assertTrue();
        expect(result.responseTime).assertLarger(0);
        console.info(`服务器连接成功，响应时间: ${result.responseTime}ms`);
      } catch (error) {
        console.error('服务器连接测试失败:', error);
        expect().assertFail(`服务器连接失败: ${error}`);
      }
    });

    it('健康检查端点应该正常工作', async () => {
      try {
        const result = await HttpUtil.get<any>('/health');
        expect(result).assertInstanceOf('object');
        console.info('健康检查端点正常');
      } catch (error) {
        console.error('健康检查失败:', error);
        expect().assertFail(`健康检查失败: ${error}`);
      }
    });

    it('应该能够创建临时用户', async () => {
      try {
        const testNickname = `TestUser_${Date.now()}`;
        const result = await ApiClient.createTempUser(testNickname);
        
        expect(result.user_id).assertInstanceOf('string');
        expect(result.nickname).assertEqual(testNickname);
        expect(result.is_temp).assertTrue();
        expect(result.access_token).assertInstanceOf('string');
        
        console.info(`临时用户创建成功: ${result.user_id}`);
      } catch (error) {
        console.error('创建临时用户失败:', error);
        expect().assertFail(`创建临时用户失败: ${error}`);
      }
    });

    it('应该能够搜索附近用户', async () => {
      try {
        // 使用北京坐标进行测试
        const latitude = 39.9042;
        const longitude = 116.4074;
        const radius = 1000;
        
        const result = await ApiClient.searchNearbyUsers(latitude, longitude, radius);
        
        expect(result.items).assertInstanceOf('Array');
        expect(result.has_more).assertInstanceOf('boolean');
        
        console.info(`搜索到 ${result.items.length} 个附近用户`);
      } catch (error) {
        console.error('搜索附近用户失败:', error);
        expect().assertFail(`搜索附近用户失败: ${error}`);
      }
    });

    it('应该能够搜索附近群组', async () => {
      try {
        // 使用北京坐标进行测试
        const latitude = 39.9042;
        const longitude = 116.4074;
        const radius = 5000;
        
        const result = await ApiClient.searchNearbyGroups(latitude, longitude, radius);
        
        expect(result.items).assertInstanceOf('Array');
        expect(result.has_more).assertInstanceOf('boolean');
        
        console.info(`搜索到 ${result.items.length} 个附近群组`);
      } catch (error) {
        console.error('搜索附近群组失败:', error);
        expect().assertFail(`搜索附近群组失败: ${error}`);
      }
    });

    it('应该能够搜索附近生活瞬间', async () => {
      try {
        // 使用北京坐标进行测试
        const latitude = 39.9042;
        const longitude = 116.4074;
        const radius = 2000;
        
        const result = await ApiClient.searchNearbyMoments(latitude, longitude, radius);
        
        expect(result.items).assertInstanceOf('Array');
        expect(result.has_more).assertInstanceOf('boolean');
        
        console.info(`搜索到 ${result.items.length} 个附近生活瞬间`);
      } catch (error) {
        console.error('搜索附近生活瞬间失败:', error);
        expect().assertFail(`搜索附近生活瞬间失败: ${error}`);
      }
    });

    it('服务器地址应该正确配置', () => {
      const debugInfo = HttpUtil.getDebugInfo();
      expect(debugInfo.baseUrl).assertEqual('http://192.168.1.5:3000');
      console.info(`当前服务器地址: ${debugInfo.baseUrl}`);
    });

    it('HTTP请求应该有合理的超时设置', async () => {
      const startTime = Date.now();
      
      try {
        // 测试一个可能不存在的端点，验证超时机制
        await HttpUtil.get('/non-existent-endpoint');
      } catch (error) {
        const duration = Date.now() - startTime;
        // 应该在合理时间内超时（不超过35秒，因为我们设置了30秒超时）
        expect(duration).assertLess(35000);
        console.info(`请求超时机制正常，耗时: ${duration}ms`);
      }
    });
  });
}
