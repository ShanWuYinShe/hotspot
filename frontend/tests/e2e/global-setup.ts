/// 全局测试设置
/// 
/// 在所有测试开始前执行的设置操作

import { chromium, FullConfig } from '@playwright/test'

async function globalSetup(config: FullConfig) {
  console.log('🚀 开始E2E测试全局设置...')
  
  const baseURL = config.projects[0].use.baseURL || 'http://localhost:3000'
  
  // 启动浏览器进行预检查
  const browser = await chromium.launch()
  const page = await browser.newPage()
  
  try {
    console.log('📡 检查应用服务器连接...')
    
    // 等待应用服务器启动
    let retries = 30
    let connected = false
    
    while (retries > 0 && !connected) {
      try {
        const response = await page.goto(baseURL, { 
          waitUntil: 'networkidle',
          timeout: 5000 
        })
        
        if (response && response.status() < 400) {
          connected = true
          console.log('✅ 应用服务器连接成功')
        }
      } catch (error) {
        console.log(`⏳ 等待应用服务器启动... (剩余重试: ${retries})`)
        await page.waitForTimeout(2000)
        retries--
      }
    }
    
    if (!connected) {
      throw new Error('❌ 无法连接到应用服务器')
    }
    
    // 检查后端API连接
    console.log('🔌 检查后端API连接...')
    try {
      const healthResponse = await page.evaluate(async () => {
        try {
          const response = await fetch('/api/health', { method: 'POST' })
          return {
            status: response.status,
            ok: response.ok
          }
        } catch (error) {
          return { error: error.message }
        }
      })
      
      if (healthResponse.ok) {
        console.log('✅ 后端API连接正常')
      } else {
        console.log('⚠️ 后端API连接异常，但测试将继续进行')
        console.log('API响应:', healthResponse)
      }
    } catch (error) {
      console.log('⚠️ 后端API健康检查失败，但测试将继续进行')
      console.log('错误信息:', error)
    }
    
    // 预热应用（访问主要页面）
    console.log('🔥 预热应用页面...')
    const pagesToWarm = ['/', '/auth', '/profile', '/groups', '/messages']
    
    for (const pagePath of pagesToWarm) {
      try {
        await page.goto(`${baseURL}${pagePath}`, { 
          waitUntil: 'networkidle',
          timeout: 10000 
        })
        console.log(`✅ 预热页面: ${pagePath}`)
        await page.waitForTimeout(1000)
      } catch (error) {
        console.log(`⚠️ 预热页面失败: ${pagePath}`)
      }
    }
    
    // 清理测试数据（如果需要）
    console.log('🧹 清理测试环境...')
    await page.evaluate(() => {
      // 清理本地存储
      localStorage.clear()
      sessionStorage.clear()
      
      // 清理IndexedDB（如果使用）
      if (window.indexedDB) {
        // 这里可以添加IndexedDB清理逻辑
      }
    })
    
    console.log('✅ 全局设置完成')
    
  } catch (error) {
    console.error('❌ 全局设置失败:', error)
    throw error
  } finally {
    await browser.close()
  }
}

export default globalSetup
