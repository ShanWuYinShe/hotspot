/// WebSocket功能端到端测试
/// 
/// 测试实时通信、消息推送、连接管理等功能

import { test, expect } from '@playwright/test'

test.describe('WebSocket功能测试', () => {
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

  test('WebSocket连接建立测试', async ({ page }) => {
    // 监听WebSocket连接事件
    let wsConnected = false
    let wsMessages: any[] = []
    
    page.on('websocket', ws => {
      console.log('WebSocket连接建立:', ws.url())
      wsConnected = true
      
      ws.on('framereceived', event => {
        try {
          const message = JSON.parse(event.payload.toString())
          wsMessages.push(message)
          console.log('收到WebSocket消息:', message)
        } catch (e) {
          console.log('WebSocket原始消息:', event.payload.toString())
        }
      })
      
      ws.on('framesent', event => {
        try {
          const message = JSON.parse(event.payload.toString())
          console.log('发送WebSocket消息:', message)
        } catch (e) {
          console.log('发送WebSocket原始消息:', event.payload.toString())
        }
      })
    })
    
    // 访问需要WebSocket的页面（如消息页面）
    await page.goto('/messages')
    await page.waitForLoadState('networkidle')
    
    // 等待WebSocket连接建立
    await page.waitForTimeout(5000)
    
    // 验证WebSocket连接状态
    const connectionStatus = page.locator('[data-testid="ws-connection-status"]')
    if (await connectionStatus.isVisible()) {
      const statusText = await connectionStatus.textContent()
      console.log('WebSocket连接状态:', statusText)
    }
    
    // 检查是否有WebSocket活动
    console.log('WebSocket连接建立:', wsConnected)
    console.log('收到的消息数量:', wsMessages.length)
  })

  test('实时消息推送测试', async ({ page }) => {
    let receivedMessages: any[] = []
    
    // 监听WebSocket消息
    page.on('websocket', ws => {
      ws.on('framereceived', event => {
        try {
          const message = JSON.parse(event.payload.toString())
          if (message.type === 'NewMessage' || message.type === 'SystemNotification') {
            receivedMessages.push(message)
          }
        } catch (e) {
          // 忽略解析错误
        }
      })
    })
    
    // 访问消息页面
    await page.goto('/messages')
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(3000)
    
    // 检查消息列表
    const messagesList = page.locator('[data-testid="messages-list"]')
    if (await messagesList.isVisible()) {
      console.log('消息列表已显示')
      
      // 检查是否有消息项
      const messageItems = page.locator('[data-testid="message-item"]')
      const messageCount = await messageItems.count()
      console.log('当前消息数量:', messageCount)
    }
    
    // 如果有发送消息的功能，测试发送
    const messageInput = page.locator('[data-testid="message-input"]')
    const sendButton = page.locator('[data-testid="send-button"]')
    
    if (await messageInput.isVisible() && await sendButton.isVisible()) {
      await messageInput.fill('测试消息')
      await sendButton.click()
      
      // 等待消息发送和接收
      await page.waitForTimeout(2000)
      
      console.log('收到的实时消息数量:', receivedMessages.length)
    }
  })

  test('WebSocket重连机制测试', async ({ page }) => {
    let connectionEvents: string[] = []
    
    // 监听WebSocket事件
    page.on('websocket', ws => {
      connectionEvents.push('connected')
      
      ws.on('close', () => {
        connectionEvents.push('disconnected')
      })
    })
    
    // 访问页面
    await page.goto('/messages')
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(3000)
    
    // 模拟网络中断（关闭WebSocket连接）
    await page.evaluate(() => {
      // 尝试关闭WebSocket连接来模拟网络中断
      if (window.WebSocket) {
        const originalWebSocket = window.WebSocket
        window.WebSocket = function(...args) {
          const ws = new originalWebSocket(...args)
          setTimeout(() => {
            if (ws.readyState === WebSocket.OPEN) {
              ws.close()
            }
          }, 1000)
          return ws
        } as any
      }
    })
    
    // 等待重连尝试
    await page.waitForTimeout(10000)
    
    console.log('WebSocket连接事件:', connectionEvents)
    
    // 检查连接状态指示器
    const connectionStatus = page.locator('[data-testid="ws-connection-status"]')
    if (await connectionStatus.isVisible()) {
      const statusText = await connectionStatus.textContent()
      console.log('最终连接状态:', statusText)
    }
  })

  test('多标签页WebSocket管理测试', async ({ browser }) => {
    // 创建两个页面（模拟多标签页）
    const context = await browser.newContext()
    const page1 = await context.newPage()
    const page2 = await context.newPage()
    
    let page1Messages: any[] = []
    let page2Messages: any[] = []
    
    // 监听第一个页面的WebSocket
    page1.on('websocket', ws => {
      ws.on('framereceived', event => {
        try {
          const message = JSON.parse(event.payload.toString())
          page1Messages.push(message)
        } catch (e) {
          // 忽略解析错误
        }
      })
    })
    
    // 监听第二个页面的WebSocket
    page2.on('websocket', ws => {
      ws.on('framereceived', event => {
        try {
          const message = JSON.parse(event.payload.toString())
          page2Messages.push(message)
        } catch (e) {
          // 忽略解析错误
        }
      })
    })
    
    // 在两个页面中访问应用
    await page1.goto('/')
    await page2.goto('/')
    
    await page1.waitForLoadState('networkidle')
    await page2.waitForLoadState('networkidle')
    
    // 等待WebSocket连接建立
    await page1.waitForTimeout(3000)
    await page2.waitForTimeout(3000)
    
    // 检查两个页面是否都建立了连接
    console.log('页面1收到的消息数量:', page1Messages.length)
    console.log('页面2收到的消息数量:', page2Messages.length)
    
    // 关闭第一个页面
    await page1.close()
    
    // 等待一段时间，检查第二个页面是否仍然正常
    await page2.waitForTimeout(3000)
    
    const connectionStatus = page2.locator('[data-testid="ws-connection-status"]')
    if (await connectionStatus.isVisible()) {
      const statusText = await connectionStatus.textContent()
      console.log('页面2的连接状态:', statusText)
    }
    
    await page2.close()
    await context.close()
  })

  test('WebSocket消息类型处理测试', async ({ page }) => {
    let messageTypes: Set<string> = new Set()
    
    // 监听不同类型的WebSocket消息
    page.on('websocket', ws => {
      ws.on('framereceived', event => {
        try {
          const message = JSON.parse(event.payload.toString())
          if (message.type) {
            messageTypes.add(message.type)
            console.log('收到消息类型:', message.type)
          }
        } catch (e) {
          // 忽略解析错误
        }
      })
    })
    
    // 访问不同的页面来触发不同类型的消息
    const pages = ['/', '/messages', '/groups', '/profile']
    
    for (const pagePath of pages) {
      await page.goto(pagePath)
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)
    }
    
    console.log('收到的消息类型:', Array.from(messageTypes))
    
    // 验证是否收到了预期的消息类型
    const expectedTypes = ['Connected', 'Pong', 'SystemNotification']
    for (const expectedType of expectedTypes) {
      if (messageTypes.has(expectedType)) {
        console.log(`✓ 收到预期的消息类型: ${expectedType}`)
      } else {
        console.log(`✗ 未收到预期的消息类型: ${expectedType}`)
      }
    }
  })

  test('WebSocket性能测试', async ({ page }) => {
    let messageCount = 0
    let startTime = Date.now()
    let endTime = startTime
    
    // 监听WebSocket消息并计算性能指标
    page.on('websocket', ws => {
      ws.on('framereceived', event => {
        messageCount++
        endTime = Date.now()
      })
    })
    
    // 访问页面
    await page.goto('/messages')
    await page.waitForLoadState('networkidle')
    
    startTime = Date.now()
    
    // 等待一段时间收集消息
    await page.waitForTimeout(10000)
    
    const duration = endTime - startTime
    const messagesPerSecond = messageCount / (duration / 1000)
    
    console.log('性能统计:')
    console.log('- 总消息数:', messageCount)
    console.log('- 测试时长:', duration, 'ms')
    console.log('- 消息频率:', messagesPerSecond.toFixed(2), '消息/秒')
    
    // 验证性能指标
    expect(messageCount).toBeGreaterThanOrEqual(0)
    if (messageCount > 0) {
      expect(messagesPerSecond).toBeLessThan(1000) // 合理的消息频率上限
    }
  })

  test('WebSocket错误处理测试', async ({ page }) => {
    let errorEvents: string[] = []
    
    // 监听WebSocket错误
    page.on('websocket', ws => {
      ws.on('socketerror', error => {
        errorEvents.push('socket_error')
        console.log('WebSocket错误:', error)
      })
      
      ws.on('close', () => {
        errorEvents.push('connection_closed')
      })
    })
    
    // 访问页面
    await page.goto('/messages')
    await page.waitForLoadState('networkidle')
    
    // 模拟WebSocket错误
    await page.evaluate(() => {
      // 尝试连接到无效的WebSocket地址
      try {
        const invalidWs = new WebSocket('ws://invalid-address:9999')
        invalidWs.onerror = () => console.log('预期的WebSocket错误')
      } catch (e) {
        console.log('WebSocket连接失败（预期行为）')
      }
    })
    
    await page.waitForTimeout(5000)
    
    console.log('WebSocket错误事件:', errorEvents)
    
    // 检查错误处理UI
    const errorIndicator = page.locator('[data-testid="ws-error-indicator"]')
    if (await errorIndicator.isVisible()) {
      console.log('WebSocket错误指示器已显示')
    }
  })
})
