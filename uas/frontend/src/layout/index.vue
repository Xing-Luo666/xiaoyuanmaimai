<template>
  <div class="app-wrapper">
    <!-- 侧边栏 -->
    <sidebar class="sidebar-container" :class="{ collapsed: isCollapse }" />
    <!-- 主区域 -->
    <div class="main-container" :class="{ collapsed: isCollapse }">
      <!-- 顶栏 -->
      <navbar />
      <!-- 标签页 -->
      <tags-view />
      <!-- 内容区 -->
      <app-main />
    </div>
  </div>
</template>

<script setup>
import { ref, provide } from 'vue'
import Sidebar from './components/Sidebar.vue'
import Navbar from './components/Navbar.vue'
import TagsView from './components/TagsView.vue'
import AppMain from './components/AppMain.vue'

const isCollapse = ref(false)
const toggleSidebar = () => { isCollapse.value = !isCollapse.value }
provide('isCollapse', isCollapse)
provide('toggleSidebar', toggleSidebar)
</script>

<style lang="scss" scoped>
@use '@/styles/variables.scss' as *;

.app-wrapper {
  display: flex;
  height: 100%;
  width: 100%;
}

.sidebar-container {
  width: $sidebarWidth;
  height: 100%;
  background: $sidebarBg;
  transition: width 0.28s;
  overflow: hidden;
  flex-shrink: 0;
  z-index: 1001;
  &.collapsed { width: $sidebarCollapseWidth; }
}

.main-container {
  flex: 1;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  transition: margin-left 0.28s;
  min-width: 0;
}
</style>
