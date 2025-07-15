<template>
  <div class="auth-page">
    <div class="auth-container">
      <div class="auth-header compact-only">
        <button @click="handleBack" class="back-btn">
          <Icon name="ion:arrow-back-outline" size="24" />
        </button>
        <span class="title">登录注册</span>
      </div>
      
      <!-- Logo 和标题 -->
      <div class="auth-header">
        <div class="logo">🔥</div>
        <h1 class="brand-title">HotSpot</h1>
        <p class="brand-subtitle">发现身边的精彩</p>
      </div>

      <!-- 切换标签 -->
      <div class="auth-tabs">
        <button 
          :class="['tab-btn', { active: pageState.isLogin }]"
          @click="switchToLogin"
        >
          登录
        </button>
        <button 
          :class="['tab-btn', { active: !pageState.isLogin }]"
          @click="switchToRegister"
        >
          注册
        </button>
      </div>

      <!-- 登录表单 -->
      <form v-if="pageState.isLogin" @submit.prevent="handleLogin" class="auth-form">
        <div class="form-group">
          <NInput
            v-model="loginForm.email"
            type="text"
            placeholder="邮箱地址"
            prefix-icon="Message"
            size="large"
            :class="{ error: errors.email }"
          />
          <span v-if="errors.email" class="error-text">{{ errors.email }}</span>
        </div>
        
        <div class="form-group">
          <NInput
            v-model="loginForm.password"
            type="password"
            placeholder="密码"
            prefix-icon="Lock"
            size="large"
            show-password
            :class="{ error: errors.password }"
          />
          <span v-if="errors.password" class="error-text">{{ errors.password }}</span>
        </div>

        <div class="form-options">
          <NCheckbox v-model="pageState.rememberMe">记住我</NCheckbox>
          <a href="#" class="forgot-link" @click.prevent="handleForgotPassword">忘记密码？</a>
        </div>

        <NButton 
          type="primary" 
          size="large" 
          class="auth-btn"
          :loading="pageState.loading"
          native-type="submit"
          block
        >
          登录
        </NButton>
      </form>

      <!-- 注册表单 -->
      <form v-else @submit.prevent="handleRegister" class="auth-form">
        <div class="form-group">
          <NInput
            v-model="registerForm.nickname"
            placeholder="昵称"
            prefix-icon="User"
            size="large"
            :class="{ error: errors.nickname }"
          />
          <span v-if="errors.nickname" class="error-text">{{ errors.nickname }}</span>
        </div>

        <div class="form-group">
          <NInput
            v-model="registerForm.email"
            type="text"
            placeholder="邮箱地址"
            prefix-icon="Message"
            size="large"
            :class="{ error: errors.email }"
          />
          <span v-if="errors.email" class="error-text">{{ errors.email }}</span>
        </div>
        
        <div class="form-group">
          <NInput
            v-model="registerForm.password"
            type="password"
            placeholder="密码"
            prefix-icon="Lock"
            size="large"
            show-password
            :class="{ error: errors.password }"
          />
          <span v-if="errors.password" class="error-text">{{ errors.password }}</span>
        </div>

        <div class="form-group">
          <NInput
            v-model="registerForm.confirmPassword"
            type="password"
            placeholder="确认密码"
            prefix-icon="Lock"
            size="large"
            show-password
            :class="{ error: errors.confirmPassword }"
          />
          <span v-if="errors.confirmPassword" class="error-text">{{ errors.confirmPassword }}</span>
        </div>

        <div class="form-options">
          <NCheckbox v-model="pageState.agreeTerms">
            我同意 
            <a href="#" class="link" @click.prevent="showTerms">用户协议</a> 
            和 
            <a href="#" class="link" @click.prevent="showPrivacy">隐私政策</a>
          </NCheckbox>
        </div>

        <NButton 
          type="primary" 
          size="large" 
          class="auth-btn"
          :loading="pageState.loading"
          :disabled="!pageState.agreeTerms"
          native-type="submit"
          block
        >
          注册
        </NButton>
      </form>

      <!-- 第三方登录 -->
      <div class="social-auth">
        <div class="divider">
          <span>或者使用以下方式</span>
        </div>
        
        <div class="social-buttons">
          <NButton 
            class="social-btn wechat" 
            @click="handleSocialLogin('wechat')"
            :loading="socialLoading.wechat"
          >
            <AppIcon name="wechat" size="20" />
            微信登录
          </NButton>
          
          <NButton 
            class="social-btn qq" 
            @click="handleSocialLogin('qq')"
            :loading="socialLoading.qq"
          >
            <AppIcon name="qq" size="20" />
            QQ登录
          </NButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// Vue相关导入
import { ref, reactive, shallowRef, onMounted, onUnmounted } from 'vue'
import { navigateTo } from '#app'
import { useMessage } from 'naive-ui'

// Naive UI 组件导入
import { NInput, NCheckbox, NButton } from 'naive-ui'

// Composables导入

// Stores导入
import { useAuthStore } from '~/stores/auth'

// API实例
const message = useMessage()

