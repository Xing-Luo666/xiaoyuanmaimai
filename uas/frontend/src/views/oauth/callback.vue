<template>
  <div class="callback-page">
    <el-result
      :icon="resultIcon"
      :title="resultTitle"
      :sub-title="resultSubTitle"
    >
      <template #extra>
        <el-button type="primary" @click="goHome">{{ redirectLabel }}</el-button>
      </template>
    </el-result>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'

const route = useRoute()
const router = useRouter()

const status = ref('loading') // loading, success, error
const errorMsg = ref('')

const resultIcon = computed(() => status.value === 'success' ? 'success' : status.value === 'error' ? 'error' : 'info')
const resultTitle = computed(() => ({
  loading: '正在处理...',
  success: '授权成功',
  error: '授权失败'
}[status.value]))
const resultSubTitle = computed(() => {
  if (status.value === 'success') return '正在跳转，请稍候...'
  if (status.value === 'error') return errorMsg.value || '授权码无效'
  return ''
})
const redirectLabel = computed(() => status.value === 'success' ? '返回首页' : '重新登录')

const goHome = () => {
  router.push('/')
}

onMounted(async () => {
  const code = route.query.code
  const error = route.query.error
  const state = route.query.state

  if (error) {
    status.value = 'error'
    errorMsg.value = '您拒绝了授权'
    return
  }

  if (!code) {
    status.value = 'error'
    errorMsg.value = '未获取到授权码'
    return
  }

  // 此页面是给第三方应用回调用的，正常情况下第三方应用应该用自己的后端用 code 换 token
  // 这里仅作演示，把code传给应用后端
  status.value = 'success'
  ElMessage.success('授权成功，正在返回应用')

  // 如果有 state 参数（通常是应用自己的回调地址），跳转回去
  setTimeout(() => {
    if (state && state.startsWith('http')) {
      window.location.href = state
    } else {
      router.push('/')
    }
  }, 1500)
})
</script>

<style lang="scss" scoped>
.callback-page {
  min-height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f7fa;
}
</style>
