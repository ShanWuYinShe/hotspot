/**
 * 测试运行入口
 * 运行所有测试用例
 */

import { TestRunner } from './TestFramework';
import { userTypeUtilTests } from './UserTypeUtil.test';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF14;
const TAG: string = 'TestMain';

/**
 * 运行所有测试
 */
export async function runAllTests(): Promise<void> {
  hilog.info(DOMAIN_NUMBER, TAG, '开始运行GeoTalk应用测试套件');
  
  const runner = new TestRunner();
  
  try {
    // 运行所有测试套件
    await runner.runSuites([
      userTypeUtilTests
      // 可以在这里添加更多测试套件
    ]);
    
    const results = runner.getResults();
    const passedCount = results.filter(r => r.passed).length;
    const totalCount = results.length;
    
    hilog.info(DOMAIN_NUMBER, TAG, `测试完成: ${passedCount}/${totalCount} 通过`);
    
    if (passedCount === totalCount) {
      hilog.info(DOMAIN_NUMBER, TAG, '🎉 所有测试都通过了！');
    } else {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ ${totalCount - passedCount} 个测试失败`);
    }
  } catch (error) {
    hilog.error(DOMAIN_NUMBER, TAG, `测试运行失败: ${JSON.stringify(error)}`);
  }
}

/**
 * 运行性能测试
 */
export async function runPerformanceTests(): Promise<void> {
  hilog.info(DOMAIN_NUMBER, TAG, '开始运行性能测试');
  
  // 这里可以添加性能测试用例
  // 例如：API响应时间测试、内存使用测试、缓存性能测试等
  
  hilog.info(DOMAIN_NUMBER, TAG, '性能测试完成');
}

/**
 * 运行集成测试
 */
export async function runIntegrationTests(): Promise<void> {
  hilog.info(DOMAIN_NUMBER, TAG, '开始运行集成测试');
  
  // 这里可以添加集成测试用例
  // 例如：API集成测试、WebSocket连接测试、权限系统测试等
  
  hilog.info(DOMAIN_NUMBER, TAG, '集成测试完成');
}

// 如果直接运行此文件，执行所有测试
if (typeof globalThis !== 'undefined') {
  runAllTests().then(() => {
    hilog.info(DOMAIN_NUMBER, TAG, '测试执行完成');
  }).catch((error) => {
    hilog.error(DOMAIN_NUMBER, TAG, `测试执行失败: ${error}`);
  });
}