// TypeScript接口定义
interface LoginForm {
  email: string
  password: string
}

interface RegisterForm {
  nickname: string
  email: string
  password: string
  confirmPassword: string
}

interface AuthPageState {
  isLogin: boolean
  loading: boolean
  rememberMe: boolean
  agreeTerms: boolean
}

interface SocialLoadingState {
  wechat: boolean
  qq: boolean
}

interface FormErrors {
  [key: string]: string
}

interface LoginRequest {
  login_id: string
  password: string
}

interface RegisterRequest {
  login_id: string
  nickname: string
  password: string
}

type SocialProvider = 'wechat' | 'qq'

// 页面配置
definePageMeta({
  title: '登录注册',
  layout: 'default'
})

// Composables
const authStore = useAuthStore()

// 页面状态管理
const pageState = reactive<AuthPageState>({
  isLogin: true,
  loading: false,
  rememberMe: false,
  agreeTerms: false
})

// 表单数据
const loginForm = ref<LoginForm>({
  email: '',
  password: ''
})

const registerForm = ref<RegisterForm>({
  nickname: '',
  email: '',
  password: '',
  confirmPassword: ''
})

// 错误状态
const errors = shallowRef<FormErrors>({})

// 第三方登录加载状态
const socialLoading = reactive<SocialLoadingState>({
  wechat: false,
  qq: false
})

// 工具函数
const validateEmail = (email: string): boolean => {
  const re = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  return re.test(email)
}

const clearErrors = (): void => {
  errors.value = {}
}

// 切换函数
const switchToLogin = (): void => {
  pageState.isLogin = true
  clearErrors()
}

const switchToRegister = (): void => {
  pageState.isLogin = false
  clearErrors()
}

// 表单验证函数
const validateLoginForm = (): boolean => {
  const newErrors: FormErrors = {}
  
  if (!loginForm.value.email) {
    newErrors.email = '请输入邮箱地址'
  } else if (!validateEmail(loginForm.value.email)) {
    newErrors.email = '请输入有效的邮箱地址'
  }
  
  if (!loginForm.value.password) {
    newErrors.password = '请输入密码'
  } else if (loginForm.value.password.length < 6) {
    newErrors.password = '密码至少6位'
  }
  
  errors.value = newErrors
  return Object.keys(newErrors).length === 0
}

const validateRegisterForm = (): boolean => {
  const newErrors: FormErrors = {}
  
  if (!registerForm.value.nickname) {
    newErrors.nickname = '请输入昵称'
  } else if (registerForm.value.nickname.length < 2) {
    newErrors.nickname = '昵称至少2位'
  }
  
  if (!registerForm.value.email) {
    newErrors.email = '请输入邮箱地址'
  } else if (!validateEmail(registerForm.value.email)) {
    newErrors.email = '请输入有效的邮箱地址'
  }
  
  if (!registerForm.value.password) {
    newErrors.password = '请输入密码'
  } else if (registerForm.value.password.length < 6) {
    newErrors.password = '密码至少6位'
  }
  
  if (!registerForm.value.confirmPassword) {
    newErrors.confirmPassword = '请确认密码'
  } else if (registerForm.value.password !== registerForm.value.confirmPassword) {
    newErrors.confirmPassword = '两次密码输入不一致'
  }
  
  errors.value = newErrors
  return Object.keys(newErrors).length === 0
}

// 业务处理函数
const handleLogin = async (): Promise<void> => {
  if (!validateLoginForm()) return
  
  pageState.loading = true
  try {
    const loginRequest: LoginRequest = {
      login_id: loginForm.value.email,
      password: loginForm.value.password
    }
    
    await authStore.login(loginRequest)
    message.success('登录成功')
    await navigateTo('/')
  } catch (error: any) {
    message.error(error.message || '登录失败')
  } finally {
    pageState.loading = false
  }
}

const handleRegister = async (): Promise<void> => {
  if (!validateRegisterForm()) return
  
  pageState.loading = true
  try {
    const registerRequest: RegisterRequest = {
      login_id: registerForm.value.email,
      nickname: registerForm.value.nickname,
      password: registerForm.value.password
    }
    
    await authStore.register(registerRequest)
    message.success('注册成功')
    switchToLogin()
  } catch (error: any) {
    message.error(error.message || '注册失败')
  } finally {
    pageState.loading = false
  }
}

const handleSocialLogin = async (provider: SocialProvider): Promise<void> => {
  socialLoading[provider] = true
  try {
    // 这里将来实现第三方登录逻辑
    message.info(`${provider === 'wechat' ? '微信' : 'QQ'}登录功能即将上线`)
  } catch (error: any) {
    message.error(`${provider === 'wechat' ? '微信' : 'QQ'}登录失败`)
  } finally {
    socialLoading[provider] = false
  }
}

const handleForgotPassword = (): void => {
  message.info('忘记密码功能正在开发中')
}

const showTerms = (): void => {
  message.info('用户协议')
}

const showPrivacy = (): void => {
  message.info('隐私政策')
}

