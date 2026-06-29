<template>
  <div class="navbar">
    <div class="left">
      <el-icon class="hamburger" @click="toggleSidebar">
        <Fold v-if="!isCollapse" />
        <Expand v-else />
      </el-icon>
      <breadcrumb />
    </div>
    <div class="right">
      <el-tooltip content="全屏" placement="bottom">
        <el-icon class="action-item" @click="toggleFullScreen"><FullScreen /></el-icon>
      </el-tooltip>
      <el-dropdown trigger="click" @command="handleCommand">
        <div class="user-info">
          <el-avatar :size="30" :src="userAvatar" class="avatar">
            <el-icon><UserFilled /></el-icon>
          </el-avatar>
          <span class="username">{{ user.nickname || user.username || 'admin' }}</span>
          <el-icon><ArrowDown /></el-icon>
        </div>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="profile">
              <el-icon><User /></el-icon>个人中心
            </el-dropdown-item>
            <el-dropdown-item divided command="logout">
              <el-icon><SwitchButton /></el-icon>退出登录
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </div>
</template>

<script setup>
import { inject, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessageBox, ElMessage } from 'element-plus'
import { logout as logoutApi } from '@/api/auth'
import Breadcrumb from './Breadcrumb.vue'

const router = useRouter()
const isCollapse = inject('isCollapse')
const toggleSidebar = inject('toggleSidebar')

const user = computed(() => {
  try {
    return JSON.parse(localStorage.getItem('uas_user') || '{}')
  } catch {
    return {}
  }
})

const userAvatar = computed(() => user.value.avatar || '')

const handleCommand = async (cmd) => {
  if (cmd === 'profile') {
    router.push('/profile')
  } else if (cmd === 'logout') {
    try {
      await ElMessageBox.confirm('确认退出登录？', '提示', { type: 'warning' })
      try { await logoutApi() } catch (e) {}
      localStorage.removeItem('uas_token')
      localStorage.removeItem('uas_user')
      ElMessage.success('退出成功')
      router.push('/login')
    } catch (e) {}
  }
}

const toggleFullScreen = () => {
  if (!document.fullscreenElement) {
    document.documentElement.requestFullscreen()
  } else {
    document.exitFullscreen()
  }
}
</script>

<style lang="scss" scoped>
@use '@/styles/variables.scss' as *;

.navbar {
  height: $navbarHeight;
  background: $navbarBg;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  border-bottom: 1px solid #ebeef5;
  box-shadow: 0 1px 3px rgba(0,0,0,0.05);
}

.left { display: flex; align-items: center; gap: 16px; }
.right { display: flex; align-items: center; gap: 16px; }

.hamburger {
  font-size: 20px;
  cursor: pointer;
  color: #5a5e66;
  &:hover { color: $primaryColor; }
}

.action-item {
  font-size: 18px;
  cursor: pointer;
  color: #5a5e66;
  &:hover { color: $primaryColor; }
}

.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 0 8px;
  height: 40px;
  border-radius: 4px;
  &:hover { background: #f5f7fa; }
  .avatar { background: $primaryColor; color: #fff; }
  .username { font-size: 14px; color: #303133; }
}
</style>
