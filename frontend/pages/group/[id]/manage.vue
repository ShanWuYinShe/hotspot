<template>
  <div class="group-manage-page">
    <!-- 应用导航栏 -->
    <div class="manage-header">
      <div class="manage-header__content">
        <h1 class="manage-header__title text-responsive-xl text-gradient-fire">群组管理</h1>
        <div class="manage-header__actions">
          <button @click="showCreateGroupDialog = true" class="manage-btn manage-btn--create"
            v-tooltip="{ category: 'navbar', key: 'createGroup' }">
            创建群组
          </button>
          <button @click="showNewMessageDialog = true" class="manage-btn manage-btn--message"
            v-tooltip="{ category: 'navbar', key: 'newMessage' }">
            新建聊天
          </button>
        </div>
      </div>
    </div>

    <!-- 桌面端头部 -->
    <div class="desktop-header">
      <div class="header-content">
        <BackButton />
        <div class="group-info">
          <h3 class="text-responsive-lg text-gold">{{ groupInfo?.name }} - 群组管理</h3>
          <span class="group-desc text-responsive-sm text-bright">管理群组设置和成员权限</span>
        </div>
      </div>
    </div>

    <!-- 主要内容区域 -->
    <div class="manage-container">
      <!-- 群组信息设置 -->
      <div class="manage-section">
        <div class="section-header">
          <h4 class="text-responsive-lg text-electric">基本信息</h4>
          <span class="section-desc text-responsive-sm">群组的基本信息设置</span>
        </div>
        <div class="section-content">
          <div class="info-item">
            <div class="info-label text-responsive-base text-bright">群组头像</div>
            <div class="info-content">
              <div class="avatar-upload">
                <NAvatar :src="groupInfo?.avatar_url" :size="64">
                  {{ groupInfo?.name?.charAt(0) }}
                </NAvatar>
                <NButton type="primary" size="small" @click="uploadAvatar" class="upload-btn"
                  v-tooltip="{ category: 'groupManage', key: 'uploadAvatar' }">
                  <NIcon>
                    <Camera />
                  </NIcon>
                  更换头像
                </NButton>
              </div>
            </div>
          </div>

          <div class="info-item">
            <div class="info-label">群组名称</div>
            <div class="info-content">
              <NInput v-model="editForm.name" placeholder="请输入群组名称" :maxlength="20" show-word-limit
                v-tooltip="{ category: 'groupManage', key: 'groupName' }" />
            </div>
          </div>

          <div class="info-item">
            <div class="info-label">群组简介</div>
            <div class="info-content">
              <NInput v-model="editForm.description" type="textarea" placeholder="请输入群组简介" :rows="3" :maxlength="200"
                show-word-limit v-tooltip="{ category: 'groupManage', key: 'groupDescription' }" />
            </div>
          </div>

          <div class="info-item">
            <div class="info-label">群组类型</div>
            <div class="info-content">
              <NSelect 
                v-model:value="editForm.type" 
                v-tooltip="{ category: 'groupManage', key: 'groupType' }"
                placement="bottom-start"
                :options="[
                  { label: '公开群组', value: 'public' },
                  { label: '私有群组', value: 'private' }
                ]"
              />
              <div class="help-text">
                <span v-if="editForm.type === 'public'">任何人都可以搜索并申请加入</span>
                <span v-else>仅通过邀请加入</span>
              </div>
            </div>
          </div>

          <div class="section-actions">
            <NButton type="primary" @click="saveBasicInfo" :loading="saving">
              保存更改
            </NButton>
          </div>
        </div>
      </div>

      <!-- 成员管理 -->
      <div class="manage-section">
        <div class="section-header">
          <h4>成员管理</h4>
          <span class="section-desc">管理群组成员和权限</span>
        </div>
        <div class="section-content">
          <div class="member-stats">
            <div class="stat-item">
              <div class="stat-number">{{ memberStats.total }}</div>
              <div class="stat-label">总成员</div>
            </div>
            <div class="stat-item">
              <div class="stat-number">{{ memberStats.admin }}</div>
              <div class="stat-label">管理员</div>
            </div>
            <div class="stat-item">
              <div class="stat-number">{{ memberStats.active }}</div>
              <div class="stat-label">活跃成员</div>
            </div>
          </div>

          <div class="member-actions">
            <NButton type="primary" @click="router.push(`/group/${groupId}/members`)">
              <NIcon>
                <User />
              </NIcon>
              查看所有成员
            </NButton>
            <NButton @click="showInviteDialog = true">
              <NIcon>
                <Plus />
              </NIcon>
              邀请成员
            </NButton>
            <NButton @click="showBatchManage = true">
              <NIcon>
                <Operation />
              </NIcon>
              批量管理
            </NButton>
          </div>
        </div>
      </div>

      <!-- 权限设置 -->
      <div class="manage-section">
        <div class="section-header">
          <h4>权限设置</h4>
          <span class="section-desc">控制群组的各项权限</span>
        </div>
        <div class="section-content">
          <div class="permission-item">
            <div class="permission-info">
              <div class="permission-label">允许成员邀请</div>
              <div class="permission-desc">普通成员是否可以邀请其他人加入</div>
            </div>
            <NSwitch v-model="permissions.allowMemberInvite" @change="updatePermission('allowMemberInvite')" />
          </div>

          <div class="permission-item">
            <div class="permission-info">
              <div class="permission-label">全员禁言</div>
              <div class="permission-desc">是否开启全员禁言模式</div>
            </div>
            <NSwitch v-model="permissions.muteAll" @change="updatePermission('muteAll')" />
          </div>

          <div class="permission-item">
            <div class="permission-info">
              <div class="permission-label">审核新成员</div>
              <div class="permission-desc">新成员加入是否需要管理员审核</div>
            </div>
            <NSwitch v-model="permissions.requireApproval" @change="updatePermission('requireApproval')" />
          </div>

          <div class="permission-item">
            <div class="permission-info">
              <div class="permission-label">允许搜索</div>
              <div class="permission-desc">是否允许其他用户搜索到此群组</div>
            </div>
            <NSwitch v-model="permissions.allowSearch" @change="updatePermission('allowSearch')" />
          </div>

          <div class="permission-item">
            <div class="permission-info">
              <div class="permission-label">消息阅后即焚</div>
              <div class="permission-desc">消息在指定时间后自动删除</div>
            </div>
            <div class="permission-value">
              <NSwitch v-model="permissions.messageExpire" @change="updatePermission('messageExpire')" />
              <NSelect 
                v-if="permissions.messageExpire" 
                v-model:value="permissions.expireTime" 
                placeholder="选择时间"
                @update:value="updatePermission('expireTime')" 
                class="expire-select"
                placement="bottom-start"
                :options="[
                  { label: '1小时', value: '1h' },
                  { label: '24小时', value: '24h' },
                  { label: '7天', value: '7d' },
                  { label: '30天', value: '30d' }
                ]"
              />
            </div>
          </div>
        </div>
      </div>

      <!-- 数据统计 -->
      <div class="manage-section">
        <div class="section-header">
          <h4>数据统计</h4>
          <span class="section-desc">群组活跃度和数据分析</span>
        </div>
        <div class="section-content">
          <div class="stats-grid">
            <div class="stats-card">
              <div class="stats-icon">💬</div>
              <div class="stats-content">
                <div class="stats-number">{{ stats.messageCount }}</div>
                <div class="stats-label">消息数量</div>
              </div>
            </div>
            <div class="stats-card">
              <div class="stats-icon">📅</div>
              <div class="stats-content">
                <div class="stats-number">{{ formatDate(stats.createdAt) }}</div>
                <div class="stats-label">创建时间</div>
              </div>
            </div>
            <div class="stats-card">
              <div class="stats-icon">⭐</div>
              <div class="stats-content">
                <div class="stats-number">{{ stats.activityScore }}</div>
                <div class="stats-label">活跃度评分</div>
              </div>
            </div>
            <div class="stats-card">
              <div class="stats-icon">📈</div>
              <div class="stats-content">
                <div class="stats-number">+{{ stats.weeklyGrowth }}%</div>
                <div class="stats-label">本周增长</div>
              </div>
            </div>
          </div>

          <div class="chart-section">
            <div class="chart-header">
              <h5>成员活跃度趋势</h5>
              <NDatePicker v-model:value="chartDateRange" type="daterange" separator="至" start-placeholder="开始日期"
                end-placeholder="结束日期" size="small" @update:value="updateChart" />
            </div>
            <div class="chart-placeholder">
              <div class="chart-icon">📊</div>
              <p>活跃度图表</p>
              <p class="chart-desc">显示群组成员在指定时间段内的活跃情况</p>
            </div>
          </div>
        </div>
      </div>

      <!-- 危险操作 -->
      <div class="manage-section danger-section">
        <div class="section-header">
          <h4>危险操作</h4>
          <span class="section-desc">这些操作可能影响群组正常使用</span>
        </div>
        <div class="section-content">
          <div class="danger-actions">
            <div class="danger-item">
              <div class="danger-info">
                <div class="danger-label">转移群主</div>
                <div class="danger-desc">将群主权限转移给其他管理员</div>
              </div>
              <NButton type="warning" @click="showTransferDialog = true" :disabled="!canTransfer">
                转移群主
              </NButton>
            </div>

            <div class="danger-item">
              <div class="danger-info">
                <div class="danger-label">解散群组</div>
                <div class="danger-desc">永久删除群组，此操作不可撤销</div>
              </div>
              <NButton type="error" @click="showDissolveDialog = true">
                解散群组
              </NButton>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 邀请成员对话框 -->
    <NDialog v-model="showInviteDialog" title="邀请成员" width="400px">
      <div class="invite-content">
        <p>分享群组邀请链接或二维码</p>
        <div class="invite-actions">
          <NButton type="primary" @click="copyInviteLink" block>
            <NIcon>
              <Link />
            </NIcon>
            复制邀请链接
          </NButton>
          <NButton @click="showQRCode" block>
            <NIcon>
              <QrCode />
            </NIcon>
            生成二维码
          </NButton>
        </div>
      </div>
    </NDialog>

    <!-- 批量管理对话框 -->
    <NDialog v-model="showBatchManage" title="批量管理成员" width="600px">
      <div class="batch-manage-content">
        <p>选择要批量操作的成员</p>
        <div class="batch-actions">
          <NButton type="warning">批量禁言</NButton>
          <NButton type="error">批量移除</NButton>
          <NButton>批量设为管理员</NButton>
        </div>
      </div>
    </NDialog>

    <!-- 转移群主对话框 -->
    <NDialog v-model="showTransferDialog" title="转移群主" width="400px">
      <div class="transfer-content">
        <NAlert title="注意" description="转移群主后，您将失去群主权限，新群主将拥有群组的完全控制权。" type="warning" show-icon :closable="false" />
        <div class="transfer-select">
          <div class="select-label">选择新群主：</div>
          <NSelect 
            v-model:value="selectedNewOwner" 
            placeholder="请选择管理员"
            placement="bottom-start"
            :options="adminOptions"
          />
        </div>
        <div class="transfer-confirm">
          <NInput v-model="transferConfirmText" placeholder="请输入'确认转移'来确认操作" />
        </div>
      </div>
      <template #action>
        <NButton @click="showTransferDialog = false">取消</NButton>
        <NButton type="error" @click="confirmTransfer" :disabled="transferConfirmText !== '确认转移' || !selectedNewOwner">
          确认转移
        </NButton>
      </template>
    </NDialog>

    <!-- 解散群组对话框 -->
    <NDialog v-model="showDissolveDialog" title="解散群组" width="400px">
      <div class="dissolve-content">
        <NAlert title="警告" description="解散群组将永久删除所有消息记录和成员信息，此操作不可撤销！" type="error" show-icon :closable="false" />
        <div class="dissolve-confirm">
          <div class="confirm-label">请输入群组名称确认解散：</div>
          <NInput v-model="dissolveConfirmText" :placeholder="`请输入：${groupInfo?.name}`" />
        </div>
      </div>
      <template #action>
        <NButton @click="showDissolveDialog = false">取消</NButton>
        <NButton type="error" @click="confirmDissolve" :disabled="dissolveConfirmText !== groupInfo?.name">
          确认解散
        </NButton>
      </template>
    </NDialog>

    <!-- 隐藏的文件输入 -->
    <input ref="fileInput" type="file" accept="image/*" @change="handleAvatarUpload" style="display: none" />

    <!-- 创建群组对话框 -->
    <div v-if="showCreateGroupDialog" class="modal-overlay">
      <div class="modal-content">
        <h3 class="modal-title">创建群组</h3>
        <p class="modal-description">创建群组功能开发中...</p>
        <div class="modal-actions">
          <button @click="showCreateGroupDialog = false" class="modal-btn modal-btn--secondary">
            取消
          </button>
        </div>
      </div>
    </div>

    <!-- 新消息对话框 -->
    <div v-if="showNewMessageDialog" class="modal-overlay">
      <div class="modal-content">
        <h3 class="modal-title">新建聊天</h3>
        <p class="modal-description">新建聊天功能开发中...</p>
        <div class="modal-actions">
          <button @click="showNewMessageDialog = false" class="modal-btn modal-btn--secondary">
            取消
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// 禁用 SSR 以避免 Naive UI 组件的 document.head 错误
definePageMeta({
  ssr: false
})