const handleBack = (): void => {
  navigateTo('/')
}
</script>

<style scoped lang="scss">
.auth-page {
  min-height: 100vh;
  width: 100%;
  margin: 0;
  padding: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  box-sizing: border-box;
  
  @media (min-width: 769px) {
    padding: 20px;
  }
  
  @media (max-width: 768px) {
    align-items: flex-start;
  }
}

.auth-container {
  background: white;
  border-radius: 16px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
  width: 100%;
  max-width: 400px;
  overflow: hidden;
  
  @media (max-width: 768px) {
    border-radius: 0;
    min-height: 100vh;
    max-width: none;
    box-shadow: none;
  }
}

.auth-header {
  text-align: center;
  padding: 40px 30px 30px;
  background: linear-gradient(135deg, #1e88e5 0%, #1565c0 100%);
  color: white;
  
  .logo {
    font-size: 48px;
    margin-bottom: 12px;
  }
  
  .brand-title {
    font-size: 32px;
    font-weight: 700;
    margin: 0 0 8px 0;
    font-family: 'Baloo 2', sans-serif;
  }
  
  .brand-subtitle {
    font-size: 16px;
    margin: 0;
    opacity: 0.9;
  }
  
  @media (max-width: 768px) {
    padding: 60px 30px 30px;
    
    .logo {
      font-size: 40px;
    }
    
    .brand-title {
      font-size: 28px;
    }
  }
}

.auth-tabs {
  display: flex;
  background: #f5f5f5;
  
  .tab-btn {
    flex: 1;
    padding: 16px;
    border: none;
    background: transparent;
    font-size: 16px;
    font-weight: 600;
    color: #666;
    cursor: pointer;
    transition: all 0.3s ease;
    
    &.active {
      background: white;
      color: #1e88e5;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    }
    
    &:hover:not(.active) {
      background: #eeeeee;
    }
  }
}

.auth-form {
  padding: 30px;
  
  .form-group {
    margin-bottom: 20px;
    
    .el-input {
      :deep(.el-input__wrapper) {
        border-radius: 12px;
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
        border: 1px solid #e0e0e0;
        transition: all 0.3s ease;
        
        &:hover {
          border-color: #1e88e5;
        }
        
        &.is-focus {
          border-color: #1e88e5;
          box-shadow: 0 2px 12px rgba(30, 136, 229, 0.2);
        }
      }
      
      &.error :deep(.el-input__wrapper) {
        border-color: #f56565;
        box-shadow: 0 2px 8px rgba(245, 101, 101, 0.15);
      }
    }
    
    .error-text {
      color: #f56565;
      font-size: 14px;
      margin-top: 8px;
      display: block;
    }
  }
  
  .form-options {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
    font-size: 14px;
    
    .forgot-link {
      color: #1e88e5;
      text-decoration: none;
      
      &:hover {
        text-decoration: underline;
      }
    }
  }
  
  .auth-btn {
    height: 48px;
    border-radius: 12px;
    font-size: 16px;
    font-weight: 600;
    background: linear-gradient(135deg, #1e88e5 0%, #1565c0 100%);
    border: none;
    box-shadow: 0 4px 16px rgba(30, 136, 229, 0.3);
    transition: all 0.3s ease;
    
    &:hover {
      transform: translateY(-2px);
      box-shadow: 0 6px 20px rgba(30, 136, 229, 0.4);
    }
    
    &:active {
      transform: translateY(0);
    }
  }
}

.social-auth {
  padding: 0 30px 30px;
  
  .divider {
    text-align: center;
    margin: 20px 0;
    position: relative;
    
    &::before {
      content: '';
      position: absolute;
      top: 50%;
      left: 0;
      right: 0;
      height: 1px;
      background: #e0e0e0;
    }
    
    span {
      background: white;
      padding: 0 16px;
      color: #666;
      font-size: 14px;
      position: relative;
    }
  }
  
  .social-buttons {
    display: flex;
    gap: 12px;
    
    .social-btn {
      flex: 1;
      height: 44px;
      border-radius: 10px;
      font-size: 14px;
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
      border: 1px solid #e0e0e0;
      
      &.wechat {
        color: #1aad19;
        border-color: #1aad19;
        
        &:hover {
          background: #f0f9ff;
        }
      }
      
      &.qq {
        color: #12b7f5;
        border-color: #12b7f5;
        
        &:hover {
          background: #f0f9ff;
        }
      }
    }
  }
}

.link {
  color: #1e88e5;
  text-decoration: none;
  
  &:hover {
    text-decoration: underline;
  }
}

// 移动端适配
@media (max-width: 768px) {
  .auth-form {
    padding: 20px;
  }
  
  .social-auth {
    padding: 0 20px 20px;
  }
  
  .social-buttons {
    flex-direction: column;
    
    .social-btn {
      width: 100%;
    }
  }
}

// 安全区域适配
@supports (padding: max(0px)) {
  @media (max-width: 768px) {
    .auth-container {
      padding-bottom: max(20px, env(safe-area-inset-bottom));
    }
  }
}
</style>