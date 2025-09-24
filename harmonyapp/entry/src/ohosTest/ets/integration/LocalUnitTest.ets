/**
 * 本地单元测试
 * 基础功能测试示例
 */

import { describe, beforeAll, beforeEach, afterEach, afterAll, it, expect } from '@ohos/hypium';

export default function LocalUnitTest() {
  describe('本地单元测试', () => {
    // 测试套件设置
    beforeAll(() => {
      console.info('开始本地单元测试套件');
    });

    beforeEach(() => {
      // 每个测试用例开始前的设置
    });

    afterEach(() => {
      // 每个测试用例结束后的清理
    });

    afterAll(() => {
      console.info('本地单元测试套件完成');
    });

    it('字符串包含测试', 0, () => {
      let a = 'abc';
      let b = 'b';
      // 测试字符串包含关系
      expect(a).assertContain(b);
      expect(a).assertEqual(a);
    });

    it('数组操作测试', 0, () => {
      const testArray = [1, 2, 3, 4, 5];
      
      // 测试数组长度
      expect(testArray.length).assertEqual(5);
      
      // 测试数组包含元素
      expect(testArray.includes(3)).assertTrue();
      expect(testArray.includes(6)).assertFalse();
      
      // 测试数组操作
      const doubled = testArray.map(x => x * 2);
      expect(doubled).assertEqual([2, 4, 6, 8, 10]);
    });

    it('对象属性测试', 0, () => {
      const testObj = {
        name: 'test',
        value: 42,
        active: true
      };
      
      // 测试对象属性
      expect(testObj.name).assertEqual('test');
      expect(testObj.value).assertEqual(42);
      expect(testObj.active).assertTrue();
      
      // 测试对象类型
      expect(typeof testObj.name).assertEqual('string');
      expect(typeof testObj.value).assertEqual('number');
      expect(typeof testObj.active).assertEqual('boolean');
    });

    it('异步操作测试', 0, async () => {
      // 测试Promise
      const result = await new Promise<string>((resolve) => {
        setTimeout(() => resolve('async result'), 10);
      });
      
      expect(result).assertEqual('async result');
    });

    it('错误处理测试', 0, () => {
      // 测试异常抛出
      expect(() => {
        throw new Error('test error');
      }).assertThrow();
      
      // 测试正常执行不抛出异常
      expect(() => {
        return 'no error';
      }).assertNotThrow();
    });

    it('数值比较测试', 0, () => {
      const num1 = 10;
      const num2 = 20;
      const num3 = 10;
      
      // 测试数值比较
      expect(num1).assertLess(num2);
      expect(num2).assertLarger(num1);
      expect(num1).assertEqual(num3);
      expect(num1).assertNotEqual(num2);
    });

    it('类型检查测试', 0, () => {
      const str = 'hello';
      const num = 123;
      const bool = true;
      const arr = [1, 2, 3];
      const obj = { key: 'value' };
      
      // 测试类型检查
      expect(str).assertInstanceOf('string');
      expect(num).assertInstanceOf('number');
      expect(bool).assertInstanceOf('boolean');
      expect(arr).assertInstanceOf('Array');
      expect(obj).assertInstanceOf('object');
    });
  });
}