// Vue 3 APIs
import { ref, computed, watchEffect, onMounted } from 'vue'

// Naive UI components
import { NButton, NIcon, NAvatar, NInput, NSelect, NSwitch, NDialog, NAlert, NTag } from 'naive-ui'
import { useMessage, NDatePicker } from 'naive-ui'
import type { SelectOption } from 'naive-ui'

// Ionicons icons
import { ChevronBackOutline as ArrowLeft, Camera, Add as Plus, Settings as Setting, Person as User, Warning } from '@vicons/ionicons5'

// 项目组件导入
import BackButton from '~/components/common/BackButton.vue'

// Stores
import { useAuthStore } from '~/stores/auth'

// Composables

// Services
import { useGroupService } from '~/services'

// 服务实例
const { searchGroupsById } = useGroupService()

// Utils
import { extractErrorMessage } from '~/utils/error-handler'

// Types
interface GroupInfo {
  group_id: string
  name: string
  description?: string
  avatar_url?: string
  type: 'public' | 'private'
  member_count: number
  user_role: 'owner' | 'admin' | 'member'
  created_at: string
}

// Composables
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

// API实例
const message = useMessage()

// 技能栏功能函数
// 滚动到指定部分
const scrollToSection = (sectionId: string) => {
  const element = document.querySelector(`[data-section="${sectionId}"]`)
  if (element) {
    element.scrollIntoView({ behavior: 'smooth' })
  } else {
    message.info(`跳转到${sectionId}部分`)
  }
}

