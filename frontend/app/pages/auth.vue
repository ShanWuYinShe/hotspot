<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50">
    <UCard class="w-full max-w-md">
      <template #header>
        <div class="text-center">
          <h1 class="text-2xl font-bold text-gray-900">HotSpot</h1>
          <p class="text-gray-600">基于位置的社交平台</p>
        </div>
      </template>

      <!-- 登录表单 -->
      <UForm v-if="mode === 'login'" :state="loginForm" @submit="handleLogin">
        <div class="space-y-4">
          <UFormGroup label="用户名" name="loginId">
            <UInput 
              v-model="loginForm.loginId" 
              placeholder="请输入用户名"
              :disabled="isLoading"
            />
          </UFormGroup>
          
          <UFormGroup label="密码" name="password">
            <UInput 
              v-model="loginForm.password" 
              type="password" 
              placeholder="请输入密码"
              :disabled="isLoading"
            />
          </UFormGroup>

          <UButton 
            type="submit" 
            color="primary" 
            block 
            :loading="isLoading"
          >
            登录
          </UButton>

          <div class="text-center">
            <UButton 
              variant="ghost" 
              @click="mode = 'register'"
              :disabled="isLoading"
            >
              还没有账号？点击注册
            </UButton>
          </div>
        </div>
      </UForm>

      <!-- 注册表单 -->
      <UForm v-else :state="registerForm" @submit="handleRegister">
        <div class="space-y-4">
          <UFormGroup label="用户名" name="loginId">
            <UInput 
              v-model="registerForm.loginId" 
              placeholder="请输入用户名"
              :disabled="isLoading"
            />
          </UFormGroup>
          
          <UFormGroup label="昵称" name="nickname">
            <UInput 
              v-model="registerForm.nickname" 
              placeholder="请输入昵称"
              :disabled="isLoading"
            />
          </UFormGroup>
          
          <UFormGroup label="密码" name="password">
            <UInput 
              v-model="registerForm.password" 
              type="password" 
              placeholder="请输入密码"
              :disabled="isLoading"
            />
          </UFormGroup>

          <UButton 
            type="submit" 
            color="primary" 
            block 
            :loading="isLoading"
          >
            注册
          </UButton>

          <div class="text-center">
            <UButton 
              variant="ghost" 
              @click="mode = 'login'"
              :disabled="isLoading"
            >
              已有账号？点击登录
            </UButton>
          </div>
        </div>
      </UForm>

      <!-- Demo 提示 -->
      <div class="mt-6 p-4 bg-blue-50 rounded-lg">
        <p class="text-sm text-blue-700 text-center">
          这是演示版本，可以使用任意用户名和密码进行注册/登录
        </p>
      </div>
    </UCard>
  </div>
</template>

<script setup>
const { login, register, isLoading } = useAuth()
const toast = useToast()

// 表单模式
const mode = ref('login')

// 登录表单
const loginForm = ref({
  loginId: '',
  password: ''
})

// 注册表单
const registerForm = ref({
  loginId: '',
  nickname: '',
  password: ''
})

// 处理登录
const handleLogin = async () => {
  try {
    await login(loginForm.value.loginId, loginForm.value.password)
    toast.add({
      title: '登录成功',
      color: 'success'
    })
    await navigateTo('/')
  } catch (error) {
    // 错误已经在 useAuth 中处理了
  }
}

// 处理注册
const handleRegister = async () => {
  try {
    await register(
      registerForm.value.loginId, 
      registerForm.value.nickname, 
      registerForm.value.password
    )
    toast.add({
      title: '注册成功',
      color: 'success'
    })
    await navigateTo('/')
  } catch (error) {
    // 错误已经在 useAuth 中处理了
  }
}

// 如果已经登录，重定向到首页
const authStore = useAuthStore()
watch(() => authStore.isAuthenticated, (isAuth) => {
  if (isAuth) {
    navigateTo('/')
  }
}, { immediate: true })
</script>
