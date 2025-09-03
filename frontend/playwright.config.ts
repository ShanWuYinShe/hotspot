import { defineConfig, devices } from '@playwright/test'

/**
 * HotSpot项目端到端测试配置
 * 
 * 配置Playwright测试环境，支持多浏览器、多设备测试
 */
export default defineConfig({
  // 测试目录
  testDir: './tests/e2e',
  
  // 全局测试超时时间（30秒）
  timeout: 30000,
  
  // 期望超时时间（5秒）
  expect: {
    timeout: 5000
  },
  
  // 并行运行测试
  fullyParallel: true,
  
  // 失败时不重试（开发阶段）
  retries: process.env.CI ? 2 : 0,
  
  // 并行工作进程数
  workers: process.env.CI ? 1 : undefined,
  
  // 测试报告配置
  reporter: [
    ['html', { outputFolder: 'test-results/html-report' }],
    ['json', { outputFile: 'test-results/results.json' }],
    ['junit', { outputFile: 'test-results/junit.xml' }],
    ['list']
  ],
  
  // 全局测试配置
  use: {
    // 基础URL
    baseURL: process.env.BASE_URL || 'http://localhost:3000',
    
    // 浏览器追踪
    trace: 'on-first-retry',
    
    // 截图配置
    screenshot: 'only-on-failure',
    
    // 视频录制
    video: 'retain-on-failure',
    
    // 忽略HTTPS错误
    ignoreHTTPSErrors: true,
    
    // 用户代理
    userAgent: 'HotSpot-E2E-Test/1.0',
    
    // 视口大小
    viewport: { width: 1280, height: 720 },
    
    // 语言设置
    locale: 'zh-CN',
    
    // 时区设置
    timezoneId: 'Asia/Shanghai'
  },

  // 项目配置（不同浏览器和设备）
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },

    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },

    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },

    // 移动端测试
    {
      name: 'Mobile Chrome',
      use: { ...devices['Pixel 5'] },
    },
    {
      name: 'Mobile Safari',
      use: { ...devices['iPhone 12'] },
    },

    // 平板端测试
    {
      name: 'Tablet',
      use: { 
        ...devices['iPad Pro'],
        viewport: { width: 1024, height: 768 }
      },
    },

    // 高分辨率桌面端测试
    {
      name: 'Desktop Large',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1920, height: 1080 }
      },
    }
  ],

  // 开发服务器配置
  webServer: process.env.CI ? undefined : {
    command: 'npm run dev',
    port: 3000,
    reuseExistingServer: !process.env.CI,
    timeout: 120000,
    env: {
      NODE_ENV: 'test'
    }
  },

  // 测试输出目录
  outputDir: 'test-results/artifacts',
  
  // 全局设置
  globalSetup: require.resolve('./tests/e2e/global-setup.ts'),
  globalTeardown: require.resolve('./tests/e2e/global-teardown.ts'),
})