// 保存群组设置
const saveGroupSettings = async () => {
  try {
    // TODO: 实现保存群组设置功能
    message.success('群组设置已保存')
  } catch (error) {
    message.error('保存失败')
  }
}

// 确认删除群组
const confirmDeleteGroup = () => {
  // TODO: 实现确认删除群组功能
  message.warning('删除群组功能开发中...')
}

// 响应式数据
const groupId = computed(() => route.params.id as string)
const currentUserId = computed(() => authStore.user?.user_id)

const groupInfo = ref<GroupInfo | null>(null)
const saving = ref(false)
const showInviteDialog = ref(false)
const showBatchManage = ref(false)
const showTransferDialog = ref(false)
const showDissolveDialog = ref(false)
const showCreateGroupDialog = ref(false)
const showNewMessageDialog = ref(false)
const chartDateRange = ref<[number, number]>([Date.now(), Date.now()])

// 表单数据
const editForm = ref({
  name: '',
  description: '',
  type: 'public'
})

const permissions = ref({
  allowMemberInvite: true,
  muteAll: false,
  requireApproval: false,
  allowSearch: true,
  messageExpire: false,
  expireTime: '24h'
})

const memberStats = ref({
  total: 0,
  admin: 0,
  active: 0
})

const stats = ref({
  messageCount: 0,
  createdAt: '',
  activityScore: 0,
  weeklyGrowth: 0
})

