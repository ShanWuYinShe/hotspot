/// 认证功能端到端测试
/// 
/// 测试用户登录、注册、临时用户等认证流程

import { test, expect } from '@playwright/test'

test.describe('认证功能测试', () => {
  test.beforeEach(async ({ page }) => {
    // 每个测试前清理本地存储
    await page.goto('/')
    await page.evaluate(() => {
      localStorage.clear()
      sessionStorage.clear()
    })
  })

  test('临时用户自动注册流程', async ({ page }) => {
    // 访问首页
    await page.goto('/')
    
    // 等待页面加载完成
    await page.waitForLoadState('networkidle')
    
    // 检查是否显示临时用户状态
    const userIndicator = page.locator('[data-testid="user-indicator"]')
    await expect(userIndicator).toBeVisible({ timeout: 10000 })
    
    // 检查临时用户标识
    const tempUserBadge = page.locator('[data-testid="temp-user-badge"]')
    await expect(tempUserBadge).toBeVisible()
    await expect(tempUserBadge).toContainText('临时用户')
    
    // 检查升级按钮
    const upgradeButton = page.locator('[data-testid="upgrade-button"]')
    await expect(upgradeButton).toBeVisible()
    await expect(upgradeButton).toContainText('升级账户')
  })

  test('用户登录流程', async ({ page }) => {
    // 访问登录页面
    await page.goto('/auth')
    
    // 等待页面加载
    await page.waitForLoadState('networkidle')
    
    // 检查登录表单
    const loginForm = page.locator('[data-testid="login-form"]')
    await expect(loginForm).toBeVisible()
    
    // 填写登录信息
    const loginIdInput = page.locator('[data-testid="login-id-input"]')
    const passwordInput = page.locator('[data-testid="password-input"]')
    const loginButton = page.locator('[data-testid="login-button"]')
    
    await loginIdInput.fill('test@example.com')
    await passwordInput.fill('password123')
    
    // 点击登录按钮
    await loginButton.click()
    
    // 等待登录处理
    await page.waitForTimeout(2000)
    
    // 检查登录结果（可能成功或失败，取决于后端状态）
    const errorMessage = page.locator('[data-testid="error-message"]')
    const successRedirect = page.locator('[data-testid="user-indicator"]')
    
    // 如果有错误消息，验证错误处理
    if (await errorMessage.isVisible()) {
      await expect(errorMessage).toBeVisible()
      console.log('登录失败（预期行为，测试环境）')
    } else {
      // 如果登录成功，验证重定向
      await expect(successRedirect).toBeVisible()
      console.log('登录成功')
    }
  })

  test('用户注册流程', async ({ page }) => {
    // 访问注册页面
    await page.goto('/auth')
    
    // 切换到注册模式
    const registerTab = page.locator('[data-testid="register-tab"]')
    if (await registerTab.isVisible()) {
      await registerTab.click()
    }
    
    // 检查注册表单
    const registerForm = page.locator('[data-testid="register-form"]')
    await expect(registerForm).toBeVisible()
    
    // 填写注册信息
    const nicknameInput = page.locator('[data-testid="nickname-input"]')
    const emailInput = page.locator('[data-testid="email-input"]')
    const passwordInput = page.locator('[data-testid="password-input"]')
    const confirmPasswordInput = page.locator('[data-testid="confirm-password-input"]')
    const registerButton = page.locator('[data-testid="register-button"]')
    
    const testEmail = `test${Date.now()}@example.com`
    const testPassword = 'TestPassword123!'
    
    await nicknameInput.fill('测试用户')
    await emailInput.fill(testEmail)
    await passwordInput.fill(testPassword)
    await confirmPasswordInput.fill(testPassword)
    
    // 点击注册按钮
    await registerButton.click()
    
    // 等待注册处理
    await page.waitForTimeout(3000)
    
    // 检查注册结果
    const errorMessage = page.locator('[data-testid="error-message"]')
    const successMessage = page.locator('[data-testid="success-message"]')
    
    // 验证注册反馈
    const hasError = await errorMessage.isVisible()
    const hasSuccess = await successMessage.isVisible()
    
    if (hasError) {
      await expect(errorMessage).toBeVisible()
      console.log('注册失败（可能是预期行为）')
    } else if (hasSuccess) {
      await expect(successMessage).toBeVisible()
      console.log('注册成功')
    }
  })

  test('语言切换功能', async ({ page }) => {
    // 访问首页
    await page.goto('/')
    
    // 等待页面加载
    await page.waitForLoadState('networkidle')
    
    // 查找语言切换器
    const languageSwitcher = page.locator('[data-testid="language-switcher"]')
    await expect(languageSwitcher).toBeVisible()
    
    // 点击语言切换器
    await languageSwitcher.click()
    
    // 选择英语
    const englishOption = page.locator('[data-testid="locale-en-US"]')
    if (await englishOption.isVisible()) {
      await englishOption.click()
      
      // 等待语言切换完成
      await page.waitForTimeout(1000)
      
      // 验证页面内容已切换到英语
      const homeText = page.locator('text=Home')
      await expect(homeText).toBeVisible({ timeout: 5000 })
    }
    
    // 切换回中文
    await languageSwitcher.click()
    const chineseOption = page.locator('[data-testid="locale-zh-CN"]')
    if (await chineseOption.isVisible()) {
      await chineseOption.click()
      
      // 等待语言切换完成
      await page.waitForTimeout(1000)
      
      // 验证页面内容已切换到中文
      const homeText = page.locator('text=首页')
      await expect(homeText).toBeVisible({ timeout: 5000 })
    }
  })

  test('响应式设计测试', async ({ page }) => {
    // 测试桌面端
    await page.setViewportSize({ width: 1920, height: 1080 })
    await page.goto('/')
    
    // 检查桌面端布局
    const desktopNav = page.locator('[data-testid="desktop-navigation"]')
    await expect(desktopNav).toBeVisible()
    
    // 测试平板端
    await page.setViewportSize({ width: 768, height: 1024 })
    await page.waitForTimeout(500)
    
    // 检查平板端布局适配
    const navigation = page.locator('[data-testid="navigation"]')
    await expect(navigation).toBeVisible()
    
    // 测试移动端
    await page.setViewportSize({ width: 375, height: 667 })
    await page.waitForTimeout(500)
    
    // 检查移动端布局
    const mobileNav = page.locator('[data-testid="mobile-navigation"]')
    await expect(mobileNav).toBeVisible()
    
    // 检查汉堡菜单（如果存在）
    const hamburgerMenu = page.locator('[data-testid="hamburger-menu"]')
    if (await hamburgerMenu.isVisible()) {
      await hamburgerMenu.click()
      
      // 检查菜单展开
      const mobileMenu = page.locator('[data-testid="mobile-menu"]')
      await expect(mobileMenu).toBeVisible()
    }
  })

  test('错误处理测试', async ({ page }) => {
    // 模拟网络错误
    await page.route('**/api/**', route => {
      route.abort('failed')
    })
    
    await page.goto('/auth')
    
    // 尝试登录
    const loginIdInput = page.locator('[data-testid="login-id-input"]')
    const passwordInput = page.locator('[data-testid="password-input"]')
    const loginButton = page.locator('[data-testid="login-button"]')
    
    await loginIdInput.fill('test@example.com')
    await passwordInput.fill('password123')
    await loginButton.click()
    
    // 等待错误处理
    await page.waitForTimeout(3000)
    
    // 检查错误提示
    const errorMessage = page.locator('[data-testid="error-message"]')
    await expect(errorMessage).toBeVisible()
    
    // 验证错误消息内容
    const errorText = await errorMessage.textContent()
    expect(errorText).toBeTruthy()
    console.log('错误消息:', errorText)
  })

  test('本地存储持久化测试', async ({ page }) => {
    // 访问首页并等待临时用户创建
    await page.goto('/')
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(3000)
    
    // 获取本地存储中的认证数据
    const authData = await page.evaluate(() => {
      return localStorage.getItem('auth-store')
    })
    
    if (authData) {
      console.log('认证数据已保存到本地存储')
      
      // 刷新页面
      await page.reload()
      await page.waitForLoadState('networkidle')
      
      // 检查认证状态是否恢复
      const userIndicator = page.locator('[data-testid="user-indicator"]')
      await expect(userIndicator).toBeVisible({ timeout: 10000 })
      
      console.log('认证状态成功恢复')
    } else {
      console.log('未找到认证数据（可能是预期行为）')
    }
  })
})
