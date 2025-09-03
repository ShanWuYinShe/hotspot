/// 全局测试清理
/// 
/// 在所有测试结束后执行的清理操作

import { chromium, FullConfig } from '@playwright/test'
import fs from 'fs'
import path from 'path'

async function globalTeardown(config: FullConfig) {
  console.log('🧹 开始E2E测试全局清理...')
  
  const baseURL = config.projects[0].use.baseURL || 'http://localhost:3000'
  
  // 启动浏览器进行清理操作
  const browser = await chromium.launch()
  const page = await browser.newPage()
  
  try {
    // 连接到应用
    await page.goto(baseURL, { 
      waitUntil: 'networkidle',
      timeout: 10000 
    })
    
    // 清理测试数据
    console.log('🗑️ 清理测试数据...')
    await page.evaluate(() => {
      // 清理本地存储
      localStorage.clear()
      sessionStorage.clear()
      
      // 清理Cookie
      document.cookie.split(";").forEach(function(c) { 
        document.cookie = c.replace(/^ +/, "").replace(/=.*/, "=;expires=" + new Date().toUTCString() + ";path=/"); 
      })
      
      // 清理IndexedDB（如果使用）
      if (window.indexedDB) {
        // 这里可以添加IndexedDB清理逻辑
      }
    })
    
    console.log('✅ 测试数据清理完成')
    
  } catch (error) {
    console.log('⚠️ 清理过程中出现错误:', error)
  } finally {
    await browser.close()
  }
  
  // 生成测试报告摘要
  console.log('📊 生成测试报告摘要...')
  try {
    const resultsPath = path.join(process.cwd(), 'test-results', 'results.json')
    
    if (fs.existsSync(resultsPath)) {
      const results = JSON.parse(fs.readFileSync(resultsPath, 'utf-8'))
      
      const summary = {
        totalTests: results.stats?.total || 0,
        passedTests: results.stats?.passed || 0,
        failedTests: results.stats?.failed || 0,
        skippedTests: results.stats?.skipped || 0,
        duration: results.stats?.duration || 0,
        timestamp: new Date().toISOString()
      }
      
      console.log('📈 测试结果摘要:')
      console.log(`   总测试数: ${summary.totalTests}`)
      console.log(`   通过: ${summary.passedTests}`)
      console.log(`   失败: ${summary.failedTests}`)
      console.log(`   跳过: ${summary.skippedTests}`)
      console.log(`   耗时: ${(summary.duration / 1000).toFixed(2)}秒`)
      
      // 保存摘要
      const summaryPath = path.join(process.cwd(), 'test-results', 'summary.json')
      fs.writeFileSync(summaryPath, JSON.stringify(summary, null, 2))
      
      // 生成简单的文本报告
      const textReport = `
HotSpot E2E测试报告
==================
测试时间: ${summary.timestamp}
总测试数: ${summary.totalTests}
通过: ${summary.passedTests}
失败: ${summary.failedTests}
跳过: ${summary.skippedTests}
耗时: ${(summary.duration / 1000).toFixed(2)}秒
成功率: ${summary.totalTests > 0 ? ((summary.passedTests / summary.totalTests) * 100).toFixed(2) : 0}%

详细报告请查看: test-results/html-report/index.html
`
      
      const textReportPath = path.join(process.cwd(), 'test-results', 'report.txt')
      fs.writeFileSync(textReportPath, textReport)
      
      console.log('✅ 测试报告生成完成')
      
      // 如果有失败的测试，显示警告
      if (summary.failedTests > 0) {
        console.log('⚠️ 发现失败的测试，请检查详细报告')
      }
      
    } else {
      console.log('⚠️ 未找到测试结果文件')
    }
    
  } catch (error) {
    console.log('⚠️ 生成测试报告时出错:', error)
  }
  
  // 清理临时文件（可选）
  console.log('🗂️ 清理临时文件...')
  try {
    const tempDirs = [
      path.join(process.cwd(), 'test-results', 'artifacts'),
      path.join(process.cwd(), '.temp')
    ]
    
    for (const tempDir of tempDirs) {
      if (fs.existsSync(tempDir)) {
        // 这里可以添加清理逻辑，但要小心不要删除重要文件
        console.log(`清理目录: ${tempDir}`)
      }
    }
    
  } catch (error) {
    console.log('⚠️ 清理临时文件时出错:', error)
  }
  
  console.log('✅ 全局清理完成')
  console.log('📋 测试报告位置:')
  console.log('   - HTML报告: test-results/html-report/index.html')
  console.log('   - JSON结果: test-results/results.json')
  console.log('   - JUnit报告: test-results/junit.xml')
  console.log('   - 文本摘要: test-results/report.txt')
}

export default globalTeardown