// 危险操作相关
const selectedNewOwner = ref('')
const transferConfirmText = ref('')
const dissolveConfirmText = ref('')
const adminList = ref<any[]>([])

// DOM 引用
const fileInput = ref<HTMLInputElement>()

// 计算属性
const canTransfer = computed(() => {
  return groupInfo.value?.user_role === 'owner' && adminList.value.length > 0
})

// 管理员选项
const adminOptions = computed(() => {
  return adminList.value.map(admin => ({
    label: admin.name,
    value: admin.id
  }))
})

// 生命周期
onMounted(async () => {
  await loadGroupInfo()
  await loadMemberStats()
  await loadStats()
  await loadAdminList()
})

// 加载群组信息
const loadGroupInfo = async () => {
  try {
    const data = await searchGroupsById({ group_id: groupId.value })
    groupInfo.value = {
      ...data,
      type: 'public', // 默认为公开群组，因为API响应中没有type字段
      user_role: data.user_role || 'member' // 确保有用户角色
    } as GroupInfo

    // 初始化编辑表单
    editForm.value = {
      name: data.name || '',
      description: data.description || '',
      type: 'public'
    }
  } catch (error) {
    const errorMessage = extractErrorMessage(error, '加载群组信息失败')
    message.error(errorMessage)
  }
}

