/// API功能端到端测试
/// 
/// 测试前后端API接口的集成和数据流

import { test, expect } from '@playwright/test'

test.describe('API功能测试', () => {
  test.beforeEach(async ({ page }) => {
    // 每个测试前清理状态
    await page.goto('/')
    await page.evaluate(() => {
      localStorage.clear()
      sessionStorage.clear()
    })
    
    // 等待页面加载和临时用户创建
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(3000)
  })

  test('健康检查API测试', async ({ page }) => {
    // 监听网络请求
    const healthRequests: any[] = []
    
    page.on('response', response => {
      if (response.url().includes('/health')) {
        healthRequests.push({
          url: response.url(),
          status: response.status(),
          statusText: response.statusText()
        })
      }
    })
    
    // 触发健康检查请求（可能在应用启动时自动触发）
    await page.evaluate(async () => {
      try {
        const response = await fetch('/api/health', { method: 'POST' })
        console.log('健康检查响应状态:', response.status)
        const data = await response.json()
        console.log('健康检查响应数据:', data)
      } catch (error) {
        console.log('健康检查请求失败:', error)
      }
    })
    
    await page.waitForTimeout(2000)
    
    console.log('健康检查请求记录:', healthRequests)
    
    // 验证健康检查请求
    if (healthRequests.length > 0) {
      const healthRequest = healthRequests[0]
      expect(healthRequest.status).toBe(200)
      console.log('✓ 健康检查API正常')
    } else {
      console.log('未检测到健康检查请求')
    }
  })

  test('用户API集成测试', async ({ page }) => {
    const apiRequests: any[] = []
    
    // 监听用户相关的API请求
    page.on('response', response => {
      const url = response.url()
      if (url.includes('/api/users') || url.includes('/api/auth')) {
        apiRequests.push({
          url,
          method: response.request().method(),
          status: response.status(),
          statusText: response.statusText()
        })
      }
    })
    
    // 访问用户资料页面
    await page.goto('/profile')
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(3000)
    
    console.log('用户API请求记录:', apiRequests)
    
    // 检查用户资料页面内容
    const profileContent = page.locator('[data-testid="profile-content"]')
    if (await profileContent.isVisible()) {
      console.log('✓ 用户资料页面加载成功')
      
      // 检查用户信息显示
      const userInfo = page.locator('[data-testid="user-info"]')
      if (await userInfo.isVisible()) {
        const userInfoText = await userInfo.textContent()
        console.log('用户信息:', userInfoText)
      }
    }
    
    // 验证API请求状态
    const successfulRequests = apiRequests.filter(req => req.status >= 200 && req.status < 300)
    console.log('成功的API请求数量:', successfulRequests.length)
  })

  test('群组API集成测试', async ({ page }) => {
    const groupRequests: any[] = []
    
    // 监听群组相关的API请求
    page.on('response', response => {
      const url = response.url()
      if (url.includes('/api/groups')) {
        groupRequests.push({
          url,
          method: response.request().method(),
          status: response.status(),
          statusText: response.statusText()
        })
      }
    })
    
    // 访问群组页面
    await page.goto('/groups')
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(3000)
    
    console.log('群组API请求记录:', groupRequests)
    
    // 检查群组列表
    const groupsList = page.locator('[data-testid="groups-list"]')
    if (await groupsList.isVisible()) {
      console.log('✓ 群组列表加载成功')
      
      // 检查群组项目
      const groupItems = page.locator('[data-testid="group-item"]')
      const groupCount = await groupItems.count()
      console.log('群组数量:', groupCount)
    }
    
    // 测试群组搜索功能（如果存在）
    const searchInput = page.locator('[data-testid="group-search-input"]')
    if (await searchInput.isVisible()) {
      await searchInput.fill('测试')
      await page.waitForTimeout(1000)
      
      // 检查搜索结果
      const searchResults = page.locator('[data-testid="search-results"]')
      if (await searchResults.isVisible()) {
        console.log('✓ 群组搜索功能正常')
      }
    }
  })

  test('消息API集成测试', async ({ page }) => {
    const messageRequests: any[] = []
    
    // 监听消息相关的API请求
    page.on('response', response => {
      const url = response.url()
      if (url.includes('/api/messages') || url.includes('/api/conversations')) {
        messageRequests.push({
          url,
          method: response.request().method(),
          status: response.status(),
          statusText: response.statusText()
        })
      }
    })
    
    // 访问消息页面
    await page.goto('/messages')
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(3000)
    
    console.log('消息API请求记录:', messageRequests)
    
    // 检查消息界面
    const messagesInterface = page.locator('[data-testid="messages-interface"]')
    if (await messagesInterface.isVisible()) {
      console.log('✓ 消息界面加载成功')
      
      // 检查对话列表
      const conversationsList = page.locator('[data-testid="conversations-list"]')
      if (await conversationsList.isVisible()) {
        const conversationItems = page.locator('[data-testid="conversation-item"]')
        const conversationCount = await conversationItems.count()
        console.log('对话数量:', conversationCount)
      }
    }
  })

  test('地理位置API集成测试', async ({ page }) => {
    const locationRequests: any[] = []
    
    // 监听地理位置相关的API请求
    page.on('response', response => {
      const url = response.url()
      if (url.includes('/api/locations') || url.includes('/api/nearby')) {
        locationRequests.push({
          url,
          method: response.request().method(),
          status: response.status(),
          statusText: response.statusText()
        })
      }
    })
    
    // 模拟地理位置权限
    await page.context().grantPermissions(['geolocation'])
    await page.setGeolocation({ latitude: 39.9042, longitude: 116.4074 })
    
    // 访问首页（可能会触发位置相关请求）
    await page.goto('/')
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(5000)
    
    console.log('地理位置API请求记录:', locationRequests)
    
    // 检查附近用户或群组功能
    const nearbySection = page.locator('[data-testid="nearby-section"]')
    if (await nearbySection.isVisible()) {
      console.log('✓ 附近功能区域已显示')
      
      // 检查附近用户
      const nearbyUsers = page.locator('[data-testid="nearby-users"]')
      if (await nearbyUsers.isVisible()) {
        const userItems = page.locator('[data-testid="nearby-user-item"]')
        const userCount = await userItems.count()
        console.log('附近用户数量:', userCount)
      }
      
      // 检查附近群组
      const nearbyGroups = page.locator('[data-testid="nearby-groups"]')
      if (await nearbyGroups.isVisible()) {
        const groupItems = page.locator('[data-testid="nearby-group-item"]')
        const groupCount = await groupItems.count()
        console.log('附近群组数量:', groupCount)
      }
    }
  })

  test('API错误处理测试', async ({ page }) => {
    const errorResponses: any[] = []
    
    // 监听API错误响应
    page.on('response', response => {
      if (response.status() >= 400) {
        errorResponses.push({
          url: response.url(),
          method: response.request().method(),
          status: response.status(),
          statusText: response.statusText()
        })
      }
    })
    
    // 模拟网络错误
    await page.route('**/api/test-error', route => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal Server Error' })
      })
    })
    
    // 触发错误请求
    await page.evaluate(async () => {
      try {
        await fetch('/api/test-error', { method: 'POST' })
      } catch (error) {
        console.log('预期的API错误:', error)
      }
    })
    
    await page.waitForTimeout(2000)
    
    console.log('API错误响应记录:', errorResponses)
    
    // 检查错误处理UI
    const errorToast = page.locator('[data-testid="error-toast"]')
    if (await errorToast.isVisible()) {
      console.log('✓ 错误提示已显示')
      const errorText = await errorToast.textContent()
      console.log('错误提示内容:', errorText)
    }
  })

  test('API性能测试', async ({ page }) => {
    const requestTimes: { [key: string]: number[] } = {}
    
    // 监听API请求并记录响应时间
    page.on('response', response => {
      const url = response.url()
      if (url.includes('/api/')) {
        const requestTime = response.request().timing()?.responseEnd || 0
        const endpoint = url.split('/api/')[1]?.split('?')[0] || 'unknown'
        
        if (!requestTimes[endpoint]) {
          requestTimes[endpoint] = []
        }
        requestTimes[endpoint].push(requestTime)
      }
    })
    
    // 访问多个页面触发不同的API请求
    const pages = ['/', '/profile', '/groups', '/messages']
    
    for (const pagePath of pages) {
      await page.goto(pagePath)
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
    }
    
    // 分析API性能
    console.log('API性能统计:')
    for (const [endpoint, times] of Object.entries(requestTimes)) {
      if (times.length > 0) {
        const avgTime = times.reduce((a, b) => a + b, 0) / times.length
        const maxTime = Math.max(...times)
        const minTime = Math.min(...times)
        
        console.log(`${endpoint}:`)
        console.log(`  - 请求次数: ${times.length}`)
        console.log(`  - 平均响应时间: ${avgTime.toFixed(2)}ms`)
        console.log(`  - 最大响应时间: ${maxTime}ms`)
        console.log(`  - 最小响应时间: ${minTime}ms`)
        
        // 验证性能指标
        expect(avgTime).toBeLessThan(5000) // 平均响应时间应小于5秒
        expect(maxTime).toBeLessThan(10000) // 最大响应时间应小于10秒
      }
    }
  })

  test('API数据一致性测试', async ({ page }) => {
    const apiResponses: { [key: string]: any } = {}
    
    // 监听API响应并保存数据
    page.on('response', async response => {
      const url = response.url()
      if (url.includes('/api/') && response.status() === 200) {
        try {
          const data = await response.json()
          const endpoint = url.split('/api/')[1]?.split('?')[0] || 'unknown'
          apiResponses[endpoint] = data
        } catch (e) {
          // 忽略非JSON响应
        }
      }
    })
    
    // 访问用户资料页面
    await page.goto('/profile')
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(3000)
    
    // 检查API响应数据结构
    console.log('API响应数据结构验证:')
    for (const [endpoint, data] of Object.entries(apiResponses)) {
      console.log(`${endpoint}:`)
      
      // 验证通用字段
      if (data && typeof data === 'object') {
        if ('code' in data) {
          console.log(`  - 响应码: ${data.code}`)
          expect(data.code).toBe(0) // 成功响应码应为0
        }
        
        if ('msg' in data || 'message' in data) {
          const message = data.msg || data.message
          console.log(`  - 响应消息: ${message}`)
          expect(typeof message).toBe('string')
        }
        
        if ('data' in data) {
          console.log(`  - 数据类型: ${typeof data.data}`)
          console.log(`  - 数据结构: ${JSON.stringify(data.data, null, 2).substring(0, 200)}...`)
        }
      }
    }
  })
})