// 加载成员统计
const loadMemberStats = async () => {
  try {
    // 从API获取真实数据
    memberStats.value = {
      total: 0,
      admin: 0,
      active: 0
    }
  } catch (error) {
    const errorMessage = extractErrorMessage(error, '加载成员统计失败')
    message.error(errorMessage)
  }
}

// 加载数据统计
const loadStats = async () => {
  try {
    // 暂时使用基本数据，待API完善
    stats.value = {
      messageCount: 0,
      createdAt: groupInfo.value?.created_at || '',
      activityScore: 0,
      weeklyGrowth: 0
    }
  } catch (error) {
    const errorMessage = extractErrorMessage(error, '加载数据统计失败')
    message.error(errorMessage)
  }
}

// 加载管理员列表
const loadAdminList = async () => {
  try {
    // 暂时使用空列表，待群组成员API完善
    adminList.value = []
  } catch (error) {
    const errorMessage = extractErrorMessage(error, '加载管理员列表失败')
    message.error(errorMessage)
  }
}

// 保存基本信息
const saveBasicInfo = async () => {
  try {
    saving.value = true

    // TODO: 调用API保存群组信息 - 等待API完善
    // await groupService.updateGroup({
    //   group_id: groupId.value,
    //   name: editForm.value.name,
    //   description: editForm.value.description
    // })

    // 更新本地数据
    if (groupInfo.value) {
      groupInfo.value.name = editForm.value.name
      groupInfo.value.description = editForm.value.description
    }

    message.success('群组信息已更新')
  } catch (error) {
    message.error('保存失败')
  } finally {
    saving.value = false
  }
}

// 更新权限设置
const updatePermission = async (key: keyof typeof permissions.value) => {
  try {
    // TODO: 调用真实的群组权限更新API
    // await groupService.updateGroupPermissions(groupId.value, { [key]: permissions.value[key] })

    message.success('权限设置已更新')
  } catch (error) {
    message.error('更新失败')
    // 恢复原状态
    if (typeof permissions.value[key] === 'boolean') {
      (permissions.value[key] as boolean) = !(permissions.value[key] as boolean)
    }
  }
}

// 上传头像
const uploadAvatar = () => {
  fileInput.value?.click()
}

// 处理头像上传
const handleAvatarUpload = async (event: Event) => {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return

  try {
    // TODO: 上传头像到服务器
    message.success('头像上传成功')
  } catch (error) {
    message.error('头像上传失败')
  }
}

// 复制邀请链接
const copyInviteLink = async () => {
  try {
    const link = `${window.location.origin}/group/${groupId.value}/join`
    await navigator.clipboard.writeText(link)
    message.success('邀请链接已复制')
    showInviteDialog.value = false
  } catch (error) {
    message.error('复制失败')
  }
}

// 显示二维码
const showQRCode = () => {
  message.info('二维码功能开发中...')
}

// 更新图表
const updateChart = () => {
  message.info('图表更新功能开发中...')
}

// 确认转移群主
const confirmTransfer = async () => {
  try {
    // TODO: 调用真实的群主转移API
    // await groupService.transferOwnership(groupId.value, transferTargetUserId.value)

    message.success('群主转移成功')
    showTransferDialog.value = false

    // 转移成功后跳转
    navigateTo('/my-groups')
  } catch (error) {
    message.error('转移失败')
  }
}

// 确认解散群组
const confirmDissolve = async () => {
  try {
    // TODO: 调用API解散群组 - 等待API完善
    // await groupService.deleteGroup(groupId.value)

    message.success('群组已解散')
    showDissolveDialog.value = false

    // 解散成功后跳转
    navigateTo('/my-groups')
  } catch (error) {
    message.error('解散失败')
  }
}

// 格式化日期
const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleDateString('zh-CN')
}

// 返回上级页面
const goBack = () => {
  router.back()
}

// Dialog 成功处理器
const handleCreateGroupSuccess = async (group: any) => {
  message.success('群组创建成功！')
  await navigateTo(`/group/${group.group_id}/chat`)
}

const handleNewMessageSuccess = async (conversation: any) => {
  message.success('消息发送成功！')
  await router.push(`/chat/${conversation.conversation_id}`)
}

// 删除群组
const handleDeleteGroup = () => {
  message.info('删除群组功能开发中...')
}

// 页面权限检查
if (typeof window !== 'undefined') {
  watchEffect(() => {
    if (groupInfo.value && groupInfo.value.user_role !== 'owner') {
      message.error('您没有权限访问此页面')
      navigateTo(`/group/${groupId.value}`)
    }
  })
}

// 设置页面标题
useHead({
  title: computed(() => `${groupInfo.value?.name || '群组'} - 管理 - HotSpot`)
})
</script>

<style lang="scss" scoped>
.group-manage-page {
  min-height: 100vh;
  background: #f5f5f5;
}

.desktop-header {
  background: white;
  border-bottom: 1px solid #e1e4e8;
  padding: 16px 24px;

  .header-content {
    display: flex;
    align-items: center;
    max-width: 1200px;
    margin: 0 auto;
  }

  .back-btn {
    margin-right: 16px;
  }

  .group-info {
    flex: 1;
    margin-left: 16px;

    h3 {
      margin: 0;
      font-size: 16px;
      font-weight: 600;
    }

    .group-desc {
      font-size: 12px;
      color: #666;
      margin-top: 4px;
    }
  }
}

.manage-container {
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}

.manage-section {
  background: white;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);

  &.danger-section {
    border: 2px solid #fecaca;
    background: #fef2f2;
  }

  .section-header {
    margin-bottom: 20px;
    padding-bottom: 16px;
    border-bottom: 1px solid #f0f0f0;

    h4 {
      margin: 0 0 4px 0;
      font-size: 18px;
      font-weight: 600;
      color: #333;
    }

    .section-desc {
      font-size: 14px;
      color: #666;
    }
  }

  .section-actions {
    margin-top: 20px;
    padding-top: 16px;
    border-top: 1px solid #f0f0f0;
  }
}

.info-item {
  display: flex;
  align-items: flex-start;
  margin-bottom: 20px;

  &:last-child {
    margin-bottom: 0;
  }

  .info-label {
    width: 120px;
    font-weight: 600;
    color: #333;
    margin-top: 8px;
    flex-shrink: 0;
  }

  .info-content {
    flex: 1;

    .avatar-upload {
      display: flex;
      align-items: center;
      gap: 16px;

      .upload-btn {
        height: 32px;
      }
    }

    .help-text {
      margin-top: 8px;
      font-size: 12px;
      color: #999;
    }
  }
}

.member-stats {
  display: flex;
  gap: 20px;
  margin-bottom: 20px;

  .stat-item {
    text-align: center;
    padding: 16px;
    background: #f8fafc;
    border-radius: 8px;
    flex: 1;

    .stat-number {
      font-size: 24px;
      font-weight: 600;
      color: #1e88e5;
      margin-bottom: 4px;
    }

    .stat-label {
      font-size: 12px;
      color: #666;
    }
  }
}

.member-actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.permission-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 0;
  border-bottom: 1px solid #f0f0f0;

  &:last-child {
    border-bottom: none;
  }

  .permission-info {
    flex: 1;

    .permission-label {
      font-weight: 600;
      color: #333;
      margin-bottom: 4px;
    }

    .permission-desc {
      font-size: 12px;
      color: #666;
    }
  }

  .permission-value {
    display: flex;
    align-items: center;
    gap: 12px;

    .expire-select {
      width: 120px;
    }
  }
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.stats-card {
  display: flex;
  align-items: center;
  padding: 16px;
  background: #f8fafc;
  border-radius: 8px;

  .stats-icon {
    font-size: 24px;
    margin-right: 12px;
  }

  .stats-content {
    .stats-number {
      font-size: 18px;
      font-weight: 600;
      color: #1e88e5;
      margin-bottom: 4px;
    }

    .stats-label {
      font-size: 12px;
      color: #666;
    }
  }
}

.chart-section {
  .chart-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;

    h5 {
      margin: 0;
      font-size: 16px;
      font-weight: 600;
    }
  }

  .chart-placeholder {
    text-align: center;
    padding: 60px 20px;
    border: 2px dashed #ddd;
    border-radius: 8px;

    .chart-icon {
      font-size: 48px;
      margin-bottom: 16px;
    }

    p {
      margin: 8px 0;
      color: #666;

      &.chart-desc {
        font-size: 12px;
        color: #999;
      }
    }
  }
}

.danger-actions {
  .danger-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 0;
    border-bottom: 1px solid #f0f0f0;

    &:last-child {
      border-bottom: none;
    }

    .danger-info {
      flex: 1;

      .danger-label {
        font-weight: 600;
        color: #dc2626;
        margin-bottom: 4px;
      }

      .danger-desc {
        font-size: 12px;
        color: #666;
      }
    }
  }
}

.invite-content {
  text-align: center;

  .invite-actions {
    display: flex;
    flex-direction: column;
    gap: 12px;
    margin-top: 20px;
  }
}

.batch-manage-content {
  text-align: center;

  .batch-actions {
    display: flex;
    justify-content: center;
    gap: 12px;
    margin-top: 20px;
  }
}

.transfer-content {
  .transfer-select {
    margin: 20px 0;

    .select-label {
      margin-bottom: 8px;
      font-weight: 600;
    }
  }

  .transfer-confirm {
    margin-top: 20px;
  }
}

// 管理页面头部样式
.manage-header {
  background: #ffffff;
  box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1);
  border-bottom: 1px solid #e5e7eb;
  padding: 0.75rem 1rem;

  &__content {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  &__title {
    font-size: 1.125rem;
    font-weight: 600;
    color: #111827;
  }

  &__actions {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
}

.manage-btn {
  padding: 0.5rem 1rem;
  border-radius: 0.5rem;
  border: none;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease-in-out;

  &--create {
    background: #16a34a;
    color: #ffffff;

    &:hover {
      background: #15803d;
    }
  }

  &--message {
    background: #2563eb;
    color: #ffffff;

    &:hover {
      background: #1d4ed8;
    }
  }
}

// 模态框样式
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: var(--z-dropdown);
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
}

.modal-content {
  background: #ffffff;
  border-radius: 0.5rem;
  padding: 1.5rem;
  max-width: 28rem;
  width: 100%;
  margin: 0 1rem;
}

.modal-title {
  font-size: 1.125rem;
  font-weight: 600;
  margin-bottom: 1rem;
  color: #111827;
}

.modal-description {
  color: #6b7280;
  margin-bottom: 1rem;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}

.modal-btn {
  padding: 0.5rem 1rem;
  border-radius: 0.25rem;
  border: none;
  cursor: pointer;
  transition: all 0.2s ease-in-out;

  &--secondary {
    color: #6b7280;
    background: transparent;

    &:hover {
      background: #f3f4f6;
    }
  }
}

.dissolve-content {
  .dissolve-confirm {
    margin-top: 20px;

    .confirm-label {
      margin-bottom: 8px;
      font-weight: 600;
      color: #dc2626;
    }
  }
}

// 移动端适配
@media (max-width: 768px) {
  .manage-container {
    padding: 12px;
  }

  .manage-section {
    padding: 16px;
    margin-bottom: 16px;
  }

  .info-item {
    flex-direction: column;
    gap: 8px;

    .info-label {
      width: auto;
      margin-top: 0;
    }
  }

  .member-stats {
    flex-direction: column;
    gap: 12px;
  }

  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }

  .permission-item,
  .danger-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .chart-section .chart-header {
    flex-direction: column;
    gap: 12px;
    align-items: flex-start;
  }
}
</style>
